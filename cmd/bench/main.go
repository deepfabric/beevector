package main

import (
	"flag"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/deepfabric/beevector/pkg/sdk"
)

var (
	addrs = flag.String("addr", "127.0.0.1:8081", "addrs")
	total = flag.Uint64("n", 1, "total")
)

func main() {
	flag.Parse()

	c := sdk.NewClient(strings.Split(*addrs, ",")...)
	time.Sleep(time.Second * 2)

	w := make(chan struct{})
	n := uint64(0)
	s := time.Now().UnixNano()
	for i := uint64(0); i < *total; i++ {
		c.AsyncSearch(1, make([]float32, 128, 128), nil, func(xqs []float32, xids []int64, err error) {
			if atomic.AddUint64(&n, 1) == *total {
				close(w)
			}
		})
	}

	<-w
	e := time.Now().UnixNano()
	log.Printf("%d ms", (e-s)/1000000)
}
