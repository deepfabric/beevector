package pb

import (
	"sync"

	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
)

var (
	searchResponse = sync.Pool{
		New: func() interface{} {
			return &rpcpb.SearchResponse{}
		},
	}
)

// AcquireSearchResponse returns value from pool
func AcquireSearchResponse() *rpcpb.SearchResponse {
	value := searchResponse.Get().(*rpcpb.SearchResponse)
	return value
}

// ReleaseSearchResponse release value to pool
func ReleaseSearchResponse(value *rpcpb.SearchResponse) {
	value.Reset()
	searchResponse.Put(value)
}
