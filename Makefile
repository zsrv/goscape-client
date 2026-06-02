# Makefile for goscape-client — a Go desktop port of the RuneScape 2
# (rev #225) Java client.
#
# Tailored from the goscape *server* Makefile: all the server/infra scaffolding
# (Docker/OCI images, Helm, mixins, jsonnet, protobuf, ClickHouse/Redpanda demo,
# Snyk, release workflows, cross-compilation, vendoring) was removed. This is a
# single-binary GUI application, so the build/test/lint surface is small.

SHELL := /usr/bin/env bash -o pipefail

BUILD_TAG    ?= $(shell ./tools/build-tag)
GIT_REVISION := $(shell git rev-parse --short HEAD)
GIT_BRANCH   := $(shell git rev-parse --abbrev-ref HEAD)

# Build flags
VPREFIX      := github.com/zsrv/goscape-client/pkg/util/build
GO_LDFLAGS   := -X $(VPREFIX).Branch=$(GIT_BRANCH) \
                -X $(VPREFIX).Version=$(BUILD_TAG) \
                -X $(VPREFIX).Revision=$(GIT_REVISION) \
                -X $(VPREFIX).BuildUser=$(shell whoami)@$(shell hostname) \
                -X $(VPREFIX).BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

GO_FLAGS     := -ldflags "-s -w $(GO_LDFLAGS)"

# The one client binary.
CMD := ./cmd/client
BIN := bin/client

# Default args for `make run`. The client now reads flags (not positional args):
# -node-id N -mem high|low -world-type free|members, plus optional
# -world-server / -ondemand-server URLs (see cmd/client/main.go).
# Override like: make run ARGS="-node-id 10 -mem low -world-type free"
ARGS ?= -node-id 10 -mem high -world-type members

# System build dependencies, shared with the devcontainer image. Evaluated
# lazily (recursive '=') so non-setup targets don't shell out to grep.
APT_PACKAGES = $(shell grep -vE '^[[:space:]]*#|^[[:space:]]*$$' .devcontainer/apt-packages.txt)

.DEFAULT_GOAL := help

.PHONY: help build run test test-race vet lint fmt check-fmt ci setup clean wasm wasm-serve

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build all packages
	go build $(GO_FLAGS) ./...

run: ## Run the client (override args with ARGS="...")
	go run $(GO_FLAGS) $(CMD) $(ARGS)

$(BIN): ## Build the client binary into bin/client
	go build $(GO_FLAGS) -o $(BIN) $(CMD)

# Browser build directory (plain go build output: index.html, main.wasm, wasm_exec.js).
WASM_OUT := build/web

wasm: ## Build the browser (js/wasm) client into build/web/
	mkdir -p $(WASM_OUT)
	GOOS=js GOARCH=wasm go build $(GO_FLAGS) -o $(WASM_OUT)/main.wasm $(CMD)
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(WASM_OUT)/wasm_exec.js
	cp web/index.html $(WASM_OUT)/index.html

wasm-serve: ## Serve the browser build at http://localhost:8080 (run `make wasm` first)
	go run $(GO_FLAGS) ./cmd/wasmserve -dir $(WASM_OUT)

test: ## Run unit tests
	go test $(GO_FLAGS) ./...

test-race: ## Run unit tests under the race detector
	go test $(GO_FLAGS) -race ./...

vet: ## Run go vet
	go vet $(GO_FLAGS) ./...

lint: ## Run golangci-lint (config: .golangci.yml)
	golangci-lint run

# Plain gofmt (no -s): this is a bug-for-bug Java port, so we enforce canonical
# formatting but deliberately avoid -s simplifications, which rewrite constructs
# that intentionally mirror the Java source (see PORTING.md / .golangci.yml).
fmt: ## Format all Go code (gofmt)
	gofmt -w .

check-fmt: ## Fail if any Go file is not gofmt-formatted
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files are not gofmt-formatted:"; echo "$$unformatted"; \
		echo "Run 'make fmt'."; exit 1; \
	fi

ci: check-fmt vet test-race lint ## Run the full CI gate (fmt check, vet, race tests, lint)

setup: ## Install Linux system build dependencies (Debian/Ubuntu; uses sudo)
	sudo apt-get update
	sudo apt-get install -y --no-install-recommends $(APT_PACKAGES)

clean: ## Remove build artifacts and clean the Go build cache
	rm -rf bin/
	go clean ./...
