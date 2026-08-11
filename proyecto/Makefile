GO ?= go
CONFIG ?= configs/local/A.json
DESTINATION ?= 127.0.0.1:6001
MESSAGE ?= hola

.PHONY: build build-router build-client build-server run-router run-client run-server test vectors local all clean

build: build-router build-client build-server

build-router:
	cd go-side && $(GO) build -o router ./cmd/router

build-client:
	cd go-side && $(GO) build -o client ./cmd/client

build-server:
	cd go-side && $(GO) build -o server ./cmd/server

run-router:
	cd go-side && $(GO) run ./cmd/router -config ../$(CONFIG)

run-client:
	cd go-side && $(GO) run ./cmd/client -to $(DESTINATION) -message "$(MESSAGE)"

run-server:
	cd go-side && $(GO) run ./cmd/server

test:
	cd go-side && $(GO) test -count=1 ./...

vectors:
	cd go-side && $(GO) run ./cmd/vector-gen

local:
	powershell -ExecutionPolicy Bypass -File scripts/run_local_topology.ps1 -Message "$(MESSAGE)"

all: vectors test build

clean:
	powershell -ExecutionPolicy Bypass -File scripts/clean_generated.ps1
