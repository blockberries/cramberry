# Migration Guide

This guide helps you migrate from other serialization formats to Cramberry.

## Table of Contents

- [From JSON](#from-json)
- [From Protocol Buffers](#from-protocol-buffers)
- [From Amino](#from-amino)
- [From MessagePack](#from-messagepack)
- [Wire Format Migration (V1 to V2)](#wire-format-migration-v1-to-v2)

## From JSON

### Basic Migration

**JSON (before):**
```go
type User struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email,omitempty"`
}

data, err := json.Marshal(user)
var decoded User
err = json.Unmarshal(data, &decoded)
```

**Cramberry (after):**
```go
type User struct {
    ID    int64  `cramberry:"1"`
    Name  string `cramberry:"2"`
    Email string `cramberry:"3,omitempty"`
}

data, err := cramberry.Marshal(user)
var decoded User
err = cramberry.Unmarshal(data, &decoded)
```

### Key Differences

| Feature | JSON | Cramberry |
|---------|------|-----------|
| Field identification | String names | Numeric field numbers |
| Format | Text | Binary |
| Size | Larger | 2-3x smaller |
| Speed | Slower | 6-27x faster decode |
| Human readable | Yes | No |
| Schema evolution | Rename fields freely | Field numbers are stable |

### Handling Optional Fields

**JSON:**
```go
type Message struct {
    Content  string  `json:"content"`
    Priority *int    `json:"priority,omitempty"` // nil = absent
}
```

**Cramberry:**
```go
type Message struct {
    Content  string `cramberry:"1"`
    Priority *int   `cramberry:"2,omitempty"` // nil pointers encoded as absent
}
```

### Handling Maps

Both JSON and Cramberry support maps, but Cramberry requires primitive key types:

```go
// Both work
type Data struct {
    Metadata map[string]string `json:"metadata" cramberry:"1"`
}

// JSON allows complex keys, Cramberry doesn't
type BadForCramberry struct {
    Data map[MyStruct]string // Works in JSON, not in Cramberry
}
```

### Dual-Format Support

During migration, you can support both formats:

```go
type User struct {
    ID    int64  `json:"id" cramberry:"1"`
    Name  string `json:"name" cramberry:"2"`
    Email string `json:"email,omitempty" cramberry:"3,omitempty"`
}

// Use JSON for external APIs
jsonData, _ := json.Marshal(user)

// Use Cramberry for internal storage/communication
cramData, _ := cramberry.Marshal(user)
```

## From Protocol Buffers

### Basic Migration

**Protocol Buffers (before):**
```protobuf
message User {
  int64 id = 1;
  string name = 2;
  string email = 3;
}
```

```go
user := &pb.User{Id: 1, Name: "Alice", Email: "alice@example.com"}
data, err := proto.Marshal(user)
```

**Cramberry (after):**
```go
type User struct {
    ID    int64  `cramberry:"1"`
    Name  string `cramberry:"2"`
    Email string `cramberry:"3"`
}

user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}
data, err := cramberry.Marshal(user)
```

### Schema Migration

**Protobuf schema:**
```protobuf
syntax = "proto3";
package example;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  SUSPENDED = 2;
}

message User {
  int64 id = 1;
  string name = 2;
  Status status = 3;
  repeated string tags = 4;
  map<string, string> metadata = 5;
}
```

**Cramberry schema:**
```cramberry
package example;

enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    SUSPENDED = 2;
}

message User {
    id: int64 = 1;
    name: string = 2;
    status: Status = 3;
    tags: []string = 4;
    metadata: map[string]string = 5;
}
```

### Type Mappings

| Protobuf | Cramberry |
|----------|-----------|
| `int32` | `int32` |
| `int64` | `int64` |
| `uint32` | `uint32` |
| `uint64` | `uint64` |
| `sint32` | `int32` (auto ZigZag) |
| `sint64` | `int64` (auto ZigZag) |
| `fixed32` | `uint32` |
| `fixed64` | `uint64` |
| `sfixed32` | `int32` |
| `sfixed64` | `int64` |
| `float` | `float32` |
| `double` | `float64` |
| `bool` | `bool` |
| `string` | `string` |
| `bytes` | `bytes` or `[]byte` |
| `repeated T` | `[]T` |
| `map<K,V>` | `map[K]V` |
| `message` | `struct` |
| `enum` | `enum` |
| `oneof` | `interface` |

### Key Differences

| Feature | Protocol Buffers | Cramberry |
|---------|-----------------|-----------|
| Code generation | Required | Optional (reflection works) |
| Wire format | Similar | Compatible concepts |
| Signed integers | Explicit sint* types | Auto ZigZag for int* |
| Polymorphism | oneof (compile-time) | interface (runtime registry) |
| Default values | Proto3: always zero | Configurable omitempty |

### Field Number Compatibility

If migrating data, keep the same field numbers:

```go
// Protobuf had: int64 id = 1;
type User struct {
    ID int64 `cramberry:"1"` // Keep field number 1
}
```

## From Amino

Cramberry is designed as a spiritual successor to Amino with improved performance.

### Basic Migration

**Amino (before):**
```go
type User struct {
    ID   int64  `amino:"id"`
    Name string `amino:"name"`
}

cdc := amino.NewCodec()
cdc.RegisterConcrete(User{}, "example/User", nil)
data, err := cdc.MarshalBinaryBare(user)
```

**Cramberry (after):**
```go
type User struct {
    ID   int64  `cramberry:"1"`
    Name string `cramberry:"2"`
}

// No codec initialization needed for basic types
data, err := cramberry.Marshal(user)
```

### Interface Registration

**Amino:**
```go
type Animal interface { Speak() string }
type Dog struct { Name string }
type Cat struct { Name string }

cdc := amino.NewCodec()
cdc.RegisterInterface((*Animal)(nil), nil)
cdc.RegisterConcrete(&Dog{}, "example/Dog", nil)
cdc.RegisterConcrete(&Cat{}, "example/Cat", nil)
```

**Cramberry:**
```go
type Animal interface { Speak() string }
type Dog struct { Name string `cramberry:"1"` }
type Cat struct { Name string `cramberry:"1"` }

func init() {
    cramberry.MustRegister[Dog]()  // Auto TypeID: 128
    cramberry.MustRegister[Cat]()  // Auto TypeID: 129
}
```

### Key Differences

| Feature | Amino | Cramberry |
|---------|-------|-----------|
| Field tags | `amino:"name"` | `cramberry:"num"` |
| Codec setup | Required | Optional |
| Type registration | String names | Numeric TypeIDs |
| Determinism | Always | Configurable |
| Performance | Baseline | 1.5-2.6x faster decode |

### TypeID Migration

If you need stable TypeIDs matching Amino prefixes:

```go
// Calculate or define stable TypeIDs
const (
    TypeIDDog TypeID = 128
    TypeIDCat TypeID = 129
)

cramberry.MustRegisterWithID[Dog](TypeIDDog)
cramberry.MustRegisterWithID[Cat](TypeIDCat)
```

## From MessagePack

### Basic Migration

**MessagePack (before):**
```go
import "github.com/vmihailenco/msgpack/v5"

type User struct {
    ID   int64  `msgpack:"id"`
    Name string `msgpack:"name"`
}

data, err := msgpack.Marshal(user)
```

**Cramberry (after):**
```go
type User struct {
    ID   int64  `cramberry:"1"`
    Name string `cramberry:"2"`
}

data, err := cramberry.Marshal(user)
```

### Key Differences

| Feature | MessagePack | Cramberry |
|---------|------------|-----------|
| Field identification | String or positional | Field numbers |
| Schema evolution | Fragile | Field numbers are stable |
| Determinism | Implementation-dependent | Guaranteed (maps sorted) |
| Streaming | Supported | Supported |

## Wire Format Migration (V1 to V2)

If you have data encoded with V1 wire format, you can migrate to V2.

### Reading V1 Data

```go
// Decode V1 data
var msg Message
err := cramberry.UnmarshalWithOptions(v1Data, &msg, cramberry.V1Options)

// Re-encode as V2
v2Data, err := cramberry.Marshal(msg)  // Default is V2
```

### Dual-Version Support

During migration, you may need to support both:

```go
func decode(data []byte) (*Message, error) {
    var msg Message

    // Try V2 first (more common)
    err := cramberry.Unmarshal(data, &msg)
    if err == nil {
        return &msg, nil
    }

    // Fall back to V1
    err = cramberry.UnmarshalWithOptions(data, &msg, cramberry.V1Options)
    return &msg, err
}
```

### V1 vs V2 Wire Format

| Feature | V1 | V2 |
|---------|----|----|
| Field tags | Full varint | Compact (1 byte for fields 1-15) |
| Message end | Field count prefix | End marker (0x00) |
| Packed arrays | No | Yes |
| Size | Baseline | ~5% smaller |

## General Migration Tips

### 1. Add Cramberry Tags Alongside Existing

```go
type User struct {
    ID    int64  `json:"id" cramberry:"1"`
    Name  string `json:"name" cramberry:"2"`
    Email string `json:"email,omitempty" cramberry:"3,omitempty"`
}
```

### 2. Keep Field Numbers Stable

Once assigned, never change field numbers:

```go
// Version 1
type User struct {
    ID   int64  `cramberry:"1"`
    Name string `cramberry:"2"`
}

// Version 2 - ADD new fields, don't renumber
type User struct {
    ID    int64  `cramberry:"1"`  // Keep 1
    Name  string `cramberry:"2"`  // Keep 2
    Email string `cramberry:"3"`  // New field gets 3
}
```

### 3. Use Schema Extraction

Generate Cramberry schema from existing Go code:

```bash
cramberry schema ./pkg/models -out schema.cram
```

### 4. Run Parallel Systems

During migration:
1. Encode with both formats
2. Verify data equivalence
3. Monitor performance
4. Gradually switch to Cramberry-only

### 5. Test Thoroughly

```go
func TestMigration(t *testing.T) {
    original := User{ID: 1, Name: "Alice"}

    // Encode with old format
    oldData, _ := oldEncoder.Marshal(original)

    // Decode and re-encode with Cramberry
    var decoded User
    _ = oldDecoder.Unmarshal(oldData, &decoded)
    newData, _ := cramberry.Marshal(decoded)

    // Verify round-trip
    var final User
    _ = cramberry.Unmarshal(newData, &final)

    assert.Equal(t, original, final)
}
```

## Troubleshooting

### "unsupported map key type"

Cramberry requires primitive map keys:

```go
// Won't work
type Bad struct {
    Data map[MyStruct]string
}

// Works
type Good struct {
    Data map[string]string
}
```

### "unregistered type" for interfaces

Register concrete types before encoding:

```go
func init() {
    cramberry.MustRegister[ConcreteType]()
}
```

### Inconsistent encoding

Ensure `Deterministic: true` (default) when data must be reproducible:

```go
data, _ := cramberry.MarshalWithOptions(v, cramberry.DefaultOptions) // Deterministic
// NOT:
data, _ := cramberry.MarshalWithOptions(v, cramberry.FastOptions) // Non-deterministic
```
