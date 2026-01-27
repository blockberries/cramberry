# Cramberry Remediation & Development Plan

This document consolidates critical security fixes, technical debt remediation, and the development roadmap into a single prioritized action plan.

**Status:** Pre-release codebase - aggressive changes permitted.

---

## Executive Summary

A comprehensive security review identified **14 issues** (3 critical, 8 high, 3 medium). The most severe vulnerabilities are in the V2 wire format implementation, which lacks the safety checks present in the V1 primitives.

**Key decisions:**
1. **Remove V1 wire format entirely** - The codebase hasn't been released; V1 adds complexity without benefit
2. **Harden V2 to match V1 rigor** - Apply all overflow/bounds checks from `internal/wire/` to V2
3. **Block release until critical issues fixed** - No release until Phase 1 complete

---

## Phase 0: Immediate Blockers (CRITICAL) ✅ COMPLETED

**Timeline:** Must complete before any release
**Goal:** Fix exploitable security vulnerabilities
**Status:** All critical security fixes implemented and tested.

### 0.1 Fix Integer Overflow in V2 Wire Format

**Issue:** `ReadCompactTag` and `DecodeCompactTag` have unbounded varint loops that can cause:
- Shift overflow (shift > 63 causes undefined behavior)
- Integer overflow on 32-bit systems
- DoS via CPU exhaustion

**Location:** `pkg/cramberry/wire_v2.go:106-117, 200-212`

**Fix:**
```go
// Add these constants at top of wire_v2.go
const (
    maxCompactVarintBytes = 10  // Match internal/wire
)

// In ReadCompactTag, replace lines 200-212:
var shift uint
for i := 0; i < maxCompactVarintBytes && r.pos < len(r.data); i++ {
    b := r.data[r.pos]
    r.pos++

    // At byte 9 (index 9), check for overflow
    if i == 9 {
        if b > 1 {
            r.setError(ErrOverflow)
            return 0, 0
        }
    }

    fieldNum |= int(b&0x7F) << shift
    if b < 0x80 {
        return fieldNum, wireType
    }
    shift += 7
}
r.setError(ErrInvalidVarint)
return 0, 0
```

**Tests required:**
- [ ] Varint with 11+ bytes → error
- [ ] Varint byte 10 with value > 1 → error
- [ ] Max valid uint64 field number → success
- [ ] Fuzz test with random continuation bytes

---

### 0.2 Fix Length Overflow in SkipValueV2

**Issue:** `uint64` → `int` conversion can produce negative values, bypassing bounds checks.

**Location:** `pkg/cramberry/wire_v2.go:238-241`

**Fix:**
```go
case WireTypeV2Bytes:
    length := r.ReadUvarint()
    if length > uint64(len(r.data)-r.pos) {
        r.setError(ErrUnexpectedEOF)
        return
    }
    r.pos += int(length)
```

**Tests required:**
- [ ] Length = MaxUint64 → error (not negative position)
- [ ] Length > remaining → ErrUnexpectedEOF
- [ ] Valid length → success

---

### 0.3 Fix Packed Slice/Array Overflow

**Issue:** `decodePackedSlice` and `decodePackedArray` use `int(r.ReadUvarint())` without overflow protection.

**Location:** `pkg/cramberry/unmarshal.go:145, 225`

**Fix:** Use the protected `ReadArrayHeader()` instead:
```go
func decodePackedSlice(r *Reader, v reflect.Value) error {
    n := r.ReadArrayHeader()  // Uses overflow-protected version
    if r.Err() != nil {
        return r.Err()
    }
    // ... rest unchanged
}
```

**Tests required:**
- [ ] Packed slice with length > MaxInt → error
- [ ] Packed array with length > MaxInt → error

---

## Phase 1: V1 Wire Format Removal ✅ COMPLETED

**Timeline:** 1-2 weeks
**Goal:** Eliminate V1 complexity and establish V2 as the sole format
**Status:** V1 code removed, documentation updated, tests passing.

### 1.1 Remove V1 Wire Types and Options

**Files to modify:**
- `pkg/cramberry/types.go` - Remove `WireVersionV1`, `V1Options`
- `pkg/cramberry/marshal.go` - Remove `encodeStructV1`, V1 dispatch
- `pkg/cramberry/unmarshal.go` - Remove `decodeStructV1`, V1 dispatch
- `pkg/cramberry/reader.go` - Remove V1-specific methods
- `pkg/cramberry/writer.go` - Remove V1-specific methods

**Tasks:**
- [ ] Remove `WireVersionV1` constant
- [ ] Remove `V1Options` preset
- [ ] Remove `WireVersion` field from `Options` (V2 is now implicit)
- [ ] Delete `encodeStructV1()` function
- [ ] Delete `decodeStructV1()` function
- [ ] Remove V1/V2 dispatch logic in `encodeStruct()` and `decodeStruct()`
- [ ] Update `DefaultOptions` to remove `WireVersion` field
- [ ] Remove all `// Deprecated: Use V2` comments (V2 is now the only option)

### 1.2 Remove V1 Tag Encoding

**Files to modify:**
- `internal/wire/tag.go` - Simplify to V2-only
- `pkg/cramberry/wire_v2.go` - Rename to `wire.go`, make canonical

**Tasks:**
- [ ] Move V2 compact tag logic to `internal/wire/tag.go`
- [ ] Add all safety checks from `internal/wire/varint.go` to compact tag decoding
- [ ] Delete `pkg/cramberry/wire_v2.go` (absorbed into internal/wire)
- [ ] Update all imports

### 1.3 Remove V1 Tests

**Files to modify:**
- `pkg/cramberry/marshal_test.go`
- `pkg/cramberry/unmarshal_test.go`
- `pkg/cramberry/reader_test.go`
- `pkg/cramberry/writer_test.go`

**Tasks:**
- [ ] Delete all `TestV1*` test functions
- [ ] Delete V1 test fixtures
- [ ] Update roundtrip tests to only use V2

### 1.4 Update Documentation

**Files to modify:**
- `README.md`
- `ARCHITECTURE.md`
- `CLAUDE.md`
- `docs/MIGRATION.md`
- `docs/SCHEMA_LANGUAGE.md`

**Tasks:**
- [ ] Remove all V1 wire format references
- [ ] Remove V1Options documentation
- [ ] Update wire format documentation to describe V2 as "the wire format"
- [ ] Remove deprecation warnings (nothing to deprecate now)

### 1.5 Update Cross-Language Runtimes

**Files to modify:**
- `typescript/src/*.ts`
- `rust/src/*.rs`

**Tasks:**
- [ ] Verify TypeScript only implements V2 format
- [ ] Verify Rust only implements V2 format
- [ ] Remove any V1 compatibility code
- [ ] Update inline documentation

---

## Phase 2: V2 Hardening ✅ COMPLETED

**Timeline:** 1-2 weeks (can parallel with Phase 1)
**Goal:** Apply V1-level rigor to all V2 code paths
**Status:** Depth limiting, NaN handling, and isZeroValue recursion limit implemented.

### 2.1 Centralize Varint Decoding

**Problem:** Multiple varint implementations with inconsistent safety checks.

**Solution:** Single implementation in `internal/wire/varint.go` used everywhere.

**Tasks:**
- [ ] Audit all varint decode sites:
  - `internal/wire/varint.go` ✅ (reference implementation)
  - `pkg/cramberry/wire_v2.go` ❌ (unbounded)
  - `pkg/cramberry/stream.go` ✅ (has limit)
  - `pkg/cramberry/reader.go` (verify)
- [ ] Replace all inline varint loops with calls to `wire.DecodeUvarint()`
- [ ] Add `wire.DecodeUvarintFromReader(io.ByteReader)` for streaming
- [ ] Delete duplicate implementations

### 2.2 Enforce Depth Limiting in Recursive Paths

**Problem:** `enterNested()` not called in `decodeValue`/`encodeValue` recursion.

**Location:** `pkg/cramberry/unmarshal.go`, `pkg/cramberry/marshal.go`

**Fix:**
```go
func decodeStruct(r *Reader, v reflect.Value) error {
    if !r.enterNested() {  // ADD THIS
        return r.Err()
    }
    defer r.exitNested()   // ADD THIS

    // ... existing logic
}

func decodeSlice(r *Reader, v reflect.Value) error {
    if !r.enterNested() {  // ADD THIS
        return r.Err()
    }
    defer r.exitNested()   // ADD THIS

    // ... existing logic
}
// Same for decodeMap, decodeArray, encodeStruct, encodeSlice, encodeMap
```

**Tests required:**
- [ ] Struct nested beyond MaxDepth → ErrMaxDepthExceeded
- [ ] Slice nested beyond MaxDepth → ErrMaxDepthExceeded
- [ ] Map nested beyond MaxDepth → ErrMaxDepthExceeded

### 2.3 Add Depth Limiting to isZeroValue

**Problem:** `isZeroValue()` recurses without depth limit.

**Location:** `pkg/cramberry/marshal.go:577-609`

**Fix:**
```go
func isZeroValue(v reflect.Value) bool {
    return isZeroValueWithDepth(v, 0)
}

func isZeroValueWithDepth(v reflect.Value, depth int) bool {
    if depth > 100 {  // Reasonable limit for zero-value checking
        return false  // Assume non-zero if too deep
    }

    switch v.Kind() {
    // ... cases ...
    case reflect.Struct:
        for i := 0; i < v.NumField(); i++ {
            if !isZeroValueWithDepth(v.Field(i), depth+1) {
                return false
            }
        }
        return true
    }
}
```

### 2.4 Fix NaN Handling in Map Sorting

**Problem:** NaN comparisons break sort invariants, causing non-deterministic output.

**Location:** `pkg/cramberry/marshal.go:631-634`

**Fix:**
```go
case reflect.Float32, reflect.Float64:
    sort.Slice(keys, func(i, j int) bool {
        fi, fj := keys[i].Float(), keys[j].Float()
        // Handle NaN: treat all NaNs as equal, sort before other values
        iNaN, jNaN := math.IsNaN(fi), math.IsNaN(fj)
        if iNaN && jNaN {
            return false  // Equal
        }
        if iNaN {
            return true   // NaN sorts first
        }
        if jNaN {
            return false
        }
        return fi < fj
    })
```

**Tests required:**
- [ ] Map with NaN keys produces deterministic output
- [ ] Multiple NaN keys handled correctly
- [ ] Mixed NaN and regular float keys sorted correctly

### 2.5 Replace Unsafe Float Conversion

**Problem:** Uses `unsafe.Pointer` instead of `math.Float32frombits`.

**Location:** `pkg/cramberry/reader.go:905, 937`

**Fix:**
```go
// Before:
result[i] = *(*float32)(unsafe.Pointer(&v))

// After:
result[i] = math.Float32frombits(v)
```

**Benefit:** Portable, idiomatic, no unsafe package needed.

### 2.6 Harden Rust Runtime

**Problem:** Unchecked `u32 → usize` casts.

**Location:** `rust/src/reader.rs:198,205,220,230`

**Fix:** Add bounds checks or use `TryFrom`:
```rust
let length = self.read_varint()?;
let length_usize = usize::try_from(length)
    .map_err(|_| Error::LengthOverflow)?;
if length_usize > MAX_MESSAGE_SIZE {
    return Err(Error::MessageTooLarge);
}
```

### 2.7 Harden TypeScript Runtime

**Problem:** 32-bit varint overflow check is incomplete.

**Location:** `typescript/src/reader.ts:87-105`

**Fix:** Tighten the check at byte 5:
```typescript
if (i === 4) {
    // Byte 5 can only contribute 4 bits (28 + 4 = 32)
    // High 4 bits must be 0, and no continuation
    if ((b & 0xf0) !== 0 || (b & 0x80) !== 0) {
        throw new DecodeError("Varint overflow: value exceeds 32 bits");
    }
}
```

---

## Phase 3: Code Generator Fixes ✅ COMPLETED

**Timeline:** 1 week
**Goal:** Fix correctness bugs in generated code
**Status:** TODO placeholders replaced, unknown field handling fixed with proper SkipValueV2.

### 3.1 Remove TODO Placeholders

**Problem:** Code generator emits `// TODO: encode/decode` comments instead of code.

**Location:** `pkg/codegen/go_generator.go:222,259,268,371,408,417`

**Tasks:**
- [ ] Identify all type combinations that trigger TODOs
- [ ] Implement proper encoding for each
- [ ] Add tests for all generated code paths
- [ ] Add CI check that generated code contains no TODOs

### 3.2 Fix Unknown Field Handling in Generated Decoders

**Problem:** Generated `decodeFrom` breaks loop on unknown fields instead of skipping.

**Location:** `pkg/codegen/go_generator.go:814-817`

**Fix:**
```go
default:
    // Skip unknown field based on wire type
    r.SkipValueV2(wireType)
```

But this requires the wire type to be available. Better approach:

```go
// Change ReadCompactTag to also store wireType in a field
fieldNum, wireType := r.ReadCompactTag()
if fieldNum == 0 {
    break
}
switch fieldNum {
// ... known fields ...
default:
    r.SkipValueV2(wireType)  // Now we have wireType
}
```

### 3.3 Add Forward Compatibility Tests ✅ COMPLETED

**Tasks:**
- [x] Create V1 and V2 schema types in tests
- [x] Verify V1 decoder can decode V2 data (skips unknown fields)
- [x] Test all wire types for unknown field skipping
- [x] Test unknown fields at various positions (start, middle, end)
- [x] Verify strict mode rejects unknown fields

**Implementation:** Comprehensive test suite in `pkg/cramberry/forward_compat_test.go`:
- Basic types, nested messages, slices
- All wire types (varint, fixed32, fixed64, bytes)
- Field ordering scenarios
- Round-trip verification
- Large unknown fields

---

## Phase 4: API Cleanup ✅ COMPLETED

**Timeline:** 1 week
**Goal:** Remove deprecated APIs and footguns
**Status:** All items complete (4.1, 4.2, 4.3).

### 4.1 Remove Deprecated Registration Functions ✅ COMPLETED

**Problem:** `MustRegister` and `MustRegisterWithID` panic on duplicates.

**Location:** `pkg/cramberry/registry.go:349-375`

**Tasks:**
- [x] Delete `MustRegister[T]()` function
- [x] Delete `MustRegisterWithID[T](id)` function
- [x] Update all examples to use `RegisterOrGet`
- [x] Update all tests to use `RegisterOrGet`
- [x] Search codebase for any remaining uses

### 4.2 ZeroCopy Safety Review ✅ COMPLETED

**Problem:** `ZeroCopyString.String()` panics on invalid access.

**Options:**
1. **Keep panic** (current) - Documents as intentional, add `MustString()` alias
2. **Return error** - Change to `func (zcs ZeroCopyString) String() (string, error)`
3. **Return empty** - Log warning, return empty string

**Decision:** Keep `String()` as panicking for `fmt.Stringer` compatibility, but add ergonomic alternatives:
- [x] Add `MustString()` / `MustBytes()` as explicit aliases
- [x] Add `StringOrEmpty()` / `BytesOrNil()` that return defaults on invalid
- [x] Add `TryString()` / `TryBytes()` for explicit (value, ok) checking
- [x] Update documentation prominently

### 4.3 Validate Field Number Uniqueness ✅ COMPLETED

**Problem:** Duplicate field numbers silently overwrite.

**Location:** `pkg/cramberry/marshal.go:439+` (`getStructInfo`)

**Implementation:** Added field number tracking to detect duplicates at struct registration time. Panics with clear error message showing the conflicting field names.

```go
// Track seen field numbers for uniqueness validation
seenFieldNums := make(map[int]string)

// In the field parsing loop:
if existingField, ok := seenFieldNums[fi.num]; ok {
    panic(fmt.Sprintf("cramberry: duplicate field number %d in %s (fields %q and %q)",
        fi.num, t.Name(), existingField, f.Name))
}
seenFieldNums[fi.num] = f.Name
```

---

## Phase 5: Testing & Validation

**Timeline:** Ongoing
**Goal:** Comprehensive test coverage for all fixes

### 5.1 Security Test Suite

**New file:** `pkg/cramberry/security_test.go`

**Tests:**
- [ ] Varint overflow (10+ bytes)
- [ ] Varint shift overflow (shift > 63)
- [ ] Length overflow (uint64 max)
- [ ] Negative length after conversion
- [ ] Deep nesting (stack overflow prevention)
- [ ] NaN map keys (determinism)
- [ ] Large allocations (DoS prevention)
- [ ] Malformed UTF-8 strings
- [ ] Field number collisions

### 5.2 Fuzz Testing Expansion

**Tasks:**
- [ ] Add fuzz target for V2 compact tag decoding
- [ ] Add fuzz target for packed array decoding
- [ ] Add fuzz target for map with float keys
- [ ] Run fuzz tests in CI for 10+ minutes

### 5.3 Cross-Language Conformance

**Tasks:**
- [ ] Add test vectors for all edge cases
- [ ] Verify Go, TypeScript, Rust produce identical output
- [ ] Add CI job that runs cross-language tests

---

## Phase 6: Documentation Update

**Timeline:** 1 week
**Goal:** Accurate documentation reflecting changes

### 6.1 Update ARCHITECTURE.md

- [ ] Remove all V1 references
- [ ] Document V2 wire format as canonical
- [ ] Add security considerations section
- [ ] Document depth limiting behavior

### 6.2 Update README.md

- [ ] Remove V1Options mention
- [ ] Update wire format description
- [ ] Add security best practices section

### 6.3 Update CLAUDE.md

- [ ] Remove V1 wire type table
- [ ] Update struct tag documentation
- [ ] Remove V1Options from examples

### 6.4 Create SECURITY.md Updates

- [ ] Document all resource limits
- [ ] Explain SecureLimits usage
- [ ] Add threat model section
- [ ] Document ZeroCopy safety model

---

## Phase 7: Roadmap Integration

The following items from the original ROADMAP.md are retained and reprioritized:

### Retained (Reprioritized)

| Original | New Priority | Rationale |
|----------|--------------|-----------|
| D1: Improved Error Messages | P2 | Helps users debug issues |
| D2: Schema Linting | P3 | Prevents mistakes |
| G1: TypeScript Generator | P4 | After Go generator fixed |
| G2: Rust Generator | P4 | After Go generator fixed |
| E3: Testing Framework | P1 | Critical for validation |

### Deferred

| Item | Reason |
|------|--------|
| P2: SIMD Acceleration | Focus on correctness first |
| P3: Arena Allocator | Not needed until perf-critical |
| F1: gRPC Integration | Major feature, post-stabilization |
| Wire Format V3 | Need V2 stable first |
| A2: Zero-Knowledge Proofs | Research item |

### Removed

| Item | Reason |
|------|--------|
| V1 Deprecation Timeline | V1 being removed entirely |
| Legacy Registration Deprecation | Functions being deleted |

---

## Implementation Order

```
Week 1:
├── Phase 0: Critical Security Fixes (BLOCKING)
│   ├── 0.1 ReadCompactTag overflow
│   ├── 0.2 SkipValueV2 overflow
│   └── 0.3 Packed slice/array overflow
└── Phase 5.1: Security test suite

Week 2:
├── Phase 1: V1 Removal
│   ├── 1.1 Remove V1 types/options
│   ├── 1.2 Remove V1 tag encoding
│   ├── 1.3 Remove V1 tests
│   └── 1.4 Update documentation
└── Phase 2.1: Centralize varint decoding

Week 3:
├── Phase 2: V2 Hardening (continued)
│   ├── 2.2 Depth limiting in recursion
│   ├── 2.3 isZeroValue depth limit
│   ├── 2.4 NaN map sorting
│   └── 2.5 Replace unsafe float conversion
└── Phase 2.6-2.7: Rust/TypeScript hardening

Week 4:
├── Phase 3: Code Generator Fixes
│   ├── 3.1 Remove TODO placeholders
│   ├── 3.2 Fix unknown field handling
│   └── 3.3 Forward compatibility tests
└── Phase 4: API Cleanup

Week 5:
├── Phase 5: Testing & Validation (completion)
├── Phase 6: Documentation Update
└── Final review and release preparation
```

---

## Success Criteria

Before release:
- [ ] All Phase 0 issues fixed and tested
- [ ] Zero V1 references in codebase
- [ ] All security tests passing
- [ ] Fuzz tests run for 1+ hour with no crashes
- [ ] Cross-language conformance tests passing
- [ ] Documentation updated and reviewed
- [ ] `make check` passes with no warnings

---

## Appendix: Issue Reference

| ID | Severity | Description | Phase |
|----|----------|-------------|-------|
| SEC-01 | CRITICAL | ReadCompactTag unbounded shift | 0.1 |
| SEC-02 | CRITICAL | DecodeCompactTag unbounded shift | 0.1 |
| SEC-03 | CRITICAL | SkipValueV2 length overflow | 0.2 |
| SEC-04 | HIGH | decodePackedSlice overflow | 0.3 |
| SEC-05 | HIGH | decodePackedArray overflow | 0.3 |
| SEC-06 | HIGH | Depth limit not enforced | 2.2 |
| SEC-07 | HIGH | isZeroValue no recursion limit | 2.3 |
| SEC-08 | HIGH | NaN map sorting non-determinism | 2.4 |
| SEC-09 | HIGH | ZeroCopy panic footgun | 4.2 |
| SEC-10 | HIGH | Code generator TODOs | 3.1 |
| SEC-11 | HIGH | Generated decoder breaks on unknown | 3.2 |
| SEC-12 | MEDIUM | Unsafe pointer for floats | 2.5 |
| SEC-13 | MEDIUM | Rust unchecked u32→usize | 2.6 |
| SEC-14 | MEDIUM | TypeScript varint check | 2.7 |
