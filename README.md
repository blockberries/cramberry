# Cramberry

A high-performance binary serialization library for Go with code generation for Go, TypeScript, and Rust.

[![Go Reference](https://pkg.go.dev/badge/github.com/blockberries/cramberry.svg)](https://pkg.go.dev/github.com/blockberries/cramberry)
[![Go Report Card](https://goreportcard.com/badge/github.com/blockberries/cramberry)](https://goreportcard.com/report/github.com/blockberries/cramberry)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Why Cramberry?

Cramberry is designed for systems that demand **speed**, **determinism**, and **cross-language interoperability**:

- **1.5-2.9x faster decoding** than Protocol Buffers
- **Deterministic encoding** - Sorted map keys produce byte-for-byte identical output, critical for consensus systems and cryptographic applications
- **Compact wire format** - 2-3x smaller than JSON, comparable to Protobuf
- **Polymorphic serialization** - Amino-style interface encoding with type registry
- **Streaming support** - Efficient streaming encoder/decoder for large data
- **Multi-language** - Full runtimes for Go, TypeScript, and Rust
- **Security hardened** - Integer overflow protection, zero-copy safety, resource limits

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Features](#features)
  - [Struct Tags](#struct-tags)
  - [Polymorphic Types](#polymorphic-types)
  - [Streaming](#streaming)
  - [Options](#options)
- [Schema Language](#schema-language)
- [Code Generation](#code-generation)
- [Performance](#performance)
- [Cross-Language Support](#cross-language-support)
- [API Reference](#api-reference)
- [Documentation](#documentation)
- [Development](#development)
- [License](#license)

## Installation

```bash
go get github.com/blockberries/cramberry
```

**CLI tool:**

```bash
go install github.com/blockberries/cramberry/cmd/cramberry@latest
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/blockberries/cramberry/pkg/cramberry"
)

type User struct {
    ID    int32  `cramberry:"1"`
    Name  string `cramberry:"2"`
    Email string `cramberry:"3"`
}

func main() {
    user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}

    // Marshal to binary
    data, err := cramberry.Marshal(user)
    if err != nil {
        panic(err)
    }

    // Unmarshal back
    var decoded User
    if err := cramberry.Unmarshal(data, &decoded); err != nil {
        panic(err)
    }

    fmt.Printf("Decoded: %+v\n", decoded)
}
```

## Features

### Struct Tags

Control serialization with struct tags:

```go
type Message struct {
    ID        int64  `cramberry:"1,required"` // Field 1, must be present on decode
    Content   string `cramberry:"2"`          // Field 2
    Timestamp int64  `cramberry:"3,omitempty"` // Field 3, omit if zero
    Internal  string `cramberry:"-"`          // Skip this field entirely
}
```

**Tag options:**
- `N` - Field number (required, must be positive integer)
- `required` - Field must be present when decoding
- `omitempty` - Omit field if it has zero value
- `-` - Skip field entirely

### Polymorphic Types

Cramberry supports interface serialization through a type registry:

```go
type Animal interface {
    Speak() string
}

type Dog struct {
    Name string `cramberry:"1"`
}
func (d *Dog) Speak() string { return "Woof!" }

type Cat struct {
    Name string `cramberry:"1"`
}
func (c *Cat) Speak() string { return "Meow!" }

func init() {
    // Register types (auto-assigns IDs starting at 128)
    // RegisterOrGet is idempotent - safe to call multiple times
    cramberry.RegisterOrGet[Dog]()
    cramberry.RegisterOrGet[Cat]()
}

// Use in a container
type Zoo struct {
    Animals []Animal `cramberry:"1"`
}
```

**Type ID ranges:**
- `0` - Nil value
- `1-63` - Reserved (built-in types)
- `64-127` - Reserved (stdlib types)
- `128+` - User-defined types

### Streaming

For large data or network streams:

```go
// Writing multiple messages
sw := cramberry.NewStreamWriter(conn)
for _, msg := range messages {
    sw.WriteDelimited(&msg)
}
sw.Flush()

// Reading with iterator pattern
it := cramberry.NewMessageIterator(conn)
var msg MyMessage
for it.Next(&msg) {
    process(msg)
}
if err := it.Err(); err != nil {
    // Handle error
}
```

### Options

Control encoding/decoding behavior with option presets:

```go
// Default: V2 format, deterministic, UTF-8 validation
data, _ := cramberry.Marshal(v)

// Fast: Skip validation, non-deterministic maps (best performance)
data, _ := cramberry.MarshalWithOptions(v, cramberry.FastOptions)

// Secure: Conservative limits for untrusted input
data, _ := cramberry.MarshalWithOptions(v, cramberry.SecureOptions)

// Strict: Reject unknown fields
err := cramberry.UnmarshalWithOptions(data, &v, cramberry.StrictOptions)
```

**Option presets:**

| Preset | Use Case |
|--------|----------|
| `DefaultOptions` | General use - deterministic, validated |
| `FastOptions` | Performance critical - skip validation |
| `SecureOptions` | Untrusted input - conservative limits |
| `StrictOptions` | Schema enforcement - reject unknown fields |

## Schema Language

Define types in `.cram` schema files for code generation:

```cramberry
package example;

enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    SUSPENDED = 2;
}

message User {
    id: int64 = 1 [required];
    name: string = 2;
    email: string = 3;
    status: Status = 4;
    tags: []string = 5;
    metadata: map[string]string = 6;
}

interface Principal {
    User = 128;
    Admin = 129;
}
```

See [docs/SCHEMA_LANGUAGE.md](docs/SCHEMA_LANGUAGE.md) for complete syntax reference.

## Code Generation

Generate type-safe code from schemas:

```bash
# Go (with MarshalCramberry/UnmarshalCramberry methods for 2x+ performance)
cramberry generate -lang go -out ./gen ./schemas/*.cram

# TypeScript
cramberry generate -lang typescript -out ./gen ./schemas/*.cram

# Rust
cramberry generate -lang rust -out ./gen ./schemas/*.cram
```

**Extract schemas from existing Go code:**

```bash
cramberry schema ./pkg/models -out schema.cram
```

## Performance

Benchmarks on Apple M4 Pro comparing Cramberry to Protocol Buffers:

### Decode Speed (Higher is Better)

| Message Type | Cramberry | Protobuf | Speedup |
|--------------|-----------|----------|---------|
| SmallMessage | 28 ns | 68 ns | **2.4x faster** |
| Metrics | 41 ns | 119 ns | **2.9x faster** |
| Person | 388 ns | 592 ns | **1.5x faster** |
| Document | 742 ns | 1394 ns | **1.9x faster** |
| Batch1000 | 27 us | 61 us | **2.3x faster** |

### Memory Efficiency

| Metric | Cramberry vs Protobuf |
|--------|----------------------|
| Encode allocations | Single allocation pattern |
| Decode allocations | 42-58% fewer |
| Metrics decode | **Zero allocations** |

### Encoded Size

| Message Type | Cramberry | Protobuf | Comparison |
|--------------|-----------|----------|------------|
| SmallMessage | 18 B | 16 B | +12% |
| Person | 212 B | 212 B | equal |
| Document | 412 B | 419 B | -2% |
| Batch1000 | 17 KB | 18 KB | -5% |

See [BENCHMARKS.md](BENCHMARKS.md) for detailed benchmark data and methodology.

## Cross-Language Support

### TypeScript

```typescript
import { Writer, Reader } from '@cramberry/runtime';

const writer = new Writer();
writer.writeInt32Field(1, 42);
writer.writeStringField(2, "hello");
const data = writer.bytes();

const reader = new Reader(data);
// ... read fields
```

### Rust

```rust
use cramberry::{Writer, Reader};

let mut writer = Writer::new();
writer.write_int32_field(1, 42)?;
writer.write_string_field(2, "hello")?;
let data = writer.into_bytes();
```

### Compatibility Notes

- **complex64/complex128** - Go only (no TypeScript/Rust support)
- **int/uint** - Platform-dependent size; prefer explicit `int32`/`int64`
- **Map keys** - Must be primitives (string, integers, floats, bool)

## API Reference

### Core Functions

```go
// Basic encoding/decoding
func Marshal(v any) ([]byte, error)
func Unmarshal(data []byte, v any) error

// With options
func MarshalWithOptions(v any, opts Options) ([]byte, error)
func UnmarshalWithOptions(data []byte, v any, opts Options) error

// Buffer reuse
func MarshalAppend(buf []byte, v any) ([]byte, error)

// Size calculation without encoding
func Size(v any) int
```

### Type Registry

```go
// Idempotent registration (recommended)
func RegisterOrGet[T any]() TypeID             // Auto-assign ID, safe to call multiple times
func RegisterOrGetWithID[T any](id TypeID) TypeID // Explicit ID, safe to call multiple times

// Error-returning registration (use when you need explicit error handling)
func Register[T any]() (TypeID, error)         // Auto-assign ID
func RegisterWithID[T any](id TypeID) error    // Explicit ID
```

### Writer/Reader (Low-Level)

```go
// Pooled writer for reduced allocations
w := cramberry.GetWriter()
defer cramberry.PutWriter(w)
w.WriteInt32(42)
w.WriteString("hello")
data := w.Bytes()

// Reader
r := cramberry.NewReader(data)
num := r.ReadInt32()
str := r.ReadString()
```

### Streaming

```go
// Writer
sw := cramberry.NewStreamWriter(w)
sw.WriteDelimited(&msg)
sw.Flush()

// Reader
sr := cramberry.NewStreamReader(r)
sr.ReadDelimited(&msg)

// Iterator pattern
it := cramberry.NewMessageIterator(r)
for it.Next(&msg) { ... }
```

## Wire Format

| Wire Type | Value | Used For |
|-----------|-------|----------|
| Varint    | 0     | uint*, bool, enum |
| Fixed64   | 1     | int64, uint64, float64 |
| Bytes     | 2     | string, []byte, messages |
| Fixed32   | 5     | int32, uint32, float32 |
| SVarint   | 6     | int* (ZigZag encoded) |
| TypeRef   | 7     | Polymorphic type ID |

V2 wire format (default) uses compact single-byte tags for fields 1-15 and end markers instead of field count prefixes.

## Documentation

- [Architecture](ARCHITECTURE.md) - Design and implementation details
- [Benchmarks](BENCHMARKS.md) - Full performance comparison
- [Roadmap](ROADMAP.md) - Development roadmap and future plans
- [Schema Language](docs/SCHEMA_LANGUAGE.md) - Complete schema syntax reference
- [Security](docs/SECURITY.md) - Security considerations and best practices
- [Migration Guide](docs/MIGRATION.md) - Migrating from other formats
- [Contributing](docs/CONTRIBUTING.md) - Contribution guidelines

## Development

```bash
make check    # Run all checks (format, vet, lint, test)
make test     # Run tests with race detection
make build    # Build CLI to bin/cramberry
make bench    # Run benchmarks
make lint     # Run golangci-lint

# Cross-language tests
make ts-test    # TypeScript tests
make rust-test  # Rust tests
```

### Running Examples

```bash
go run ./examples/basic
go run ./examples/polymorphic
go run ./examples/streaming
```

## Project Status

Cramberry is **production-ready** with comprehensive security hardening (v1.2.0).

### Recent Releases

**v1.2.0** - Zero-copy memory safety with generation tracking (breaking API change)
**v1.1.0** - Security hardening, schema compatibility checker, cross-language consistency

### What's Next

Upcoming priorities include:
- Performance optimizations (reflection caching, SIMD acceleration)
- TypeScript/Rust code generator improvements
- gRPC integration
- Python code generator

See [ROADMAP.md](ROADMAP.md) for the full development roadmap and [CHANGELOG.md](CHANGELOG.md) for release history.

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
