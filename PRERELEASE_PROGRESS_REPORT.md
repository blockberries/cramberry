# Pre-release Progress Report

This document tracks the implementation progress of the Cramberry remediation plan.

---

## Phase 0: Security Hardening - COMPLETED

**Date:** 2026-01-27

### 0.1 Fix ReadCompactTag Unbounded Varint

**Files modified:**
- `pkg/cramberry/wire_v2.go` - Added 10-byte limit and overflow checks to ReadCompactTag

**Implementation:**
- Added import for `internal/wire` package to use `MaxVarintLen64` constant
- Limited varint loop to 10 iterations maximum
- Added byte-9 overflow check (value must be ≤ 1)
- Returns `ErrInvalidVarint` for too many bytes
- Returns `ErrOverflow` for value overflow

### 0.2 Fix DecodeCompactTag Unbounded Varint

**Files modified:**
- `pkg/cramberry/wire_v2.go` - Same fixes applied to standalone DecodeCompactTag

**Implementation:**
- Mirror implementation of ReadCompactTag fixes
- Returns (0, 0, 0) on invalid input

### 0.3 Fix SkipValueV2 Length Overflow

**Files modified:**
- `pkg/cramberry/wire_v2.go` - Fixed length overflow in WireTypeV2Bytes and WireTypeV2Varint cases

**Implementation:**
- For WireTypeV2Bytes: Compare length against remaining bytes before conversion
- For WireTypeV2Varint: Limit skip loop to 10 iterations with proper termination
- Returns ErrUnexpectedEOF for truncated data
- Returns ErrInvalidVarint for malformed varints

### 0.4 Fix decodePackedSlice/Array Overflow

**Files modified:**
- `pkg/cramberry/unmarshal.go` - Changed direct varint read to use ReadArrayHeader

**Implementation:**
- Replaced `int(r.ReadUvarint())` with `r.ReadArrayHeader()` which includes:
  - Overflow protection (uint64 → int)
  - MaxArrayLength limit checking
  - Proper error propagation

### 0.5 Security Test Suite

**Files created:**
- `pkg/cramberry/security_test.go` - Comprehensive security test coverage

**Test coverage:**
- `TestSecurityVarintOverflow` - Tests 11-byte varint, MaxUint64, byte-10 overflow
- `TestSecurityCompactTagVarintOverflow` - Tests extended tag format overflow
- `TestSecuritySkipValueV2LengthOverflow` - Tests MaxUint64 length, truncated data
- `TestSecuritySkipVarintOverflow` - Tests varint skip with too many bytes
- `TestSecurityPackedSliceOverflow` - Tests length > MaxInt and limit enforcement
- `TestSecurityDepthLimiting` - Documents Phase 2 work needed
- `TestSecurityNaNMapKeys` - Documents Phase 2 work needed
- `TestSecurityResourceLimits` - Tests all limit types
- `TestSecurityMalformedInput` - Tests truncated/invalid input handling
- `TestSecurityEdgeCases` - Tests empty input, zero-length collections, max field numbers

---

## Phase 1: V1 Wire Format Removal - COMPLETED

**Date:** 2026-01-27

### 1.1 Remove V1 Wire Types and Options

**Files modified:**
- `pkg/cramberry/types.go`
  - Removed `WireVersion` type and constants (`WireVersionV1`, `WireVersionV2`)
  - Removed `WireVersion` field from `Options` struct
  - Removed `V1Options` preset
  - Updated all option presets to remove WireVersion references

### 1.2 Remove V1 Marshal/Unmarshal Code

**Files modified:**
- `pkg/cramberry/marshal.go`
  - Removed `encodeStructV1` function
  - Removed V1/V2 dispatch logic from `encodeStruct`
  - Removed unused `getWireType` function (was V1-only)

- `pkg/cramberry/unmarshal.go`
  - Removed `decodeStructV1` function
  - Removed `sizeStructV1` function
  - Removed V1/V2 dispatch logic from `decodeStruct` and `sizeStruct`

### 1.3 Update Tests

**Files modified:**
- `pkg/cramberry/wire_v2_test.go`
  - Replaced `TestV2VsV1Size` with `TestV2StructEncodeSize` (no V1 comparison)
  - Replaced `TestPackedVsUnpackedSize` with `TestPackedSliceEncodingSize` (no V1 comparison)

- `pkg/cramberry/security_test.go`
  - Removed all WireVersion references from Options initialization

### 1.4 Update Documentation

**Files modified:**
- `README.md` - Removed V1Options example and from presets table
- `ARCHITECTURE.md` - Removed WireVersion from Options struct, removed V1 Wire Format section
- `CHANGELOG.md` - Removed V1 wire format mentions, removed V1Options deprecation
- `docs/MIGRATION.md` - Removed entire "Wire Format Migration (V1 to V2)" section
- `docs/CONTRIBUTING.md` - Removed V1Options deprecation notice
- `ROADMAP.md` - Marked S1 and S2 items as completed
- `REMEDIATION_PLAN.md` - Marked Phase 0 and Phase 1 as completed

### Test Summary

All 163 tests pass with race detection enabled:
- `pkg/cramberry`: 72.6% coverage
- `internal/wire`: 94.3% coverage
- `pkg/schema`: 80.5% coverage
- `pkg/codegen`: 72.5% coverage
- `pkg/extract`: 82.8% coverage

---

## Phase 2: V2 Hardening - COMPLETED

**Date:** 2026-01-27

### 2.1 Add Depth Limiting to Recursive Encode/Decode

**Files modified:**
- `pkg/cramberry/marshal.go` - Added enterNested/exitNested calls
- `pkg/cramberry/unmarshal.go` - Added enterNested/exitNested calls

**Implementation:**
- `encodeStruct`, `encodeSlice`, `encodeArray`, `encodeMap` now call `w.enterNested()` at start and `w.exitNested()` on exit
- `decodeStruct`, `decodeSlice`, `decodeArray`, `decodeMap` now call `r.enterNested()` at start and `r.exitNested()` on exit
- Packed encoding/decoding paths skip depth tracking (primitives only, no recursion)

**Tests added:**
- `TestSecurityDepthLimiting/DeepNestedStructEncode` - Verifies encode fails at depth limit
- `TestSecurityDepthLimiting/DeepNestedStructDecode` - Verifies decode fails at depth limit
- `TestSecurityDepthLimiting/DeepNestedSliceEncode` - Verifies slice depth limiting
- `TestSecurityDepthLimiting/DeepNestedMapEncode` - Verifies map depth limiting

### 2.2 Fix NaN Handling in Map Sorting

**Files modified:**
- `pkg/cramberry/marshal.go` - Added `compareFloatKeys` function

**Implementation:**
```go
func compareFloatKeys(a, b float64) bool {
    // NaN values sort after everything else
    // Different NaN bit patterns compared by raw bits for full determinism
    // -0.0 and +0.0 treated as equal
}
```

**Tests added:**
- `TestSecurityNaNMapKeys/NaNKeyDeterminism` - Multiple encodes produce identical output
- `TestSecurityNaNMapKeys/NaNSortsAfterInfinity` - NaN sorts after +Inf
- `TestSecurityNaNMapKeys/NegativeZeroEqualsPositiveZero` - -0 and +0 handled correctly
- `TestSecurityNaNMapKeys/MultipleNaNValues` - Multiple NaN keys handled deterministically

### 2.3 Add Recursion Limit to isZeroValue

**Files modified:**
- `pkg/cramberry/marshal.go` - Refactored isZeroValue with depth tracking

**Implementation:**
- Added `maxZeroValueDepth = 100` constant
- Created `isZeroValueWithDepth(v reflect.Value, depth int)` helper
- If depth exceeds limit, returns false (conservative: encode the field)
- Original `isZeroValue` calls helper with depth=0

### Test Summary

All 173 tests pass with race detection enabled:
- `pkg/cramberry`: 72.9% coverage (up from 72.6%)
- Coverage increased due to new security test paths

---

## Phase 3: Code Generator Fixes - COMPLETED

**Date:** 2026-01-27

### 3.1 Remove TODO Placeholders from Generated Code

**Files modified:**
- `pkg/codegen/go_generator.go` - Multiple function updates

**Implementation:**
- `encodeValueV2`: Added `PointerType` case, changed default to informative comment
- `encodeScalarV2`: Changed default from TODO to informative comment
- `encodePackedElementV2`: Changed default from TODO to informative comment
- `decodeValueV2`: Added `PointerType` case with proper allocation and decoding
- `decodeScalarV2`: Changed default from TODO to informative comment
- `decodePackedElementV2`: Changed default from TODO to informative comment

**PointerType handling:**
```go
// Encode: recursively encode the underlying element
case *schema.PointerType:
    return c.encodeValueV2(typ.Element, varName, true)

// Decode: allocate and decode into new value
case *schema.PointerType:
    elemType := c.goTypeInternal(typ.Element, false)
    return fmt.Sprintf(`{
        var v %s
        %s
        %s = &v
    }`, elemType, c.decodeValueV2(typ.Element, "v"), varName)
```

### 3.2 Fix Unknown Field Handling in Generated Decoder

**Files modified:**
- `pkg/codegen/go_generator.go` - Template update for decodeFrom

**Previous behavior:**
```go
fieldNum, _ := r.ReadCompactTag()  // Wire type discarded
// ...
default:
    // Skip unknown field - read wire type would have been needed
    // For now, just break as we can't determine how to skip
    break  // Incorrectly breaks, continues loop without skipping
```

**Fixed behavior:**
```go
fieldNum, wireType := r.ReadCompactTag()  // Wire type preserved
// ...
default:
    // Skip unknown field for forward compatibility
    r.SkipValueV2(wireType)  // Properly skips unknown field data
```

### Test Summary

All code generator tests pass:
- `pkg/codegen`: 72.6% coverage

---

## Phase 4: API Cleanup - COMPLETED (4.1 & 4.2)

**Date:** 2026-01-27

### 4.1 Remove Deprecated MustRegister Functions

**Files modified:**
- `pkg/cramberry/registry.go` - Removed MustRegister and MustRegisterWithID functions
- `pkg/cramberry/registry_test.go` - Updated tests to use RegisterOrGet
- `pkg/cramberry/marshal_test.go` - Updated tests to use RegisterOrGet
- `examples/polymorphic/main.go` - Updated to use RegisterOrGet

**Implementation:**
- Deleted `MustRegister[T]()` function entirely
- Deleted `MustRegisterWithID[T](id TypeID)` function entirely
- Updated all test files to use `RegisterOrGet` instead
- Updated example code to demonstrate idempotent registration

**Rationale:**
- `MustRegister` panics on duplicate registration, which can crash production services
- `RegisterOrGet` is idempotent and safe to call multiple times
- No need for Must* variants when idempotent alternatives exist

### 4.2 Improve ZeroCopy API Ergonomics

**Files modified:**
- `pkg/cramberry/reader.go` - Added ergonomic accessor methods
- `pkg/cramberry/reader_test.go` - Added comprehensive tests for new methods
- `docs/MIGRATION.md` - Updated documentation
- `docs/SECURITY.md` - Updated panic alternatives documentation
- `CHANGELOG.md` - Added new methods to release notes

**Implementation:**

For `ZeroCopyString`:
```go
// Explicit panic alias
func (zcs ZeroCopyString) MustString() string

// Non-panicking alternatives
func (zcs ZeroCopyString) StringOrEmpty() string  // Returns "" if invalid
func (zcs ZeroCopyString) TryString() (string, bool)  // Returns ("", false) if invalid
```

For `ZeroCopyBytes`:
```go
// Explicit panic alias
func (zcb ZeroCopyBytes) MustBytes() []byte

// Non-panicking alternatives
func (zcb ZeroCopyBytes) BytesOrNil() []byte  // Returns nil if invalid
func (zcb ZeroCopyBytes) TryBytes() ([]byte, bool)  // Returns (nil, false) if invalid
func (zcb ZeroCopyBytes) StringOrEmpty() string  // Returns "" if invalid
func (zcb ZeroCopyBytes) TryString() (string, bool)  // Returns ("", false) if invalid
```

**Design decisions:**
- Kept `String()` and `Bytes()` as panicking for `fmt.Stringer` interface compatibility
- Added `MustString()` / `MustBytes()` as explicit aliases that document panic behavior
- Added `*OrEmpty()` / `*OrNil()` for simple non-panicking access
- Added `Try*()` methods for explicit error handling with (value, ok) pattern

### Test Summary

All tests pass including 12 new tests for ergonomic API:
- `pkg/cramberry`: 73.8% coverage
- New tests cover all accessor methods, valid and invalid states, empty values

---

### 4.3 Add Field Number Uniqueness Validation

**Files modified:**
- `pkg/cramberry/marshal.go` - Added validation in getStructInfo
- `pkg/cramberry/marshal_test.go` - Added comprehensive tests

**Implementation:**
- Added `seenFieldNums` map to track field numbers during struct parsing
- After parsing each field's tag, check if field number was already used
- If duplicate detected, panic with clear error message showing both field names
- Validation happens at first use of struct type (cached afterward)

**Test coverage:**
- Duplicate explicit field numbers (panic expected)
- Valid field numbers with gaps (allowed)
- Skipped fields with conflicting numbers (panic expected)

### Test Summary

All tests pass:
- `pkg/cramberry`: 73.9% coverage
- 4 new tests for field number validation

---

### 3.3 Add Forward Compatibility Tests

**Files created:**
- `pkg/cramberry/forward_compat_test.go` - Comprehensive forward compatibility test suite

**Test coverage:**
- **Basic types**: String, int32, float64, bool fields added in V2
- **Slices and maps**: New slice fields, scalar fields after slices
- **Nested messages**: V2 nested messages with extra fields decoded by V1
- **All wire types**: Varint, fixed32, fixed64, bytes unknown fields skipped correctly
- **Field ordering**: Unknown fields at start, middle, end of message
- **Strict mode**: Verifies unknown fields are rejected in strict mode
- **Round-trip**: Confirms V1 re-encoding doesn't include V2 fields
- **Edge cases**: Empty strings, zero values, large unknown fields

### Test Summary

All tests pass:
- `pkg/cramberry`: 74.1% coverage
- 10 new test functions with multiple subtests

---

## Phase 5: Testing & Validation - COMPLETED

**Date:** 2026-01-27

### 5.1 Extended Fuzz Testing

**Test execution summary:**

| Test | Duration | Executions | Crashes |
|------|----------|------------|---------|
| FuzzReaderVarint | 15 min | ~80M | 0 |
| FuzzReaderString | 15 min | ~72M | 0 |
| FuzzWriterReader | 15 min | ~67M | 0 |
| FuzzFloatRoundTrip | 15 min | ~72M | 0 |
| FuzzUnmarshalBytes | 20 min | ~87M | 0 |
| FuzzMarshalRoundTrip | 20 min | ~98M | 0 |
| FuzzSchemaParser | 15 min | ~67M | 0 |
| FuzzLexer | 15 min | ~120M | 0 |
| **Total** | **~2h 10min** | **~663M** | **0** |

**Implementation details:**
- All fuzz tests run with 12 parallel workers
- Tests covered: varint decoding, string handling, reader/writer round-trips, float serialization, unmarshaling, marshaling round-trips, schema parsing, and lexing
- Zero crashes or panics detected across all test targets
- Exceeded the required threshold of 1+ hour with no crashes

**Tests validated:**
- `pkg/cramberry`: FuzzReaderVarint, FuzzReaderString, FuzzWriterReader, FuzzFloatRoundTrip, FuzzUnmarshalBytes, FuzzMarshalRoundTrip
- `pkg/schema`: FuzzSchemaParser, FuzzLexer

---

## Remaining Work

### Cross-language conformance tests
- Verify Go, TypeScript, Rust produce identical output for test vectors
