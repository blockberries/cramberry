# Cramberry Architecture

This document describes the architecture of Cramberry, a high-performance binary serialization library for Go with code generation for Go, TypeScript, and Rust.

## Overview

Cramberry is designed for systems requiring:
- **Compact wire format** - 2-3x smaller than JSON, comparable to Protocol Buffers
- **Fast decoding** - 1.5-2.6x faster than Protocol Buffers
- **Deterministic output** - Reproducible encoding for consensus and cryptographic systems
- **Polymorphic types** - Amino-style interface serialization with type registry
- **Cross-language support** - Code generation for Go, TypeScript, and Rust

## Project Structure

```
cramberry/
├── cmd/cramberry/         # CLI tool
├── pkg/
│   ├── cramberry/         # Core runtime library
│   ├── schema/            # Schema language parser
│   ├── codegen/           # Code generators
│   └── extract/           # Schema extraction from Go
├── internal/wire/         # Low-level wire primitives
├── typescript/            # TypeScript runtime
├── rust/                  # Rust runtime
├── benchmark/             # Performance benchmarks
└── tests/integration/     # Cross-language tests
```

## Core Runtime (pkg/cramberry)

### Marshal/Unmarshal API

The primary API mirrors Go's `encoding/json`:

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

### Options System

Encoding behavior is controlled via `Options`:

```go
type Options struct {
    Limits        Limits       // Resource limits (size, depth, etc.)
    WireVersion   WireVersion  // V1 or V2 wire format
    StrictMode    bool         // Reject unknown fields
    ValidateUTF8  bool         // Validate UTF-8 strings
    OmitEmpty     bool         // Omit zero-value fields
    Deterministic bool         // Sort map keys
}
```

Pre-configured option sets:
- `DefaultOptions` - V2 format, deterministic, UTF-8 validation
- `SecureOptions` - Conservative limits for untrusted input
- `FastOptions` - Skips validation, non-deterministic maps
- `StrictOptions` - Rejects unknown fields
- `V1Options` - Legacy wire format compatibility

### Resource Limits

Protection against malicious input:

```go
type Limits struct {
    MaxMessageSize  int64  // Total message size (default: 64MB)
    MaxDepth        int    // Nesting depth (default: 100)
    MaxStringLength int    // String length (default: 10MB)
    MaxBytesLength  int    // Byte slice length (default: 100MB)
    MaxArrayLength  int    // Array elements (default: 1M)
    MaxMapSize      int    // Map entries (default: 1M)
}
```

### Writer and Reader

Low-level encoding primitives with pooling:

```go
// Writer with sync.Pool recycling
w := cramberry.GetWriter()
defer cramberry.PutWriter(w)

w.WriteTag(1, WireVarint)
w.WriteInt32(42)
w.WriteTag(2, WireBytes)
w.WriteString("hello")
data := w.Bytes()

// Reader
r := cramberry.NewReader(data)
tag, wireType := r.ReadTag()
num := r.ReadInt32()
str := r.ReadString()
```

Writer features:
- Pooled via `sync.Pool` for reduced allocations
- Exponential buffer growth (doubling, capped at 256MB)
- Inline fast paths for small varints
- Zero-copy variants for performance-critical code

### Streaming Support

For large data or network streams:

```go
// StreamWriter wraps io.Writer with buffering
sw := cramberry.GetStreamWriter(conn)
defer cramberry.PutStreamWriter(sw)

sw.WriteDelimited(&msg1)
sw.WriteDelimited(&msg2)
sw.Flush()

// MessageIterator for reading streams
it := cramberry.NewMessageIterator(conn)
var msg Message
for it.Next(&msg) {
    process(msg)
}
```

### Type Registry

Polymorphic type serialization:

```go
// Register types for interface encoding
cramberry.MustRegister[Dog]()           // Auto-assigns TypeID (128+)
cramberry.RegisterWithID[Cat](129)      // Explicit TypeID

// TypeID ranges
// 0      - Nil value
// 1-63   - Reserved (built-in)
// 64-127 - Reserved (stdlib)
// 128+   - User types
```

The registry is thread-safe (`sync.RWMutex`) and supports:
- Lookup by TypeID, reflect.Type, or type name
- Interface-to-implementations mapping
- Collision detection on registration

## Wire Format

### Wire Types

| Value | Name     | Description                           |
|-------|----------|---------------------------------------|
| 0     | Varint   | Unsigned integers, bools, enums       |
| 1     | Fixed64  | 64-bit values (int64, float64)        |
| 2     | Bytes    | Length-prefixed (strings, messages)   |
| 5     | Fixed32  | 32-bit values (int32, float32)        |
| 6     | SVarint  | Signed integers (ZigZag encoded)      |
| 7     | TypeRef  | Polymorphic type ID                   |

### V2 Wire Format (Default)

The V2 format optimizes for size and decode speed:

**Compact Tags** - Single byte for fields 1-15:
```
Fields 1-15:  [fieldNum:4][wireType:3][0:1] = 1 byte
Fields 16+:   [0:4][wireType:3][1:1] + varint(fieldNum)
```

**End Marker** - Messages terminated by 0x00 instead of field count prefix:
```
message = *field end_marker
field   = tag value
end_marker = 0x00
```

**Packed Arrays** - Primitive arrays use single tag:
```
packed_array = tag length *element
```

### V1 Wire Format (Legacy)

The V1 format uses field count prefix:
```
message = field_count *field
field   = tag value
```

V1 is available via `V1Options` for compatibility with older encoded data.

### Type Mappings

| Go Type         | Wire Type | Notes                    |
|-----------------|-----------|--------------------------|
| bool            | Varint    | 0 or 1                   |
| int8-int64      | SVarint   | ZigZag encoded           |
| uint8-uint64    | Varint    | LEB128 encoded           |
| float32         | Fixed32   | IEEE 754                 |
| float64         | Fixed64   | IEEE 754                 |
| complex64       | Fixed64   | Two float32 (Go only)    |
| complex128      | Fixed128  | Two float64 (Go only)    |
| string          | Bytes     | Length-prefixed UTF-8    |
| []byte          | Bytes     | Length-prefixed          |
| slice           | Bytes     | Packed or length+elements|
| map             | Bytes     | Size + sorted key-values |
| struct          | Bytes     | Tagged fields + end mark |
| interface       | TypeRef   | TypeID + value           |

## Struct Tags

```go
type Message struct {
    ID      int64  `cramberry:"1,required"` // Field 1, required
    Name    string `cramberry:"2"`          // Field 2
    Tags    []string `cramberry:"3,omitempty"` // Omit if empty
    Skip    string `cramberry:"-"`          // Ignored
}
```

Options:
- Field number (required)
- `required` - Must be present during decode
- `omitempty` - Omit zero values during encode

## Schema Language

Schema files (`.cram`) define types for code generation:

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

### Parser Architecture

```
.cram file → Lexer → Tokens → Parser → AST → Validator → Schema
```

- **Lexer** (`lexer.go`) - Tokenizes input with UTF-8 support
- **Parser** (`parser.go`) - Produces AST with error recovery
- **Validator** (`validator.go`) - Semantic validation
- **Writer** (`io.go`) - Serializes AST back to text

## Code Generation

### Generator Interface

```go
type Generator interface {
    Generate(w io.Writer, schema *schema.Schema, options Options) error
    Language() Language
    FileExtension() string
}
```

### Generated Code

For each message, generators produce:
- Type definition with proper field types
- `MarshalCramberry()` - Direct field encoding (no reflection)
- `UnmarshalCramberry()` - Direct field decoding (no reflection)
- Enum encode/decode helpers

Generated code uses V2 wire format with compact tags and achieves 2x+ speedup over reflection-based encoding.

### Language Support

| Language   | File       | Features                    |
|------------|------------|-----------------------------|
| Go         | `*.go`     | Full marshaling, validation |
| TypeScript | `*.ts`     | Writer/Reader, types        |
| Rust       | `*.rs`     | Writer/Reader, types        |

## Schema Extraction

Extract schemas from existing Go code:

```bash
cramberry schema ./pkg/models -out schema.cram
```

The extractor:
- Analyzes Go AST for `cramberry` struct tags
- Detects enums from const blocks
- Identifies interfaces with registered implementations
- Warns about platform-dependent types (int/uint)
- Detects field number collisions

## Performance Optimizations

### Memory Management

1. **Writer Pooling** - `sync.Pool` reduces allocations in hot paths
2. **Size-Tiered Buffer Pools** - 6 size classes (64B to 64KB)
3. **Single-Allocation Encoding** - Pre-calculate size, allocate once
4. **Zero-Copy Reads** - Optional unsafe string/bytes decode

### Encoding Optimizations

1. **Compact Tags** - Single byte for common field numbers
2. **End Markers** - No two-pass encoding for field count
3. **Packed Arrays** - Single tag for primitive slices
4. **Inline Varints** - Fast path for small values (<16384)

### Struct Metadata Caching

```go
var structInfoCache sync.Map // map[reflect.Type]*structInfo
```

Field metadata is cached on first access for O(1) lookup.

## Benchmarks vs Protocol Buffers

| Metric         | Cramberry vs Protobuf |
|----------------|----------------------|
| Encode speed   | 0.95x - 1.95x        |
| Decode speed   | 1.54x - 2.60x faster |
| Encode memory  | 35-100% of Protobuf  |
| Decode memory  | 42-58% of Protobuf   |
| Encoded size   | 95-112% of Protobuf  |

Key findings:
- Decode is 1.5-2.6x faster across all message types
- Single-allocation encoding reduces GC pressure
- Metrics decoding achieves zero allocations
- Larger messages encode 2-5% smaller than Protobuf

## Cross-Language Compatibility

### Go-Only Types

- `complex64/complex128` - No TypeScript/Rust equivalent
- `int/uint` - Platform-dependent size (prefer explicit int32/int64)

### Nil/Null Semantics

| Language   | Nil Representation       |
|------------|--------------------------|
| Go         | nil pointer/interface    |
| TypeScript | null                     |
| Rust       | Option::None             |

Nil values encode as `TypeIDNil` (0) for polymorphic types.

### Map Key Restrictions

Map keys must be primitive types for deterministic sorting:
- Allowed: string, integers, floats, bool
- Not allowed: structs, slices, maps

## Thread Safety

| Component     | Safety                              |
|---------------|-------------------------------------|
| Registry      | Thread-safe (RWMutex)               |
| Writer/Reader | Single-threaded per instance        |
| Marshal/Unmarshal | Thread-safe (no shared state)   |
| Pools         | Thread-safe (sync.Pool)             |

## Error Handling

Errors support `errors.Is` and `errors.As`:

```go
// Sentinel errors
ErrUnexpectedEOF, ErrInvalidVarint, ErrInvalidWireType
ErrUnknownType, ErrUnregisteredType, ErrTypeMismatch
ErrMaxDepthExceeded, ErrMaxSizeExceeded, etc.

// Rich error types
DecodeError{Type, Field, FieldNumber, Offset, Cause}
EncodeError{Type, Field, Cause}
ValidationError{Field, Message}
```

## CLI Tool

```bash
# Generate code from schemas
cramberry generate -lang go -out ./gen ./schemas/*.cram

# Extract schema from Go code
cramberry schema ./pkg/models -out schema.cram

# Show version
cramberry version
```

## Directory Reference

| Path                    | Purpose                           |
|-------------------------|-----------------------------------|
| `pkg/cramberry/`        | Core runtime (Marshal/Unmarshal)  |
| `pkg/cramberry/writer.go` | Binary encoder with pooling     |
| `pkg/cramberry/reader.go` | Binary decoder                  |
| `pkg/cramberry/stream.go` | Streaming support               |
| `pkg/cramberry/registry.go` | Type registry               |
| `pkg/cramberry/types.go` | Options, Limits, WireType       |
| `pkg/cramberry/pool.go` | Buffer pooling                   |
| `pkg/cramberry/errors.go` | Error types                     |
| `internal/wire/`        | Low-level encoding primitives     |
| `pkg/schema/`           | Schema parser and AST             |
| `pkg/codegen/`          | Code generators                   |
| `pkg/extract/`          | Go schema extraction              |
| `typescript/src/`       | TypeScript runtime                |
| `rust/src/`             | Rust runtime                      |
