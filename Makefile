# Makefile for goscape-client — a Go (Gio) desktop port of the RuneScape 2
# (rev #225) Java client.
#
# Tailored from the goscape *server* Makefile: all the server/infra scaffolding
# (Docker/OCI images, Helm, mixins, jsonnet, protobuf, ClickHouse/Redpanda demo,
# Snyk, release workflows, cross-compilation, vendoring) was removed. This is a
# single-binary GUI application, so the build/test/lint surface is small.

SHELL := /usr/bin/env bash -o pipefail

# The one client binary.
CMD := ./cmd/client
BIN := bin/client

# Default args for `make run`: node-id port-offset lowmem|highmem free|members [host].
# Override like: make run ARGS="10 0 lowmem free"
ARGS ?= 10 0 highmem members

# System build dependencies, shared with the devcontainer image. Evaluated
# lazily (recursive '=') so non-setup targets don't shell out to grep.
APT_PACKAGES = $(shell grep -vE '^[[:space:]]*#|^[[:space:]]*$$' .devcontainer/apt-packages.txt)

.DEFAULT_GOAL := help

.PHONY: help build run test test-race vet lint fmt check-fmt ci setup clean

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build all packages
	go build ./...

run: ## Run the client (override args with ARGS="...")
	go run $(CMD) $(ARGS)

$(BIN): ## Build the client binary into bin/client
	go build -o $(BIN) $(CMD)

test: ## Run unit tests
	go test ./...

test-race: ## Run unit tests under the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

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
