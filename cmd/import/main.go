package main

import (
	"log"
	"time"

	"github.com/deepfabric/beevector/pkg/sdk"
)

func main() {
	c := sdk.NewClient("172.24.96.1:8091")
	var xids []int64
	var xqs []float32
	values := make([]float32, 128, 128)
	for i := int64(1); i <= 5000000; i++ {
		xqs = append(xqs, values...)
		xids = append(xids, i)

		if i%1000 == 0 {
			for {
				err := c.Add(xqs, xids)
				if err == nil {
					break
				}

				time.Sleep(time.Second)
			}

			log.Printf("%d completed", i)
			xids = xids[:0]
			xqs = xqs[:0]
		}
	}
}
