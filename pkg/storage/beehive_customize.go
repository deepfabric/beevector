package storage

import (
	bhmetapb "github.com/deepfabric/beehive/pb/metapb"
	"github.com/deepfabric/beevector/pkg/db"
	"github.com/deepfabric/beevector/pkg/pb/metapb"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/protoc"
)

func (s *storage) Created(shard bhmetapb.Shard) {
	log.Infof("ready to create db %d",
		shard.ID)

	if _, ok := s.dbs.Load(shard.ID); ok {
		log.Fatalf("BUG: db %d already created", shard.ID)
	}

	metadata := &metapb.DB{}
	protoc.MustUnmarshal(metadata, shard.Data)

	db, err := s.cfg.dbCreateFunc(s.cfg.DataPath, shard.ID,
		s.cfg.Dim, metadata.State)
	if err != nil {
		log.Fatalf("db %d created failed with %+v",
			shard.ID,
			err)
	}

	s.dbs.Store(shard.ID, db)

	total, err := db.Records()
	if err != nil {
		log.Fatalf("db %d created failed with %+v",
			shard.ID,
			err)
	}

	log.Infof("db %d created with state %s, %d records",
		shard.ID,
		metadata.State.String(),
		total)
}

func (s *storage) Destory(shard bhmetapb.Shard) {
	log.Infof("ready to remove db %d",
		shard.ID)

	if v, ok := s.dbs.Load(shard.ID); ok {
		s.dbs.Delete(shard.ID)
		err := v.(db.DB).Destroy()
		if err != nil {
			log.Fatalf("db %d destory failed with %+v",
				shard.ID,
				err)
		}
	}

	log.Infof("db %d removed",
		shard.ID)
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
		n := uint64(0)
		if len(shard.Start) > 0 {
			n = goetty.Byte2UInt64(shard.Start)
		}

		return goetty.Uint64ToBytes(total + n), true
	}

	return nil, false
}

func (s *storage) customSplitCompleted(old *bhmetapb.Shard, new *bhmetapb.Shard) {
	db := s.mustLoadDB(old.ID)
	db.UpdateState(metapb.RU)
	old.Data = protoc.MustMarshal(&metapb.DB{
		State: metapb.RU,
	})

	new.Start = goetty.Uint64ToBytes(goetty.Byte2UInt64(old.End) + s.cfg.MaxRecords)
	new.Data = protoc.MustMarshal(&metapb.DB{
		State: metapb.RWU,
	})

	total, err := db.Records()
	if err != nil {
		log.Fatalf("db %d check total records failed with %+v",
			old.ID,
			err)
	}
	log.Infof("db %d split at %d records",
		old.ID,
		total)
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
	value := s.mustLoadDB(shard.ID).State() == metapb.RU
	return value
}
