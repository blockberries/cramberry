# Cramberry Refactoring Document

This document captures findings from a comprehensive code review comparing the implementation against the architecture documentation. It identifies potential issues, missing items, and variations that may need attention.

---

## Table of Contents

1. [Critical Issues](#critical-issues)
2. [Wire Format / Encoding Issues](#wire-format--encoding-issues)
3. [Go Runtime Issues](#go-runtime-issues)
4. [TypeScript Runtime Issues](#typescript-runtime-issues)
5. [Rust Runtime Issues](#rust-runtime-issues)
6. [Schema Parser Issues](#schema-parser-issues)
7. [Code Generation Issues](#code-generation-issues)
8. [Schema Extraction Issues](#schema-extraction-issues)
9. [Cross-Language Compatibility Issues](#cross-language-compatibility-issues)
10. [Test Coverage Gaps](#test-coverage-gaps)
11. [Documentation Gaps](#documentation-gaps)
12. [Performance Considerations](#performance-considerations)
13. [Recommendations](#recommendations)

---

## Critical Issues

### 1. Wire Type Values Inconsistency

**Location:** Multiple files across runtimes

**Issue:** The architecture specifies wire types as:
- 0 = Varint
- 1 = Fixed64
- 2 = Bytes (length-prefixed)
- 5 = Fixed32
- 6 = SVarint (ZigZag signed)
- 7 = TypeRef (polymorphic)

However, wire types 3 and 4 are reserved/undefined. The implementation uses these values consistently across Go, TypeScript, and Rust, but the gap (skipping 3, 4) should be documented.

**Files:**
- `internal/wire/tag.go:11-18`
- `typescript/src/types.ts:4-17`
- `rust/src/types.rs`

**Recommendation:** Add documentation explaining the wire type value assignments and why 3, 4 are skipped.

---

### 2. Deprecated `strings.Title` Usage

**Location:** `pkg/codegen/generator.go:103`

**Issue:** The `ToPascalCase` function uses `strings.Title()` which is deprecated in Go 1.18+ and doesn't handle Unicode properly.

```go
func ToPascalCase(s string) string {
    parts := splitName(s)
    for i, p := range parts {
        parts[i] = strings.Title(strings.ToLower(p)) // Deprecated
    }
    return strings.Join(parts, "")
}
```

**Recommendation:** Replace with `cases.Title(language.English).String()` from `golang.org/x/text/cases`.

---

### 3. Type ID Auto-Assignment Logic Missing

**Location:** `pkg/extract/builder.go:186-189`

**Issue:** Interface implementation type IDs are hardcoded to start at 128:

```go
for i, impl := range iface.Implementations {
    schemaIface.Implementations = append(schemaIface.Implementations, &schema.Implementation{
        Type:   &schema.NamedType{Name: impl.Name},
        TypeID: 128 + i, // Start type IDs at 128
    })
}
```

Per architecture, type IDs should be controllable via `@typeID` annotations and validated for uniqueness. The hardcoded approach conflicts with explicit type IDs from schema definitions.

**Recommendation:**
- Implement proper type ID registry
- Support explicit `@typeID` annotations
- Validate for collisions
- Generate deterministic IDs when not explicitly specified

---

## Wire Format / Encoding Issues

### 4. Missing Fixed-Width Integer Support in High-Level API

**Location:** `pkg/cramberry/writer.go`, `pkg/cramberry/reader.go`

**Issue:** Architecture mentions support for fixed-width integers (`sfixed32`, `sfixed64`) but only unsigned fixed variants are exposed in the Writer/Reader API.

**Missing methods:**
- `WriteSFixed32(int32)`
- `WriteSFixed64(int64)`
- `ReadSFixed32() int32`
- `ReadSFixed64() int64`

**Recommendation:** Add signed fixed-width integer methods for completeness.

---

### 5. Complex Number Encoding Not Documented in Architecture

**Location:** `pkg/cramberry/marshal.go:252-267`

**Issue:** Go implementation supports `complex64` and `complex128` types, but this isn't mentioned in the architecture document. Cross-language compatibility may be affected.

**Current encoding:**
```go
case reflect.Complex64:
    c := rv.Complex()
    w.WriteTag(1, wire.WireFixed32)
    w.WriteFixed32(math.Float32bits(float32(real(c))))
    w.WriteTag(2, wire.WireFixed32)
    w.WriteFixed32(math.Float32bits(float32(imag(c))))
```

**Issues:**
- Not documented in architecture
- TypeScript/Rust runtimes don't support complex numbers
- Field numbers 1, 2 are hardcoded for components

**Recommendation:** Either document complex number support as Go-specific or remove it and suggest struct-based alternative.

---

### 6. Varint Overflow Handling Differences

**Location:** Multiple reader implementations

**Issue:** Different runtimes handle varint overflow slightly differently:

- **Go:** `pkg/cramberry/reader.go` - Returns `ErrVarintOverflow`
- **TypeScript:** `typescript/src/reader.ts:71-86` - Throws `DecodeError("Varint overflow")`
- **Rust:** `rust/src/reader.rs:76` - Returns `Error::VarintOverflow`

The maximum number of bytes differs:
- Go: Checks `shift < 64` (10 bytes max)
- TypeScript: Checks `shift < 35` for 32-bit (5 bytes), `shift < 70n` for 64-bit (10 bytes)
- Rust: Same as TypeScript

**Recommendation:** Ensure consistent overflow detection across all runtimes. 32-bit varints should fail after 5 bytes, 64-bit after 10 bytes.

---

## Go Runtime Issues

### 7. Interface Nil Check in Type Registry

**Location:** `pkg/cramberry/registry.go:63-70`

**Issue:** When looking up type by name, the function doesn't handle the case where a type was registered but its encoder/decoder is nil.

```go
func (r *Registry) LookupByName(name string) (TypeID, *TypeInfo, bool) {
    for id, info := range r.types {
        if info.Name == name {
            return id, info, true  // Could return nil encoder/decoder
        }
    }
    return 0, nil, false
}
```

**Recommendation:** Add validation during registration and/or lookup.

---

### 8. Struct Field Cache Not Synchronized for Concurrent Access

**Location:** `pkg/cramberry/marshal.go` (implicit)

**Issue:** The reflection-based encoder/decoder builds field metadata on first use. If multiple goroutines marshal/unmarshal the same type simultaneously for the first time, there could be a race condition.

**Recommendation:**
- Add sync.Once or sync.Map for caching type metadata
- Consider code generation for performance-critical paths

---

### 9. Map Key Type Restrictions Not Enforced

**Location:** `pkg/cramberry/marshal.go:178-188`

**Issue:** Architecture specifies map keys must be scalar types (integers or strings). The current implementation doesn't validate this at encode time - it will panic or produce invalid output for struct/slice keys.

```go
func (w *Writer) WriteMap(m reflect.Value) error {
    // Sort keys for deterministic output
    keys := m.MapKeys()
    sort.Slice(keys, func(i, j int) bool {
        return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
    })
    // No validation of key types
    ...
}
```

**Recommendation:** Add validation that map key types are valid (integers, strings, or enum types).

---

### 10. Nested Message Length Prefix Calculation

**Location:** `pkg/cramberry/marshal.go:205-220`

**Issue:** For nested messages, the code calculates the length by encoding to a temporary buffer, which is inefficient for deeply nested structures.

**Current approach:**
```go
// Encode nested struct to get its length
nested := NewWriter()
if err := encodeStruct(nested, rv); err != nil {
    return err
}
data := nested.Bytes()
w.WriteVarint(uint32(len(data)))  // Length prefix
w.WriteBytes(data)
```

**Recommendation:** Consider two-pass encoding (size calculation then encoding) for deeply nested structures, or provide Size() methods for generated types.

---

## TypeScript Runtime Issues

### 11. Missing TypeRef Support in Writer

**Location:** `typescript/src/writer.ts`

**Issue:** The TypeScript Writer class has methods for all wire types except TypeRef (7) for polymorphic encoding.

**Missing:**
- `writeTypeRef(typeId: number, data: Uint8Array): void`
- `writeTypeRefField(fieldNumber: number, typeId: number, data: Uint8Array): void`

**Recommendation:** Add TypeRef encoding support to enable polymorphic serialization.

---

### 12. No Registry Implementation

**Location:** `typescript/src/`

**Issue:** TypeScript runtime lacks a type registry for polymorphic type resolution. The Go and Rust runtimes have this.

**Missing components:**
- Type registry class
- Type registration functions
- Polymorphic encode/decode helpers

**Recommendation:** Implement a TypeRegistry class similar to Go/Rust implementations.

---

### 13. TextEncoder/TextDecoder Instantiation Per Call

**Location:** `typescript/src/writer.ts:199-200`, `typescript/src/reader.ts:211`

**Issue:** Creates new TextEncoder/TextDecoder for each string operation:

```typescript
writeString(value: string): void {
    const encoder = new TextEncoder();  // Created every call
    const bytes = encoder.encode(value);
    ...
}
```

**Recommendation:** Use module-level singleton instances:
```typescript
const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();
```

---

### 14. BigInt Shift Operations Without Bounds Check

**Location:** `typescript/src/types.ts:65-67`

**Issue:** ZigZag encoding for 64-bit integers assumes valid range:

```typescript
export function zigzagEncode64(n: bigint): bigint {
  return (n << 1n) ^ (n >> 63n);  // Assumes 64-bit signed
}
```

If passed a BigInt larger than 64 bits, this produces incorrect results silently.

**Recommendation:** Add input validation or document the constraint.

---

## Rust Runtime Issues

### 15. Error Type Incompleteness

**Location:** `rust/src/error.rs` (implied from `rust/src/reader.rs:39`)

**Issue:** Based on usage, the Error enum should include variants for:
- BufferUnderflow
- VarintOverflow
- InvalidWireType
- InvalidUtf8

Need to verify all error cases are properly defined.

**Recommendation:** Review and document all error variants.

---

### 16. No Streaming/Incremental Parsing

**Location:** `rust/src/reader.rs`

**Issue:** Unlike the Go implementation which has `StreamReader`, the Rust implementation only supports full buffer parsing. This limits use with async I/O.

**Recommendation:** Consider adding:
- `async fn` variants for tokio compatibility
- Incremental parsing for partial buffers

---

### 17. Missing Marshal/Unmarshal Derive Macros

**Location:** `rust/`

**Issue:** Architecture mentions "derive macro for automatic implementations" but no proc-macro crate exists for automatic trait derivation.

**Recommendation:** Create `cramberry-derive` crate with:
```rust
#[derive(CramberryEncode, CramberryDecode)]
struct MyMessage {
    #[cramberry(field = 1)]
    name: String,
}
```

---

## Schema Parser Issues

### 18. Position Tracking Inaccurate for Multi-byte Characters

**Location:** `pkg/schema/lexer.go:439-451`

**Issue:** The advance() function increments column by 1 regardless of character width:

```go
func (l *Lexer) advance() {
    if l.input[l.pos] == '\n' {
        l.line++
        l.column = 1
    } else {
        l.column++  // Doesn't account for multi-byte characters
    }
    _, size := utf8.DecodeRuneInString(l.input[l.pos:])
    l.pos += size  // Pos advances correctly by byte count
}
```

For UTF-8 characters wider than 1 byte, the column position becomes inaccurate.

**Recommendation:** Either:
- Track column as byte offset (simpler, consistent)
- Track column as rune offset (user-friendlier but requires care)

---

### 19. Parser Error Recovery Limited

**Location:** `pkg/schema/parser.go:853-864`

**Issue:** The `synchronize()` function for error recovery is basic and may skip valid declarations:

```go
func (p *Parser) synchronize() {
    for !p.check(TokenEOF) {
        if p.previous.Type == TokenSemicolon || p.previous.Type == TokenRBrace {
            return
        }
        switch p.current.Type {
        case TokenPackage, TokenImport, TokenMessage, TokenEnum, TokenInterface:
            return
        }
        p.advance()
    }
}
```

**Issue:** Doesn't handle nested braces correctly - could synchronize mid-message.

**Recommendation:** Track brace depth and only synchronize at top-level.

---

### 20. No Support for Service/RPC Definitions

**Location:** `pkg/schema/`

**Issue:** Architecture mentions potential RPC support but the parser has no tokens or AST nodes for:
- `service` definitions
- `rpc` methods
- Streaming modifiers

**Recommendation:** Either:
- Add service/rpc parsing (if planned)
- Document this as out of scope

---

## Code Generation Issues

### 21. Go Generator Import Not Conditional

**Location:** `pkg/codegen/go_generator.go:248-257`

**Issue:** The cramberry import is always included even when GenerateMarshal is false:

```go
const goTemplate = `// Code generated by cramberry. DO NOT EDIT.
package {{goPackage}}

import (
{{- if generateMarshal}}
	"github.com/cramberry/cramberry-go/pkg/cramberry"
{{- end}}
)
```

But even with the conditional, an empty import block `import ()` may be generated.

**Recommendation:** Only emit import block if there are imports.

---

### 22. TypeScript Generator Missing

**Location:** `pkg/codegen/typescript_generator.go`

**Issue:** File exists but need to verify it generates proper TypeScript with:
- Proper type annotations
- Interface definitions
- Class-based or interface-based output option

---

### 23. Rust Generator Missing

**Location:** `pkg/codegen/rust_generator.go`

**Issue:** File exists but need to verify it generates:
- Proper struct definitions with derive macros
- Lifetime annotations for borrowed data
- Proper error handling with Result types

---

### 24. Generated Validation Logic Incomplete

**Location:** `pkg/codegen/go_generator.go:316-325`

**Issue:** The Validate() method only checks for zero values, not actual field presence:

```go
// Field {{.Name}} is required
if m.{{goFieldName .}} == {{if eq (goFieldType .) "string"}}""{{else if eq (goFieldType .) "bool"}}false{{else}}0{{end}} {
    return cramberry.NewValidationError(...)
}
```

This incorrectly rejects:
- Required bool fields that are legitimately `false`
- Required int fields that are legitimately `0`
- Required string fields that are legitimately empty

**Recommendation:** Track field presence separately (via bitmap or optional wrapper).

---

## Schema Extraction Issues

### 25. Field Number Auto-Assignment Starts at 1

**Location:** `pkg/extract/collector.go:155`

**Issue:** Default field numbers start at struct field index + 1:

```go
structTag := c.parseTag(tag, i+1)  // i is 0-indexed, so numbers start at 1
```

But if explicit field numbers are mixed with auto-assigned ones, collisions can occur.

**Recommendation:**
- Validate no field number collisions
- Consider reserving auto-assigned numbers in a separate range

---

### 26. Enum Detection Heuristic May Miss Valid Enums

**Location:** `pkg/extract/collector.go:191-203`

**Issue:** Enums are detected as integer types with constants, but this may miss:
- Enums based on `uint` instead of `int`
- Enums with aliased underlying types

```go
case *types.Basic:
    if t.Info()&types.IsInteger != 0 {
        info := &EnumInfo{...}
        c.enums[qualifiedName] = info
    }
```

**Recommendation:** Also check for:
- `types.IsUnsigned`
- Named types with Integer underlying types

---

### 27. Interface Detection Requires Methods

**Location:** `pkg/extract/collector.go:176-189`

**Issue:** Only interfaces with methods are collected:

```go
case *types.Interface:
    if t.NumMethods() > 0 {  // Empty interfaces skipped
        ...
    }
```

This misses marker interfaces (empty interfaces used for type grouping).

**Recommendation:** Add option to include empty interfaces for polymorphic grouping.

---

## Cross-Language Compatibility Issues

### 28. Complex Numbers (Go-only)

**Location:** Various

**Issue:** Go supports complex64/complex128 but TypeScript and Rust don't have native complex number support.

**Impact:** Data containing complex numbers encoded in Go cannot be decoded in TypeScript/Rust.

**Recommendation:** Either:
- Remove complex number support from Go
- Document as Go-only feature
- Add struct-based representation for cross-language use

---

### 29. nil vs undefined/None Semantics

**Location:** Various

**Issue:** Different languages handle null/nil differently:
- Go: `nil` pointer, empty slice, nil map
- TypeScript: `undefined` vs `null`
- Rust: `Option<T>::None`

The wire format doesn't distinguish between "field not present" and "field present with null value."

**Recommendation:** Document the semantic interpretation in each language.

---

### 30. Integer Size Differences

**Location:** Various

**Issue:**
- Go `int` is platform-dependent (32 or 64 bit)
- TypeScript numbers are 64-bit floats (safe integers up to 2^53)
- Rust has explicit sizes

Schema type `int` in Go becomes `int` but generates `int32` in schema extraction.

**Recommendation:**
- Always use explicitly-sized types in schemas
- Warn when Go `int` is detected during extraction

---

## Test Coverage Gaps

### 31. Missing Cross-Runtime Tests

**Issue:** While integration tests exist for individual runtimes, there's no systematic testing of:
- Go encode -> TypeScript decode
- TypeScript encode -> Rust decode
- Rust encode -> Go decode
- All permutations with all types

**Recommendation:** Add comprehensive cross-runtime compatibility test suite.

---

### 32. Missing Edge Case Tests

**Missing test cases:**
- Maximum varint values (2^63-1, 2^64-1)
- Minimum varint values (-2^63)
- Very long strings (> 2GB if length is 32-bit varint)
- Deeply nested structures (100+ levels)
- Maximum field numbers
- Very large maps/arrays
- Malformed input fuzzing

---

### 33. Missing Concurrent Access Tests

**Issue:** No tests for:
- Concurrent Marshal calls with same type
- Concurrent Unmarshal calls
- Concurrent registry modifications

---

## Documentation Gaps

### 34. Wire Format Specification Incomplete

**Missing from ARCHITECTURE.md:**
- Exact byte-level format examples
- Canonical encoding requirements
- Forward/backward compatibility rules
- Field ordering requirements

---

### 35. Migration Guide Missing

**Missing:**
- How to upgrade schema versions
- How to handle deprecated fields
- How to add new required fields safely

---

### 36. Performance Characteristics Undocumented

**Missing:**
- Expected encode/decode performance
- Memory allocation patterns
- Buffer reuse recommendations

---

## Performance Considerations

### 37. Reflection Overhead in Go

**Location:** `pkg/cramberry/marshal.go`, `pkg/cramberry/unmarshal.go`

**Issue:** Extensive use of reflection for every encode/decode operation.

**Recommendation:**
- Consider code generation for hot paths
- Cache reflection metadata
- Provide interface-based encoding for generated types

---

### 38. String Allocations in TypeScript

**Location:** `typescript/src/reader.ts:208-212`

**Issue:** Every string decode creates a new TextDecoder and allocates a new string.

**Recommendation:**
- Reuse TextDecoder
- Consider string interning for repeated values

---

### 39. Buffer Growth Strategy

**Location:** `typescript/src/writer.ts:50-54`

**Issue:** Buffer growth uses 2x multiplier which can lead to excessive memory:

```typescript
let newCapacity = this.buffer.length * GROWTH_FACTOR;
while (newCapacity < required) {
  newCapacity *= GROWTH_FACTOR;
}
```

For a 1GB message, this could allocate 2GB.

**Recommendation:** Consider growth cap or more conservative strategy for large buffers.

---

## Recommendations

### High Priority

1. **Fix deprecated `strings.Title` usage** - Simple fix, prevents future issues
2. **Add cross-runtime compatibility tests** - Critical for cross-language use
3. **Implement TypeScript type registry** - Required for polymorphic support
4. **Fix generated validation logic** - Currently produces incorrect validation

### Medium Priority

5. **Add signed fixed-width integer methods** - API completeness
6. **Implement Rust derive macros** - Ergonomics for Rust users
7. **Document complex number limitation** - Prevent user confusion
8. **Add map key type validation** - Prevent silent data corruption

### Lower Priority

9. **Optimize string handling in TypeScript** - Performance improvement
10. **Add streaming support to Rust** - Feature enhancement
11. **Improve parser error recovery** - Developer experience
12. **Add service/RPC support** - Future feature

---

## Summary

The Cramberry implementation is generally well-structured and functional. The main areas needing attention are:

1. **Cross-language compatibility** - Complex numbers, nil semantics, and integer sizes need clarification
2. **TypeScript feature parity** - Missing type registry and TypeRef encoding
3. **Code generation correctness** - Validation logic needs fixing
4. **Test coverage** - Cross-runtime and edge case tests needed
5. **Performance optimization** - Reflection overhead and allocation patterns

The codebase follows the architecture document well, with the issues identified being primarily edge cases and completeness items rather than fundamental design problems.
