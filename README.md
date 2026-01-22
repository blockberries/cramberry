# Cramberry

A high-performance binary serialization library for Go with code generation for Go, TypeScript, and Rust.

## Features

- **Fast** - 1.5-2.6x faster decoding than Protocol Buffers
- **Compact** - Comparable size to Protobuf, 2-3x smaller than JSON
- **Deterministic** - Sorted map keys for reproducible encoding
- **Polymorphic** - Amino-style interface serialization with type registry
- **Streaming** - Efficient streaming encoder/decoder for large data
- **Multi-language** - Code generation for Go, TypeScript, and Rust

## Installation

```bash
go get github.com/blockberries/cramberry
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

    // Marshal
    data, err := cramberry.Marshal(user)
    if err != nil {
        panic(err)
    }

    // Unmarshal
    var decoded User
    if err := cramberry.Unmarshal(data, &decoded); err != nil {
        panic(err)
    }

    fmt.Printf("Decoded: %+v\n", decoded)
}
```

## Struct Tags

```go
type Message struct {
    ID        int64  `cramberry:"1,required"` // Field 1, required
    Content   string `cramberry:"2"`          // Field 2
    Timestamp int64  `cramberry:"3,omitempty"` // Field 3, omit if zero
    Internal  string `cramberry:"-"`          // Skip this field
}
```

## Polymorphic Types

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

// Register types (auto-assigns IDs starting at 128)
cramberry.MustRegister[Dog]()
cramberry.MustRegister[Cat]()

// Use in a container
type Zoo struct {
    Animals []Animal `cramberry:"1"`
}
```

## Streaming

For large data or network streams:

```go
// Writing
sw := cramberry.NewStreamWriter(conn)
for _, msg := range messages {
    sw.WriteDelimited(&msg)
}
sw.Flush()

// Reading
it := cramberry.NewMessageIterator(conn)
var msg MyMessage
for it.Next(&msg) {
    process(msg)
}
```

## Options

Control encoding behavior with options:

```go
// Default: V2 format, deterministic, UTF-8 validation
data, _ := cramberry.Marshal(v)

// Fast: Skip validation, non-deterministic maps
data, _ := cramberry.MarshalWithOptions(v, cramberry.FastOptions)

// Secure: Conservative limits for untrusted input
data, _ := cramberry.MarshalWithOptions(v, cramberry.SecureOptions)

// Strict: Reject unknown fields
err := cramberry.UnmarshalWithOptions(data, &v, cramberry.StrictOptions)
```

## Schema Language

Define types in `.cram` schema files:

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

## Code Generation

Generate type-safe code from schemas:

```bash
# Go
cramberry generate -lang go -out ./gen ./schemas/*.cram

# TypeScript
cramberry generate -lang typescript -out ./gen ./schemas/*.cram

# Rust
cramberry generate -lang rust -out ./gen ./schemas/*.cram
```

Generated code includes zero-reflection `MarshalCramberry()`/`UnmarshalCramberry()` methods for 2x+ performance improvement.

## Schema Extraction

Extract schemas from existing Go code:

```bash
cramberry schema ./pkg/models -out schema.cram
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

## Performance

Benchmarks on Apple M4 Pro comparing Cramberry to Protocol Buffers:

### Decode Speed (Higher is Better)

| Message Type | Cramberry | Protobuf | Speedup |
|--------------|-----------|----------|---------|
| SmallMessage | 27 ns | 68 ns | **2.5x faster** |
| Metrics | 43 ns | 112 ns | **2.6x faster** |
| Person | 387 ns | 596 ns | **1.5x faster** |
| Document | 750 ns | 1392 ns | **1.9x faster** |
| Batch1000 | 27 μs | 61 μs | **2.3x faster** |

### Encode Speed

| Message Type | Cramberry | Protobuf | Comparison |
|--------------|-----------|----------|------------|
| SmallMessage | 47 ns | 45 ns | ~equal |
| Document | 590 ns | 985 ns | **1.7x faster** |
| Event | 275 ns | 536 ns | **1.9x faster** |

### Memory Allocations

- Single-allocation encoding pattern
- Metrics decoding: **zero allocations**
- 42-58% fewer allocations than Protobuf during decode

### Encoded Size

| Message Type | Cramberry | Protobuf | Comparison |
|--------------|-----------|----------|------------|
| SmallMessage | 18 B | 16 B | +12% |
| Person | 212 B | 212 B | equal |
| Document | 412 B | 419 B | -2% |
| Batch1000 | 17 KB | 18 KB | -5% |

## API Reference

### Core Functions

```go
func Marshal(v any) ([]byte, error)
func Unmarshal(data []byte, v any) error
func MarshalWithOptions(v any, opts Options) ([]byte, error)
func UnmarshalWithOptions(data []byte, v any, opts Options) error
func MarshalAppend(buf []byte, v any) ([]byte, error)
func Size(v any) int
```

### Type Registry

```go
func Register[T any]() (TypeID, error)
func RegisterWithID[T any](id TypeID) error
func MustRegister[T any]() TypeID
func MustRegisterWithID[T any](id TypeID)
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
sw := cramberry.NewStreamWriter(w)
sw.WriteDelimited(&msg)
sw.Flush()

sr := cramberry.NewStreamReader(r)
sr.ReadDelimited(&msg)

it := cramberry.NewMessageIterator(r)
for it.Next(&msg) { ... }
```

## Cross-Language Runtimes

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

## Cross-Language Compatibility Notes

- `complex64/complex128` - Go only (no TypeScript/Rust support)
- `int/uint` - Platform-dependent size; prefer explicit `int32`/`int64`
- Map keys must be primitives (string, int, float, bool)

## Documentation

- [Architecture](ARCHITECTURE.md) - Design and implementation details
- [Benchmarks](BENCHMARKS.md) - Full performance comparison

## Development

```bash
make check    # Run all checks (format, vet, lint, test)
make test     # Run tests with race detection
make build    # Build CLI to bin/cramberry
make bench    # Run benchmarks
```

## License

Apache License 2.0
