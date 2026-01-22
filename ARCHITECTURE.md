# Cramberry Serialization Schema

## Architecture Design Document

**Version:** 1.0.0-draft
**Status:** Design Phase
**Authors:** Engineering Team
**Last Updated:** 2026-01-21

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Goals and Non-Goals](#2-goals-and-non-goals)
3. [Design Principles](#3-design-principles)
4. [Wire Format Specification](#4-wire-format-specification)
5. [Type System](#5-type-system)
6. [Interface and Polymorphic Support](#6-interface-and-polymorphic-support)
7. [Custom Type Registration](#7-custom-type-registration)
8. [Schema Definition Language (SDL)](#8-schema-definition-language-sdl)
9. [Code Generation](#9-code-generation)
10. [Streaming Protocol](#10-streaming-protocol)
11. [Go Reference Implementation](#11-go-reference-implementation)
12. [Cross-Language Support](#12-cross-language-support)
13. [Schema Generation Tool](#13-schema-generation-tool)
14. [Performance Considerations](#14-performance-considerations)
15. [Security Considerations](#15-security-considerations)
16. [Appendices](#16-appendices)

---

## 1. Executive Summary

Cramberry is a high-performance, deterministic binary serialization schema designed for systems requiring compact wire formats, polymorphic type handling, and cross-language interoperability. The design prioritizes:

- **Minimal wire size** through variable-length encoding and field omission
- **Deterministic output** for cryptographic hashing and consensus systems
- **Full Go type system support** including interfaces and generics
- **Amino-style polymorphism** with improved type safety and performance
- **Streaming capabilities** for large data and real-time processing
- **Schema-driven code generation** for type-safe multi-language support

The reference implementation will be in Go, with code generation support for TypeScript, Rust, and Python planned for initial release.

---

## 2. Goals and Non-Goals

### 2.1 Goals

| Priority | Goal | Rationale |
|----------|------|-----------|
| P0 | Deterministic encoding | Required for consensus, hashing, signatures |
| P0 | Support all Go primitives | Complete language coverage |
| P0 | Polymorphic interface serialization | Dynamic type handling like Amino |
| P0 | Compact binary format | Bandwidth and storage efficiency |
| P1 | Type-safe decoding | Avoid runtime type assertions |
| P1 | Custom type registration | Domain-specific optimizations |
| P1 | Schema-driven code generation | Cross-language support |
| P1 | Streaming encode/decode | Handle large/infinite data |
| P2 | Compression-friendly encoding | Enhance compressibility |
| P2 | Zero-copy decoding paths | Performance optimization |
| P2 | Schema evolution | Backward/forward compatibility |

### 2.2 Non-Goals

- **Human readability**: This is a binary format; use JSON for debugging
- **Self-describing format**: Schema required for decoding (unlike MessagePack)
- **Query capabilities**: This is serialization, not a database format
- **Encryption**: Handle at transport layer, not serialization layer
- **Arbitrary precision numbers**: Use fixed-width or custom types

---

## 3. Design Principles

### 3.1 Determinism First

Every encoding decision must produce identical output for identical input:

1. **Canonical field ordering**: Fields encoded in schema-defined order
2. **No optional padding**: No alignment bytes
3. **Normalized representations**: Single valid encoding per value
4. **Map key ordering**: Lexicographic by encoded key bytes
5. **No floating-point special cases**: NaN/Inf have defined encodings

### 3.2 Size Optimization Strategies

```
┌─────────────────────────────────────────────────────────────────┐
│                    Size Reduction Techniques                     │
├─────────────────────────────────────────────────────────────────┤
│  1. Varint encoding for integers (1-10 bytes based on value)    │
│  2. Field tags instead of names (1-2 bytes vs N bytes)          │
│  3. Implicit zero/nil omission with presence bitmap             │
│  4. Delta encoding for sorted sequences                          │
│  5. String interning for repeated values                         │
│  6. Prefix compression for related strings                       │
│  7. Type ID compaction (registered types use small IDs)          │
└─────────────────────────────────────────────────────────────────┘
```

### 3.3 Compression Affinity

The encoding is designed to compress well:

- **Grouped similar data**: Fields of same type cluster together
- **Predictable patterns**: Fixed-position type markers
- **Low entropy headers**: Consistent structure aids dictionary compression
- **Optional run-length encoding**: For repeated values in arrays

---

## 4. Wire Format Specification

### 4.1 Byte Order

All multi-byte values are encoded in **little-endian** order. This matches Go's native memory layout on common architectures and Protocol Buffers convention.

### 4.2 Variable-Length Integer Encoding (Varint)

Cramberry uses a modified LEB128 encoding optimized for small positive values:

#### 4.2.1 Unsigned Varint (uvarint)

```
┌────────────────────────────────────────────────────────────────┐
│  Bit 7 (MSB): Continuation flag (1 = more bytes follow)        │
│  Bits 0-6:    7 bits of data                                    │
├────────────────────────────────────────────────────────────────┤
│  Value Range          │ Bytes │ Encoding                        │
├───────────────────────┼───────┼─────────────────────────────────┤
│  0 - 127              │   1   │ [0xxxxxxx]                      │
│  128 - 16383          │   2   │ [1xxxxxxx] [0xxxxxxx]           │
│  16384 - 2097151      │   3   │ [1xxxxxxx] [1xxxxxxx] [0xxxxxxx]│
│  ...                  │  ...  │ ...                             │
│  max uint64           │  10   │ [1xxxxxxx]×9 [0xxxxxxx]         │
└───────────────────────┴───────┴─────────────────────────────────┘
```

#### 4.2.2 Signed Varint (svarint)

Uses ZigZag encoding to map signed integers to unsigned:

```
zigzag(n) = (n << 1) ^ (n >> 63)    // for int64
unzigzag(n) = (n >> 1) ^ -(n & 1)

Examples:
  0 → 0,  -1 → 1,  1 → 2,  -2 → 3,  2 → 4, ...
```

This ensures small absolute values use fewer bytes regardless of sign.

### 4.3 Wire Types

Each field is prefixed with a tag encoding both field number and wire type:

```
tag = (field_number << 3) | wire_type

┌──────────┬───────┬─────────────────────────────────────────────┐
│Wire Type │ Value │ Used For                                     │
├──────────┼───────┼─────────────────────────────────────────────┤
│ VARINT   │   0   │ int*, uint*, bool, enum                     │
│ FIXED64  │   1   │ fixed64, sfixed64, float64                  │
│ BYTES    │   2   │ string, []byte, embedded messages, arrays   │
│ FIXED32  │   5   │ fixed32, sfixed32, float32                  │
│ SVARINT  │   6   │ sint* (zigzag-encoded signed integers)      │
│ TYPEREF  │   7   │ polymorphic type (type ID + value)          │
└──────────┴───────┴─────────────────────────────────────────────┘
```

### 4.4 Length-Prefixed Data

All variable-length data uses uvarint length prefix:

```
┌─────────────────────────────────────────────┐
│  [length: uvarint] [data: length bytes]     │
└─────────────────────────────────────────────┘
```

### 4.5 Message Encoding

Messages are encoded as a sequence of tagged fields:

```
┌──────────────────────────────────────────────────────────────────┐
│ Message := Field*                                                 │
│ Field   := Tag Value                                              │
│ Tag     := uvarint  // (field_num << 3) | wire_type              │
│ Value   := (varies by wire_type)                                  │
└──────────────────────────────────────────────────────────────────┘
```

#### 4.5.1 Field Omission Rules

To minimize size, the following are omitted by default:

| Type | Omitted When |
|------|-------------|
| Numeric | Value is 0 |
| Bool | Value is false |
| String | Value is "" |
| Bytes | Value is nil or len()==0 |
| Slice | Value is nil or len()==0 |
| Map | Value is nil or len()==0 |
| Pointer | Value is nil |
| Struct | All fields omitted |

**Important**: A presence bitmap can be used when zero-values must be distinguished from absent values.

### 4.6 Presence Bitmap

For structs requiring explicit presence tracking:

```
┌─────────────────────────────────────────────────────────────────┐
│  [bitmap_length: uvarint] [bitmap: ceil(num_fields/8) bytes]    │
│  [field_values...]  // only present fields, in field order      │
└─────────────────────────────────────────────────────────────────┘

Bitmap bit positions correspond to field numbers (0-indexed).
Bit = 1 means field is present (even if zero value).
```

### 4.7 Polymorphic Type Encoding (TYPEREF)

For interface values and dynamic types:

```
┌─────────────────────────────────────────────────────────────────┐
│  [type_id: uvarint] [value_length: uvarint] [value: bytes]      │
└─────────────────────────────────────────────────────────────────┘

type_id:
  - 0 = nil (value_length must be 0)
  - 1-127 = built-in types (reserved)
  - 128+ = registered custom types
```

---

## 5. Type System

### 5.1 Primitive Type Mappings

```
┌─────────────────────────────────────────────────────────────────────┐
│ Go Type       │ Wire Type  │ Encoding                               │
├───────────────┼────────────┼────────────────────────────────────────┤
│ bool          │ VARINT     │ 0 or 1                                 │
│ int8          │ SVARINT    │ zigzag varint                          │
│ int16         │ SVARINT    │ zigzag varint                          │
│ int32         │ SVARINT    │ zigzag varint                          │
│ int64         │ SVARINT    │ zigzag varint                          │
│ int           │ SVARINT    │ zigzag varint (as int64)               │
│ uint8         │ VARINT     │ unsigned varint                        │
│ uint16        │ VARINT     │ unsigned varint                        │
│ uint32        │ VARINT     │ unsigned varint                        │
│ uint64        │ VARINT     │ unsigned varint                        │
│ uint          │ VARINT     │ unsigned varint (as uint64)            │
│ float32       │ FIXED32    │ IEEE 754 binary32, little-endian       │
│ float64       │ FIXED64    │ IEEE 754 binary64, little-endian       │
│ complex64     │ BYTES      │ 8 bytes: real(float32) + imag(float32) │
│ complex128    │ BYTES      │ 16 bytes: real(float64) + imag(float64)│
│ string        │ BYTES      │ length-prefixed UTF-8                  │
│ []byte        │ BYTES      │ length-prefixed raw bytes              │
│ uintptr       │ VARINT     │ unsigned varint (as uint64)            │
└───────────────┴────────────┴────────────────────────────────────────┘
```

### 5.2 Float Canonicalization

To ensure determinism, floating-point values are canonicalized:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Canonical Float Rules:                                              │
├─────────────────────────────────────────────────────────────────────┤
│ 1. Positive zero: +0.0 encoded as all zero bits                    │
│ 2. Negative zero: -0.0 converted to +0.0 before encoding           │
│ 3. NaN: Canonical quiet NaN (0x7FF8000000000000 for float64)       │
│ 4. +Inf: 0x7FF0000000000000 for float64                            │
│ 5. -Inf: 0xFFF0000000000000 for float64                            │
│ 6. Subnormals: Preserved (no flush-to-zero)                        │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.3 Composite Types

#### 5.3.1 Arrays and Slices

```
Encoding: BYTES wire type with packed elements

┌─────────────────────────────────────────────────────────────────────┐
│ Fixed-size elements (primitives):                                   │
│   [total_byte_length: uvarint] [element]* // packed, no tags       │
├─────────────────────────────────────────────────────────────────────┤
│ Variable-size elements (strings, nested messages):                  │
│   [total_byte_length: uvarint]                                      │
│   [element_count: uvarint]                                          │
│   [element_length: uvarint] [element_data]  // for each element    │
└─────────────────────────────────────────────────────────────────────┘
```

#### 5.3.2 Maps

Maps require deterministic ordering. Keys are sorted lexicographically by their encoded byte representation:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Map Encoding:                                                       │
│   [total_byte_length: uvarint]                                      │
│   [entry_count: uvarint]                                            │
│   [key_length: uvarint] [key_bytes]                                 │
│   [value_length: uvarint] [value_bytes]                             │
│   ... (repeated for each entry, sorted by key_bytes)               │
└─────────────────────────────────────────────────────────────────────┘

Key Ordering Algorithm:
1. Encode each key to bytes
2. Sort entries by key bytes (lexicographic, unsigned byte comparison)
3. Encode entries in sorted order
```

#### 5.3.3 Structs

Structs are encoded as messages with tagged fields:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Struct Encoding (standard mode):                                    │
│   [field_tag] [field_value]  // repeated for non-zero fields       │
│   // Fields ordered by field number                                 │
├─────────────────────────────────────────────────────────────────────┤
│ Struct Encoding (presence-tracking mode):                           │
│   [presence_bitmap]                                                 │
│   [field_value]*  // only present fields, in field number order    │
└─────────────────────────────────────────────────────────────────────┘
```

#### 5.3.4 Pointers

```
┌─────────────────────────────────────────────────────────────────────┐
│ nil pointer:     Field omitted (or bitmap bit = 0)                 │
│ non-nil pointer: Encode dereferenced value                         │
└─────────────────────────────────────────────────────────────────────┘
```

#### 5.3.5 Time and Duration

```
┌─────────────────────────────────────────────────────────────────────┐
│ time.Time:                                                          │
│   BYTES wire type, 12 bytes:                                        │
│   [seconds: int64 LE] [nanoseconds: int32 LE]                      │
│   // Seconds since Unix epoch (1970-01-01 00:00:00 UTC)            │
│   // Nanoseconds in range [0, 999999999]                           │
├─────────────────────────────────────────────────────────────────────┤
│ time.Duration:                                                      │
│   SVARINT wire type (nanoseconds as int64)                         │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.4 Type Aliases and Named Types

Named types inherit their underlying type's encoding unless custom encoding is registered:

```go
type UserID int64      // Encoded as SVARINT
type Amount float64    // Encoded as FIXED64
type Data []byte       // Encoded as BYTES
```

---

## 6. Interface and Polymorphic Support

### 6.1 Design Philosophy

Cramberry provides Amino-style polymorphic serialization with improvements:

1. **Compile-time registration**: Type mappings known at build time
2. **Stable type identifiers**: Based on registered name, not reflection
3. **Interface hierarchies**: Support for embedded interfaces
4. **Nil handling**: Explicit nil representation

### 6.2 Type Registration Model

```go
// Registration creates bidirectional mapping:
//   TypeID <-> Go Type <-> Type Name

type TypeRegistry struct {
    // Forward mappings
    idToType   map[TypeID]reflect.Type
    idToName   map[TypeID]string

    // Reverse mappings
    typeToID   map[reflect.Type]TypeID
    nameToID   map[string]TypeID

    // Interface registrations
    interfaces map[reflect.Type][]TypeID  // interface -> concrete implementations
}
```

### 6.3 Type ID Assignment

```
┌─────────────────────────────────────────────────────────────────────┐
│ Type ID Ranges:                                                     │
├───────────────┬─────────────────────────────────────────────────────┤
│ 0             │ nil (reserved)                                      │
│ 1-63          │ Built-in primitives                                 │
│ 64-127        │ Standard library types (time.Time, etc.)           │
│ 128-16383     │ Application types (2-byte varint)                  │
│ 16384+        │ Extended types (3+ byte varint)                    │
└───────────────┴─────────────────────────────────────────────────────┘
```

### 6.4 Interface Encoding

When encoding a value through an interface type:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Interface Value Encoding:                                           │
│                                                                     │
│ 1. Look up concrete type in registry → TypeID                      │
│ 2. Encode: [type_id: uvarint] [value_length: uvarint] [value]      │
│                                                                     │
│ Decoding:                                                           │
│ 1. Read type_id                                                     │
│ 2. Look up type in registry → reflect.Type                         │
│ 3. Create new instance of type                                      │
│ 4. Decode value into instance                                       │
│ 5. Return as interface value                                        │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.5 Interface Registration API

```go
// Define an interface
type Message interface {
    Validate() error
}

// Register interface and implementations
registry := cramberry.NewRegistry()

// Register the interface itself
registry.RegisterInterface((*Message)(nil), "app.Message")

// Register concrete implementations with unique names
registry.RegisterImplementation((*Message)(nil), &TextMessage{}, "app.TextMessage")
registry.RegisterImplementation((*Message)(nil), &BinaryMessage{}, "app.BinaryMessage")

// Type IDs are derived from registration order or explicit assignment
registry.RegisterImplementationWithID((*Message)(nil), &PriorityMessage{}, "app.PriorityMessage", 200)
```

### 6.6 Nested Interface Handling

For interfaces containing other interfaces:

```go
type Container struct {
    Items []Message  // slice of interface type
}

// Encoding:
// 1. Container encoded as struct
// 2. Items field encoded as BYTES (array)
// 3. Each element encoded with TYPEREF (type_id + value)
```

### 6.7 any/interface{} Support

The empty interface requires special handling:

```go
// Register types that may be stored in interface{}
registry.RegisterAnyType(&User{}, "app.User")
registry.RegisterAnyType(&Order{}, "app.Order")

// Encoding interface{} value:
// 1. Determine concrete type
// 2. Look up in any-type registry
// 3. Encode as TYPEREF

// Decoding to interface{}:
// 1. Read type_id
// 2. Instantiate registered type
// 3. Return as interface{}
```

### 6.8 Type Discrimination Strategies

Three modes for polymorphic encoding:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Mode 1: Type ID Prefix (default)                                    │
│   [type_id] [value]                                                 │
│   + Compact for registered types                                    │
│   - Requires registry synchronization                               │
├─────────────────────────────────────────────────────────────────────┤
│ Mode 2: Type Name Prefix                                            │
│   [name_length] [type_name_utf8] [value]                           │
│   + Self-describing                                                 │
│   - Larger wire size                                                │
├─────────────────────────────────────────────────────────────────────┤
│ Mode 3: Type Hash Prefix                                            │
│   [type_hash: 4 bytes] [value]                                      │
│   + Fixed size prefix                                               │
│   + No registry needed for decoding (with schema)                   │
│   - Collision possibility (mitigated by 32-bit hash)               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. Custom Type Registration

### 7.1 Custom Encoder Interface

```go
// Marshaler allows types to define custom encoding
type Marshaler interface {
    MarshalCramberry() ([]byte, error)
}

// Unmarshaler allows types to define custom decoding
type Unmarshaler interface {
    UnmarshalCramberry([]byte) error
}

// Sizer allows types to pre-compute encoded size (for buffer pre-allocation)
type Sizer interface {
    CramberrySize() int
}
```

### 7.2 Registration with Custom Codec

```go
// Full custom codec registration
type Codec[T any] interface {
    Encode(w *Writer, value T) error
    Decode(r *Reader) (T, error)
    Size(value T) int
}

// Register with custom codec
registry.RegisterCodec[BigInt](bigIntCodec{})
registry.RegisterCodec[UUID](uuidCodec{})
```

### 7.3 Example: Optimized UUID Encoding

```go
type UUID [16]byte

type uuidCodec struct{}

func (c uuidCodec) Encode(w *Writer, u UUID) error {
    // Direct 16-byte encoding (no length prefix needed - fixed size)
    return w.WriteFixedBytes(u[:])
}

func (c uuidCodec) Decode(r *Reader) (UUID, error) {
    var u UUID
    _, err := io.ReadFull(r, u[:])
    return u, err
}

func (c uuidCodec) Size(u UUID) int {
    return 16
}
```

### 7.4 Example: Compressed BigInt

```go
type bigIntCodec struct{}

func (c bigIntCodec) Encode(w *Writer, b *big.Int) error {
    if b == nil {
        return w.WriteUvarint(0)  // nil marker
    }

    sign := byte(0)
    if b.Sign() < 0 {
        sign = 1
    }

    bytes := b.Bytes()  // big-endian, no leading zeros

    // Format: [sign: 1 byte] [length: uvarint] [magnitude: bytes]
    w.WriteByte(sign)
    w.WriteUvarint(uint64(len(bytes)))
    return w.WriteBytes(bytes)
}

func (c bigIntCodec) Decode(r *Reader) (*big.Int, error) {
    sign, _ := r.ReadByte()
    length, _ := r.ReadUvarint()

    if length == 0 && sign == 0 {
        return nil, nil  // nil BigInt
    }

    bytes := make([]byte, length)
    r.Read(bytes)

    b := new(big.Int).SetBytes(bytes)
    if sign == 1 {
        b.Neg(b)
    }
    return b, nil
}
```

### 7.5 Field-Level Custom Encoding

Support for per-field encoding options via struct tags:

```go
type Record struct {
    // Use delta encoding for sorted timestamps
    Timestamps []time.Time `cramberry:"delta"`

    // Use run-length encoding for sparse data
    Flags []bool `cramberry:"rle"`

    // Use dictionary encoding for repeated strings
    Tags []string `cramberry:"dict"`

    // Fixed-size encoding (no length prefix)
    ID [32]byte `cramberry:"fixed"`

    // Custom codec by name
    Amount Decimal `cramberry:"codec=decimal128"`
}
```

### 7.6 Built-in Optimized Encodings

```
┌─────────────────────────────────────────────────────────────────────┐
│ Encoding     │ Use Case                    │ Tag                    │
├──────────────┼─────────────────────────────┼────────────────────────┤
│ delta        │ Sorted numeric sequences    │ cramberry:"delta"      │
│ rle          │ Repeated values             │ cramberry:"rle"        │
│ dict         │ Repeated strings            │ cramberry:"dict"       │
│ fixed        │ Fixed-size arrays           │ cramberry:"fixed"      │
│ packed       │ Primitive slices            │ cramberry:"packed"     │
│ sparse       │ Sparse arrays with defaults │ cramberry:"sparse"     │
│ bitpack      │ Small integers in sequence  │ cramberry:"bitpack=N"  │
└──────────────┴─────────────────────────────┴────────────────────────┘
```

---

## 8. Schema Definition Language (SDL)

### 8.1 Design Goals

1. **Language-agnostic**: Not tied to Go syntax
2. **Complete type information**: Enough for code generation
3. **Human-readable**: Easy to review and edit
4. **Machine-parseable**: Simple grammar for tool creation
5. **Extensible**: Support for custom annotations

### 8.2 File Format

Schema files use the `.cramberry` extension.

### 8.3 Grammar Specification

```ebnf
Schema      = PackageDecl? ImportDecl* Declaration*
PackageDecl = "package" QualifiedName ";"
ImportDecl  = "import" StringLit ("as" Identifier)? ";"
Declaration = TypeDecl | EnumDecl | InterfaceDecl | ServiceDecl

TypeDecl      = Annotation* "type" Identifier GenericParams? ("extends" TypeRef)? "{" FieldDecl* "}"
FieldDecl     = Annotation* Identifier ":" TypeRef ("=" DefaultValue)? ";"
GenericParams = "<" Identifier ("," Identifier)* ">"

EnumDecl    = Annotation* "enum" Identifier "{" EnumValue* "}"
EnumValue   = Identifier ("=" IntLit)? ";"

InterfaceDecl = Annotation* "interface" Identifier "{" MethodDecl* "}"
MethodDecl    = Identifier "(" ParamList? ")" (":" TypeRef)? ";"

ServiceDecl  = "service" Identifier "{" RpcDecl* "}"
RpcDecl      = "rpc" Identifier "(" TypeRef ")" ":" TypeRef ";"

TypeRef     = PrimitiveType | QualifiedName GenericArgs? | ArrayType | MapType | OptionalType
ArrayType   = "[" "]" TypeRef
MapType     = "map" "<" TypeRef "," TypeRef ">"
OptionalType = TypeRef "?"

PrimitiveType = "bool" | "int8" | "int16" | "int32" | "int64"
              | "uint8" | "uint16" | "uint32" | "uint64"
              | "float32" | "float64" | "string" | "bytes"
              | "int" | "uint" | "any"

Annotation  = "@" Identifier ("(" AnnotationArgs ")")?
AnnotationArgs = AnnotationArg ("," AnnotationArg)*
AnnotationArg  = Identifier "=" Literal

Literal     = StringLit | IntLit | FloatLit | BoolLit
QualifiedName = Identifier ("." Identifier)*
GenericArgs = "<" TypeRef ("," TypeRef)* ">"
```

### 8.4 Example Schema

```cramberry
// Package declaration
package myapp.models;

// Imports
import "google/protobuf/timestamp.cramberry" as timestamp;
import "cramberry/stdlib.cramberry";

// Enum definition
@doc("Status of an order")
enum OrderStatus {
    PENDING = 0;
    CONFIRMED = 1;
    SHIPPED = 2;
    DELIVERED = 3;
    CANCELLED = 4;
}

// Interface definition (for polymorphism)
@typeprefix("myapp.payment")
interface PaymentMethod {
    validate(): bool;
    getAmount(): int64;
}

// Concrete types implementing interface
@implements(PaymentMethod)
@typeid(128)
type CreditCardPayment {
    @field(1) cardNumber: string;
    @field(2) expiryMonth: uint8;
    @field(3) expiryYear: uint16;
    @field(4) cvv: string;
    @field(5) amount: int64;
}

@implements(PaymentMethod)
@typeid(129)
type CryptoPayment {
    @field(1) walletAddress: string;
    @field(2) currency: string;
    @field(3) amount: int64;
    @field(4) txHash: string?;  // optional
}

// Complex type with various features
@doc("Represents a customer order")
type Order {
    @field(1) id: bytes;  // UUID as bytes
    @field(2) customerId: int64;
    @field(3) status: OrderStatus;
    @field(4) items: []OrderItem;
    @field(5) payment: PaymentMethod;  // polymorphic field
    @field(6) metadata: map<string, string>;
    @field(7) createdAt: timestamp.Timestamp;
    @field(8) updatedAt: timestamp.Timestamp?;
    @field(9) tags: []string = [];  // default empty
}

type OrderItem {
    @field(1) productId: int64;
    @field(2) quantity: uint32;
    @field(3) unitPrice: int64;  // cents
    @field(4) @encoding("delta") discounts: []int64;
}

// Generic type
type Result<T, E> {
    @field(1) @oneof value: T;
    @field(2) @oneof error: E;
}

// Service definition (for RPC code generation)
service OrderService {
    rpc CreateOrder(Order): Order;
    rpc GetOrder(GetOrderRequest): Order;
    rpc ListOrders(ListOrdersRequest): ListOrdersResponse;
    rpc StreamOrders(StreamOrdersRequest): stream Order;
}
```

### 8.5 Annotations Reference

```
┌─────────────────────────────────────────────────────────────────────┐
│ Annotation          │ Target    │ Description                       │
├─────────────────────┼───────────┼───────────────────────────────────┤
│ @doc("...")         │ Any       │ Documentation string              │
│ @deprecated("...")  │ Any       │ Mark as deprecated                │
│ @field(N)           │ Field     │ Field number for wire format      │
│ @typeid(N)          │ Type      │ Explicit type ID assignment       │
│ @typeprefix("...")  │ Interface │ Prefix for implementation names   │
│ @implements(I)      │ Type      │ Declare interface implementation  │
│ @encoding("...")    │ Field     │ Custom encoding strategy          │
│ @oneof              │ Field     │ Union/variant field               │
│ @packed             │ Field     │ Use packed encoding for arrays    │
│ @optional           │ Field     │ Field may be omitted              │
│ @required           │ Field     │ Field must be present             │
│ @validate("...")    │ Field     │ Validation rule                   │
│ @json("name")       │ Field     │ JSON field name override          │
│ @go_type("...")     │ Type      │ Go type override                  │
│ @ts_type("...")     │ Type      │ TypeScript type override          │
└─────────────────────┴───────────┴───────────────────────────────────┘
```

### 8.6 Built-in Types Library

Standard library (`cramberry/stdlib.cramberry`):

```cramberry
package cramberry.stdlib;

// Temporal types
type Timestamp {
    @field(1) seconds: int64;
    @field(2) nanos: int32;
}

type Duration {
    @field(1) nanos: int64;
}

// Wrapper types for nullable primitives
type BoolValue { @field(1) value: bool; }
type Int32Value { @field(1) value: int32; }
type Int64Value { @field(1) value: int64; }
type UInt32Value { @field(1) value: uint32; }
type UInt64Value { @field(1) value: uint64; }
type FloatValue { @field(1) value: float32; }
type DoubleValue { @field(1) value: float64; }
type StringValue { @field(1) value: string; }
type BytesValue { @field(1) value: bytes; }

// Common utility types
type Empty {}

type Any {
    @field(1) typeUrl: string;
    @field(2) value: bytes;
}
```

---

## 9. Code Generation

### 9.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Code Generation Pipeline                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  .cramberry files                                                    │
│        │                                                             │
│        ▼                                                             │
│  ┌──────────┐    ┌─────────────────────────────────────────────┐   │
│  │  Parser  │───▶│  Abstract Schema Representation (ASR)       │   │
│  └──────────┘    └─────────────────────────────────────────────┘   │
│                              │                                       │
│         ┌────────────────────┼────────────────────┐                 │
│         ▼                    ▼                    ▼                 │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐          │
│  │ Go Generator│     │ TS Generator│     │Rust Generator│          │
│  └─────────────┘     └─────────────┘     └─────────────┘          │
│         │                    │                    │                 │
│         ▼                    ▼                    ▼                 │
│    *.gen.go             *.gen.ts            *.gen.rs               │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 9.2 Generated Code Structure (Go)

For each type, generate:

```go
// ============================================================
// Generated from: myapp/models.cramberry
// DO NOT EDIT - changes will be overwritten
// ============================================================

package models

import (
    "io"
    cramberry "github.com/cramberry/cramberry-go"
)

// --- Type: Order ---

// MarshalCramberry implements cramberry.Marshaler
func (x *Order) MarshalCramberry() ([]byte, error) {
    size := x.CramberrySize()
    buf := make([]byte, 0, size)
    return x.AppendCramberry(buf)
}

// AppendCramberry appends the encoded form to the buffer
func (x *Order) AppendCramberry(buf []byte) ([]byte, error) {
    var err error

    // Field 1: id (bytes)
    if len(x.Id) > 0 {
        buf = cramberry.AppendTag(buf, 1, cramberry.WireBytes)
        buf = cramberry.AppendBytes(buf, x.Id)
    }

    // Field 2: customerId (int64)
    if x.CustomerId != 0 {
        buf = cramberry.AppendTag(buf, 2, cramberry.WireSVarint)
        buf = cramberry.AppendSVarint(buf, x.CustomerId)
    }

    // Field 3: status (enum)
    if x.Status != 0 {
        buf = cramberry.AppendTag(buf, 3, cramberry.WireVarint)
        buf = cramberry.AppendUVarint(buf, uint64(x.Status))
    }

    // Field 4: items ([]OrderItem)
    if len(x.Items) > 0 {
        buf = cramberry.AppendTag(buf, 4, cramberry.WireBytes)
        buf, err = cramberry.AppendSlice(buf, x.Items)
        if err != nil {
            return nil, err
        }
    }

    // Field 5: payment (interface PaymentMethod)
    if x.Payment != nil {
        buf = cramberry.AppendTag(buf, 5, cramberry.WireTypeRef)
        buf, err = cramberry.AppendInterface(buf, x.Payment)
        if err != nil {
            return nil, err
        }
    }

    // ... remaining fields

    return buf, nil
}

// UnmarshalCramberry implements cramberry.Unmarshaler
func (x *Order) UnmarshalCramberry(data []byte) error {
    r := cramberry.NewReader(data)
    return x.ReadCramberry(r)
}

// ReadCramberry reads from a cramberry.Reader
func (x *Order) ReadCramberry(r *cramberry.Reader) error {
    for !r.EOF() {
        tag, wireType, err := r.ReadTag()
        if err != nil {
            return err
        }

        switch tag {
        case 1: // id
            x.Id, err = r.ReadBytes()
        case 2: // customerId
            x.CustomerId, err = r.ReadSVarint64()
        case 3: // status
            v, err := r.ReadUVarint64()
            if err == nil {
                x.Status = OrderStatus(v)
            }
        case 4: // items
            err = cramberry.ReadSlice(r, &x.Items)
        case 5: // payment
            x.Payment, err = cramberry.ReadInterface[PaymentMethod](r)
        // ... remaining fields
        default:
            err = r.SkipField(wireType)
        }

        if err != nil {
            return err
        }
    }
    return nil
}

// CramberrySize returns the encoded size in bytes
func (x *Order) CramberrySize() int {
    size := 0

    if len(x.Id) > 0 {
        size += cramberry.TagSize(1)
        size += cramberry.BytesSize(x.Id)
    }

    if x.CustomerId != 0 {
        size += cramberry.TagSize(2)
        size += cramberry.SVarintSize(x.CustomerId)
    }

    // ... remaining fields

    return size
}

// --- Registration ---

func init() {
    cramberry.RegisterType[Order]("myapp.models.Order")
    cramberry.RegisterType[OrderItem]("myapp.models.OrderItem")

    // Interface implementations
    cramberry.RegisterImplementation[PaymentMethod, CreditCardPayment](128, "myapp.payment.CreditCardPayment")
    cramberry.RegisterImplementation[PaymentMethod, CryptoPayment](129, "myapp.payment.CryptoPayment")
}
```

### 9.3 Generated Code Structure (TypeScript)

```typescript
// ============================================================
// Generated from: myapp/models.cramberry
// DO NOT EDIT - changes will be overwritten
// ============================================================

import { Reader, Writer, Registry } from '@cramberry/runtime';

// --- Enums ---

export enum OrderStatus {
    PENDING = 0,
    CONFIRMED = 1,
    SHIPPED = 2,
    DELIVERED = 3,
    CANCELLED = 4,
}

// --- Interfaces ---

export interface PaymentMethod {
    readonly _cramberryType: string;
    validate(): boolean;
    getAmount(): bigint;
}

// --- Types ---

export interface Order {
    id: Uint8Array;
    customerId: bigint;
    status: OrderStatus;
    items: OrderItem[];
    payment: PaymentMethod | null;
    metadata: Map<string, string>;
    createdAt: Timestamp;
    updatedAt: Timestamp | null;
    tags: string[];
}

export interface OrderItem {
    productId: bigint;
    quantity: number;
    unitPrice: bigint;
    discounts: bigint[];
}

export interface CreditCardPayment extends PaymentMethod {
    readonly _cramberryType: 'myapp.payment.CreditCardPayment';
    cardNumber: string;
    expiryMonth: number;
    expiryYear: number;
    cvv: string;
    amount: bigint;
}

export interface CryptoPayment extends PaymentMethod {
    readonly _cramberryType: 'myapp.payment.CryptoPayment';
    walletAddress: string;
    currency: string;
    amount: bigint;
    txHash: string | null;
}

// --- Codec Functions ---

export const Order = {
    encode(writer: Writer, value: Order): void {
        // Field 1: id
        if (value.id.length > 0) {
            writer.tag(1, WireType.Bytes);
            writer.bytes(value.id);
        }

        // Field 2: customerId
        if (value.customerId !== 0n) {
            writer.tag(2, WireType.SVarint);
            writer.svarint(value.customerId);
        }

        // ... remaining fields
    },

    decode(reader: Reader): Order {
        const result: Order = {
            id: new Uint8Array(0),
            customerId: 0n,
            status: OrderStatus.PENDING,
            items: [],
            payment: null,
            metadata: new Map(),
            createdAt: { seconds: 0n, nanos: 0 },
            updatedAt: null,
            tags: [],
        };

        while (!reader.eof()) {
            const [fieldNum, wireType] = reader.tag();

            switch (fieldNum) {
                case 1:
                    result.id = reader.bytes();
                    break;
                case 2:
                    result.customerId = reader.svarint();
                    break;
                // ... remaining fields
                default:
                    reader.skip(wireType);
            }
        }

        return result;
    },

    size(value: Order): number {
        let size = 0;
        if (value.id.length > 0) {
            size += tagSize(1) + bytesSize(value.id);
        }
        // ... remaining fields
        return size;
    },
};

// --- Registry Setup ---

const registry = new Registry();
registry.registerType('myapp.models.Order', Order);
registry.registerType('myapp.models.OrderItem', OrderItem);
registry.registerImplementation<PaymentMethod>(
    128,
    'myapp.payment.CreditCardPayment',
    CreditCardPayment
);
registry.registerImplementation<PaymentMethod>(
    129,
    'myapp.payment.CryptoPayment',
    CryptoPayment
);

export { registry };
```

### 9.4 Code Generator CLI

```bash
# Basic usage
cramberry generate --lang=go --out=./gen schema/*.cramberry

# Multiple languages
cramberry generate \
    --lang=go --out=./gen/go \
    --lang=typescript --out=./gen/ts \
    --lang=rust --out=./gen/rust \
    schema/*.cramberry

# With options
cramberry generate \
    --lang=go \
    --out=./gen \
    --package-prefix=github.com/myorg/myapp \
    --with-json \           # Generate JSON tags
    --with-validation \     # Generate validation code
    --with-builders \       # Generate builder pattern
    --with-equals \         # Generate equality methods
    --with-clone \          # Generate clone methods
    schema/*.cramberry
```

### 9.5 Generated Feature Matrix

```
┌─────────────────────────────────────────────────────────────────────┐
│ Feature              │ Go │ TypeScript │ Rust │ Python │           │
├──────────────────────┼────┼────────────┼──────┼────────┤           │
│ Encode/Decode        │ ✓  │     ✓      │  ✓   │   ✓    │           │
│ Size calculation     │ ✓  │     ✓      │  ✓   │   ✓    │           │
│ Streaming            │ ✓  │     ✓      │  ✓   │   ✓    │           │
│ Type-safe interfaces │ ✓  │     ✓      │  ✓   │   ◐    │           │
│ JSON interop         │ ✓  │     ✓      │  ✓   │   ✓    │           │
│ Validation           │ ✓  │     ✓      │  ✓   │   ✓    │           │
│ Builder pattern      │ ✓  │     ✓      │  ✓   │   ✗    │           │
│ Zero-copy decode     │ ✓  │     ◐      │  ✓   │   ✗    │           │
│ Generics             │ ✓  │     ✓      │  ✓   │   ✓    │           │
│ Async streaming      │ ✗  │     ✓      │  ✓   │   ✓    │           │
└──────────────────────┴────┴────────────┴──────┴────────┴───────────┘
  ✓ = Full support, ◐ = Partial support, ✗ = Not supported
```

---

## 10. Streaming Protocol

### 10.1 Use Cases

1. **Large messages**: Encode/decode messages exceeding memory
2. **Real-time data**: Process data as it arrives
3. **Multiplexed streams**: Multiple logical streams over single connection
4. **Incremental updates**: Append-only data structures

### 10.2 Stream Frame Format

```
┌─────────────────────────────────────────────────────────────────────┐
│ Stream Frame:                                                        │
│                                                                      │
│   [frame_header: 1 byte] [payload_length: uvarint] [payload: bytes] │
│                                                                      │
│ Frame Header Bits:                                                   │
│   Bits 0-3: Frame type                                              │
│   Bit 4:    End of message flag                                     │
│   Bit 5:    Compressed flag                                         │
│   Bits 6-7: Reserved                                                │
└─────────────────────────────────────────────────────────────────────┘
```

### 10.3 Frame Types

```
┌──────────┬───────┬──────────────────────────────────────────────────┐
│ Type     │ Value │ Description                                       │
├──────────┼───────┼──────────────────────────────────────────────────┤
│ DATA     │  0x0  │ Message data (potentially partial)               │
│ HEADER   │  0x1  │ Stream metadata/headers                          │
│ TRAILER  │  0x2  │ Stream trailers (final metadata)                 │
│ RESET    │  0x3  │ Abort stream with error                          │
│ PING     │  0x4  │ Keep-alive / latency measurement                 │
│ PONG     │  0x5  │ Response to PING                                 │
│ WINDOW   │  0x6  │ Flow control window update                       │
│ SCHEMA   │  0x7  │ Inline schema for self-describing streams        │
└──────────┴───────┴──────────────────────────────────────────────────┘
```

### 10.4 Streaming API (Go)

```go
// Writer interface for streaming encoding
type StreamWriter interface {
    // Write a complete message
    WriteMessage(msg any) error

    // Start writing a large message in chunks
    StartMessage(typeID TypeID) (MessageWriter, error)

    // Write stream metadata
    WriteHeader(key string, value []byte) error
    WriteTrailer(key string, value []byte) error

    // Flow control
    Flush() error
    Close() error
}

// MessageWriter for chunked message writing
type MessageWriter interface {
    // Write a field
    WriteField(fieldNum int, value any) error

    // Write raw bytes (for large byte fields)
    WriteFieldBytes(fieldNum int, data []byte) error

    // Start a nested message field
    StartNestedMessage(fieldNum int, typeID TypeID) (MessageWriter, error)

    // Start an array field for streaming elements
    StartArray(fieldNum int) (ArrayWriter, error)

    // Complete the message
    Complete() error
}

// Reader interface for streaming decoding
type StreamReader interface {
    // Read next complete message
    ReadMessage() (any, error)

    // Read next message into specific type
    ReadMessageInto(target any) error

    // Read messages as a channel
    Messages() <-chan MessageOrError

    // Access stream metadata
    Headers() map[string][]byte
    Trailers() map[string][]byte

    // Check for more data
    HasMore() bool
    Close() error
}
```

### 10.5 Streaming Example

```go
// Writing a stream of orders
func StreamOrders(w cramberry.StreamWriter, orders <-chan *Order) error {
    // Write stream header
    w.WriteHeader("content-type", []byte("cramberry/stream"))
    w.WriteHeader("schema-version", []byte("1.0"))

    // Stream each order
    for order := range orders {
        if err := w.WriteMessage(order); err != nil {
            return err
        }
    }

    // Write trailer with count
    w.WriteTrailer("message-count", cramberry.EncodeUVarint(count))
    return w.Close()
}

// Reading a stream of orders
func ProcessOrderStream(r cramberry.StreamReader) error {
    // Type-safe streaming decode
    for r.HasMore() {
        order := &Order{}
        if err := r.ReadMessageInto(order); err != nil {
            return err
        }
        processOrder(order)
    }

    // Check trailers
    if count := r.Trailers()["message-count"]; count != nil {
        log.Printf("Processed %d orders", cramberry.DecodeUVarint(count))
    }

    return nil
}

// Channel-based streaming
func StreamOrdersChan(r cramberry.StreamReader) {
    for msg := range r.Messages() {
        if msg.Error != nil {
            log.Printf("Stream error: %v", msg.Error)
            continue
        }

        order := msg.Value.(*Order)
        processOrder(order)
    }
}
```

### 10.6 Large Field Streaming

For messages with large byte arrays or nested structures:

```go
func WriteLargeDocument(w cramberry.StreamWriter, doc *Document) error {
    mw, err := w.StartMessage(DocumentTypeID)
    if err != nil {
        return err
    }

    // Write small fields normally
    mw.WriteField(1, doc.ID)
    mw.WriteField(2, doc.Title)

    // Stream large content field in chunks
    contentWriter, err := mw.StartFieldStream(3)
    if err != nil {
        return err
    }

    // Write content in 64KB chunks
    buf := make([]byte, 64*1024)
    for {
        n, err := doc.ContentReader.Read(buf)
        if n > 0 {
            contentWriter.Write(buf[:n])
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }
    contentWriter.Close()

    // Complete the message
    return mw.Complete()
}
```

### 10.7 Compression Integration

Streaming supports frame-level compression:

```go
// Enable compression for the stream
w := cramberry.NewStreamWriter(conn, cramberry.StreamOptions{
    Compression: cramberry.CompressionLZ4,
    CompressionLevel: cramberry.CompressionFast,
    CompressionThreshold: 1024,  // Only compress frames > 1KB
})

// Reader automatically detects and decompresses
r := cramberry.NewStreamReader(conn)
```

Supported compression algorithms:
- **LZ4**: Fast compression/decompression, moderate ratio
- **Zstd**: Better ratio, still fast
- **Snappy**: Google's fast compression
- **None**: No compression (default)

---

## 11. Go Reference Implementation

### 11.1 Package Structure

```
github.com/cramberry/cramberry-go/
├── cramberry.go           # Core types and interfaces
├── registry.go            # Type registration
├── writer.go              # Encoding implementation
├── reader.go              # Decoding implementation
├── stream.go              # Streaming support
├── wire/
│   ├── varint.go          # Varint encoding
│   ├── fixed.go           # Fixed-width encoding
│   └── tag.go             # Tag encoding
├── reflect/
│   ├── encoder.go         # Reflection-based encoder
│   ├── decoder.go         # Reflection-based decoder
│   └── cache.go           # Type info caching
├── gen/
│   └── templates/         # Code generation templates
└── cmd/
    └── cramberry/         # CLI tool
```

### 11.2 Core Interfaces

```go
package cramberry

import "io"

// Marshaler is implemented by types that can marshal themselves
type Marshaler interface {
    MarshalCramberry() ([]byte, error)
}

// Unmarshaler is implemented by types that can unmarshal themselves
type Unmarshaler interface {
    UnmarshalCramberry([]byte) error
}

// Appender is implemented by types that can append to a buffer
type Appender interface {
    AppendCramberry([]byte) ([]byte, error)
}

// Sizer is implemented by types that know their encoded size
type Sizer interface {
    CramberrySize() int
}

// StreamEncoder can encode to a streaming writer
type StreamEncoder interface {
    EncodeCramberry(w *Writer) error
}

// StreamDecoder can decode from a streaming reader
type StreamDecoder interface {
    DecodeCramberry(r *Reader) error
}

// TypeID uniquely identifies a registered type
type TypeID uint32

// WireType indicates how a value is encoded on the wire
type WireType uint8

const (
    WireVarint  WireType = 0
    WireFixed64 WireType = 1
    WireBytes   WireType = 2
    WireFixed32 WireType = 5
    WireSVarint WireType = 6
    WireTypeRef WireType = 7
)
```

### 11.3 Writer Implementation

```go
package cramberry

// Writer provides methods for encoding cramberry data
type Writer struct {
    buf      []byte
    registry *Registry
    err      error
}

// NewWriter creates a new Writer with optional initial capacity
func NewWriter(capacity int) *Writer {
    return &Writer{
        buf:      make([]byte, 0, capacity),
        registry: DefaultRegistry,
    }
}

// Reset clears the buffer for reuse
func (w *Writer) Reset() {
    w.buf = w.buf[:0]
    w.err = nil
}

// Bytes returns the encoded bytes
func (w *Writer) Bytes() []byte {
    return w.buf
}

// WriteUVarint writes an unsigned varint
func (w *Writer) WriteUVarint(v uint64) {
    w.buf = appendUVarint(w.buf, v)
}

// WriteSVarint writes a signed varint (zigzag encoded)
func (w *Writer) WriteSVarint(v int64) {
    w.buf = appendSVarint(w.buf, v)
}

// WriteFixed32 writes a 32-bit value
func (w *Writer) WriteFixed32(v uint32) {
    w.buf = append(w.buf,
        byte(v),
        byte(v>>8),
        byte(v>>16),
        byte(v>>24),
    )
}

// WriteFixed64 writes a 64-bit value
func (w *Writer) WriteFixed64(v uint64) {
    w.buf = append(w.buf,
        byte(v),
        byte(v>>8),
        byte(v>>16),
        byte(v>>24),
        byte(v>>32),
        byte(v>>40),
        byte(v>>48),
        byte(v>>56),
    )
}

// WriteBytes writes length-prefixed bytes
func (w *Writer) WriteBytes(data []byte) {
    w.WriteUVarint(uint64(len(data)))
    w.buf = append(w.buf, data...)
}

// WriteString writes length-prefixed string
func (w *Writer) WriteString(s string) {
    w.WriteUVarint(uint64(len(s)))
    w.buf = append(w.buf, s...)
}

// WriteTag writes a field tag
func (w *Writer) WriteTag(fieldNum int, wireType WireType) {
    w.WriteUVarint(uint64(fieldNum<<3) | uint64(wireType))
}

// WriteInterface writes a polymorphic interface value
func (w *Writer) WriteInterface(v any) error {
    if v == nil {
        w.WriteUVarint(0) // nil type ID
        w.WriteUVarint(0) // zero length
        return nil
    }

    typeID, ok := w.registry.TypeIDFor(v)
    if !ok {
        return fmt.Errorf("cramberry: unregistered type %T", v)
    }

    w.WriteUVarint(uint64(typeID))

    // Encode value to temporary buffer to get length
    tmpW := NewWriter(256)
    tmpW.registry = w.registry
    if err := tmpW.WriteValue(v); err != nil {
        return err
    }

    w.WriteBytes(tmpW.Bytes())
    return nil
}

// WriteValue writes any value using reflection or custom codec
func (w *Writer) WriteValue(v any) error {
    // Check for Marshaler interface
    if m, ok := v.(Marshaler); ok {
        data, err := m.MarshalCramberry()
        if err != nil {
            return err
        }
        w.buf = append(w.buf, data...)
        return nil
    }

    // Check for StreamEncoder interface
    if e, ok := v.(StreamEncoder); ok {
        return e.EncodeCramberry(w)
    }

    // Fall back to reflection
    return w.writeReflect(reflect.ValueOf(v))
}
```

### 11.4 Reader Implementation

```go
package cramberry

// Reader provides methods for decoding cramberry data
type Reader struct {
    data     []byte
    pos      int
    registry *Registry
}

// NewReader creates a Reader from bytes
func NewReader(data []byte) *Reader {
    return &Reader{
        data:     data,
        registry: DefaultRegistry,
    }
}

// EOF returns true if all data has been read
func (r *Reader) EOF() bool {
    return r.pos >= len(r.data)
}

// Remaining returns the number of unread bytes
func (r *Reader) Remaining() int {
    return len(r.data) - r.pos
}

// ReadUVarint reads an unsigned varint
func (r *Reader) ReadUVarint() (uint64, error) {
    v, n := decodeUVarint(r.data[r.pos:])
    if n <= 0 {
        return 0, ErrInvalidVarint
    }
    r.pos += n
    return v, nil
}

// ReadSVarint reads a signed varint (zigzag encoded)
func (r *Reader) ReadSVarint() (int64, error) {
    v, err := r.ReadUVarint()
    if err != nil {
        return 0, err
    }
    return unzigzag(v), nil
}

// ReadFixed32 reads a 32-bit value
func (r *Reader) ReadFixed32() (uint32, error) {
    if r.Remaining() < 4 {
        return 0, io.ErrUnexpectedEOF
    }
    v := uint32(r.data[r.pos]) |
        uint32(r.data[r.pos+1])<<8 |
        uint32(r.data[r.pos+2])<<16 |
        uint32(r.data[r.pos+3])<<24
    r.pos += 4
    return v, nil
}

// ReadFixed64 reads a 64-bit value
func (r *Reader) ReadFixed64() (uint64, error) {
    if r.Remaining() < 8 {
        return 0, io.ErrUnexpectedEOF
    }
    v := uint64(r.data[r.pos]) |
        uint64(r.data[r.pos+1])<<8 |
        uint64(r.data[r.pos+2])<<16 |
        uint64(r.data[r.pos+3])<<24 |
        uint64(r.data[r.pos+4])<<32 |
        uint64(r.data[r.pos+5])<<40 |
        uint64(r.data[r.pos+6])<<48 |
        uint64(r.data[r.pos+7])<<56
    r.pos += 8
    return v, nil
}

// ReadBytes reads length-prefixed bytes
func (r *Reader) ReadBytes() ([]byte, error) {
    length, err := r.ReadUVarint()
    if err != nil {
        return nil, err
    }
    if uint64(r.Remaining()) < length {
        return nil, io.ErrUnexpectedEOF
    }
    data := make([]byte, length)
    copy(data, r.data[r.pos:r.pos+int(length)])
    r.pos += int(length)
    return data, nil
}

// ReadString reads length-prefixed string
func (r *Reader) ReadString() (string, error) {
    data, err := r.ReadBytes()
    if err != nil {
        return "", err
    }
    return string(data), nil
}

// ReadTag reads a field tag
func (r *Reader) ReadTag() (fieldNum int, wireType WireType, err error) {
    v, err := r.ReadUVarint()
    if err != nil {
        return 0, 0, err
    }
    return int(v >> 3), WireType(v & 0x7), nil
}

// ReadInterface reads a polymorphic interface value
func ReadInterface[T any](r *Reader) (T, error) {
    var zero T

    typeID, err := r.ReadUVarint()
    if err != nil {
        return zero, err
    }

    length, err := r.ReadUVarint()
    if err != nil {
        return zero, err
    }

    if typeID == 0 {
        return zero, nil // nil value
    }

    // Look up type in registry
    typ, ok := r.registry.TypeForID(TypeID(typeID))
    if !ok {
        return zero, fmt.Errorf("cramberry: unknown type ID %d", typeID)
    }

    // Create instance and decode
    instance := reflect.New(typ).Interface()

    valueData := r.data[r.pos : r.pos+int(length)]
    r.pos += int(length)

    if u, ok := instance.(Unmarshaler); ok {
        if err := u.UnmarshalCramberry(valueData); err != nil {
            return zero, err
        }
    } else {
        subReader := NewReader(valueData)
        subReader.registry = r.registry
        if err := subReader.readReflect(reflect.ValueOf(instance).Elem()); err != nil {
            return zero, err
        }
    }

    // Type assertion
    result, ok := instance.(T)
    if !ok {
        return zero, fmt.Errorf("cramberry: type %T does not implement %T", instance, zero)
    }

    return result, nil
}

// SkipField skips a field based on wire type
func (r *Reader) SkipField(wireType WireType) error {
    switch wireType {
    case WireVarint, WireSVarint:
        _, err := r.ReadUVarint()
        return err
    case WireFixed32:
        if r.Remaining() < 4 {
            return io.ErrUnexpectedEOF
        }
        r.pos += 4
        return nil
    case WireFixed64:
        if r.Remaining() < 8 {
            return io.ErrUnexpectedEOF
        }
        r.pos += 8
        return nil
    case WireBytes, WireTypeRef:
        length, err := r.ReadUVarint()
        if err != nil {
            return err
        }
        if uint64(r.Remaining()) < length {
            return io.ErrUnexpectedEOF
        }
        r.pos += int(length)
        return nil
    default:
        return fmt.Errorf("cramberry: unknown wire type %d", wireType)
    }
}
```

### 11.5 Registry Implementation

```go
package cramberry

import (
    "reflect"
    "sync"
)

// Registry manages type registrations for polymorphic encoding
type Registry struct {
    mu sync.RWMutex

    // Forward mappings
    idToType map[TypeID]reflect.Type
    idToName map[TypeID]string

    // Reverse mappings
    typeToID map[reflect.Type]TypeID
    nameToID map[string]TypeID

    // Interface -> implementations
    interfaces map[reflect.Type][]TypeID

    // Custom codecs
    codecs map[reflect.Type]any

    // Next auto-assigned ID
    nextID TypeID
}

// DefaultRegistry is the global default registry
var DefaultRegistry = NewRegistry()

// NewRegistry creates a new type registry
func NewRegistry() *Registry {
    r := &Registry{
        idToType:   make(map[TypeID]reflect.Type),
        idToName:   make(map[TypeID]string),
        typeToID:   make(map[reflect.Type]TypeID),
        nameToID:   make(map[string]TypeID),
        interfaces: make(map[reflect.Type][]TypeID),
        codecs:     make(map[reflect.Type]any),
        nextID:     128, // User types start at 128
    }
    r.registerBuiltins()
    return r
}

// RegisterType registers a concrete type with auto-assigned ID
func RegisterType[T any](name string) TypeID {
    return DefaultRegistry.RegisterType(reflect.TypeOf((*T)(nil)).Elem(), name, 0)
}

// RegisterTypeWithID registers a concrete type with explicit ID
func RegisterTypeWithID[T any](name string, id TypeID) TypeID {
    return DefaultRegistry.RegisterType(reflect.TypeOf((*T)(nil)).Elem(), name, id)
}

func (r *Registry) RegisterType(typ reflect.Type, name string, id TypeID) TypeID {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Check for duplicate registration
    if existing, ok := r.typeToID[typ]; ok {
        return existing
    }

    // Auto-assign ID if not provided
    if id == 0 {
        id = r.nextID
        r.nextID++
    }

    // Register mappings
    r.idToType[id] = typ
    r.idToName[id] = name
    r.typeToID[typ] = id
    r.nameToID[name] = id

    return id
}

// RegisterInterface registers an interface type
func RegisterInterface[I any](name string) {
    DefaultRegistry.RegisterInterface(reflect.TypeOf((*I)(nil)).Elem(), name)
}

func (r *Registry) RegisterInterface(iface reflect.Type, name string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, ok := r.interfaces[iface]; !ok {
        r.interfaces[iface] = make([]TypeID, 0)
    }
}

// RegisterImplementation registers a concrete type as an interface implementation
func RegisterImplementation[I any, T any](id TypeID, name string) {
    iface := reflect.TypeOf((*I)(nil)).Elem()
    impl := reflect.TypeOf((*T)(nil)).Elem()
    DefaultRegistry.RegisterImplementation(iface, impl, id, name)
}

func (r *Registry) RegisterImplementation(iface, impl reflect.Type, id TypeID, name string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Register the implementation type
    typeID := r.RegisterType(impl, name, id)

    // Add to interface's implementations
    r.interfaces[iface] = append(r.interfaces[iface], typeID)
}

// RegisterCodec registers a custom codec for a type
func RegisterCodec[T any](codec Codec[T]) {
    typ := reflect.TypeOf((*T)(nil)).Elem()
    DefaultRegistry.codecs[typ] = codec
}

// TypeIDFor returns the TypeID for a value
func (r *Registry) TypeIDFor(v any) (TypeID, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    typ := reflect.TypeOf(v)
    if typ.Kind() == reflect.Ptr {
        typ = typ.Elem()
    }

    id, ok := r.typeToID[typ]
    return id, ok
}

// TypeForID returns the reflect.Type for a TypeID
func (r *Registry) TypeForID(id TypeID) (reflect.Type, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    typ, ok := r.idToType[id]
    return typ, ok
}

// NameForID returns the type name for a TypeID
func (r *Registry) NameForID(id TypeID) (string, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    name, ok := r.idToName[id]
    return name, ok
}
```

### 11.6 Top-Level API

```go
package cramberry

// Marshal encodes a value to bytes
func Marshal(v any) ([]byte, error) {
    // Fast path for Marshaler
    if m, ok := v.(Marshaler); ok {
        return m.MarshalCramberry()
    }

    // Fast path for Appender with Sizer
    if a, ok := v.(Appender); ok {
        var buf []byte
        if s, ok := v.(Sizer); ok {
            buf = make([]byte, 0, s.CramberrySize())
        }
        return a.AppendCramberry(buf)
    }

    // Reflection path
    w := NewWriter(256)
    if err := w.WriteValue(v); err != nil {
        return nil, err
    }
    return w.Bytes(), nil
}

// Unmarshal decodes bytes into a value
func Unmarshal(data []byte, v any) error {
    // Fast path for Unmarshaler
    if u, ok := v.(Unmarshaler); ok {
        return u.UnmarshalCramberry(data)
    }

    // Reflection path
    r := NewReader(data)
    rv := reflect.ValueOf(v)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return ErrNotPointer
    }
    return r.readReflect(rv.Elem())
}

// MustMarshal encodes a value, panicking on error
func MustMarshal(v any) []byte {
    data, err := Marshal(v)
    if err != nil {
        panic(err)
    }
    return data
}

// Clone creates a deep copy of a value via serialization
func Clone[T any](v T) (T, error) {
    data, err := Marshal(v)
    if err != nil {
        var zero T
        return zero, err
    }

    var result T
    if err := Unmarshal(data, &result); err != nil {
        var zero T
        return zero, err
    }
    return result, nil
}

// Equal compares two values for equality via serialization
func Equal(a, b any) bool {
    dataA, errA := Marshal(a)
    dataB, errB := Marshal(b)

    if errA != nil || errB != nil {
        return false
    }

    return bytes.Equal(dataA, dataB)
}

// Size returns the encoded size of a value
func Size(v any) int {
    if s, ok := v.(Sizer); ok {
        return s.CramberrySize()
    }

    // Fall back to encoding
    data, err := Marshal(v)
    if err != nil {
        return -1
    }
    return len(data)
}
```

---

## 12. Cross-Language Support

### 12.1 Runtime Libraries

Each target language requires a runtime library:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Language   │ Package                  │ Features                    │
├────────────┼──────────────────────────┼─────────────────────────────┤
│ Go         │ github.com/cramberry/go  │ Reference implementation    │
│ TypeScript │ @cramberry/runtime       │ Browser + Node.js           │
│ Rust       │ cramberry                │ no_std optional             │
│ Python     │ cramberry-py             │ Pure Python + Cython        │
│ C          │ libcramberry             │ For FFI bindings            │
└────────────┴──────────────────────────┴─────────────────────────────┘
```

### 12.2 TypeScript Runtime

```typescript
// @cramberry/runtime

export class Writer {
    private buffer: Uint8Array;
    private view: DataView;
    private pos: number;

    constructor(initialSize = 256) {
        this.buffer = new Uint8Array(initialSize);
        this.view = new DataView(this.buffer.buffer);
        this.pos = 0;
    }

    private grow(needed: number): void {
        if (this.pos + needed > this.buffer.length) {
            const newSize = Math.max(this.buffer.length * 2, this.pos + needed);
            const newBuffer = new Uint8Array(newSize);
            newBuffer.set(this.buffer);
            this.buffer = newBuffer;
            this.view = new DataView(this.buffer.buffer);
        }
    }

    uvarint(value: bigint | number): void {
        let v = BigInt(value);
        this.grow(10);
        while (v >= 0x80n) {
            this.buffer[this.pos++] = Number(v & 0x7fn) | 0x80;
            v >>= 7n;
        }
        this.buffer[this.pos++] = Number(v);
    }

    svarint(value: bigint | number): void {
        const v = BigInt(value);
        this.uvarint((v << 1n) ^ (v >> 63n)); // zigzag
    }

    fixed32(value: number): void {
        this.grow(4);
        this.view.setUint32(this.pos, value, true); // little-endian
        this.pos += 4;
    }

    fixed64(value: bigint): void {
        this.grow(8);
        this.view.setBigUint64(this.pos, value, true);
        this.pos += 8;
    }

    bytes(data: Uint8Array): void {
        this.uvarint(data.length);
        this.grow(data.length);
        this.buffer.set(data, this.pos);
        this.pos += data.length;
    }

    string(s: string): void {
        const encoded = new TextEncoder().encode(s);
        this.bytes(encoded);
    }

    tag(fieldNum: number, wireType: WireType): void {
        this.uvarint((fieldNum << 3) | wireType);
    }

    finish(): Uint8Array {
        return this.buffer.slice(0, this.pos);
    }
}

export class Reader {
    private data: Uint8Array;
    private view: DataView;
    private pos: number;

    constructor(data: Uint8Array) {
        this.data = data;
        this.view = new DataView(data.buffer, data.byteOffset, data.byteLength);
        this.pos = 0;
    }

    eof(): boolean {
        return this.pos >= this.data.length;
    }

    uvarint(): bigint {
        let result = 0n;
        let shift = 0n;

        while (true) {
            if (this.pos >= this.data.length) {
                throw new Error('Unexpected end of data');
            }

            const b = this.data[this.pos++];
            result |= BigInt(b & 0x7f) << shift;

            if ((b & 0x80) === 0) {
                return result;
            }

            shift += 7n;
            if (shift > 63n) {
                throw new Error('Varint too long');
            }
        }
    }

    svarint(): bigint {
        const v = this.uvarint();
        return (v >> 1n) ^ -(v & 1n); // un-zigzag
    }

    fixed32(): number {
        if (this.pos + 4 > this.data.length) {
            throw new Error('Unexpected end of data');
        }
        const value = this.view.getUint32(this.pos, true);
        this.pos += 4;
        return value;
    }

    fixed64(): bigint {
        if (this.pos + 8 > this.data.length) {
            throw new Error('Unexpected end of data');
        }
        const value = this.view.getBigUint64(this.pos, true);
        this.pos += 8;
        return value;
    }

    bytes(): Uint8Array {
        const length = Number(this.uvarint());
        if (this.pos + length > this.data.length) {
            throw new Error('Unexpected end of data');
        }
        const data = this.data.slice(this.pos, this.pos + length);
        this.pos += length;
        return data;
    }

    string(): string {
        return new TextDecoder().decode(this.bytes());
    }

    tag(): [number, WireType] {
        const v = Number(this.uvarint());
        return [v >> 3, v & 0x7];
    }

    skip(wireType: WireType): void {
        switch (wireType) {
            case WireType.Varint:
            case WireType.SVarint:
                this.uvarint();
                break;
            case WireType.Fixed32:
                this.pos += 4;
                break;
            case WireType.Fixed64:
                this.pos += 8;
                break;
            case WireType.Bytes:
            case WireType.TypeRef:
                const len = Number(this.uvarint());
                this.pos += len;
                break;
            default:
                throw new Error(`Unknown wire type: ${wireType}`);
        }
    }
}

export enum WireType {
    Varint = 0,
    Fixed64 = 1,
    Bytes = 2,
    Fixed32 = 5,
    SVarint = 6,
    TypeRef = 7,
}

export class Registry {
    private idToCodec = new Map<number, TypeCodec<any>>();
    private nameToId = new Map<string, number>();

    registerType<T>(
        id: number,
        name: string,
        codec: TypeCodec<T>
    ): void {
        this.idToCodec.set(id, codec);
        this.nameToId.set(name, id);
    }

    encode<T>(writer: Writer, value: T, typeName: string): void {
        const id = this.nameToId.get(typeName);
        if (id === undefined) {
            throw new Error(`Unknown type: ${typeName}`);
        }

        const codec = this.idToCodec.get(id)!;
        writer.uvarint(id);

        const tempWriter = new Writer();
        codec.encode(tempWriter, value);
        writer.bytes(tempWriter.finish());
    }

    decode<T>(reader: Reader): T {
        const id = Number(reader.uvarint());
        const codec = this.idToCodec.get(id);
        if (!codec) {
            throw new Error(`Unknown type ID: ${id}`);
        }

        const data = reader.bytes();
        const subReader = new Reader(data);
        return codec.decode(subReader);
    }
}

export interface TypeCodec<T> {
    encode(writer: Writer, value: T): void;
    decode(reader: Reader): T;
    size(value: T): number;
}
```

### 12.3 Rust Runtime

```rust
// cramberry crate

use std::io::{Read, Write, Result};

pub trait CramberryEncode {
    fn encode<W: Write>(&self, writer: &mut W) -> Result<()>;
    fn encoded_size(&self) -> usize;
}

pub trait CramberryDecode: Sized {
    fn decode<R: Read>(reader: &mut R) -> Result<Self>;
}

pub struct Writer<W: Write> {
    inner: W,
}

impl<W: Write> Writer<W> {
    pub fn new(inner: W) -> Self {
        Self { inner }
    }

    pub fn write_uvarint(&mut self, mut value: u64) -> Result<()> {
        loop {
            let mut byte = (value & 0x7F) as u8;
            value >>= 7;
            if value != 0 {
                byte |= 0x80;
            }
            self.inner.write_all(&[byte])?;
            if value == 0 {
                break;
            }
        }
        Ok(())
    }

    pub fn write_svarint(&mut self, value: i64) -> Result<()> {
        let zigzag = ((value << 1) ^ (value >> 63)) as u64;
        self.write_uvarint(zigzag)
    }

    pub fn write_fixed32(&mut self, value: u32) -> Result<()> {
        self.inner.write_all(&value.to_le_bytes())
    }

    pub fn write_fixed64(&mut self, value: u64) -> Result<()> {
        self.inner.write_all(&value.to_le_bytes())
    }

    pub fn write_bytes(&mut self, data: &[u8]) -> Result<()> {
        self.write_uvarint(data.len() as u64)?;
        self.inner.write_all(data)
    }

    pub fn write_string(&mut self, s: &str) -> Result<()> {
        self.write_bytes(s.as_bytes())
    }

    pub fn write_tag(&mut self, field_num: u32, wire_type: WireType) -> Result<()> {
        self.write_uvarint(((field_num as u64) << 3) | (wire_type as u64))
    }
}

pub struct Reader<R: Read> {
    inner: R,
}

impl<R: Read> Reader<R> {
    pub fn new(inner: R) -> Self {
        Self { inner }
    }

    pub fn read_uvarint(&mut self) -> Result<u64> {
        let mut result: u64 = 0;
        let mut shift = 0;

        loop {
            let mut byte = [0u8; 1];
            self.inner.read_exact(&mut byte)?;

            result |= ((byte[0] & 0x7F) as u64) << shift;

            if byte[0] & 0x80 == 0 {
                return Ok(result);
            }

            shift += 7;
            if shift > 63 {
                return Err(std::io::Error::new(
                    std::io::ErrorKind::InvalidData,
                    "varint too long"
                ));
            }
        }
    }

    pub fn read_svarint(&mut self) -> Result<i64> {
        let zigzag = self.read_uvarint()?;
        Ok(((zigzag >> 1) as i64) ^ (-((zigzag & 1) as i64)))
    }

    pub fn read_fixed32(&mut self) -> Result<u32> {
        let mut bytes = [0u8; 4];
        self.inner.read_exact(&mut bytes)?;
        Ok(u32::from_le_bytes(bytes))
    }

    pub fn read_fixed64(&mut self) -> Result<u64> {
        let mut bytes = [0u8; 8];
        self.inner.read_exact(&mut bytes)?;
        Ok(u64::from_le_bytes(bytes))
    }

    pub fn read_bytes(&mut self) -> Result<Vec<u8>> {
        let len = self.read_uvarint()? as usize;
        let mut data = vec![0u8; len];
        self.inner.read_exact(&mut data)?;
        Ok(data)
    }

    pub fn read_string(&mut self) -> Result<String> {
        let bytes = self.read_bytes()?;
        String::from_utf8(bytes).map_err(|e| {
            std::io::Error::new(std::io::ErrorKind::InvalidData, e)
        })
    }

    pub fn read_tag(&mut self) -> Result<(u32, WireType)> {
        let v = self.read_uvarint()?;
        let field_num = (v >> 3) as u32;
        let wire_type = WireType::try_from((v & 0x7) as u8)?;
        Ok((field_num, wire_type))
    }
}

#[repr(u8)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WireType {
    Varint = 0,
    Fixed64 = 1,
    Bytes = 2,
    Fixed32 = 5,
    SVarint = 6,
    TypeRef = 7,
}

impl TryFrom<u8> for WireType {
    type Error = std::io::Error;

    fn try_from(value: u8) -> std::result::Result<Self, Self::Error> {
        match value {
            0 => Ok(WireType::Varint),
            1 => Ok(WireType::Fixed64),
            2 => Ok(WireType::Bytes),
            5 => Ok(WireType::Fixed32),
            6 => Ok(WireType::SVarint),
            7 => Ok(WireType::TypeRef),
            _ => Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                format!("unknown wire type: {}", value)
            )),
        }
    }
}

// Registry for polymorphic types
pub struct Registry {
    encoders: std::collections::HashMap<std::any::TypeId, Box<dyn Fn(&dyn std::any::Any, &mut dyn Write) -> Result<()>>>,
    decoders: std::collections::HashMap<u32, Box<dyn Fn(&mut dyn Read) -> Result<Box<dyn std::any::Any>>>>,
    type_ids: std::collections::HashMap<std::any::TypeId, u32>,
}
```

---

## 13. Schema Generation Tool

### 13.1 Overview

The `cramberry-gen` tool extracts schema definitions from Go source code.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Schema Generation Pipeline                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Go Source Files (*.go)                                              │
│         │                                                            │
│         ▼                                                            │
│  ┌──────────────────┐                                               │
│  │  Go AST Parser   │  Uses go/ast, go/types                        │
│  └──────────────────┘                                               │
│         │                                                            │
│         ▼                                                            │
│  ┌──────────────────┐                                               │
│  │  Type Analyzer   │  Resolves interfaces, embedded types          │
│  └──────────────────┘                                               │
│         │                                                            │
│         ▼                                                            │
│  ┌──────────────────┐                                               │
│  │ Schema Generator │  Produces .cramberry files                    │
│  └──────────────────┘                                               │
│         │                                                            │
│         ▼                                                            │
│  .cramberry Schema Files                                             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 13.2 CLI Usage

```bash
# Generate schema from Go package
cramberry-gen schema \
    --package ./pkg/models \
    --output ./schema/models.cramberry \
    --include-private=false

# Generate from multiple packages
cramberry-gen schema \
    --package ./pkg/models \
    --package ./pkg/events \
    --output ./schema/ \
    --recursive

# With type filtering
cramberry-gen schema \
    --package ./pkg/models \
    --output ./schema/models.cramberry \
    --include "User,Order,*Message" \
    --exclude "*Internal,*test*"

# Generate with interface detection
cramberry-gen schema \
    --package ./pkg/models \
    --output ./schema/ \
    --detect-interfaces \
    --interface-prefix "I"
```

### 13.3 Struct Tag Support

The generator recognizes struct tags for schema customization:

```go
type User struct {
    // Field number override (default: auto-increment)
    ID int64 `cramberry:"1"`

    // Custom field name in schema
    EmailAddress string `cramberry:"2,name=email"`

    // Mark as optional (even if non-pointer)
    Nickname string `cramberry:"3,optional"`

    // Custom encoding
    Scores []int `cramberry:"4,encoding=delta"`

    // Skip field (don't include in schema)
    cache map[string]any `cramberry:"-"`

    // Deprecated field
    OldField string `cramberry:"5,deprecated=use NewField instead"`

    // Multiple tags
    Metadata map[string]string `cramberry:"6" json:"metadata,omitempty"`
}
```

### 13.4 Interface Detection

The generator can detect interface implementations:

```go
// Define interface
type Validator interface {
    Validate() error
}

// Implementations (detected automatically)
type EmailValidator struct { ... }
func (e *EmailValidator) Validate() error { ... }

type PhoneValidator struct { ... }
func (p *PhoneValidator) Validate() error { ... }

// Generated schema:
// interface Validator { validate(): error; }
// @implements(Validator) type EmailValidator { ... }
// @implements(Validator) type PhoneValidator { ... }
```

### 13.5 Type ID Assignment Strategy

```go
// Option 1: Explicit via struct tag
type User struct {
    _ struct{} `cramberry_type:"128"`
}

// Option 2: Explicit via comment directive
//cramberry:typeid=128
type User struct { ... }

// Option 3: Auto-assignment (deterministic from fully-qualified name)
// TypeID = hash("github.com/org/pkg.User") % 2^31 + 128

// Option 4: Configuration file
// cramberry.yaml:
// types:
//   User: 128
//   Order: 129
```

### 13.6 Generator Implementation

```go
package gen

import (
    "go/ast"
    "go/parser"
    "go/token"
    "go/types"
    "golang.org/x/tools/go/packages"
)

// SchemaGenerator extracts Cramberry schemas from Go source
type SchemaGenerator struct {
    packages   []*packages.Package
    typeInfo   map[types.Type]*TypeInfo
    interfaces map[string]*InterfaceInfo
    config     *Config
}

type Config struct {
    IncludePrivate    bool
    IncludePatterns   []string
    ExcludePatterns   []string
    DetectInterfaces  bool
    InterfacePrefix   string
    TypeIDStrategy    TypeIDStrategy
    TypeIDMap         map[string]TypeID
}

type TypeIDStrategy int

const (
    TypeIDExplicit TypeIDStrategy = iota  // Require explicit IDs
    TypeIDHash                             // Hash from name
    TypeIDAuto                             // Sequential assignment
)

type TypeInfo struct {
    Name        string
    Package     string
    GoType      types.Type
    Fields      []*FieldInfo
    Implements  []string  // Interface names
    TypeID      TypeID
    Annotations []string
}

type FieldInfo struct {
    Name       string
    FieldNum   int
    GoType     types.Type
    SchemaType string
    Optional   bool
    Encoding   string
    Deprecated string
}

type InterfaceInfo struct {
    Name           string
    Package        string
    Methods        []*MethodInfo
    Implementations []*TypeInfo
}

type MethodInfo struct {
    Name    string
    Params  []*ParamInfo
    Returns []*ParamInfo
}

// Generate produces schema files from Go packages
func (g *SchemaGenerator) Generate(pkgPaths []string, outputDir string) error {
    // Load packages with type information
    cfg := &packages.Config{
        Mode: packages.NeedName |
              packages.NeedFiles |
              packages.NeedSyntax |
              packages.NeedTypes |
              packages.NeedTypesInfo,
    }

    pkgs, err := packages.Load(cfg, pkgPaths...)
    if err != nil {
        return err
    }

    g.packages = pkgs

    // Phase 1: Collect all types
    for _, pkg := range pkgs {
        g.collectTypes(pkg)
    }

    // Phase 2: Detect interface implementations
    if g.config.DetectInterfaces {
        g.detectImplementations()
    }

    // Phase 3: Assign type IDs
    g.assignTypeIDs()

    // Phase 4: Generate schema files
    return g.writeSchemas(outputDir)
}

func (g *SchemaGenerator) collectTypes(pkg *packages.Package) {
    scope := pkg.Types.Scope()

    for _, name := range scope.Names() {
        obj := scope.Lookup(name)

        // Skip unexported if configured
        if !g.config.IncludePrivate && !obj.Exported() {
            continue
        }

        // Check include/exclude patterns
        if !g.matchesPatterns(name) {
            continue
        }

        switch t := obj.Type().Underlying().(type) {
        case *types.Struct:
            g.processStruct(pkg, name, t, obj)
        case *types.Interface:
            g.processInterface(pkg, name, t)
        }
    }
}

func (g *SchemaGenerator) processStruct(pkg *packages.Package, name string, st *types.Struct, obj types.Object) {
    info := &TypeInfo{
        Name:    name,
        Package: pkg.PkgPath,
        GoType:  obj.Type(),
        Fields:  make([]*FieldInfo, 0, st.NumFields()),
    }

    // Extract fields with tags
    for i := 0; i < st.NumFields(); i++ {
        field := st.Field(i)
        tag := st.Tag(i)

        fieldInfo := g.parseField(field, tag, i+1)
        if fieldInfo != nil {
            info.Fields = append(info.Fields, fieldInfo)
        }
    }

    g.typeInfo[obj.Type()] = info
}

func (g *SchemaGenerator) parseField(field *types.Var, tag string, defaultNum int) *FieldInfo {
    cramTag := reflect.StructTag(tag).Get("cramberry")

    // Skip field
    if cramTag == "-" {
        return nil
    }

    info := &FieldInfo{
        Name:       field.Name(),
        FieldNum:   defaultNum,
        GoType:     field.Type(),
        SchemaType: g.goTypeToSchema(field.Type()),
    }

    // Parse tag components
    if cramTag != "" {
        parts := strings.Split(cramTag, ",")

        // First part is field number if numeric
        if num, err := strconv.Atoi(parts[0]); err == nil {
            info.FieldNum = num
            parts = parts[1:]
        }

        // Parse key=value pairs
        for _, part := range parts {
            if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
                switch kv[0] {
                case "name":
                    info.Name = kv[1]
                case "encoding":
                    info.Encoding = kv[1]
                case "deprecated":
                    info.Deprecated = kv[1]
                }
            } else if part == "optional" {
                info.Optional = true
            }
        }
    }

    return info
}

func (g *SchemaGenerator) goTypeToSchema(t types.Type) string {
    switch t := t.(type) {
    case *types.Basic:
        switch t.Kind() {
        case types.Bool:
            return "bool"
        case types.Int8:
            return "int8"
        case types.Int16:
            return "int16"
        case types.Int32:
            return "int32"
        case types.Int64, types.Int:
            return "int64"
        case types.Uint8:
            return "uint8"
        case types.Uint16:
            return "uint16"
        case types.Uint32:
            return "uint32"
        case types.Uint64, types.Uint:
            return "uint64"
        case types.Float32:
            return "float32"
        case types.Float64:
            return "float64"
        case types.String:
            return "string"
        }
    case *types.Slice:
        if basic, ok := t.Elem().(*types.Basic); ok && basic.Kind() == types.Uint8 {
            return "bytes"
        }
        return "[]" + g.goTypeToSchema(t.Elem())
    case *types.Array:
        return fmt.Sprintf("[%d]%s", t.Len(), g.goTypeToSchema(t.Elem()))
    case *types.Map:
        return fmt.Sprintf("map<%s, %s>", g.goTypeToSchema(t.Key()), g.goTypeToSchema(t.Elem()))
    case *types.Pointer:
        return g.goTypeToSchema(t.Elem()) + "?"
    case *types.Named:
        return t.Obj().Name()
    case *types.Interface:
        if t.Empty() {
            return "any"
        }
        // Named interface
        return t.String()
    }
    return "any"
}

func (g *SchemaGenerator) detectImplementations() {
    for _, info := range g.typeInfo {
        for ifaceName, iface := range g.interfaces {
            if types.Implements(info.GoType, iface.GoType.(*types.Interface)) ||
               types.Implements(types.NewPointer(info.GoType), iface.GoType.(*types.Interface)) {
                info.Implements = append(info.Implements, ifaceName)
                iface.Implementations = append(iface.Implementations, info)
            }
        }
    }
}

func (g *SchemaGenerator) writeSchemas(outputDir string) error {
    // Group by package
    byPackage := make(map[string][]*TypeInfo)
    for _, info := range g.typeInfo {
        byPackage[info.Package] = append(byPackage[info.Package], info)
    }

    for pkg, types := range byPackage {
        schema := g.buildSchema(pkg, types)
        filename := filepath.Join(outputDir, g.packageToFilename(pkg))
        if err := os.WriteFile(filename, []byte(schema), 0644); err != nil {
            return err
        }
    }

    return nil
}

func (g *SchemaGenerator) buildSchema(pkg string, types []*TypeInfo) string {
    var buf strings.Builder

    // Package declaration
    buf.WriteString(fmt.Sprintf("package %s;\n\n", g.toSchemaPackage(pkg)))

    // Interfaces
    for _, iface := range g.interfaces {
        if iface.Package == pkg {
            g.writeInterface(&buf, iface)
        }
    }

    // Types
    for _, t := range types {
        g.writeType(&buf, t)
    }

    return buf.String()
}

func (g *SchemaGenerator) writeType(buf *strings.Builder, info *TypeInfo) {
    // Annotations
    for _, impl := range info.Implements {
        buf.WriteString(fmt.Sprintf("@implements(%s)\n", impl))
    }
    buf.WriteString(fmt.Sprintf("@typeid(%d)\n", info.TypeID))

    // Type declaration
    buf.WriteString(fmt.Sprintf("type %s {\n", info.Name))

    // Fields
    for _, field := range info.Fields {
        g.writeField(buf, field)
    }

    buf.WriteString("}\n\n")
}

func (g *SchemaGenerator) writeField(buf *strings.Builder, field *FieldInfo) {
    // Field annotations
    if field.Encoding != "" {
        buf.WriteString(fmt.Sprintf("    @encoding(\"%s\")\n", field.Encoding))
    }
    if field.Deprecated != "" {
        buf.WriteString(fmt.Sprintf("    @deprecated(\"%s\")\n", field.Deprecated))
    }

    // Field declaration
    optional := ""
    if field.Optional {
        optional = "?"
    }

    buf.WriteString(fmt.Sprintf("    @field(%d) %s: %s%s;\n",
        field.FieldNum, field.Name, field.SchemaType, optional))
}
```

### 13.7 Configuration File

Support for `cramberry.yaml` configuration:

```yaml
# cramberry.yaml
version: "1.0"

# Package configuration
packages:
  - path: "./pkg/models"
    output: "./schema/models.cramberry"
    include:
      - "User"
      - "Order"
      - "*Message"
    exclude:
      - "*Internal"
      - "*Test"

  - path: "./pkg/events"
    output: "./schema/events.cramberry"
    recursive: true

# Global settings
settings:
  include_private: false
  detect_interfaces: true
  interface_prefix: ""

# Type ID assignments
type_ids:
  User: 128
  Order: 129
  CreateOrderEvent: 200
  UpdateOrderEvent: 201

# Custom type mappings
type_mappings:
  "time.Time": "cramberry.stdlib.Timestamp"
  "github.com/google/uuid.UUID": "bytes"
  "math/big.Int": "bytes"

# Code generation options
codegen:
  go:
    package_prefix: "github.com/myorg/myapp/gen"
    with_json: true
    with_validation: true
  typescript:
    output: "./web/src/gen"
    module_type: "esm"
  rust:
    output: "./rust/src/gen"
```

---

## 14. Performance Considerations

### 14.1 Encoding Performance

```
┌─────────────────────────────────────────────────────────────────────┐
│ Optimization                      │ Impact         │ Trade-off      │
├───────────────────────────────────┼────────────────┼────────────────┤
│ Pre-computed field encoders       │ 30-50% faster  │ Init time      │
│ Buffer pooling                    │ 20-40% faster  │ Memory         │
│ Unsafe string/[]byte conversion   │ 10-20% faster  │ Safety         │
│ Cached size calculations          │ 15-25% faster  │ Accuracy       │
│ Generated code vs reflection      │ 3-10x faster   │ Code size      │
│ Inline varint encoding            │ 5-15% faster   │ Code size      │
└───────────────────────────────────┴────────────────┴────────────────┘
```

### 14.2 Decoding Performance

```
┌─────────────────────────────────────────────────────────────────────┐
│ Optimization                      │ Impact         │ Trade-off      │
├───────────────────────────────────┼────────────────┼────────────────┤
│ Zero-allocation decoding          │ 40-60% faster  │ API complexity │
│ Field lookup table                │ 20-30% faster  │ Memory         │
│ Bounds check elimination          │ 5-10% faster   │ Safety         │
│ Parallel field decoding           │ Variable       │ Complexity     │
│ Memory-mapped file support        │ Large files    │ Portability    │
└───────────────────────────────────┴────────────────┴────────────────┘
```

### 14.3 Memory Management

```go
// Buffer pool for encoding
var bufferPool = sync.Pool{
    New: func() any {
        return make([]byte, 0, 4096)
    },
}

// GetBuffer retrieves a buffer from the pool
func GetBuffer() []byte {
    return bufferPool.Get().([]byte)[:0]
}

// PutBuffer returns a buffer to the pool
func PutBuffer(buf []byte) {
    if cap(buf) <= 64*1024 { // Don't pool huge buffers
        bufferPool.Put(buf[:0])
    }
}

// Usage
func MarshalPooled(v any) ([]byte, error) {
    buf := GetBuffer()
    defer PutBuffer(buf)

    result, err := marshalAppend(buf, v)
    if err != nil {
        return nil, err
    }

    // Copy to new slice (caller owns the result)
    out := make([]byte, len(result))
    copy(out, result)
    return out, nil
}
```

### 14.4 Benchmark Targets

```
┌─────────────────────────────────────────────────────────────────────┐
│ Operation              │ Target Throughput    │ Comparison          │
├────────────────────────┼──────────────────────┼─────────────────────┤
│ Small message encode   │ > 5M msgs/sec        │ 2x protobuf         │
│ Small message decode   │ > 4M msgs/sec        │ 1.5x protobuf       │
│ Large message encode   │ > 2 GB/sec           │ Match protobuf      │
│ Large message decode   │ > 1.5 GB/sec         │ Match protobuf      │
│ Streaming throughput   │ > 1 GB/sec           │ N/A                 │
│ Size overhead          │ < 5% vs protobuf     │ Comparable          │
└────────────────────────┴──────────────────────┴─────────────────────┘
```

### 14.5 Profiling Integration

```go
// Built-in profiling hooks
type ProfileHooks struct {
    OnEncode func(typeName string, size int, duration time.Duration)
    OnDecode func(typeName string, size int, duration time.Duration)
    OnAlloc  func(size int)
}

// Enable profiling
cramberry.SetProfileHooks(&cramberry.ProfileHooks{
    OnEncode: func(name string, size int, dur time.Duration) {
        metrics.RecordEncode(name, size, dur)
    },
})
```

---

## 15. Security Considerations

### 15.1 Input Validation

```
┌─────────────────────────────────────────────────────────────────────┐
│ Threat                        │ Mitigation                          │
├───────────────────────────────┼─────────────────────────────────────┤
│ Excessive message size        │ Configurable max message size       │
│ Deep nesting (stack overflow) │ Max nesting depth limit             │
│ Invalid UTF-8 strings         │ UTF-8 validation (optional)         │
│ Integer overflow              │ Bounds checking on varints          │
│ Unknown type IDs              │ Strict mode rejects, lenient skips  │
│ Malformed varints             │ Max 10 bytes for varint64           │
│ Denial of service             │ Timeout and resource limits         │
└───────────────────────────────┴─────────────────────────────────────┘
```

### 15.2 Configuration

```go
type DecoderConfig struct {
    // Maximum message size (0 = no limit)
    MaxMessageSize int64

    // Maximum nesting depth (0 = no limit)
    MaxDepth int

    // Maximum string length (0 = no limit)
    MaxStringLength int

    // Maximum array/slice length (0 = no limit)
    MaxArrayLength int

    // Validate UTF-8 strings
    ValidateUTF8 bool

    // Reject unknown fields
    StrictMode bool

    // Timeout for decode operations
    Timeout time.Duration
}

// Secure defaults
var SecureDefaults = DecoderConfig{
    MaxMessageSize:  64 * 1024 * 1024,  // 64 MB
    MaxDepth:        100,
    MaxStringLength: 10 * 1024 * 1024,   // 10 MB
    MaxArrayLength:  1_000_000,
    ValidateUTF8:    true,
    StrictMode:      false,
    Timeout:         30 * time.Second,
}
```

### 15.3 Type Safety

```go
// Type-safe decode prevents type confusion attacks
func UnmarshalStrict[T any](data []byte) (T, error) {
    var result T

    // Verify type ID matches expected type
    r := NewReader(data)
    typeID, _ := r.PeekTypeID()

    expectedID := DefaultRegistry.TypeIDFor(result)
    if typeID != expectedID {
        return result, fmt.Errorf("type mismatch: expected %d, got %d", expectedID, typeID)
    }

    err := Unmarshal(data, &result)
    return result, err
}
```

---

## 16. Appendices

### 16.1 Comparison with Existing Formats

```
┌────────────────────────────────────────────────────────────────────────────┐
│ Feature             │Cramberry│Protobuf│Amino   │MsgPack │CBOR    │JSON   │
├─────────────────────┼─────────┼────────┼────────┼────────┼────────┼───────┤
│ Binary format       │ ✓       │ ✓      │ ✓      │ ✓      │ ✓      │ ✗     │
│ Schema required     │ ✓       │ ✓      │ ✗      │ ✗      │ ✗      │ ✗     │
│ Deterministic       │ ✓       │ ◐      │ ✓      │ ◐      │ ◐      │ ✗     │
│ Polymorphism        │ ✓       │ ◐      │ ✓      │ ◐      │ ✓      │ ◐     │
│ Streaming           │ ✓       │ ✓      │ ✗      │ ✓      │ ✓      │ ◐     │
│ Cross-language      │ ✓       │ ✓      │ ✗      │ ✓      │ ✓      │ ✓     │
│ Go reflection       │ ✓       │ ✓      │ ✓      │ ✓      │ ✓      │ ✓     │
│ Code generation     │ ✓       │ ✓      │ ✗      │ ✗      │ ✗      │ ✗     │
│ Schema evolution    │ ✓       │ ✓      │ ◐      │ ◐      │ ◐      │ ◐     │
│ Self-describing     │ ✗       │ ✗      │ ✗      │ ✓      │ ✓      │ ✓     │
└─────────────────────┴─────────┴────────┴────────┴────────┴────────┴───────┘
  ✓ = Full support, ◐ = Partial support, ✗ = Not supported
```

### 16.2 Wire Format Quick Reference

```
Varint:         [0-9 bytes] MSB continuation bit
Fixed32:        [4 bytes] little-endian
Fixed64:        [8 bytes] little-endian
Bytes:          [length: varint] [data]
Tag:            (field_num << 3) | wire_type

Wire Types:
  0 = Varint (unsigned)
  1 = Fixed64
  2 = Bytes (length-prefixed)
  5 = Fixed32
  6 = SVarint (zigzag)
  7 = TypeRef (polymorphic)
```

### 16.3 Schema Language Quick Reference

```cramberry
package foo.bar;
import "other.cramberry";

@doc("Description")
enum Status { A = 0; B = 1; }

@typeprefix("foo")
interface Handler { handle(msg: Message): Result; }

@implements(Handler)
@typeid(128)
type MyHandler {
    @field(1) name: string;
    @field(2) @optional data: bytes?;
    @field(3) @encoding("delta") values: []int64;
}

type Generic<T> {
    @field(1) value: T;
}

service MyService {
    rpc Process(Request): Response;
    rpc Stream(Request): stream Response;
}
```

### 16.4 Migration Guide from Amino

```go
// Amino registration
cdc := amino.NewCodec()
cdc.RegisterInterface((*Message)(nil), nil)
cdc.RegisterConcrete(&TextMsg{}, "app/TextMsg", nil)

// Cramberry equivalent
registry := cramberry.NewRegistry()
cramberry.RegisterInterface[Message]("app.Message")
cramberry.RegisterImplementation[Message, TextMsg](128, "app.TextMsg")

// Amino encode/decode
data, _ := cdc.MarshalBinaryBare(msg)
cdc.UnmarshalBinaryBare(data, &msg)

// Cramberry equivalent
data, _ := cramberry.Marshal(msg)
cramberry.Unmarshal(data, &msg)
```

### 16.5 Glossary

| Term | Definition |
|------|------------|
| **Wire Type** | Encoding category (varint, fixed, bytes) |
| **Field Number** | Unique identifier for struct field |
| **Tag** | Encoded field number + wire type |
| **Varint** | Variable-length integer encoding |
| **ZigZag** | Signed integer encoding for varints |
| **Type ID** | Unique identifier for polymorphic types |
| **SDL** | Schema Definition Language |
| **Presence Bitmap** | Bit field tracking present fields |

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0-draft | 2026-01-21 | Engineering | Initial draft |

---

*End of Architecture Design Document*
