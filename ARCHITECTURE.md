# Cramberry Architecture

**Version:** 1.5.5
**Module:** `github.com/blockberries/cramberry`
**Go Version:** 1.25.6
**License:** Apache 2.0

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture Principles](#architecture-principles)
3. [System Architecture](#system-architecture)
4. [Go Package Architecture](#go-package-architecture)
5. [Core Components](#core-components)
6. [Wire Protocol Specification](#wire-protocol-specification)
7. [Concurrency Architecture](#concurrency-architecture)
8. [Performance Optimizations](#performance-optimizations)
9. [Code Generation System](#code-generation-system)
10. [Cross-Language Support](#cross-language-support)
11. [Testing Strategy](#testing-strategy)
12. [Error Handling](#error-handling)
13. [Security Considerations](#security-considerations)
14. [Build and Deployment](#build-and-deployment)
15. [Future Roadmap](#future-roadmap)

---

## Project Overview

### Purpose

Cramberry is a **high-performance binary serialization library** designed for deterministic encoding, making it ideal for:

- **Blockchain and consensus systems** requiring byte-for-byte reproducibility
- **Cryptographic operations** where encoding determinism is critical
- **Distributed systems** requiring cross-language serialization
- **Performance-critical applications** needing compact encoding

### Key Features

- **Deterministic Encoding**: Maps sorted by key, fields in schema order, canonical float encoding
- **Compact Wire Format**: 37-65% smaller than JSON, comparable to Protocol Buffers
- **High Performance**: 2.7-3x faster deserialization than JSON, competitive with Protobuf
- **Cross-Language Support**: Native runtimes for Go, TypeScript, and Rust
- **Schema Language**: Protocol Buffer-like `.cram` schema files
- **Polymorphic Types**: Amino-style type registry for interface serialization
- **Streaming Support**: Delimited messages for large datasets
- **Deterministic JSON**: Human-readable JSON output for blockchain SignDoc generation

### Use Cases

1. **Blockchain Transaction Signing**: Deterministic JSON for human-readable SignDocs
2. **Consensus Protocols**: Byte-identical encoding across nodes
3. **Microservices Communication**: Compact, cross-language RPC
4. **Data Persistence**: Efficient storage with forward compatibility
5. **Cryptographic Operations**: Reproducible hashing and signing

---

## Architecture Principles

### 1. Determinism First

Every encoding decision prioritizes determinism:

- **Map keys**: Sorted lexicographically by UTF-8 bytes
- **Fields**: Encoded in field number order (not declaration order)
- **Floats**: NaN canonicalization (`0x7FC00000`), `-0.0` normalized to `+0.0`
- **Nil values**: Consistent encoding as `TypeIDNil` (0)

### 2. Performance Through Pooling

Amortize allocation costs:

- **Writer pooling**: `sync.Pool` with 256-byte initial capacity
- **Tiered buffer pools**: Size-appropriate reuse (64B → 64KB)
- **Struct info caching**: `sync.Map` for reflection metadata
- **Zero-copy strings**: Unsafe pointers with generation tracking

### 3. Cross-Language Parity

All features work identically across Go, TypeScript, and Rust:

- **Wire protocol**: Identical binary encoding
- **JSON rules**: Same deterministic JSON output
- **Type system**: Unified schema language
- **Code generation**: Feature parity across languages

### 4. Safety and Limits

Prevent resource exhaustion and attacks:

- **Max depth**: 32 levels (default), prevents stack overflow
- **Max message size**: 64MB (default), prevents OOM
- **UTF-8 validation**: Fast in-place validation for strings
- **Bounds checking**: All buffer operations checked
- **Integer overflow**: Explicit checks in varint decoder

### 5. Forward/Backward Compatibility

Schema evolution without breaking changes:

- **Unknown fields**: Skipped during decoding (forward compat)
- **Optional fields**: Pointer types, omitempty support
- **Reserved fields**: Schema-level field number reservation
- **Versioning**: Type registry supports multiple versions

---

## System Architecture

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        User Application                       │
└───────────┬──────────────────────────────────────────────────┘
            │
            ├──────────── Marshal/Unmarshal API ────────────────┤
            │                                                    │
            v                                                    v
┌───────────────────────┐                     ┌────────────────────────┐
│    cramberry.Writer   │                     │   cramberry.Reader     │
│  ┌─────────────────┐  │                     │  ┌──────────────────┐  │
│  │  Buffer Pool    │  │                     │  │  Zero-Copy Refs  │  │
│  │  (sync.Pool)    │  │                     │  │  (Unsafe)        │  │
│  └─────────────────┘  │                     │  └──────────────────┘  │
└───────────┬───────────┘                     └──────────┬─────────────┘
            │                                            │
            └────────────────┬───────────────────────────┘
                             │
                             v
                ┌────────────────────────┐
                │   internal/wire/*      │
                │  ┌──────────────────┐  │
                │  │ Varint Encoding  │  │
                │  │ Fixed Encoding   │  │
                │  │ Field Tags       │  │
                │  └──────────────────┘  │
                └────────────────────────┘
                             │
                             v
                    ┌────────────────┐
                    │  Binary Stream │
                    └────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                     Code Generation Pipeline                  │
└───────────┬──────────────────────────────────────────────────┘
            │
            v
┌─────────────────────┐
│  .cram Schema File  │
│  ┌───────────────┐  │
│  │ package user; │  │
│  │ message User {│  │
│  │   id: int64;  │  │
│  │ }             │  │
│  └───────────────┘  │
└──────────┬──────────┘
           │
           v
┌──────────────────────┐
│  pkg/schema/*        │
│  ┌────────────────┐  │
│  │ Lexer → Parser │  │
│  │ AST → Validate │  │
│  └────────────────┘  │
└──────────┬───────────┘
           │
           v
┌──────────────────────┐
│  pkg/codegen/*       │
│  ┌────────────────┐  │
│  │ Go Generator   │  │
│  │ TS Generator   │  │
│  │ Rust Generator │  │
│  └────────────────┘  │
└──────────┬───────────┘
           │
           v
┌────────────────────────────────────────────────────┐
│  Generated Code (Go/TypeScript/Rust)               │
│  • EncodeTo(Writer) / encode_to(&Writer)           │
│  • DecodeFrom(Reader) / decode_from(&Reader)       │
│  • ToJSON() / to_json()                            │
│  • FromJSON(string) / from_json(&str)              │
└────────────────────────────────────────────────────┘
```

### Data Flow

**Encoding Path:**
```
Go Value (interface{})
  → reflect.ValueOf()
  → Type Switch (struct/slice/map/primitive)
  → encodeField() [recursive for nested types]
  → Writer.WriteUvarint() / WriteBytes() / etc.
  → wire/* encoding (varint, fixed, tag)
  → Buffer ([]byte)
```

**Decoding Path:**
```
Buffer ([]byte)
  → Reader
  → ReadFieldTag()
  → Field Number → Struct Field (via cached reflection)
  → decodeValue() [recursive for nested types]
  → reflect.Set()
  → Go Value (interface{})
```

---

## Go Package Architecture

### Directory Structure

```
github.com/blockberries/cramberry/
├── cmd/cramberry/              # CLI application
│   └── main.go                 # Entry point (generate, validate, extract)
├── pkg/                        # Public library APIs
│   ├── cramberry/              # Main runtime (26 files, ~200KB)
│   │   ├── marshal.go          # Marshal() API
│   │   ├── unmarshal.go        # Unmarshal() API
│   │   ├── writer.go           # Binary writer with pooling
│   │   ├── reader.go           # Binary reader with zero-copy
│   │   ├── stream.go           # StreamWriter/StreamReader
│   │   ├── registry.go         # Type registry (polymorphism)
│   │   ├── json.go             # Deterministic JSON
│   │   ├── types.go            # Wire types, options, limits
│   │   ├── errors.go           # Error types
│   │   ├── pool.go             # Writer/buffer pooling
│   │   └── *_test.go           # Extensive tests (12 files)
│   ├── schema/                 # Schema language (12 files, ~100KB)
│   │   ├── lexer.go            # Tokenization
│   │   ├── parser.go           # Recursive descent parser
│   │   ├── ast.go              # AST node definitions
│   │   ├── validator.go        # Schema validation
│   │   ├── io.go               # File I/O, formatting
│   │   └── *_test.go           # Parser tests
│   ├── codegen/                # Code generation (4 files, ~80KB)
│   │   ├── generator.go        # Generator interface
│   │   ├── go_generator.go     # Go codegen (~48KB)
│   │   ├── typescript_generator.go # TS codegen (~29KB)
│   │   ├── rust_generator.go   # Rust codegen (~33KB)
│   │   └── *_test.go
│   └── extract/                # Go → Schema extraction (5 files)
│       ├── collector.go        # AST walking
│       ├── builder.go          # Schema building
│       ├── loader.go           # Package loading
│       └── *_test.go
├── internal/                   # Private implementation
│   ├── wire/                   # Wire protocol (6 files, ~50KB)
│   │   ├── varint.go           # LEB128 varint encoding
│   │   ├── fixed.go            # Fixed-size encoding
│   │   ├── tag.go              # Field tag encoding
│   │   └── *_test.go
│   └── bench/                  # Benchmark suite
│       ├── benchmark_test.go   # Cramberry vs Protobuf vs JSON
│       ├── schemas/            # Benchmark schemas
│       └── gen/                # Generated benchmark code
├── typescript/                 # TypeScript runtime
│   ├── src/                    # Source (~70KB)
│   │   ├── writer.ts
│   │   ├── reader.ts
│   │   ├── stream.ts
│   │   ├── registry.ts
│   │   ├── json.ts
│   │   └── *.test.ts
│   └── package.json
├── rust/                       # Rust runtime
│   ├── src/                    # Source (~60KB)
│   │   ├── writer.rs
│   │   ├── reader.rs
│   │   ├── stream.rs
│   │   ├── registry.rs
│   │   ├── json.rs
│   │   └── lib.rs
│   └── Cargo.toml
├── test/integration/           # Cross-language tests
│   ├── interop_test.go         # Go golden file tests
│   ├── ts/                     # TypeScript tests
│   ├── rust/                   # Rust tests
│   └── gen/                    # Generated test code
├── testdata/                   # Test fixtures
│   ├── golden/                 # Binary golden files
│   ├── schemas/                # Test schemas
│   └── generated/              # Generated test code
├── examples/                   # Example applications
│   ├── basic/
│   ├── streaming/
│   └── polymorphic/
├── go.mod, go.sum              # Go modules
├── Makefile                    # Build automation
└── CLAUDE.md                   # Developer guide
```

### Package Dependency Graph

```
cmd/cramberry
  ↓
  ├─→ pkg/schema    (parse .cram files)
  ├─→ pkg/codegen   (generate Go/TS/Rust)
  ├─→ pkg/extract   (Go → schema)
  └─→ pkg/cramberry (validate with runtime)

pkg/codegen
  ↓
  └─→ pkg/schema    (AST types)

pkg/extract
  ↓
  ├─→ pkg/schema    (build schema)
  └─→ golang.org/x/tools/go/packages

pkg/cramberry
  ↓
  └─→ internal/wire (low-level encoding)

internal/wire
  (no dependencies - pure Go stdlib)
```

### Exported vs Internal APIs

**Public APIs (`pkg/`):**

- **`pkg/cramberry`**: Core encoding/decoding APIs
  - `Marshal(v interface{}) ([]byte, error)`
  - `Unmarshal(data []byte, v interface{}) error`
  - `Writer`, `Reader`, `StreamWriter`, `StreamReader`
  - `Registry`, `Register[T]()`, `RegisterWithID[T](TypeID)`

- **`pkg/schema`**: Schema language APIs
  - `ParseFile(path string) (*Schema, error)`
  - `ParseString(source string) (*Schema, error)`
  - `Validate(schema *Schema) error`
  - `FormatFile(path string) error`

- **`pkg/codegen`**: Code generation APIs
  - `Generator` interface
  - `Get(lang Language) (Generator, bool)`
  - `Generate(w io.Writer, schema *Schema, opts Options) error`

- **`pkg/extract`**: Schema extraction APIs
  - `ExtractSchema(pkgs []string) (*schema.Schema, error)`
  - `Options` configuration

**Internal APIs (`internal/`):**

- **`internal/wire`**: Wire protocol internals (not exported)
  - Varint encoding/decoding
  - Fixed-size encoding
  - Field tag encoding
  - Used exclusively by `pkg/cramberry`

- **`internal/bench`**: Benchmarking (not exported)
  - Performance comparisons
  - Not part of public API

---

## Core Components

### 1. Runtime Library (`pkg/cramberry/`)

**Purpose**: Binary encoding/decoding with reflection-based marshaling.

**Key Files:**

- **`marshal.go`** (1,200 lines): Marshal API, reflection-based encoding
- **`unmarshal.go`** (1,100 lines): Unmarshal API, reflection-based decoding
- **`writer.go`** (800 lines): Buffered binary writer with pooling
- **`reader.go`** (700 lines): Binary reader with zero-copy optimization
- **`stream.go`** (400 lines): Delimited message streaming
- **`registry.go`** (500 lines): Polymorphic type registry
- **`json.go`** (600 lines): Deterministic JSON serialization
- **`types.go`** (300 lines): Wire type definitions, options, limits
- **`errors.go`** (200 lines): Error types and helpers
- **`pool.go`** (100 lines): Writer/buffer pooling

**Responsibilities:**

1. **Marshaling**: Go value → binary encoding
2. **Unmarshaling**: Binary encoding → Go value
3. **Buffering**: Efficient memory management via pooling
4. **Streaming**: Delimited messages for large datasets
5. **Type Registry**: Polymorphic type resolution
6. **JSON Conversion**: Deterministic JSON for SignDocs
7. **Error Handling**: Structured error types with context

**Design Patterns:**

- **Facade Pattern**: `Marshal()`/`Unmarshal()` hide complexity
- **Object Pool Pattern**: `sync.Pool` for Writer/Reader reuse
- **Registry Pattern**: Type ID → Go type mapping
- **Options Pattern**: Fluent configuration (limits, validation)

### 2. Schema Language (`pkg/schema/`)

**Purpose**: Parse, validate, and manipulate `.cram` schema files.

**Key Files:**

- **`lexer.go`** (600 lines): Tokenization of schema syntax
- **`parser.go`** (1,200 lines): Recursive descent parser
- **`ast.go`** (400 lines): AST node definitions
- **`validator.go`** (600 lines): Schema validation rules
- **`io.go`** (300 lines): File I/O, formatting
- **`compat.go`** (200 lines): Backward compatibility checks

**Responsibilities:**

1. **Lexical Analysis**: Source → Tokens
2. **Syntax Parsing**: Tokens → AST
3. **Semantic Validation**: AST → Validated Schema
4. **Schema I/O**: Read/write `.cram` files
5. **Formatting**: Canonical schema formatting

**Pipeline:**

```
.cram file → Lexer → Tokens → Parser → AST → Validator → Schema
```

**Example Schema:**

```cramberry
package user;

enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    INACTIVE = 2;
}

message User {
    id: int64 = 1 [required];
    name: string = 2;
    email: string = 3;
    status: Status = 4;
    tags: []string = 5;
    metadata: map[string]string = 6;
    address: *Address = 7;  // Optional pointer
}

message Address {
    street: string = 1;
    city: string = 2;
    country: string = 3;
}

interface Principal {
    User = 128;
    Organization = 129;
}
```

### 3. Code Generation (`pkg/codegen/`)

**Purpose**: Generate Go/TypeScript/Rust code from schemas.

**Key Files:**

- **`generator.go`** (200 lines): Generator interface, registry
- **`go_generator.go`** (2,500 lines): Go code generation
- **`typescript_generator.go`** (1,500 lines): TypeScript generation
- **`rust_generator.go`** (1,700 lines): Rust generation

**Responsibilities:**

1. **Go Generation**: Structs, `EncodeTo()`, `DecodeFrom()`, `ToJSON()`, `FromJSON()`
2. **TypeScript Generation**: Interfaces, `encode_*()`, `decode_*()`, `toJSON_*()`, `fromJSON_*()`
3. **Rust Generation**: Structs (with derive), `encode_to()`, `decode_from()`, `to_json()`, `from_json()`
4. **Template Rendering**: Go uses `text/template`, TS/Rust use programmatic generation

**Generated Code Example (Go):**

```go
// Code generated by cramberry. DO NOT EDIT.
package user

import "github.com/blockberries/cramberry/pkg/cramberry"

type User struct {
    ID       int64             `cramberry:"1,required"`
    Name     string            `cramberry:"2"`
    Email    string            `cramberry:"3"`
    Status   Status            `cramberry:"4"`
    Tags     []string          `cramberry:"5"`
    Metadata map[string]string `cramberry:"6"`
    Address  *Address          `cramberry:"7"`
}

func (m *User) EncodeTo(w *cramberry.Writer) error {
    // Generated encoding logic...
}

func (m *User) DecodeFrom(r *cramberry.Reader) error {
    // Generated decoding logic...
}

func (m *User) ToJSON() (string, error) {
    // Generated JSON encoding...
}

func (m *User) FromJSON(s string) error {
    // Generated JSON decoding...
}
```

### 4. Schema Extraction (`pkg/extract/`)

**Purpose**: Reverse-engineer `.cram` schema from existing Go code.

**Key Files:**

- **`collector.go`** (800 lines): AST walking, type collection
- **`builder.go`** (600 lines): Schema building from Go types
- **`loader.go`** (300 lines): Package loading via `go/packages`
- **`writer.go`** (200 lines): Schema file writing

**Responsibilities:**

1. **Go AST Analysis**: Walk Go source files
2. **Type Inference**: Extract struct fields, tags, comments
3. **Schema Building**: Construct `.cram` schema
4. **Validation**: Ensure extracted schema is valid

**Usage:**

```bash
cramberry schema -out user.cram ./pkg/models
```

**Example Extraction:**

```go
// Go source:
package models

type User struct {
    ID   int64  `cramberry:"1,required"`
    Name string `cramberry:"2"`
}

// Extracted schema:
package models;

message User {
    id: int64 = 1 [required];
    name: string = 2;
}
```

### 5. Wire Protocol (`internal/wire/`)

**Purpose**: Low-level binary encoding primitives.

**Key Files:**

- **`varint.go`** (400 lines): LEB128 varint encoding/decoding
- **`fixed.go`** (300 lines): Fixed-size encoding (32/64-bit)
- **`tag.go`** (200 lines): Field tag encoding

**Responsibilities:**

1. **Varint Encoding**: Variable-length integers (LEB128)
2. **ZigZag Encoding**: Signed integer encoding
3. **Fixed Encoding**: Little-endian fixed-size values
4. **Tag Encoding**: Field number + wire type

**Not Exported**: Used exclusively by `pkg/cramberry`.

---

## Wire Protocol Specification

### Field Tag Encoding (V2 Format)

**Compact Tag Format:**

```
Fields 1-15:  [fieldNum:4][wireType:3][0:1] = 1 byte
Fields 16+:   [0:4][wireType:3][1:1] + varint(fieldNum)
End marker:   0x00
```

**Examples:**

```
Field 1, WireTypeV2Varint (0):
  → 0001 000 0 = 0x08 (1 byte)

Field 5, WireTypeV2Bytes (2):
  → 0101 010 0 = 0x52 (1 byte)

Field 100, WireTypeV2Fixed64 (1):
  → 0000 001 1 + varint(100)
  → 0x09 0xE4 0x01 (3 bytes)
```

### Wire Types

| Value | Name | Use Case | Encoding |
|-------|------|----------|----------|
| 0 | Varint | `uint8`-`uint64`, `bool`, `enum` | LEB128 |
| 1 | Fixed64 | `float64`, `int64`, `uint64` | 8 bytes LE |
| 2 | Bytes | `string`, `[]byte`, messages, arrays, maps | Length-prefixed |
| 3 | Fixed32 | `float32`, `int32`, `uint32` | 4 bytes LE |
| 4 | SVarint | `int8`-`int64` | ZigZag + LEB128 |

### Varint Encoding (LEB128)

**Algorithm:**

```
Encode:
  while value >= 128:
    output.append(value & 0x7F | 0x80)  // MSB set (continuation)
    value >>= 7
  output.append(value & 0x7F)           // MSB clear (end)

Decode:
  result = 0, shift = 0
  for each byte:
    result |= (byte & 0x7F) << shift
    if byte & 0x80 == 0: break          // End marker
    shift += 7
```

**Examples:**

```
0      → [0x00]
127    → [0x7F]
128    → [0x80, 0x01]
300    → [0xAC, 0x02]
16383  → [0xFF, 0x7F]
16384  → [0x80, 0x80, 0x01]
```

### ZigZag Encoding (Signed Integers)

**Formula:**

```go
func EncodeZigZag(n int64) uint64 {
    return uint64((n << 1) ^ (n >> 63))
}

func DecodeZigZag(n uint64) int64 {
    return int64((n >> 1) ^ -(n & 1))
}
```

**Mapping:**

```
 0 → 0
-1 → 1
 1 → 2
-2 → 3
 2 → 4
```

### Fixed-Size Encoding

**Little-Endian Format:**

```
Fixed32: [b0][b1][b2][b3]
  Example: 0x12345678 → [0x78, 0x56, 0x34, 0x12]

Fixed64: [b0][b1][b2][b3][b4][b5][b6][b7]
  Example: 0x0102030405060708 → [0x08, 0x07, 0x06, 0x05, ...]
```

### Float Canonicalization

**Rules:**

1. **NaN**: All NaN values → Canonical NaN (`0x7FC00000` for float32, `0x7FF8000000000000` for float64)
2. **Negative Zero**: `-0.0` → `+0.0` (clear sign bit)
3. **Infinity**: Preserved as-is (`+Inf`, `-Inf`)

**Implementation:**

```go
func canonicalizeFloat32(v float32) uint32 {
    bits := math.Float32bits(v)

    // NaN detection: exponent all 1s, mantissa nonzero
    if bits&0x7F800000 == 0x7F800000 && bits&0x007FFFFF != 0 {
        return 0x7FC00000  // Canonical NaN
    }

    // -0.0 detection
    if bits == 0x80000000 {
        return 0  // +0.0
    }

    return bits
}
```

### Message Encoding

**Format:**

```
message User {
    id: int64 = 1;
    name: string = 2;
}

Encoding:
  [tag1][value1][tag2][value2]...[0x00]

Example (id=42, name="alice"):
  0x08              // tag (field 1, varint)
  0x2A              // value (42)
  0x12              // tag (field 2, bytes)
  0x05              // length (5)
  0x61 0x6C 0x69 0x63 0x65  // "alice"
  0x00              // end marker
```

### Array Encoding

**Unpacked (Tagged):**

```
repeated int32 numbers = 1;

Encoding:
  [tag][elem1][tag][elem2]...[0x00]

Example ([1, 2, 3]):
  0x08 0x01  // field 1, varint, value 1
  0x08 0x02  // field 1, varint, value 2
  0x08 0x03  // field 1, varint, value 3
  0x00       // end marker
```

**Packed (No Tags):**

```
Encoding:
  [tag][length][elem1][elem2]...[0x00]

Example ([1, 2, 3]):
  0x12       // field 1, bytes (packed)
  0x03       // length (3 elements)
  0x01 0x02 0x03  // values
  0x00       // end marker
```

### Map Encoding

**Format:**

```
map[string]int32 scores = 1;

Encoding:
  [tag][count][key1][value1][key2][value2]...[0x00]

Example ({"a":1, "z":26}):
  0x12       // field 1, bytes
  0x02       // count (2 entries)
  0x01 0x61  // key "a" (length 1, byte 'a')
  0x01       // value 1
  0x01 0x7A  // key "z" (length 1, byte 'z')
  0x1A       // value 26
  0x00       // end marker
```

**Determinism**: Keys sorted lexicographically by UTF-8 bytes.

---

## Concurrency Architecture

### Thread-Safe Components

**1. Registry (`registry.go`)**

- **Mutex**: `sync.RWMutex`
- **Read Lock**: Type lookups (hot path)
- **Write Lock**: Type registration (cold path)
- **Safe for Concurrent Use**: Yes

```go
func (r *Registry) GetByID(id TypeID) (*TypeRegistration, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    reg, ok := r.byID[id]
    return reg, ok
}

func (r *Registry) RegisterType(typ reflect.Type, id TypeID) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    // Registration logic...
}
```

**2. Struct Info Cache**

- **Type**: `sync.Map`
- **Purpose**: Cache reflection metadata for structs
- **Concurrency**: Built-in concurrency in `sync.Map`
- **Read-Heavy**: Optimized for concurrent reads

```go
var structInfoCache sync.Map  // reflect.Type → *structInfo

func getStructInfo(typ reflect.Type) *structInfo {
    if cached, ok := structInfoCache.Load(typ); ok {
        return cached.(*structInfo)
    }

    // Compute once
    info := computeStructInfo(typ)
    structInfoCache.Store(typ, info)
    return info
}
```

**3. Writer/Reader Pools**

- **Type**: `sync.Pool`
- **Thread-Safe**: Yes (built-in)
- **Purpose**: Reuse Writer/Reader objects
- **Pattern**: Get → Use → Put

```go
var writerPool = sync.Pool{
    New: func() any {
        return &Writer{buf: make([]byte, 0, 256)}
    },
}

func GetWriter() *Writer {
    return writerPool.Get().(*Writer)
}

func PutWriter(w *Writer) {
    if cap(w.buf) <= 65536 {  // Don't pool large buffers
        w.Reset()
        writerPool.Put(w)
    }
}
```

### Not Thread-Safe (Single-Goroutine Use)

**1. Writer**

- **Not Safe**: Single-goroutine use only
- **Reason**: Buffered state, no locking
- **Pattern**: Get from pool → Use → Return to pool

**2. Reader**

- **Not Safe**: Single-goroutine use only
- **Reason**: Position tracking, no locking
- **Pattern**: Create → Use → Discard (lightweight)

**3. StreamWriter / StreamReader**

- **Not Safe**: Single-goroutine use only
- **Reason**: Buffered I/O state
- **Pattern**: Create → Sequential writes/reads → Close

### Goroutine Safety Guidelines

**Safe Patterns:**

```go
// ✓ Concurrent marshaling with pooled writers
func worker(data []Message) {
    for _, msg := range data {
        w := cramberry.GetWriter()
        if err := msg.EncodeTo(w); err != nil {
            // Handle error
        }
        encoded := w.BytesCopy()
        cramberry.PutWriter(w)

        // Send encoded data...
    }
}

// Launch multiple workers
for i := 0; i < runtime.NumCPU(); i++ {
    go worker(partition[i])
}
```

```go
// ✓ Concurrent type registration (idempotent)
func init() {
    cramberry.Register[User]()
    cramberry.Register[Order]()
}
```

**Unsafe Patterns:**

```go
// ✗ Sharing Writer across goroutines
w := cramberry.GetWriter()
go func() {
    msg1.EncodeTo(w)  // UNSAFE: concurrent write
}()
go func() {
    msg2.EncodeTo(w)  // UNSAFE: concurrent write
}()
```

```go
// ✗ Reusing Reader with zero-copy strings
r := cramberry.NewReader(data)
s := r.ReadStringZeroCopy()

go func() {
    r.Reset(newData)  // UNSAFE: invalidates zero-copy string
}()

fmt.Println(s.String())  // May panic or return garbage
```

### Context and Cancellation

**Not Supported**: Cramberry does not use `context.Context` for encoding/decoding.

**Rationale:**

1. **Fast Operations**: Encoding/decoding typically completes in microseconds
2. **No I/O**: Operations are pure CPU (no network/disk)
3. **Simplicity**: Avoid overhead of context checking

**Workaround for Cancellation:**

```go
func marshalWithTimeout(msg Message, timeout time.Duration) ([]byte, error) {
    done := make(chan struct {
        data []byte
        err  error
    }, 1)

    go func() {
        data, err := cramberry.Marshal(msg)
        done <- struct{ data []byte; err error }{data, err}
    }()

    select {
    case result := <-done:
        return result.data, result.err
    case <-time.After(timeout):
        return nil, errors.New("marshal timeout")
    }
}
```

---

## Performance Optimizations

### 1. Writer Pooling

**Strategy**: Reuse Writer objects with pre-allocated buffers.

**Implementation:**

```go
var writerPool = sync.Pool{
    New: func() any {
        return &Writer{buf: make([]byte, 0, 256)}
    },
}

func GetWriter() *Writer {
    w := writerPool.Get().(*Writer)
    w.Reset()
    return w
}

func PutWriter(w *Writer) {
    // Don't pool buffers >64KB (prevent memory bloat)
    if cap(w.buf) <= 65536 {
        w.buf = w.buf[:0]  // Reset length, keep capacity
        writerPool.Put(w)
    }
}
```

**Benefits:**

- **~50% fewer allocations** for repeated marshaling
- **Amortized buffer growth** across operations
- **Reduced GC pressure** (fewer short-lived objects)

**Benchmark Results:**

```
BenchmarkMarshal-12                1,000,000    1,200 ns/op    256 B/op    3 allocs/op
BenchmarkMarshalWithPool-12        2,000,000      600 ns/op    128 B/op    1 alloc/op
```

### 2. Buffer Growth Strategy

**Doubling with Cap:**

```go
func (w *Writer) grow(n int) {
    needed := len(w.buf) + n
    if needed <= cap(w.buf) {
        return  // Sufficient capacity
    }

    // Double capacity, capped at 256MB
    newCap := cap(w.buf) * 2
    if newCap == 0 {
        newCap = 256  // Initial capacity
    }
    if newCap > 256*1024*1024 {
        newCap = needed  // Don't double beyond 256MB
    }

    newBuf := make([]byte, len(w.buf), newCap)
    copy(newBuf, w.buf)
    w.buf = newBuf
}
```

**Rationale:**

- **Doubling**: Amortized O(1) append operations
- **Cap at 256MB**: Prevent excessive memory use for large messages
- **Linear growth**: Beyond 256MB, grow exactly as needed

### 3. Zero-Copy String/Bytes

**Optimization**: Return `unsafe.String` from Reader without allocation.

**Implementation:**

```go
type ZeroCopyString struct {
    s          string
    generation uint64  // Validity tracking
    reader     *Reader
}

func (r *Reader) ReadStringZeroCopy() ZeroCopyString {
    length := r.ReadUvarint()
    if r.err != nil {
        return ZeroCopyString{}
    }

    // unsafe.String: no allocation
    s := unsafe.String(&r.data[r.pos], length)
    r.pos += length

    return ZeroCopyString{
        s:          s,
        generation: r.generation,
        reader:     r,
    }
}

func (zs ZeroCopyString) String() string {
    if zs.reader.generation != zs.generation {
        panic("zero-copy string used after Reset()")
    }
    return zs.s
}
```

**Safety Mechanisms:**

- **Generation Counter**: Invalidates on `Reset()`
- **Panic on Invalid Access**: Fail-fast for memory safety
- **Documentation Warnings**: Clearly document lifetime constraints

**Benchmark Impact:**

```
BenchmarkReadString-12                  5,000,000    300 ns/op    64 B/op    2 allocs/op
BenchmarkReadStringZeroCopy-12         10,000,000    150 ns/op     0 B/op    0 allocs/op
```

### 4. Struct Info Caching

**Strategy**: Cache reflection metadata for structs.

**Implementation:**

```go
var structInfoCache sync.Map  // reflect.Type → *structInfo

type structInfo struct {
    fields []fieldInfo
}

type fieldInfo struct {
    name       string
    index      []int           // Reflect field index path
    typ        reflect.Type
    wireType   WireType
    number     int
    required   bool
    omitEmpty  bool
}

func getStructInfo(typ reflect.Type) *structInfo {
    if cached, ok := structInfoCache.Load(typ); ok {
        return cached.(*structInfo)
    }

    // Compute once
    info := computeStructInfo(typ)

    // Double-checked locking
    if actual, loaded := structInfoCache.LoadOrStore(typ, info); loaded {
        return actual.(*structInfo)
    }

    return info
}
```

**Benefits:**

- **~10x faster** repeated marshaling of same type
- **Reflection overhead** amortized across all instances
- **Thread-safe** via `sync.Map`

### 5. Inline Fast Paths

**Varint Decoding Optimization:**

```go
func (r *Reader) ReadUvarint() uint64 {
    if r.pos >= len(r.data) {
        r.setError(ErrUnexpectedEOF)
        return 0
    }

    b := r.data[r.pos]

    // Fast path: 1 byte (90% of cases, values <128)
    if b < 0x80 {
        r.pos++
        return uint64(b)
    }

    // Fast path: 2 bytes (covers 0-16383)
    if r.pos+1 < len(r.data) && r.data[r.pos+1] < 0x80 {
        v := uint64(b&0x7F) | uint64(r.data[r.pos+1])<<7
        r.pos += 2
        return v
    }

    // General case (rare)
    return r.readUvarintSlow()
}
```

**Impact:**

- **~3x faster** for small integers (1-2 byte varints)
- **Branch prediction friendly** (common case first)
- **No allocation** in fast path

### 6. Packed Array Encoding

**Optimization**: Omit field tags for primitive arrays.

**Unpacked Encoding (wasteful):**

```
[tag][value1][tag][value2][tag][value3]
  ^            ^            ^
  5 bytes each for field tag + varint value
```

**Packed Encoding (optimized):**

```
[tag][length][value1][value2][value3]
  ^           ^
  1 byte tag + 1 byte length + N values
```

**Savings:**

```
Unpacked: 3 tags × 1 byte + 3 values × 1 byte = 6 bytes
Packed:   1 tag + 1 length + 3 values = 5 bytes
Savings:  ~17% for small arrays, >50% for large arrays
```

**Implementation:**

```go
func (w *Writer) WritePackedInt32Slice(s []int32) {
    w.WriteUvarint(uint64(len(s)))
    for _, v := range s {
        w.WriteInt32(v)  // No tag
    }
}
```

### 7. Deterministic Map Sorting

**Challenge**: Maps must be sorted for deterministic encoding, but sorting is expensive.

**Optimization**: Sort keys once, encode in order.

```go
func (w *Writer) WriteMap(m map[string]int64) {
    // Extract keys
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }

    // Sort once (lexicographic)
    sort.Strings(keys)

    // Encode in order
    w.WriteUvarint(uint64(len(m)))
    for _, k := range keys {
        w.WriteString(k)
        w.WriteInt64(m[k])
    }
}
```

**Alternative (avoid allocation):**

```go
// For small maps (<16 entries), use insertion sort in-place
func sortSmallMap(keys []string) {
    for i := 1; i < len(keys); i++ {
        for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
            keys[j], keys[j-1] = keys[j-1], keys[j]
        }
    }
}
```

---

## Code Generation System

### Generator Architecture

**Interface:**

```go
type Generator interface {
    Generate(w io.Writer, schema *Schema, opts Options) error
    Language() Language
    FileExtension() string
}
```

**Registry Pattern:**

```go
var generators = make(map[Language]Generator)

func Register(gen Generator) {
    generators[gen.Language()] = gen
}

func Get(lang Language) (Generator, bool) {
    gen, ok := generators[lang]
    return gen, ok
}
```

### Go Generator

**Strategy**: Template-based generation using `text/template`.

**Template Functions:**

```go
funcMap := template.FuncMap{
    "goType":       c.goType,        // schema type → Go type
    "goFieldType":  c.goFieldType,   // field type with pointers
    "wireTypeV2":   c.wireTypeV2,    // compute wire type
    "encodeField":  c.encodeField,   // generate encode code
    "decodeField":  c.decodeField,   // generate decode code
    "jsonEncode":   c.jsonEncode,    // JSON encoding
    "jsonDecode":   c.jsonDecode,    // JSON decoding
}
```

**Type Mapping:**

| Schema Type | Go Type |
|-------------|---------|
| `bool` | `bool` |
| `int8`-`int64` | `int8`-`int64` |
| `uint8`-`uint64` | `uint8`-`uint64` |
| `float32` | `float32` |
| `float64` | `float64` |
| `string` | `string` |
| `bytes` | `[]byte` |
| `[]T` | `[]T` |
| `map[K]V` | `map[K]V` |
| `*T` | `*T` |
| `User` | `User` |

**Generated Methods:**

```go
// Binary encoding
func (m *User) EncodeTo(w *cramberry.Writer) error

// Binary decoding
func (m *User) DecodeFrom(r *cramberry.Reader) error

// JSON encoding (deterministic)
func (m *User) ToJSON() (string, error)

// JSON decoding
func (m *User) FromJSON(s string) error
```

### TypeScript Generator

**Strategy**: Programmatic string building (no templates).

**Type Mapping:**

| Schema Type | TypeScript Type |
|-------------|-----------------|
| `bool` | `boolean` |
| `int8`-`int64` | `bigint` (avoid JS precision loss) |
| `uint8`-`uint64` | `bigint` |
| `float32` | `number` |
| `float64` | `number` |
| `string` | `string` |
| `bytes` | `Uint8Array` |
| `[]T` | `Array<T>` |
| `map[K]V` | `Map<K, V>` |
| `*T` | `T \| null` |
| `User` | `User` |

**Generated Code:**

```typescript
export interface User {
    id: bigint;
    name: string;
    email: string;
}

export function encode_User(w: Writer, m: User): void {
    // Encoding logic...
}

export function decode_User(r: Reader): User {
    // Decoding logic...
}

export function toJSON_User(m: User): string {
    // JSON encoding...
}

export function fromJSON_User(s: string): User {
    // JSON decoding...
}
```

**Challenges:**

- **No int64**: Use `bigint` for all integers
- **No pointers**: Use `null` for optional fields
- **No method receivers**: Use standalone functions

### Rust Generator

**Strategy**: Programmatic generation with builder pattern.

**Type Mapping:**

| Schema Type | Rust Type |
|-------------|-----------|
| `bool` | `bool` |
| `int8`-`int64` | `i8`-`i64` |
| `uint8`-`uint64` | `u8`-`u64` |
| `float32` | `f32` |
| `float64` | `f64` |
| `string` | `String` |
| `bytes` | `Vec<u8>` |
| `[]T` | `Vec<T>` |
| `map[K]V` | `HashMap<K, V>` |
| `*T` | `Option<T>` |
| `User` | `User` |

**Generated Code:**

```rust
#[derive(Debug, Clone)]
pub struct User {
    pub id: i64,
    pub name: String,
    pub email: String,
}

impl User {
    pub fn encode_to(&self, w: &mut Writer) -> Result<()> {
        // Encoding logic...
    }

    pub fn decode_from(r: &mut Reader) -> Result<Self> {
        // Decoding logic...
    }

    pub fn to_json(&self) -> Result<String> {
        // JSON encoding...
    }

    pub fn from_json(s: &str) -> Result<Self> {
        // JSON decoding...
    }
}
```

**Challenges:**

- **Ownership**: Use `&self` for encoding, `Self` for decoding
- **No null**: Use `Option<T>` for optional fields
- **Error Handling**: Use `Result<T, Error>` everywhere

### CLI Usage

**Generate Command:**

```bash
cramberry generate -lang go -out ./gen ./schemas/*.cram
cramberry generate -lang typescript -out ./gen ./schemas/*.cram
cramberry generate -lang rust -out ./gen ./schemas/*.cram
```

**Schema Extraction:**

```bash
cramberry schema -out user.cram ./pkg/models
```

**Validation:**

```bash
cramberry validate ./schemas/*.cram
```

**Formatting:**

```bash
cramberry format ./schemas/*.cram
```

---

## Cross-Language Support

### Integration Testing Strategy

**Golden File Approach:**

1. **Go generates canonical binaries** (authoritative)
2. **TypeScript/Rust verify byte-for-byte compatibility**
3. **Golden files checked into git** (`.bin` and `.hex`)

**Test Data Categories:**

- **Scalar Types**: All primitive types (int, float, bool, string, bytes)
- **Repeated Types**: Arrays of primitives and messages
- **Nested Messages**: Deep nesting (10+ levels)
- **Complex Types**: Maps, slices, pointers, enums
- **Edge Cases**: Empty values, nil, zero, max/min int

**Test Structure:**

```
test/integration/
├── interop_test.go          # Go test (generates golden files)
├── gen/interop.go           # Generated Go code
├── ts/
│   ├── interop.ts           # Generated TS code
│   └── interop.test.ts      # TS test (verifies golden files)
├── rust/
│   ├── src/interop.rs       # Generated Rust code
│   └── src/interop_test.rs  # Rust test (verifies golden files)
└── testdata/
    └── schemas/interop.cram # Test schema
```

**Golden File Example:**

```
testdata/golden/scalar_types.bin   (binary)
testdata/golden/scalar_types.hex   (hex dump for debugging)
```

**Test Flow:**

```
1. Go: Marshal(testData) → binary
2. Go: Write binary to golden file
3. Go: Unmarshal(binary) → verify roundtrip

4. TypeScript: Read golden file
5. TypeScript: Decode(binary) → verify matches expected
6. TypeScript: Encode(data) → verify matches golden

7. Rust: Read golden file
8. Rust: Decode(binary) → verify matches expected
9. Rust: Encode(data) → verify matches golden
```

### Runtime Feature Parity

| Feature | Go | TypeScript | Rust |
|---------|----|-----------|----|
| **Binary Encoding** | ✅ | ✅ | ✅ |
| **Binary Decoding** | ✅ | ✅ | ✅ |
| **Streaming (Delimited)** | ✅ | ✅ | ✅ |
| **Type Registry** | ✅ | ✅ | ✅ |
| **Deterministic JSON** | ✅ | ✅ | ✅ |
| **Zero-Copy Strings** | ✅ | ❌ | ❌ |
| **Writer Pooling** | ✅ | ❌ | ❌ |

**Note**: Zero-copy and pooling are Go-specific optimizations not applicable to TS/Rust.

### JSON Interoperability

**Guarantees:**

1. **Identical JSON Output**: Same schema + same data → byte-identical JSON across all languages
2. **Field Order**: Fields appear in field number order (not declaration order)
3. **Map Key Sorting**: Lexicographic UTF-8 byte order
4. **Integer Encoding**: All integers as quoted strings (`"123"`)
5. **Float Precision**: Fixed significant digits (9 for float32, 17 for float64)

**Example:**

```go
// Go
user := User{ID: 123, Name: "alice"}
json, _ := user.ToJSON()
// json = `{"id":"123","name":"alice"}`
```

```typescript
// TypeScript
const user = { id: 123n, name: "alice" };
const json = toJSON_User(user);
// json = `{"id":"123","name":"alice"}`
```

```rust
// Rust
let user = User { id: 123, name: "alice".to_string() };
let json = to_json_user(&user)?;
// json = r#"{"id":"123","name":"alice"}"#
```

**Verification**: All three produce identical JSON (byte-for-byte).

---

## Testing Strategy

### Test Categories

**1. Unit Tests** (`*_test.go`, `*.test.ts`, `*_test.rs`)

- **Collocated**: Tests in same directory as source
- **Coverage Target**: >70% (Go packages: 73-92%)
- **Focus**: Individual functions, edge cases, error paths

**2. Integration Tests** (`test/integration/`)

- **Cross-Language**: Go/TS/Rust interop verification
- **Golden Files**: Canonical binaries for byte-for-byte comparison
- **Focus**: End-to-end encoding/decoding compatibility

**3. Benchmark Tests** (`internal/bench/`)

- **Comparison**: Cramberry vs Protobuf vs JSON
- **Metrics**: ns/op, B/op, allocs/op, encoded size
- **Focus**: Performance regression detection

**4. Fuzz Tests** (`fuzz_test.go`)

- **Randomized Input**: `testing.F` Go fuzzing
- **Coverage**: Edge cases, malformed input, overflow
- **Focus**: Robustness and security

**5. Security Tests** (`security_test.go`)

- **Bounds Checking**: Buffer overflow prevention
- **Integer Overflow**: Varint decoding limits
- **Depth Limits**: Stack overflow prevention
- **Focus**: Prevent crashes and memory corruption

**6. Forward Compatibility Tests** (`forward_compat_test.go`)

- **Unknown Fields**: Skip unrecognized field numbers
- **New Wire Types**: Graceful handling of future types
- **Focus**: Schema evolution without breaking

### Test Organization

```
pkg/cramberry/
├── marshal_test.go          # Marshal API tests
├── unmarshal_test.go        # Unmarshal API tests
├── writer_test.go           # Writer tests
├── reader_test.go           # Reader tests
├── stream_test.go           # Streaming tests
├── registry_test.go         # Type registry tests
├── json_test.go             # JSON serialization tests
├── types_test.go            # Type system tests
├── errors_test.go           # Error handling tests
├── concurrent_test.go       # Race condition tests (-race)
├── edge_cases_test.go       # Boundary value tests
├── forward_compat_test.go   # Compatibility tests
├── fuzz_test.go             # Fuzzing tests
├── security_test.go         # Security tests
└── benchmark_test.go        # Micro-benchmarks
```

### Table-Driven Tests

**Pattern:**

```go
func TestMarshalInt64(t *testing.T) {
    tests := []struct {
        name     string
        input    int64
        expected []byte
    }{
        {"zero", 0, []byte{0x00}},
        {"positive", 123, []byte{0x7B}},
        {"negative", -123, []byte{0xF5, 0x01}},  // ZigZag
        {"max", math.MaxInt64, []byte{...}},
        {"min", math.MinInt64, []byte{...}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w := cramberry.GetWriter()
            defer cramberry.PutWriter(w)

            w.WriteInt64(tt.input)

            if !bytes.Equal(w.Bytes(), tt.expected) {
                t.Errorf("got %x, want %x", w.Bytes(), tt.expected)
            }
        })
    }
}
```

### Coverage Metrics

**Current Coverage (Go):**

| Package | Coverage |
|---------|----------|
| `pkg/cramberry` | 73.8% |
| `pkg/schema` | 78.8% |
| `pkg/codegen` | 73.5% |
| `pkg/extract` | 82.7% |
| `internal/wire` | 92.3% |

**Coverage Tools:**

```bash
make test              # Run tests with coverage
make coverage          # Generate HTML coverage report
go tool cover -html=coverage.out
```

### CI/CD Testing

**GitHub Actions:**

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25.6'

      - run: make check  # fmt, vet, lint, test
      - run: make bench
      - run: make ts-test
      - run: make rust-test
```

---

## Error Handling

### Error Hierarchy

```
error (interface)
├── EncodeError
│   ├── MaxDepthExceededError
│   ├── MaxSizeExceededError
│   ├── RequiredFieldMissingError
│   └── TypeNotRegisteredError
├── DecodeError
│   ├── UnexpectedEOFError
│   ├── InvalidUTF8Error
│   ├── InvalidWireTypeError
│   ├── InvalidFieldNumberError
│   ├── UnknownFieldError (if strict mode)
│   └── IntegerOverflowError
├── ValidationError
│   ├── DuplicateFieldNumberError
│   ├── InvalidTypeReferenceError
│   └── CircularReferenceError
└── RegistrationError
    ├── TypeIDConflictError
    └── InterfaceNotImplementedError
```

### Structured Errors

**DecodeError:**

```go
type DecodeError struct {
    Type        string  // "User"
    Field       string  // "Email"
    FieldNumber int     // 3
    Offset      int     // 42 (byte offset in buffer)
    Message     string  // "invalid UTF-8 sequence"
    Cause       error   // Wrapped error (if any)
}

func (e *DecodeError) Error() string {
    return fmt.Sprintf("decode error in %s.%s (field %d at offset %d): %s",
        e.Type, e.Field, e.FieldNumber, e.Offset, e.Message)
}

func (e *DecodeError) Unwrap() error {
    return e.Cause
}
```

**Usage:**

```go
err := cramberry.Unmarshal(data, &user)
if err != nil {
    var decodeErr *cramberry.DecodeError
    if errors.As(err, &decodeErr) {
        log.Printf("Failed to decode %s.%s at byte %d: %v",
            decodeErr.Type, decodeErr.Field, decodeErr.Offset, decodeErr.Message)
    }
}
```

### Sentinel Errors

**Predefined Errors:**

```go
var (
    ErrUnexpectedEOF        = errors.New("unexpected end of input")
    ErrMaxDepthExceeded     = errors.New("maximum nesting depth exceeded")
    ErrMaxSizeExceeded      = errors.New("maximum message size exceeded")
    ErrInvalidUTF8          = errors.New("invalid UTF-8 string")
    ErrInvalidWireType      = errors.New("invalid wire type")
    ErrRequiredFieldMissing = errors.New("required field missing")
    ErrTypeNotRegistered    = errors.New("type not registered")
)
```

**Usage:**

```go
if errors.Is(err, cramberry.ErrMaxDepthExceeded) {
    // Handle depth limit error
}
```

### Error Propagation

**First-Error Semantics:**

```go
func (w *Writer) WriteString(s string) {
    if w.err != nil {
        return  // Already errored, no-op
    }

    if !utf8.ValidString(s) {
        w.setError(ErrInvalidUTF8)
        return
    }

    // ... encoding logic ...
}

func (w *Writer) Bytes() ([]byte, error) {
    return w.buf, w.err  // Return accumulated error
}
```

**Benefits:**

- **Single Error Check**: At end of encoding/decoding
- **No Error Cascades**: Subsequent operations are no-ops
- **Clear Failure Point**: First error is root cause

### Error Context

**Adding Context:**

```go
func (m *User) DecodeFrom(r *cramberry.Reader) error {
    for {
        tag := r.ReadFieldTag()
        if tag == 0 {
            break  // End marker
        }

        fieldNum := tag.FieldNumber()

        switch fieldNum {
        case 1:  // ID
            m.ID = r.ReadInt64()
        case 2:  // Name
            m.Name = r.ReadString()
        case 3:  // Email
            m.Email = r.ReadString()
        default:
            r.SkipField(tag.WireType())
        }

        if r.Err() != nil {
            return &cramberry.DecodeError{
                Type:        "User",
                Field:       fieldName(fieldNum),
                FieldNumber: fieldNum,
                Offset:      r.Offset(),
                Cause:       r.Err(),
            }
        }
    }

    return nil
}
```

---

## Security Considerations

### Attack Surface

**1. Malformed Input**

- **Buffer Overruns**: All buffer accesses bounds-checked
- **Integer Overflow**: Varint decoder detects overflow at 10th byte
- **Invalid UTF-8**: Strings validated before encoding/decoding

**2. Resource Exhaustion**

- **Max Depth**: 32 levels (default), prevents stack overflow
- **Max Message Size**: 64MB (default), prevents OOM
- **Max Field Number**: 536,870,911 (2^29-1), prevents excessive memory

**3. Denial of Service**

- **Infinite Loops**: End markers (0x00) prevent infinite field reads
- **Recursive Bombs**: Depth limit prevents exponential expansion
- **Memory Bombs**: Size limit prevents allocating huge buffers

### Mitigation Strategies

**1. Depth Limiting**

```go
type Options struct {
    Limits struct {
        MaxDepth int  // Default: 32
    }
}

func (r *Reader) enterDepth() error {
    r.depth++
    if r.depth > r.opts.Limits.MaxDepth {
        return ErrMaxDepthExceeded
    }
    return nil
}

func (r *Reader) exitDepth() {
    r.depth--
}
```

**2. Size Limiting**

```go
func (r *Reader) ReadBytes() []byte {
    length := r.ReadUvarint()
    if length > uint64(r.opts.Limits.MaxSize) {
        r.setError(ErrMaxSizeExceeded)
        return nil
    }

    // ... read bytes ...
}
```

**3. UTF-8 Validation**

```go
func (r *Reader) ReadString() string {
    data := r.ReadBytes()
    if !utf8.Valid(data) {
        r.setError(ErrInvalidUTF8)
        return ""
    }
    return string(data)
}
```

**4. Integer Overflow Detection**

```go
func decodeUvarint(data []byte) (uint64, int, error) {
    var result uint64
    var shift uint

    for i := 0; i < len(data); i++ {
        if i >= 10 {
            return 0, 0, ErrIntegerOverflow  // Max 10 bytes for uint64
        }

        b := data[i]
        result |= uint64(b&0x7F) << shift

        if b < 0x80 {
            return result, i+1, nil
        }

        shift += 7
        if shift >= 64 {
            return 0, 0, ErrIntegerOverflow  // Shift overflow
        }
    }

    return 0, 0, ErrUnexpectedEOF
}
```

### Security Best Practices

**1. Validate Untrusted Input**

```go
opts := cramberry.DefaultOptions()
opts.Limits.MaxDepth = 16       // Reduce for untrusted input
opts.Limits.MaxSize = 1024*1024 // 1MB max
opts.ValidateUTF8 = true        // Enable validation

err := cramberry.UnmarshalWithOptions(untrustedData, &msg, opts)
```

**2. Use Strict Mode (Reject Unknown Fields)**

```go
opts.RejectUnknownFields = true
err := cramberry.UnmarshalWithOptions(data, &msg, opts)
if err != nil {
    // Reject if extra fields present
}
```

**3. Sanitize Error Messages**

```go
// Don't leak internal details to untrusted clients
if err != nil {
    log.Printf("Decode error: %v", err)  // Detailed logging
    return errors.New("invalid request")  // Generic client error
}
```

**4. Rate Limiting**

```go
// Prevent DoS via excessive decode operations
limiter := rate.NewLimiter(1000, 100)  // 1000 req/sec, burst 100

func handler(data []byte) error {
    if !limiter.Allow() {
        return errors.New("rate limit exceeded")
    }
    return cramberry.Unmarshal(data, &msg)
}
```

### Security Auditing

**Static Analysis:**

```bash
# gosec: Security vulnerability scanner
gosec ./...

# staticcheck: Bug finder
staticcheck ./...

# golangci-lint: Aggregated linting
golangci-lint run
```

**Fuzzing:**

```bash
# Go native fuzzing (Go 1.18+)
go test -fuzz=FuzzUnmarshal -fuzztime=10m

# AFL fuzzing (external)
go-fuzz-build github.com/blockberries/cramberry/pkg/cramberry
go-fuzz -bin=./cramberry-fuzz.zip -workdir=fuzz
```

**Dependency Scanning:**

```bash
# govulncheck: Vulnerability database
govulncheck ./...

# nancy: Dependency vulnerability scanner
go list -json -deps | nancy sleuth
```

---

## Build and Deployment

### Build Configuration

**Makefile Targets:**

```makefile
build:              # Build CLI to bin/cramberry
test:               # Tests with race detection & coverage
test-short:         # Fast tests (no race detection)
bench:              # Run benchmarks
fmt:                # Code formatting (gofmt)
vet:                # Go vet analysis
lint:               # golangci-lint
check:              # All checks: fmt, vet, lint, test
ts-build/ts-test:   # TypeScript runtime
rust-build/rust-test: # Rust runtime
```

**Build Command:**

```bash
# Development build
make build

# Production build with version info
VERSION=1.5.5 COMMIT=$(git rev-parse --short HEAD) make build

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o bin/cramberry-linux ./cmd/cramberry
GOOS=darwin GOARCH=arm64 go build -o bin/cramberry-mac ./cmd/cramberry
GOOS=windows GOARCH=amd64 go build -o bin/cramberry.exe ./cmd/cramberry
```

### Linker Flags

**Version Embedding:**

```makefile
LDFLAGS := -X github.com/blockberries/cramberry/pkg/cramberry.Version=$(VERSION) \
           -X github.com/blockberries/cramberry/pkg/cramberry.GitCommit=$(COMMIT) \
           -X github.com/blockberries/cramberry/pkg/cramberry.BuildDate=$(BUILD_DATE)

go build -ldflags "$(LDFLAGS)" ./cmd/cramberry
```

**Usage:**

```bash
cramberry version
# cramberry v1.5.5 (commit: 182ceb8, built: 2026-02-02T15:30:00Z)
```

### Distribution

**Binary Releases:**

```bash
# GitHub Releases (automated via CI)
.github/workflows/release.yml

# go install
go install github.com/blockberries/cramberry/cmd/cramberry@latest

# Homebrew (future)
brew install cramberry
```

**Docker:**

```dockerfile
# Multi-stage build
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o cramberry ./cmd/cramberry

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/cramberry /usr/local/bin/
ENTRYPOINT ["cramberry"]
```

### CI/CD Pipeline

**GitHub Actions:**

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    strategy:
      matrix:
        go-version: [1.24, 1.25]
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}
      - run: make check
      - run: make ts-test
      - run: make rust-test

  release:
    if: startsWith(github.ref, 'refs/tags/v')
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: goreleaser/goreleaser-action@v4
```

### Deployment Strategies

**1. Library Usage (Go Modules)**

```bash
# Add to go.mod
go get github.com/blockberries/cramberry@latest

# In code
import "github.com/blockberries/cramberry/pkg/cramberry"
```

**2. CLI Tool**

```bash
# Install globally
go install github.com/blockberries/cramberry/cmd/cramberry@latest

# Use in project
cramberry generate -lang go -out ./gen ./schemas/*.cram
```

**3. Code Generation (Build Step)**

```makefile
# Makefile
gen:
	cramberry generate -lang go -out ./gen ./schemas/*.cram
	go fmt ./gen/...

build: gen
	go build ./...
```

**4. TypeScript/Rust Runtimes**

```bash
# TypeScript (npm)
npm install @cramberry/runtime

# Rust (crates.io)
cargo add cramberry
```

---

## Future Roadmap

### Short-Term (v1.6)

**1. Performance Enhancements**

- [ ] Generated code optimization (inline encoding)
- [ ] Buffer reuse for Reader (similar to Writer pooling)
- [ ] SIMD-accelerated varint encoding (amd64/arm64)
- [ ] Lazy field decoding (skip unused fields)

**2. Schema Evolution**

- [ ] Schema versioning (explicit version numbers)
- [ ] Deprecation warnings (mark old fields)
- [ ] Migration tools (v1 → v2 schema conversion)

**3. Developer Experience**

- [ ] godoc examples for all exported APIs
- [ ] Interactive schema playground (web-based)
- [ ] VS Code extension (schema syntax highlighting)
- [ ] Schema linter (detect anti-patterns)

### Mid-Term (v1.7-v1.8)

**1. Language Support**

- [ ] Python runtime and code generator
- [ ] Java runtime and code generator
- [ ] C++ runtime (header-only library)

**2. Protocol Features**

- [ ] Compression (zstd/gzip) at wire level
- [ ] Incremental encoding (append-only updates)
- [ ] Schema registry server (centralized schema management)

**3. Tooling**

- [ ] Benchmarking dashboard (track performance over time)
- [ ] Schema diff tool (compare versions)
- [ ] Binary inspector (decode and inspect .bin files)

### Long-Term (v2.0+)

**1. Breaking Changes**

- [ ] Wire protocol v3 (lessons learned from v2)
- [ ] Simplified type system (remove complex types?)
- [ ] Built-in versioning in messages

**2. Advanced Features**

- [ ] RPC framework (gRPC-like, but with Cramberry)
- [ ] Message validation (schema constraints)
- [ ] Reflection API (runtime schema introspection)

**3. Ecosystem**

- [ ] Package registry (share schemas publicly)
- [ ] Community schemas (common message formats)
- [ ] Integration with Buf (Protocol Buffers tooling)

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **Deterministic Encoding** | Encoding that produces identical binary output for identical logical input, regardless of encoding order or implementation details. |
| **Wire Format** | The binary representation of data on the wire (network, disk, etc.). |
| **Wire Type** | The encoding method for a field (varint, fixed32, bytes, etc.). |
| **Field Tag** | A compact encoding of field number and wire type. |
| **Varint** | Variable-length integer encoding (LEB128). |
| **ZigZag** | Signed integer encoding that maps negative numbers to small positive numbers. |
| **Packed Encoding** | Array encoding without per-element tags. |
| **Type Registry** | Mapping between type IDs and Go types for polymorphic serialization. |
| **Golden File** | Canonical binary file used for cross-language testing. |
| **Zero-Copy** | Technique to avoid copying data, using unsafe pointers. |
| **Schema Language** | Domain-specific language for defining message schemas (.cram files). |
| **Code Generation** | Automatic generation of encoding/decoding code from schemas. |

---

## Appendix B: References

**Documentation:**

- [Go Serialization Formats](https://go.dev/blog/json-and-go)
- [Protocol Buffers Encoding](https://developers.google.com/protocol-buffers/docs/encoding)
- [LEB128 (Variable-Length Encoding)](https://en.wikipedia.org/wiki/LEB128)
- [Effective Go](https://go.dev/doc/effective_go)

**Related Projects:**

- [Protocol Buffers](https://github.com/protocolbuffers/protobuf)
- [Cap'n Proto](https://capnproto.org/)
- [FlatBuffers](https://google.github.io/flatbuffers/)
- [MessagePack](https://msgpack.org/)
- [Apache Thrift](https://thrift.apache.org/)

**Blockchain Serialization:**

- [Cosmos SDK Amino](https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-019-protobuf-state-encoding.md)
- [Tendermint Encoding](https://docs.tendermint.com/v0.34/spec/core/encoding.html)

---

**Document Version:** 1.0
**Last Updated:** 2026-02-02
**Maintainer:** Blockberries Team
**License:** Apache 2.0
