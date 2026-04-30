.PHONY: all build test test-short bench lint fmt fmt-check vet generate clean install coverage help
.PHONY: tidy deps verify check ci pre-commit generate-test generate-fixtures
.PHONY: examples example-basic example-streaming example-polymorphic
.PHONY: schema-generate schema-extract
.PHONY: ts-build ts-test ts-fmt ts-lint rust-build rust-test rust-fmt rust-lint
.PHONY: rust-codegen-check ts-codegen-check go-codegen-check codegen-check codegen-parity-check
.PHONY: runtimes runtimes-test ts-integration-test rust-integration-test integration-test
.PHONY: lint-all fmt-all

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

# Default target — checks-only (no mutations) so running it on a clean
# tree doesn't dirty the working copy.
all: fmt-check vet lint test build ## Run fmt-check, vet, lint, test, build (no mutations)

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

fmt: ## Format code (mutates files)
	$(GO) fmt $(PKG)
	@echo "Code formatted"

fmt-check: ## Verify code is formatted (CI-safe; doesn't mutate)
	@out=$$(gofmt -l $(shell find . -type f -name '*.go' -not -path './internal/bench/gen/*' -not -path './test/integration/gen/*')) ; \
	if [ -n "$$out" ]; then \
		echo "Files not gofmt'd:"; echo "$$out"; exit 1; \
	fi

vet: ## Run go vet
	$(GO) vet $(PKG)

# `make lint` requires golangci-lint to be installed and fails if it's
# missing. CLAUDE.md mandates a clean lint after every change, so a
# silently-skipped lint would let regressions slip through.
lint: ## Run golangci-lint (errors if not installed)
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint is required but not installed."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	}
	golangci-lint run $(PKG)

## Generation targets

generate: ## Run go generate
	$(GO) generate $(PKG)

generate-fixtures: build ## Regenerate every checked-in code-generated fixture
	@echo "Regenerating test/integration/gen/interop.go..."
	@$(BINARY_DIR)/$(BINARY) generate -lang go -json=false -out test/integration/gen testdata/schemas/interop.cram
	@gofmt -w test/integration/gen/interop.go
	@echo "Regenerating testdata/generated/json_test.go..."
	@$(BINARY_DIR)/$(BINARY) generate -lang go -out testdata/generated testdata/schemas/json_test.cram
	@gofmt -w testdata/generated/json_test.go
	@echo "Done. Run 'git diff' to inspect changes."

generate-test: generate-fixtures ## Regenerate fixtures + verify they compile + tests still pass
	@echo "Verifying generated code compiles..."
	@$(GO) build ./testdata/generated/...
	@$(GO) build ./test/integration/gen/...
	@echo "Verifying tests still pass..."
	@$(GO) test ./pkg/cramberry/... ./pkg/codegen/... ./test/integration/...
	@if git diff --quiet testdata/generated/ test/integration/gen/; then \
		echo "OK: regenerated output matches committed fixtures."; \
	else \
		echo "WARNING: fixtures changed; review and commit:"; \
		git diff --stat testdata/generated/ test/integration/gen/; \
	fi

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

check: fmt-check vet lint test ## Run Go-only checks (fmt-check, vet, lint, test)

ci: check build integration-test ## Run the full CI pipeline (Go checks + cross-language integration)
	@echo "CI pipeline complete"

pre-commit: fmt vet lint test ## Format + run all Go checks before committing
	@echo "Pre-commit checks passed"

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

ts-test: ## Run TypeScript unit tests
	@echo "Running TypeScript unit tests..."
	@cd typescript && npm test

ts-integration-test: ## Run TypeScript integration tests against Go-produced golden bytes
	@echo "Running TypeScript integration tests..."
	@cd test/integration/ts && npm test

rust-build: ## Build Rust runtime
	@echo "Building Rust runtime..."
	@cd rust && cargo build

rust-test: ## Run Rust unit tests
	@echo "Running Rust unit tests..."
	@cd rust && cargo test

rust-integration-test: ## Run Rust integration tests against Go-produced golden bytes
	@echo "Running Rust integration tests..."
	@cd test/integration/rust && cargo test

rust-codegen-check: build ## Generate + compile-check Rust output for every example/testdata schema
	@echo "Compile-checking generated Rust output..."
	@./scripts/rust-codegen-check.sh

ts-codegen-check: build ts-build ## Generate + tsc-check TypeScript output for every schema
	@echo "Compile-checking generated TypeScript output..."
	@./scripts/ts-codegen-check.sh

go-codegen-check: build ## Generate + compile-check Go output for every schema
	@echo "Compile-checking generated Go output..."
	@./scripts/go-codegen-check.sh

codegen-check: go-codegen-check ts-codegen-check rust-codegen-check ## Compile-check generated output for all three languages

codegen-parity-check: build ## End-to-end byte-parity: Go codegen == Rust codegen for the same schema
	@echo "Verifying Go codegen and Rust codegen produce identical bytes..."
	@./scripts/codegen-parity-check.sh

integration-test: ts-integration-test rust-integration-test codegen-check codegen-parity-check ## Run cross-language integration tests

runtimes: ts-build rust-build ## Build all cross-language runtimes

runtimes-test: ts-test rust-test integration-test ## Test all runtimes (unit + cross-language integration)

# TypeScript / Rust formatting + linting. These are best-effort: the
# tools are run only if installed locally, so contributors without them
# don't get a blocking failure.

ts-fmt: ## Format TypeScript sources (npx prettier)
	@cd typescript && npx --no -- prettier --write 'src/**/*.ts' || \
		echo "prettier not available; skipping ts-fmt"

ts-lint: ## Lint TypeScript sources (npx eslint)
	@cd typescript && npx --no -- eslint 'src/**/*.ts' || \
		echo "eslint not available; skipping ts-lint"

rust-fmt: ## Format Rust sources (cargo fmt)
	@cd rust && cargo fmt --all
	@cd test/integration/rust && cargo fmt --all

rust-lint: ## Lint Rust sources (cargo clippy)
	@cd rust && cargo clippy --all-targets -- -D warnings

lint-all: lint rust-lint ts-lint ## Run linters across Go, Rust, and TS

fmt-all: fmt rust-fmt ts-fmt ## Format Go, Rust, and TS sources

## Help

help: ## Show this help
	@echo "Cramberry Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@if [ -t 1 ] && [ -z "$$NO_COLOR" ]; then \
		grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
			awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'; \
	else \
		grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
			awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'; \
	fi
