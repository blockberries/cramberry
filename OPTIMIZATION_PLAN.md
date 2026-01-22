# Cramberry Performance Optimization Plan v2

## Executive Summary

This document outlines a comprehensive plan to make Cramberry's serialization performance **match or exceed Protocol Buffers**. Since wire format compatibility is not a constraint (v1 in development), we can make aggressive format changes for optimal performance.

**Current State:** Cramberry is ~2x slower than Protocol Buffers due to reflection and suboptimal wire format.

**Target State:** Match Protocol Buffers performance (within 10%) or exceed it.

---

## Wire Format Changes (Breaking)

The current format has inefficiencies that we can eliminate:

| Current | Problem | New Format |
|---------|---------|------------|
| Field count prefix | Requires two-pass or patching | Remove - use sentinel or length-prefix |
| Field tags on every field | Overhead for known schemas | Optional: positional encoding mode |
| Map key sorting | O(n log n) overhead | Make optional (deterministic flag) |
| Varint for small field numbers | Overkill for fields 1-15 | Single byte for common cases |

---

## Phase 1: Wire Format Optimization

### 1.1 Remove Field Count Prefix

**Current format:**
```
[field_count: varint] [tag1] [value1] [tag2] [value2] ...
```

**New format (Option A - End Marker):**
```
[tag1] [value1] [tag2] [value2] ... [0x00 end marker]
```

**New format (Option B - Length Prefix):**
```
[total_length: varint] [tag1] [value1] [tag2] [value2] ...
```

**Recommendation:** Option A (end marker) - simpler, no pre-calculation needed.

**Benefits:**
- Single-pass encoding (no counting or patching)
- Encoder can write fields directly
- Streaming-friendly

**Files to modify:**
- `pkg/cramberry/marshal.go`
- `pkg/cramberry/unmarshal.go`
- `pkg/cramberry/writer.go`
- `pkg/cramberry/reader.go`

**Effort:** 3 hours

---

### 1.2 Compact Field Tags

**Current:** Field tag is `(fieldNum << 3) | wireType` encoded as varint.

**Problem:** Even field 1 with wire type 0 = `0x08` (1 byte), but the varint encoding adds overhead.

**New format:** Optimize for common cases.

```
Field numbers 1-15 with wire types 0-7: Single byte
  [4 bits: field_num] [3 bits: wire_type] [1 bit: 0 = inline]

Field numbers 16+: Two+ bytes
  [4 bits: 0] [3 bits: wire_type] [1 bit: 1 = extended] [varint: field_num]
```

**Benefits:**
- Fields 1-15 (most common) use exactly 1 byte for tag
- No varint decoding overhead for common fields

**Effort:** 4 hours

---

### 1.3 Packed Repeated Primitives

**Current:** Each element has its own tag:
```
[tag] [value1] [tag] [value2] [tag] [value3]
```

**New format:** Single tag, length-prefixed array:
```
[tag] [count: varint] [value1] [value2] [value3]
```

**Benefits:**
- Eliminates N-1 redundant tags for N elements
- Better cache locality
- Faster iteration (no tag parsing per element)

**Example savings for `[]int32` with 100 elements:**
- Current: 100 tags × 1-2 bytes = 100-200 bytes overhead
- New: 1 tag + 1 count = 2-3 bytes overhead

**Effort:** 4 hours

---

### 1.4 Optional Deterministic Mode

**Current:** Maps are always sorted for deterministic output.

**New:** Sorting is opt-in via option flag.

```go
// Default: fast, non-deterministic
data, _ := cramberry.Marshal(msg)

// Opt-in: deterministic (sorted maps)
data, _ := cramberry.MarshalDeterministic(msg)
```

**Benefits:**
- Eliminates O(n log n) sort for maps
- Removes key slice allocation
- Most use cases don't need determinism

**Effort:** 2 hours

---

## Phase 2: Generated Code (Zero Reflection)

### 2.1 Overview

Generate complete marshal/unmarshal implementations that:
- Access struct fields directly (no reflection)
- Use optimized wire format from Phase 1
- Inline common operations
- Provide size hints for buffer allocation

### 2.2 Generated Encoder

```go
// Generated - zero reflection, optimized format
func (m *SmallMessage) MarshalCramberry() ([]byte, error) {
    // Size hint: 3 fields, ~20 bytes typical
    w := cramberry.GetWriterWithHint(24)

    // Field 1: Id (int64) - compact tag 0x08
    w.WriteByte(0x08)
    w.WriteVarint(m.Id)

    // Field 2: Name (string) - compact tag 0x12
    w.WriteByte(0x12)
    w.WriteStringDirect(m.Name)

    // Field 3: Active (bool) - compact tag 0x18
    w.WriteByte(0x18)
    w.WriteByte(boolToByte(m.Active))

    // End marker
    w.WriteByte(0x00)

    return w.Bytes(), nil
}
```

### 2.3 Generated Decoder

```go
// Generated - zero reflection, single pass
func (m *SmallMessage) UnmarshalCramberry(data []byte) error {
    r := cramberry.NewReader(data)

    for {
        tag := r.ReadByte()
        if tag == 0x00 { // End marker
            break
        }

        fieldNum := tag >> 4
        if tag&0x01 == 1 { // Extended field number
            fieldNum = r.ReadVarint()
        }

        switch fieldNum {
        case 1:
            m.Id = r.ReadVarint64()
        case 2:
            m.Name = r.ReadStringDirect()
        case 3:
            m.Active = r.ReadByte() != 0
        default:
            r.SkipField(tag & 0x0E >> 1) // wire type
        }
    }

    return r.Err()
}
```

### 2.4 Generated Size Calculator

```go
// Generated - for pre-allocation
func (m *SmallMessage) CramberrySize() int {
    size := 1 // end marker

    // Field 1: Id
    size += 1 + varintSize(m.Id)

    // Field 2: Name
    size += 1 + varintSize(len(m.Name)) + len(m.Name)

    // Field 3: Active
    size += 1 + 1

    return size
}

// Encoder uses size hint
func (m *SmallMessage) MarshalCramberry() ([]byte, error) {
    buf := make([]byte, m.CramberrySize())
    n, err := m.MarshalCramberryTo(buf)
    return buf[:n], err
}

// Zero-allocation encode to provided buffer
func (m *SmallMessage) MarshalCramberryTo(buf []byte) (int, error) {
    // Direct writes to buf, no intermediate buffer
}
```

### 2.5 Handling Complex Types

#### Nested Messages (with size prefix)
```go
// Field 5: Address (*Address)
if m.Address != nil {
    w.WriteByte(0x2A) // field 5, wire type 2 (bytes)

    // Write nested message with length prefix
    size := m.Address.CramberrySize()
    w.WriteVarint(size)
    m.Address.MarshalCramberryTo(w.NextBytes(size))
}
```

#### Packed Repeated Primitives
```go
// Field 6: Scores ([]int32) - packed
if len(m.Scores) > 0 {
    w.WriteByte(0x32) // field 6, wire type 2
    w.WriteVarint(len(m.Scores))
    for _, v := range m.Scores {
        w.WriteVarint32(v)
    }
}
```

#### Maps (unsorted by default)
```go
// Field 7: Metadata (map[string]string)
if len(m.Metadata) > 0 {
    w.WriteByte(0x3A) // field 7, wire type 2
    w.WriteVarint(len(m.Metadata))
    for k, v := range m.Metadata {
        w.WriteString(k)
        w.WriteString(v)
    }
}
```

### 2.6 Implementation Tasks

| Task | Description | Effort |
|------|-------------|--------|
| 2.6.1 | Implement new wire format in writer.go | 4 hours |
| 2.6.2 | Implement new wire format in reader.go | 4 hours |
| 2.6.3 | Add `generateFieldEncode()` template function | 3 hours |
| 2.6.4 | Add `generateFieldDecode()` template function | 3 hours |
| 2.6.5 | Add `generateSizeCalc()` template function | 2 hours |
| 2.6.6 | Handle all scalar types | 2 hours |
| 2.6.7 | Handle pointer/optional fields | 2 hours |
| 2.6.8 | Handle packed repeated fields | 3 hours |
| 2.6.9 | Handle maps (sorted and unsorted) | 2 hours |
| 2.6.10 | Handle nested messages | 2 hours |
| 2.6.11 | Handle enums | 1 hour |
| 2.6.12 | Update CLI and tests | 3 hours |

**Total Phase 2 effort:** ~31 hours

---

## Phase 3: Low-Level Optimizations

### 3.1 Zero-Copy String/Bytes Decode

**Problem:** Current `ReadString()` always allocates a new string.

**Solution:** Add unsafe zero-copy option for read-only access.

```go
// Zero-copy (string points into original buffer)
// Only safe if buffer outlives the string
func (r *Reader) ReadStringZeroCopy() string {
    length := r.ReadVarint()
    s := unsafe.String(&r.data[r.pos], length)
    r.pos += length
    return s
}
```

**When to use:** Generated code can use this when the decoded struct doesn't escape.

**Effort:** 2 hours

---

### 3.2 Inline Varint Encoding

**Problem:** Varint encode/decode are function calls.

**Solution:** Inline common cases (1-2 byte varints).

```go
// Inline for values < 128 (single byte)
func (w *Writer) WriteVarintInline(v uint64) {
    if v < 128 {
        w.buf = append(w.buf, byte(v))
        return
    }
    w.writeVarintSlow(v)
}
```

**Effort:** 2 hours

---

### 3.3 SIMD-Friendly Packed Arrays

For large arrays of fixed-size types (int32, int64, float32, float64), use direct memory copy instead of element-by-element encoding.

```go
// For []int32 on little-endian systems
func (w *Writer) WritePackedInt32Fast(values []int32) {
    w.WriteVarint(len(values))
    // Direct memory copy (4 bytes per element)
    header := (*reflect.SliceHeader)(unsafe.Pointer(&values))
    w.buf = append(w.buf, (*[1<<30]byte)(unsafe.Pointer(header.Data))[:len(values)*4]...)
}
```

**Note:** This uses unsafe but is well-contained and testable.

**Effort:** 3 hours

---

### 3.4 Buffer Pool Improvements

```go
var bufferPools = [...]sync.Pool{
    {New: func() any { return make([]byte, 0, 64) }},    // Tiny
    {New: func() any { return make([]byte, 0, 256) }},   // Small
    {New: func() any { return make([]byte, 0, 1024) }},  // Medium
    {New: func() any { return make([]byte, 0, 4096) }},  // Large
}

func getBuffer(sizeHint int) []byte {
    idx := bits.Len(uint(sizeHint) / 64)
    if idx >= len(bufferPools) {
        return make([]byte, 0, sizeHint)
    }
    return bufferPools[idx].Get().([]byte)
}
```

**Effort:** 2 hours

---

## Phase 4: TypeScript and Rust Codegen

Apply the same optimizations to other language generators:

### 4.1 TypeScript Generator Updates

```typescript
// Generated TypeScript - no reflection
export class SmallMessage {
  static encode(msg: SmallMessage, w: Writer): void {
    w.writeByte(0x08);
    w.writeVarint(msg.id);
    w.writeByte(0x12);
    w.writeString(msg.name);
    w.writeByte(0x18);
    w.writeByte(msg.active ? 1 : 0);
    w.writeByte(0x00); // end marker
  }

  static decode(r: Reader): SmallMessage {
    const msg = new SmallMessage();
    while (true) {
      const tag = r.readByte();
      if (tag === 0x00) break;
      const fieldNum = tag >> 4;
      switch (fieldNum) {
        case 1: msg.id = r.readVarint(); break;
        case 2: msg.name = r.readString(); break;
        case 3: msg.active = r.readByte() !== 0; break;
        default: r.skipField(tag);
      }
    }
    return msg;
  }
}
```

**Effort:** 8 hours

### 4.2 Rust Generator Updates

```rust
// Generated Rust - zero-copy where possible
impl SmallMessage {
    pub fn encode(&self, w: &mut Writer) -> Result<(), Error> {
        w.write_byte(0x08);
        w.write_varint(self.id as u64);
        w.write_byte(0x12);
        w.write_string(&self.name);
        w.write_byte(0x18);
        w.write_byte(if self.active { 1 } else { 0 });
        w.write_byte(0x00);
        Ok(())
    }

    pub fn decode(r: &mut Reader) -> Result<Self, Error> {
        let mut msg = Self::default();
        loop {
            let tag = r.read_byte()?;
            if tag == 0x00 { break; }
            match tag >> 4 {
                1 => msg.id = r.read_varint()? as i64,
                2 => msg.name = r.read_string()?,
                3 => msg.active = r.read_byte()? != 0,
                _ => r.skip_field(tag)?,
            }
        }
        Ok(msg)
    }
}
```

**Effort:** 8 hours

---

## Implementation Order

```
Phase 1: Wire Format (1 week)
├── 1.1 Remove field count prefix
├── 1.2 Compact field tags
├── 1.3 Packed repeated primitives
├── 1.4 Optional deterministic mode
└── Update all tests

Phase 2: Go Code Generation (2 weeks)
├── 2.6.1-2.6.2 New wire format in reader/writer
├── 2.6.3-2.6.5 Core template functions
├── 2.6.6-2.6.11 All field types
└── 2.6.12 CLI and tests

Phase 3: Low-Level Optimizations (1 week)
├── 3.1 Zero-copy string decode
├── 3.2 Inline varint
├── 3.3 Fast packed arrays
└── 3.4 Buffer pools

Phase 4: Cross-Language (1 week)
├── 4.1 TypeScript generator
├── 4.2 Rust generator
└── Cross-runtime tests
```

---

## Success Metrics

| Metric | Current | Protobuf | Target | Stretch |
|--------|---------|----------|--------|---------|
| SmallMessage encode | 93ns | 43ns | 40ns | 35ns |
| SmallMessage decode | 114ns | 67ns | 60ns | 50ns |
| Metrics encode | 220ns | 78ns | 75ns | 65ns |
| Metrics decode | 479ns | 109ns | 100ns | 90ns |
| Document encode | 1566ns | 938ns | 900ns | 800ns |
| Document decode | 2402ns | 1340ns | 1200ns | 1000ns |
| Batch1000 encode | 68μs | 29μs | 28μs | 25μs |
| Batch1000 decode | 92μs | 62μs | 55μs | 50μs |

**Target:** Match or beat Protocol Buffers.

---

## Wire Format Specification v2

### Message Encoding

```
Message := Field* EndMarker
Field := CompactTag Value | ExtendedTag Value
CompactTag := [fieldNum:4][wireType:3][0:1]  (fieldNum 1-15)
ExtendedTag := [0:4][wireType:3][1:1] FieldNum
FieldNum := Varint
EndMarker := 0x00
```

### Wire Types

```
0 = Varint (int32, int64, uint32, uint64, bool, enum)
1 = Fixed64 (fixed64, sfixed64, double)
2 = Bytes (string, bytes, nested message, packed repeated)
3 = Fixed32 (fixed32, sfixed32, float)
```

### Packed Repeated

```
PackedRepeated := Tag Length Count Elements
Length := Varint (total bytes of Count + Elements)
Count := Varint (number of elements)
Elements := Element*
```

### Map Encoding

```
Map := Tag Length Count Entries
Count := Varint (number of entries)
Entries := Entry*
Entry := Key Value
```

---

## Migration Path

Since wire format is changing:

1. **Bump format version** - Add version byte at start of encoded data
2. **Support both formats** - Reader can detect and handle v1 and v2
3. **Default to v2** - New code uses optimized format
4. **Deprecate v1** - Remove after transition period (or immediately if not in use)

```go
const (
    FormatV1 = 0x01 // Legacy reflection-based format
    FormatV2 = 0x02 // Optimized format
)

func Unmarshal(data []byte, v any) error {
    if len(data) == 0 {
        return ErrEmptyData
    }
    switch data[0] {
    case FormatV1:
        return unmarshalV1(data[1:], v)
    case FormatV2:
        return unmarshalV2(data[1:], v)
    default:
        // Assume v1 for backward compatibility
        return unmarshalV1(data, v)
    }
}
```

---

## Appendix: Files to Modify

### Phase 1
- `pkg/cramberry/wire.go` (new - wire format constants)
- `pkg/cramberry/writer.go`
- `pkg/cramberry/reader.go`
- `pkg/cramberry/marshal.go`
- `pkg/cramberry/unmarshal.go`

### Phase 2
- `pkg/codegen/go_generator.go`
- `pkg/codegen/generator.go`
- `cmd/cramberry/main.go`

### Phase 3
- `pkg/cramberry/writer.go`
- `pkg/cramberry/reader.go`
- `pkg/cramberry/pool.go` (new)

### Phase 4
- `pkg/codegen/typescript_generator.go`
- `pkg/codegen/rust_generator.go`
- `runtimes/ts/src/encoder.ts`
- `runtimes/ts/src/decoder.ts`
- `runtimes/rust/src/encoder.rs`
- `runtimes/rust/src/decoder.rs`

### Tests
- `pkg/cramberry/*_test.go`
- `pkg/codegen/*_test.go`
- `benchmark/benchmark_test.go`
- `tests/integration/*`
