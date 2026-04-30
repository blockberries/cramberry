# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed (seventh review pass — JSON cross-runtime parity)

Adding cross-language JSON byte-parity to `codegen-parity-check.sh`
turned out to be more than a doc improvement: it surfaced four real
bugs across the JSON paths that the existing wire-format-only parity
check had been missing.

**Critical (cross-runtime byte divergence in JSON):**

1. Rust's polymorphic JSON helper (`to_json_<iface>`) used
   `serde_json::to_string(msg)` which formats integers as JSON
   numbers. Go and TS use codegen-emitted helpers that format
   integers as JSON strings (cramberry's deterministic-JSON
   convention). For any schema with a polymorphic field whose
   variants contain numeric fields, JSON output diverged:
   - Go/TS: `{"_type":"Dog","age":"5"}` (integer-as-string)
   - Rust:  `{"_type":"Dog","age":5}` (integer-as-number)
   Replaced the serde delegation with an explicit match-on-variant
   that splices `_type` into the codegen-emitted `to_json_<concrete>`
   output, matching Go and TS exactly.

2. TypeScript's enum JSON encoder emitted `value.toString()`,
   producing `"1"` for `Status.StatusActive`. Go and Rust emit the
   variant's source name `"STATUS_ACTIVE"` via `String()` /
   `match`. Diverged for every schema with an enum field.
   Replaced TS's encoder with a switch that maps each enum value to
   its source name. The TS decoder already accepted name-form;
   round-trip identity preserved.

3. Generated Go code unconditionally imported `"bytes"` whenever
   JSON was enabled, but `bytes.Equal` is only emitted by
   `jsonDecodePointer` (schema `*Type` fields) — schemas that use
   `optional` but no explicit `*Type` got an unused import. Tightened
   `hasOptionalPointer` to check for `*schema.PointerType` only.

4. Optional non-scalar NamedType fields (`optional Address office`)
   produced JSON decoder code that assigned a value to a pointer
   field: `m.Office = msg` where `m.Office` is `*Address`. The
   previous `jsonDecodeField` only allocated a tempVal for *scalar*
   pointer fields. Generalized the wrap to fire for any
   isPointerField + non-PointerType + non-MapType + non-interface
   case.

**Test infrastructure:**

`scripts/codegen-parity-check.sh` now generates code with `-json`
enabled (was `-json=false`), runs JSON encode probes in all three
runtimes against the same `parity_fixture.cram` data, and asserts
byte-identical output. The TS probe explicitly populates the empty/
zero fields (`emptyStr`, `emptyList`, `emptyBytes`, `zeroCount`,
`zeroRatio`) since TS object literals don't auto-default missing
fields.

After all four fixes, both wire-format AND JSON output is
byte-identical across:

    Go reflection == Go codegen == Rust codegen == TS codegen

…for the same `Sample` fixture, including the polymorphic Animal
field, all numeric edge cases, and the recursive Tree.

### Fixed (sixth review pass — JSON+interface, validator gaps, dropped warnings)

A sixth review found that the polymorphic codegen fix from round 5
covered the wire-format path but left the JSON path broken across all
three runtimes (Go and TS at least; Rust escaped via serde derive),
plus a regression in the bytes-import gating, plus three validator
gaps that had been silently accepting malformed schemas.

**Critical (uncompilable Go for any schema with -json + interface):**

1. The Go generator's JSON path emitted `m.X.ToJSON()` and
   `var msg Animal; msg.FromJSON(...)` for an interface field —
   `Animal` is a bare interface with only the marker method
   `is<Iface>()`, so neither method exists on the field type.
   Compile error:
   ```
   m.Pet.ToJSON undefined (type Animal has no field or method ToJSON)
   msg.FromJSON undefined (type Animal has no field or method FromJSON)
   ```
   Same root cause as the round-5 wire-format bug: the JSON path
   didn't check `isNamedInterface` before dispatching.

   Fixed by emitting `ToJSON<Iface>(v Iface)` / `FromJSON<Iface>(s string)`
   helpers next to the existing wire helpers, gated on
   `generateJSON`. JSON shape: `{"_type": "Variant", ...inner-fields-flat}`
   matching Rust's serde-tagged enum and the new TS helper. The field
   codec dispatches through these via new `jsonEncodeInterface` /
   `jsonDecodeInterface` paths.

   The TS template had the same bug: emitted `toJSON_Animal(...)` /
   `fromJSON_Animal(...)` calls without ever defining the helpers.
   Added matching `toJSON_<Iface>` / `fromJSON_<Iface>` functions to
   the TS template.

   Rust dodged this because the round-5 fix added `to_json_<iface>` /
   `from_json_<iface>` via `serde_json` (the enum derives
   `Serialize`/`Deserialize` with `#[serde(tag = "_type")]`). All
   three runtimes now emit the same JSON shape for the same logical
   input.

**High (regression I introduced in round 4):**

2. Generated Go code unconditionally imported `"bytes"` whenever JSON
   helpers were emitted, but `bytes.Equal` is only used in
   `jsonDecodePointer` (optional pointer JSON decode). Schemas
   without `optional <Type>` fields ended up with an unused import —
   Go's strict-import check failed the build. Now gated on a new
   `hasOptionalPointer` template helper.

**High (validator silently accepted malformed schemas):**

3. `enum E {}` — empty enum was accepted (the prior "must have a zero
   value" check used `len(enum.Values) > 0` as a guard, so empty
   enums skipped it entirely). The codegen then emitted `const ()`
   blocks with no entries — bizarre. Now rejected with "enum E must
   have at least one value".

4. `message M @999999999 { ... }` — `@TypeID` had a lower bound check
   (must be ≥ 128) but no upper bound. Type IDs share the wire-format
   field-number budget; values past `wire.MaxFieldNumber` (2^29-1)
   push the encoded varint past the documented spec and risk overflow
   on 32-bit runtime types. Now rejected with "type ID N exceeds
   maximum (536870911)" for both message TypeIDs and interface-impl
   TypeIDs.

5. `pkg/schema/io.go::loadFileInternal` filtered out every
   `SeverityWarning` from the validator before propagating errors to
   the CLI. Result: a schema like `int32 x = 19500;` (reserved
   protobuf field-number range) printed `Valid: file.cram` instead of
   the warning the validator produced. The CLI even had a path to
   surface warnings (`Severity == SeverityWarning` check, exit code
   2) but it was unreachable. Now warnings propagate; the CLI surfaces
   them and exits 2 if any are present.

**Regression tests added:**

- `TestGoGenerator_JSONInterfaceFieldCompiles` (codegen)
- `TestGoGenerator_NoOptionalPointer_NoBytesImport` (codegen)
- `TestValidate_EmptyEnumRejected` (validator)
- `TestValidate_TypeIDExceedsMaxRejected` with two sub-cases for
  message TypeIDs and interface-impl TypeIDs

Plus end-to-end runtime verification: a schema with `Animal pet = 1`
encodes to JSON `{"pet":{"_type":"Dog","name":"Rex"}}` and round-trips
correctly in all three runtimes.

### Fixed (fifth review pass — polymorphic codegen across all 3 runtimes)

A fifth review uncovered a major correctness bug: codegen for any
schema with an interface field referencing the interface type produced
**uncompilable code in all three runtimes**. The bug went unnoticed
because every committed example schema declared `interface` types but
never used one as a field type — so `make codegen-check` (which
compiles every fixture in all three languages) had no test that hit
the broken path.

**Critical (uncompilable code):**

1. Go: emitted `m.Resident.EncodeTo(w)` / `m.Resident.DecodeFrom(r)` for an
   interface field. The interface type only has the marker method
   `is<Iface>()`; `EncodeTo`/`DecodeFrom` exist on each concrete
   implementation, not on the interface. Any non-trivial schema with an
   interface field failed to compile (`undefined method`).

2. Rust: emitted `encode_animal(&mut __sub, &msg.resident)` /
   `decode_animal(...)` calls — but those functions were never
   generated. Same compile failure.

3. TypeScript: emitted `encodeAnimal(__sub, msg.resident)` /
   `decodeAnimal(...)` — also never generated.

**Fix:** each generator now emits a polymorphic `Encode<Iface>` /
`Decode<Iface>` pair next to the interface type definition. Wire layout
inside the surrounding length-prefix:

    [type_id varint] [concrete-type body terminated by end-marker]

Mirrors the Go reflection marshaller's `encodeInterface`. The field-
level encoder/decoder dispatches to these helpers when the field type
is a NamedType referencing an interface (rather than calling the
non-existent `EncodeTo` method).

The TypeScript surface changed materially: the old `Animal = Dog | Cat`
bare-union has no runtime discriminator, so it could never have been
encoded correctly. The new tagged union `Animal = { kind: 'Dog'; value: Dog }
| { kind: 'Cat'; value: Cat }` plus factory helpers
(`Animal.dog(v)`, `Animal.cat(v)`) gives an unambiguous runtime
discriminator. Existing code using the bare union was already broken
on encode; this is a strict improvement.

**Test gaps fixed alongside:**

- `pkg/schema/parser_test.go::TestParseErrorRecovery` previously
  asserted only `len(schema.Messages) > 0` — passing trivially when
  the parser stopped at the first error and returned just the leading
  good message. Now asserts both `Good1` (before the bad message) and
  `Good2` (after, requires recovery) are present.

- `pkg/cramberry/sort_test.go::TestCodegenMapDeterminism` was
  comparing sorted iterations to sorted iterations and would have
  passed even if `SortedMapKeys` was reduced to a no-op `for k := range m`.
  Now (1) explicitly asserts `sort.StringsAreSorted` on the returned
  keys, and (2) compares against an unsorted-iteration baseline,
  failing if the two ever match for 100 consecutive Go-runtime random
  seeds (statistically impossible for a real sort).

**Bonus — parity-fixture coverage:**

`scripts/parity_fixture.cram` now includes a polymorphic field:

    interface Animal {
      130 = Dog;
      131 = Cat;
    }
    message Sample {
      ...
      Animal pet = 31;
    }

`make codegen-parity-check` now exercises the polymorphic encode/
decode path in all four runtimes (Go reflection, Go codegen, Rust
codegen, TS codegen). Round 1's bug class would have been caught
automatically had this fixture existed.

Verified: all four runtimes produce byte-identical encoding for the
same logical input — including the polymorphic Dog variant
(TypeID=130).

### Fixed (fourth review pass — 3 issues + 6 regression tests)

A fourth independent review surfaced three more verified bugs (out of
many agent reports — most were filtered out as false positives).

**Critical (runtime panic in generated code):**

1. The Go generator's `jsonDecodePointer` emitted `*m.Value = strVal`
   directly, but `m.Value` is nil before any decode — so the very
   first `FromJSON` call with a non-null optional pointer field
   panicked on a nil-pointer dereference. Two compounding bugs were
   present in this single function:

   - The non-null branch dereferenced before allocating. Fixed by
     decoding into a fresh local of the element type and assigning
     its address.
   - The null-detection used `json.Unmarshal(rawValue, &isNull)` and
     branched on `err == nil && isNull`, but `Unmarshal(null, &bool)`
     returns no error AND leaves the bool at zero — so the null
     branch was unreachable. Fixed by using
     `bytes.Equal(rawValue, []byte("null"))` directly.

   The generated Go code now imports `"bytes"` automatically when
   JSON helpers are emitted. Regression test
   (`TestOptionalPointer_FromJSON_NonNullValue`) covers four cases
   that would have caught both bugs: string-set, number-set,
   both-set, and the previously-unreachable both-null branch.

**High (silent data corruption):**

2. `pkg/schema/io.go::WriteSchema` swallowed every `fmt.Fprintf`
   write error and unconditionally returned `nil`. Combined with
   `WriteToFile` (which wraps `WriteSchema` inside `atomicfile.Write`),
   a disk-full / broken-pipe / quota-exceeded condition would commit
   a truncated `.cram` file with no signal to the caller — the
   atomicfile guarantee held but the content was incomplete. Fixed
   by wrapping `out` in `bufio.Writer` once at entry; the deferred
   `Flush()` surfaces the cached underlying-write error without
   touching every Fprintf call site.

   Two new tests: `TestWriteSchema_PropagatesUnderlyingWriteError`
   uses a `failWriter` stub that errors after N bytes and asserts
   the error reaches the caller. `TestWriteSchema_HappyPath`
   ensures the wrapped writer doesn't break normal usage.

3. `pkg/cramberry/unmarshal.go::decodeMap` merged into a non-nil
   destination map (preserving stale entries not in the wire), while
   the slice path always replaced. Asymmetric and broke the
   determinism contract that `decode(encode(x)) == x` for reused
   destinations. Now always allocates a fresh map matching the
   slice path. Two regression tests
   (`TestUnmarshal_MapFieldReplacesExistingEntries`,
   `TestUnmarshal_SliceFieldReplacesExistingEntries`) lock both
   in.

### Fixed (third review pass — 5 issues + 2 regression tests)

After two prior cleanup passes, a third independent review surfaced
five more genuine bugs (filtered from many agent false positives —
"missing cycle detection", "MaxMessageSize divergence", "**T loses nil
distinction" all turned out not to be real after verification).

**Critical (cross-runtime divergence):**

1. TypeScript template wrapped every field emission in `if
   (tsZeroCheck)`, but `tsZeroCheck` for non-optional `NamedType` /
   `MapType` / `ArrayType` fields returned the `presence` expression
   (`field !== undefined && field !== null`). For type-correct callers
   this evaluated true (always emitted). For type-violating callers
   passing `undefined`, TS silently skipped the field while Go and Rust
   emitted it unconditionally — producing different bytes for the same
   logical input. Fixed by: (a) `tsZeroCheck` now returns empty string
   for non-optional composite fields, mirroring `goContext.zeroCheck`;
   (b) the TS template gained an `{{else}}` branch matching Rust's
   pattern, emitting the tag + body without an `if`-wrapper when
   zeroCheck is empty. Regression test added.

2. Go `StreamReader.ReadUvarint` accepted non-canonical varint
   encodings (`[0x80, 0x00]` decoding to 0). The non-stream
   `Reader.ReadUvarint`, the Rust stream, and the TS stream all rejected
   them. The Go stream layer was the odd one out — meaning a peer could
   send overlong varints that Go accepts and other runtimes reject.
   Added the `i > 0 && b == 0` rejection inside the byte-by-byte loop,
   matching `internal/wire.DecodeUvarint`. Regression test
   (`TestStreamReader_RejectsNonCanonicalVarint`) covers
   2-byte, 2-byte (overlong one), and 3-byte overlong encodings.

**Cleanup (Medium):**

3. `rust/src/registry.rs::TypeRegistration.type_id` was stored on the
   registration struct but never read — entries already key by ID in
   the `by_id` HashMap. Clippy reported it as dead. Removed.

4. `rust/src/json.rs::sort_map_keys_lexicographic` took `&mut
   Vec<String>`. Clippy: `&mut [String]` is more flexible and avoids
   forcing the caller to own a `Vec`. Codegen-emitted callers still
   pass `&mut keys` where `keys: Vec<String>` — auto-deref works.

5. Two unnecessary closure warnings in `rust/src/registry.rs`
   (`get_type_name`, `decode_polymorphic`):
   `.ok_or_else(|| Error::UnknownTypeId(id))` → `.ok_or(Error::…)`.
   Constructor is cheap; the closure indirection added nothing.

`cargo clippy --no-deps` is now clean for the lib build.

### Fixed (second review pass — 6 issues + bonus parity coverage)

A follow-up review surfaced six more bugs, all verified by reading the
actual code (rather than agent claims taken at face value). Five of six
were genuine; the parallel agents reported many false positives that
were filtered out before this list.

**Critical (cross-runtime byte / accept-reject divergence):**

1. TypeScript `Reader.skipValue` and `Reader.readTypeRef` read length
   prefixes via the 32-bit-capped `readVarint()`. The recent length-prefix
   fix updated `readString` and `readLengthPrefixedBytes` but missed
   these two siblings — Go's writer emits 64-bit length prefixes, so a
   skip on an unknown `Bytes` field with length > 5 varint bytes (or a
   polymorphic body > 4 GB) silently truncated. Both now go through
   `readSafeLength` (varint64 + narrow with `Number.MAX_SAFE_INTEGER`
   cap).

2. TypeScript `StreamReader.readVarint` looped only 5 iterations (32-bit
   cap) AND had no non-canonical-encoding rejection. Mirrors the
   `Reader.readVarint` bug fixed last pass — the stream layer was a
   parallel implementation that wasn't updated. Now reads via the same
   64-bit varint shape with overlong-zero rejection and
   `MAX_SAFE_INTEGER` cap.

3. Rust `Reader::read_string`, `read_length_prefixed_bytes`, and
   `skip_value` (Bytes branch) used `read_varint()` (u32, 5 bytes max).
   Same root cause as the TS bug. Replaced with a new
   `read_safe_length()` helper that goes through `read_varint64()` and
   guards against `usize::MAX` overflow on 32-bit targets.

**High (correctness):**

4. `pkg/schema/compat.go` did not detect field-modifier transitions.
   `optional → required` is breaking (an old encoder may have omitted
   the field; a new strict decoder rejects). `required → optional` is
   wire-compatible but worth a warning. Added `FieldOptionalToRequired`
   breaking-change type and emit it from `checkMessageCompat`. Added
   unit tests for both transitions.

5. TypeScript `Registry.register` silently overwrote existing
   registrations on duplicate name or duplicate type ID. Go's
   `Registry.RegisterTypeWithID` returns `ErrDuplicateTypeID`.
   Now throws `DuplicateTypeRegistrationError` on conflicting bindings,
   while still being idempotent for the same `(name, typeId)` pair.

**Medium:**

6. `Writer.Reset()` cleared `buf` / `depth` / `err` / `frozen` but not
   `opts`. `MarshalWithOptions` masks the leak by calling `SetOptions`
   immediately after `GetWriter`, but direct `GetWriter` callers
   inherited the prior caller's `SecureLimits`. Now restored to
   `DefaultOptions` on every Reset.

**Bonus — parity-fixture extension:**

`scripts/parity_fixture.cram` now includes a 20 000-byte `string`
field. The encoded length prefix is `varint(20000) = 0xa0 0x9c 0x01`
(3 bytes) — exercising the multi-byte length-prefix decode path on
every runtime. Combined with the existing `repeated int64` field
(added in the previous pass), the parity check now covers both
hotspot bug classes that round 1 missed: wrong wire-type tag for
repeated scalars, and 32-bit-truncation of multi-byte length prefixes.

`make codegen-parity-check` confirms Go reflection == Go codegen ==
Rust codegen == TS codegen byte-for-byte for the extended fixture.

### Fixed (review pass — 12 verified issues)

A fresh code review surfaced 12 real bugs across the runtime, generators,
and tooling. Three are consensus-affecting (cross-runtime byte
divergence); the rest range from DoS hardening to cleanup.

**Critical (cross-runtime byte divergence):**

1. TypeScript generator emitted the *element*'s wire type for repeated
   scalar fields (`WireType.SVarint` for `repeated int64`). The body is
   length-prefixed bytes, so Go used `WireBytes`. Same bug as the
   recently-fixed Rust generator. Caught by adding a `repeated int64`
   field to `parity_fixture.cram` — the parity check now catches this
   class of drift automatically.

2. TypeScript reader accepted non-canonical varints (`[0x80, 0x00]`
   decoding to 0). Go's `internal/wire/varint.go` rejects them as
   `ErrVarintNonCanonical`. Same input → different
   accept/reject across runtimes; hashing-over-bytes diverged. Both
   `readVarint` and `readVarint64` now reject overlong encodings.

3. Validator did not check for duplicate `@TypeID` annotations across
   messages within a single schema. Two messages claiming the same
   polymorphic ID would pass validation, then collide at runtime
   registration. `collectTypes` now tracks IDs across messages and
   reports them inline.

**High (correctness / hardening):**

4. TypeScript reader had zero per-field limit enforcement (Go enforces
   `MaxStringLength`, `MaxBytesLength`, `MaxArrayLength`, `MaxMapSize`,
   `MaxDepth`). Ported the `Limits` struct, `DEFAULT_LIMITS` /
   `SECURE_LIMITS` presets, and `ReaderOptions{limits, validateUtf8}`.
   Generated `unmarshal*` functions accept the options through; the
   codegen-emitted `readArray` / `readMap` helpers and nested-message
   decoders now invoke `checkArrayLimit` / `checkMapLimit` /
   `enterNested` and propagate the parent reader's limits to
   sub-readers.

5. `internal/atomicfile/atomicfile.go` swallowed dir-fsync errors
   (`_ = dirF.Sync()`). Now propagates them — a successful return is
   genuinely durable, not just renamed.

6. `cmd/cramberry/main.go` initialized `importPaths` as a nil
   `importPathFlag` map; downstream codegen received nil when `-M`
   was never passed. Initialised as an empty map so iteration is safe.

**Medium:**

7. TypeScript readers used a 32-bit-capped `readVarint()` for length
   prefixes, rejecting valid Go-encoded payloads >4 GB. New
   `readSafeLength()` reads via `readVarint64` and narrows to a JS
   number with a `Number.MAX_SAFE_INTEGER` cap. Cross-runtime parity
   restored for any practical length.

8. TypeScript reader had no UTF-8 validation option. `TextDecoder` in
   default lenient mode silently substitutes U+FFFD for invalid bytes,
   diverging from Go's `ValidateUTF8` flag. Added
   `ReaderOptions.validateUtf8` using `TextDecoder("utf-8", { fatal: true })`.

9. Documented that `pkg/cramberry/stream.NewStreamReader` uses
   `DefaultLimits` (100 MB per ReadBytes call); recommended
   `NewStreamReaderWithOptions(r, Options{Limits: SecureLimits})` for
   adversarial endpoints. Behaviour unchanged; doc clarified.

**Low (cleanup):**

10. Removed `SortedMapKeysFloat32`/`SortedMapKeysFloat64`/
    `SortedMapKeysBool`. The schema validator rejects bool/float map
    keys at parse time, so the codegen path never emitted calls to
    these — they were unreachable from schema-driven flows. The
    reflection marshaller's inline `CompareFloatKeys` /
    `CompareFloat32Keys` cover its own use case.

11. `pkg/extract/collector.go` flattened anonymous (embedded) Go
    fields into the schema as if they were named fields, losing the
    embedding semantic on round-trip. Now skipped — users who want an
    embedded type to participate in the wire format must declare it as
    a named field.

12. `pkg/schema/lexer.go::Peek` saved `pos`/`line`/`column` but not
    `start`/`startPos`. The latter is what `Next()` stamps into the
    next token's `Position`, so a Peek between tokens corrupted the
    next real token's position. All five fields are now restored.

### Added (cross-language benchmark suite)

Extended `internal/bench/` to cover all three runtimes against
protobuf and JSON baselines:

- **Rust** (`internal/bench/rust/`) — criterion benches with
  `prost`-generated protobuf baseline and `serde_json`. Covers
  SmallMessage, Metrics, Person, Document, Event, Batch100, and
  Batch1000 — encode + decode for every codec.
- **TypeScript** (`internal/bench/ts/`) — `tinybench` harness with
  `protobufjs` and native JSON. Same scenarios as Rust.
- **Makefile** — new targets `bench-go`, `bench-rust`, `bench-ts`,
  `bench-cross`, `bench-sizes`. `bench-cross` runs all three
  languages sequentially.
- **README** (`internal/bench/README.md`) — fully rewritten to
  document all three runtimes, how to filter, what each tool
  reports, and the caveats around cross-language perf comparisons.

The Go side already exercised cramberry codegen + reflection vs
protobuf vs JSON; this extension brings Rust and TS to parity so
the same fixtures can be measured under every runtime.

### Fixed (Rust generator — four cross-runtime drift bugs)

The new cross-language bench harness exercises shapes the existing
parity fixture didn't (`type` keyword fields, `repeated` scalars,
`repeated` messages, large maps in real fixtures), and surfaced
four real Rust-generator bugs:

1. **`type` keyword escape at access sites.** The struct field was
   declared `r#type` but `rustWriteField`, `rustZeroCheck`, and
   the JSON encode/decode field generators used the bare
   `msg.type`. Any schema with a `type` field failed to compile
   (`expected identifier, found keyword`). Fixed by routing every
   field access through `rustFieldName(f)`, which already knew
   about Rust reserved-keyword escaping.

2. **Repeated/map decode read elements from the outer reader.**
   The generated body shadowed `__data` and `sub_reader`, then
   read the count from `sub_reader` — but the loop body called
   `reader.read_*()?` (inherited from `rustReadValue` recursion),
   draining the *outer* stream. Decoding a `repeated int64` field
   walked past the next field's tag and failed with a multi-KB
   `BufferUnderflow`. Fixed by shadowing the outer `reader` so
   element reads consume the sub-buffer.

3. **Wrong wire-type tag for repeated scalar fields.** The
   generator emitted the *element's* wire type in the field tag
   (`WireType::SVarint` for `repeated int64`), but the body is
   length-prefixed bytes (count + packed scalars) — Go used
   `WireType::Bytes` for this. Bytes-on-wire diverged across
   runtimes for any schema with a packed scalar array. Fixed by
   forcing `rustWireType` to return `WireType::Bytes` for any
   repeated field.

4. **Spurious length-prefix wrapper on repeated message decode.**
   For `repeated Tag`, the encoder concatenated `body 0x00`-
   separated tag bodies inside one outer length-prefix. The
   decoder, reusing the single-message decode pattern, expected
   each element to start with its own length prefix and called
   `read_length_prefixed_bytes` per element — reading the first
   tag's first byte as a length and immediately failing. Fixed
   by special-casing the repeated-message branch in
   `rustReadValue` to call `decode_<name>(&mut reader)` directly.

After all four fixes, Rust codegen produces byte-identical output
to Go codegen for every Document/Person/Event/Batch fixture in the
new bench suite — confirmed end-to-end by encoding the same input
through both runtimes and diffing the hex.

### Extended (parity fixture — recursive types)

The codegen-parity fixture now also exercises a recursive type
(`message Tree { string label; repeated Tree children }`). A
small 4-node tree (root → [left → [leftleaf], right]) round-trips
byte-identically across all four runtimes:

    Go reflection == Go codegen == Rust codegen == TS codegen

Recursive types stress every code path that has to forward-declare
or self-reference a generated type definition (Rust struct that
contains `Vec<Self>`, Go's `[]Tree` field on `Tree`, TS interface
referencing itself). All four generators handle it cleanly. No
drift.

### Extended (parity fixture surface — omit-empty + multi-byte lengths)

The codegen-parity fixture now also exercises:

- Five intentionally zero/empty fields (`empty_str`, `empty_list`,
  `empty_bytes`, `zero_count`, `zero_ratio`). All four runtimes
  agree these should be omitted on the wire. Previously the
  omit-empty paths were code-reviewed but never byte-compared
  across runtimes.
- A long string (≈ 215 bytes — UTF-8 + 200 ASCII repeats), forcing
  the length prefix into a multi-byte varint encoding. Confirms
  multi-byte varint length encoding is byte-identical across all
  four runtimes (`d7 01` for length 215).

No drift surfaced — clean confirmation that the omit-empty and
multi-byte-length paths are already aligned across runtimes.

### Fixed (consensus-critical: int8/uint8 wire format mismatch)

`Writer.WriteInt8`/`WriteUint8` (and the matching readers) used **raw
byte** encoding, while `computeWireType` claimed `SVarint`/`Varint`
for those types in the tag. The Rust runtime correctly encoded them
as svarint/varint per the wire-type tag.

Concrete consequence: `int8(-7)` produced `0xf9` on the Go side
and `0x0d` on the Rust/TS side — different bytes for the same
logical value, and Rust decoding Go's bytes would treat `0xf9` as
a varint continuation byte and consume the next field.

Fixed: `WriteInt8`/`WriteUint8`/`ReadInt8`/`ReadUint8` now route
through the varint paths, matching `WriteInt16`/`WriteInt32` and
the Rust/TS runtimes. Surfaced by adding `int8`/`uint8`/`int16`/
`uint16` fields to the parity fixture.

### Fixed (consensus-critical: `*int32` etc. wire-type wrong)

`computeWireType` returned `WireBytes` for ALL pointer types,
including pointer-to-scalar like `*int32` / `*float32`. Codegen
correctly used the underlying scalar's wire type (`SVarint` for
`*int32`, `Fixed32` for `*float32`). Reflection's `WireBytes`
choice required an outer length prefix (the pass-7 fix), which
Codegen didn't emit. So:

- Reflection: `tag(WireBytes) | length | svarint`
- Codegen:    `tag(SVarint) | svarint`

Different bytes, different framing.

Fixed: `computeWireType` now recurses into the pointee type for
`reflect.Ptr`, so `*int32` → `SVarint` (matching codegen). The
pass-7 length-prefix wrapping is no longer needed and was removed
from `needsBodyLengthPrefix` for pointer-to-scalar.

### Extended (parity fixture surface, again)

The codegen-parity fixture now also exercises `float32`,
`uint32`/`uint64`, `int8`/`uint8`/`int16`/`uint16`, an `int32`-keyed
map, an `optional int32` (pointer-to-non-self-delimiting-scalar),
and an `optional Address` (pointer-to-message). All four runtimes
produce byte-identical output:

    Go reflection == Go codegen == Rust codegen == TS codegen

over a 222-byte payload covering every common type.

### Fixed (consensus-critical: `*string` / `*[]byte` were double-wrapped by reflection)

The earlier "pass-7" fix to `needsBodyLengthPrefix` over-wrapped
pointer-to-self-delimiting fields. A `*string` field was emitting:

    tag(WireBytes) | outer-len | inner-len | bytes

while the Go codegen path emitted:

    tag(WireBytes) | inner-len | bytes

These are different byte streams for the same logical input — Go
reflection and Go codegen disagreed. (Rust and TS codegen
produced the codegen shape, so reflection was the outlier.)

The bug was masked because no parity test exercised an
`optional string` field; this pass's expanded fixture (nested
messages, maps, optional pointer) surfaced it. `WriteString`
already emits `varint(len) | bytes`, which is exactly what
`SkipValue(WireBytes)` consumes — wrapping again was redundant.

`needsBodyLengthPrefix` now returns `false` for pointer-to-string,
matching the codegen path. The forward-compat skip test for
`*string` was added so the regression is caught.

### Extended (parity fixture surface)

The codegen-parity fixture schema now exercises:

- a 0-valued enum (cross-language default consistency)
- a nested message (length-prefixed body)
- a repeated nested message (count + length-prefix-wrapped)
- a `map<string, int32>` (sorted-key emission)
- an `optional string` (pointer-to-self-delimiting)

Confirmed all four paths produce byte-identical output:

    Go reflection == Go codegen == Rust codegen == TS codegen

over a 178-byte payload that touches every common drift hotspot.

### Added (cross-language byte-parity check)

- New `make codegen-parity-check` target. The script generates Go,
  Rust, and TypeScript code from a fixture schema covering the
  common scalar + string + bytes + repeated types, compiles each,
  encodes the same logical data through every runtime, and asserts
  the byte streams are all identical:

      Go reflection == Go codegen == Rust codegen == TS codegen

  The TS path uses `npm install file:typescript` to wire the
  generated code's `from '@cramberry/runtime'` import to the local
  package via Node's "exports" field, so the probe runs against the
  same surface real users would consume.
- Wired into `make integration-test`. CI now catches drift between
  the four encoders the moment any of them produces different
  bytes for the same input. This was previously **untested** —
  the existing integration tests use hand-rolled encoders against
  Go-produced golden bytes, not generator output.

### Fixed (TypeScript codegen — wrong package name)

The TypeScript generator emitted `import { ... } from 'cramberry'`,
but the npm package is published as `@cramberry/runtime` (per
`typescript/package.json`'s `name` field). Users running `npm install
@cramberry/runtime` and then `cramberry generate -lang typescript`
got `Cannot find module 'cramberry'` at compile time. The generator
now emits the correct `from '@cramberry/runtime'`. The
`ts-codegen-check` script's tsconfig path-alias was updated to
match.

### Verified (cross-language byte parity)

Probed the generated Go and Rust output for an all-types schema
(bool, int32, int64, string, bytes, float64, repeated string)
encoding the same logical data. Both runtimes produced the same
105-byte hex output:
`1001285338a8e8c8e99707440e68656c6c6f2c20e4b896e7958c215404deadbeef626e861bf0f921094074120305616c70686104626574610567616d6d6100`
This confirms the wire format is byte-identical across the Go
codegen and Rust codegen paths for at least this surface — a
property that previously had no end-to-end test (the existing
integration tests use hand-rolled encoders, not generated code).

### Fixed (codegen drift + `-json=false`)

- **`-json=false` actually works now**. The flag was registered and
  threaded through to `Options.GenerateJSON`, but the Go template
  emitted `ToJSON` / `FromJSON` unconditionally — the generated
  helpers were always present, with `encoding/json` / `fmt` /
  `strings` imports they no longer needed. Now the JSON helpers
  AND their imports are gated behind `{{if generateJSON}}`.
- **JSON struct tags decoupled from `-json=false`**. Struct tags
  (`json:"foo"`) are useful even when the cramberry-specific
  `ToJSON` helpers are off — they let users feed the same struct
  into Go's stdlib `encoding/json`. The flag now only controls
  the helper methods, not the struct tags.
- **Committed fixtures were stale**. `test/integration/gen/interop.go`
  and `testdata/generated/json_test.go` had been hand-trimmed and
  drifted from current generator output (most importantly,
  `testdata/generated/json_test.go` was missing the
  length-prefix-wrap that was added to repeated fields in an
  earlier pass — the wire bytes it produced no longer matched what
  reflection produced). Both regenerated.

### Added (tooling)

- New `make generate-fixtures` target regenerates every checked-in
  code-generated file from its source schema. `make generate-test`
  now invokes it as a prerequisite, so a forgotten regeneration
  surfaces immediately as drift.

### Fixed (TypeScript codegen — `tsc --strict` errors)

The TypeScript generator emitted code that didn't pass `tsc --strict`
for any non-trivial schema. Verified by tsc-checking the output of
every example/testdata schema against the runtime.

- **Optional non-pointer field encode** passed `Option<T>` (i.e.
  `string | undefined`) directly to helpers like `escapeJSONString`
  that require `T`. The JSON encoder now wraps each optional field
  in an `if (msg.x !== undefined && msg.x !== null) { ... } else
  { result += 'null'; }` block.
- **`parseNumberFromJSON` / `parseBigIntFromJSON` got `unknown`** from
  `Object.entries` and from `obj['key']` lookups — `tsc --strict`
  rejected the call. Added `as string | number` casts. The runtime
  parsers do their own narrowing internally, so the cast is purely
  a strict-mode shim.
- **`new Set([])` for empty allowed-fields list** inferred as
  `Set<never>`, so `allowedFields.has(key: string)` failed. Now
  emits `new Set<string>([])` so the empty case still typechecks.
- **TS runtime `StreamWriter.view`** was assigned twice but never
  read — `tsup`'s DTS build (strict mode) failed with TS6133.
  Removed the dead field.
- **Unused `decodeTag` import in `reader.ts`** triggered the same
  TS6133. Dropped from the import list.

### Added (tooling)

- New `make ts-codegen-check` and `make go-codegen-check` targets,
  matching the existing `rust-codegen-check`. Each generates output
  for every schema in `examples/` and `testdata/` and compiles it
  against the appropriate runtime. Wired into `make codegen-check`
  (the umbrella) and `make integration-test`. CI now catches any
  generator output that doesn't compile.

### Fixed (Rust codegen — multiple compile errors)

The Rust generator emitted code that did not type-check for any
non-trivial schema. Fixed by compiling every generated output for
`examples/schemas/user.cram`, `testdata/schemas/interop.cram`, and
`testdata/schemas/json_test.cram` against the Rust runtime crate.

- **`int8`/`int16`/`uint8`/`uint16` literally appeared as Rust type
  names** in JSON decode (`as int8`) — fixed by routing through
  `rustScalarType` to get `i8`/`i16`/`u8`/`u16`.
- **`write_svarint(msg.int8_val)` mismatched types** — Rust's
  `write_svarint` takes `i32`; narrow integer fields now widen via
  `as i32` / `as u32` before the call. Same on the read side
  (`as i8`, `as u16`, etc.).
- **`format!("invalid key: {{}}", e)` had no placeholder** — `{{`
  and `}}` are Rust's `format!` escapes, so the macro saw no
  argument and rejected `e`. Now emits `format!("invalid key: {}",
  e)`.
- **`HashMap<String, V>::get(&k)` failed on `&&str`** — `let k =
  key_str.as_str()` made `k` a `&str`, then `&k` was `&&str` which
  doesn't satisfy `Borrow<&str>`. Fixed by passing `k` directly for
  string keys; numeric keys still pass `&k`.
- **`reader.read_tag()` returns `FieldTag` (struct), not a tuple**
  — codegen unpacked it as `let (field_num, wire_type)` which
  doesn't destructure a struct. Now uses `tag.field_number` /
  `tag.wire_type`.
- **`(*v) =` in JSON decode of pointer fields** failed because `v`
  was bound non-mut. Now `let mut v: Box<T> = ...`.
- **Pointer-field encode passed `&Box<T>` to `write_svarintN`**
  — used `as_deref()` so `inner` is `&T`, then dereferences once
  for scalar values.
- **`optional T x` Rust type is `Option<T>`** — encoder used the
  Option directly as if it were T; now unwraps via `.as_ref().unwrap()`
  inside the rustZeroCheck guard. Decoder wraps the result in
  `Some(...)`.
- **`!= 0 as _`** failed type inference — now uses `!= 0 as <T>`
  with the explicit Rust scalar type.

A new `make rust-codegen-check` target (added to `make integration-test`)
now generates Rust output for every schema in `examples/` and
`testdata/` and `cargo build`s it against the runtime crate. CI will
catch this class of regression going forward.

### Fixed (codegen)

- **Rust JSON helpers now compile**. The generator emitted
  `Result<String, String>` while `cramberry::Result<T>` (in scope from
  the prelude `use`) is single-type-arg — every Rust target's JSON
  path failed `cargo build`. JSON helpers now use
  `std::result::Result<...>` qualified.
- **Rust `optional` field encode now uses `is_some()`**, not
  `is_empty()`. The previous `rustZeroCheck` fell through and
  produced `!msg.foo.is_empty()` for `Option<String>` fields, which
  doesn't typecheck. Same fix on the TS side: `optional` fields now
  check presence only, not length.
- **Validator rejects `map<bool, V>`**. Bool keys produced
  uncompilable Go (the JSON template assumed string-or-numeric keys
  and indexed `map[keyStr]` against a `map[bool]V`). Bool keys offer
  no value (only two possible keys); the validator now refuses them.
- **Validator requires a 0-valued enum variant** (was a warning).
  Without one, the three runtimes disagreed on the default: Rust's
  `#[default]` derive picks the first declared variant, Go's
  `var x EnumType` is `0` (not a valid variant), TS leaves the field
  at `0`. Requiring `UNKNOWN = 0` (or any 0 variant) keeps decode
  defaults consistent across languages.

### Fixed (memory safety)

- **`ZeroCopyBytes` now uses a full-slice expression** so its cap
  equals its len. The doc comment said "do not modify"; nothing
  enforced it. With a full-cap slice into the Reader's data,
  `append(zcb.Bytes(), 'x')` could overwrite bytes the reader hadn't
  yet decoded, silently corrupting subsequent reads.
- **`Writer.Reset` no longer aliases a frozen writer's `Bytes()`
  result**. The previous flow — `b := w.Bytes(); w.Reset(); w.WriteX();
  use(b)` — overwrote `b[0:N]` in place; `len(b)` was unchanged so
  the user saw silently corrupted data. Reset on a frozen writer now
  allocates a fresh backing array.

### Fixed (docs)

- `internal/bench/README.md` perf-ratio claims updated. The README
  said "1.7-2.4x faster for encoding" while measured ratios are
  ≈3x. Now phrased as "roughly 3x" with a note that users should
  run the benchmarks locally.

### Fixed (extract round-trip)

- **`[]byte` and `[N]byte` extracted as `repeated bytes`** — that's
  not a thing in the schema; it produced unparseable round-trip
  output and would round-trip to a `[][]byte` Go field with totally
  different wire bytes from the original `[]byte`. The extractor
  now treats byte slices as the schema scalar `bytes` (not as a
  repeated field).
- **Extracted field names that collide with reserved words** are now
  escaped with a trailing `_`. A Go field named `Optional` /
  `Required` / `Repeated` / `Map` / `Package` (etc.) used to extract
  as `optional optional = 1;` — the parser then rejected the
  resulting schema with `expected field name`. Now produces
  `optional_`. Same for built-in scalar type names (`bytes`,
  `int32`, …) where the collision was confusing rather than fatal.

### Fixed (UX / docs)

- **README install command** now points at `make install` /
  `make build`, with a note that `go install` produces a binary
  reporting `cramberry version dev (unknown, unknown)` because the
  ldflags injection happens in the Makefile.
- **ARCHITECTURE.md interface example** now uses the actual schema
  syntax (`128 = User;`) rather than the swapped form
  (`User = 128;`) which the parser rejects.
- **ARCHITECTURE.md per-field omitempty** note updated: the tag is
  honored as of the T1-10 fix; the doc said "currently ignored."
- **Duplicate `examples/user.cram`** removed. The canonical schema
  lives at `examples/schemas/user.cram`; the older top-level copy
  defined a different (and lesser) shape, leaving new users to
  guess which one was authoritative.

### Fixed (memory safety)

- **`PutStreamWriter` auto-flushes** any buffered bytes before
  dropping the underlying writer reference. The previous behavior
  silently dropped up to 4 KiB of output every time a caller forgot
  to call `Flush` — no observable signal. Flush errors are stored
  on the StreamWriter for callers who do explicit Flush; auto-flush
  is the safety net.

### Fixed (consensus-critical, seventh review pass)

- **Forward-compat: pointer-to-primitive fields used to corrupt the
  message tail when skipped by an older decoder.** A `*int32`,
  `*float64`, `*bool` (etc.) field emitted `tag(WireBytes) | <raw
  varint or fixed body>` — no length prefix — so an old decoder
  calling `SkipValue(WireBytes)` read the first body byte as the
  length, mis-framed the rest of the message, and corrupted every
  following field. `needsBodyLengthPrefix` now returns true for these
  pointer-to-scalar shapes so the field body is wrapped exactly the
  way `SkipValue(WireBytes)` expects.
- **Forward-compat: `complex128` fields had the same bug.** They
  emitted `tag(WireBytes) | <16 raw bytes>`; the first byte (a
  float64 mantissa byte) was almost always a varint continuation
  byte. Now wrapped with a length prefix.
- **NaN map keys are rejected at encode time.** Distinct NaN bit
  patterns canonicalize to the same wire bytes (information loss),
  AND `MapIndex(nan)` always returns the zero `reflect.Value`
  (because Go map lookups use `==` and NaN ≠ NaN), so values silently
  became zero on encode. Three NaN-keyed entries with values 1, 2, 3
  used to round-trip to three NaN-keyed entries with values 0, 0, 0.
  The encoder now returns an explicit error, matching the JSON
  encoder which already refused NaN.

### Fixed (memory safety)

- **`PutWriter` on a frozen Writer no longer pools the buffer.**
  Calling `Bytes()` froze the writer (so subsequent writes would
  error), but `PutWriter` happily reset and pooled the buffer
  anyway. A user pattern of `b := w.Bytes(); PutWriter(w);
  go consume(b)` could then race the caller's read against another
  goroutine's writes via a future `GetWriter()`. The contract is
  now: call `BytesCopy()` if you want to safely return the writer to
  the pool.

### Removed

- `Reader.Data()` — zero production callers, and it returned the
  underlying buffer in mutable form, letting users invalidate the
  zero-copy string/bytes contract without ever tripping the
  generation check. The one test caller was switched to
  `Reader.Remaining()`.

### Changed (UX & DX)

- **`-lang ts` and `-lang rs` now work**. The README's primary usage
  example showed `cramberry generate -lang ts ...`, which used to fail
  with `unsupported language: ts`. The generator registry now accepts
  the short aliases (`ts`, `rs`, `golang`, `js`) alongside the
  canonical names.
- **`cramberry --version` / `-V`** are now recognised at the top
  level, alongside the existing `version` subcommand.
- **`cramberry help <subcommand>`** now prints the subcommand's own
  usage rather than the top-level help.
- **`cramberry format -d`** actually emits a unified-style diff
  instead of silently printing the formatted output (the flag was
  parsed and discarded — `_ = diff // TODO`). Exits 1 if any file
  would change, so the command can gate a CI step.
- **`cramberry generate` no longer rejects warnings**. Previously the
  loader's diagnostic slice was treated as fatal regardless of
  severity; a schema that `cramberry validate` accepted (warnings
  only) would still fail to generate. Errors and warnings are now
  classified the same way both subcommands.
- **CLI errors are prefixed with the subcommand name** and the
  affected file path: `cramberry generate: schemas/foo.cram: ...`
  instead of the bare `Error generating code: ...` that hid which
  command and which file produced the message.
- **Schema parser surfaces the underlying parse-int error** on
  field-number / type-ID / enum-value / array-size literals
  (`invalid field number: integer literal "..." out of int range`)
  instead of swallowing it.
- **Decode errors gain field context**. A varint failure deep in a
  nested struct field now reports `Type.Field` and the byte offset
  rather than just `cramberry: invalid varint at offset 27`.
- **Bad `cramberry:"..."` struct tags are returned errors, not
  panics**. Importing a third-party type with a malformed tag
  previously crashed the host process the first time `Marshal` was
  called on it; now `Marshal` returns an error.

### Changed (Makefile / dev tooling)

- `.PHONY` now lists every phony target (`test-short`, `tidy`,
  `deps`, `verify`, `check`, `ci`, `pre-commit`, `generate-test`,
  the new `ts-fmt`/`ts-lint`/`rust-fmt`/`rust-lint`/`lint-all`/
  `fmt-all`). Without this a file or directory named `check` or
  `ci` could shadow the target.
- `make lint` now errors out if `golangci-lint` is missing
  (CLAUDE.md mandates a clean lint after every change; a silent
  skip let regressions slip past). The install hint is unchanged.
- `make ci` is now a strict superset of `make check`: it runs the
  Go-only checks plus the cross-language integration tests.
- `make all` no longer mutates files. It uses `fmt-check` (the new
  `gofmt -l` based target that fails on unformatted files) instead
  of `fmt`, so running the default target on a clean tree leaves
  the working copy clean.
- New targets: `pre-commit` (mutating fmt + checks), `ts-fmt`,
  `ts-lint`, `rust-fmt`, `rust-lint`, `lint-all`, `fmt-all`.
- `make help` honours `NO_COLOR` and detects TTY before emitting
  ANSI escape codes.

### Refactored (DRY)

- `resolveNamedEnum` / `isNamedEnum` consolidated into
  `pkg/codegen/generator.go::ResolveNamedEnum` / `IsNamedEnum`. The
  three generators previously had byte-identical 20-line
  implementations of the same lookup; now they call the shared
  helpers, eliminating drift risk between languages.
- `pool.go` `GetBuffer` / `PutBuffer` removed (zero callers, and the
  size-class lookup was wrong — a returned buffer's capacity could
  be smaller than the requested size hint).

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
