package storage

import (
	"github.com/deepfabric/beehive/pb"
	"github.com/deepfabric/beehive/pb/raftcmdpb"
	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/protoc"
)

var (
	emptyBytes = protoc.MustMarshal(&rpcpb.SearchResponse{})
)

type batchReader struct {
	store *storage

	// reset
	shard                     uint64
	idx                       int
	responses                 []*raftcmdpb.Response
	topVectorTrueSearchBatch  *searchBatch
	topVectorFalseSearchBatch *searchBatch

	// tmp
	request  *rpcpb.SearchRequest
	response *rpcpb.SearchResponse
}

func newBatchReader(store *storage) *batchReader {
	b := &batchReader{
		store:    store,
		request:  &rpcpb.SearchRequest{},
		response: &rpcpb.SearchResponse{},
	}

	b.topVectorTrueSearchBatch = newSearchBatch(b, true)
	b.topVectorFalseSearchBatch = newSearchBatch(b, false)

	return b
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

	if b.request.TopVectors {
		b.topVectorTrueSearchBatch.add(b.request, b.idx)
	} else {
		b.topVectorFalseSearchBatch.add(b.request, b.idx)
	}

	b.shard = shard
	b.responses = append(b.responses, pb.AcquireResponse())
	b.idx++
	return true, nil
}

func (b *batchReader) Execute() ([]*raftcmdpb.Response, error) {
	err := b.topVectorTrueSearchBatch.exec()
	if err != nil {
		return nil, err
	}

	err = b.topVectorFalseSearchBatch.exec()
	if err != nil {
		return nil, err
	}

	return b.responses, nil
}

func (b *batchReader) Reset() {
	b.shard = 0
	b.idx = 0
	b.responses = b.responses[:0]
	b.topVectorTrueSearchBatch.reset()
	b.topVectorFalseSearchBatch.reset()
}

type searchBatch struct {
	b          *batchReader
	topVectors bool

	// reset
	maxTopk int
	topks   []int
	xqs     []float32
	bitmaps [][]byte
	scores  []float32
	xids    []int64
	indexes []int
}

func newSearchBatch(b *batchReader, topVectors bool) *searchBatch {
	return &searchBatch{
		b:          b,
		topVectors: topVectors,
	}
}

func (sb *searchBatch) add(req *rpcpb.SearchRequest, idx int) {
	topk := int(req.Topk)
	if topk > sb.maxTopk {
		sb.maxTopk = topk
	}

	sb.topks = append(sb.topks, topk)
	sb.xqs = append(sb.xqs, req.Xqs...)
	sb.bitmaps = append(sb.bitmaps, req.Bitmap)
	sb.indexes = append(sb.indexes, idx)
}

func (sb *searchBatch) exec() error {
	if len(sb.topks) == 0 {
		return nil
	}

	db := sb.b.store.mustLoadDB(sb.b.shard)
	err := db.Search(sb.maxTopk, sb.xqs, sb.bitmaps, sb.cb, sb.topVectors)
	if err != nil {
		return err
	}

	// empty response
	if len(sb.scores) == 0 {
		for idx := range sb.topks {
			sb.b.responses[sb.indexes[idx]].Value = emptyBytes
		}
	}

	return nil
}

func (sb *searchBatch) cb(i, j, n int, score float32, xid int64) bool {
	sb.scores = append(sb.scores, score)
	sb.xids = append(sb.xids, xid)

	if j >= sb.topks[i]-1 || j == n-1 {
		sb.b.response.Reset()
		sb.b.response.Xids = sb.xids
		sb.b.response.Scores = sb.scores

		sb.b.responses[sb.indexes[i]].Value = protoc.MustMarshal(sb.b.response)
		return false
	}

	return true
}

func (sb *searchBatch) reset() {
	sb.maxTopk = 0
	sb.xqs = sb.xqs[:0]
	sb.scores = sb.scores[:0]
	sb.xids = sb.xids[:0]
	sb.bitmaps = sb.bitmaps[:0]
	sb.topks = sb.topks[:0]
	sb.indexes = sb.indexes[:0]
}
