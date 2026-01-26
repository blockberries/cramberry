# Cramberry Prerelease Implementation Plan

This document provides a comprehensive, prioritized implementation plan for preparing Cramberry for production release. Despite being functionally complete, the code review has identified **critical security vulnerabilities** that must be addressed before production deployment.

**Current Status**: Functionally 100% complete, **SECURITY ISSUES IDENTIFIED**
**Target**: v1.1.0 stable release with security hardening

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Critical Security Issues](#critical-security-issues)
3. [Phase 1: Security Hardening](#phase-1-security-hardening)
4. [Phase 2: Cross-Language Consistency](#phase-2-cross-language-consistency)
5. [Phase 3: Performance Optimization](#phase-3-performance-optimization)
6. [Phase 4: Fuzzing & Testing](#phase-4-fuzzing--testing)
7. [Phase 5: Developer Experience](#phase-5-developer-experience)
8. [Testing Requirements](#testing-requirements)
9. [Release Checklist](#release-checklist)

---

## Executive Summary

Cramberry is a high-performance binary serialization library with excellent architectural foundations. However, the comprehensive code review identified **2 critical** and **6 high-severity** security issues that must be resolved before production use.

### Severity Breakdown

| Severity | Count | Status |
|----------|-------|--------|
| CRITICAL | 2 | Must fix before release |
| HIGH | 6 | Must fix before release |
| MEDIUM | 8 | Should fix for v1.1 |
| LOW | 5 | Nice to have |

### Current Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| Architecture | A | Clean layering, good separation |
| Security | **D+** | Critical bugs identified |
| Correctness | C+ | Cross-language issues |
| Performance | A- | 1.5-2.6x faster than Protobuf |
| Testing | A- | Missing fuzzing |

### NOT Safe For Production Until Fixed

1. Integer overflow in packed array writers (CRITICAL)
2. Zero-copy string memory safety (CRITICAL)
3. Cross-language varint consistency (HIGH)
4. Array bounds checking in unmarshal (HIGH)
5. NaN canonicalization in packed floats (HIGH)

---

## Critical Security Issues

### Issue Dependency Graph

```
┌─────────────────────────────────────────────────────────────────┐
│                    CRITICAL PATH                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  C1 (Integer Overflow) ──┬── needed for ── C2 (Zero-Copy)      │
│                          │                                      │
│  H1 (Varint Consistency) ┤                                      │
│  H2 (Aliasing Bugs) ─────┤                                      │
│  H3 (MaxInt Check) ──────┼── all must pass for ── RELEASE     │
│  H4 (MustRegister) ──────┤                                      │
│  H5 (NaN Packed) ────────┤                                      │
│  H6 (Array Bounds) ──────┘                                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Security Hardening

Focus: Fix all critical and high-severity security issues.

### C1: Integer Overflow in Packed Array Writers

**Priority**: CRITICAL
**Location**: `pkg/cramberry/writer.go:693-726`
**Effort**: 2 hours

#### Problem

```go
// CURRENT (VULNERABLE)
func (w *Writer) WritePackedFloat32(values []float32) {
    if len(values) == 0 { return }
    byteSize := len(values) * 4  // *** NO OVERFLOW CHECK ***
    w.grow(byteSize)
    // ...
}
```

With `len(values) = 536870912` (2^29), multiplying by 4 gives `2147483648`, which overflows int32 to a negative value, bypassing bounds checking.

#### Solution

```go
// pkg/cramberry/writer.go

import "math"

const (
    // Maximum safe array length for packed float32
    MaxPackedFloat32Length = math.MaxInt / 4

    // Maximum safe array length for packed float64
    MaxPackedFloat64Length = math.MaxInt / 8

    // Maximum safe array length for packed fixed32
    MaxPackedFixed32Length = math.MaxInt / 4

    // Maximum safe array length for packed fixed64
    MaxPackedFixed64Length = math.MaxInt / 8
)

func (w *Writer) WritePackedFloat32(values []float32) {
    if len(values) == 0 {
        return
    }
    if len(values) > MaxPackedFloat32Length {
        w.setError(ErrMaxArrayLength)
        return
    }
    byteSize := len(values) * 4
    w.grow(byteSize)
    for _, v := range values {
        bits := wire.CanonicalizeFloat32(v)  // Also fix NaN issue
        w.buf = append(w.buf, byte(bits), byte(bits>>8), byte(bits>>16), byte(bits>>24))
    }
}

func (w *Writer) WritePackedFloat64(values []float64) {
    if len(values) == 0 {
        return
    }
    if len(values) > MaxPackedFloat64Length {
        w.setError(ErrMaxArrayLength)
        return
    }
    byteSize := len(values) * 8
    w.grow(byteSize)
    for _, v := range values {
        bits := wire.CanonicalizeFloat64(v)  // Also fix NaN issue
        // ... encoding
    }
}

func (w *Writer) WritePackedFixed32(values []uint32) {
    if len(values) == 0 {
        return
    }
    if len(values) > MaxPackedFixed32Length {
        w.setError(ErrMaxArrayLength)
        return
    }
    // ...
}

func (w *Writer) WritePackedFixed64(values []uint64) {
    if len(values) == 0 {
        return
    }
    if len(values) > MaxPackedFixed64Length {
        w.setError(ErrMaxArrayLength)
        return
    }
    // ...
}
```

#### Tests Required

```go
func TestWriter_PackedFloat32_OverflowProtection(t *testing.T) {
    w := NewWriter()

    // Create slice that would overflow if multiplied by 4
    // We can't actually allocate this much, so we need to test the check
    // by mocking or by testing the limit constant

    // Test that the limit constant is correct
    require.Equal(t, math.MaxInt/4, MaxPackedFloat32Length)

    // Test that error is returned for oversized input
    // (can't actually create such a large slice, so test the logic separately)
}

func TestWriter_PackedFloat64_OverflowProtection(t *testing.T) {
    // Similar
}
```

---

### C2: Zero-Copy String Memory Safety

**Priority**: CRITICAL
**Location**: `pkg/cramberry/reader.go:447-451`
**Effort**: 4 hours

#### Problem

```go
// CURRENT (DANGEROUS)
func (r *Reader) ReadStringZeroCopy() string {
    // ... bounds checking ...
    s := unsafe.String(&r.data[r.pos], n)  // Points into buffer
    r.pos += n
    return s
}
```

If the buffer is freed, modified, or reused, the returned string becomes invalid, causing use-after-free or data corruption.

#### Solution Options

**Option A: Remove zero-copy methods entirely (Recommended for safety)**

```go
// pkg/cramberry/reader.go

// ReadStringZeroCopy is DEPRECATED and removed for safety.
// Use ReadString() instead.
//
// Previous behavior: returned a string pointing directly into the
// Reader's buffer, which could cause memory corruption if the buffer
// was modified or freed while the string was in use.
//
// Migration: Replace all ReadStringZeroCopy() calls with ReadString().
// The performance impact is minimal for most workloads.
//
// func (r *Reader) ReadStringZeroCopy() string - REMOVED
```

**Option B: Add generation counter for detection (If performance is critical)**

```go
// pkg/cramberry/reader.go

type Reader struct {
    // ... existing fields
    generation uint64  // Incremented on Reset()
}

// ZeroCopyString represents a string that references the Reader's buffer.
// It is only valid while the Reader's generation matches.
type ZeroCopyString struct {
    s          string
    generation uint64
    reader     *Reader
}

// String returns the string value, panicking if the Reader has been reset.
func (zcs ZeroCopyString) String() string {
    if zcs.reader.generation != zcs.generation {
        panic("cramberry: zero-copy string used after Reader.Reset()")
    }
    return zcs.s
}

// Valid returns true if the string is still valid.
func (zcs ZeroCopyString) Valid() bool {
    return zcs.reader.generation == zcs.generation
}

func (r *Reader) ReadStringZeroCopy() ZeroCopyString {
    length := r.readLength()
    if r.err != nil {
        return ZeroCopyString{}
    }

    n := int(length)
    if r.pos+n > len(r.data) {
        r.setErrorAt(ErrUnexpectedEOF, "string data")
        return ZeroCopyString{}
    }

    s := unsafe.String(&r.data[r.pos], n)
    r.pos += n

    return ZeroCopyString{
        s:          s,
        generation: r.generation,
        reader:     r,
    }
}

func (r *Reader) Reset(data []byte) {
    r.data = data
    r.pos = 0
    r.err = nil
    r.errLoc = ""
    r.generation++  // Invalidate all zero-copy references
}
```

**Option C: Document lifetime requirements explicitly (Minimum fix)**

```go
// ReadStringZeroCopy returns a string that points directly into the Reader's buffer.
//
// WARNING: The returned string is only valid under these conditions:
//   1. The Reader must NOT be Reset() while the string is in use
//   2. The underlying data buffer must NOT be modified or freed
//   3. The Reader must remain in scope
//
// Failure to observe these constraints will cause undefined behavior,
// including memory corruption, crashes, or data races.
//
// For safe usage, prefer ReadString() instead. Use ReadStringZeroCopy()
// only when:
//   - Performance is critical
//   - You can guarantee the Reader outlives all returned strings
//   - You will not call Reset() while strings are in use
//
// Example of UNSAFE usage (DO NOT DO THIS):
//
//   r := cramberry.NewReader(data)
//   s := r.ReadStringZeroCopy()
//   r.Reset(newData)  // UNDEFINED BEHAVIOR: s now points to invalid memory
//   fmt.Println(s)    // CRASH or data corruption
//
// Example of safe usage:
//
//   func processMessage(data []byte) string {
//       r := cramberry.NewReader(data)
//       s := r.ReadStringZeroCopy()
//       result := processString(s)  // Use s immediately, don't store it
//       return result               // Don't return s itself
//   }
func (r *Reader) ReadStringZeroCopy() string
```

**Recommendation**: Use Option A (removal) for v1.0 release, then consider Option B for v1.1 if users request it with a clear use case.

---

### H1: Cross-Language Varint Consistency

**Priority**: HIGH
**Locations**:
- Go: `internal/wire/varint.go:77-92`
- TypeScript: `typescript/src/reader.ts:78-88`
- Rust: `rust/src/reader.rs:65-76`
**Effort**: 3 hours

#### Problem

Go enforces a 10-byte maximum for varints (correct per protobuf spec), but TypeScript and Rust use different limits.

```go
// Go: Correct - 10 byte max
if i == 9 && b > 1 { return 0, 0, ErrVarintOverflow }

// TypeScript: WRONG
while (shift < 35) { shift += 7; }

// Rust: WRONG
while (shift < 70) { shift += 7; }
```

#### Solution

**TypeScript Fix** (`typescript/src/reader.ts`):

```typescript
// typescript/src/reader.ts

export class Reader {
    // Maximum bytes for a varint (64-bit value encoded as varint)
    private static readonly MAX_VARINT_BYTES = 10;

    readVarint(): bigint {
        let result = 0n;
        let shift = 0n;
        let byteCount = 0;

        while (true) {
            if (byteCount >= Reader.MAX_VARINT_BYTES) {
                throw new Error('Varint overflow: exceeded 10 bytes');
            }

            const b = this.readByte();
            result |= BigInt(b & 0x7F) << shift;

            if ((b & 0x80) === 0) {
                return result;
            }

            shift += 7n;
            byteCount++;

            // Additional check: 10th byte must have only 1 significant bit
            if (byteCount === 10 && b > 1) {
                throw new Error('Varint overflow: 10th byte must be 0 or 1');
            }
        }
    }
}
```

**Rust Fix** (`rust/src/reader.rs`):

```rust
// rust/src/reader.rs

impl Reader {
    const MAX_VARINT_BYTES: usize = 10;

    pub fn read_varint(&mut self) -> Result<u64, CramberryError> {
        let mut result: u64 = 0;
        let mut shift: u32 = 0;

        for i in 0..Self::MAX_VARINT_BYTES {
            let b = self.read_byte()?;
            result |= ((b & 0x7F) as u64) << shift;

            if (b & 0x80) == 0 {
                return Ok(result);
            }

            shift += 7;

            // 10th byte must have only 1 significant bit (for 64-bit max)
            if i == 9 && b > 1 {
                return Err(CramberryError::VarintOverflow);
            }
        }

        Err(CramberryError::VarintOverflow)
    }
}
```

#### Cross-Language Test

```go
// test/interop/varint_test.go

func TestVarintCrossLanguageConsistency(t *testing.T) {
    testCases := []struct {
        name     string
        value    uint64
        expected []byte
    }{
        {"zero", 0, []byte{0x00}},
        {"one", 1, []byte{0x01}},
        {"127", 127, []byte{0x7F}},
        {"128", 128, []byte{0x80, 0x01}},
        {"max_uint64", math.MaxUint64, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test Go encoding
            goEncoded := wire.AppendVarint(nil, tc.value)
            require.Equal(t, tc.expected, goEncoded)

            // Test Go decoding
            goDecoded, n, err := wire.ReadVarint(tc.expected)
            require.NoError(t, err)
            require.Equal(t, tc.value, goDecoded)
            require.Equal(t, len(tc.expected), n)

            // Test TypeScript via subprocess
            // Test Rust via subprocess
        })
    }
}
```

---

### H2: ReadBytesNoCopy Aliasing Safety

**Priority**: HIGH
**Location**: `reader.go:506`
**Effort**: 2 hours

Apply the same treatment as C2 - either remove, add generation tracking, or document clearly.

```go
// Apply same fix as ReadStringZeroCopy
func (r *Reader) ReadBytesNoCopy() []byte  // REMOVE or add safety
```

---

### H3: MaxInt Bounds Check

**Priority**: HIGH
**Location**: `reader.go:399, 434, 463, 493, 577`
**Effort**: 1 hour

#### Problem

```go
// CURRENT
if length > uint64(MaxInt) {
    r.setErrorAt(ErrOverflow, "string length overflow")
}
n := int(length)  // Still unchecked on 32-bit systems
```

#### Solution

```go
// pkg/cramberry/reader.go

import "math"

func (r *Reader) ReadString() string {
    length := r.readLength()
    if r.err != nil {
        return ""
    }

    // Use math.MaxInt for platform-independent check
    if length > uint64(math.MaxInt) {
        r.setErrorAt(ErrOverflow, "string length exceeds platform int size")
        return ""
    }

    n := int(length)

    // Explicit bounds check for available data
    if n < 0 || r.pos+n > len(r.data) {
        r.setErrorAt(ErrUnexpectedEOF, "string data")
        return ""
    }

    s := string(r.data[r.pos : r.pos+n])
    r.pos += n
    return s
}
```

---

### H4: MustRegister Panic Recovery

**Priority**: HIGH
**Location**: `registry.go:271-282`
**Effort**: 2 hours

#### Problem

MustRegister panics on duplicate registration, which can crash production services.

#### Solution

**Option A: Deprecate MustRegister**

```go
// MustRegister is DEPRECATED. Use Register() with error handling instead.
//
// Deprecated: MustRegister can crash production services if called with
// a duplicate type. Use Register() and handle errors appropriately.
func (r *Registry) MustRegister[T any]() TypeID {
    id, err := r.RegisterType(reflect.TypeOf((*T)(nil)).Elem())
    if err != nil {
        panic(fmt.Sprintf("cramberry.MustRegister: %v", err))
    }
    return id
}
```

**Option B: Add idempotent registration**

```go
// RegisterOrGet registers a type and returns its ID, or returns the existing
// ID if the type is already registered.
func (r *Registry) RegisterOrGet[T any]() TypeID {
    t := reflect.TypeOf((*T)(nil)).Elem()

    r.mu.Lock()
    defer r.mu.Unlock()

    // Check if already registered
    if id, ok := r.typeToID[t]; ok {
        return id
    }

    // Register new type
    id := r.nextID
    r.nextID++
    r.typeToID[t] = id
    r.idToType[id] = t

    return id
}
```

---

### H5: NaN Canonicalization in Packed Arrays

**Priority**: HIGH
**Location**: `writer.go:684-726`
**Effort**: 2 hours

#### Problem

Packed float arrays don't canonicalize NaN values, breaking deterministic encoding.

#### Solution

Already included in C1 fix. Ensure all packed float writes use canonicalization:

```go
// internal/wire/float.go

// CanonicalFloat32Bits returns the canonical bit representation of a float32.
// NaN values are converted to the canonical NaN (0x7FC00000).
func CanonicalFloat32Bits(f float32) uint32 {
    bits := math.Float32bits(f)
    if bits&0x7F800000 == 0x7F800000 && bits&0x007FFFFF != 0 {
        // NaN detected - return canonical NaN
        return 0x7FC00000
    }
    return bits
}

// CanonicalFloat64Bits returns the canonical bit representation of a float64.
// NaN values are converted to the canonical NaN (0x7FF8000000000000).
func CanonicalFloat64Bits(f float64) uint64 {
    bits := math.Float64bits(f)
    if bits&0x7FF0000000000000 == 0x7FF0000000000000 && bits&0x000FFFFFFFFFFFFF != 0 {
        // NaN detected - return canonical NaN
        return 0x7FF8000000000000
    }
    return bits
}
```

---

### H6: Array Bounds Checking in Unmarshal

**Priority**: HIGH
**Location**: `unmarshal.go:130, 155, 159`
**Effort**: 2 hours

#### Problem

```go
// CURRENT - No limit check
func decodeSlice(r *Reader, v reflect.Value) error {
    n := r.ReadArrayHeader()  // No limit check
    slice := reflect.MakeSlice(v.Type(), n, n)  // Unchecked allocation
}
```

#### Solution

```go
// pkg/cramberry/unmarshal.go

func decodeSlice(r *Reader, v reflect.Value, opts *Options) error {
    n := r.ReadArrayHeader()
    if r.HasError() {
        return r.Error()
    }

    // Check array length limit
    if opts.Limits.MaxArrayLength > 0 && n > opts.Limits.MaxArrayLength {
        return fmt.Errorf("array length %d exceeds limit %d", n, opts.Limits.MaxArrayLength)
    }

    // Check that we won't allocate excessive memory
    elemSize := int(v.Type().Elem().Size())
    if elemSize > 0 {
        maxElems := opts.Limits.MaxMessageSize / int64(elemSize)
        if int64(n) > maxElems {
            return fmt.Errorf("array would exceed memory limit")
        }
    }

    slice := reflect.MakeSlice(v.Type(), n, n)
    // ... decode elements
}
```

---

## Phase 2: Cross-Language Consistency

Focus: Ensure identical behavior across Go, TypeScript, and Rust.

### P2-1: Thread-Safe Rust Registry

**Priority**: HIGH (from ROADMAP)
**Location**: `rust/src/registry.rs`
**Effort**: 3 hours

```rust
// rust/src/registry.rs

use std::collections::HashMap;
use std::sync::RwLock;

pub struct Registry {
    type_to_id: RwLock<HashMap<TypeInfo, TypeId>>,
    id_to_type: RwLock<HashMap<TypeId, TypeInfo>>,
    next_id: RwLock<TypeId>,
}

impl Registry {
    pub fn new() -> Self {
        Self {
            type_to_id: RwLock::new(HashMap::new()),
            id_to_type: RwLock::new(HashMap::new()),
            next_id: RwLock::new(1),
        }
    }

    pub fn register(&self, type_info: TypeInfo) -> Result<TypeId, RegistryError> {
        // Check if already registered (read lock)
        {
            let type_to_id = self.type_to_id.read().unwrap();
            if let Some(&id) = type_to_id.get(&type_info) {
                return Ok(id);
            }
        }

        // Register new type (write lock)
        let mut type_to_id = self.type_to_id.write().unwrap();
        let mut id_to_type = self.id_to_type.write().unwrap();
        let mut next_id = self.next_id.write().unwrap();

        // Double-check after acquiring write lock
        if let Some(&id) = type_to_id.get(&type_info) {
            return Ok(id);
        }

        let id = *next_id;
        *next_id += 1;

        type_to_id.insert(type_info.clone(), id);
        id_to_type.insert(id, type_info);

        Ok(id)
    }
}

// Implement Send + Sync
unsafe impl Send for Registry {}
unsafe impl Sync for Registry {}
```

---

### P2-2: TypeScript BigInt Precision Warning

**Priority**: MEDIUM
**Location**: `typescript/src/reader.ts`
**Effort**: 2 hours

Add explicit handling for large integers:

```typescript
// typescript/src/reader.ts

export class Reader {
    /**
     * Reads an int64 value.
     *
     * WARNING: JavaScript numbers can only safely represent integers
     * up to Number.MAX_SAFE_INTEGER (2^53-1). Values larger than this
     * may lose precision.
     *
     * For values that may exceed 2^53-1, use readInt64AsBigInt() instead.
     */
    readInt64(): number {
        const value = this.readInt64AsBigInt();
        if (value > BigInt(Number.MAX_SAFE_INTEGER) ||
            value < BigInt(Number.MIN_SAFE_INTEGER)) {
            console.warn(
                `cramberry: int64 value ${value} exceeds safe integer range, precision may be lost`
            );
        }
        return Number(value);
    }

    /**
     * Reads an int64 value as BigInt for full precision.
     */
    readInt64AsBigInt(): bigint {
        // ... implementation
    }
}
```

---

### P2-3: Rust Streaming Support

**Priority**: HIGH (from ROADMAP)
**Location**: `rust/src/stream.rs` (new)
**Effort**: 1-2 days

```rust
// rust/src/stream.rs

use std::io::{Read, Write, BufReader, BufWriter};

pub struct StreamWriter<W: Write> {
    inner: BufWriter<W>,
}

impl<W: Write> StreamWriter<W> {
    pub fn new(writer: W) -> Self {
        Self {
            inner: BufWriter::new(writer),
        }
    }

    pub fn write_message<T: Marshal>(&mut self, msg: &T) -> Result<(), CramberryError> {
        let data = msg.marshal()?;

        // Write length-delimited message
        self.write_varint(data.len() as u64)?;
        self.inner.write_all(&data)?;

        Ok(())
    }

    pub fn flush(&mut self) -> Result<(), std::io::Error> {
        self.inner.flush()
    }
}

pub struct StreamReader<R: Read> {
    inner: BufReader<R>,
}

impl<R: Read> StreamReader<R> {
    pub fn new(reader: R) -> Self {
        Self {
            inner: BufReader::new(reader),
        }
    }

    pub fn read_message<T: Unmarshal>(&mut self) -> Result<T, CramberryError> {
        let len = self.read_varint()? as usize;

        let mut data = vec![0u8; len];
        self.inner.read_exact(&mut data)?;

        T::unmarshal(&data)
    }
}

// Async variants (feature-gated)
#[cfg(feature = "tokio")]
pub mod async_stream {
    // ... tokio-based async implementation
}
```

---

## Phase 3: Performance Optimization

Focus: Implement planned performance improvements from ROADMAP.

### P3-1: Reflection Caching for Go Runtime

**Priority**: HIGH (from ROADMAP)
**Effort**: 1-2 days
**Impact**: 20-40% performance improvement

```go
// pkg/cramberry/cache.go

package cramberry

import (
    "reflect"
    "sync"
)

// structInfo holds cached information about a struct type.
type structInfo struct {
    fields []fieldInfo
}

// fieldInfo holds cached information about a struct field.
type fieldInfo struct {
    index     int
    fieldNum  int
    wireType  WireType
    encoder   func(*Writer, reflect.Value)
    decoder   func(*Reader, reflect.Value) error
}

// Global cache of struct info
var (
    structCache sync.Map // map[reflect.Type]*structInfo
)

// getStructInfo returns cached struct info, computing it if necessary.
func getStructInfo(t reflect.Type) *structInfo {
    if cached, ok := structCache.Load(t); ok {
        return cached.(*structInfo)
    }

    info := computeStructInfo(t)
    structCache.Store(t, info)
    return info
}

func computeStructInfo(t reflect.Type) *structInfo {
    info := &structInfo{
        fields: make([]fieldInfo, 0, t.NumField()),
    }

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)

        // Skip unexported fields
        if !field.IsExported() {
            continue
        }

        // Parse struct tag
        tag := field.Tag.Get("cramberry")
        if tag == "-" {
            continue
        }

        fi := fieldInfo{
            index: i,
            // Parse field number, wire type from tag
        }

        // Pre-compute encoder/decoder functions
        fi.encoder = getEncoder(field.Type)
        fi.decoder = getDecoder(field.Type)

        info.fields = append(info.fields, fi)
    }

    return info
}

// WarmCache pre-populates the cache for the given types.
// Call this at startup for latency-sensitive applications.
func WarmCache(types ...reflect.Type) {
    for _, t := range types {
        getStructInfo(t)
    }
}
```

---

### P3-2: Buffer Pooling

**Priority**: MEDIUM
**Effort**: 4 hours

```go
// pkg/cramberry/pool.go

var (
    // Pool for small buffers (up to 4KB)
    smallBufferPool = sync.Pool{
        New: func() interface{} {
            b := make([]byte, 0, 4096)
            return &b
        },
    }

    // Pool for large buffers (up to 64KB)
    largeBufferPool = sync.Pool{
        New: func() interface{} {
            b := make([]byte, 0, 65536)
            return &b
        },
    }
)

func getBuffer(size int) *[]byte {
    if size <= 4096 {
        return smallBufferPool.Get().(*[]byte)
    }
    if size <= 65536 {
        return largeBufferPool.Get().(*[]byte)
    }
    b := make([]byte, 0, size)
    return &b
}

func putBuffer(b *[]byte) {
    cap := cap(*b)
    *b = (*b)[:0]

    if cap <= 4096 {
        smallBufferPool.Put(b)
    } else if cap <= 65536 {
        largeBufferPool.Put(b)
    }
    // Don't pool very large buffers
}
```

---

## Phase 4: Fuzzing & Testing

Focus: Comprehensive fuzzing and security testing.

### P4-1: Go Fuzz Targets

**Priority**: HIGH
**Effort**: 2-3 days

```go
// fuzz/parser_fuzz.go

//go:build go1.18

package fuzz

import (
    "testing"

    "github.com/blockberries/cramberry/pkg/schema"
)

func FuzzSchemaParser(f *testing.F) {
    // Seed corpus
    f.Add([]byte(`message Foo { bar: int32 = 1; }`))
    f.Add([]byte(`interface Bar { Baz }`))

    f.Fuzz(func(t *testing.T, data []byte) {
        // Parser should never panic on any input
        _, _ = schema.Parse(string(data))
    })
}
```

```go
// fuzz/decoder_fuzz.go

func FuzzVarintDecoder(f *testing.F) {
    f.Add([]byte{0x00})
    f.Add([]byte{0x7F})
    f.Add([]byte{0x80, 0x01})
    f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01})

    f.Fuzz(func(t *testing.T, data []byte) {
        // Should never panic
        _, _, _ = wire.ReadVarint(data)
    })
}

func FuzzMessageDecoder(f *testing.F) {
    // Add valid message corpus
    f.Add(encodeTestMessage())

    f.Fuzz(func(t *testing.T, data []byte) {
        r := NewReader(data)
        var msg TestMessage
        // Should never panic, only return error
        _ = Unmarshal(r, &msg)
    })
}
```

---

### P4-2: Cross-Language Interop Tests

**Priority**: HIGH
**Effort**: 2 days

```go
// test/interop/interop_test.go

func TestCrossLanguageRoundTrip(t *testing.T) {
    testCases := []struct {
        name string
        msg  interface{}
    }{
        {"SmallMessage", &SmallMessage{A: 1, B: "test", C: 3.14}},
        {"WithNaN", &SmallMessage{C: math.NaN()}},
        {"MaxInt64", &LargeInts{I64: math.MaxInt64}},
        {"MaxUint64", &LargeInts{U64: math.MaxUint64}},
        {"DeepNesting", createDeeplyNested(100)},
        {"LargeArray", &LargeArray{Values: make([]int32, 10000)}},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Encode in Go
            goEncoded, err := Marshal(tc.msg)
            require.NoError(t, err)

            // Decode in TypeScript
            tsDecoded := runTypeScript("decode.ts", goEncoded)
            require.Equal(t, tc.msg, tsDecoded)

            // Encode in TypeScript
            tsEncoded := runTypeScript("encode.ts", tc.msg)

            // Verify byte-for-byte identical
            require.Equal(t, goEncoded, tsEncoded, "Go and TypeScript encodings differ")

            // Same for Rust
            rustDecoded := runRust("decode", goEncoded)
            require.Equal(t, tc.msg, rustDecoded)

            rustEncoded := runRust("encode", tc.msg)
            require.Equal(t, goEncoded, rustEncoded, "Go and Rust encodings differ")
        })
    }
}
```

---

### P4-3: Concurrent Stress Tests

**Priority**: MEDIUM
**Effort**: 1 day

```go
func TestConcurrentMarshalUnmarshal(t *testing.T) {
    const goroutines = 100
    const iterations = 1000

    var wg sync.WaitGroup
    errors := make(chan error, goroutines*iterations)

    for g := 0; g < goroutines; g++ {
        wg.Add(1)
        go func() {
            defer wg.Done()

            for i := 0; i < iterations; i++ {
                msg := &TestMessage{ID: i, Data: fmt.Sprintf("test-%d", i)}

                encoded, err := Marshal(msg)
                if err != nil {
                    errors <- err
                    return
                }

                var decoded TestMessage
                if err := Unmarshal(encoded, &decoded); err != nil {
                    errors <- err
                    return
                }

                if !reflect.DeepEqual(msg, &decoded) {
                    errors <- fmt.Errorf("round-trip mismatch")
                    return
                }
            }
        }()
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Error(err)
    }
}
```

---

## Phase 5: Developer Experience

### P5-1: Wire Format Version Detection

**Priority**: MEDIUM (from ROADMAP)
**Effort**: 1 day

```go
// pkg/cramberry/version.go

const (
    WireFormatV1 byte = 0x01
    WireFormatV2 byte = 0x02
)

// DetectVersion returns the wire format version of the given data.
// Returns 0 if the version cannot be determined (legacy format).
func DetectVersion(data []byte) byte {
    if len(data) == 0 {
        return 0
    }

    // Check for version prefix
    switch data[0] {
    case WireFormatV1, WireFormatV2:
        return data[0]
    default:
        // Legacy format (no version prefix)
        return 0
    }
}

// ReaderWithVersion creates a Reader configured for the appropriate version.
func ReaderWithVersion(data []byte) *Reader {
    version := DetectVersion(data)

    var offset int
    if version != 0 {
        offset = 1 // Skip version byte
    }

    r := NewReader(data[offset:])
    r.version = version
    return r
}
```

---

### P5-2: Schema Compatibility Checker

**Priority**: HIGH (from ROADMAP)
**Effort**: 2 days

```go
// pkg/schema/compat.go

// BreakingChange represents an incompatible schema change.
type BreakingChange struct {
    Type     BreakingChangeType
    Message  string
    Location string
}

type BreakingChangeType int

const (
    FieldNumberReused BreakingChangeType = iota
    FieldTypeChanged
    RequiredFieldAdded
    RequiredFieldRemoved
    EnumValueReused
)

// CheckCompatibility compares two schemas and returns breaking changes.
func CheckCompatibility(old, new *Schema) []BreakingChange {
    var changes []BreakingChange

    for name, oldMsg := range old.Messages {
        newMsg, exists := new.Messages[name]
        if !exists {
            // Message removed - might be breaking
            continue
        }

        changes = append(changes, checkMessageCompat(oldMsg, newMsg)...)
    }

    return changes
}

func checkMessageCompat(old, new *Message) []BreakingChange {
    var changes []BreakingChange

    // Check for field number reuse with different type
    oldFields := make(map[int]*Field)
    for _, f := range old.Fields {
        oldFields[f.Number] = f
    }

    for _, newF := range new.Fields {
        if oldF, exists := oldFields[newF.Number]; exists {
            if oldF.Type != newF.Type {
                changes = append(changes, BreakingChange{
                    Type:     FieldTypeChanged,
                    Message:  fmt.Sprintf("field %d type changed from %s to %s", newF.Number, oldF.Type, newF.Type),
                    Location: newF.Name,
                })
            }
        }
    }

    return changes
}
```

CLI integration:

```bash
$ cramberry check-compat old.cram new.cram
Breaking changes detected:
  - Field 5 type changed from int32 to int64 (Message.field_name)
  - Field 10 number reused with different type (OtherMessage)
```

---

## Testing Requirements

### Unit Test Coverage Targets

| Package | Current | Target |
|---------|---------|--------|
| pkg/cramberry | 85% | 95% |
| internal/wire | 90% | 95% |
| pkg/schema | 80% | 90% |
| pkg/codegen | 75% | 85% |
| typescript | 70% | 85% |
| rust | 70% | 85% |

### Security Test Requirements

- [ ] All integer overflow paths tested
- [ ] Zero-copy lifetime violations detected
- [ ] Varint overflow detection consistent
- [ ] Array bounds enforced in unmarshal
- [ ] NaN canonicalization verified
- [ ] Cross-language encoding identical

### Fuzzing Requirements

- [ ] 24-hour continuous fuzzing with no crashes
- [ ] Schema parser fuzz clean
- [ ] Varint decoder fuzz clean
- [ ] Message decoder fuzz clean
- [ ] Cross-language interop fuzz clean

---

## Release Checklist

### Pre-Release (Must Complete)

- [ ] All CRITICAL issues fixed (C1, C2)
- [ ] All HIGH issues fixed (H1-H6)
- [ ] Fuzz testing runs clean for 24h
- [ ] Cross-language tests pass
- [ ] Concurrent stress tests pass
- [ ] No known security vulnerabilities
- [ ] Documentation updated
- [ ] CHANGELOG updated
- [ ] Breaking changes documented

### Release Process

1. [ ] Create release branch `release/v1.1.0`
2. [ ] Run full test suite (Go, TypeScript, Rust)
3. [ ] Run fuzzing for 24h
4. [ ] Run cross-language interop tests
5. [ ] Update version constants
6. [ ] Generate CHANGELOG
7. [ ] Tag release `v1.1.0`
8. [ ] Publish npm package (TypeScript)
9. [ ] Publish crate (Rust)
10. [ ] Create GitHub release

### Post-Release

- [ ] Monitor for security reports
- [ ] Address critical bugs in patch releases
- [ ] Collect performance feedback
- [ ] Plan v1.2.0 based on ROADMAP

---

## Summary

Despite Cramberry's excellent architecture and performance, the code review revealed critical security issues that must be addressed before production use. This plan prioritizes:

1. **Security hardening** (Phase 1) - Fix all critical and high-severity issues
2. **Cross-language consistency** (Phase 2) - Ensure identical behavior
3. **Fuzzing and testing** (Phase 4) - Comprehensive security testing

After completing Phases 1-4, Cramberry will be safe for production use in security-critical applications including blockchain consensus systems.

---

*Last Updated: January 2026*
*Target Release: v1.1.0*
