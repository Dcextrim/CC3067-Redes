PYTHON ?= python3
GO ?= go
NAME ?= A
CONFIG ?= $(NAME)_config.json

.PHONY: build run-router-py run-router-go test test-python test-go test-integration experiments all clean

build:
	cd go-side && $(GO) build -o router ./cmd/router

# Levanta un nodo router en Python. Sin CONFIG pide los datos por consola,
# igual que el ejemplo del profesor (make run-router-py NAME=U).
run-router-py:
	cd python-side && PYTHONPATH=. $(PYTHON) -m router.main $(if $(wildcard $(CONFIG)),--config $(CONFIG),)

run-router-go: build
	./go-side/router $(if $(wildcard $(CONFIG)),-config $(CONFIG),)

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
