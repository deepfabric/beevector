package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"time"
	"unsafe"

	"github.com/deepfabric/beevector/pkg/sdk"
	"github.com/deepfabric/beevector/pkg/util"
)

var (
	addr  = flag.String("addr", "127.0.0.1:8081", "address")
	batch = flag.Int64("b", 10000, "batch")
	total = flag.Int64("n", 1000000, "total")
	dim   = flag.Int64("dim", 512, "dim")
	file  = flag.String("file", "/data/sift1M/sift_base.fvecs", "import file")
)

func main() {
	flag.Parse()

	cli := sdk.NewClient([]string{*addr})

	n := (*dim) / 128
	destXqs := make([]float32, 0, (*dim)*(*batch))
	var xids []int64
	completed := int64(0)
	cb := func(xqs []float32, c int64) {
		destXqs = destXqs[:0]

		for i := int64(0); i < c; i++ {
			completed++
			xids = append(xids, int64(util.EncodeXID(uint64(n), uint64(n))))

			v := xqs[128*i : 128*(i+1)]
			for j := int64(0); j < n; j++ {
				destXqs = append(destXqs, v...)
			}
		}

		normalizeVec(int(*dim), destXqs)
		for {
			err := cli.Add(destXqs, xids)
			if err == nil {
				break
			}

			time.Sleep(time.Second)
		}

		xids = xids[:0]
		log.Printf("%d completed", completed)
	}

	for {
		err := read(*file, *batch, 128, cb)
		if err != nil {
			log.Fatalf("import file %s failed with %+v", *file, err)
		}

		if n >= *total {
			break
		}
	}

	log.Printf("all completed")
}

func read(file string, batch, dim int64, cb func([]float32, int64)) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	buf := make([]byte, 4, 4)
	total := stat.Size()

	if total%(4*dim+4) != 0 {
		return fmt.Errorf("weird file size dim %d, %d", dim, total)
	}

	n := total / (4*dim + 4)
	log.Printf("total %d records", n)

	data := make([]float32, 0, dim*batch)
	buf = make([]byte, batch*(4*dim+4), batch*(4*dim+4))

	for {
		readed, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if readed == 0 {
			return nil
		}

		c := int64(readed) / (4*dim + 4)
		for i := int64(0); i < c; i++ {
			for j := int64(0); j < dim; j++ {
				data = append(data, *(*float32)(unsafe.Pointer(&buf[j*4+4])))
			}
		}
		cb(data, c)
		data = data[:0]
	}
}

func normalizeVec(d int, v []float32) {
	var norm float64
	for i := 0; i < d; i++ {
		norm += float64(v[i]) * float64(v[i])
	}
	norm = math.Sqrt(norm)
	for i := 0; i < d; i++ {
		v[i] = float32(float64(v[i]) / norm)
	}
}
