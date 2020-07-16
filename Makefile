VERSION    = $(version)
ifeq ("$(VERSION)","")
	VERSION := "dev"
endif

VECTODB_VERSION    = $(vectodb_version)
ifeq ("$(VECTODB_VERSION)","")
	VECTODB_VERSION := "IndexFlatDisk"
endif

ROOT_DIR = $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))/
LD_GIT_COMMIT      = -X 'github.com/deepfabric/beevector/pkg/util.GitCommit=`git rev-parse --short HEAD`'
LD_BUILD_TIME      = -X 'github.com/deepfabric/beevector/pkg/util.BuildTime=`date +%FT%T%z`'
LD_GO_VERSION      = -X 'github.com/deepfabric/beevector/pkg/util.GoVersion=`go version`'
LD_BIN_VERSION     = -X 'github.com/deepfabric/beevector/pkg/util.Version=$(VERSION)'
LD_FLAGS = -ldflags "$(LD_GIT_COMMIT) $(LD_BUILD_TIME) $(LD_GO_VERSION) $(LD_BIN_VERSION) -w -s"

GOOS 		= linux
DIST_DIR 	= $(ROOT_DIR)dist/

.PHONY: clean_dist_dir
clean_dist_dir: ; $(info ======== clean distribute dir:)
	@rm -rf $(DIST_DIR)

.PHONY: dist_dir
dist_dir: clean_dist_dir; $(info ======== prepare distribute dir:)
	mkdir -p $(DIST_DIR)

.PHONY: beevector
beevector: dist_dir; $(info ======== compiled beevector)
	env GO111MODULE=off GOOS=$(GOOS) go build -o $(DIST_DIR)beevector $(LD_FLAGS) $(ROOT_DIR)cmd/beevector/*.go

.PHONY: checker
checker: dist_dir; $(info ======== compiled checker)
	env GO111MODULE=off GOOS=$(GOOS) go build -o $(DIST_DIR)checker $(LD_FLAGS) $(ROOT_DIR)cmd/checker/*.go

.PHONY: docker
docker: dist_dir; $(info ======== compiled beevector docker)
	docker build -t deepfabric/beevector:$(VERSION) -f Dockerfile .
	docker tag deepfabric/beevector:$(VERSION) deepfabric/beevector

.PHONY: runtime
runtime: dist_dir; $(info ======== make runtime docker image)
	docker build -t deepfabric/vectodb-runtime -f Dockerfile-runtime .

.PHONY: base
base: dist_dir; $(info ======== make base docker image)
	docker build -t deepfabric/vectodb-build -f Dockerfile-base .

.PHONY: vectodb
vectodb: dist_dir; $(info ======== make vectodb build docker image)
	git clone https://github.com/deepfabric/vectodb.git $(DIST_DIR)vectodb
	cd $(DIST_DIR)vectodb && git checkout $(VECTODB_VERSION)
	docker build -t deepfabric/vectodb:$(VECTODB_VERSION) -f Dockerfile-vectodb .
	docker tag deepfabric/vectodb:$(VECTODB_VERSION) deepfabric/vectodb

.PHONY: test
test: ; $(info ======== test busybee)
	env GO111MODULE=off go test -count=1 github.com/deepfabric/busybee/pkg/core
	env GO111MODULE=off go test -count=1 github.com/deepfabric/busybee/pkg/storage
	env GO111MODULE=off go test -count=1 github.com/deepfabric/busybee/pkg/queue
	env GO111MODULE=off go test -count=1 github.com/deepfabric/busybee/pkg/notify
	env GO111MODULE=off go test -count=1 github.com/deepfabric/busybee/pkg/expr
	env GO111MODULE=off go test -count=1 github.com/deepfabric/busybee/pkg/api
	env GO111MODULE=off go test -count=1 github.com/deepfabric/busybee/pkg/util
.DEFAULT_GOAL := vectodb