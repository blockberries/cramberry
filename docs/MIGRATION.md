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
    cramberry.RegisterOrGet[Dog]()  // Auto TypeID: 128
    cramberry.RegisterOrGet[Cat]()  // Auto TypeID: 129
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

cramberry.RegisterOrGetWithID[Dog](TypeIDDog)
cramberry.RegisterOrGetWithID[Cat](TypeIDCat)
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
    cramberry.RegisterOrGet[ConcreteType]()
}
```

### Inconsistent encoding

Ensure `Deterministic: true` (default) when data must be reproducible:

```go
data, _ := cramberry.MarshalWithOptions(v, cramberry.DefaultOptions) // Deterministic
// NOT:
data, _ := cramberry.MarshalWithOptions(v, cramberry.FastOptions) // Non-deterministic
```

## Upgrading from v1.0.0 to v1.1.0

Version 1.1.0 includes important security hardening. Most code will work without changes.

### Breaking Changes

#### Rust: Registry API Change

The Rust `Registry` is now thread-safe. Methods that previously required `&mut self` now take `&self`:

**Before (v1.0.0):**
```rust
let mut registry = Registry::new();
registry.register(...)?;  // Required &mut self
```

**After (v1.1.0):**
```rust
let registry = Registry::new();
registry.register(...)?;  // Now takes &self

// Can now be shared across threads with Arc
let registry = Arc::new(Registry::new());
```

### Removed Functions

#### Go: MustRegister Functions (Removed)

`MustRegister()` and `MustRegisterWithID()` have been removed because they panic on duplicate registration, which can crash production services.

**Use instead:**
```go
// Option 1: Idempotent registration (safe to call multiple times)
cramberry.RegisterOrGet[MyType]()

// Option 2: Explicit error handling
id, err := cramberry.Register[MyType]()
if err != nil {
    // Handle duplicate registration
}
```

### New Features to Consider

#### Schema Compatibility Checking

Before deploying schema changes, verify backward compatibility:

```go
import "github.com/blockberries/cramberry/pkg/schema"

report := schema.CheckCompatibility(oldSchema, newSchema)
if !report.IsCompatible() {
    for _, change := range report.Breaking {
        log.Printf("Breaking change: %s", change.Error())
    }
}
```

#### Rust Streaming

New streaming support for processing large datasets:

```rust
use cramberry::stream::{StreamWriter, StreamReader};

// Write messages
let mut writer = StreamWriter::new(file);
writer.write_message(data)?;
writer.flush()?;

// Read messages
let mut reader = StreamReader::new(file);
while let Some(data) = reader.try_read_message()? {
    // Process message
}
```

#### TypeScript BigInt Precision

New methods warn about precision loss with large integers:

```typescript
// Old: silently loses precision for values > 2^53
const value = reader.readInt64();

// New: warns if precision would be lost
const value = reader.readInt64AsNumber();  // Logs warning for large values

// Or use BigInt for full precision
const value = reader.readSVarint64();  // Returns bigint
```

## Upgrading from v1.1.0 to v1.2.0

Version 1.2.0 introduces a breaking API change to fix a critical memory safety issue with zero-copy methods.

### Breaking Changes

#### Zero-Copy Methods Return Wrapper Types

Zero-copy methods now return wrapper types that validate references are still valid before allowing access. This prevents use-after-free bugs that could cause memory corruption.

**Before (v1.1.0):**
```go
r := cramberry.NewReader(data)
s := r.ReadStringZeroCopy()  // Returns string
b := r.ReadBytesNoCopy()     // Returns []byte
raw := r.ReadRawBytesNoCopy(10)  // Returns []byte

// DANGEROUS: s, b, raw now point to invalid memory!
r.Reset(newData)
fmt.Println(s)  // Undefined behavior - may crash or return garbage
```

**After (v1.2.0):**
```go
r := cramberry.NewReader(data)
s := r.ReadStringZeroCopy()  // Returns ZeroCopyString
b := r.ReadBytesNoCopy()     // Returns ZeroCopyBytes
raw := r.ReadRawBytesNoCopy(10)  // Returns ZeroCopyBytes

// Use the wrapper methods to get values
str := s.String()           // Get string value
bytes := b.Bytes()          // Get []byte value
rawBytes := raw.Bytes()     // Get []byte value

// SAFE: After Reset(), accessing wrappers will panic
// instead of returning corrupted data
r.Reset(newData)
_ = s.String()  // PANICS: "cramberry: ZeroCopyString accessed after Reader.Reset()"
```

#### Migration Options

**Option 1: Use the safe accessor methods (recommended)**
```go
r := cramberry.NewReader(data)
zcs := r.ReadStringZeroCopy()

// Check validity before use in long-lived code
if zcs.Valid() {
    processString(zcs.String())
}
```

**Option 2: Use unsafe accessors to bypass validation**

If you're certain the Reader won't be reset while references are in use:
```go
r := cramberry.NewReader(data)
s := r.ReadStringZeroCopy().UnsafeString()  // Returns string directly
b := r.ReadBytesNoCopy().UnsafeBytes()      // Returns []byte directly
```

**Option 3: Use copying methods instead**

For simplicity, use the copying versions which don't have this constraint:
```go
r := cramberry.NewReader(data)
s := r.ReadString()     // Returns owned copy - always safe
b := r.ReadBytes()      // Returns owned copy - always safe
raw := r.ReadRawBytes(10)  // Returns owned copy - always safe
```

### New Features

#### Generation Counter

The Reader now tracks a generation counter that increments on each `Reset()`:

```go
r := cramberry.NewReader(data)
gen1 := r.Generation()  // Returns 0

r.Reset(newData)
gen2 := r.Generation()  // Returns 1
```

#### Wrapper Type Methods

The new wrapper types provide several useful methods:

```go
zcs := r.ReadStringZeroCopy()

// Validation
if zcs.Valid() { ... }      // Check if reference is still valid

// Panicking access (use when you're certain the reader hasn't been reset)
s := zcs.String()           // Get value (panics if invalid)
s := zcs.MustString()       // Same as String() - explicit panic in name
s := zcs.UnsafeString()     // Get value (no validation - use with caution)

// Non-panicking access (safe alternatives)
s := zcs.StringOrEmpty()    // Returns "" if invalid (no panic)
s, ok := zcs.TryString()    // Returns ("", false) if invalid

// Utility
length := zcs.Len()         // Length without validation
empty := zcs.IsEmpty()      // Empty check without validation
```

```go
zcb := r.ReadBytesNoCopy()

// Validation
if zcb.Valid() { ... }      // Check if reference is still valid

// Panicking access (use when you're certain the reader hasn't been reset)
b := zcb.Bytes()            // Get value (panics if invalid)
b := zcb.MustBytes()        // Same as Bytes() - explicit panic in name
b := zcb.UnsafeBytes()      // Get value (no validation - use with caution)
s := zcb.String()           // Get as string (panics if invalid)

// Non-panicking access (safe alternatives)
b := zcb.BytesOrNil()       // Returns nil if invalid (no panic)
b, ok := zcb.TryBytes()     // Returns (nil, false) if invalid
s := zcb.StringOrEmpty()    // Returns "" if invalid (no panic)
s, ok := zcb.TryString()    // Returns ("", false) if invalid

// Utility
length := zcb.Len()         // Length without validation
empty := zcb.IsEmpty()      // Empty check without validation
```
