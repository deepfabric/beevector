package storage

import (
	"fmt"

	bhmetapb "github.com/deepfabric/beehive/pb/metapb"
	"github.com/deepfabric/beevector/pkg/db"
	"github.com/deepfabric/beevector/pkg/pb/metapb"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/protoc"
)

func (s *storage) Created(shard bhmetapb.Shard) {
	if _, ok := s.dbs.Load(shard.ID); ok {
		log.Fatalf("BUG: db %d already created", shard.ID)
	}

	metadata := &metapb.DB{}
	protoc.MustUnmarshal(metadata, shard.Data)

	db, err := s.cfg.dbCreateFunc(fmt.Sprintf("%s/dbs-%d", s.cfg.DataPath, shard.ID),
		s.cfg.Dim, metadata.State)
	if err != nil {
		log.Fatalf("db %d created failed with %+v",
			shard.ID,
			err)
	}

	s.dbs.Store(shard.ID, db)

	log.Infof("db %d created with state %s",
		shard.ID,
		metadata.State.String())
}

func (s *storage) Destory(shard bhmetapb.Shard) {
	if v, ok := s.dbs.Load(shard.ID); ok {
		err := v.(db.DB).Destroy()
		if err != nil {
			log.Fatalf("db %d destory failed with %+v",
				shard.ID,
				err)
		}
	}
}

func (s *storage) Splited(shard bhmetapb.Shard) {

}

func (s *storage) BecomeLeader(shard bhmetapb.Shard) {

}

func (s *storage) BecomeFollower(shard bhmetapb.Shard) {

}

func (s *storage) customInitShardCreate() []bhmetapb.Shard {
	return []bhmetapb.Shard{
		{
			Start:           goetty.Uint64ToBytes(0),
			DisableSplit:    true,
			DataAppendToMsg: true,
			Data: protoc.MustMarshal(&metapb.DB{
				State: metapb.RWU,
			}),
		},
	}
}

func (s *storage) customSplitCheck(shard bhmetapb.Shard) ([]byte, bool) {
	db := s.mustLoadDB(shard.ID)
	if db.State() == metapb.RU {
		return nil, false
	}

	total, err := db.Records()
	if err != nil {
		log.Errorf("db %d check split failed with %+v",
			shard.ID,
			err)
		return nil, false
	}

	if total >= s.cfg.MaxRecords {
		return goetty.Uint64ToBytes(total), true
	}

	return nil, false
}

func (s *storage) customSplitCompleted(old *bhmetapb.Shard, new *bhmetapb.Shard) {
	s.mustLoadDB(old.ID).UpdateState(metapb.RU)
	old.Data = protoc.MustMarshal(&metapb.DB{
		State: metapb.RU,
	})

	new.Data = protoc.MustMarshal(&metapb.DB{
		State: metapb.RWU,
	})
}

func (s *storage) customSnapshotDataCreate(path string, shard bhmetapb.Shard) error {
	db := s.mustLoadDB(shard.ID)
	return db.CreateSnap(path)
}

func (s *storage) customSnapshotDataApply(path string, shard bhmetapb.Shard) error {
	db := s.mustLoadDB(shard.ID)
	return db.ApplySnap(path)
}

func (s *storage) customCanReadLocalFunc(shard bhmetapb.Shard) bool {
	return s.mustLoadDB(shard.ID).State() == metapb.RU
}
