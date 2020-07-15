package storage

import (
	"time"

	"github.com/deepfabric/beevector/pkg/db"
	"github.com/deepfabric/beevector/pkg/pb/metapb"
)

// Cfg cfg
type Cfg struct {
	DataPath             string
	MaxRecords           uint64
	Dim                  int
	LimitRebuildIndex    int
	RebuildIndexInterval time.Duration

	dbCreateFunc func(string, uint64, int, metapb.DBState) (db.DB, error)
}
