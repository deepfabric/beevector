package sdk

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deepfabric/beehive/util"
	"github.com/deepfabric/beevector/pkg/api"
	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/task"
)

const (
	batch          = int64(64)
	defaultTimeout = time.Second * 30
)

var (
	// ErrTimeout err timeout
	ErrTimeout = errors.New("Request timeout")
)

var (
	logger log.Logger
)

func init() {
	logger = log.NewLoggerWithPrefix("[beevector-sdk]")
}

// SetLogger set sdk logger
func SetLogger(l log.Logger) {
	logger = l
}

// Client beevector sdk
type Client interface {
	// Add add vectors with xids
	Add(xbs []float32, xids []int64) error
	// Search search topk with xb and bitmaps
	Search(topk int64, xq []float32, bitmap []byte) ([]float32, []int64, error)
}

type client struct {
	id    uint64
	op    uint64
	addrs []string
	conns []goetty.IOSession
	msgs  []*task.Queue

	ctxs sync.Map
}

// NewClient create a beevector client
func NewClient(addrs ...string) Client {
	logger.Infof("create client with beevector servers %+v", addrs)

	c := &client{}

	for _, addr := range addrs {
		c.addrs = append(addrs, addr)
		c.conns = append(c.conns, goetty.NewConnector(addr,
			goetty.WithClientDecoder(api.Decoder),
			goetty.WithClientEncoder(api.Encoder)))
		c.msgs = append(c.msgs, task.New(1024))
	}

	c.start()
	return c
}

func (c *client) Add(xbs []float32, xids []int64) error {
	req := &rpcpb.Request{}
	req.Type = rpcpb.Add
	req.Add.Xbs = xbs
	req.Add.Xids = xids

	resp, err := c.do(req)
	if err != nil {
		return err
	}

	if resp.Error.Error != "" {
		return errors.New(resp.Error.Error)
	}

	return nil
}

func (c *client) Search(topk int64, xq []float32, bitmap []byte) ([]float32, []int64, error) {
	req := &rpcpb.Request{}
	req.Type = rpcpb.Search
	req.Search.Xqs = xq
	req.Search.Topk = topk
	req.Search.Bitmaps = [][]byte{bitmap}

	resp, err := c.do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.Error.Error != "" {
		return nil, nil, errors.New(resp.Error.Error)
	}

	return resp.Search.Scores, resp.Search.Xids, nil
}

func (c *client) do(req *rpcpb.Request) (*rpcpb.Response, error) {
	req.ID = c.nextID()
	ctx := &ctx{
		req: req,
		c:   make(chan struct{}),
	}

	c.ctxs.Store(req.ID, ctx)
	c.addCtxToQueue(ctx, -1)
	util.DefaultTimeoutWheel().Schedule(defaultTimeout, c.timeout, req.ID)

	ctx.wait()
	return ctx.resp, ctx.err
}

func (c *client) nextID() uint64 {
	return atomic.AddUint64(&c.id, 1)
}

func (c *client) timeout(arg interface{}) {
	if v, ok := c.ctxs.Load(arg); ok {
		c.ctxs.Delete(arg)
		v.(*ctx).done(nil, ErrTimeout)
	}
}

func (c *client) start() {
	for idx := range c.conns {
		go c.writeLoop(idx)
		go c.readLoop(idx)
	}
}

func (c *client) writeLoop(idx int) {
	defer func() {
		if err := recover(); err != nil {
			logger.Infof("%s write loop failed with %+v, restart later",
				c.addrs[idx], err)
			go c.writeLoop(idx)
		}
	}()

	logger.Infof("%s write loop started", c.addrs[idx])

	q := c.msgs[idx]
	conn := c.conns[idx]
	items := make([]interface{}, batch, batch)

	for {
		n, err := q.Get(batch, items)
		if err != nil {
			logger.Fatalf("BUG: queue failed with %+v", err)
		}

		if !conn.IsConnected() {
			c.retry(items, n, idx)
			continue
		}

		for i := int64(0); i < n; i++ {
			req := items[i].(*ctx).req
			conn.Write(req)
			logger.Debugf("%s write request-%d",
				c.addrs[idx],
				req.ID)
		}

		err = conn.Flush()
		logger.Debugf("%s flush %d requests with error %+v",
			c.addrs[idx],
			n,
			err)

		if err != nil {
			c.retry(items, n, idx)
		}
	}
}

func (c *client) readLoop(idx int) {
	defer func() {
		if err := recover(); err != nil {
			logger.Infof("%s read loop failed with %+v, restart later",
				c.addrs[idx], err)
			go c.readLoop(idx)
		}
	}()

	logger.Infof("%s read loop started", c.addrs[idx])

	conn := c.conns[idx]
	for {
		if !conn.IsConnected() {
			_, err := conn.Connect()
			if err != nil {
				logger.Errorf("%s connect failed with %+v, retry after 5s",
					c.addrs[idx],
					err)
				time.Sleep(time.Second * 5)
				continue
			}
		}

		for {
			data, err := conn.Read()
			if err != nil {
				logger.Errorf("%s read failed with %+v",
					c.addrs[idx],
					err)
				conn.Close()
				break
			}

			logger.Debugf("%s read response %+v",
				c.addrs[idx],
				data)

			resp := data.(*rpcpb.Response)
			if v, ok := c.ctxs.Load(resp.ID); ok {
				c.ctxs.Delete(resp.ID)
				v.(*ctx).done(resp, nil)
			}
		}
	}
}

func (c *client) addCtxToQueue(ctx interface{}, exclude int) {
	if len(c.msgs) == 1 {
		c.msgs[0].Put(ctx)
		return
	}

	for {
		i := int(atomic.AddUint64(&c.op, 1) % uint64(len(c.msgs)))
		if i != exclude {
			c.msgs[i].Put(ctx)
			return
		}
	}
}

func (c *client) retry(items []interface{}, n int64, from int) {
	logger.Warningf("%s was disconnected, retry %d requests to other servers",
		c.addrs[from],
		n)

	for i := int64(0); i < n; i++ {
		c.addCtxToQueue(items[i], from)
	}
}

type ctx struct {
	state uint64
	req   *rpcpb.Request
	resp  *rpcpb.Response
	err   error
	c     chan struct{}
}

func (c *ctx) done(resp *rpcpb.Response, err error) {
	if atomic.CompareAndSwapUint64(&c.state, 0, 1) {
		logger.Debugf("request-%d responsed with %+v, error %+v",
			c.req.ID,
			resp,
			err)

		c.resp = resp
		c.err = err
		close(c.c)
	}
}

func (c *ctx) wait() {
	<-c.c
}
