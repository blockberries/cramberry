# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
