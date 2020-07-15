package storage

import (
	"github.com/deepfabric/beehive/pb"
	"github.com/deepfabric/beehive/pb/raftcmdpb"
	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/protoc"
)

type batchReader struct {
	shard uint64
	store *storage

	// reset
	maxTopk   int
	topks     []int
	xqs       []float32
	bitmaps   [][]byte
	scores    []float32
	xids      []int64
	responses []*raftcmdpb.Response

	// tmp
	request  *rpcpb.SearchRequest
	response *rpcpb.SearchResponse
}

func newBatchReader(store *storage) *batchReader {
	return &batchReader{
		store:    store,
		request:  &rpcpb.SearchRequest{},
		response: &rpcpb.SearchResponse{},
	}
}

func (b *batchReader) Add(shard uint64, req *raftcmdpb.Request, attrs map[string]interface{}) (bool, error) {
	if b.shard != 0 && b.shard != shard {
		log.Fatalf("BUG: diffent shard opts in a read batch, %d, %d",
			b.shard,
			shard)
	}

	if req.CustemType != uint64(rpcpb.Search) {
		return false, nil
	}

	b.request.Reset()
	protoc.MustUnmarshal(b.request, req.Cmd)

	topk := int(b.request.Topk)
	if topk > b.maxTopk {
		b.maxTopk = topk
	}

	b.shard = shard
	b.topks = append(b.topks, topk)
	b.xqs = append(b.xqs, b.request.Xqs...)
	b.bitmaps = append(b.bitmaps, b.request.Bitmaps...)
	return true, nil
}

func (b *batchReader) Execute() ([]*raftcmdpb.Response, error) {
	db := b.store.mustLoadDB(b.shard)
	err := db.Search(b.maxTopk, b.xqs, b.bitmaps, b.cb)
	if err != nil {
		return nil, err
	}

	return b.responses, nil
}

func (b *batchReader) cb(i int, j int, score float32, xid int64) bool {
	b.scores = append(b.scores, score)
	b.xids = append(b.xids, xid)

	if j >= b.topks[i]-1 {
		b.response.Reset()
		b.response.Xids = b.xids
		b.response.Scores = b.scores

		resp := pb.AcquireResponse()
		resp.Value = protoc.MustMarshal(b.response)
		b.responses = append(b.responses, resp)
		return false
	}

	return true
}

func (b *batchReader) Reset() {
	b.shard = 0
	b.maxTopk = 0
	b.xqs = b.xqs[:0]
	b.scores = b.scores[:0]
	b.xids = b.xids[:0]
	b.bitmaps = b.bitmaps[:0]
	b.topks = b.topks[:0]
	b.responses = b.responses[:0]
}
