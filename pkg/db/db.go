package db

import (
	"github.com/deepfabric/beevector/pkg/pb/metapb"
)

// DB vector db interface
type DB interface {
	State() metapb.DBState
	UpdateState(metapb.DBState)

	// Add add vectors with xids
	Add(xbs []float32, xids []int64) error
	// Search search topk with xb and bitmaps
	Search(xqs []float32, topks []int64, bitmaps [][]byte) (scores []float32, xids []int64, err error)
	UpdateIndex() error

	Destroy() error
	Clean() error
	Records() (uint64, error)
	CreateSnap(path string) error
	ApplySnap(path string) error

	// just for test
	Set([]byte, []byte) error
	Get([]byte) ([]byte, error)
	RebuildTimes() int
}

// NewDB create a db
func NewDB(path string, dim int, state metapb.DBState) (DB, error) {
	return nil, nil
}
