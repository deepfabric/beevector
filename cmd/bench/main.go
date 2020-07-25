package main

import (
	"flag"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deepfabric/beevector/pkg/sdk"
	"github.com/fagongzi/log"
)

var (
	con   = flag.Uint64("c", 1, "The clients.")
	cn    = flag.Uint64("cn", 64, "The concurrency per client.")
	top   = flag.Int64("top", 1, "Top k.")
	dim   = flag.Int64("dim", 128, "dim")
	num   = flag.Uint64("n", 0, "The total number.")
	addrs = flag.String("addr", "172.19.0.106:8091,172.19.0.107:8091,172.19.0.108:8091,172.19.0.103:8091,172.19.0.101:8091", "addrs")
)

func main() {
	flag.Parse()
	log.InitLog()
	sdk.SetLogger(log.NewLoggerWithPrefix("sdk"))

	gCount := *con
	total := *num
	if total < 0 {
		total = 0
	}

	ready := make(chan struct{}, gCount)
	complate := &sync.WaitGroup{}
	wg := &sync.WaitGroup{}

	countPerG := total / gCount

	c := sdk.NewClient(strings.Split(*addrs, ","),
		sdk.WithBatch(int64(*cn)),
		sdk.WithTimeout(time.Minute*5))
	ans := newAnalysis()

	var index uint64
	for index = uint64(0); index < gCount; index++ {
		start := index * countPerG
		end := (index + 1) * countPerG
		if index == gCount-1 {
			end = total
		}

		wg.Add(1)
		complate.Add(1)
		go startG(end-start, wg, complate, ready, ans, c)
	}

	wg.Wait()
	ans.start()

	for index = 0; index < gCount; index++ {
		ready <- struct{}{}
	}

	go func() {
		for {
			ans.print()
			time.Sleep(time.Second * 1)
		}
	}()

	complate.Wait()
	ans.print()

}

func startG(total uint64, wg, complate *sync.WaitGroup, ready chan struct{}, ans *analysis, c sdk.Client) {
	if total <= 0 {
		total = math.MaxInt64
	}

	wg.Done()
	<-ready

	xqs := make([]float32, *dim, *dim)
	start := time.Now()
	for index := uint64(0); index < total; index += *cn {
		for k := int64(0); k < *dim; k++ {
			xqs[k] = float32(total) / float32(k+1)
		}

		n := uint64(0)
		w := make(chan struct{})
		for i := uint64(0); i < *cn; i++ {
			c.AsyncSearch(*top, xqs, nil, func(xqs []float32, xids []int64, err error) {
				if err != nil {
					log.Fatalf("exit: %+v", err)
				}

				if atomic.AddUint64(&n, 1) == *cn {
					close(w)
				}
			})
		}

		s := time.Now()
		ans.incrSent(int64(*cn))
		<-w
		ans.incrRecv(int64(*cn), time.Now().Sub(s).Nanoseconds())
	}

	end := time.Now()
	log.Infof("%s sent %d reqs\n", end.Sub(start), total)
}

type analysis struct {
	sync.RWMutex
	startAt                            time.Time
	recv, sent                         int64
	avgLatency, maxLatency, minLatency int64
	totalCost, prevCost                int64
}

func newAnalysis() *analysis {
	return &analysis{}
}

func (a *analysis) setLatency(latency int64) {
	if a.minLatency == 0 || a.minLatency > latency {
		a.minLatency = latency
	}

	if a.maxLatency == 0 || a.maxLatency < latency {
		a.maxLatency = latency
	}

	a.totalCost += latency
	a.avgLatency = a.totalCost / a.recv
}

func (a *analysis) reset() {
	a.Lock()
	a.maxLatency = 0
	a.minLatency = 0
	a.Unlock()
}

func (a *analysis) start() {
	a.startAt = time.Now()
}

func (a *analysis) incrRecv(n, latency int64) {
	a.Lock()
	a.recv += n
	a.setLatency(latency)
	a.Unlock()
}

func (a *analysis) incrSent(n int64) {
	a.Lock()
	a.sent += n
	a.Unlock()
}

func (a *analysis) print() {
	a.Lock()
	defer a.Unlock()

	pass := int64(time.Now().Sub(a.startAt).Seconds())
	if pass == 0 {
		return
	}

	fmt.Printf("[%d, %d, %d](%d s), tps: <%d>/s, avg: %s, min: %s, max: %s \n",
		a.sent,
		a.recv,
		(a.sent - a.recv),
		pass,
		a.recv/pass,
		time.Duration(a.avgLatency),
		time.Duration(a.minLatency),
		time.Duration(a.maxLatency))
	a.prevCost = a.totalCost
}
