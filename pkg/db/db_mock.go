package db

import (
	"fmt"
	"sync"
	"time"

	"github.com/deepfabric/beevector/pkg/pb/metapb"
)

var (
	vv = time.Second * 8 / 1000
)

type mockDB struct {
	sync.RWMutex

	destoried bool
	id        uint64
	name      string
	path      string
	dbPath    string
	dim       int
	state     metapb.DBState
}

// NewMockDB create a vectordb
func NewMockDB(path string, id uint64, dim int, state metapb.DBState) (DB, error) {
	name := fmt.Sprintf("dbs-%d", id)
	dbPath := fmt.Sprintf("%s/%s", path, name)

	return &mockDB{
		state:  state,
		path:   path,
		dbPath: dbPath,
		dim:    dim,
		name:   name,
		id:     id,
	}, nil
}

func (v *mockDB) State() metapb.DBState {
	v.RLock()
	defer v.RUnlock()

	return v.state
}

func (v *mockDB) UpdateState(state metapb.DBState) {
	v.Lock()
	defer v.Unlock()

	v.state = state
}

func (v *mockDB) Add(xbs []float32, xids []int64) error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return errDestoried
	}

	return nil
}

func (v *mockDB) Search(topk int, xqs []float32, bitmaps [][]byte, handler func(int, int, int, float32, int64) bool, topVectors bool) error {
	v.RLock()
	defer v.RUnlock()

	if v.destoried {
		return errDestoried
	}

	n := len(xqs) / v.dim
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if !handler(i, j, n, 0, 1) {
				break
			}
		}
	}

	time.Sleep(time.Duration(n) * vv)
	return nil
}

func (v *mockDB) Destroy() error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return nil
	}

	v.destoried = true
	return nil
}

func (v *mockDB) Clean() error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return errDestoried
	}

	return nil
}

func (v *mockDB) Records() (uint64, error) {
	v.RLock()
	defer v.RUnlock()

	if v.destoried {
		return 0, errDestoried
	}

	return 900000, nil

}

func (v *mockDB) CreateSnap(path string) error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return errDestoried
	}

	return nil
}

func (v *mockDB) ApplySnap(path string) error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return errDestoried
	}

	return nil
}
