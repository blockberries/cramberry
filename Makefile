.PHONY: all build test bench lint fmt vet generate clean install coverage help
.PHONY: examples example-basic example-streaming example-polymorphic
.PHONY: schema-generate schema-extract
.PHONY: ts-build ts-test rust-build rust-test runtimes runtimes-test

# Go parameters
GO := go
GOFLAGS := -v
TESTFLAGS := -race -coverprofile=coverage.out -covermode=atomic
BENCHFLAGS := -bench=. -benchmem -benchtime=3s

# Binary name
BINARY := cramberry
BINARY_DIR := bin

# Package paths
PKG := ./...
CMD_PKG := ./cmd/cramberry

# Version info (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags for version info
LDFLAGS := -X github.com/blockberries/cramberry/pkg/cramberry.Version=$(VERSION) \
           -X github.com/blockberries/cramberry/pkg/cramberry.GitCommit=$(COMMIT) \
           -X github.com/blockberries/cramberry/pkg/cramberry.BuildDate=$(BUILD_DATE)

# Default target
all: fmt vet lint test build

## Build targets

build: ## Build the cramberry CLI
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_DIR)/$(BINARY) $(CMD_PKG)

install: ## Install the cramberry CLI
	$(GO) install -ldflags "$(LDFLAGS)" $(CMD_PKG)

## Test targets

test: ## Run tests with race detection and coverage
	$(GO) test $(TESTFLAGS) $(PKG)

test-short: ## Run tests without race detection (faster)
	$(GO) test -short $(PKG)

bench: ## Run benchmarks
	$(GO) test $(BENCHFLAGS) $(PKG)

coverage: test ## Generate coverage report
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Code quality targets

fmt: ## Format code
	$(GO) fmt $(PKG)
	@echo "Code formatted"

vet: ## Run go vet
	$(GO) vet $(PKG)

lint: ## Run golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run $(PKG); \
	else \
		echo "golangci-lint not installed, skipping..."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## Generation targets

generate: ## Run go generate
	$(GO) generate $(PKG)

## Utility targets

clean: ## Clean build artifacts
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean -cache -testcache

tidy: ## Tidy go.mod
	$(GO) mod tidy

deps: ## Download dependencies
	$(GO) mod download

verify: ## Verify dependencies
	$(GO) mod verify

## Development helpers

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

ci: ## Run CI pipeline locally
	@echo "Running CI pipeline..."
	$(MAKE) fmt
	$(MAKE) vet
	$(MAKE) test
	$(MAKE) build
	@echo "CI pipeline complete"

## Example targets

examples: build ## Run all example applications
	@echo "\n=== Basic Example ==="
	@$(GO) run ./examples/basic/
	@echo "\n=== Streaming Example ==="
	@$(GO) run ./examples/streaming/
	@echo "\n=== Polymorphic Example ==="
	@$(GO) run ./examples/polymorphic/

example-basic: ## Run basic example
	@$(GO) run ./examples/basic/

example-streaming: ## Run streaming example
	@$(GO) run ./examples/streaming/

example-polymorphic: ## Run polymorphic example
	@$(GO) run ./examples/polymorphic/

## Schema targets

schema-generate: build ## Generate code from example schemas
	@mkdir -p gen/
	@$(BINARY_DIR)/$(BINARY) generate -lang go -out gen/ examples/schemas/*.cram
	@echo "Generated Go code in gen/"

schema-extract: build ## Extract schema from example code
	@mkdir -p gen/
	@$(BINARY_DIR)/$(BINARY) schema -out gen/extracted.cram ./examples/basic/...
	@echo "Extracted schema to gen/extracted.cram"

## Cross-language runtime targets

ts-build: ## Build TypeScript runtime
	@echo "Building TypeScript runtime..."
	@cd typescript && npm install && npm run build

ts-test: ## Run TypeScript tests
	@echo "Running TypeScript tests..."
	@cd typescript && npm test

rust-build: ## Build Rust runtime
	@echo "Building Rust runtime..."
	@cd rust && cargo build

rust-test: ## Run Rust tests
	@echo "Running Rust tests..."
	@cd rust && cargo test

runtimes: ts-build rust-build ## Build all cross-language runtimes

runtimes-test: ts-test rust-test ## Test all cross-language runtimes

## Help

help: ## Show this help
	@echo "Cramberry Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
