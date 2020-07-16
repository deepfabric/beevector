package db

import (
	"testing"

	"github.com/deepfabric/beevector/pkg/pb/metapb"
	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	db, err := NewVectoDB("/tmp", 1, 128, metapb.RWU)
	assert.NoError(t, err, "TestAdd failed")

	values := make([]float32, 128, 128)
	assert.NoError(t, db.Add(values, []int64{1}), "TestAdd failed")

	var scores []float32
	var xids []int64
	c := 0
	cb := func(i, j, n int, score float32, xid int64) bool {
		scores = append(scores, score)
		xids = append(xids, xid)

		if j == n-1 {
			c++
			return false
		}

		return true
	}
	assert.NoError(t, db.Search(1, values, nil, cb), "TestAdd failed")
	assert.Equal(t, 1, c, "TestAdd failed")
}
