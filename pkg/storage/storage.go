package storage

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/deepfabric/beehive"
	"github.com/deepfabric/beehive/pb/raftcmdpb"
	"github.com/deepfabric/beehive/raftstore"
	"github.com/deepfabric/beehive/server"
	bhstorage "github.com/deepfabric/beehive/storage"
	"github.com/deepfabric/beevector/pkg/db"
	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/protoc"
	"github.com/fagongzi/util/task"
)

var (
	defaultRPCTimeout = time.Second * 30
)

// Storage beevector storage
type Storage interface {
	Start() error
	Close()

	AsyncExecCommand(interface{}, func(interface{}, []byte, error), interface{})
}

type storage struct {
	cfg Cfg

	runner *task.Runner
	dbs    sync.Map
	app    *server.Application
	store  raftstore.Store
}

// NewStorage returns a beehive request handler
func NewStorage(cfg Cfg,
	metadataStorages []bhstorage.MetadataStorage,
	dataStorages []bhstorage.DataStorage) (Storage, error) {
	return NewStorageWithOptions(cfg, metadataStorages, dataStorages)
}

// NewStorageWithOptions returns a beehive request handler
func NewStorageWithOptions(cfg Cfg,
	metadataStorages []bhstorage.MetadataStorage,
	dataStorages []bhstorage.DataStorage, opts ...raftstore.Option) (Storage, error) {

	if cfg.dbCreateFunc == nil {
		cfg.dbCreateFunc = db.NewDB
	}

	s := &storage{
		cfg:    cfg,
		runner: task.NewRunner(),
	}
	opts = append(opts, raftstore.WithShardStateAware(s))
	opts = append(opts, raftstore.WithCustomInitShardCreateFunc(s.customInitShardCreate))
	opts = append(opts, raftstore.WithCustomSplitCheckFunc(s.customSplitCheck))
	opts = append(opts, raftstore.WithCustomSplitCompletedFunc(s.customSplitCompleted))
	opts = append(opts, raftstore.WithCustomCanReadLocalFunc(s.customCanReadLocalFunc))

	store, err := beehive.CreateRaftStoreFromFile(cfg.DataPath,
		metadataStorages,
		dataStorages,
		opts...)
	if err != nil {
		return nil, err
	}

	s.store = store
	s.app = server.NewApplication(server.Cfg{
		Store:          s.store,
		Handler:        s,
		ExternalServer: true,
	})

	s.initHandleFuncs()
	err = s.app.Start()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *storage) Start() error {
	s.runner.RunCancelableTask(s.rebuildIndex)
	return nil
}

func (s *storage) AsyncExecCommand(cmd interface{}, cb func(interface{}, []byte, error), arg interface{}) {
	s.app.AsyncExecWithTimeout(cmd, cb, defaultRPCTimeout, arg)
}

func (s *storage) Close() {
	s.runner.Stop()
	s.app.Stop()
	s.store.Stop()
}

func (s *storage) rebuildIndex(ctx context.Context) {
	log.Infof("rebuild db index task started")
	timer := time.NewTicker(s.cfg.RebuildIndexInterval)
	defer timer.Stop()

	var builds []db.DB
	alreadyRebuilds := make(map[interface{}]db.DB)
	for {
		select {
		case <-ctx.Done():
			log.Infof("rebuild db index task stopped")
			return
		case <-timer.C:
			c := 0
			builds = builds[:0]
			s.dbs.Range(func(key, value interface{}) bool {
				if _, ok := alreadyRebuilds[key]; !ok && len(builds) < s.cfg.LimitRebuildIndex {
					builds = append(builds, value.(db.DB))
					alreadyRebuilds[key] = value.(db.DB)
				}
				c++
				return true
			})

			if c > 0 {
				if len(builds) == 0 {
					for key := range alreadyRebuilds {
						delete(alreadyRebuilds, key)
					}
				} else {
					for _, db := range builds {
						db.UpdateIndex()
					}
				}
			}
		}
	}
}

func (s *storage) initHandleFuncs() {
	s.AddWriteFunc("add", uint64(rpcpb.Add), s.add)
	s.AddReadFunc("search", uint64(rpcpb.Search), s.search)
}

var (
	lastShardKey = goetty.Uint64ToBytes(math.MaxUint64)
)

func (s *storage) BuildRequest(req *raftcmdpb.Request, cmd interface{}) error {
	switch cmd.(type) {
	case *rpcpb.AddRequest:
		msg := cmd.(*rpcpb.AddRequest)
		req.Key = lastShardKey
		req.CustemType = uint64(rpcpb.Add)
		req.Type = raftcmdpb.Write
		req.Cmd = protoc.MustMarshal(msg)
	case *rpcpb.SearchRequest:
		msg := cmd.(*rpcpb.SearchRequest)
		req.CustemType = uint64(rpcpb.Search)
		req.Type = raftcmdpb.Read
		req.Cmd = protoc.MustMarshal(msg)
	default:
		log.Fatalf("not support request %+v(%+T)", cmd, cmd)
	}

	return nil
}

func (s *storage) Codec() (goetty.Decoder, goetty.Encoder) {
	return nil, nil
}

func (s *storage) AddReadFunc(cmd string, cmdType uint64, cb raftstore.ReadCommandFunc) {
	s.store.RegisterReadFunc(cmdType, cb)
}

func (s *storage) AddWriteFunc(cmd string, cmdType uint64, cb raftstore.WriteCommandFunc) {
	s.store.RegisterWriteFunc(cmdType, cb)
}

func (s *storage) AddLocalFunc(cmd string, cmdType uint64, cb raftstore.LocalCommandFunc) {
	s.store.RegisterLocalFunc(cmdType, cb)
}

func (s *storage) mustLoadDB(id uint64) db.DB {
	v, ok := s.dbs.Load(id)
	if !ok {
		log.Fatalf("BUG: missing db %d")
	}

	return v.(db.DB)
}
