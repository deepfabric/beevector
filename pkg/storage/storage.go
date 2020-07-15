package storage

import (
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
	AsyncBroadcastCommand(interface{}, func(interface{}, [][]byte, error), interface{}, bool)
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
	metadataStorage bhstorage.MetadataStorage,
	dataStorages []bhstorage.DataStorage) (Storage, error) {
	return NewStorageWithOptions(cfg, metadataStorage, dataStorages)
}

// NewStorageWithOptions returns a beehive request handler
func NewStorageWithOptions(cfg Cfg,
	metadataStorage bhstorage.MetadataStorage,
	dataStorages []bhstorage.DataStorage, opts ...raftstore.Option) (Storage, error) {

	if cfg.dbCreateFunc == nil {
		cfg.dbCreateFunc = db.NewVectoDB
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
	opts = append(opts, raftstore.WithMaxProposalBytes(32*1024*1024))
	opts = append(opts, raftstore.WithReadBatchFunc(s.readBatch))

	store, err := beehive.CreateRaftStoreFromFile(cfg.DataPath,
		metadataStorage,
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
	return nil
}

func (s *storage) AsyncExecCommand(cmd interface{}, cb func(interface{}, []byte, error), arg interface{}) {
	s.app.AsyncExecWithTimeout(cmd, cb, defaultRPCTimeout, arg)
}

func (s *storage) AsyncBroadcastCommand(cmd interface{}, cb func(interface{}, [][]byte, error), arg interface{}, mustLeader bool) {
	s.app.AsyncBroadcast(cmd, 0, cb, defaultRPCTimeout, arg, mustLeader)
}

func (s *storage) Close() {
	s.runner.Stop()
	s.app.Stop()
	s.store.Stop()
}

func (s *storage) initHandleFuncs() {
	s.AddWriteFunc("add", uint64(rpcpb.Add), s.add)
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

func (s *storage) readBatch() raftstore.CommandReadBatch {
	return newBatchReader(s)
}
