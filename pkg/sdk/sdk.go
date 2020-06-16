package sdk

import (
	"sync"

	"github.com/deepfabric/beevector/pkg/api"
	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/util/task"
)

// Client beevector sdk
type Client interface {
	// Add add vectors with xids
	Add(xbs []float32, xids []int64) error
	// Search search topk with xb and bitmaps
	Search(xq []float32, topk int64, bitmap []byte) ([]float32, []int64, error)
}

type client struct {
	sync.RWMutex

	id    uint64
	conns []goetty.IOSession
	msgs  []*task.Queue
}

// NewClient create a beevector client
func NewClient(addrs ...string) Client {
	c := &client{}

	for _, addr := range addrs {
		c.conns = append(c.conns, goetty.NewConnector(addr,
			goetty.WithClientDecoder(api.Decoder),
			goetty.WithClientEncoder(api.Encoder)))
	}

	return c
}

func (c *client) Add(xbs []float32, xids []int64) error {
	return nil
}

func (c *client) Search(xq []float32, topk int64, bitmap []byte) ([]float32, []int64, error) {
	return nil, nil, nil
}

func (c *client) addToSend(req *rpcpb.Request) {

}
