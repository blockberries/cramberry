# Cramberry

A high-performance, compact binary serialization format for Go with code generation support for Go, TypeScript, and Rust.

## Features

- **Compact**: 37-65% smaller than JSON encoding
- **Fast**: 2.7-3x faster deserialization than JSON
- **Deterministic**: Maps are sorted for reproducible encoding
- **Type-safe**: Strong type system with schema validation
- **Polymorphic**: Amino-style interface serialization with type registry
- **Streaming**: Efficient streaming encoder/decoder for large data
- **Multi-language**: Code generation for Go, TypeScript, and Rust

## Installation

```bash
go get github.com/blockberries/cramberry
```

## Quick Start

### Basic Usage

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

### Struct Tags

Cramberry uses struct tags to specify field numbers and options:

```go
type Message struct {
    ID        int64  `cramberry:"1,required"` // Field 1, required
    Content   string `cramberry:"2"`          // Field 2
    Timestamp int64  `cramberry:"3,omitempty"` // Field 3, omit if zero
    Internal  string `cramberry:"-"`          // Skip this field
}
```

### Polymorphic Types

Cramberry supports polymorphic serialization through a type registry:

```go
// Define interface and implementations
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

// Register types
cramberry.MustRegister[Dog]()
cramberry.MustRegister[Cat]()

// Use in a container struct
type Zoo struct {
    Animals []Animal `cramberry:"1"`
}
```

### Streaming

For large data or multiple messages:

```go
// Writing multiple messages
sw := cramberry.NewStreamWriter(conn)
for _, msg := range messages {
    if err := sw.WriteDelimited(&msg); err != nil {
        return err
    }
}
sw.Flush()

// Reading multiple messages
it := cramberry.NewMessageIterator(conn)
var msg MyMessage
for it.Next(&msg) {
    process(msg)
}
if err := it.Err(); err != nil {
    return err
}
```

## Schema Language

Cramberry includes a schema definition language for defining message types:

```cramberry
// user.cramberry
package example;

enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    SUSPENDED = 2;
}

message Address {
    street: string = 1;
    city: string = 2;
    country: string = 3;
}

message User {
    id: int64 = 1 [required];
    name: string = 2;
    email: string = 3;
    status: Status = 4;
    address: *Address = 5;  // optional pointer
    tags: []string = 6;     // repeated
}

interface Person {
    User = 128;
    Admin = 129;
}
```

### Code Generation

Generate code from schema files:

```bash
# Generate Go code
cramberry generate -lang go -out ./gen ./schemas/*.cramberry

# Generate TypeScript code
cramberry generate -lang typescript -out ./gen ./schemas/*.cramberry

# Generate Rust code
cramberry generate -lang rust -out ./gen ./schemas/*.cramberry
```

CLI options:
- `-lang`: Target language (go, typescript, rust)
- `-out`: Output directory
- `-package`: Override package name
- `-json`: Generate JSON serialization support (default: true)
- `-marshal`: Generate marshal/unmarshal methods (default: true)
- `-prefix`: Add prefix to type names
- `-suffix`: Add suffix to type names
- `-I`: Add import search path

## Wire Format

### Encoding Types

| Wire Type | Value | Description |
|-----------|-------|-------------|
| Varint    | 0     | Variable-length unsigned integer |
| Fixed64   | 1     | Fixed 64-bit value |
| Bytes     | 2     | Length-prefixed bytes |
| Fixed32   | 5     | Fixed 32-bit value |
| SVarint   | 6     | ZigZag-encoded signed integer |
| TypeRef   | 7     | Type ID for polymorphic values |

### Type Mappings

| Go Type | Wire Type | Notes |
|---------|-----------|-------|
| bool | Varint | 0 or 1 |
| int8, uint8 | Fixed byte | Single byte |
| int16, int32, int64 | SVarint | ZigZag encoded |
| uint16, uint32, uint64 | Varint | LEB128 encoded |
| float32 | Fixed32 | IEEE 754 |
| float64 | Fixed64 | IEEE 754 |
| string | Bytes | Length-prefixed UTF-8 |
| []byte | Bytes | Length-prefixed |
| slice | Bytes | Length + elements |
| map | Bytes | Size + sorted key-value pairs |
| struct | Bytes | Field count + tagged fields |
| interface | TypeRef | Type ID + value |

## Performance

Benchmarks on Apple M4 Pro:

### vs JSON

| Operation | Cramberry | JSON | Speedup |
|-----------|-----------|------|---------|
| Marshal Small | 86ns | 83ns | ~1x |
| Marshal Large | 912ns | 836ns | ~1x |
| Unmarshal Small | 98ns | 289ns | **3x** |
| Unmarshal Large | 1379ns | 3704ns | **2.7x** |

### Size Comparison

| Data Type | Cramberry | JSON | Reduction |
|-----------|-----------|------|-----------|
| Small struct | 14 bytes | 28 bytes | **50%** |
| Medium struct | 63 bytes | 118 bytes | **47%** |
| Large struct | 309 bytes | 490 bytes | **37%** |
| Nested struct | 29 bytes | 83 bytes | **65%** |

## API Reference

### Core Functions

```go
// Marshal encodes a Go value to binary
func Marshal(v any) ([]byte, error)

// Unmarshal decodes binary data to a Go value
func Unmarshal(data []byte, v any) error

// Size returns the encoded size without allocating
func Size(v any) int

// MarshalAppend appends encoded data to existing buffer
func MarshalAppend(buf []byte, v any) ([]byte, error)
```

### Type Registry

```go
// Register a type for polymorphic encoding
func Register[T any]() error
func MustRegister[T any]()

// Register with specific type ID
func RegisterWithID[T any](id TypeID) error

// Lookup types
func (r *Registry) TypeIDFor(v any) TypeID
func (r *Registry) TypeFor(id TypeID) (reflect.Type, bool)
```

### Writer/Reader

```go
// Low-level encoding
w := cramberry.NewWriter()
w.WriteInt32(42)
w.WriteString("hello")
data := w.Bytes()

// Low-level decoding
r := cramberry.NewReader(data)
num := r.ReadInt32()
str := r.ReadString()
```

### Streaming

```go
// Stream writer
sw := cramberry.NewStreamWriter(w)
sw.WriteDelimited(&msg)
sw.Flush()

// Stream reader
sr := cramberry.NewStreamReader(r)
sr.ReadDelimited(&msg)

// Iterator pattern
it := cramberry.NewMessageIterator(r)
for it.Next(&msg) { ... }
```

## Cross-Language Support

Cramberry provides runtime libraries for multiple languages:

### TypeScript

```typescript
import { Writer, Reader, WireType } from '@cramberry/runtime';

// Encoding
const writer = new Writer();
writer.writeInt32Field(1, 42);
writer.writeStringField(2, "hello");
const data = writer.bytes();

// Decoding
const reader = new Reader(data);
while (reader.hasMore) {
  const { fieldNumber, wireType } = reader.readTag();
  switch (fieldNumber) {
    case 1: console.log(reader.readInt32()); break;
    case 2: console.log(reader.readString()); break;
    default: reader.skipField(wireType);
  }
}
```

### Rust

```rust
use cramberry::{Writer, Reader, WireType, Result};

fn main() -> Result<()> {
    // Encoding
    let mut writer = Writer::new();
    writer.write_int32_field(1, 42)?;
    writer.write_string_field(2, "hello")?;
    let data = writer.into_bytes();

    // Decoding
    let mut reader = Reader::new(&data);
    while reader.has_more() {
        let tag = reader.read_tag()?;
        match tag.field_number {
            1 => println!("{}", reader.read_int32()?),
            2 => println!("{}", reader.read_string()?),
            _ => reader.skip_field(tag.wire_type)?,
        }
    }
    Ok(())
}
```

### Schema Extraction

Extract Cramberry schemas from existing Go code:

```bash
# Extract schema from Go packages
cramberry schema ./pkg/models -out schema.cramberry

# Include unexported types
cramberry schema -private ./...

# Filter by type name patterns
cramberry schema -include "User*" -exclude "*Internal" ./...
```

## Documentation

- [Architecture Design](ARCHITECTURE.md)
- [Implementation Plan](IMPLEMENTATION_PLAN.md)

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
