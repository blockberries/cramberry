# Cramberry Development Roadmap

This document outlines the development roadmap for Cramberry, including planned features, improvements, and long-term goals.

## Current Status (Pre-Release)

Cramberry is in **pre-release stabilization** with active security hardening:

- High-performance binary serialization (1.5-2.9x faster decode than Protobuf)
- Deterministic encoding for consensus/cryptographic applications
- Cross-language support (Go, TypeScript, Rust)
- Schema language with compatibility checking
- Streaming support across all runtimes

**See [REMEDIATION_PLAN.md](REMEDIATION_PLAN.md) for the detailed stabilization work.**

---

## Stabilization Phase (v1.0.0-rc)

### S1: Security Hardening (BLOCKING)

**Priority:** P0 - Must complete before any release

- [x] Fix integer overflow in V2 compact tag decoding
- [x] Fix length overflow in SkipValueV2
- [x] Fix packed slice/array decoding overflow
- [x] Enforce depth limiting in all recursive paths
- [x] Fix NaN handling in deterministic map sorting
- [x] Add comprehensive security test suite
- [ ] Fuzz testing validation (1+ hour, no crashes)

### S2: Wire Format Consolidation

**Priority:** P0 - Must complete before any release

- [x] Remove V1 wire format entirely (never released, no compatibility needed)
- [x] Centralize varint decoding in `internal/wire/`
- [x] Apply V1-level safety rigor to all V2 code paths
- [x] Update all documentation to remove V1 references

### S3: Code Generator Fixes

**Priority:** P0 - Must complete before any release

- [x] Remove TODO placeholders from generated code
- [x] Fix unknown field handling (skip, don't break)
- [x] Add forward compatibility tests

### S4: API Cleanup

**Priority:** P1

- [x] Remove deprecated `MustRegister`/`MustRegisterWithID` functions
- [x] Improve ZeroCopy API ergonomics
- [x] Add field number uniqueness validation

---

## Short-Term Goals (v1.1.x)

### Performance Optimizations

#### P1: Reflection Caching Improvements
- **Current**: Struct metadata cached via `sync.Map`
- **Goal**: Implement tiered caching with fast paths for common types
- **Benefit**: 10-20% decode speedup for reflection-based marshaling

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
| v1.0.0-rc | In Progress | Security hardening, V1 removal, stabilization |
| v1.0.0 | Planned | First stable release |
| v1.1.0 | Planned | Performance optimizations |
| v1.2.0 | Planned | TypeScript/Rust generator improvements |
| v1.3.0 | Planned | gRPC integration |
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
