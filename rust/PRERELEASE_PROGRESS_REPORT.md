
## Phase: Deterministic JSON Serialization - Rust Implementation (Phase 3)

**Status**: ✅ COMPLETED

**Date**: 2026-02-02

### Summary
Implemented deterministic JSON serialization for Rust with to_json/from_json functions. Completes the cross-language JSON serialization suite with consistent API and determinism guarantees across Go, TypeScript, and Rust.

### Files Created
- `rust/src/json.rs` - Runtime JSON helpers (formatters, validators, base64, sorting)
- `rust/test/json_test.rs` - Generated code from test schema
- `rust/test/json_test_test.rs` - Integration test scaffolding

### Files Modified
- `pkg/codegen/rust_generator.go` - Added 350+ lines of JSON template functions
  - `jsonEncodeField()`, `jsonDecodeField()` - Field-level encoding/decoding
  - `jsonEncodeValue()`, `jsonDecodeValue()` - Type-specific encoding/decoding
  - `jsonEncodeScalar()`, `jsonDecodeScalar()` - Primitive type handling
  - `jsonEncodeEnum()`, `jsonDecodeEnum()` - Enum as string names
  - `jsonEncodeMap()`, `jsonDecodeMap()` - Lexicographic key sorting
  - `jsonEncodeArray()`, `jsonDecodeArray()` - Array preservation
  - Template generates `to_json_message_name()` and `from_json_message_name()` functions
  - Added `to_string()` method to enums for JSON encoding
- `rust/Cargo.toml` - Added dependencies: `base64`, `serde`, `serde_json`
- `rust/src/lib.rs` - Added json module export
- Fixed bug: Changed WireTypeV2 to WireType (correct type name for Rust)

### Key Functionality Implemented

#### JSON Encoding Features
- **i64/u64 as strings**: All integers formatted as strings
- **Base64 byte arrays**: RFC 4648 standard encoding
- **Enum string names**: to_string() method for readable enum values
- **Lexicographic map keys**: Sorted UTF-8 byte order
- **Compact format**: No whitespace
- **Float precision**: format!() with appropriate precision
- **NaN/Infinity validation**: Errors on special float values
- **Nested messages**: Recursive to_json() calls
- **Option<T> handling**: Proper encoding of None as null
- **Box<T> handling**: Proper dereferencing for boxed types

#### JSON Decoding Features
- **Flexible input**: Accepts string or numeric integers via serde_json::Value
- **Strict unknown fields**: Rejects unrecognized fields
- **Enum validation**: String name to enum variant mapping
- **Base64 decoding**: Proper error handling
- **Type coercion**: Handles various JSON representations
- **HashMap initialization**: Proper collection creation
- **Result<T, String>**: Idiomatic Rust error handling

#### Runtime Helpers
- `format_i64_to_string()`, `format_u64_to_string()`: Integer formatting
- `format_i32_to_string()`, `format_u32_to_string()`: Smaller integer formatting
- `format_f32()`, `format_f64()`: Float formatting with validation
- `encode_base64()`, `decode_base64()`: Base64 encoding/decoding
- `parse_i64_from_json()`, `parse_u64_from_json()`: Parse from JSON value
- `parse_i32_from_json()`, `parse_u32_from_json()`: Parse smaller ints
- `sort_map_keys_lexicographic()`: In-place lexicographic sorting
- `escape_json_string()`: JSON-safe string escaping using serde_json
- `JsonBuilder`: Efficient string building utility

### Test Coverage
- ✅ Unit tests: 9 tests for all JSON helper functions (all pass)
- ✅ Format functions: i64, u64, f32, f64
- ✅ Base64: Encoding/decoding
- ✅ Parsing: i64, u64 from JSON values
- ✅ Map sorting: Lexicographic order verified
- ✅ String escaping: Special characters
- ✅ Library compilation: cargo check passes
- ✅ All existing tests: 38 tests pass (no regressions)

### Design Decisions
1. **Function-based API**: `to_json_message_name()` pattern
2. **Result<String, String>**: Idiomatic error handling
3. **Serde dependency**: Leverages ecosystem standard for JSON
4. **Base64 crate**: Industry-standard base64 implementation
5. **Default trait**: Uses Default::default() for initialization
6. **Enum to_string()**: Static str return for zero-allocation
7. **HashMap for maps**: std::collections::HashMap for key-value pairs
8. **Bug fix**: Corrected WireTypeV2 to WireType (existing bug in generator)

### Bug Fixes
- Fixed Rust generator to use `WireType` instead of non-existent `WireTypeV2`
- This bug existed in the original Rust generator and has now been corrected

### Known Limitations
- Nested message encoding could call specific to_json functions (currently uses to_json recursively)
- Float formatting precision may differ slightly from Go/TypeScript due to Rust's formatting
- No validation for required fields yet (would need schema metadata in generated code)

### Dependencies Added
- `serde = { version = "1.0", features = ["derive"] }` - JSON serialization framework
- `base64 = "0.22"` - Base64 encoding/decoding
- `serde_json = "1.0"` - JSON parsing and Value type

### Performance Notes
- Uses String::push_str() for efficient string building
- Map sorting is O(n log n) for determinism
- No reflection (direct field access in generated code)
- Base64 crate provides optimized encoding

### Next Steps
- Integration testing with complex message types
- Cross-language JSON compatibility verification (Go ↔ TS ↔ Rust)
- Documentation updates (CLAUDE.md, README.md)
- Golden file testing across all three languages

