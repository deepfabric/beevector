package api

import (
	"fmt"
	"sort"

	"github.com/deepfabric/beehive/util"
	"github.com/deepfabric/beevector/pkg/pb"
	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/protoc"
)

type ctx struct {
	sid interface{}
	req *rpcpb.Request
}

func (s *server) onReq(sid interface{}, req *rpcpb.Request) error {
	ctx := ctx{sid: sid, req: req}

	switch req.Type {
	case rpcpb.Add:
		return s.doAdd(ctx)
	case rpcpb.Search:
		return s.doSearch(ctx)
	}

	return fmt.Errorf("not support type %d", req.Type)
}

func (s *server) doAdd(ctx ctx) error {
	s.store.AsyncExecCommand(&ctx.req.Add, s.onResp, ctx)
	return nil
}

func (s *server) doSearch(ctx ctx) error {
	s.store.AsyncBroadcastCommand(&ctx.req.Search, s.onBroadcastResp, ctx, false)
	return nil
}

func (s *server) onResp(arg interface{}, value []byte, err error) {
	ctx := arg.(ctx)
	if log.DebugEnabled() {
		log.Debugf("%d api received response", ctx.req.ID)
	}
	if rs, ok := s.sessions.Load(ctx.sid); ok {
		resp := &rpcpb.Response{}
		resp.Type = ctx.req.Type
		resp.ID = ctx.req.ID
		if err != nil {
			resp.Error.Error = err.Error()
			rs.(*util.Session).OnResp(resp)
			return
		}

		switch ctx.req.Type {
		case rpcpb.Add:
			// empty response
		}

		rs.(*util.Session).OnResp(resp)
	} else {
		if log.DebugEnabled() {
			log.Debugf("%d api received response, missing session", ctx.req.ID)
		}
	}
}

func (s *server) onBroadcastResp(arg interface{}, values [][]byte, err error) {
	ctx := arg.(ctx)
	if log.DebugEnabled() {
		log.Debugf("%d api received broadcast response", ctx.req.ID)
	}
	if rs, ok := s.sessions.Load(ctx.sid); ok {
		resp := &rpcpb.Response{}
		resp.Type = ctx.req.Type
		resp.ID = ctx.req.ID
		if err != nil {
			resp.Error.Error = err.Error()
			rs.(*util.Session).OnResp(resp)
			return
		}

		switch ctx.req.Type {
		case rpcpb.Search:
			if len(values) > 0 {
				var responses []*rpcpb.SearchResponse
				for _, value := range values {
					resp := pb.AcquireSearchResponse()
					protoc.MustUnmarshal(resp, value)
					if len(resp.Scores) == 0 {
						pb.ReleaseSearchResponse(resp)
						continue
					}
					responses = append(responses, resp)
				}

				v := newValues(responses)
				sort.Sort(v)
				v.pop(int(ctx.req.Search.Topk), &resp.Search)
			}
		}

		rs.(*util.Session).OnResp(resp)
	} else {
		if log.DebugEnabled() {
			log.Debugf("%d api received response, missing session", ctx.req.ID)
		}
	}
}

type values struct {
	size      int
	responses []*rpcpb.SearchResponse
	subSize   []int
}

func newValues(responses []*rpcpb.SearchResponse) *values {
	var subSize []int
	size := 0

	subSize = append(subSize, 0)
	for _, resp := range responses {
		size += len(resp.Scores)
		subSize = append(subSize, size)
	}

	return &values{size, responses, subSize}
}

func (v *values) pop(topk int, resp *rpcpb.SearchResponse) {
	if len(v.responses) == 0 {
		return
	}

	n := len(v.responses[0].Scores)
	if n == topk {
		resp.Scores = v.responses[0].Scores
		resp.Xids = v.responses[0].Xids
		return
	}

	if n > topk {
		resp.Scores = v.responses[0].Scores[:topk]
		resp.Xids = v.responses[0].Xids[:topk]
		return
	}

	if topk > v.size {
		topk = v.size
	}

	resp.Scores = append(resp.Scores, v.responses[0].Scores...)
	resp.Xids = append(resp.Xids, v.responses[0].Xids...)
	for i := n; i < topk; i++ {
		i1, i2 := v.get(i)
		vi := v.responses[i1]

		resp.Scores = append(resp.Scores, vi.Scores[i2])
		resp.Xids = append(resp.Xids, vi.Xids[i2])
	}

	return
}

func (v *values) Len() int {
	return v.size
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (v *values) Less(i, j int) bool {
	i1, i2 := v.get(i)
	j1, j2 := v.get(j)

	vi := v.responses[i1]
	vj := v.responses[j1]

	return vi.Scores[i2] > vj.Scores[j2]
}

// Swap swaps the elements with indexes i and j.
func (v *values) Swap(i, j int) {
	i1, i2 := v.get(i)
	j1, j2 := v.get(j)

	vi := v.responses[i1]
	vj := v.responses[j1]

	vi.Scores[i2], vj.Scores[j2] = vj.Scores[j2], vi.Scores[i2]
	vi.Xids[i2], vj.Xids[j2] = vj.Xids[j2], vi.Xids[i2]
}

func (v *values) get(i int) (int, int) {
	if i == 0 {
		return 0, 0
	}

	target := 0
	for idx, item := range v.subSize {
		if i < item {
			break
		}

		target = idx
	}

	return target, i - v.subSize[target]
}
