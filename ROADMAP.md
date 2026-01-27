# Cramberry Development Roadmap

This document outlines the development roadmap for Cramberry, including planned features, improvements, and long-term goals.

## Current Status (v1.3.0)

Cramberry is **production-ready** with comprehensive security hardening:

- High-performance binary serialization (1.5-2.7x faster decode than Protobuf)
- Deterministic encoding for consensus/cryptographic applications
- Cross-language V2 wire format conformance (Go, TypeScript, Rust)
- Schema language with compatibility checking
- Streaming support (Go and Rust; TypeScript pending)
- Fuzz-tested: 663M+ executions, zero crashes

**See [CHANGELOG.md](CHANGELOG.md) for release history.**

---

## Completed: Stabilization Phase (v1.0.0-v1.3.0)

### S1: Security Hardening ✓

- [x] Fix integer overflow in V2 compact tag decoding
- [x] Fix length overflow in SkipValueV2
- [x] Fix packed slice/array decoding overflow
- [x] Enforce depth limiting in all recursive paths
- [x] Fix NaN handling in deterministic map sorting
- [x] Add comprehensive security test suite
- [x] Fuzz testing validation (2+ hours, 663M+ executions, no crashes)

### S2: Wire Format Consolidation ✓

- [x] Remove V1 wire format entirely (never released, no compatibility needed)
- [x] Centralize varint decoding in `internal/wire/`
- [x] Apply V1-level safety rigor to all V2 code paths
- [x] Update all documentation to remove V1 references
- [x] Cross-language V2 conformance (Go, TypeScript, Rust)

### S3: Code Generator Fixes ✓

- [x] Remove TODO placeholders from generated code
- [x] Fix unknown field handling (skip, don't break)
- [x] Add forward compatibility tests

### S4: API Cleanup ✓

- [x] Remove deprecated `MustRegister`/`MustRegisterWithID` functions
- [x] Improve ZeroCopy API ergonomics
- [x] Add field number uniqueness validation

---

## Short-Term Goals (v1.4.0)

### Performance Optimizations

#### P1: Reflection Caching Improvements ✓
- [x] Pre-computed `fieldByNum` map in structInfo (eliminates per-decode allocation)
- [x] Wire type cache for `getWireTypeV2()`
- [x] Packable type cache for `isPackableType()`
- **Result**: 13-29% decode speedup achieved (exceeds 10-20% target)

#### P2: SIMD-Accelerated Encoding
- **Target**: ARM64 NEON and x86-64 AVX2 acceleration
- **Scope**: Packed array encoding/decoding, string validation
- **Benefit**: 2-4x speedup for large arrays and bulk string operations

#### P3: Arena Allocator Support
- **Goal**: Optional arena allocator for batch decoding
- **API**: `UnmarshalWithArena(data, &msg, arena)`
- **Benefit**: Near-zero allocations for batch processing

### Developer Experience

#### D1: Improved Error Messages
- Include field paths in decode errors
- Add suggestions for common mistakes
- Provide wire format debug output

#### D2: Schema Linting
- Warn about potential issues (complex types, platform-dependent sizes)
- Suggest field number gaps for future compatibility
- Check for cross-language compatibility issues

#### D3: IDE Integration
- VS Code extension for `.cram` files
- Syntax highlighting and auto-completion
- Go to definition for message types

---

## Medium-Term Goals (v1.2.x - v1.3.x)

### Code Generation Enhancements

#### G1: TypeScript Generator Improvements
- Generate full marshal/unmarshal methods (currently runtime-only)
- Support for discriminated unions
- ESM and CommonJS dual output

#### G2: Rust Generator Enhancements
- Derive macro support (`#[derive(Cramberry)]`)
- `no_std` compatibility option
- Async read/write support

#### G3: Python Code Generator
- New language target for Python
- Type hints (PEP 484) support
- Protocol buffer migration tooling

### Protocol Features

#### F1: gRPC Integration
- Cramberry as alternative serialization for gRPC
- Service definition in schema language
- Generated client/server stubs

#### F2: Schema Evolution Tools
- Automated migration script generation
- Version negotiation protocol
- Field deprecation workflow

#### F3: Compression Support
- Built-in LZ4/Zstd compression option
- Per-field compression hints
- Streaming compression

### Cross-Language Improvements

#### C1: WebAssembly Runtime
- Compile Go runtime to WASM
- Browser and Node.js support
- Shared memory zero-copy decode

#### C2: C/C++ Runtime
- Native C runtime for embedded systems
- C++ wrapper with RAII semantics
- CMake integration

---

## Long-Term Goals (v2.0+)

### Wire Format Enhancements

#### W1: Backward-Compatible Improvements
- Optional field presence tracking
- Default value encoding optimization
- Extended type metadata

#### W2: Breaking Changes (Major Version)
- Unified tag format across all wire types
- Native timestamp/duration types
- Built-in decimal type for financial applications

### Ecosystem

#### E1: Schema Registry
- Centralized schema management service
- Version control and compatibility checking
- Schema discovery and documentation

#### E2: Observability Integration
- OpenTelemetry trace context propagation
- Metrics for encode/decode performance
- Error rate tracking

#### E3: Testing Framework
- Property-based testing for schemas
- Fuzz testing integration
- Cross-language conformance test suite

### Advanced Features

#### A1: Streaming RPC
- Bidirectional streaming protocol
- Flow control and backpressure
- Connection multiplexing

#### A2: Zero-Knowledge Proofs
- Selective field disclosure
- Merkle tree encoding for field proofs
- Compatible with ZK circuits

#### A3: Time-Travel Debugging
- Encode/decode history tracking
- Binary diff visualization
- Replay and inspection tools

---

## Version History

| Version | Status | Highlights |
|---------|--------|------------|
| v1.0.0 | Released | Initial stable release |
| v1.1.0 | Released | Security hardening, cross-language consistency, schema compatibility |
| v1.2.0 | Released | Zero-copy memory safety with generation tracking |
| v1.3.0 | Released | Cross-language V2 wire format conformance |
| v1.4.0 | Completed | Reflection caching (13-29% decode speedup), TypeScript streaming |
| v1.5.0 | Planned | SIMD acceleration, arena allocator support |
| v2.0.0 | Future | Wire format enhancements, breaking changes |

---

## Contributing

We welcome contributions! Priority areas:

1. **Security review** - Help identify and fix vulnerabilities
2. **Cross-language testing** - Ensure compatibility across runtimes
3. **Documentation** - Improve examples and tutorials
4. **Performance benchmarks** - Help identify optimization opportunities

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

---

## Feedback

Have suggestions for the roadmap? Open an issue on GitHub or reach out via:
- GitHub Issues: https://github.com/blockberries/cramberry/issues
- Discussions: https://github.com/blockberries/cramberry/discussions
