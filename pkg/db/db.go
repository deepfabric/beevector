package db

import (
	"github.com/deepfabric/beevector/pkg/pb/metapb"
)

// DB vector db interface
type DB interface {
	State() metapb.DBState
	UpdateState(metapb.DBState)
	Add(xbs []float32, xids []int64) error
	Search(topk int, xqs []float32, bitmaps [][]byte, handler func(int, int, int, float32, int64) bool) error
	Destroy() error
	Clean() error
	Records() (uint64, error)
	CreateSnap(path string) error
	ApplySnap(path string) error
}
