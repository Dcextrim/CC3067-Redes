GO ?= go
CONFIG ?= configs/local/A.json
DESTINATION ?= 127.0.0.1:6001
MESSAGE ?= hola
NOISE ?= 0

# Cajero (A.json): gateway 127.0.0.1:5000, cajero 127.0.0.1:6000
ATM_IP ?= 127.0.0.1
ATM_PORT ?= 6000
ATM_GATEWAY_IP ?= 127.0.0.1
ATM_GATEWAY_PORT ?= 5000
BANK_ADDR ?= 127.0.0.1:6001

# Banco (F.json): gateway 127.0.0.1:5005, banco 127.0.0.1:6001
BANK_IP ?= 127.0.0.1
BANK_PORT ?= 6001
BANK_GATEWAY_IP ?= 127.0.0.1
BANK_GATEWAY_PORT ?= 5005

.PHONY: build build-router build-client build-server build-atm build-bank run-router run-client run-server run-atm run-bank test vectors experiment local all clean

build: build-router build-client build-server build-atm build-bank

build-router:
	$(GO) build -o router ./cmd/router

build-client:
	$(GO) build -o client ./cmd/client

build-server:
	$(GO) build -o server ./cmd/server

build-atm:
	$(GO) build -o atm ./cmd/atm

build-bank:
	$(GO) build -o bank ./cmd/bank

run-router:
	$(GO) run ./cmd/router -config $(CONFIG) -noise $(NOISE)

run-client:
	$(GO) run ./cmd/client -to $(DESTINATION) -message "$(MESSAGE)" -noise $(NOISE)

run-server:
	$(GO) run ./cmd/server

run-atm:
	$(GO) run ./cmd/atm -ip $(ATM_IP) -port $(ATM_PORT) -gateway-ip $(ATM_GATEWAY_IP) -gateway-port $(ATM_GATEWAY_PORT) -bank $(BANK_ADDR) -noise $(NOISE)

run-bank:
	$(GO) run ./cmd/bank -ip $(BANK_IP) -port $(BANK_PORT) -gateway-ip $(BANK_GATEWAY_IP) -gateway-port $(BANK_GATEWAY_PORT) -noise $(NOISE)

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
