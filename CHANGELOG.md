# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.5.0] - 2026-01-28

### Added
- **Import path mapping for code generation**: Cross-package type references now generate proper Go import statements
  - New `-M` CLI flag: `cramberry generate -M alias=go/import/path`
  - `ImportPaths` option in `codegen.Options` for programmatic use
  - Generated code includes appropriate import statements for external packages

### Changed
- **Exported EncodeTo/DecodeFrom methods**: Generated `EncodeTo()` and `DecodeFrom()` methods are now exported (uppercase) to enable cross-package access
  - This is a breaking change for generated code that references these methods

### Fixed
- Schema parser now allows import statements to appear before, after, or intermixed with option statements
  - Previously, all imports had to come before all options

## [1.4.3] - 2026-01-27

### Security
- **[MEDIUM]** Fixed missing bounds check in `SkipValueV2` for Fixed32/Fixed64 wire types
  - Added `ensure(4)` and `ensure(8)` calls before incrementing position
  - Prevents improper field skipping when malicious message has Fixed32/Fixed64 tag near buffer end

## [1.4.2] - 2026-01-27

### Security
- **[HIGH]** Fixed integer multiplication overflow in packed array readers on 32-bit systems
  - `ReadPackedFloat32`, `ReadPackedFloat64`, `ReadPackedFixed32`, `ReadPackedFixed64` now check `count > math.MaxInt/elementSize` before multiplication
  - Prevents potential memory corruption when malicious input specifies large array counts that overflow 32-bit int

## [1.4.1] - 2026-01-27

### Fixed
- Removed unused deprecated functions (`isPackableType`, `getWireTypeV2`) to pass lint

## [1.4.0] - 2026-01-27

### Added
- **TypeScript streaming support**: Full streaming parity with Go and Rust
  - `StreamWriter` class for writing length-delimited messages
  - `StreamReader` class for reading length-delimited messages with async iteration
  - `MessageIterator<T>` class for automatic decoding during iteration
  - New error classes: `EndOfStreamError`, `MessageSizeExceededError`, `StreamClosedError`
  - Wire format: `[length: varint][message_data: bytes]` (compatible with Go/Rust)

### Changed
- **Go reflection caching improvements**: 13-29% decode speedup
  - Pre-computed `fieldByNum` map in `structInfo` eliminates per-decode allocation
  - Added wire type cache (`wireTypeCache`) for `getWireTypeV2()` lookups
  - Added packable type cache (`packableCache`) for `isPackableType()` lookups
  - Benchmark results:
    - `UnmarshalSmall`: 106ns → 75ns (29% faster)
    - `UnmarshalMedium`: 298ns → 256ns (14% faster)
    - `UnmarshalLarge`: 1482ns → 1288ns (13% faster, 3 fewer allocations)
    - `UnmarshalNested`: 336ns → 255ns (24% faster)

### Performance
- Go reflection-based decode is now 13-29% faster (exceeds 10-20% target)
- Large struct unmarshaling reduced from 40 to 37 allocations

## [1.3.0] - 2026-01-27

### Breaking Changes
- **TypeScript runtime**: Wire type values changed to match Go V2 format:
  - `Fixed32` changed from `5` to `3`
  - `SVarint` changed from `6` to `4`
  - `TypeRef` wire type removed (use `Bytes` for polymorphic types)
- **Rust runtime**: Same wire type value changes as TypeScript
- **Struct encoding format**: Field count prefix replaced with end marker (`0x00`)

### Added
- **Cross-language V2 wire format conformance**: Go, TypeScript, and Rust now produce identical binary encodings
- V2 compact tag format in TypeScript and Rust:
  - Single-byte tags for fields 1-15: `[fieldNum:4][wireType:3][ext:1]`
  - Extended format for fields 16+: marker byte + varint field number
- `Writer.writeEndMarker()` in TypeScript and Rust for struct termination
- `Reader.isEndMarker()` in TypeScript and Rust for end-of-struct detection
- `encodeCompactTag()` / `decodeCompactTag()` functions exported in TypeScript
- `decode_compact_tag()` function and `CompactTagResult` struct exported in Rust
- V2 tag constants exported: `END_MARKER`, `TAG_EXTENDED_BIT`, `TAG_WIRE_TYPE_MASK`, etc.
- Comprehensive cross-language integration tests verifying identical encoding

### Fixed
- TypeScript and Rust integration tests now pass against Go-generated golden files
- Polymorphic type encoding in Rust now uses `Bytes` wire type correctly

### Security
- All pre-release security remediation items completed (see REMEDIATION_PLAN.md)
- Fuzz testing validated: 663M+ executions across 8 test targets with zero crashes

## [1.2.0] - 2026-01-26

### Breaking Changes
- **[CRITICAL]** Zero-copy methods now return wrapper types for memory safety:
  - `ReadStringZeroCopy()` returns `ZeroCopyString` instead of `string`
  - `ReadBytesNoCopy()` returns `ZeroCopyBytes` instead of `[]byte`
  - `ReadRawBytesNoCopy()` returns `ZeroCopyBytes` instead of `[]byte`
  - Call `.String()` or `.Bytes()` to get underlying values
  - Use `.UnsafeString()` / `.UnsafeBytes()` to bypass validation if needed

### Added
- `ZeroCopyString` and `ZeroCopyBytes` wrapper types with generation-based validity tracking
- Generation counter in `Reader` to detect use-after-reset of zero-copy references
- `Reader.Generation()` method to access current generation counter
- `Valid()` method on wrapper types to check if reference is still valid
- Ergonomic accessor methods for zero-copy types:
  - `MustString()` / `MustBytes()` - explicit naming for panicking methods
  - `StringOrEmpty()` / `BytesOrNil()` - non-panicking accessors returning default values
  - `TryString()` / `TryBytes()` - return (value, ok) tuple for explicit error checking
- Field number uniqueness validation - panics with clear error message if two fields have same number
- Comprehensive tests for zero-copy safety mechanisms

### Security
- **[CRITICAL]** Zero-copy methods now detect use-after-reset and panic with clear error message instead of silently returning corrupted data

## [1.1.0] - 2026-01-26

### Added
- Schema compatibility checker (`pkg/schema/compat.go`) for detecting breaking changes between schema versions
- Idempotent type registration with `RegisterOrGet()` and `RegisterOrGetWithID()` functions
- Go fuzz testing targets for parser, lexer, marshal/unmarshal, and varint encoding
- Rust streaming support with `StreamWriter` and `StreamReader` for length-delimited messages
- TypeScript `readInt64AsNumber()` and `readUint64AsNumber()` methods with BigInt precision warnings
- Overflow protection constants: `MaxPackedFloat32Length`, `MaxPackedFloat64Length`, `MaxPackedFixed32Length`, `MaxPackedFixed64Length`
- Exported NaN canonicalization functions: `wire.CanonicalFloat32Bits()`, `wire.CanonicalFloat64Bits()`

### Changed
- Rust `Registry` is now thread-safe using `RwLock` internally
- Rust `Registry.register()` now takes `&self` instead of `&mut self`
- All runtimes (Go, TypeScript, Rust) now enforce consistent 10-byte maximum for varint encoding

### Fixed
- **[CRITICAL]** Integer overflow in packed array writers (`WritePackedFloat32`, `WritePackedFloat64`, etc.) - now checks array length before multiplication
- **[CRITICAL]** Added comprehensive safety documentation for zero-copy methods (`ReadStringZeroCopy`, `ReadBytesNoCopy`)
- **[HIGH]** Cross-language varint consistency - TypeScript and Rust now match Go's 10-byte maximum with proper overflow checking
- **[HIGH]** NaN canonicalization in packed float arrays - all NaN values now encode to canonical quiet NaN
- **[HIGH]** Overflow protection in packed readers (`ReadPackedFloat32`, `ReadPackedFloat64`, etc.)

### Removed
- `MustRegister()` - Use `RegisterOrGet()` for idempotent registration or `Register()` with error handling
- `MustRegisterWithID()` - Use `RegisterOrGetWithID()` for idempotent registration or `RegisterWithID()` with error handling

### Security
- Integer overflow protection prevents memory corruption from maliciously large arrays
- Consistent varint decoding across all languages prevents cross-language parsing discrepancies
- NaN canonicalization ensures deterministic encoding for consensus-critical applications
- Zero-copy method documentation warns users of memory safety requirements

## [1.0.0] - 2026-01-22

### Added
- Initial public release of Cramberry serialization library
- Core runtime library (`pkg/cramberry`) with reflection-based Marshal/Unmarshal
- V2 wire format with compact tags and end markers for optimal performance
- Polymorphic type serialization via type registry
- Streaming support with `StreamWriter`, `StreamReader`, and `MessageIterator`
- Writer/Reader pooling for reduced allocations
- Configurable resource limits for secure decoding of untrusted input
- Pre-configured option sets: `DefaultOptions`, `SecureOptions`, `FastOptions`, `StrictOptions`
- Schema language parser (`pkg/schema`) for `.cram` files
- Code generators (`pkg/codegen`) for Go, TypeScript, and Rust
- Schema extraction (`pkg/extract`) from existing Go code
- CLI tool (`cmd/cramberry`) for code generation and schema management
- TypeScript runtime with Writer/Reader/Registry
- Rust runtime with Writer/Reader/Registry
- Cross-language integration tests
- Performance benchmarks comparing to Protocol Buffers
- Comprehensive documentation:
  - Architecture guide (ARCHITECTURE.md)
  - Benchmark results (BENCHMARKS.md)
  - Development roadmap (ROADMAP.md)
  - Schema language reference (docs/SCHEMA_LANGUAGE.md)
  - Security guide (docs/SECURITY.md)
  - Migration guide (docs/MIGRATION.md)
  - Contributing guide (docs/CONTRIBUTING.md)
- Example applications: basic, polymorphic, streaming

### Performance Highlights
- 1.5-2.6x faster decoding than Protocol Buffers
- Single-allocation encoding pattern
- Zero-allocation decoding for simple messages (e.g., Metrics)
- 42-58% fewer allocations than Protobuf during decode
- Comparable or smaller encoded sizes than Protobuf

### Wire Format
- V2 format: Compact single-byte tags for fields 1-15, end markers
- Packed arrays for primitive types
- Deterministic map encoding with sorted keys
- ZigZag encoding for signed integers

### Type Support
- All Go primitive types (bool, integers, floats, complex)
- Strings with UTF-8 validation
- Byte slices
- Slices and arrays (with packed encoding for primitives)
- Maps with primitive keys
- Nested structs
- Pointers (nil handling)
- Interfaces (via type registry)

### Changed

### Deprecated

### Removed

### Fixed

### Security
- Configurable resource limits prevent denial-of-service attacks
- Depth limiting prevents stack overflow from deeply nested structures
- Size limits prevent memory exhaustion
- UTF-8 validation prevents invalid string injection
- Strict mode rejects unknown fields
