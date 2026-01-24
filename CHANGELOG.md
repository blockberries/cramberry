# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial public release of Cramberry serialization library
- Core runtime library (`pkg/cramberry`) with reflection-based Marshal/Unmarshal
- V2 wire format with compact tags and end markers for optimal performance
- V1 wire format support for backward compatibility
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
- V2 format (default): Compact single-byte tags for fields 1-15, end markers
- V1 format (legacy): Field count prefix, full varint tags
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
- `V1Options` - Use V2 format for new code; V1 maintained for compatibility

### Removed

### Fixed

### Security
- Configurable resource limits prevent denial-of-service attacks
- Depth limiting prevents stack overflow from deeply nested structures
- Size limits prevent memory exhaustion
- UTF-8 validation prevents invalid string injection
- Strict mode rejects unknown fields
