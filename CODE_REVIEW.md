# Code Review Report

This document tracks issues identified during code review and their resolution status.

## Review Date: 2026-01-29

## Summary

A comprehensive code review was conducted across all Go source files in the Cramberry project. The review focused on bugs, logic errors, security vulnerabilities, performance issues, and code quality.

**All high and medium severity issues have been fixed.** See Resolution History for details.

---

## Resolved Issues

### Issue 1: Ignored Errors in DecodeComplex64/DecodeComplex128

**File:** `internal/wire/fixed.go`
**Lines:** 189-190, 206-207
**Severity:** HIGH
**Status:** FIXED

**Description:** The `DecodeComplex64` and `DecodeComplex128` functions ignored errors returned by `DecodeFloat32` and `DecodeFloat64` using the blank identifier `_`. While the length check at the start of each function should prevent truncated data from reaching these calls, silently ignoring errors violates defensive programming principles.

**Fix Applied:** Properly handle errors from the float decoding functions with early return on error.

---

### Issue 2: Inconsistent Error Return in DecodeTag

**File:** `internal/wire/tag.go`
**Line:** 128
**Severity:** MEDIUM
**Status:** FIXED

**Description:** When `DecodeTag` returns `ErrInvalidWireType`, it returned `n` (bytes consumed) instead of `0` like other error cases. This inconsistency could cause callers to incorrectly advance their read position on error.

**Fix Applied:** Return 0 for bytes consumed on wire type validation error to match other error cases.

---

### Issue 3: Missing Nil Pointer Check in collectEnumValues

**File:** `pkg/extract/collector.go`
**Line:** 225
**Severity:** MEDIUM
**Status:** FIXED

**Description:** The `collectEnumValues` function called `named.Obj().Pkg().Path()` without checking if `Pkg()` returns nil. Edge cases with type aliases or certain builtin types could cause a nil pointer dereference.

**Fix Applied:** Added nil check before calling `.Path()` to match the pattern in `builder.go`.

---

### Issue 4: Cross-Package Enum Wire Type Detection

**Files:** `pkg/codegen/typescript_generator.go`, `pkg/codegen/rust_generator.go`
**Severity:** HIGH
**Status:** FIXED

**Description:** The wire type determination for NamedTypes only checked enums by name without considering the package qualification. A cross-package message named the same as a local enum would incorrectly be treated as an enum.

**Fix Applied:** Added package qualification check - only local types (empty package field) are matched against local enums. Cross-package types are correctly treated as messages. Added documentation noting that cross-package enum detection requires access to imported schemas (future enhancement).

---

### Issue 5: Missing Nil Pointer Checks in collectType

**File:** `pkg/extract/collector.go`
**Lines:** 141, 187, 204, 405
**Severity:** MEDIUM
**Status:** FIXED

**Description:** The `collectType` function called `typeName.Pkg().Name()` without checking if `Pkg()` returns nil. This followed the same pattern as Issue 3. Additionally, the `typeToString` function's qualifier callback could receive a nil package.

**Fix Applied:** Added nil check at the start of `collectType` that extracts the package name safely, then reused this value. Added nil check to the qualifier function in `typeToString`.

---

### Issue 6: Misleading Comment in Varint Overflow Check

**File:** `pkg/cramberry/wire_v2.go`
**Line:** 116
**Severity:** LOW (Documentation)
**Status:** FIXED

**Description:** The comment incorrectly stated "9th varint byte" when the check was actually for the 10th varint byte (index 10 in data, after the tag byte at index 0).

**Fix Applied:** Updated comment to accurately describe the byte position.

---

### Issue 7: Integer Overflow in Generated Decode Code

**File:** `pkg/codegen/go_generator.go`
**Lines:** 318, 334, 342, 371-377, 381-389
**Severity:** HIGH
**Status:** FIXED

**Description:** The Go code generator produced decode functions that converted `ReadUvarint()` results directly to `int` without overflow checking: `n := int(r.ReadUvarint())`. On 32-bit systems, values exceeding `math.MaxInt32` would overflow, potentially causing panics from `make()` with negative capacity or resource exhaustion from large allocations.

**Fix Applied:** Changed generated code to use the overflow-safe `ReadArrayHeader()` and `ReadMapHeader()` methods from the runtime, which include proper overflow detection and limit checking.

---

## Low Priority Issues (Documented, Not Fixing)

### Regex Compilation in Loop

**File:** `pkg/extract/collector.go`
**Line:** 390
**Severity:** LOW (Performance)

The `parseTypeIDFromDoc` function compiles regex patterns on every call. While inefficient for processing many types, this is not on a hot path and the performance impact is negligible for typical use cases.

---

### toSnakeCase ASCII-Only Conversion

**File:** `pkg/extract/builder.go`
**Line:** 389
**Severity:** LOW

The `r + 32` conversion only works correctly for ASCII uppercase letters. Since Go identifiers rarely contain non-ASCII uppercase letters, this is acceptable.

---

### Missing Bounds Check Documentation

**Files:** `internal/wire/varint.go`, `internal/wire/tag.go`
**Severity:** LOW (Documentation)

The `PutUvarint` and `PutTag` functions assume sufficient buffer space. This is documented behavior and callers are expected to ensure adequate buffer size.

---

### Unbounded Recursion in Schema Parser

**File:** `pkg/schema/parser.go`
**Function:** `parseTypeRef()`
**Severity:** LOW

The type reference parser recursively descends for pointer (`*`) and array (`[]`) types without depth limiting. Deeply nested types like `*****[][]...int` could cause stack overflow. This is acceptable because schema files are trusted developer input, not untrusted data. If processing untrusted schemas becomes necessary, a depth limit should be added.

---

## Review Statistics

- **Files Reviewed:** 64 Go source files
- **Critical Issues:** 0
- **High Severity Issues:** 3 (fixed)
- **Medium Severity Issues:** 3 (fixed)
- **Low Severity Issues:** 5 (documented, not fixing)

---

## Resolution History

| Date | Issue | Status | Description |
|------|-------|--------|-------------|
| 2026-01-29 | Issue 1: DecodeComplex error handling | FIXED | Added proper error handling |
| 2026-01-29 | Issue 2: DecodeTag consistency | FIXED | Consistent error return value |
| 2026-01-29 | Issue 3: Nil pointer check | FIXED | Added nil guard |
| 2026-01-29 | Issue 4: Enum wire type detection | FIXED | Package qualification check |
| 2026-01-29 | Issue 5: Nil pointer checks in collectType | FIXED | Defensive nil checks for package names |
| 2026-01-29 | Issue 6: Varint comment correction | FIXED | Documentation fix |
| 2026-01-29 | Issue 7: Integer overflow in codegen | FIXED | Use overflow-safe ReadArrayHeader/ReadMapHeader |
