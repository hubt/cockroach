GO ?= go
DOCKER ?= docker

GOPATH  := $(CURDIR)/_vendor:$(GOPATH)
ROCKSDB := $(CURDIR)/_vendor/rocksdb

CGO_CFLAGS  := "-I$(ROCKSDB)/include"
CGO_LDFLAGS := "-L$(ROCKSDB)"

CGO_FLAGS := CGO_LDFLAGS=$(CGO_LDFLAGS) \
             CGO_CFLAGS=$(CGO_CFLAGS)

PKG       := "./..."
TESTS     := ".*"
TESTFLAGS := -logtostderr -timeout 10s

all: build test

rocksdb:
	cd $(ROCKSDB); make static_lib

build: rocksdb
	$(CGO_FLAGS) $(GO) build

goget:
	$(CGO_FLAGS) $(GO) get ./...

test: rocksdb
	$(CGO_FLAGS) $(GO) test -run $(TESTS) $(PKG) $(TESTFLAGS)

testrace: rocksdb
	$(CGO_FLAGS) $(GO) test -race -run $(TESTS) $(PKG) $(TESTFLAGS)

coverage: rocksdb
	$(CGO_FLAGS) $(GO) test -cover -run $(TESTS) $(PKG) $(TESTFLAGS)

dockerbuild: 
	$(DOCKER) build -t cockroach:build .
	@echo Docker build complete use the command to start the build server
	@echo 
	@echo $(DOCKER) run --rm --name crbuild --hostname crbuild -t -i -v `pwd`:/go/src/github.com/cockroachdb/cockroach cockroach:build
	@echo 

clean:
	$(GO) clean
	cd $(ROCKSDB); make clean
