package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	beehiveStorage "github.com/deepfabric/beehive/storage"
	"github.com/deepfabric/beehive/storage/badger"
	"github.com/deepfabric/beevector/pkg/storage"
	"github.com/deepfabric/beevector/pkg/util"
	"github.com/deepfabric/prophet"
	"github.com/fagongzi/log"
)

var (
	addr      = flag.String("addr", "127.0.0.1:8081", "beehive api address")
	addrPPROF = flag.String("addr-pprof", "", "pprof")
	data      = flag.String("data", "", "data path")
	stores    = flag.Uint64("stores", 1, "Number of store count.")
	wait      = flag.Int("wait", 0, "wait")
	version   = flag.Bool("version", false, "Show version info")

	// about vectordb
	dim                     = flag.Int("dim", 512, "VectorDB: dim")
	limitRebuildIndex       = flag.Int("limit-rebuild", 2, "VectorDB: Max number of vectordbs to rebuild index")
	rebuildIndexIntervalSec = flag.Int("rebuild-interval", 10, "VectorDB(sec): rebuild index interval")
)

var (
	stopping = false
)

func main() {
	flag.Parse()
	if *version {
		util.PrintVersion()
		os.Exit(0)
	}

	log.InitLog()
	prophet.SetLogger(log.NewLoggerWithPrefix("[prophet]"))

	if *wait > 0 {
		time.Sleep(time.Second * time.Duration(*wait))
	}

	if *addrPPROF != "" {
		runtime.SetBlockProfileRate(1)
		go func() {
			log.Errorf("start pprof failed, errors:\n%+v",
				http.ListenAndServe(*addrPPROF, nil))
		}()
	}

	cfg := storage.Cfg{
		DataPath:             *data,
		Dim:                  *dim,
		LimitRebuildIndex:    *limitRebuildIndex,
		RebuildIndexInterval: time.Second * time.Duration(*rebuildIndexIntervalSec),
	}

	var metaStores []beehiveStorage.MetadataStorage
	var dataStores []beehiveStorage.DataStorage

	for i := uint64(0); i < *stores; i++ {
		store, err := badger.NewStorage(filepath.Join(*data, fmt.Sprintf("badger-%d", i)))
		if err != nil {
			log.Fatalf("create badger failed with %+v", err)
		}

		metaStores = append(metaStores, store)
		dataStores = append(dataStores, store)
	}

	store, err := storage.NewStorage(cfg, metaStores, dataStores)
	if err != nil {
		log.Fatalf("create storage failed with %+v", err)
	}

	go store.Start()

	sc := make(chan os.Signal, 2)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	for {
		sig := <-sc

		if !stopping {
			stopping = true
			go func() {
				store.Close()
				log.Infof("exit: signal=<%d>.", sig)
				switch sig {
				case syscall.SIGTERM:
					log.Infof("exit: bye :-).")
					os.Exit(0)
				default:
					log.Infof("exit: bye :-(.")
					os.Exit(1)
				}
			}()
			continue
		}

		log.Infof("exit: bye :-).")
		os.Exit(0)
	}
}
