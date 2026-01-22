# Cramberry Implementation Plan

## Production Roadmap

**Document Version:** 1.0.0
**Created:** 2026-01-21
**Target Production Release:** v1.0.0

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Phase Overview](#2-phase-overview)
3. [Phase 1: Foundation](#3-phase-1-foundation)
4. [Phase 2: Core Serialization](#4-phase-2-core-serialization)
5. [Phase 3: Polymorphism & Registry](#5-phase-3-polymorphism--registry)
6. [Phase 4: Schema Language & Parser](#6-phase-4-schema-language--parser)
7. [Phase 5: Code Generation](#7-phase-5-code-generation)
8. [Phase 6: Schema Extraction Tool](#8-phase-6-schema-extraction-tool)
9. [Phase 7: Streaming Support](#9-phase-7-streaming-support)
10. [Phase 8: Cross-Language Runtimes](#10-phase-8-cross-language-runtimes)
11. [Phase 9: Performance Optimization](#11-phase-9-performance-optimization)
12. [Phase 10: Security Hardening](#12-phase-10-security-hardening)
13. [Phase 11: Documentation & Examples](#13-phase-11-documentation--examples)
14. [Phase 12: Production Readiness](#14-phase-12-production-readiness)
15. [Risk Assessment & Mitigation](#15-risk-assessment--mitigation)
16. [Resource Requirements](#16-resource-requirements)
17. [Success Metrics](#17-success-metrics)
18. [Appendix: Task Breakdown](#18-appendix-task-breakdown)

---

## 1. Executive Summary

This document outlines the implementation plan for Cramberry, a high-performance binary serialization schema. The plan is organized into 12 phases, progressing from foundational primitives to production-ready release.

### Key Principles

1. **Incremental Value Delivery**: Each phase produces usable, tested artifacts
2. **Test-Driven Development**: Comprehensive tests before implementation
3. **Documentation as Code**: API docs and guides written alongside code
4. **Performance from Day One**: Benchmarks established early, optimized continuously
5. **Security by Design**: Threat modeling and hardening integrated throughout

### Critical Path

```
Phase 1 → Phase 2 → Phase 3 → Phase 5 → Phase 9 → Phase 12
(Foundation) (Core)  (Poly)    (CodeGen) (Perf)    (Release)
```

Phases 4, 6, 7, 8, 10, 11 can progress in parallel after their dependencies are met.

---

## 2. Phase Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Phase │ Name                    │ Dependencies │ Deliverables               │
├───────┼─────────────────────────┼──────────────┼────────────────────────────┤
│   1   │ Foundation              │ None         │ Wire primitives, project   │
│   2   │ Core Serialization      │ 1            │ Encode/decode, reflection  │
│   3   │ Polymorphism & Registry │ 2            │ Interface support          │
│   4   │ Schema Language         │ 1            │ SDL parser, AST            │
│   5   │ Code Generation         │ 2, 3, 4      │ Go code generator          │
│   6   │ Schema Extraction       │ 4            │ Go→Schema tool             │
│   7   │ Streaming Support       │ 2            │ Stream encode/decode       │
│   8   │ Cross-Language          │ 4, 5         │ TS, Rust runtimes          │
│   9   │ Performance             │ 2, 3, 5      │ Optimizations, benchmarks  │
│  10   │ Security Hardening      │ 2, 3, 7      │ Limits, validation         │
│  11   │ Documentation           │ All          │ Docs, examples, guides     │
│  12   │ Production Readiness    │ All          │ CI/CD, release, support    │
└───────┴─────────────────────────┴──────────────┴────────────────────────────┘
```

### Dependency Graph

```
                                    ┌─────────┐
                                    │ Phase 1 │
                                    │Foundation│
                                    └────┬────┘
                                         │
                         ┌───────────────┼───────────────┐
                         │               │               │
                         ▼               ▼               ▼
                    ┌─────────┐    ┌─────────┐    ┌─────────┐
                    │ Phase 2 │    │ Phase 4 │    │         │
                    │  Core   │    │ Schema  │    │         │
                    └────┬────┘    └────┬────┘    │         │
                         │              │         │         │
              ┌──────────┼──────────┐   │         │         │
              │          │          │   │         │         │
              ▼          ▼          ▼   ▼         │         │
         ┌─────────┐┌─────────┐┌─────────┐       │         │
         │ Phase 3 ││ Phase 7 ││ Phase 6 │       │         │
         │  Poly   ││ Stream  ││ Extract │       │         │
         └────┬────┘└────┬────┘└─────────┘       │         │
              │          │                        │         │
              │          │    ┌───────────────────┘         │
              ▼          │    ▼                             │
         ┌─────────┐     │┌─────────┐                      │
         │ Phase 5 │     ││ Phase 8 │                      │
         │ CodeGen │◄────┘│Cross-Lang│                      │
         └────┬────┘      └─────────┘                      │
              │                                             │
    ┌─────────┼─────────┐                                  │
    │         │         │                                  │
    ▼         ▼         ▼                                  │
┌─────────┐┌─────────┐┌─────────┐                         │
│ Phase 9 ││Phase 10 ││Phase 11 │◄────────────────────────┘
│  Perf   ││Security ││  Docs   │
└────┬────┘└────┬────┘└────┬────┘
     │          │          │
     └──────────┼──────────┘
                ▼
           ┌─────────┐
           │Phase 12 │
           │Production│
           └─────────┘
```

---

## 3. Phase 1: Foundation

### Objective
Establish project structure, build system, and implement wire format primitives.

### Prerequisites
- Go 1.21+ installed
- Git repository initialized

### 3.1 Project Setup

#### Task 1.1.1: Repository Structure
```
cramberry/
├── .github/
│   └── workflows/
│       ├── ci.yml
│       ├── release.yml
│       └── benchmark.yml
├── cmd/
│   └── cramberry/          # CLI tool (later phases)
├── pkg/
│   └── cramberry/          # Public API
│       ├── cramberry.go    # Top-level exports
│       ├── options.go      # Configuration
│       └── errors.go       # Error types
├── internal/
│   ├── wire/               # Wire format primitives
│   │   ├── varint.go
│   │   ├── varint_test.go
│   │   ├── fixed.go
│   │   ├── fixed_test.go
│   │   └── tag.go
│   ├── encoding/           # Encoding logic
│   ├── decoding/           # Decoding logic
│   ├── registry/           # Type registry
│   ├── reflect/            # Reflection utilities
│   ├── schema/             # Schema parser
│   └── codegen/            # Code generator
├── testdata/               # Test fixtures
├── examples/               # Example code
├── docs/                   # Documentation
├── schema/                 # Standard library schemas
├── go.mod
├── go.sum
├── Makefile
├── ARCHITECTURE.md
├── IMPLEMENTATION_PLAN.md
├── README.md
├── LICENSE
└── CHANGELOG.md
```

#### Task 1.1.2: Go Module Initialization
```bash
go mod init github.com/blockberries/cramberry
```

#### Task 1.1.3: Makefile Setup
```makefile
.PHONY: all build test bench lint fmt vet generate clean

GO := go
GOFLAGS := -v
TESTFLAGS := -race -coverprofile=coverage.out

all: fmt vet lint test build

build:
	$(GO) build $(GOFLAGS) ./...

test:
	$(GO) test $(TESTFLAGS) ./...

bench:
	$(GO) test -bench=. -benchmem ./...

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

generate:
	$(GO) generate ./...

clean:
	rm -rf bin/ coverage.out
```

#### Task 1.1.4: CI Configuration
```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.21', '1.22', '1.23']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
      - run: make test
      - uses: codecov/codecov-action@v4

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golangci/golangci-lint-action@v4

  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: make bench | tee bench.txt
      - uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: bench.txt
```

### 3.2 Wire Format Primitives

#### Task 1.2.1: Varint Encoding/Decoding

**File:** `internal/wire/varint.go`

```go
package wire

// AppendUvarint appends a uint64 as a varint to buf
func AppendUvarint(buf []byte, v uint64) []byte

// AppendSvarint appends an int64 as a zigzag-encoded varint to buf
func AppendSvarint(buf []byte, v int64) []byte

// DecodeUvarint decodes a varint from data, returning value and bytes consumed
func DecodeUvarint(data []byte) (uint64, int, error)

// DecodeSvarint decodes a zigzag varint from data
func DecodeSvarint(data []byte) (int64, int, error)

// UvarintSize returns the encoded size of a uint64
func UvarintSize(v uint64) int

// SvarintSize returns the encoded size of an int64 (zigzag)
func SvarintSize(v int64) int
```

**Test Cases (Task 1.2.2):**
- Boundary values: 0, 1, 127, 128, 16383, 16384, max uint64
- Negative values: -1, -128, min int64
- Round-trip encoding/decoding
- Invalid input handling (truncated, overflow)
- Benchmark: encode/decode throughput

#### Task 1.2.3: Fixed-Width Encoding/Decoding

**File:** `internal/wire/fixed.go`

```go
package wire

// AppendFixed32 appends a uint32 in little-endian format
func AppendFixed32(buf []byte, v uint32) []byte

// AppendFixed64 appends a uint64 in little-endian format
func AppendFixed64(buf []byte, v uint64) []byte

// DecodeFixed32 decodes a little-endian uint32
func DecodeFixed32(data []byte) (uint32, error)

// DecodeFixed64 decodes a little-endian uint64
func DecodeFixed64(data []byte) (uint64, error)

// Float32 encoding (with canonicalization)
func AppendFloat32(buf []byte, v float32) []byte
func DecodeFloat32(data []byte) (float32, error)

// Float64 encoding (with canonicalization)
func AppendFloat64(buf []byte, v float64) []byte
func DecodeFloat64(data []byte) (float64, error)
```

**Test Cases (Task 1.2.4):**
- All bit patterns
- Float special cases: +0, -0, NaN, +Inf, -Inf, subnormals
- Canonicalization: -0 → +0, NaN normalization
- Endianness verification

#### Task 1.2.5: Tag Encoding/Decoding

**File:** `internal/wire/tag.go`

```go
package wire

type WireType uint8

const (
    WireVarint  WireType = 0
    WireFixed64 WireType = 1
    WireBytes   WireType = 2
    WireFixed32 WireType = 5
    WireSVarint WireType = 6
    WireTypeRef WireType = 7
)

// AppendTag appends a field tag (field_num << 3 | wire_type)
func AppendTag(buf []byte, fieldNum int, wireType WireType) []byte

// DecodeTag decodes a field tag
func DecodeTag(data []byte) (fieldNum int, wireType WireType, n int, err error)

// TagSize returns the encoded size of a tag
func TagSize(fieldNum int) int
```

### 3.3 Core Types and Errors

#### Task 1.3.1: Error Definitions

**File:** `pkg/cramberry/errors.go`

```go
package cramberry

import "errors"

var (
    ErrInvalidVarint     = errors.New("cramberry: invalid varint")
    ErrUnexpectedEOF     = errors.New("cramberry: unexpected end of data")
    ErrInvalidWireType   = errors.New("cramberry: invalid wire type")
    ErrUnknownType       = errors.New("cramberry: unknown type")
    ErrTypeMismatch      = errors.New("cramberry: type mismatch")
    ErrNotPointer        = errors.New("cramberry: target must be a pointer")
    ErrNilPointer        = errors.New("cramberry: nil pointer")
    ErrMaxDepthExceeded  = errors.New("cramberry: max nesting depth exceeded")
    ErrMaxSizeExceeded   = errors.New("cramberry: max message size exceeded")
    ErrInvalidUTF8       = errors.New("cramberry: invalid UTF-8 string")
    ErrDuplicateType     = errors.New("cramberry: duplicate type registration")
    ErrUnregisteredType  = errors.New("cramberry: unregistered type")
)

// DecodeError provides context for decoding failures
type DecodeError struct {
    Field   string
    Offset  int
    Message string
    Cause   error
}

func (e *DecodeError) Error() string
func (e *DecodeError) Unwrap() error
```

#### Task 1.3.2: Type Definitions

**File:** `pkg/cramberry/types.go`

```go
package cramberry

// TypeID uniquely identifies a registered type
type TypeID uint32

// Reserved type ID ranges
const (
    TypeIDNil      TypeID = 0
    TypeIDBuiltinStart    = 1
    TypeIDBuiltinEnd      = 63
    TypeIDStdlibStart     = 64
    TypeIDStdlibEnd       = 127
    TypeIDUserStart       = 128
)

// WireType re-exported from internal
type WireType = wire.WireType

const (
    WireVarint  = wire.WireVarint
    WireFixed64 = wire.WireFixed64
    WireBytes   = wire.WireBytes
    WireFixed32 = wire.WireFixed32
    WireSVarint = wire.WireSVarint
    WireTypeRef = wire.WireTypeRef
)
```

### 3.4 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| Repository | Initialized with structure | All directories created, go.mod present |
| CI Pipeline | GitHub Actions configured | Tests run on push, linting enforced |
| Varint | Encode/decode functions | 100% test coverage, benchmarks passing |
| Fixed | Fixed-width encode/decode | Float canonicalization verified |
| Tags | Tag encode/decode | All wire types supported |
| Errors | Error types defined | Wrapped errors with context |
| Types | Core type definitions | TypeID, WireType exported |

### 3.5 Exit Criteria

- [ ] All tests passing with >95% coverage
- [ ] Benchmarks established as baseline
- [ ] golangci-lint passing with no warnings
- [ ] Package documentation complete

---

## 4. Phase 2: Core Serialization

### Objective
Implement Writer/Reader types and reflection-based encoding/decoding for all Go primitives and composite types.

### Prerequisites
- Phase 1 complete

### 4.1 Writer Implementation

#### Task 2.1.1: Buffer Writer

**File:** `internal/encoding/writer.go`

```go
package encoding

// Writer provides methods for encoding cramberry data
type Writer struct {
    buf      []byte
    err      error
}

// Core methods
func NewWriter(capacity int) *Writer
func (w *Writer) Reset()
func (w *Writer) Bytes() []byte
func (w *Writer) Len() int
func (w *Writer) Error() error

// Primitive writes
func (w *Writer) WriteUvarint(v uint64)
func (w *Writer) WriteSvarint(v int64)
func (w *Writer) WriteFixed32(v uint32)
func (w *Writer) WriteFixed64(v uint64)
func (w *Writer) WriteFloat32(v float32)
func (w *Writer) WriteFloat64(v float64)
func (w *Writer) WriteBool(v bool)
func (w *Writer) WriteByte(v byte)

// Compound writes
func (w *Writer) WriteBytes(data []byte)
func (w *Writer) WriteString(s string)
func (w *Writer) WriteTag(fieldNum int, wireType WireType)

// Raw writes (no length prefix)
func (w *Writer) WriteRaw(data []byte)
```

#### Task 2.1.2: Writer Tests
- All primitive types
- Buffer growth behavior
- Error accumulation
- Concurrent safety (or documented as unsafe)

### 4.2 Reader Implementation

#### Task 2.2.1: Buffer Reader

**File:** `internal/decoding/reader.go`

```go
package decoding

// Reader provides methods for decoding cramberry data
type Reader struct {
    data   []byte
    pos    int
    config *Config
}

// Core methods
func NewReader(data []byte) *Reader
func NewReaderWithConfig(data []byte, config *Config) *Reader
func (r *Reader) EOF() bool
func (r *Reader) Remaining() int
func (r *Reader) Position() int
func (r *Reader) Peek(n int) ([]byte, error)

// Primitive reads
func (r *Reader) ReadUvarint() (uint64, error)
func (r *Reader) ReadSvarint() (int64, error)
func (r *Reader) ReadFixed32() (uint32, error)
func (r *Reader) ReadFixed64() (uint64, error)
func (r *Reader) ReadFloat32() (float32, error)
func (r *Reader) ReadFloat64() (float64, error)
func (r *Reader) ReadBool() (bool, error)
func (r *Reader) ReadByte() (byte, error)

// Compound reads
func (r *Reader) ReadBytes() ([]byte, error)
func (r *Reader) ReadString() (string, error)
func (r *Reader) ReadTag() (fieldNum int, wireType WireType, err error)

// Utility
func (r *Reader) Skip(wireType WireType) error
func (r *Reader) SubReader(length int) (*Reader, error)
```

#### Task 2.2.2: Reader Configuration

**File:** `internal/decoding/config.go`

```go
package decoding

type Config struct {
    MaxMessageSize  int64
    MaxDepth        int
    MaxStringLength int
    MaxArrayLength  int
    ValidateUTF8    bool
    StrictMode      bool
}

var DefaultConfig = &Config{
    MaxMessageSize:  64 * 1024 * 1024,
    MaxDepth:        100,
    MaxStringLength: 10 * 1024 * 1024,
    MaxArrayLength:  1_000_000,
    ValidateUTF8:    true,
    StrictMode:      false,
}
```

### 4.3 Reflection-Based Encoding

#### Task 2.3.1: Type Info Cache

**File:** `internal/reflect/typeinfo.go`

```go
package reflect

import (
    "reflect"
    "sync"
)

// TypeInfo caches encoding information for a type
type TypeInfo struct {
    Type       reflect.Type
    Kind       reflect.Kind
    Fields     []*FieldInfo  // For structs
    Elem       *TypeInfo     // For slices, arrays, pointers, maps
    Key        *TypeInfo     // For maps
    Encoder    EncoderFunc
    Decoder    DecoderFunc
    Sizer      SizerFunc
}

// FieldInfo caches information for a struct field
type FieldInfo struct {
    Name       string
    Index      []int         // Field index for embedded structs
    FieldNum   int           // Wire field number
    WireType   WireType
    OmitEmpty  bool
    TypeInfo   *TypeInfo
    Encoder    FieldEncoderFunc
    Decoder    FieldDecoderFunc
}

// TypeInfoCache provides thread-safe type info caching
type TypeInfoCache struct {
    mu    sync.RWMutex
    cache map[reflect.Type]*TypeInfo
}

func (c *TypeInfoCache) Get(t reflect.Type) *TypeInfo
func (c *TypeInfoCache) GetOrCreate(t reflect.Type) *TypeInfo
```

#### Task 2.3.2: Struct Tag Parsing

**File:** `internal/reflect/tags.go`

```go
package reflect

// StructTag represents parsed cramberry struct tag
type StructTag struct {
    FieldNum   int
    Name       string
    OmitEmpty  bool
    Optional   bool
    Skip       bool
    Encoding   string
    Deprecated string
    Fixed      bool
}

// ParseTag parses a cramberry struct tag
func ParseTag(tag string, fieldIndex int) (*StructTag, error)
```

**Tag Format:**
```
cramberry:"fieldNum[,option]*"

Options:
  name=value    - Override field name
  omitempty     - Omit zero values
  optional      - Field is optional (presence tracked)
  -             - Skip field
  encoding=X    - Custom encoding (delta, rle, dict, etc.)
  deprecated=X  - Mark deprecated with message
  fixed         - Fixed-size encoding (no length prefix)
```

#### Task 2.3.3: Reflection Encoder

**File:** `internal/reflect/encoder.go`

```go
package reflect

// EncodeValue encodes a reflect.Value to the writer
func EncodeValue(w *encoding.Writer, v reflect.Value, info *TypeInfo) error

// Type-specific encoders
func encodeBool(w *encoding.Writer, v reflect.Value) error
func encodeInt(w *encoding.Writer, v reflect.Value) error
func encodeUint(w *encoding.Writer, v reflect.Value) error
func encodeFloat32(w *encoding.Writer, v reflect.Value) error
func encodeFloat64(w *encoding.Writer, v reflect.Value) error
func encodeComplex64(w *encoding.Writer, v reflect.Value) error
func encodeComplex128(w *encoding.Writer, v reflect.Value) error
func encodeString(w *encoding.Writer, v reflect.Value) error
func encodeBytes(w *encoding.Writer, v reflect.Value) error
func encodeSlice(w *encoding.Writer, v reflect.Value, info *TypeInfo) error
func encodeArray(w *encoding.Writer, v reflect.Value, info *TypeInfo) error
func encodeMap(w *encoding.Writer, v reflect.Value, info *TypeInfo) error
func encodeStruct(w *encoding.Writer, v reflect.Value, info *TypeInfo) error
func encodePointer(w *encoding.Writer, v reflect.Value, info *TypeInfo) error
func encodeTime(w *encoding.Writer, v reflect.Value) error
func encodeDuration(w *encoding.Writer, v reflect.Value) error
```

#### Task 2.3.4: Reflection Decoder

**File:** `internal/reflect/decoder.go`

```go
package reflect

// DecodeValue decodes from reader into a reflect.Value
func DecodeValue(r *decoding.Reader, v reflect.Value, info *TypeInfo) error

// Type-specific decoders (parallel to encoders)
func decodeBool(r *decoding.Reader, v reflect.Value) error
func decodeInt(r *decoding.Reader, v reflect.Value) error
// ... etc
```

### 4.4 Map Key Ordering

#### Task 2.4.1: Deterministic Map Encoding

**File:** `internal/reflect/mapkeys.go`

```go
package reflect

// SortMapKeys returns map keys sorted by their encoded representation
func SortMapKeys(keys []reflect.Value, keyInfo *TypeInfo) []reflect.Value

// CompareEncodedKeys compares two encoded key byte slices
func CompareEncodedKeys(a, b []byte) int
```

**Algorithm:**
1. Encode each key to bytes
2. Sort by lexicographic byte comparison
3. Encode key-value pairs in sorted order

### 4.5 Public API

#### Task 2.5.1: Top-Level Functions

**File:** `pkg/cramberry/cramberry.go`

```go
package cramberry

// Marshal encodes a value to bytes
func Marshal(v any) ([]byte, error)

// MarshalAppend appends encoded value to existing buffer
func MarshalAppend(buf []byte, v any) ([]byte, error)

// Unmarshal decodes bytes into a value
func Unmarshal(data []byte, v any) error

// Size returns the encoded size of a value
func Size(v any) int

// Equal compares two values by their encoded representation
func Equal(a, b any) bool

// Clone creates a deep copy via encode/decode
func Clone[T any](v T) (T, error)
```

#### Task 2.5.2: Interfaces for Custom Types

**File:** `pkg/cramberry/interfaces.go`

```go
package cramberry

// Marshaler is implemented by types with custom encoding
type Marshaler interface {
    MarshalCramberry() ([]byte, error)
}

// Unmarshaler is implemented by types with custom decoding
type Unmarshaler interface {
    UnmarshalCramberry([]byte) error
}

// Appender can append encoded form to a buffer (more efficient)
type Appender interface {
    AppendCramberry([]byte) ([]byte, error)
}

// Sizer returns the encoded size (enables buffer pre-allocation)
type Sizer interface {
    CramberrySize() int
}

// StreamEncoder encodes to a streaming writer
type StreamEncoder interface {
    EncodeCramberry(w *Writer) error
}

// StreamDecoder decodes from a streaming reader
type StreamDecoder interface {
    DecodeCramberry(r *Reader) error
}
```

### 4.6 Test Suite

#### Task 2.6.1: Primitive Type Tests
- All Go primitive types
- Boundary values
- Zero values

#### Task 2.6.2: Composite Type Tests
- Slices (empty, single, multiple)
- Arrays (fixed size)
- Maps (empty, various key types, ordering)
- Nested structs
- Pointers (nil and non-nil)
- Embedded structs

#### Task 2.6.3: Special Type Tests
- time.Time (various timezones, edge cases)
- time.Duration
- Custom types with underlying primitives

#### Task 2.6.4: Round-Trip Tests
- Encode → Decode produces identical value
- Determinism: multiple encodes produce identical bytes

#### Task 2.6.5: Error Handling Tests
- Truncated data
- Invalid wire types
- Malformed varints
- Exceeded limits

### 4.7 Benchmarks

#### Task 2.7.1: Benchmark Suite

**File:** `pkg/cramberry/benchmark_test.go`

```go
func BenchmarkMarshalSmallStruct(b *testing.B)
func BenchmarkMarshalMediumStruct(b *testing.B)
func BenchmarkMarshalLargeStruct(b *testing.B)
func BenchmarkMarshalSlice1000(b *testing.B)
func BenchmarkMarshalMap1000(b *testing.B)
func BenchmarkMarshalNested10(b *testing.B)

func BenchmarkUnmarshalSmallStruct(b *testing.B)
func BenchmarkUnmarshalMediumStruct(b *testing.B)
func BenchmarkUnmarshalLargeStruct(b *testing.B)
func BenchmarkUnmarshalSlice1000(b *testing.B)
func BenchmarkUnmarshalMap1000(b *testing.B)
func BenchmarkUnmarshalNested10(b *testing.B)

func BenchmarkSize(b *testing.B)
func BenchmarkEqual(b *testing.B)
```

### 4.8 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| Writer | Buffer-based encoder | All primitives, compounds |
| Reader | Buffer-based decoder | All primitives, compounds |
| TypeInfo | Reflection cache | Cached field info |
| Struct Tags | Tag parsing | All options supported |
| Encoder | Reflection encoder | All Go types |
| Decoder | Reflection decoder | All Go types |
| Map Ordering | Deterministic maps | Byte-sorted keys |
| Public API | Marshal/Unmarshal | Simple top-level API |
| Interfaces | Custom type support | All interfaces defined |
| Tests | Comprehensive suite | >95% coverage |
| Benchmarks | Performance baseline | Established metrics |

### 4.9 Exit Criteria

- [ ] All Go primitive types encode/decode correctly
- [ ] All composite types (slice, array, map, struct, pointer) work
- [ ] Determinism verified: same input → same output
- [ ] Round-trip integrity verified
- [ ] Benchmark baseline established
- [ ] >95% test coverage

---

## 5. Phase 3: Polymorphism & Registry

### Objective
Implement type registration system and interface-based polymorphic serialization.

### Prerequisites
- Phase 2 complete

### 5.1 Type Registry

#### Task 3.1.1: Registry Core

**File:** `internal/registry/registry.go`

```go
package registry

import (
    "reflect"
    "sync"
)

// Registry manages type registrations for polymorphic encoding
type Registry struct {
    mu sync.RWMutex

    // Forward mappings
    idToType   map[TypeID]reflect.Type
    idToName   map[TypeID]string
    idToCodec  map[TypeID]any  // Codec[T]

    // Reverse mappings
    typeToID   map[reflect.Type]TypeID
    nameToID   map[string]TypeID

    // Interface → implementations
    interfaces map[reflect.Type][]TypeID

    // Next auto-assigned ID
    nextID TypeID
}

// NewRegistry creates a new type registry
func NewRegistry() *Registry

// Clone creates a copy of the registry
func (r *Registry) Clone() *Registry

// Merge combines another registry into this one
func (r *Registry) Merge(other *Registry) error
```

#### Task 3.1.2: Type Registration

**File:** `internal/registry/register.go`

```go
package registry

// RegisterType registers a concrete type
func (r *Registry) RegisterType(typ reflect.Type, name string, id TypeID) (TypeID, error)

// RegisterInterface registers an interface type
func (r *Registry) RegisterInterface(iface reflect.Type, name string) error

// RegisterImplementation registers a type as implementing an interface
func (r *Registry) RegisterImplementation(iface, impl reflect.Type, id TypeID, name string) error

// RegisterCodec registers a custom codec for a type
func (r *Registry) RegisterCodec(typ reflect.Type, codec any) error

// RegisterAnyType registers a type that may be stored in interface{}
func (r *Registry) RegisterAnyType(typ reflect.Type, name string, id TypeID) error
```

#### Task 3.1.3: Type Lookup

**File:** `internal/registry/lookup.go`

```go
package registry

// TypeIDFor returns the TypeID for a value
func (r *Registry) TypeIDFor(v any) (TypeID, bool)

// TypeForID returns the reflect.Type for a TypeID
func (r *Registry) TypeForID(id TypeID) (reflect.Type, bool)

// NameForID returns the type name for a TypeID
func (r *Registry) NameForID(id TypeID) (string, bool)

// IDForName returns the TypeID for a type name
func (r *Registry) IDForName(name string) (TypeID, bool)

// ImplementationsOf returns all implementations of an interface
func (r *Registry) ImplementationsOf(iface reflect.Type) []TypeID

// CodecFor returns the custom codec for a type, if registered
func (r *Registry) CodecFor(typ reflect.Type) (any, bool)
```

### 5.2 Generic Registration API

#### Task 3.2.1: Type-Safe Registration Functions

**File:** `pkg/cramberry/register.go`

```go
package cramberry

// RegisterType registers a concrete type with auto-assigned ID
func RegisterType[T any](name string) TypeID

// RegisterTypeWithID registers a concrete type with explicit ID
func RegisterTypeWithID[T any](name string, id TypeID) TypeID

// RegisterInterface registers an interface type
func RegisterInterface[I any](name string)

// RegisterImplementation registers a type as implementing an interface
func RegisterImplementation[I any, T any](id TypeID, name string)

// RegisterCodec registers a custom codec for a type
func RegisterCodec[T any](codec Codec[T])

// MustRegister panics if registration fails (for init())
func MustRegisterType[T any](name string) TypeID
func MustRegisterImplementation[I any, T any](id TypeID, name string)
```

### 5.3 Interface Encoding/Decoding

#### Task 3.3.1: Interface Writer

**File:** `internal/encoding/interface.go`

```go
package encoding

// WriteInterface writes a polymorphic interface value
func (w *Writer) WriteInterface(v any, registry *Registry) error

// WriteInterfaceWithMode writes using specified discrimination mode
func (w *Writer) WriteInterfaceWithMode(v any, registry *Registry, mode DiscriminationMode) error

type DiscriminationMode int

const (
    DiscriminationTypeID   DiscriminationMode = iota  // Compact: type ID
    DiscriminationTypeName                             // Self-describing: type name
    DiscriminationTypeHash                             // Fixed-size: 4-byte hash
)
```

#### Task 3.3.2: Interface Reader

**File:** `internal/decoding/interface.go`

```go
package decoding

// ReadInterface reads a polymorphic interface value
func ReadInterface[T any](r *Reader, registry *Registry) (T, error)

// ReadInterfaceAny reads to interface{} (requires type in registry)
func ReadInterfaceAny(r *Reader, registry *Registry) (any, error)
```

### 5.4 Built-in Type Registration

#### Task 3.4.1: Standard Library Types

**File:** `internal/registry/builtins.go`

```go
package registry

func (r *Registry) registerBuiltins() {
    // Primitives (1-63)
    r.registerBuiltin(reflect.TypeOf(false), "bool", 1)
    r.registerBuiltin(reflect.TypeOf(int8(0)), "int8", 2)
    r.registerBuiltin(reflect.TypeOf(int16(0)), "int16", 3)
    r.registerBuiltin(reflect.TypeOf(int32(0)), "int32", 4)
    r.registerBuiltin(reflect.TypeOf(int64(0)), "int64", 5)
    r.registerBuiltin(reflect.TypeOf(int(0)), "int", 6)
    r.registerBuiltin(reflect.TypeOf(uint8(0)), "uint8", 7)
    r.registerBuiltin(reflect.TypeOf(uint16(0)), "uint16", 8)
    r.registerBuiltin(reflect.TypeOf(uint32(0)), "uint32", 9)
    r.registerBuiltin(reflect.TypeOf(uint64(0)), "uint64", 10)
    r.registerBuiltin(reflect.TypeOf(uint(0)), "uint", 11)
    r.registerBuiltin(reflect.TypeOf(float32(0)), "float32", 12)
    r.registerBuiltin(reflect.TypeOf(float64(0)), "float64", 13)
    r.registerBuiltin(reflect.TypeOf(complex64(0)), "complex64", 14)
    r.registerBuiltin(reflect.TypeOf(complex128(0)), "complex128", 15)
    r.registerBuiltin(reflect.TypeOf(""), "string", 16)
    r.registerBuiltin(reflect.TypeOf([]byte{}), "bytes", 17)

    // Standard library (64-127)
    r.registerBuiltin(reflect.TypeOf(time.Time{}), "time.Time", 64)
    r.registerBuiltin(reflect.TypeOf(time.Duration(0)), "time.Duration", 65)
}
```

### 5.5 Custom Codec Support

#### Task 3.5.1: Codec Interface

**File:** `pkg/cramberry/codec.go`

```go
package cramberry

// Codec defines custom encoding/decoding for a type
type Codec[T any] interface {
    Encode(w *Writer, value T) error
    Decode(r *Reader) (T, error)
    Size(value T) int
}

// CodecFunc creates a Codec from functions
func CodecFunc[T any](
    encode func(*Writer, T) error,
    decode func(*Reader) (T, error),
    size func(T) int,
) Codec[T]
```

#### Task 3.5.2: Built-in Codecs

**File:** `pkg/cramberry/codecs.go`

```go
package cramberry

// UUIDCodec for [16]byte UUIDs
var UUIDCodec = CodecFunc[UUID](...)

// BigIntCodec for *big.Int
var BigIntCodec = CodecFunc[*big.Int](...)

// DecimalCodec for decimal types
var DecimalCodec = CodecFunc[Decimal](...)
```

### 5.6 Default Registry

#### Task 3.6.1: Global Registry

**File:** `pkg/cramberry/default.go`

```go
package cramberry

// DefaultRegistry is the global default registry
var DefaultRegistry = NewRegistry()

// UseRegistry sets a custom registry for subsequent operations
func UseRegistry(r *Registry)

// WithRegistry returns options using a specific registry
func WithRegistry(r *Registry) Option
```

### 5.7 Test Suite

#### Task 3.7.1: Registry Tests
- Register/lookup types
- Duplicate registration handling
- Interface implementation tracking
- Thread safety

#### Task 3.7.2: Polymorphic Encoding Tests
- Interface with single implementation
- Interface with multiple implementations
- Nested interfaces
- nil interface values
- Unregistered type handling

#### Task 3.7.3: Custom Codec Tests
- UUID encoding (16 bytes fixed)
- BigInt encoding (variable)
- Round-trip with custom codecs

### 5.8 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| Registry | Type registration system | Thread-safe, bi-directional |
| Register API | Type-safe generics | Compile-time safety |
| Interface Encode | Polymorphic writing | Type ID prefix |
| Interface Decode | Polymorphic reading | Type-safe return |
| Built-ins | Standard type registration | All primitives, time |
| Custom Codecs | Codec interface | UUID, BigInt examples |
| Default Registry | Global registry | Auto-registration in init |
| Tests | Polymorphism tests | All scenarios covered |

### 5.9 Exit Criteria

- [ ] Types can be registered with explicit or auto IDs
- [ ] Interfaces can have multiple registered implementations
- [ ] Polymorphic encode/decode works for interfaces
- [ ] Custom codecs override default encoding
- [ ] Thread-safe concurrent access verified
- [ ] Amino-style API compatibility demonstrated

---

## 6. Phase 4: Schema Language & Parser

### Objective
Design and implement the Cramberry Schema Definition Language (SDL) parser.

### Prerequisites
- Phase 1 complete (for wire types)

### 6.1 Lexer

#### Task 4.1.1: Token Definitions

**File:** `internal/schema/token/token.go`

```go
package token

type TokenType int

const (
    // Literals
    IDENT TokenType = iota
    INT
    FLOAT
    STRING

    // Keywords
    PACKAGE
    IMPORT
    TYPE
    ENUM
    INTERFACE
    SERVICE
    RPC
    STREAM
    MAP
    EXTENDS
    AS

    // Delimiters
    LBRACE      // {
    RBRACE      // }
    LPAREN      // (
    RPAREN      // )
    LBRACKET    // [
    RBRACKET    // ]
    LANGLE      // <
    RANGLE      // >
    SEMICOLON   // ;
    COLON       // :
    COMMA       // ,
    DOT         // .
    EQUALS      // =
    QUESTION    // ?
    AT          // @

    // Special
    EOF
    ILLEGAL
    COMMENT
)

type Token struct {
    Type    TokenType
    Literal string
    Line    int
    Column  int
}
```

#### Task 4.1.2: Lexer Implementation

**File:** `internal/schema/lexer/lexer.go`

```go
package lexer

type Lexer struct {
    input   string
    pos     int
    line    int
    column  int
}

func New(input string) *Lexer
func (l *Lexer) NextToken() token.Token
func (l *Lexer) PeekToken() token.Token
```

### 6.2 Abstract Syntax Tree

#### Task 4.2.1: AST Node Definitions

**File:** `internal/schema/ast/ast.go`

```go
package ast

// Node is the base interface for all AST nodes
type Node interface {
    Pos() Position
    End() Position
}

// Position in source file
type Position struct {
    Filename string
    Line     int
    Column   int
}

// Schema is the root node
type Schema struct {
    Package     *PackageDecl
    Imports     []*ImportDecl
    Declarations []Declaration
}

// Declaration is a top-level declaration
type Declaration interface {
    Node
    declarationNode()
}

// PackageDecl: package foo.bar;
type PackageDecl struct {
    Name *QualifiedName
}

// ImportDecl: import "path" as alias;
type ImportDecl struct {
    Path  string
    Alias string
}

// TypeDecl: type Foo { ... }
type TypeDecl struct {
    Annotations []*Annotation
    Name        string
    TypeParams  []string        // Generic parameters
    Extends     *TypeRef        // Optional base type
    Fields      []*FieldDecl
}

// FieldDecl: @field(1) name: type;
type FieldDecl struct {
    Annotations []*Annotation
    Name        string
    Type        *TypeRef
    Default     Expression      // Optional default value
}

// EnumDecl: enum Status { ... }
type EnumDecl struct {
    Annotations []*Annotation
    Name        string
    Values      []*EnumValue
}

// EnumValue: PENDING = 0;
type EnumValue struct {
    Name  string
    Value int
}

// InterfaceDecl: interface Handler { ... }
type InterfaceDecl struct {
    Annotations []*Annotation
    Name        string
    Methods     []*MethodDecl
}

// MethodDecl: handle(msg: Message): Result;
type MethodDecl struct {
    Name    string
    Params  []*ParamDecl
    Returns *TypeRef
}

// ParamDecl: name: type
type ParamDecl struct {
    Name string
    Type *TypeRef
}

// ServiceDecl: service FooService { ... }
type ServiceDecl struct {
    Name    string
    Methods []*RpcDecl
}

// RpcDecl: rpc Method(Request): Response;
type RpcDecl struct {
    Name          string
    Input         *TypeRef
    Output        *TypeRef
    ClientStream  bool
    ServerStream  bool
}

// TypeRef represents a type reference
type TypeRef struct {
    Name       *QualifiedName  // For named types
    TypeArgs   []*TypeRef      // Generic arguments
    IsArray    bool
    ArraySize  int             // -1 for slices
    IsMap      bool
    KeyType    *TypeRef
    ValueType  *TypeRef
    IsOptional bool
}

// QualifiedName: foo.bar.Baz
type QualifiedName struct {
    Parts []string
}

// Annotation: @name(args)
type Annotation struct {
    Name string
    Args map[string]Expression
}

// Expression for default values and annotation args
type Expression interface {
    Node
    expressionNode()
}

type StringLit struct { Value string }
type IntLit struct { Value int64 }
type FloatLit struct { Value float64 }
type BoolLit struct { Value bool }
type ArrayLit struct { Elements []Expression }
```

### 6.3 Parser

#### Task 4.3.1: Parser Implementation

**File:** `internal/schema/parser/parser.go`

```go
package parser

type Parser struct {
    lexer   *lexer.Lexer
    current token.Token
    peek    token.Token
    errors  []error
}

func New(l *lexer.Lexer) *Parser
func (p *Parser) Parse() (*ast.Schema, error)

// Top-level parsing
func (p *Parser) parseSchema() *ast.Schema
func (p *Parser) parsePackageDecl() *ast.PackageDecl
func (p *Parser) parseImportDecl() *ast.ImportDecl
func (p *Parser) parseDeclaration() ast.Declaration

// Declaration parsing
func (p *Parser) parseTypeDecl() *ast.TypeDecl
func (p *Parser) parseEnumDecl() *ast.EnumDecl
func (p *Parser) parseInterfaceDecl() *ast.InterfaceDecl
func (p *Parser) parseServiceDecl() *ast.ServiceDecl

// Component parsing
func (p *Parser) parseFieldDecl() *ast.FieldDecl
func (p *Parser) parseTypeRef() *ast.TypeRef
func (p *Parser) parseAnnotations() []*ast.Annotation
func (p *Parser) parseExpression() ast.Expression
```

#### Task 4.3.2: Error Recovery

**File:** `internal/schema/parser/errors.go`

```go
package parser

type ParseError struct {
    Position token.Position
    Message  string
    Token    token.Token
}

func (e *ParseError) Error() string

// Synchronize to next statement after error
func (p *Parser) synchronize()
```

### 6.4 Semantic Analysis

#### Task 4.4.1: Type Resolution

**File:** `internal/schema/semantic/resolver.go`

```go
package semantic

// Resolver resolves type references and validates schema
type Resolver struct {
    schema    *ast.Schema
    types     map[string]*TypeSymbol
    imports   map[string]*ast.Schema
    errors    []error
}

func NewResolver() *Resolver
func (r *Resolver) Resolve(schema *ast.Schema) error

// Resolution
func (r *Resolver) resolveTypeRef(ref *ast.TypeRef) (*TypeSymbol, error)
func (r *Resolver) resolveImports() error
func (r *Resolver) buildSymbolTable() error
```

#### Task 4.4.2: Validation

**File:** `internal/schema/semantic/validator.go`

```go
package semantic

// Validator performs semantic validation
type Validator struct {
    schema *ast.Schema
    errors []error
}

func (v *Validator) Validate() []error

// Validation checks
func (v *Validator) checkDuplicateNames()
func (v *Validator) checkFieldNumbers()
func (v *Validator) checkTypeIDs()
func (v *Validator) checkInterfaceImplementations()
func (v *Validator) checkCyclicDependencies()
```

### 6.5 Schema Loading

#### Task 4.5.1: File Loading

**File:** `internal/schema/loader/loader.go`

```go
package loader

// Loader loads and parses schema files
type Loader struct {
    searchPaths []string
    cache       map[string]*ast.Schema
}

func NewLoader(searchPaths ...string) *Loader
func (l *Loader) Load(filename string) (*ast.Schema, error)
func (l *Loader) LoadAll(filenames []string) ([]*ast.Schema, error)

// Resolve imports recursively
func (l *Loader) resolveImports(schema *ast.Schema) error
```

### 6.6 Standard Library Schema

#### Task 4.6.1: stdlib.cramberry

**File:** `schema/cramberry/stdlib.cramberry`

```cramberry
package cramberry.stdlib;

type Timestamp {
    @field(1) seconds: int64;
    @field(2) nanos: int32;
}

type Duration {
    @field(1) nanos: int64;
}

type BoolValue { @field(1) value: bool; }
type Int32Value { @field(1) value: int32; }
type Int64Value { @field(1) value: int64; }
type UInt32Value { @field(1) value: uint32; }
type UInt64Value { @field(1) value: uint64; }
type FloatValue { @field(1) value: float32; }
type DoubleValue { @field(1) value: float64; }
type StringValue { @field(1) value: string; }
type BytesValue { @field(1) value: bytes; }

type Empty {}

type Any {
    @field(1) typeUrl: string;
    @field(2) value: bytes;
}
```

### 6.7 Test Suite

#### Task 4.7.1: Lexer Tests
- All token types
- Keywords vs identifiers
- String escapes
- Comments
- Error positions

#### Task 4.7.2: Parser Tests
- Valid schemas (comprehensive examples)
- Invalid syntax (error messages)
- Annotations
- Generics
- All declaration types

#### Task 4.7.3: Semantic Tests
- Type resolution
- Import resolution
- Validation errors
- Cyclic dependency detection

### 6.8 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| Lexer | Token stream | All tokens, positions |
| AST | Node definitions | Complete grammar coverage |
| Parser | Schema parsing | All declarations |
| Resolver | Type resolution | Imports, references |
| Validator | Semantic checks | All validations |
| Loader | File loading | Import resolution |
| stdlib | Standard library | Core types defined |
| Tests | Parser tests | >95% coverage |

### 6.9 Exit Criteria

- [ ] Lexer correctly tokenizes all valid input
- [ ] Parser produces correct AST for all schema constructs
- [ ] Semantic analysis catches all validation errors
- [ ] Import resolution works with search paths
- [ ] Standard library schema parses correctly
- [ ] Error messages include position information

---

## 7. Phase 5: Code Generation

### Objective
Implement code generator that produces Go code from Cramberry schemas.

### Prerequisites
- Phase 2, 3, 4 complete

### 7.1 Generator Architecture

#### Task 5.1.1: Generator Framework

**File:** `internal/codegen/generator.go`

```go
package codegen

// Generator produces code from schemas
type Generator struct {
    schema    *ast.Schema
    resolver  *semantic.Resolver
    options   *Options
    output    *OutputSet
}

type Options struct {
    Language       string
    OutputDir      string
    PackagePrefix  string
    WithJSON       bool
    WithValidation bool
    WithBuilders   bool
    WithEquals     bool
    WithClone      bool
}

// OutputSet collects generated files
type OutputSet struct {
    Files []*OutputFile
}

type OutputFile struct {
    Path    string
    Content []byte
}

func NewGenerator(schema *ast.Schema, opts *Options) *Generator
func (g *Generator) Generate() (*OutputSet, error)
```

### 7.2 Go Code Generator

#### Task 5.2.1: Go Generator Core

**File:** `internal/codegen/golang/generator.go`

```go
package golang

type GoGenerator struct {
    schema   *ast.Schema
    options  *Options
    imports  map[string]string  // path -> alias
    buf      *bytes.Buffer
}

func NewGoGenerator(schema *ast.Schema, opts *Options) *GoGenerator
func (g *GoGenerator) Generate() ([]*codegen.OutputFile, error)

// Generation methods
func (g *GoGenerator) generateFile(pkg string, decls []ast.Declaration) *codegen.OutputFile
func (g *GoGenerator) generateHeader()
func (g *GoGenerator) generateImports()
func (g *GoGenerator) generateType(decl *ast.TypeDecl)
func (g *GoGenerator) generateEnum(decl *ast.EnumDecl)
func (g *GoGenerator) generateInterface(decl *ast.InterfaceDecl)
```

#### Task 5.2.2: Type Generation

**File:** `internal/codegen/golang/types.go`

```go
package golang

// Generate struct definition
func (g *GoGenerator) generateStructDef(decl *ast.TypeDecl)

// Generate field definition with tags
func (g *GoGenerator) generateField(field *ast.FieldDecl)

// Map schema types to Go types
func (g *GoGenerator) goType(ref *ast.TypeRef) string
```

**Output Example:**
```go
// Order represents a customer order
type Order struct {
    ID         []byte            `cramberry:"1" json:"id,omitempty"`
    CustomerID int64             `cramberry:"2" json:"customer_id,omitempty"`
    Status     OrderStatus       `cramberry:"3" json:"status,omitempty"`
    Items      []OrderItem       `cramberry:"4" json:"items,omitempty"`
    Payment    PaymentMethod     `cramberry:"5" json:"payment,omitempty"`
    Metadata   map[string]string `cramberry:"6" json:"metadata,omitempty"`
    CreatedAt  time.Time         `cramberry:"7" json:"created_at,omitempty"`
    UpdatedAt  *time.Time        `cramberry:"8" json:"updated_at,omitempty"`
    Tags       []string          `cramberry:"9" json:"tags,omitempty"`
}
```

#### Task 5.2.3: Encoder Generation

**File:** `internal/codegen/golang/encoder.go`

```go
package golang

// Generate MarshalCramberry method
func (g *GoGenerator) generateMarshalMethod(decl *ast.TypeDecl)

// Generate AppendCramberry method
func (g *GoGenerator) generateAppendMethod(decl *ast.TypeDecl)

// Generate CramberrySize method
func (g *GoGenerator) generateSizeMethod(decl *ast.TypeDecl)

// Generate field encoding
func (g *GoGenerator) generateFieldEncode(field *ast.FieldDecl)
```

**Output Example:**
```go
// MarshalCramberry implements cramberry.Marshaler
func (x *Order) MarshalCramberry() ([]byte, error) {
    size := x.CramberrySize()
    buf := make([]byte, 0, size)
    return x.AppendCramberry(buf)
}

// AppendCramberry appends the encoded form to buf
func (x *Order) AppendCramberry(buf []byte) ([]byte, error) {
    var err error

    // Field 1: ID (bytes)
    if len(x.ID) > 0 {
        buf = cramberry.AppendTag(buf, 1, cramberry.WireBytes)
        buf = cramberry.AppendBytes(buf, x.ID)
    }

    // Field 2: CustomerID (int64)
    if x.CustomerID != 0 {
        buf = cramberry.AppendTag(buf, 2, cramberry.WireSVarint)
        buf = cramberry.AppendSVarint(buf, x.CustomerID)
    }

    // ... remaining fields

    return buf, nil
}

// CramberrySize returns the encoded size in bytes
func (x *Order) CramberrySize() int {
    size := 0

    if len(x.ID) > 0 {
        size += cramberry.TagSize(1) + cramberry.BytesSize(x.ID)
    }

    if x.CustomerID != 0 {
        size += cramberry.TagSize(2) + cramberry.SVarintSize(x.CustomerID)
    }

    // ... remaining fields

    return size
}
```

#### Task 5.2.4: Decoder Generation

**File:** `internal/codegen/golang/decoder.go`

```go
package golang

// Generate UnmarshalCramberry method
func (g *GoGenerator) generateUnmarshalMethod(decl *ast.TypeDecl)

// Generate ReadCramberry method
func (g *GoGenerator) generateReadMethod(decl *ast.TypeDecl)

// Generate field decoding switch
func (g *GoGenerator) generateFieldDecode(field *ast.FieldDecl)
```

**Output Example:**
```go
// UnmarshalCramberry implements cramberry.Unmarshaler
func (x *Order) UnmarshalCramberry(data []byte) error {
    r := cramberry.NewReader(data)
    return x.ReadCramberry(r)
}

// ReadCramberry reads from a cramberry.Reader
func (x *Order) ReadCramberry(r *cramberry.Reader) error {
    for !r.EOF() {
        fieldNum, wireType, err := r.ReadTag()
        if err != nil {
            return err
        }

        switch fieldNum {
        case 1: // ID
            x.ID, err = r.ReadBytes()
        case 2: // CustomerID
            x.CustomerID, err = r.ReadSVarint64()
        case 3: // Status
            v, e := r.ReadUVarint64()
            if e == nil {
                x.Status = OrderStatus(v)
            }
            err = e
        // ... remaining fields
        default:
            err = r.Skip(wireType)
        }

        if err != nil {
            return err
        }
    }
    return nil
}
```

#### Task 5.2.5: Registration Generation

**File:** `internal/codegen/golang/register.go`

```go
package golang

// Generate init() function with type registrations
func (g *GoGenerator) generateRegistration()

// Generate interface registration
func (g *GoGenerator) generateInterfaceRegistration(decl *ast.InterfaceDecl)
```

**Output Example:**
```go
func init() {
    // Type registrations
    cramberry.MustRegisterType[Order]("myapp.models.Order")
    cramberry.MustRegisterType[OrderItem]("myapp.models.OrderItem")

    // Interface registrations
    cramberry.RegisterInterface[PaymentMethod]("myapp.PaymentMethod")
    cramberry.MustRegisterImplementation[PaymentMethod, CreditCardPayment](
        128, "myapp.payment.CreditCardPayment")
    cramberry.MustRegisterImplementation[PaymentMethod, CryptoPayment](
        129, "myapp.payment.CryptoPayment")
}
```

#### Task 5.2.6: Enum Generation

**File:** `internal/codegen/golang/enum.go`

```go
package golang

// Generate enum type and constants
func (g *GoGenerator) generateEnum(decl *ast.EnumDecl)

// Generate String() method
func (g *GoGenerator) generateEnumString(decl *ast.EnumDecl)
```

**Output Example:**
```go
type OrderStatus int32

const (
    OrderStatus_PENDING   OrderStatus = 0
    OrderStatus_CONFIRMED OrderStatus = 1
    OrderStatus_SHIPPED   OrderStatus = 2
    OrderStatus_DELIVERED OrderStatus = 3
    OrderStatus_CANCELLED OrderStatus = 4
)

func (x OrderStatus) String() string {
    switch x {
    case OrderStatus_PENDING:
        return "PENDING"
    // ...
    default:
        return fmt.Sprintf("OrderStatus(%d)", x)
    }
}
```

#### Task 5.2.7: Optional Features

**File:** `internal/codegen/golang/optional.go`

```go
package golang

// Generate builder pattern
func (g *GoGenerator) generateBuilder(decl *ast.TypeDecl)

// Generate Equal method
func (g *GoGenerator) generateEqual(decl *ast.TypeDecl)

// Generate Clone method
func (g *GoGenerator) generateClone(decl *ast.TypeDecl)

// Generate validation
func (g *GoGenerator) generateValidation(decl *ast.TypeDecl)
```

### 7.3 CLI Tool

#### Task 5.3.1: Generate Command

**File:** `cmd/cramberry/generate.go`

```go
package main

var generateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate code from schema files",
    RunE:  runGenerate,
}

func init() {
    generateCmd.Flags().StringP("lang", "l", "go", "Target language")
    generateCmd.Flags().StringP("out", "o", "./gen", "Output directory")
    generateCmd.Flags().String("package-prefix", "", "Package prefix for imports")
    generateCmd.Flags().Bool("with-json", false, "Generate JSON tags")
    generateCmd.Flags().Bool("with-validation", false, "Generate validation code")
    generateCmd.Flags().Bool("with-builders", false, "Generate builder pattern")
    generateCmd.Flags().Bool("with-equals", false, "Generate equality methods")
    generateCmd.Flags().Bool("with-clone", false, "Generate clone methods")
}

func runGenerate(cmd *cobra.Command, args []string) error
```

**Usage:**
```bash
cramberry generate --lang=go --out=./gen schema/*.cramberry
```

### 7.4 Test Suite

#### Task 5.4.1: Generator Tests
- All type declarations generate valid Go
- Generated code compiles
- Generated code passes tests
- All options work correctly

#### Task 5.4.2: Integration Tests
- Schema → Generate → Compile → Test round-trip
- Generated code matches reflection behavior

### 7.5 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| Generator Framework | Base generator | Language-agnostic core |
| Go Struct Gen | Type definitions | Valid Go structs |
| Go Encoder Gen | Marshal methods | Correct encoding |
| Go Decoder Gen | Unmarshal methods | Correct decoding |
| Go Registration | init() generation | Type registry setup |
| Go Enums | Enum generation | Constants, String() |
| Optional Features | Builders, equals, etc. | All options work |
| CLI Command | generate command | Full flag support |
| Tests | Generator tests | Compile and execute |

### 7.6 Exit Criteria

- [ ] Generated Go code compiles without errors
- [ ] Generated code produces identical output to reflection
- [ ] All schema constructs supported
- [ ] CLI tool functional with all options
- [ ] Integration tests passing

---

## 8. Phase 6: Schema Extraction Tool

### Objective
Build tool to extract Cramberry schemas from Go source code.

### Prerequisites
- Phase 4 complete

### 8.1 Go Source Analysis

#### Task 6.1.1: Package Loading

**File:** `internal/extract/loader.go`

```go
package extract

import "golang.org/x/tools/go/packages"

type PackageLoader struct {
    config *packages.Config
}

func NewPackageLoader() *PackageLoader
func (l *PackageLoader) Load(patterns []string) ([]*packages.Package, error)
```

#### Task 6.1.2: Type Collection

**File:** `internal/extract/collector.go`

```go
package extract

import "go/types"

type TypeCollector struct {
    packages   []*packages.Package
    types      map[string]*TypeInfo
    interfaces map[string]*InterfaceInfo
    config     *Config
}

type Config struct {
    IncludePrivate   bool
    IncludePatterns  []string
    ExcludePatterns  []string
    DetectInterfaces bool
}

type TypeInfo struct {
    Name      string
    Package   string
    GoType    types.Type
    Fields    []*FieldInfo
    TypeID    TypeID
    Implements []string
}

type FieldInfo struct {
    Name       string
    FieldNum   int
    GoType     types.Type
    Tag        *StructTag
}

type InterfaceInfo struct {
    Name           string
    Package        string
    Methods        []*MethodInfo
    Implementations []*TypeInfo
}

func NewTypeCollector(pkgs []*packages.Package, cfg *Config) *TypeCollector
func (c *TypeCollector) Collect() error
func (c *TypeCollector) Types() map[string]*TypeInfo
func (c *TypeCollector) Interfaces() map[string]*InterfaceInfo
```

#### Task 6.1.3: Struct Tag Parsing

**File:** `internal/extract/tags.go`

```go
package extract

// ParseCramberryTag parses the cramberry struct tag
func ParseCramberryTag(tag string, index int) (*StructTag, error)

// StructTag represents parsed tag
type StructTag struct {
    FieldNum   int
    Name       string
    OmitEmpty  bool
    Optional   bool
    Skip       bool
    Encoding   string
    Deprecated string
}
```

### 8.2 Interface Detection

#### Task 6.2.1: Implementation Detection

**File:** `internal/extract/interfaces.go`

```go
package extract

import "go/types"

// DetectImplementations finds all types implementing each interface
func (c *TypeCollector) DetectImplementations()

// Implements checks if a type implements an interface
func Implements(typ types.Type, iface *types.Interface) bool
```

### 8.3 Schema Generation

#### Task 6.3.1: Schema Builder

**File:** `internal/extract/builder.go`

```go
package extract

// SchemaBuilder constructs schema from collected types
type SchemaBuilder struct {
    collector *TypeCollector
    config    *Config
}

func NewSchemaBuilder(c *TypeCollector, cfg *Config) *SchemaBuilder
func (b *SchemaBuilder) Build() (*ast.Schema, error)

// Conversion methods
func (b *SchemaBuilder) typeToDecl(info *TypeInfo) *ast.TypeDecl
func (b *SchemaBuilder) interfaceToDecl(info *InterfaceInfo) *ast.InterfaceDecl
func (b *SchemaBuilder) goTypeToTypeRef(t types.Type) *ast.TypeRef
```

#### Task 6.3.2: Schema Writer

**File:** `internal/extract/writer.go`

```go
package extract

// SchemaWriter formats schema to .cramberry file
type SchemaWriter struct {
    indent int
    buf    *bytes.Buffer
}

func NewSchemaWriter() *SchemaWriter
func (w *SchemaWriter) Write(schema *ast.Schema) ([]byte, error)

// Writing methods
func (w *SchemaWriter) writePackage(pkg *ast.PackageDecl)
func (w *SchemaWriter) writeType(decl *ast.TypeDecl)
func (w *SchemaWriter) writeInterface(decl *ast.InterfaceDecl)
func (w *SchemaWriter) writeEnum(decl *ast.EnumDecl)
```

### 8.4 Type ID Assignment

#### Task 6.4.1: ID Strategies

**File:** `internal/extract/typeid.go`

```go
package extract

type TypeIDStrategy int

const (
    TypeIDExplicit TypeIDStrategy = iota  // From tags/comments
    TypeIDHash                             // Hash of qualified name
    TypeIDAuto                             // Sequential
    TypeIDConfig                           // From config file
)

type TypeIDAssigner struct {
    strategy TypeIDStrategy
    config   map[string]TypeID
    nextID   TypeID
}

func NewTypeIDAssigner(strategy TypeIDStrategy) *TypeIDAssigner
func (a *TypeIDAssigner) Assign(info *TypeInfo) TypeID

// Hash-based assignment
func HashTypeID(qualifiedName string) TypeID
```

### 8.5 CLI Tool

#### Task 6.5.1: Schema Command

**File:** `cmd/cramberry/schema.go`

```go
package main

var schemaCmd = &cobra.Command{
    Use:   "schema",
    Short: "Extract schema from Go source",
    RunE:  runSchema,
}

func init() {
    schemaCmd.Flags().StringArrayP("package", "p", nil, "Go packages to analyze")
    schemaCmd.Flags().StringP("output", "o", "", "Output directory or file")
    schemaCmd.Flags().Bool("recursive", false, "Include sub-packages")
    schemaCmd.Flags().Bool("include-private", false, "Include unexported types")
    schemaCmd.Flags().StringArray("include", nil, "Type name patterns to include")
    schemaCmd.Flags().StringArray("exclude", nil, "Type name patterns to exclude")
    schemaCmd.Flags().Bool("detect-interfaces", true, "Auto-detect interface implementations")
    schemaCmd.Flags().String("typeid-strategy", "hash", "Type ID strategy (explicit|hash|auto|config)")
    schemaCmd.Flags().String("config", "", "Configuration file (cramberry.yaml)")
}

func runSchema(cmd *cobra.Command, args []string) error
```

**Usage:**
```bash
cramberry schema --package ./pkg/models --output ./schema/models.cramberry
```

### 8.6 Configuration File

#### Task 6.6.1: YAML Configuration

**File:** `internal/extract/config.go`

```go
package extract

type ConfigFile struct {
    Version  string                    `yaml:"version"`
    Packages []PackageConfig           `yaml:"packages"`
    Settings SettingsConfig            `yaml:"settings"`
    TypeIDs  map[string]TypeID         `yaml:"type_ids"`
    TypeMappings map[string]string     `yaml:"type_mappings"`
}

type PackageConfig struct {
    Path      string   `yaml:"path"`
    Output    string   `yaml:"output"`
    Recursive bool     `yaml:"recursive"`
    Include   []string `yaml:"include"`
    Exclude   []string `yaml:"exclude"`
}

type SettingsConfig struct {
    IncludePrivate   bool   `yaml:"include_private"`
    DetectInterfaces bool   `yaml:"detect_interfaces"`
    TypeIDStrategy   string `yaml:"typeid_strategy"`
}

func LoadConfig(path string) (*ConfigFile, error)
```

**Example cramberry.yaml:**
```yaml
version: "1.0"

packages:
  - path: "./pkg/models"
    output: "./schema/models.cramberry"
    include: ["User", "Order", "*Message"]
    exclude: ["*Internal"]

settings:
  include_private: false
  detect_interfaces: true
  typeid_strategy: "hash"

type_ids:
  User: 128
  Order: 129

type_mappings:
  "time.Time": "cramberry.stdlib.Timestamp"
  "github.com/google/uuid.UUID": "bytes"
```

### 8.7 Test Suite

#### Task 6.7.1: Extraction Tests
- Extract from simple structs
- Extract from complex nested types
- Interface detection
- Tag parsing
- Type mappings

### 8.8 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| Package Loader | Go package analysis | Uses go/packages |
| Type Collector | Struct/interface collection | All types found |
| Tag Parser | cramberry tag parsing | All options parsed |
| Interface Detection | Implementation finding | Correct detection |
| Schema Builder | Type → AST conversion | Valid AST |
| Schema Writer | AST → .cramberry | Valid syntax |
| Type ID Assigner | ID strategies | All strategies work |
| CLI Command | schema command | Full flag support |
| Config File | YAML config | Loaded and applied |
| Tests | Extraction tests | Round-trip verified |

### 8.9 Exit Criteria

- [ ] Can extract schema from Go packages
- [ ] Struct tags correctly parsed
- [ ] Interface implementations detected
- [ ] Generated schema parses correctly
- [ ] Type IDs assigned consistently
- [ ] CLI tool works with config file

---

## 9. Phase 7: Streaming Support

### Objective
Implement streaming encode/decode for large messages and real-time data.

### Prerequisites
- Phase 2 complete

### 9.1 Stream Writer

#### Task 7.1.1: Stream Writer Interface

**File:** `pkg/cramberry/stream_writer.go`

```go
package cramberry

// StreamWriter provides streaming encoding
type StreamWriter interface {
    // Write a complete message
    WriteMessage(msg any) error

    // Start a message for chunked writing
    StartMessage(typeID TypeID) (MessageWriter, error)

    // Metadata
    WriteHeader(key string, value []byte) error
    WriteTrailer(key string, value []byte) error

    // Control
    Flush() error
    Close() error
}

// MessageWriter for chunked message writing
type MessageWriter interface {
    WriteField(fieldNum int, value any) error
    WriteFieldBytes(fieldNum int, data []byte) error
    StartNestedMessage(fieldNum int, typeID TypeID) (MessageWriter, error)
    StartArray(fieldNum int) (ArrayWriter, error)
    Complete() error
}

// ArrayWriter for streaming array elements
type ArrayWriter interface {
    WriteElement(value any) error
    Complete() error
}
```

#### Task 7.1.2: Stream Writer Implementation

**File:** `internal/stream/writer.go`

```go
package stream

type streamWriter struct {
    w           io.Writer
    registry    *Registry
    options     *StreamOptions
    frameBuffer []byte
    compressed  bool
}

type StreamOptions struct {
    Compression          CompressionType
    CompressionLevel     int
    CompressionThreshold int
    MaxFrameSize         int
}

type CompressionType int

const (
    CompressionNone CompressionType = iota
    CompressionLZ4
    CompressionZstd
    CompressionSnappy
)

func NewStreamWriter(w io.Writer, opts *StreamOptions) *streamWriter
```

### 9.2 Stream Reader

#### Task 7.2.1: Stream Reader Interface

**File:** `pkg/cramberry/stream_reader.go`

```go
package cramberry

// StreamReader provides streaming decoding
type StreamReader interface {
    // Read next complete message
    ReadMessage() (any, error)

    // Read next message into specific type
    ReadMessageInto(target any) error

    // Read messages as channel
    Messages() <-chan MessageOrError

    // Metadata
    Headers() map[string][]byte
    Trailers() map[string][]byte

    // Control
    HasMore() bool
    Close() error
}

type MessageOrError struct {
    Value any
    Error error
}
```

#### Task 7.2.2: Stream Reader Implementation

**File:** `internal/stream/reader.go`

```go
package stream

type streamReader struct {
    r           io.Reader
    registry    *Registry
    options     *StreamOptions
    headers     map[string][]byte
    trailers    map[string][]byte
    frameBuffer []byte
}

func NewStreamReader(r io.Reader, opts *StreamOptions) *streamReader
```

### 9.3 Frame Protocol

#### Task 7.3.1: Frame Encoding

**File:** `internal/stream/frame.go`

```go
package stream

type FrameType uint8

const (
    FrameData    FrameType = 0x0
    FrameHeader  FrameType = 0x1
    FrameTrailer FrameType = 0x2
    FrameReset   FrameType = 0x3
    FramePing    FrameType = 0x4
    FramePong    FrameType = 0x5
    FrameWindow  FrameType = 0x6
    FrameSchema  FrameType = 0x7
)

type FrameHeader struct {
    Type       FrameType
    EndMessage bool
    Compressed bool
    Length     uint32
}

func EncodeFrameHeader(h *FrameHeader) []byte
func DecodeFrameHeader(data []byte) (*FrameHeader, error)
```

### 9.4 Compression Integration

#### Task 7.4.1: Compression Codecs

**File:** `internal/stream/compression.go`

```go
package stream

// Compressor interface for compression algorithms
type Compressor interface {
    Compress(dst, src []byte) ([]byte, error)
    Decompress(dst, src []byte) ([]byte, error)
}

// LZ4 compression
type lz4Compressor struct{ level int }
func (c *lz4Compressor) Compress(dst, src []byte) ([]byte, error)
func (c *lz4Compressor) Decompress(dst, src []byte) ([]byte, error)

// Zstd compression
type zstdCompressor struct{ level int }
// ... similar

// Snappy compression
type snappyCompressor struct{}
// ... similar
```

### 9.5 Test Suite

#### Task 7.5.1: Streaming Tests
- Stream multiple messages
- Large message chunking
- Compression/decompression
- Header/trailer handling
- Error recovery

### 9.6 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| StreamWriter | Streaming encoder | All methods |
| StreamReader | Streaming decoder | All methods |
| MessageWriter | Chunked writing | Nested support |
| Frame Protocol | Frame encode/decode | All frame types |
| Compression | LZ4, Zstd, Snappy | Transparent |
| Tests | Streaming tests | All scenarios |

### 9.7 Exit Criteria

- [ ] Can stream encode/decode large messages
- [ ] Chunked writing works for nested structures
- [ ] Compression transparent to user
- [ ] Headers/trailers accessible
- [ ] Performance meets targets

---

## 10. Phase 8: Cross-Language Runtimes

### Objective
Implement TypeScript and Rust runtime libraries and code generators.

### Prerequisites
- Phase 4, 5 complete

### 10.1 TypeScript Runtime

#### Task 8.1.1: Core Runtime

**File:** `typescript/runtime/src/index.ts`

```typescript
export { Writer } from './writer';
export { Reader } from './reader';
export { Registry } from './registry';
export { WireType } from './types';
export * from './errors';
```

#### Task 8.1.2: Writer Implementation

**File:** `typescript/runtime/src/writer.ts`

```typescript
export class Writer {
    private buffer: Uint8Array;
    private view: DataView;
    private pos: number;

    constructor(initialSize?: number);
    uvarint(value: bigint | number): void;
    svarint(value: bigint | number): void;
    fixed32(value: number): void;
    fixed64(value: bigint): void;
    bytes(data: Uint8Array): void;
    string(s: string): void;
    tag(fieldNum: number, wireType: WireType): void;
    finish(): Uint8Array;
}
```

#### Task 8.1.3: Reader Implementation

**File:** `typescript/runtime/src/reader.ts`

```typescript
export class Reader {
    constructor(data: Uint8Array);
    eof(): boolean;
    uvarint(): bigint;
    svarint(): bigint;
    fixed32(): number;
    fixed64(): bigint;
    bytes(): Uint8Array;
    string(): string;
    tag(): [number, WireType];
    skip(wireType: WireType): void;
}
```

#### Task 8.1.4: Registry Implementation

**File:** `typescript/runtime/src/registry.ts`

```typescript
export interface TypeCodec<T> {
    encode(writer: Writer, value: T): void;
    decode(reader: Reader): T;
    size(value: T): number;
}

export class Registry {
    registerType<T>(id: number, name: string, codec: TypeCodec<T>): void;
    encode<T>(writer: Writer, value: T, typeName: string): void;
    decode<T>(reader: Reader): T;
}
```

### 10.2 TypeScript Code Generator

#### Task 8.2.1: TypeScript Generator

**File:** `internal/codegen/typescript/generator.go`

```go
package typescript

type TSGenerator struct {
    schema  *ast.Schema
    options *Options
}

type Options struct {
    OutputDir   string
    ModuleType  string  // "esm" or "commonjs"
    RuntimePath string
}

func NewTSGenerator(schema *ast.Schema, opts *Options) *TSGenerator
func (g *TSGenerator) Generate() ([]*codegen.OutputFile, error)
```

**Generated Output:**
```typescript
// order.gen.ts
import { Reader, Writer, Registry } from '@cramberry/runtime';

export enum OrderStatus {
    PENDING = 0,
    CONFIRMED = 1,
    SHIPPED = 2,
    DELIVERED = 3,
    CANCELLED = 4,
}

export interface Order {
    id: Uint8Array;
    customerId: bigint;
    status: OrderStatus;
    items: OrderItem[];
    // ...
}

export const OrderCodec = {
    encode(writer: Writer, value: Order): void {
        // ...
    },
    decode(reader: Reader): Order {
        // ...
    },
    size(value: Order): number {
        // ...
    },
};
```

### 10.3 Rust Runtime

#### Task 8.3.1: Core Runtime

**File:** `rust/cramberry/src/lib.rs`

```rust
pub mod wire;
pub mod reader;
pub mod writer;
pub mod registry;
pub mod error;

pub use reader::Reader;
pub use writer::Writer;
pub use registry::Registry;
pub use error::Error;
```

#### Task 8.3.2: Traits

**File:** `rust/cramberry/src/traits.rs`

```rust
pub trait CramberryEncode {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> Result<(), Error>;
    fn encoded_size(&self) -> usize;
}

pub trait CramberryDecode: Sized {
    fn decode<R: std::io::Read>(reader: &mut R) -> Result<Self, Error>;
}
```

### 10.4 Rust Code Generator

#### Task 8.4.1: Rust Generator

**File:** `internal/codegen/rust/generator.go`

```go
package rust

type RustGenerator struct {
    schema  *ast.Schema
    options *Options
}

func NewRustGenerator(schema *ast.Schema, opts *Options) *RustGenerator
func (g *RustGenerator) Generate() ([]*codegen.OutputFile, error)
```

### 10.5 Cross-Language Testing

#### Task 8.5.1: Compatibility Tests
- Same message encoded in Go decodes in TypeScript
- Same message encoded in TypeScript decodes in Go
- Same message encoded in Rust decodes in Go
- All three produce identical bytes

### 10.6 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| TS Runtime | @cramberry/runtime | Full encode/decode |
| TS Generator | TypeScript codegen | Valid TypeScript |
| Rust Runtime | cramberry crate | Full encode/decode |
| Rust Generator | Rust codegen | Valid Rust |
| Compat Tests | Cross-language tests | All languages match |

### 10.7 Exit Criteria

- [ ] TypeScript runtime fully functional
- [ ] TypeScript generated code compiles and runs
- [ ] Rust runtime fully functional
- [ ] Rust generated code compiles and runs
- [ ] Cross-language compatibility verified

---

## 11. Phase 9: Performance Optimization

### Objective
Optimize encoding/decoding performance to meet benchmark targets.

### Prerequisites
- Phase 2, 3, 5 complete

### 11.1 Profiling Infrastructure

#### Task 9.1.1: Benchmark Suite

**File:** `benchmark/benchmark_test.go`

```go
package benchmark

// Message sizes
func BenchmarkEncode_Small(b *testing.B)   // ~100 bytes
func BenchmarkEncode_Medium(b *testing.B)  // ~1KB
func BenchmarkEncode_Large(b *testing.B)   // ~100KB
func BenchmarkEncode_XLarge(b *testing.B)  // ~10MB

func BenchmarkDecode_Small(b *testing.B)
func BenchmarkDecode_Medium(b *testing.B)
func BenchmarkDecode_Large(b *testing.B)
func BenchmarkDecode_XLarge(b *testing.B)

// Type complexity
func BenchmarkEncode_Flat(b *testing.B)
func BenchmarkEncode_Nested(b *testing.B)
func BenchmarkEncode_Polymorphic(b *testing.B)

// Comparison
func BenchmarkProtobuf_Encode(b *testing.B)
func BenchmarkProtobuf_Decode(b *testing.B)
func BenchmarkJSON_Encode(b *testing.B)
func BenchmarkJSON_Decode(b *testing.B)
```

#### Task 9.1.2: Profiling Hooks

**File:** `pkg/cramberry/profile.go`

```go
package cramberry

type ProfileHooks struct {
    OnEncode func(typeName string, size int, duration time.Duration)
    OnDecode func(typeName string, size int, duration time.Duration)
    OnAlloc  func(size int)
}

var profileHooks *ProfileHooks

func SetProfileHooks(hooks *ProfileHooks)
func ClearProfileHooks()
```

### 11.2 Encoding Optimizations

#### Task 9.2.1: Buffer Pooling

**File:** `internal/pool/buffer.go`

```go
package pool

var bufferPool = sync.Pool{
    New: func() any {
        return make([]byte, 0, 4096)
    },
}

func GetBuffer() []byte
func PutBuffer(buf []byte)

// Sized pools for different message sizes
var smallPool  = sync.Pool{...}  // 256 bytes
var mediumPool = sync.Pool{...}  // 4KB
var largePool  = sync.Pool{...}  // 64KB
```

#### Task 9.2.2: Pre-computed Encoders

**File:** `internal/encoding/precomputed.go`

```go
package encoding

// PrecomputedEncoder caches encoding functions for a type
type PrecomputedEncoder struct {
    fields []fieldEncoder
}

type fieldEncoder struct {
    offset   uintptr
    encode   func(w *Writer, ptr unsafe.Pointer)
    size     func(ptr unsafe.Pointer) int
    wireType WireType
    fieldNum int
}

func BuildPrecomputedEncoder(t reflect.Type) *PrecomputedEncoder
```

#### Task 9.2.3: Inline Varint Encoding

**File:** `internal/wire/varint_inline.go`

```go
package wire

// Inline encoding for hot paths
//go:inline
func AppendUvarint64(buf []byte, v uint64) []byte {
    // Unrolled loop for small values
    if v < 0x80 {
        return append(buf, byte(v))
    }
    if v < 0x4000 {
        return append(buf, byte(v)|0x80, byte(v>>7))
    }
    // ... continue for larger values
}
```

### 11.3 Decoding Optimizations

#### Task 9.3.1: Zero-Allocation Decoding

**File:** `internal/decoding/zero_alloc.go`

```go
package decoding

// DecodeStringNoCopy returns a string without copying
// SAFETY: string only valid while reader data is alive
func (r *Reader) DecodeStringNoCopy() (string, error)

// DecodeBytesNoCopy returns bytes without copying
func (r *Reader) DecodeBytesNoCopy() ([]byte, error)

// ZeroAllocReader provides zero-copy decoding
type ZeroAllocReader struct {
    *Reader
    // Tracks borrowed strings/bytes
    borrowed [][]byte
}
```

#### Task 9.3.2: Field Lookup Table

**File:** `internal/decoding/lookup.go`

```go
package decoding

// FieldLookup provides O(1) field lookup
type FieldLookup struct {
    // For small field numbers (1-16), use array
    small [16]fieldDecoder
    // For larger, use map
    large map[int]fieldDecoder
}

type fieldDecoder struct {
    decode   func(r *Reader, v reflect.Value) error
    wireType WireType
}

func BuildFieldLookup(t reflect.Type) *FieldLookup
```

### 11.4 Generated Code Optimizations

#### Task 9.4.1: Specialized Encoders

**File:** `internal/codegen/golang/optimized.go`

```go
package golang

// Generate specialized encoder without reflection
func (g *GoGenerator) generateOptimizedEncoder(decl *ast.TypeDecl)

// Generate inlined field encoding
func (g *GoGenerator) generateInlinedFieldEncode(field *ast.FieldDecl)
```

### 11.5 Memory Optimizations

#### Task 9.5.1: Arena Allocation

**File:** `internal/pool/arena.go`

```go
package pool

// Arena provides bump-pointer allocation
type Arena struct {
    chunks [][]byte
    current []byte
    pos    int
}

func NewArena(chunkSize int) *Arena
func (a *Arena) Alloc(size int) []byte
func (a *Arena) Reset()
```

### 11.6 Performance Targets

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Small message encode | >5M msgs/sec | BenchmarkEncode_Small |
| Small message decode | >4M msgs/sec | BenchmarkDecode_Small |
| Large message encode | >2 GB/sec | BenchmarkEncode_Large |
| Large message decode | >1.5 GB/sec | BenchmarkDecode_Large |
| Memory per encode | <500 bytes | Benchmark allocs |
| Generated vs reflect | 3-10x faster | Comparison benchmark |

### 11.7 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| Benchmark Suite | Comprehensive benchmarks | All scenarios covered |
| Buffer Pooling | Memory reuse | Reduced allocations |
| Precomputed Encoders | Cached field info | Faster encoding |
| Zero-Alloc Decode | No-copy paths | Reduced allocations |
| Field Lookup | O(1) field access | Faster decoding |
| Generated Optimizations | Optimized codegen | Meets targets |
| Arena Allocation | Bump allocator | Optional feature |

### 11.8 Exit Criteria

- [ ] Benchmark targets met
- [ ] Generated code 3-10x faster than reflection
- [ ] Memory allocations minimized
- [ ] Performance regression tests in CI

---

## 12. Phase 10: Security Hardening

### Objective
Implement security limits, input validation, and protection against malicious input.

### Prerequisites
- Phase 2, 3, 7 complete

### 12.1 Input Limits

#### Task 10.1.1: Configurable Limits

**File:** `pkg/cramberry/limits.go`

```go
package cramberry

type Limits struct {
    MaxMessageSize  int64  // Maximum total message size
    MaxDepth        int    // Maximum nesting depth
    MaxStringLength int    // Maximum string length
    MaxBytesLength  int    // Maximum []byte length
    MaxArrayLength  int    // Maximum array/slice length
    MaxMapSize      int    // Maximum map entries
}

var DefaultLimits = Limits{
    MaxMessageSize:  64 * 1024 * 1024,  // 64 MB
    MaxDepth:        100,
    MaxStringLength: 10 * 1024 * 1024,  // 10 MB
    MaxBytesLength:  100 * 1024 * 1024, // 100 MB
    MaxArrayLength:  1_000_000,
    MaxMapSize:      1_000_000,
}

var SecureLimits = Limits{
    MaxMessageSize:  1 * 1024 * 1024,   // 1 MB
    MaxDepth:        32,
    MaxStringLength: 1 * 1024 * 1024,   // 1 MB
    MaxBytesLength:  10 * 1024 * 1024,  // 10 MB
    MaxArrayLength:  10_000,
    MaxMapSize:      10_000,
}
```

#### Task 10.1.2: Limit Enforcement

**File:** `internal/decoding/limits.go`

```go
package decoding

func (r *Reader) checkStringLimit(length int) error
func (r *Reader) checkBytesLimit(length int) error
func (r *Reader) checkArrayLimit(length int) error
func (r *Reader) checkMapLimit(size int) error
func (r *Reader) checkDepth() error
func (r *Reader) pushDepth() error
func (r *Reader) popDepth()
```

### 12.2 Input Validation

#### Task 10.2.1: UTF-8 Validation

**File:** `internal/decoding/validate.go`

```go
package decoding

// ValidateUTF8 checks if bytes are valid UTF-8
func ValidateUTF8(data []byte) bool

// ValidateUTF8String checks string during decode
func (r *Reader) ReadStringValidated() (string, error)
```

#### Task 10.2.2: Varint Validation

**File:** `internal/wire/validate.go`

```go
package wire

// Maximum bytes for a valid varint64
const MaxVarintLen64 = 10

// ValidateVarint checks for malformed varints
func ValidateVarint(data []byte) error

// Errors
var ErrVarintTooLong = errors.New("varint exceeds maximum length")
var ErrVarintOverflow = errors.New("varint overflows uint64")
```

### 12.3 Strict Mode

#### Task 10.3.1: Strict Decoding

**File:** `pkg/cramberry/strict.go`

```go
package cramberry

// StrictUnmarshal rejects unknown fields
func StrictUnmarshal(data []byte, v any) error

// UnmarshalOptions configures decoding behavior
type UnmarshalOptions struct {
    Limits         *Limits
    StrictMode     bool  // Reject unknown fields
    ValidateUTF8   bool  // Validate UTF-8 strings
    AllowNaN       bool  // Allow NaN float values
    AllowInf       bool  // Allow Inf float values
}

func UnmarshalWithOptions(data []byte, v any, opts *UnmarshalOptions) error
```

### 12.4 Fuzzing

#### Task 10.4.1: Fuzz Tests

**File:** `fuzz/fuzz_test.go`

```go
package fuzz

import "testing"

// FuzzDecode tests decoding with random input
func FuzzDecode(f *testing.F) {
    // Add seed corpus
    f.Add([]byte{...})

    f.Fuzz(func(t *testing.T, data []byte) {
        var msg TestMessage
        // Should not panic, may return error
        _ = cramberry.Unmarshal(data, &msg)
    })
}

// FuzzRoundTrip tests encode/decode roundtrip
func FuzzRoundTrip(f *testing.F) {
    f.Fuzz(func(t *testing.T, data []byte) {
        // Generate message from random data
        msg := generateFromBytes(data)

        // Encode
        encoded, err := cramberry.Marshal(msg)
        if err != nil {
            return
        }

        // Decode
        var decoded TestMessage
        err = cramberry.Unmarshal(encoded, &decoded)
        if err != nil {
            t.Fatalf("decode failed: %v", err)
        }

        // Compare
        if !reflect.DeepEqual(msg, decoded) {
            t.Fatalf("roundtrip mismatch")
        }
    })
}
```

#### Task 10.4.2: OSS-Fuzz Integration

**File:** `fuzz/oss_fuzz.go`

```go
//go:build gofuzz

package fuzz

func Fuzz(data []byte) int {
    var msg TestMessage
    if err := cramberry.Unmarshal(data, &msg); err != nil {
        return 0
    }
    return 1
}
```

### 12.5 Security Testing

#### Task 10.5.1: Malicious Input Tests

**File:** `security/malicious_test.go`

```go
package security

func TestExcessiveDepth(t *testing.T)
func TestExcessiveSize(t *testing.T)
func TestMalformedVarint(t *testing.T)
func TestInvalidUTF8(t *testing.T)
func TestNegativeLength(t *testing.T)
func TestIntegerOverflow(t *testing.T)
func TestResourceExhaustion(t *testing.T)
```

### 12.6 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| Limits Config | Configurable limits | All limits enforced |
| UTF-8 Validation | String validation | Invalid rejected |
| Varint Validation | Overflow protection | Malformed rejected |
| Strict Mode | Unknown field rejection | Option works |
| Fuzz Tests | Fuzzing infrastructure | No panics |
| Security Tests | Malicious input tests | All pass |

### 12.7 Exit Criteria

- [ ] All limits properly enforced
- [ ] No panics on malformed input
- [ ] Fuzz testing finds no issues
- [ ] Security test suite passes
- [ ] Resource exhaustion prevented

---

## 13. Phase 11: Documentation & Examples

### Objective
Create comprehensive documentation, examples, and tutorials.

### Prerequisites
- Core functionality complete

### 13.1 API Documentation

#### Task 11.1.1: GoDoc Comments

All public APIs documented with:
- Function/method description
- Parameter descriptions
- Return value descriptions
- Usage examples
- Error conditions

#### Task 11.1.2: Package Documentation

**File:** `pkg/cramberry/doc.go`

```go
// Package cramberry provides high-performance binary serialization.
//
// Basic usage:
//
//     type Person struct {
//         Name string `cramberry:"1"`
//         Age  int    `cramberry:"2"`
//     }
//
//     // Encode
//     data, err := cramberry.Marshal(&Person{Name: "Alice", Age: 30})
//
//     // Decode
//     var p Person
//     err = cramberry.Unmarshal(data, &p)
//
// For polymorphic types, register implementations:
//
//     cramberry.RegisterInterface[Message]("app.Message")
//     cramberry.RegisterImplementation[Message, TextMessage](1, "app.TextMessage")
//
package cramberry
```

### 13.2 User Guide

#### Task 11.2.1: Getting Started Guide

**File:** `docs/getting-started.md`

Contents:
- Installation
- Basic encoding/decoding
- Struct tags
- Common patterns
- Error handling

#### Task 11.2.2: Schema Language Guide

**File:** `docs/schema-language.md`

Contents:
- File format
- Type definitions
- Interfaces and implementations
- Annotations
- Best practices

#### Task 11.2.3: Code Generation Guide

**File:** `docs/code-generation.md`

Contents:
- CLI usage
- Generated code structure
- Options and flags
- Multi-language generation
- Integration with build systems

#### Task 11.2.4: Migration Guide

**File:** `docs/migration.md`

Contents:
- Migrating from Amino
- Migrating from Protobuf
- Migrating from JSON
- Compatibility considerations

### 13.3 Examples

#### Task 11.3.1: Basic Examples

**File:** `examples/basic/`

- `hello.go` - Simple encode/decode
- `structs.go` - Struct serialization
- `nested.go` - Nested structures
- `maps.go` - Map handling
- `slices.go` - Slice handling

#### Task 11.3.2: Advanced Examples

**File:** `examples/advanced/`

- `polymorphic.go` - Interface serialization
- `custom_codec.go` - Custom type encoding
- `streaming.go` - Streaming encode/decode
- `performance.go` - Optimization techniques

#### Task 11.3.3: Integration Examples

**File:** `examples/integration/`

- `grpc/` - gRPC integration
- `http/` - HTTP API serialization
- `database/` - Database storage
- `files/` - File format handling

### 13.4 Tutorials

#### Task 11.4.1: Video Tutorial Scripts

- Introduction to Cramberry
- Building a chat protocol
- Performance optimization
- Cross-language development

### 13.5 Reference Documentation

#### Task 11.5.1: Wire Format Specification

**File:** `docs/wire-format.md`

Detailed specification of:
- Byte encoding
- Wire types
- Message structure
- Determinism rules

#### Task 11.5.2: Type Mapping Reference

**File:** `docs/type-mapping.md`

Complete mapping:
- Go → Wire format
- Go → TypeScript
- Go → Rust
- Schema → All languages

### 13.6 Deliverables Checklist

| Item | Description | Acceptance Criteria |
|------|-------------|---------------------|
| GoDoc | API documentation | All public APIs |
| Getting Started | Beginner guide | Clear, tested |
| Schema Guide | SDL documentation | Complete grammar |
| CodeGen Guide | Generator docs | All options |
| Migration Guide | Upgrade paths | Amino, Protobuf |
| Basic Examples | Simple examples | Compilable, tested |
| Advanced Examples | Complex examples | Compilable, tested |
| Integration Examples | Real-world use | Working demos |
| Wire Format Spec | Protocol spec | Precise, complete |
| Type Mapping | Cross-language | All types covered |

### 13.7 Exit Criteria

- [ ] All public APIs documented
- [ ] All examples compile and run
- [ ] Documentation reviewed for accuracy
- [ ] Tutorial feedback incorporated

---

## 14. Phase 12: Production Readiness

### Objective
Prepare for production release with CI/CD, versioning, and support infrastructure.

### Prerequisites
- All previous phases complete

### 14.1 Version Management

#### Task 12.1.1: Semantic Versioning

```
MAJOR.MINOR.PATCH

1.0.0 - Initial stable release
1.1.0 - New features, backward compatible
1.1.1 - Bug fixes
2.0.0 - Breaking changes
```

#### Task 12.1.2: Version Embedding

**File:** `pkg/cramberry/version.go`

```go
package cramberry

// Set by ldflags at build time
var (
    Version   = "dev"
    GitCommit = "unknown"
    BuildDate = "unknown"
)

func VersionInfo() string
```

### 14.2 Release Process

#### Task 12.2.1: Release Workflow

**File:** `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test
      - run: make build-all

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            dist/*
          generate_release_notes: true
```

#### Task 12.2.2: Changelog

**File:** `CHANGELOG.md`

Format:
```markdown
# Changelog

## [Unreleased]

## [1.0.0] - 2026-XX-XX

### Added
- Initial stable release
- Full Go primitive support
- Polymorphic interface serialization
- Schema definition language
- Code generation for Go, TypeScript, Rust
- Streaming support
- Custom codec registration

### Security
- Input validation and limits
- Fuzz testing coverage
```

### 14.3 Package Publishing

#### Task 12.3.1: Go Module

```bash
# Tag release
git tag v1.0.0
git push origin v1.0.0

# Module will be available at
# github.com/blockberries/cramberry
```

#### Task 12.3.2: npm Package

**File:** `typescript/runtime/package.json`

```json
{
  "name": "@cramberry/runtime",
  "version": "1.0.0",
  "main": "dist/index.js",
  "types": "dist/index.d.ts"
}
```

#### Task 12.3.3: Cargo Crate

**File:** `rust/cramberry/Cargo.toml`

```toml
[package]
name = "cramberry"
version = "1.0.0"
edition = "2021"
```

### 14.4 CLI Distribution

#### Task 12.4.1: Binary Builds

Build matrix:
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64

#### Task 12.4.2: Installation Methods

```bash
# Go install
go install github.com/blockberries/cramberry/cmd/cramberry@latest

# Homebrew
brew install cramberry/tap/cramberry

# Direct download
curl -sSL https://cramberry.dev/install.sh | bash
```

### 14.5 Monitoring and Telemetry

#### Task 12.5.1: Optional Metrics

**File:** `pkg/cramberry/metrics.go`

```go
package cramberry

// Metrics interface for optional instrumentation
type Metrics interface {
    EncodeCount(typeName string)
    DecodeCount(typeName string)
    EncodeLatency(typeName string, duration time.Duration)
    DecodeLatency(typeName string, duration time.Duration)
    EncodeSize(typeName string, size int)
}

func SetMetrics(m Metrics)
```

### 14.6 Support Infrastructure

#### Task 12.6.1: Issue Templates

**File:** `.github/ISSUE_TEMPLATE/`

- bug_report.md
- feature_request.md
- question.md

#### Task 12.6.2: Contributing Guide

**File:** `CONTRIBUTING.md`

Contents:
- Development setup
- Code style
- Testing requirements
- PR process
- Release process

### 14.7 Stability Guarantees

#### Task 12.7.1: Compatibility Policy

**File:** `docs/compatibility.md`

Define:
- Wire format stability (never break)
- API stability (semver)
- Schema language stability
- Deprecation policy

### 14.8 Final Checklist

| Category | Item | Status |
|----------|------|--------|
| Code | All tests pass | [ ] |
| Code | >90% coverage | [ ] |
| Code | No lint warnings | [ ] |
| Code | Fuzz testing complete | [ ] |
| Docs | API documented | [ ] |
| Docs | User guide complete | [ ] |
| Docs | Examples working | [ ] |
| Release | Version tagged | [ ] |
| Release | Changelog updated | [ ] |
| Release | Binaries built | [ ] |
| Publish | Go module | [ ] |
| Publish | npm package | [ ] |
| Publish | Cargo crate | [ ] |
| Support | Issue templates | [ ] |
| Support | Contributing guide | [ ] |

### 14.9 Launch Tasks

1. [ ] Final security review
2. [ ] Performance validation
3. [ ] Documentation review
4. [ ] Create GitHub release
5. [ ] Publish packages
6. [ ] Announce release
7. [ ] Monitor for issues

---

## 15. Risk Assessment & Mitigation

### 15.1 Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Performance targets not met | High | Medium | Early benchmarking, profiling, optimization phase |
| Cross-language incompatibility | High | Low | Comprehensive compatibility tests |
| Schema evolution issues | High | Medium | Design for extensibility, version fields |
| Security vulnerabilities | High | Medium | Fuzzing, security review, limits |
| Memory leaks | Medium | Low | Profiling, leak detection tests |

### 15.2 Project Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Scope creep | Medium | High | Strict phase gates, prioritization |
| Integration complexity | Medium | Medium | Early integration testing |
| Documentation debt | Low | Medium | Document alongside code |
| Dependency issues | Low | Low | Minimal dependencies |

### 15.3 Contingency Plans

1. **Performance shortfall**: Fall back to pure generated code (no reflection)
2. **Cross-language issues**: Prioritize Go, delay other languages
3. **Scope overrun**: Cut optional features (builders, validation gen)

---

## 16. Resource Requirements

### 16.1 Team Skills

| Role | Skills Required |
|------|-----------------|
| Core Developer | Go, binary protocols, performance |
| Compiler Developer | Parsing, code generation, AST |
| TypeScript Developer | TypeScript, npm ecosystem |
| Rust Developer | Rust, systems programming |
| DevOps | CI/CD, release automation |
| Technical Writer | Documentation, tutorials |

### 16.2 Infrastructure

| Resource | Purpose |
|----------|---------|
| GitHub Actions | CI/CD |
| Codecov | Coverage tracking |
| Benchmark Action | Performance tracking |
| Release automation | Binary distribution |

### 16.3 Dependencies

**Go Dependencies (minimal):**
```go
require (
    golang.org/x/tools v0.x.x  // For schema extraction
)
```

**Optional:**
```go
require (
    github.com/klauspost/compress v1.x.x  // For compression
)
```

---

## 17. Success Metrics

### 17.1 Quality Metrics

| Metric | Target |
|--------|--------|
| Test coverage | >90% |
| Fuzz coverage | No panics in 24h run |
| Lint issues | Zero |
| Documentation | 100% public API |

### 17.2 Performance Metrics

| Metric | Target |
|--------|--------|
| Small message encode | >5M msgs/sec |
| Small message decode | >4M msgs/sec |
| Large message encode | >2 GB/sec |
| Generated vs reflect | >5x faster |

### 17.3 Adoption Metrics (Post-Launch)

| Metric | Target (6 months) |
|--------|-------------------|
| GitHub stars | 500+ |
| Go module downloads | 1000+/month |
| npm downloads | 500+/month |
| Active issues | <20 open |

---

## 18. Appendix: Task Breakdown

### Complete Task List

See individual phase sections for detailed task breakdowns.

### Task Dependencies

```
Task 1.1.1 → Task 1.1.2 → Task 1.1.3 → Task 1.1.4
     ↓
Task 1.2.1 ←→ Task 1.2.3 ←→ Task 1.2.5
     ↓           ↓           ↓
Task 2.1.1 ← Task 2.1.2 → Task 2.2.1
     ↓
Task 2.3.1 → Task 2.3.2 → Task 2.3.3 → Task 2.3.4
     ↓
Task 3.1.1 → Task 3.2.1 → Task 3.3.1 → Task 3.3.2
     ↓
...
```

### Parallel Work Streams

**Stream A: Core Runtime**
Phases 1 → 2 → 3 → 9

**Stream B: Schema & CodeGen**
Phases 4 → 5 → 6

**Stream C: Extensions**
Phases 7, 8 (after dependencies)

**Stream D: Quality**
Phases 10, 11, 12 (continuous)

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2026-01-21 | Engineering | Initial plan |

---

*End of Implementation Plan*
