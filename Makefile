PYTHON ?= python
GO ?= go

.PHONY: build test test-python test-go test-integration experiments all clean

build:
	cd go-side && $(GO) build -o bank-server.exe ./cmd/server

test-python:
	PYTHONPATH=python-side $(PYTHON) -m unittest discover -s tests -p "test_python_*.py" -v

test-go:
	cd go-side && $(GO) test ./...

test-integration:
	$(PYTHON) tests/integration_cross_language.py

test: test-python test-go test-integration

experiments:
	$(PYTHON) scripts/run_experiments.py

all: build test experiments

clean:
	$(PYTHON) scripts/clean_generated.py
