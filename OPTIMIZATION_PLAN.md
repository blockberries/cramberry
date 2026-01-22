# Cramberry Performance Optimization Plan

## Executive Summary

This document outlines a comprehensive plan to improve Cramberry's serialization performance to match or approach Protocol Buffers (~2x improvement target). The plan prioritizes **safe, maintainable optimizations** over risky approaches like unsafe pointer access.

**Current State:** Cramberry is ~2x slower than Protocol Buffers due to heavy reflection usage in hot paths.

**Target State:** Within 20-30% of Protocol Buffers performance while maintaining safety and ease of use.

---

## Phase 1: Quick Wins (Low Effort, Safe)

### 1.1 Single-Pass Struct Encoding

**Problem:** Current implementation iterates struct fields twice:
1. First pass: Count non-zero fields (marshal.go:232-238)
2. Second pass: Encode fields (marshal.go:240-258)

**Solution:** Write a placeholder for field count, encode fields in single pass, then patch the count.

**Files to modify:**
- `pkg/cramberry/marshal.go` - `encodeStruct()` function

**Implementation:**
```go
func encodeStruct(w *Writer, v reflect.Value) error {
    info := getStructInfo(v.Type())

    // Remember position for field count
    countPos := w.Len()
    w.WriteUvarint(0) // Placeholder

    fieldCount := 0
    for _, field := range info.fields {
        fv := v.Field(field.index)
        if w.Options().OmitEmpty && isZeroValue(fv) {
            continue
        }
        fieldCount++
        w.WriteTag(field.num, getWireType(fv.Type()))
        if err := encodeValue(w, fv); err != nil {
            return err
        }
    }

    // Patch field count (requires new Writer method)
    w.PatchUvarint(countPos, uint64(fieldCount))
    return w.Err()
}
```

**Estimated improvement:** 5-10%
**Risk:** Low
**Effort:** 1-2 hours

---

### 1.2 Optimize isZeroValue() for Common Types

**Problem:** `isZeroValue()` uses reflection for every check, including simple scalars.

**Solution:** Fast-path common types before falling back to reflection.

**Files to modify:**
- `pkg/cramberry/marshal.go` - `isZeroValue()` function

**Implementation:**
```go
func isZeroValue(v reflect.Value) bool {
    // Fast path for common types using type switch on interface
    if v.CanInterface() {
        switch x := v.Interface().(type) {
        case int64:
            return x == 0
        case int32:
            return x == 0
        case string:
            return x == ""
        case bool:
            return !x
        case float64:
            return x == 0
        case []byte:
            return len(x) == 0
        }
    }
    // Fall back to reflection for complex types
    // ... existing implementation
}
```

**Estimated improvement:** 3-5%
**Risk:** Low
**Effort:** 1 hour

---

### 1.3 Pre-allocate Field Maps in Struct Decode

**Problem:** `decodeStruct()` allocates a new `fieldMap` on every call (unmarshal.go:200).

**Solution:** Store pre-built field map in cached `structInfo`.

**Files to modify:**
- `pkg/cramberry/marshal.go` - `structInfo` struct and `getStructInfo()`
- `pkg/cramberry/unmarshal.go` - `decodeStruct()` function

**Implementation:**
```go
type structInfo struct {
    fields   []fieldInfo
    fieldMap map[int]*fieldInfo  // Pre-built lookup map
}

func getStructInfo(t reflect.Type) *structInfo {
    // ... existing code ...

    // Build field map once
    info.fieldMap = make(map[int]*fieldInfo, len(info.fields))
    for i := range info.fields {
        info.fieldMap[info.fields[i].num] = &info.fields[i]
    }

    structInfoCache.Store(t, info)
    return info
}
```

**Estimated improvement:** 3-5%
**Risk:** Low
**Effort:** 30 minutes

---

### 1.4 Reduce sync.Map Overhead with Type Registry

**Problem:** `sync.Map.Load()` has atomic operation overhead on every struct encode/decode.

**Solution:** Use a two-tier cache: fast `map` protected by `RWMutex` for reads, with `sync.Map` as fallback for concurrent first-access.

**Files to modify:**
- `pkg/cramberry/marshal.go` - cache implementation

**Implementation:**
```go
var (
    structInfoMu    sync.RWMutex
    structInfoFast  = make(map[reflect.Type]*structInfo)
    structInfoSlow  sync.Map // For concurrent first-access
)

func getStructInfo(t reflect.Type) *structInfo {
    // Fast path: read lock
    structInfoMu.RLock()
    if info, ok := structInfoFast[t]; ok {
        structInfoMu.RUnlock()
        return info
    }
    structInfoMu.RUnlock()

    // Slow path: compute and store
    info := computeStructInfo(t)

    structInfoMu.Lock()
    structInfoFast[t] = info
    structInfoMu.Unlock()

    return info
}
```

**Estimated improvement:** 2-3%
**Risk:** Low
**Effort:** 1 hour

---

## Phase 2: Enhanced Code Generation (High Impact)

### 2.1 Overview

The current code generator creates methods that delegate to reflection:

```go
// Current generated code
func (m *SmallMessage) MarshalCramberry() ([]byte, error) {
    return cramberry.Marshal(m)  // Still uses reflection!
}
```

**Goal:** Generate actual encoding/decoding logic that bypasses reflection entirely.

### 2.2 New Generated Marshal Implementation

**Files to modify:**
- `pkg/codegen/go_generator.go` - Template and generation logic

**Target output:**
```go
// Generated marshal - NO REFLECTION
func (m *SmallMessage) MarshalCramberry() ([]byte, error) {
    w := cramberry.GetWriter()
    defer cramberry.PutWriter(w)

    // Field 1: Id (int64)
    w.WriteTag(1, cramberry.WireVarint)
    w.WriteInt64(m.Id)

    // Field 2: Name (string)
    w.WriteTag(2, cramberry.WireBytes)
    w.WriteString(m.Name)

    // Field 3: Active (bool)
    w.WriteTag(3, cramberry.WireVarint)
    w.WriteBool(m.Active)

    return w.BytesCopy(), w.Err()
}
```

### 2.3 New Generated Unmarshal Implementation

**Target output:**
```go
// Generated unmarshal - NO REFLECTION
func (m *SmallMessage) UnmarshalCramberry(data []byte) error {
    r := cramberry.NewReader(data)

    fieldCount := r.ReadUvarint()
    for i := uint64(0); i < fieldCount; i++ {
        fieldNum, wireType := r.ReadTag()
        switch fieldNum {
        case 1: // Id
            m.Id = r.ReadInt64()
        case 2: // Name
            m.Name = r.ReadString()
        case 3: // Active
            m.Active = r.ReadBool()
        default:
            r.SkipValue(wireType)
        }
    }

    return r.Err()
}
```

### 2.4 Handling Complex Types

#### Nested Messages
```go
// Field 5: Address (*Address)
if m.Address != nil {
    w.WriteTag(5, cramberry.WireBytes)
    nested, err := m.Address.MarshalCramberry()
    if err != nil {
        return nil, err
    }
    w.WriteBytes(nested)
}
```

#### Slices/Arrays
```go
// Field 6: Tags ([]string)
w.WriteTag(6, cramberry.WireBytes)
w.WriteArrayHeader(len(m.Tags))
for _, v := range m.Tags {
    w.WriteString(v)
}
```

#### Maps
```go
// Field 7: Metadata (map[string]string)
w.WriteTag(7, cramberry.WireBytes)
w.WriteMapHeader(len(m.Metadata))
// Sort keys for deterministic output
keys := make([]string, 0, len(m.Metadata))
for k := range m.Metadata {
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    w.WriteString(k)
    w.WriteString(m.Metadata[k])
}
```

#### Optional Fields (OmitEmpty)
```go
// Field 4: MiddleName (*string, optional)
if m.MiddleName != nil {
    w.WriteTag(4, cramberry.WireBytes)
    w.WriteString(*m.MiddleName)
}
```

### 2.5 Template Structure

```go
const goMarshalTemplate = `
{{range $msg := .Schema.Messages}}
// MarshalCramberry encodes {{$msg.Name}} to binary format.
// This is a generated method that bypasses reflection for performance.
func (m *{{goMessageType $msg}}) MarshalCramberry() ([]byte, error) {
    w := cramberry.GetWriter()
    defer cramberry.PutWriter(w)

    {{- range $field := $msg.Fields}}
    {{generateFieldEncode $field}}
    {{- end}}

    return w.BytesCopy(), w.Err()
}

// UnmarshalCramberry decodes {{$msg.Name}} from binary format.
func (m *{{goMessageType $msg}}) UnmarshalCramberry(data []byte) error {
    r := cramberry.NewReader(data)

    fieldCount := r.ReadUvarint()
    for i := uint64(0); i < fieldCount; i++ {
        fieldNum, wireType := r.ReadTag()
        switch fieldNum {
        {{- range $field := $msg.Fields}}
        case {{$field.Number}}:
            {{generateFieldDecode $field}}
        {{- end}}
        default:
            r.SkipValue(wireType)
        }
    }

    return r.Err()
}
{{end}}
`
```

### 2.6 Implementation Tasks

| Task | Description | Effort |
|------|-------------|--------|
| 2.6.1 | Add `generateFieldEncode()` template function | 4 hours |
| 2.6.2 | Add `generateFieldDecode()` template function | 4 hours |
| 2.6.3 | Handle all scalar types (bool, int*, uint*, float*, string, bytes) | 2 hours |
| 2.6.4 | Handle pointer/optional fields | 2 hours |
| 2.6.5 | Handle slices and arrays | 3 hours |
| 2.6.6 | Handle maps with key sorting | 3 hours |
| 2.6.7 | Handle nested messages | 2 hours |
| 2.6.8 | Handle enums | 1 hour |
| 2.6.9 | Add `--fast` flag to codegen CLI | 1 hour |
| 2.6.10 | Update tests and benchmarks | 2 hours |

**Total Phase 2 effort:** ~24 hours

**Estimated improvement:** 80-150% (approaching protobuf performance)

---

## Phase 3: Additional Optimizations

### 3.1 Buffer Size Hints

**Problem:** Writer starts with 256-byte buffer, may need multiple reallocations.

**Solution:** Add size estimation based on struct metadata.

```go
func (m *SmallMessage) MarshalCramberry() ([]byte, error) {
    // Estimated size: 3 fields * avg 10 bytes = 30 bytes
    w := cramberry.GetWriterWithCapacity(32)
    defer cramberry.PutWriter(w)
    // ...
}
```

**Estimated improvement:** 2-5%
**Effort:** 2 hours

---

### 3.2 Batch Operations API

**Problem:** Encoding many small messages has per-message overhead.

**Solution:** Add batch encode/decode API that reuses buffers.

```go
// New API for batch operations
func MarshalBatch[T any](items []T) ([][]byte, error) {
    w := GetWriter()
    defer PutWriter(w)

    results := make([][]byte, len(items))
    for i, item := range items {
        w.Reset()
        if err := encodeValue(w, reflect.ValueOf(item)); err != nil {
            return nil, err
        }
        results[i] = w.BytesCopy()
    }
    return results, nil
}
```

**Estimated improvement:** 10-20% for batch workloads
**Effort:** 3 hours

---

### 3.3 Streaming Optimizations

**Problem:** StreamWriter/StreamReader have per-message overhead.

**Solution:** Add framing optimizations and buffer pooling.

**Effort:** 4 hours

---

## Phase 4: Testing & Validation

### 4.1 Correctness Testing

- [ ] All existing tests pass with optimizations
- [ ] Generated code produces identical output to reflection-based code
- [ ] Fuzz testing for edge cases
- [ ] Cross-runtime compatibility maintained (Go ↔ TypeScript ↔ Rust)

### 4.2 Performance Validation

- [ ] Benchmark suite updated with new scenarios
- [ ] A/B comparison: reflection vs generated code
- [ ] Memory allocation profiling
- [ ] CPU profiling to identify remaining hotspots

### 4.3 Regression Prevention

- [ ] Add benchmark threshold tests to CI
- [ ] Document performance characteristics
- [ ] Add `go test -bench` to CI pipeline

---

## Implementation Order

```
Week 1: Phase 1 (Quick Wins)
├── 1.1 Single-pass encoding
├── 1.2 Optimize isZeroValue()
├── 1.3 Pre-allocate field maps
└── 1.4 Reduce sync.Map overhead

Week 2-3: Phase 2 (Code Generation)
├── 2.6.1-2.6.2 Core template functions
├── 2.6.3-2.6.4 Scalar and optional types
├── 2.6.5-2.6.7 Collections and nested types
└── 2.6.8-2.6.10 Enums, CLI, tests

Week 4: Phase 3 & 4 (Polish)
├── 3.1 Buffer size hints
├── 3.2 Batch operations
├── 4.1-4.3 Testing and validation
└── Documentation updates
```

---

## Success Metrics

| Metric | Current | Target | Stretch |
|--------|---------|--------|---------|
| SmallMessage encode | 93ns | 50ns | 45ns |
| SmallMessage decode | 114ns | 70ns | 65ns |
| Document encode | 1566ns | 1000ns | 900ns |
| Document decode | 2402ns | 1500ns | 1300ns |
| Batch1000 encode | 68μs | 35μs | 30μs |
| Batch1000 decode | 92μs | 65μs | 60μs |

**Overall target:** Within 20-30% of Protocol Buffers performance.

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Generated code has bugs | High | Extensive testing, fuzzing, golden file comparison |
| Breaking wire format | Critical | Version tests, cross-runtime validation |
| Increased code complexity | Medium | Good documentation, clear separation of concerns |
| Maintenance burden | Medium | Generated code is deterministic, easy to regenerate |

---

## Appendix: Files to Modify

### Phase 1
- `pkg/cramberry/marshal.go`
- `pkg/cramberry/unmarshal.go`
- `pkg/cramberry/writer.go` (add PatchUvarint)

### Phase 2
- `pkg/codegen/go_generator.go`
- `pkg/codegen/generator.go` (add --fast flag)
- `cmd/cramberry/main.go` (CLI updates)

### Phase 3
- `pkg/cramberry/writer.go`
- `pkg/cramberry/marshal.go`
- `pkg/cramberry/stream.go`

### Tests
- `pkg/cramberry/marshal_test.go`
- `pkg/cramberry/unmarshal_test.go`
- `pkg/codegen/go_generator_test.go`
- `benchmark/benchmark_test.go`
