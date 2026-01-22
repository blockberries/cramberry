# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cramberry is a high-performance binary serialization library for Go with code generation for Go, TypeScript, and Rust. Key features: deterministic encoding (for consensus/cryptographic operations), compact wire format (37-65% smaller than JSON), 2.7-3x faster deserialization, Amino-style polymorphic type registry, and streaming support.

## Common Commands

```bash
# Core development
make check          # Run all checks: format, vet, lint, test (use before commits)
make test           # Run tests with race detection and coverage
make test-short     # Faster tests without race detection
make build          # Build CLI to bin/cramberry
make lint           # Run golangci-lint

# Single test file/function
go test -v ./pkg/cramberry -run TestMarshal
go test -v ./pkg/schema -run TestParser

# Benchmarks
make bench

# Cross-language runtimes
make ts-test        # TypeScript tests (cd typescript && npm test)
make rust-test      # Rust tests (cd rust && cargo test)

# Code generation from schemas
./bin/cramberry generate -lang go -out ./gen ./schemas/*.cramberry
./bin/cramberry schema -out schema.cramberry ./pkg/models  # Extract schema from Go
```

## Architecture

### Core Packages

- **pkg/cramberry/** - Main runtime: Marshal/Unmarshal APIs, Writer/Reader for binary encoding, StreamWriter/StreamReader for delimited messages, Registry for polymorphic types
- **pkg/schema/** - Schema language parser: Lexer → Parser → AST → Validator. Handles `.cramberry` files with messages, enums, interfaces
- **pkg/codegen/** - Code generators implementing Generator interface for Go/TypeScript/Rust output
- **pkg/extract/** - Extracts Cramberry schemas from existing Go code via AST analysis
- **internal/wire/** - Low-level wire protocol: field tags (field_number << 3 | wire_type), varint encoding, fixed-size values

### Wire Types

| Type | Value | Used For |
|------|-------|----------|
| Varint | 0 | uint*, bool, enum |
| Fixed64 | 1 | int64, uint64, float64 |
| Bytes | 2 | string, []byte, nested messages, slices, maps |
| Fixed32 | 5 | int32, uint32, float32 |
| SVarint | 6 | int* (ZigZag encoded) |
| TypeRef | 7 | Polymorphic type ID |

### Struct Tags

```go
type Message struct {
    ID      int64  `cramberry:"1,required"` // Field 1, required
    Content string `cramberry:"2"`          // Field 2
    Data    string `cramberry:"3,omitempty"` // Omit if zero value
    Skip    string `cramberry:"-"`          // Ignore field
}
```

### Type Registry (Polymorphism)

```go
// Register types for interface serialization
cramberry.MustRegister[Dog]()           // Auto-assigns TypeID
cramberry.RegisterWithID[Cat](129)      // Explicit TypeID

// TypeID ranges: 0-63 (builtin), 64-127 (stdlib), 128+ (user)
```

### Cross-Language Structure

- **typescript/** - TypeScript runtime with Writer/Reader/Registry (npm/vitest)
- **rust/** - Rust runtime with Writer/Reader/Registry (cargo)
- **tests/integration/** - Cross-language interop tests

## Schema Language (.cramberry files)

```cramberry
package example;

enum Status { UNKNOWN = 0; ACTIVE = 1; }

message User {
    id: int64 = 1 [required];
    name: string = 2;
    tags: []string = 3;          // repeated
    metadata: map[string]string = 4;
    address: *Address = 5;       // optional pointer
}

interface Principal {
    User = 128;
    Organization = 129;
}
```

## Key Design Decisions

- **Deterministic encoding**: Maps sorted by key, fields in schema order - critical for consensus systems
- **Writer pooling**: sync.Pool for reduced allocations in hot paths
- **Field tags**: Single varint combines field number and wire type for compact encoding
- **No reflection caching**: Uses reflect package directly; generated code recommended for hot paths

## Cross-Language Compatibility Notes

### Go-Only Types

- **complex64/complex128**: Go supports complex number types, but TypeScript and Rust don't have native complex number support. Data containing complex numbers encoded in Go cannot be decoded in TypeScript/Rust. Use two separate float fields if cross-language compatibility is needed.
- **platform-dependent int/uint**: The `int` and `uint` types have platform-dependent sizes (32 or 64 bits). Schema extraction maps these to int32/uint32, which may lose precision on 64-bit platforms. Prefer explicit int32/int64/uint32/uint64 for cross-language schemas.

### Nil/Null Semantics

Different languages handle null/nil differently:
- **Go**: `nil` for pointers, interfaces, maps, slices, channels
- **TypeScript**: `null` and `undefined` are distinct
- **Rust**: `Option<T>` with `None`

Cramberry encodes nil/null as `TypeIDNil` (0) for polymorphic types. For optional fields:
- Nil pointers are omitted when `omitempty` is set
- Empty slices/maps may be encoded differently than nil slices/maps

### Map Key Type Restrictions

Map keys must be primitive types (string, integers, floats, bool) for deterministic sorting during encoding. Complex types (structs, slices, maps) are not supported as map keys.

## Mandatory Workflow for Implementation

**ALWAYS follow these steps after completing each item in a plan:**

1. **Write comprehensive tests** - Add unit tests covering the new functionality, edge cases, and error conditions. Tests should be in the same package with `_test.go` suffix.

2. **Run all tests and fix failures** - Run `make test` to execute all unit tests with race detection. If any tests fail (including pre-existing ones), fix the implementation or tests until all pass.

3. **Run integration tests** - Run `make ts-test` and `make rust-test` if changes affect cross-language compatibility. Fix any failures.

4. **Verify build with no errors or warnings** - Run `make build` and `make lint`. Fix any compiler errors, warnings, or linter issues before proceeding.

5. **Commit with comprehensive message** - After all tests pass and the build is clean, commit the changes with a detailed commit message describing what was implemented and why.
