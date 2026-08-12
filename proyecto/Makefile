GO ?= go
CONFIG ?= configs/local/A.json
DESTINATION ?= 127.0.0.1:6001
MESSAGE ?= hola
NOISE ?= 0

.PHONY: build build-router build-client build-server run-router run-client run-server test vectors experiment local all clean

build: build-router build-client build-server

build-router:
	$(GO) build -o router ./cmd/router

build-client:
	$(GO) build -o client ./cmd/client

build-server:
	$(GO) build -o server ./cmd/server

run-router:
	$(GO) run ./cmd/router -config $(CONFIG) -noise $(NOISE)

run-client:
	$(GO) run ./cmd/client -to $(DESTINATION) -message "$(MESSAGE)" -noise $(NOISE)

run-server:
	$(GO) run ./cmd/server

test:
	$(GO) test -count=1 ./...

vectors:
	$(GO) run ./cmd/vector-gen

experiment:
	$(GO) run ./cmd/experiment

local:
	powershell -ExecutionPolicy Bypass -File scripts/run_local_topology.ps1 -Message "$(MESSAGE)"

all: vectors experiment test build

clean:
	powershell -ExecutionPolicy Bypass -File scripts/clean_generated.ps1
