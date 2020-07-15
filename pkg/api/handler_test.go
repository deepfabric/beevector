package api

import (
	"sort"
	"testing"

	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/stretchr/testify/assert"
)

func TestValuesSort(t *testing.T) {
	var resps []*rpcpb.SearchResponse
	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{1.0, 2.0, 3.0, 4.0, 5.0},
		Xids:   []int64{1, 2, 3, 4, 5},
	})

	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{6.0, 7.0, 8.0, 9.0, 10.0},
		Xids:   []int64{6, 7, 8, 9, 10},
	})

	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{11.0, 12.0, 13.0, 14.0, 15.0},
		Xids:   []int64{11, 12, 13, 14, 15},
	})

	v := newValues(resps)
	sort.Sort(v)

	assert.Equal(t, float32(15.0), resps[0].Scores[0], "TestValuesSort failed")
	assert.Equal(t, int64(15), resps[0].Xids[0], "TestValuesSort failed")
}

func TestValuesPop(t *testing.T) {
	var resps []*rpcpb.SearchResponse
	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{1.0, 2.0, 3.0, 4.0, 5.0},
		Xids:   []int64{1, 2, 3, 4, 5},
	})

	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{6.0, 7.0, 8.0, 9.0, 10.0},
		Xids:   []int64{6, 7, 8, 9, 10},
	})

	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{11.0, 12.0, 13.0, 14.0, 15.0},
		Xids:   []int64{11, 12, 13, 14, 15},
	})

	v := newValues(resps)
	sort.Sort(v)

	resp := &rpcpb.SearchResponse{}
	v.pop(1, resp)
	assert.Equal(t, 1, len(resp.Scores), "TestValuesPop")
	assert.Equal(t, float32(15.0), resp.Scores[0], "TestValuesSort failed")
	assert.Equal(t, int64(15), resp.Xids[0], "TestValuesSort failed")

	resp.Reset()
	v.pop(5, resp)
	assert.Equal(t, 5, len(resp.Scores), "TestValuesPop")
	assert.Equal(t, float32(11.0), resp.Scores[4], "TestValuesSort failed")
	assert.Equal(t, int64(11), resp.Xids[4], "TestValuesSort failed")

	resp.Reset()
	v.pop(12, resp)
	assert.Equal(t, 12, len(resp.Scores), "TestValuesPop")
	assert.Equal(t, float32(4.0), resp.Scores[11], "TestValuesSort failed")
	assert.Equal(t, int64(4), resp.Xids[11], "TestValuesSort failed")

	resp.Reset()
	v.pop(20, resp)
	assert.Equal(t, 15, len(resp.Scores), "TestValuesPop")
	assert.Equal(t, float32(1.0), resp.Scores[14], "TestValuesSort failed")
	assert.Equal(t, int64(1), resp.Xids[14], "TestValuesSort failed")
}

func TestValuesGet(t *testing.T) {
	var resps []*rpcpb.SearchResponse
	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{1.0, 2.0, 3.0, 4.0, 5.0},
		Xids:   []int64{1, 2, 3, 4, 5},
	})

	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{6.0, 7.0, 8.0, 9.0, 10.0},
		Xids:   []int64{6, 7, 8, 9, 10},
	})

	resps = append(resps, &rpcpb.SearchResponse{
		Scores: []float32{11.0, 12.0, 13.0, 14.0, 15.0},
		Xids:   []int64{11, 12, 13, 14, 15},
	})

	v := newValues(resps)
	i, j := v.get(0)
	assert.Equal(t, 0, i, "TestValuesGet failed")
	assert.Equal(t, 0, j, "TestValuesGet failed")

	i, j = v.get(4)
	assert.Equal(t, 0, i, "TestValuesGet failed")
	assert.Equal(t, 4, j, "TestValuesGet failed")

	i, j = v.get(5)
	assert.Equal(t, 1, i, "TestValuesGet failed")
	assert.Equal(t, 0, j, "TestValuesGet failed")

	i, j = v.get(6)
	assert.Equal(t, 1, i, "TestValuesGet failed")
	assert.Equal(t, 1, j, "TestValuesGet failed")
}
