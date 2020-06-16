package storage

import (
	"log"

	"github.com/deepfabric/beehive/pb"
	bhmetapb "github.com/deepfabric/beehive/pb/metapb"
	"github.com/deepfabric/beehive/pb/raftcmdpb"
	"github.com/deepfabric/beevector/pkg/pb/metapb"
	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/util/protoc"
)

func (s *storage) add(shard bhmetapb.Shard, req *raftcmdpb.Request, attrs map[string]interface{}) (uint64, int64, *raftcmdpb.Response) {
	resp := pb.AcquireResponse()
	add := &rpcpb.AddRequest{}
	protoc.MustUnmarshal(add, resp.Value)

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

func (s *storage) search(shard bhmetapb.Shard, req *raftcmdpb.Request, attrs map[string]interface{}) *raftcmdpb.Response {
	resp := pb.AcquireResponse()
	search := &rpcpb.SearchRequest{}
	protoc.MustUnmarshal(search, resp.Value)

	db := s.mustLoadDB(shard.ID)
	scores, xids, err := db.Search(search.Xqs, search.Topks, search.Bitmaps)
	if err != nil {
		log.Fatalf("shard %d search failed with %+v",
			shard.ID,
			err)
	}

	searchResp := &rpcpb.SearchResponse{}
	searchResp.Scores = scores
	searchResp.Xids = xids

	if req.LastBroadcast && db.State() != metapb.RWU {
		resp.ContinueBroadcast = true
	}

	resp.Value = protoc.MustMarshal(searchResp)
	return resp
}
