package sdk

import (
	"sync"

	"github.com/deepfabric/beevector/pkg/api"
	"github.com/fagongzi/goetty"
)

// Client beevector sdk
type Client interface {
	// Add add vectors with xids
	Add(xbs []float32, xids []int64) error
	// Search search topk with xb and bitmaps
	Search(xqs []float32, topks []int64, bitmaps [][]byte) (scores []float32, xids []int64, err error)
}

type client struct {
	sync.RWMutex

	id    uint64
	conns []goetty.IOSession
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

func (c *client) Search(xqs []float32, topks []int64, bitmaps [][]byte) ([]float32, []int64, error) {
	return nil, nil, nil
}
