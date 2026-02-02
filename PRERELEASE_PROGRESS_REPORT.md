
## Phase: Deterministic JSON Serialization - Go Implementation (Phase 1)

**Status**: ✅ COMPLETED

**Date**: 2026-02-02

### Summary
Successfully implemented deterministic JSON serialization for Go with ToJSON()/FromJSON() methods. This provides blockchain-grade JSON encoding for SignDoc generation with perfect round-trip guarantees.

### Files Created
- `pkg/cramberry/json.go` - Runtime JSON helpers (formatters, validators, base64, sorting)
- `pkg/cramberry/json_test.go` - Comprehensive unit tests (all pass)
- `tests/testdata/json_test.cram` - Test schema covering all types
- `tests/generated/json_test_test.go` - Integration tests for JSON functionality

### Files Modified
- `pkg/codegen/go_generator.go` - Added 500+ lines of JSON template functions and template updates
  - `jsonEncodeField()`, `jsonDecodeField()` - Field-level encoding/decoding
  - `jsonEncodeValue()`, `jsonDecodeValue()` - Type-specific encoding/decoding
  - `jsonEncodeScalar()`, `jsonDecodeScalar()` - Primitive type handling
  - `jsonEncodeEnum()`, `jsonDecodeEnum()` - Enum as string names
  - `jsonEncodeMap()`, `jsonDecodeMap()` - Lexicographic key sorting
  - `jsonEncodeArray()`, `jsonDecodeArray()` - Array preservation
  - Template now generates ToJSON()/FromJSON() methods for all messages

### Key Functionality Implemented

#### JSON Encoding Features
- **All integers as strings**: Prevents JavaScript precision loss (>2^53-1)
- **Base64 byte arrays**: Standard RFC 4648 encoding
- **Enum string names**: Human-readable (e.g., `"ACTIVE"` not `1`)
- **Lexicographic map keys**: Sorted UTF-8 byte order for determinism
- **Compact format**: No whitespace, minimal size
- **Float precision**: 9 digits (float32), 17 digits (float64)
- **Required field validation**: Errors on missing required fields (both encode and decode)
- **Nested messages**: Recursive ToJSON() calls
- **Pointer fields**: Proper handling of required scalar pointers
- **All fields included**: Even zero values (empty string, 0, false, [])

#### JSON Decoding Features
- **Flexible input**: Accepts both string and numeric integers
- **Strict unknown fields**: Rejects unrecognized fields
- **Perfect round-trip**: FromJSON(ToJSON(msg)) produces byte-identical output
- **Same-package imports**: Correctly handles unqualified type names
- **Validation**: Required fields checked after decoding

#### Error Handling
- NaN/Infinity rejected with clear error messages
- Complex numbers rejected (not supported cross-language)
- Invalid base64 detected and reported
- Unknown enum values rejected
- Type mismatches caught with descriptive errors

### Test Coverage
- ✅ Unit tests: All JSON helper functions tested
- ✅ Scalar types: bool, int8-64, uint8-64, float32/64, string, bytes
- ✅ Enums: String name encoding, round-trip
- ✅ Maps: Lexicographic key sorting verified
- ✅ Nested messages: Recursive encoding
- ✅ Required fields: Validation in both directions
- ✅ Zero values: All fields included
- ✅ Unknown fields: Strict rejection
- ✅ Round-trip: Byte-identical guarantee verified
- ✅ No regressions: Full test suite passes (make test)

### Design Decisions
1. **String integers**: Chosen for JavaScript compatibility (Cosmos SDK standard)
2. **Enum string names**: Human-readable for SignDoc inspection
3. **Compact format**: No whitespace for minimal size (blockchain use)
4. **Lexicographic sorting**: UTF-8 byte order (Cosmos SDK compatible)
5. **Strict unknown fields**: Catches typos and schema mismatches
6. **Pointer dereference**: Required scalar fields automatically handled
7. **Same-package optimization**: Unqualified type names for cleaner code

### Performance Notes
- Uses `strings.Builder` for efficient string concatenation
- Map sorting is O(n log n) but unavoidable for determinism
- No reflection in generated code (direct field access)
- Format helpers pre-tested and optimized

### Known Limitations
- Complex numbers not supported (design choice for cross-language)
- Maps with non-string keys require string conversion (JSON limitation)
- strconv import removed for schemas without integer map keys
- Floating point determinism assumes consistent Go formatting across platforms

### Next Steps
- **Phase 2**: TypeScript implementation (runtime + codegen)
- **Phase 3**: Rust implementation (runtime + codegen)
- **Documentation**: Update CLAUDE.md and README with JSON examples
- **Golden files**: Generate cross-language JSON test files


## Phase: Deterministic JSON Serialization - TypeScript Implementation (Phase 2)

**Status**: ✅ IMPLEMENTED (pending full integration testing)

**Date**: 2026-02-02

### Summary
Implemented deterministic JSON serialization for TypeScript with toJSON/fromJSON functions. Mirrors Go implementation with language-specific adaptations for TypeScript/JavaScript ecosystem.

### Files Created
- `typescript/src/json.ts` - Runtime JSON helpers (formatters, validators, base64, sorting)
- `typescript/src/json.test.ts` - Comprehensive unit tests (36 tests, all pass)
- `typescript/test/json_test.ts` - Generated code from test schema
- `typescript/test/json-integration.test.ts` - Integration tests for generated code

### Files Modified
- `pkg/codegen/typescript_generator.go` - Added 300+ lines of JSON template functions
  - `jsonEncodeField()`, `jsonDecodeField()` - Field-level encoding/decoding
  - `jsonEncodeValue()`, `jsonDecodeValue()` - Type-specific encoding/decoding
  - `jsonEncodeScalar()`, `jsonDecodeScalar()` - Primitive type handling
  - `jsonEncodeEnum()`, `jsonDecodeEnum()` - Enum as string names
  - `jsonEncodeMap()`, `jsonDecodeMap()` - Lexicographic key sorting
  - `jsonEncodeArray()`, `jsonDecodeArray()` - Array preservation
  - Template generates `toJSON_MessageName()` and `fromJSON_MessageName()` functions
- `typescript/src/index.ts` - Added JSON helper exports

### Key Functionality Implemented

#### JSON Encoding Features
- **BigInt handling**: int64/uint64 as strings for precision
- **Number formatting**: int8-32, uint8-32 as strings
- **Base64 byte arrays**: Browser and Node.js compatible
- **Enum string names**: Human-readable enum values
- **Lexicographic map keys**: Sorted for determinism
- **Compact format**: No whitespace
- **Float precision**: toPrecision(9/17) for float32/float64
- **Nested messages**: JSON.stringify for now (TODO: call specific toJSON functions)

#### JSON Decoding Features
- **Flexible input**: Accepts string or numeric integers
- **Strict unknown fields**: Rejects unrecognized fields
- **Type coercion**: Proper type conversion for all scalars
- **Enum validation**: String name to enum value mapping
- **Base64 decoding**: Both browser and Node.js

#### Runtime Helpers
- `formatBigIntToString()`: Convert bigint to string
- `formatNumberToString()`: Convert number to string
- `formatFloat32/64()`: Fixed precision formatting with NaN/Infinity validation
- `encodeBase64/decodeBase64()`: Cross-platform base64
- `parseBigIntFromJSON()`: Parse bigint from string or number
- `parseNumberFromJSON()`: Parse number from string or number
- `sortMapKeysLexicographic()`: Lexicographic sorting
- `escapeJSONString()`: JSON-safe string escaping
- `JSONWriter/JSONReader`: Helper classes for building/parsing JSON

### Test Coverage
- ✅ Unit tests: 36 tests for all JSON helper functions (all pass)
- ✅ Format functions: BigInt, Number, Float32, Float64
- ✅ Base64: Encoding/decoding with browser/Node compatibility
- ✅ Parsing: BigInt and Number from JSON
- ✅ Map sorting: Lexicographic order verified
- ✅ String escaping: Special characters, unicode
- ✅ JSONWriter/JSONReader: Building and parsing

### Design Decisions
1. **Function-based API**: `toJSON_MessageName()` rather than methods (TypeScript interfaces)
2. **Browser + Node.js**: Cross-platform base64 encoding
3. **BigInt for int64/uint64**: Native TypeScript bigint type
4. **Number for int32/uint32**: Standard number type (safe range)
5. **toPrecision for floats**: Deterministic float formatting
6. **Nested messages**: Currently uses JSON.stringify (simpler, works for now)
7. **Partial<T>**: Allows flexible field initialization in fromJSON

### Known Limitations
- Nested message encoding uses JSON.stringify rather than calling specific toJSON functions
- Would benefit from more integration testing with actual message types
- No validation for required fields yet (would need schema metadata)

### Next Steps
- Complete integration testing with complex message types
- Consider adding toJSON function calls for nested messages
- Run full test suite to ensure no regressions
- **Phase 3**: Rust implementation (runtime + codegen)

