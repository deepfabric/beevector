package db

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/deepfabric/beehive/util"
	"github.com/deepfabric/beevector/pkg/pb/metapb"
	"github.com/fagongzi/log"
	"github.com/infinivision/vectodb"
)

var (
	errDestoried = errors.New("vectodb was already destoried")
)

type vdb struct {
	sync.RWMutex

	destoried bool
	id        uint64
	name      string
	path      string
	dbPath    string
	dim       int
	db        *vectodb.VectoDB
	state     metapb.DBState
}

// NewVectoDB create a vectordb
func NewVectoDB(path string, id uint64, dim int, state metapb.DBState) (DB, error) {
	name := fmt.Sprintf("dbs-%d", id)
	dbPath := fmt.Sprintf("%s/%s", path, name)

	db, err := vectodb.NewVectoDB(dbPath, dim)
	if err != nil {
		return nil, err
	}

	return &vdb{
		state:  state,
		path:   path,
		dbPath: dbPath,
		dim:    dim,
		name:   name,
		id:     id,
		db:     db,
	}, nil
}

func (v *vdb) State() metapb.DBState {
	v.RLock()
	defer v.RUnlock()

	return v.state
}

func (v *vdb) UpdateState(state metapb.DBState) {
	v.Lock()
	defer v.Unlock()

	v.state = state
}

func (v *vdb) Add(xbs []float32, xids []int64) error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return errDestoried
	}

	return v.db.AddWithIds(xbs, xids)
}

func (v *vdb) Search(topk int, xqs []float32, bitmaps [][]byte, handler func(int, int, int, float32, int64) bool, topVectors bool) error {
	v.RLock()
	defer v.RUnlock()

	if v.destoried {
		return errDestoried
	}

	n := len(xqs) / v.dim
	s := time.Now()
	log.Infof("start query %d requests from shard %d",
		n,
		v.id)
	values, err := v.db.Search(topk, topVectors, xqs, bitmaps)
	e := time.Now()
	log.Infof("end query %d requests from shard %d, cost %f, err %+v",
		n,
		v.id,
		e.Sub(s).Seconds(),
		err)

	if err != nil {
		return err
	}

	for i, scores := range values {
		n := len(scores)
		for j, score := range scores {
			if !handler(i, j, n, score.Score, score.Xid) {
				break
			}
		}
	}

	return nil
}

func (v *vdb) Destroy() error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return nil
	}

	v.destoried = true
	return v.db.Destroy()
}

func (v *vdb) Clean() error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return errDestoried
	}

	v.db.Reset()
	return nil
}

func (v *vdb) Records() (uint64, error) {
	v.RLock()
	defer v.RUnlock()

	if v.destoried {
		return 0, errDestoried
	}

	total, err := v.db.GetTotal()
	if err != nil {
		return 0, err
	}

	return uint64(total), nil

}

func (v *vdb) CreateSnap(path string) error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return errDestoried
	}

	file := fmt.Sprintf("%s.gz", v.dbPath)
	os.Remove(file)

	err := util.GZIP(v.dbPath)
	if err != nil {

		return err
	}

	return os.Rename(file, fmt.Sprintf("%s/%s.gz", path, v.name))
}

func (v *vdb) ApplySnap(path string) error {
	v.Lock()
	defer v.Unlock()

	if v.destoried {
		return errDestoried
	}

	err := os.RemoveAll(v.dbPath)
	if err != nil {
		return err
	}

	file := fmt.Sprintf("%s/%s.gz", path, v.name)
	err = util.UnGZIP(file, v.path)
	if err != nil {
		return err
	}

	os.RemoveAll(file)
	return v.resetDB()
}

func (v *vdb) resetDB() error {
	if v.db != nil {
		err := v.db.Destroy()
		if err != nil {
			return err
		}
	}

	db, err := vectodb.NewVectoDB(v.dbPath, v.dim)
	if err != nil {
		return err
	}

	total, err := db.GetTotal()
	if err != nil {
		return err
	}

	v.db = db
	v.destoried = false
	log.Infof("db %d reset to %d records",
		v.id,
		total)
	return nil
}
