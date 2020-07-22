package main

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/deepfabric/beevector/pkg/sdk"
)

func main() {
	c := sdk.NewClient("127.0.0.1:8081")

	w := make(chan struct{})
	n := uint64(0)
	s := time.Now().UnixNano()
	for i := 0; i < 1; i++ {
		c.AsyncSearch(1, make([]float32, 128, 128), nil, func(xqs []float32, xids []int64, err error) {
			if atomic.AddUint64(&n, 1) == 1 {
				close(w)
			}
		})
	}

	<-w
	e := time.Now().UnixNano()
	log.Printf("%d ms", (e-s)/1000000)
}
