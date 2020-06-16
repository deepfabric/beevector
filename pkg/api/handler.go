package api

import (
	"fmt"

	"github.com/deepfabric/beehive/util"
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
	s.store.AsyncExecCommand(&ctx.req.Search, s.onResp, ctx)
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
		case rpcpb.Search:
			// TODO: Aggregation result
			protoc.MustUnmarshal(&resp.Search, value)
		}

		rs.(*util.Session).OnResp(resp)
	} else {
		if log.DebugEnabled() {
			log.Debugf("%d api received response, missing session", ctx.req.ID)
		}
	}
}
