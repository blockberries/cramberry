# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed (correctness — fifth review pass)

- **`Reader.ReadTag` no longer treats truncated input as a clean
  end-of-message**. EOF without an explicit `0x00` end marker now
  returns `ErrUnexpectedEOF`. Encoders always emit the marker, so a
  missing one means the wire bytes are short — silently accepting
  it caused truncated payloads to round-trip to a struct whose tail
  fields all held zero values with no error reported.
- **`Writer.EndMessage` validates `checkpoint` is in range**. A buggy
  caller passing a stale checkpoint (e.g. from a Reset'd writer)
  would compute a negative `msgLen`, encode a giant `uvarint` length
  prefix, and `copy()` it into a bogus offset — silent wire
  corruption. Now sets an explicit error.
- **`Reader.Skip(-n)` rejected**. The `ensure(n)` bound check was
  vacuous for negative `n`, so `Skip(-5)` rewound the read position
  by 5 with no error.
- **Generated Go `ToJSON` for optional message fields no longer
  emits uncompilable `*m.X.ToJSON()`**. Operator precedence parses
  that as `*(m.X.ToJSON())`, which fails because `ToJSON` returns
  `(string, error)`. The generator now wraps in parentheses
  (`(*m.X)`), which Go auto-dereferences for method calls anyway.
- **Schema lexer accepts hex literals (`0xFF`, `0xdeadbeef`)** —
  previously split into `Int(0)` + `Ident(xFF)`, silently dropping
  the user's intent.
- **Schema lexer rejects malformed exponents (`1e`, `1e-`)** —
  previously produced a `TokenFloat` with a value that
  `strconv.ParseFloat` then failed on at codegen time.
- **`SubReader` removed**. No production callers; sub-reader errors
  did not propagate back to the parent — anyone using it would have
  silently read past corruption.

### Fixed (consensus-critical)

- **Non-canonical varint encodings now rejected** in Go and Rust. Two
  byte sequences could decode to the same value (e.g. `0x80 0x00` and
  `0x00` both decoded to 0), so a hash over received bytes diverged
  across runtimes for the same logical input — a silent consensus
  split. The decoders now return `ErrVarintNonCanonical` (Go) /
  `Error::VarintOverflow` (Rust) on any multi-byte varint whose
  terminating byte is zero. The fast-path `Reader.ReadUvarintInline`
  and the standalone tag decoders (`DecodeTag`, `Reader.ReadTag`)
  carry the same check.
- **Length-prefix amplification DoS** in `decodeMap`, `decodeSlice`,
  and `decodePackedSlice`: a 3-byte varint header could claim
  1,000,000 entries (the default `MaxArrayLength` / `MaxMapSize`),
  forcing a multi-MB allocation against a few input bytes before any
  body byte was read. The decoders now bound the declared count by
  the reader's remaining bytes — every entry needs at least one
  wire byte — so the amplification factor is at most ~1×.
- **Rust `Writer::write_string` / `write_length_prefixed_bytes`** now
  use the 64-bit varint writer for the length prefix. The previous
  `value.len() as u32` silently wrapped strings larger than
  `u32::MAX` bytes, producing a corrupt length prefix that Go and
  Rust would each parse differently.
- **Rust `StreamReader::read_varint`** now applies the same 10th-byte
  overflow check the in-memory `Reader::read_varint64` does and the
  same canonical-encoding rejection. Without the checks, a crafted
  10-byte varint with byte 9 = `0x10` would silently shift bits
  beyond 64 and return a wrong large value.
- **Rust `Writer::write_tag(0, ...)`** now returns
  `Error::InvalidFieldNumber(0)` instead of silently emitting an
  empty `Vec<u8>`. The empty-emit case meant the next value byte was
  decoded as a malformed tag, corrupting the rest of the message.
- **TypeScript `Writer.writeVarint`** validates `value ≤ u32::MAX`
  and throws on overflow. The previous `value >>>= 7` coerced to
  int32, so a length above 2^32 silently became 0 followed by all
  the data bytes — a corrupt frame. `writeString` and
  `writeLengthPrefixedBytes` now route through `writeVarint64` so
  any `bytes.length` is encoded losslessly.

### Fixed (correctness)

- **Streaming pool didn't reset options**: `StreamWriter::Reset` and
  `StreamReader::Reset` now restore `DefaultOptions`. A pooled
  writer that was previously used with `SecureLimits` would
  otherwise inherit those tighter limits and reject otherwise-valid
  input.
- **`PutStreamWriter` documented**: the function does not flush; the
  contract is now clearly that the caller must call `Flush()` (or
  `Close()`) before returning the writer to the pool.
- **`atomicfile.Write` no longer leaks the file descriptor on panic**:
  if the user-supplied write callback panicked, the deferred cleanup
  removed the temp file but did not close the file handle. The close
  is now idempotent and always runs.
- **`cramberry format -w` now uses `atomicfile.Write`**: the previous
  `os.WriteFile` truncated the user's source `.cram` first, so a
  crash mid-write left it half-written or empty.
- **Schema import path traversal**: `(*Loader).resolveImportPath` now
  rejects any `import "..."` directive whose resolved path escapes
  the importing file's directory or any `SearchPaths` entry. A
  malicious `.cram` file with `import "../../etc/passwd"` previously
  resolved to wherever `filepath.Join` collapsed to.

### Fixed

- **T1-10 (per-field `,omitempty`)**: the reflection marshaller now
  honors the per-field `omitempty` tag. Previously the flag was parsed
  on the struct-info but never consulted, so `cramberry:"3,omitempty"`
  produced different bytes from any codegen path that respected the
  tag. With the fix, a field is skipped if either the global
  `Options.OmitEmpty` is on, or the field carries an explicit tag.
- **Map sort divergence**: map keys are now sorted unconditionally on
  encode, matching the codegen path. The reflection path previously
  gated sorting on `Options.Deterministic`, so calling
  `MarshalWithOptions(x, FastOptions)` produced different bytes from
  `x.EncodeTo(w)` for the same input — a violation of CLAUDE.md's
  determinism rule #1.
- **Rust generator emitted tags for absent fields**: every field's
  `write_tag` was unconditional, so a `None` `Option<T>` field produced
  a tag with no body and a zero scalar produced `tag + zero` instead of
  being skipped. The generator now wraps the tag write in a per-field
  zero/presence check that mirrors the Go generator and the reflection
  marshaller.
- **TypeScript generator only checked undefined / null**: zero scalars,
  empty strings, and empty arrays were emitted as `tag + payload`,
  diverging from Go's reflection canon. The generator now skips
  zero-valued fields the same way Go codegen does.
- **Schema reserved type-ID range now enforced**: validator rejects
  message and interface-implementation type IDs in the runtime-reserved
  1-127 range (1-63 builtin, 64-127 stdlib). User-defined IDs must be
  ≥ 128 to avoid silent collisions across the runtimes' built-in
  registries.
- **`schema.WriteToFile` is now atomic**: it stages to a temp file in
  the destination directory and renames over the target. A crash or
  encode error mid-write previously left a half-written `.cram` on
  disk that the next codegen step would silently feed garbage to.

### Changed

- Schema validator now rejects fields that stack mutually-exclusive
  modifiers (e.g. `required repeated`, `optional repeated`). Previously
  the parser accepted these silently because each modifier is
  independent on the AST. The error names every offending modifier so
  authors can pick one.
- Rust integration tests now resolve the Go-produced golden fixtures
  correctly. The path was `../../golden`, which silently missed the
  fixtures at `testdata/golden/` — the affected tests passed only
  because `load_golden` falls back to "skip" on missing files. Path
  corrected to `../../../testdata/golden`.

### Removed (BREAKING)

- `Options.Deterministic` field, `FastOptions` preset, `NoLimits`
  preset, and the `MaxTagSize` constant. Determinism (sorted map keys,
  canonical floats, fixed field order) is now an unconditional
  invariant — there is no way to opt out, because doing so would split
  reflection and codegen output. `MaxTagSize` aliased `MaxVarintLen64`
  with no callers.
- Dead public helpers in `pkg/cramberry`: `Reader.ReadSvarintInline`,
  `BufferPoolStats` and `GetBufferPoolStats`, `JSONStringValue`,
  `WrapError`, `NewFieldEncodeError`, the sentinel errors
  `ErrInvalidWireType` and `ErrTypeMismatch` (neither was returned
  anywhere), and the narrow JSON helpers
  `Format{Int,Uint}{8,16,32}ToString` /
  `Parse{Int,Uint}{8,16,32}FromString` (codegen only emits the 64-bit
  variants).
- `pkg/codegen.ToKebabCase` (no template referenced it) and
  `(*schema.Loader).GetSchema` (no callers).

### Added

- Rust integration tests for `ComplexTypes` (status enum, optional /
  required nested messages, repeated message lists, sorted maps) and
  `EdgeCases` (i32/i64/u32/u64 boundaries, unicode strings). Each
  suite cross-checks both round-trip identity and byte-for-byte
  equality against the Go-produced golden bytes.
- `cramberry.isOmittableZero` (internal) — the canonical zero-test
  the encoder uses to decide whether a field is skippable. The rule is
  now spelled out in the runtime so codegen, the reflection
  marshaller, and the TS / Rust generators all agree on it: skip
  bool-false / numeric-0 / empty-string / empty-bytes /
  empty-repeated / nil-pointer; always emit struct, map, array, and
  named-type fields.

### Earlier in Unreleased

- Dead public helpers in `internal/wire`:
  `PutFixed32`, `PutFixed64`, `PutFloat32`, `PutFloat64`,
  `IsNaN32`, `IsNaN64`, `IsNegativeZero32`, `IsNegativeZero64`,
  `PutUvarint`, `PutSvarint`. All callers used the `Append*` variants;
  the `Put*` and predicate forms had no production callers.
- Dead public helpers in `pkg/cramberry`:
  `GetWriterWithHint` (no callers), `IsRetryable` (always returned
  `false`), `ZeroCopyString.UnsafeString`, `ZeroCopyBytes.UnsafeBytes`
  (the safe `String`/`Bytes` accessors already provide the same access
  with generation validation).

## [2.0.0] - 2026-04-29

### Changed (BREAKING)

- **Wire format consolidation**: removed the legacy V1 wire-type surface and
  renamed the V2 names to drop the suffix. There is now a single canonical
  wire format. No on-the-wire bytes change — only the Go/TS/Rust API names.
  - `WireTypeV2Varint`/`WireTypeV2Fixed64`/`WireTypeV2Bytes`/`WireTypeV2Fixed32`/`WireTypeV2SVarint`
    → `WireVarint`/`WireFixed64`/`WireBytes`/`WireFixed32`/`WireSVarint`.
  - The legacy `WireFixed32 = 5`, `WireSVarint = 6`, `WireTypeRef = 7` constants
    and the `WireType` named type (in `pkg/cramberry/types.go`) are removed.
  - `Writer.WriteCompactTag` → `Writer.WriteTag`.
  - `Reader.ReadCompactTag` → `Reader.ReadTag`.
  - `Reader.SkipValueV2` → `Reader.SkipValue` (the legacy `SkipValue` and its
    V1 wire-type switch are removed).
  - `EncodeCompactTag`/`DecodeCompactTag`/`CompactTagSize` → `EncodeTag`/`DecodeTag`/`TagSize`.
  - `getWireTypeV2Cached`/`computeWireTypeV2` → `getWireTypeCached`/`computeWireType`.
  - `StreamWriter.WriteTag` and `StreamReader.ReadTag` (legacy V1 protobuf-style
    tag layout) are removed; nest a regular `Writer`/`Reader` inside a stream
    message instead.
  - TypeScript: `WireTypeV2` enum → `WireType`; `encodeCompactTag`/`decodeCompactTag`
    → `encodeTag`/`decodeTag`; `Reader.skipField` → `Reader.skipValue`.
  - Rust: `write_compact_tag`/`read_compact_tag`/`skip_field`/`skip_value_v2`
    → `write_tag`/`read_tag`/`skip_value`.
  - `internal/wire/tag.go` is reduced to `MaxFieldNumber` (the V1 protobuf-style
    tag helpers are removed).
- `pkg/cramberry/wire_v2.go` is renamed to `pkg/cramberry/wire.go`; tests in
  `wire_v2_test.go` move to `wire_test.go`.

## [1.5.5] - 2026-01-29

### Fixed
- **Writer integer overflow on 32-bit systems**: WritePackedFloat32/64 and WritePackedFixed32/64 now include `math.MaxInt` overflow checks before multiplication, matching their reader counterparts. Previously, large slice lengths could overflow on 32-bit systems causing buffer underallocation.
- **Go generator enum wire type detection**: The Go code generator now correctly detects local enums and uses `WireTypeV2SVarint` instead of incorrectly using `WireTypeV2Bytes`. This matches the TypeScript and Rust generators and prevents wire format corruption.
- **Deterministic interface implementation ordering**: Schema extraction now sorts interface implementations by name before processing, ensuring deterministic output critical for consensus systems.

## [1.5.4] - 2026-01-29

### Fixed
- **Defensive error handling in DecodeComplex64/DecodeComplex128**: Previously ignored errors from float decoding functions are now properly propagated
- **Consistent error returns in DecodeTag**: Returns 0 bytes consumed on ErrInvalidWireType to match other error cases
- **Nil pointer safety in schema extraction**: Added nil check for package in collectEnumValues to prevent crashes on builtin types
- **Cross-package enum wire type detection**: Fixed false positive where cross-package messages were incorrectly identified as local enums when they shared a name

## [1.5.3] - 2026-01-28

### Fixed
- **Code generator: Validate() no longer compares struct values to nil**: Required struct fields (NamedType) are value types and cannot be nil. The Validate() method now only generates nil checks for fields that are actually pointers (required scalars, optional fields, schema pointer types).
- **Code generator: Schema pointer fields decode correctly**: Fields with schema pointer types (e.g., `*Hash`) now decode correctly. Previously, the generated code would create double indirection (`var tmp *Hash; m.Field = &tmp` resulting in `**Hash`). Now correctly generates `var v Hash; m.Field = &v`.

## [1.5.2] - 2026-01-28

### Fixed
- **Interface validation for same-package imports**: Interfaces can now reference message types from multiple imported schema files that share the same package name
  - Added `findMessageInSamePackageImports()` for interface-specific type lookup
  - Completes same-package import support added in v1.5.1 (which only worked for field type references)

## [1.5.1] - 2026-01-28

### Added
- **Same-package import support**: Types from imported schemas with the same package name can be referenced without qualification
  - `Loader.GetImportedSchemas()` method to retrieve imported schemas by alias
  - `ImportedSchemas` option in `codegen.Options` for same-package detection
  - Validator now correctly allows unqualified type references for same-package imports
  - Code generator omits package qualifier and import statements for same-package types

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
