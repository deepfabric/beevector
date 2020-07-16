package storage

import (
	"github.com/deepfabric/beehive/pb"
	bhmetapb "github.com/deepfabric/beehive/pb/metapb"
	"github.com/deepfabric/beehive/pb/raftcmdpb"
	"github.com/deepfabric/beevector/pkg/pb/metapb"
	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/protoc"
)

func (s *storage) add(shard bhmetapb.Shard, req *raftcmdpb.Request, attrs map[string]interface{}) (uint64, int64, *raftcmdpb.Response) {
	resp := pb.AcquireResponse()
	add := &rpcpb.AddRequest{}
	protoc.MustUnmarshal(add, req.Cmd)

	db := s.mustLoadDB(shard.ID)
	if db.State() != metapb.RWU {
		resp.Stale = true
		return 0, 0, resp
	}

	err := db.Add(add.Xbs, add.Xids)
	if err != nil {
		log.Fatalf("shard %d add failed with %+v",
			shard.ID,
			err)
	}

	return 0, 0, resp
}
