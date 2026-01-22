# Cramberry Roadmap

This document outlines the development roadmap for Cramberry, organized into phases with clear priorities and rationale.

---

## Vision

Cramberry aims to be the premier binary serialization library for deterministic, high-performance encoding across Go, TypeScript, and Rust. Our focus areas are:

1. **Performance**: Fastest-in-class decoding, minimal allocations
2. **Determinism**: Byte-for-byte reproducible encoding for consensus systems
3. **Cross-Language**: Full feature parity across all supported runtimes
4. **Developer Experience**: Excellent tooling, clear errors, comprehensive docs

---

## Phase 1: Foundation Hardening

**Focus**: Address critical gaps and improve core reliability

### 1.1 Reflection Caching for Go Runtime

**Priority**: High
**Impact**: 20-40% performance improvement for reflection-based marshaling

**Current State**: The Go runtime re-parses struct tags on every `Marshal`/`Unmarshal` call, adding unnecessary overhead.

**Proposed Changes**:
- Add `structInfoCache` with `sync.Map` for concurrent access
- Cache field metadata: field numbers, wire types, encoder/decoder functions
- Implement cache warming API for latency-sensitive applications
- Add benchmarks comparing cached vs uncached performance

**Files to modify**:
- `pkg/cramberry/marshal.go`
- `pkg/cramberry/unmarshal.go`
- New file: `pkg/cramberry/cache.go`

### 1.2 Thread-Safe Rust Registry

**Priority**: High
**Impact**: Enables safe concurrent use of Rust runtime

**Current State**: Rust `Registry` uses `HashMap` without synchronization, making it unsafe for concurrent access.

**Proposed Changes**:
- Wrap registry maps in `RwLock<HashMap<...>>`
- Implement `Send + Sync` traits
- Add concurrent access tests
- Document thread-safety guarantees

**Files to modify**:
- `rust/src/registry.rs`
- `rust/src/lib.rs` (re-export thread-safe types)

### 1.3 Fuzzing Infrastructure

**Priority**: High
**Impact**: Discover edge cases and security vulnerabilities

**Proposed Changes**:
- Add `go-fuzz` targets for:
  - Schema parser (`pkg/schema/parser.go`)
  - Wire protocol decoder (`pkg/cramberry/reader.go`)
  - Varint decoder (`internal/wire/varint.go`)
- Add `cargo-fuzz` targets for Rust runtime
- Integrate with CI for continuous fuzzing
- Create corpus of interesting inputs

**New files**:
- `fuzz/parser/fuzz.go`
- `fuzz/decoder/fuzz.go`
- `rust/fuzz/` directory

### 1.4 Wire Format Version Detection

**Priority**: Medium
**Impact**: Enables seamless version upgrades

**Current State**: V1 and V2 wire formats are incompatible and require explicit version selection.

**Proposed Changes**:
- Add version byte prefix option (0x01 for V1, 0x02 for V2)
- Implement auto-detection in `Reader`
- Maintain backward compatibility with unversioned data
- Add migration guide documentation

**Files to modify**:
- `pkg/cramberry/reader.go`
- `pkg/cramberry/writer.go`
- `pkg/cramberry/options.go`

---

## Phase 2: Feature Parity

**Focus**: Bring TypeScript and Rust runtimes to full feature parity with Go

### 2.1 Rust Streaming Support

**Priority**: High
**Impact**: Enables processing large datasets in Rust

**Proposed Changes**:
- Implement `StreamWriter` wrapping `BufWriter<W: Write>`
- Implement `StreamReader` wrapping `BufReader<R: Read>`
- Add `MessageIterator` for reading delimited streams
- Add async variants using `tokio::io`

**New files**:
- `rust/src/stream.rs`
- `rust/src/async_stream.rs` (feature-gated)

### 2.2 TypeScript Streaming Completion

**Priority**: Medium
**Impact**: Full streaming support for Node.js/Deno

**Proposed Changes**:
- Add `StreamWriter` class wrapping `WritableStream`
- Add `StreamReader` class wrapping `ReadableStream`
- Implement async iterator for message sequences
- Support both Node.js streams and Web Streams API

**Files to modify**:
- `typescript/src/stream.ts` (new)
- `typescript/src/index.ts`

### 2.3 Rust Async Support

**Priority**: Medium
**Impact**: Integration with async Rust ecosystems

**Proposed Changes**:
- Add `async-std` and `tokio` feature flags
- Implement `AsyncReader`/`AsyncWriter` traits
- Non-blocking varint reading/writing
- Integrate with popular async frameworks

**New files**:
- `rust/src/async_reader.rs`
- `rust/src/async_writer.rs`

### 2.4 Interface Support for TypeScript/Rust

**Priority**: Medium
**Impact**: Full polymorphic serialization support

**Current State**: Go has full interface registration; TS/Rust have basic registry but no interface semantics.

**Proposed Changes**:
- TypeScript: Add discriminated union generation
- Rust: Add enum-based interface generation with `#[derive(Cramberry)]`
- Generate type guards/match helpers
- Update code generators

**Files to modify**:
- `pkg/codegen/typescript_generator.go`
- `pkg/codegen/rust_generator.go`
- `typescript/src/registry.ts`
- `rust/src/registry.rs`

---

## Phase 3: Performance Optimization

**Focus**: Push performance boundaries and reduce allocations

### 3.1 Zero-Copy String Decoding Option

**Priority**: Medium
**Impact**: Eliminate string allocations for read-only access

**Proposed Changes**:
- Add `UnmarshalZeroCopy` variant returning borrowed strings
- Use `unsafe` with clear lifetime constraints
- Document safety requirements
- Add benchmarks showing allocation reduction

**Files to modify**:
- `pkg/cramberry/unmarshal.go`
- `pkg/cramberry/reader.go`

### 3.2 SIMD-Accelerated Encoding

**Priority**: Low
**Impact**: 2-4x faster bulk encoding for specific patterns

**Proposed Changes**:
- Add SIMD paths for packed arrays (AVX2, NEON)
- Implement vectorized varint encoding for batches
- Feature-gate behind build tags
- Benchmark across different CPU architectures

**New files**:
- `internal/wire/simd_amd64.go`
- `internal/wire/simd_arm64.go`

### 3.3 Arena Allocator Support

**Priority**: Low
**Impact**: Reduce GC pressure for batch processing

**Proposed Changes**:
- Add `Arena` type for batch allocations
- Integrate with Go 1.22+ arena package (when stable)
- Pool-based arena for older Go versions
- Configurable via `Options`

**New files**:
- `pkg/cramberry/arena.go`

### 3.4 Compile-Time Code Generation

**Priority**: Medium
**Impact**: Eliminate reflection overhead entirely

**Proposed Changes**:
- Add `go generate` support via `//go:generate cramberry`
- Generate type-specific marshal/unmarshal functions
- Integrate with build system
- Benchmark generated vs reflection code

**New files**:
- `cmd/cramberry/generate_go.go`
- Documentation updates

---

## Phase 4: Schema Evolution

**Focus**: Support backward/forward compatible schema changes

### 4.1 Schema Compatibility Checker

**Priority**: High
**Impact**: Prevent breaking changes in production

**Proposed Changes**:
- CLI command: `cramberry check-compat old.cram new.cram`
- Detect breaking changes:
  - Field number reuse
  - Type changes
  - Required field additions
- Generate compatibility reports
- Integration with CI pipelines

**New files**:
- `pkg/schema/compat.go`
- `cmd/cramberry/compat.go`

### 4.2 Field Deprecation Support

**Priority**: Medium
**Impact**: Graceful field retirement

**Proposed Changes**:
- Add `deprecated` option with message: `[deprecated = "Use field X instead"]`
- Generate deprecation warnings in code
- Emit compiler warnings for deprecated field usage
- Document deprecation workflow

**Files to modify**:
- `pkg/schema/ast.go`
- `pkg/schema/parser.go`
- `pkg/codegen/*.go`

### 4.3 Optional Field Semantics

**Priority**: Medium
**Impact**: Distinguish "not set" from "set to zero"

**Proposed Changes**:
- Add `optional` modifier with explicit presence tracking
- Generate `Has*()` methods for optional fields
- Wire format: presence bitmap or sentinel values
- Document optional vs omitempty semantics

**Files to modify**:
- `pkg/schema/ast.go`
- `pkg/codegen/go_generator.go`
- `pkg/cramberry/marshal.go`

### 4.4 Schema Registry Service

**Priority**: Low
**Impact**: Centralized schema management for large systems

**Proposed Changes**:
- HTTP API for schema storage/retrieval
- Schema versioning with semantic versioning
- Compatibility enforcement on registration
- Client libraries for Go/TypeScript/Rust

**New package**:
- `pkg/registry/` (schema registry server)
- `cmd/cramberry-registry/`

---

## Phase 5: Ecosystem Integration

**Focus**: Integrate with popular frameworks and tools

### 5.1 gRPC Integration

**Priority**: Medium
**Impact**: Alternative to Protobuf for gRPC

**Proposed Changes**:
- Implement `grpc.Codec` interface
- Add service definition syntax to schema language
- Generate gRPC service stubs
- Benchmark against protobuf-based gRPC

**New files**:
- `pkg/grpc/codec.go`
- `pkg/codegen/grpc_generator.go`

### 5.2 Database Serialization Adapters

**Priority**: Low
**Impact**: Native storage in databases

**Proposed Changes**:
- PostgreSQL custom type adapter
- Redis serialization helpers
- BadgerDB/BoltDB integration examples
- SQL `BLOB` helpers with type safety

**New package**:
- `contrib/postgres/`
- `contrib/redis/`

### 5.3 OpenTelemetry Integration

**Priority**: Low
**Impact**: Observability for serialization operations

**Proposed Changes**:
- Add tracing spans for marshal/unmarshal
- Metrics: encode/decode latency, message sizes
- Baggage propagation through serialized data
- Optional via build tags

**New files**:
- `pkg/cramberry/otel.go`

### 5.4 IDE Plugins

**Priority**: Medium
**Impact**: Better developer experience

**Proposed Changes**:
- VS Code extension for `.cram` files:
  - Syntax highlighting
  - Error diagnostics (via LSP)
  - Go to definition
  - Code completion
- JetBrains plugin (GoLand, WebStorm)
- Language server implementation

**New repository**:
- `cramberry-vscode`
- `cramberry-lsp`

---

## Phase 6: Documentation & Community

**Focus**: Lower barriers to adoption

### 6.1 Interactive Documentation Site

**Priority**: Medium
**Impact**: Easier onboarding

**Proposed Changes**:
- Docusaurus or similar static site
- Interactive schema playground (WASM-based)
- Copy-paste examples for common patterns
- Migration guides from Protobuf/JSON

### 6.2 Benchmark Suite Expansion

**Priority**: Medium
**Impact**: Informed decision-making for users

**Proposed Changes**:
- Add comparisons with:
  - MessagePack
  - FlatBuffers
  - Cap'n Proto
  - CBOR
- Real-world message patterns (blockchain, gaming, IoT)
- Memory profiling deep dives
- Publish results on documentation site

### 6.3 Example Repository

**Priority**: Low
**Impact**: Accelerate adoption

**Proposed Changes**:
- Full application examples:
  - REST API with Cramberry serialization
  - WebSocket real-time messaging
  - Blockchain transaction encoding
  - Cross-language microservices
- Docker Compose setups
- CI/CD pipeline examples

---

## Cross-Cutting Concerns

### Security Hardening

**Ongoing throughout all phases**:
- Regular dependency audits
- Security-focused code review checklist
- CVE monitoring and response process
- Penetration testing for schema parser
- Input validation documentation

### Performance Regression Prevention

**Ongoing throughout all phases**:
- Benchmark baselines in CI
- Automated alerts for >5% regressions
- Memory allocation budgets per operation
- Flame graph analysis quarterly

### Backward Compatibility

**Policy**:
- Wire format compatibility guaranteed within major versions
- API compatibility following semver
- Deprecation warnings before removals
- Migration tooling for breaking changes

---

## Version Milestones

### v1.1 (Phase 1)
- Reflection caching
- Thread-safe Rust registry
- Fuzzing infrastructure
- Wire version detection

### v1.2 (Phase 2)
- Rust streaming
- TypeScript streaming completion
- Full interface support across languages

### v1.3 (Phase 3)
- Zero-copy decoding options
- Compile-time code generation
- Performance optimizations

### v2.0 (Phase 4-5)
- Schema evolution tools
- gRPC integration
- Breaking changes (if necessary) for long-term improvements

---

## Contributing

We welcome contributions to any roadmap item. Before starting work:

1. Check if an issue exists for the item
2. Comment on the issue to claim it
3. For large changes, create an RFC document first
4. Follow the contribution guidelines in CONTRIBUTING.md

**Priority labels**:
- `priority:critical` - Blocking production use
- `priority:high` - Significant impact, needed soon
- `priority:medium` - Important but not urgent
- `priority:low` - Nice to have

---

## Feedback

This roadmap is a living document. To suggest changes:

1. Open a GitHub issue with the `roadmap` label
2. Describe the feature/change and its impact
3. Provide use cases or benchmarks if applicable

We review and update this roadmap quarterly.
