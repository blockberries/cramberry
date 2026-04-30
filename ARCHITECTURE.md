# Cramberry — Architecture

## Goals

1. **Determinism**: byte-identical output across Go, TS, Rust for the same
   logical input. Required for SignDocs and block hashes.
2. **Compactness**: comparable to Protobuf, 2–3× smaller than JSON.
3. **Performance**: faster than Protobuf for typical messages thanks to a
   1-byte tag for fields 1–15.
4. **Schema-driven**: `.cram` files generate idiomatic Go / TS / Rust types.
5. **Cross-language conformance**: a single golden-file suite verifies all
   three ports stay in sync.

## Wire format

A message is a sequence of `(tag, value)` pairs terminated by a single
`0x00` end marker.

```
message      := body 0x00
body         := field*
field        := tag value
tag          := compact_tag | extended_tag
value        := scalar | length-prefixed-body
```

A top-level `Marshal` output is `body 0x00`. When a struct value appears as
a *field value* in another message, its body is wrapped in a length-prefixed
payload: `tag length:varint body 0x00`. The wrapping lets `SkipValue(WireBytes)`
skip a field whose schema the decoder doesn't recognize without misreading
the first body byte as a length.

The same length-prefix wrapping applies to repeated fields (count + elements)
and map fields (count + entries). Strings and `[]byte` already self-prefix
via their write helpers and aren't double-wrapped.

### Compact tag (fields 1–15)

```
bit:  7 6 5 4 3 2 1 0
      [fieldNum ] [W] E
                       ^── extended bit (0 = compact)
                  ^─^──── wire type (1 bit reserved)
      ^─^─^─^──────────── field number 1..15
```

Equivalently `(fieldNum << 4) | (wireType << 1) | 0`.

### Extended tag (fields ≥ 16)

```
[wireType<<1 | 1] [varint fieldNum]
```

The marker has the extended bit set; the next varint carries the field
number.

### Wire types

| Code | Name      | Value encoding                                  |
|------|-----------|-------------------------------------------------|
| 0    | Varint    | LEB128 (uint, int, bool, enum)                  |
| 1    | Fixed64   | 8 bytes little-endian (float64, fixed64)        |
| 2    | Bytes     | Length-prefixed payload (string, bytes, message, packed array) |
| 3    | Fixed32   | 4 bytes little-endian (float32, fixed32)        |
| 4    | SVarint   | ZigZag-LEB128 (int8/16/32/64)                   |

### Determinism rules

- Maps: keys sorted lexicographically (strings/bytes), numerically
  (ints/floats), or by primitive bool order. NaN sorts last.
- Structs: fields emitted in tag-number order. Cached on first encode.
- Floats: NaN canonicalized to a fixed bit pattern; `-0.0` → `+0.0`.
- Optional fields: omitted when zero if `Options.OmitEmpty` is true OR
  the field carries an explicit `,omitempty` tag.
- NaN map keys are rejected at encode time: distinct NaN bit patterns
  collapse to the same wire bytes (information loss) and Go's
  `MapIndex(nan)` returns the zero value (NaN ≠ NaN under `==`), so
  encoded values would silently become zero.

### JSON float formatting

The deterministic JSON encoder formats floats using Go's `strconv.FormatFloat`
with the `'g'` verb:

- 9 significant digits for `float32`, 17 for `float64`.
- Decimal form when the decimal exponent satisfies `-4 <= exp < precision`;
  otherwise scientific.
- Scientific form: `mantissa "e" sign digits` with a lowercase `e`, the
  exponent always signed, and the exponent zero-padded to at least two
  digits (e.g. `1e-07`, `1.23e+10`).
- Trailing zeros in the fractional part of the mantissa are stripped.
- `NaN` and `±Inf` are rejected with an error (they have no JSON spelling).
- `-0.0` is normalized to `+0.0`.

The TypeScript and Rust ports must produce byte-identical output. Reference
values for cross-language conformance live in `pkg/cramberry/json_test.go`
(authoritative); the same expected strings are duplicated in
`typescript/src/json.test.ts` and `rust/src/json.rs` tests.

## Limits

| Limit           | Default (`DefaultLimits`) | Secure (`SecureLimits`) |
|-----------------|---------------------------|--------------------------|
| MaxMessageSize  | 64 MB                     | 1 MB                     |
| MaxStringLength | 10 MB                     | 1 MB                     |
| MaxBytesLength  | 100 MB                    | 10 MB                    |
| MaxArrayLength  | 1 000 000                 | 10 000                   |
| MaxMapSize      | 1 000 000                 | 10 000                   |
| MaxDepth        | 100                       | 32                       |

`Reader.BeginMessage`, `WriteString`/`ReadString`, `WriteBytes`/`ReadBytes`,
`WriteArrayHeader`/`ReadArrayHeader`, `WriteMapHeader`/`ReadMapHeader`, and
`enterNested` all enforce limits. The decoder additionally rejects any
length-prefix that exceeds the remaining wire bytes — defense against
amplification (a 4-byte varint claiming 1M map entries can otherwise force
megabytes of pre-allocation).

## Forward compatibility

Every field's body is structured so that a decoder which doesn't know the
field can call `SkipValue(wireType)` and consume exactly the right bytes:

- `WireVarint` / `WireSVarint` — read a varint and stop.
- `WireFixed32` / `WireFixed64` — read N bytes.
- `WireBytes` — read a varint length, skip that many bytes.

Composite kinds (struct, map, repeated of struct, repeated of scalar slice)
emit `WireBytes` and length-prefix their bodies at the field boundary.
Pointer-to-scalar fields (e.g. `*int32`) use the underlying scalar's wire
type directly — no extra wrap, since SkipValue knows the body shape.

Non-canonical varints are rejected on decode: a multi-byte varint whose
terminating byte is zero is necessarily over-long, and accepting it would
make hashes-over-bytes diverge across runtimes for the same logical value.

## Polymorphic types

The `Registry` maps a `TypeID` to a Go/TS/Rust type:

- `0–63` reserved for built-in primitives.
- `64–127` reserved for stdlib.
- `128+` for user types.

Generated `EncodeAny` / `DecodeAny` use the registry to dispatch on `TypeID`.

## Streaming

`StreamWriter` / `StreamReader` / `MessageIterator` emit length-delimited
messages on a `io.Writer` / `io.Reader`. Used by glueberry for both
encrypted and unencrypted streams.

## Pool & zero-copy

- `GetWriter()` / `PutWriter(w)` use a `sync.Pool` of writers, but only
  pool when `cap(buf) <= 64KB` to bound per-pool memory.
- `Reader.ZeroCopyString` / `ZeroCopyBytes` return slices into the input
  buffer with a generation counter; calling `Reader.Reset` increments the
  generation and zero-copy values panic on use.

## Codegen

Three generators live in `pkg/codegen/`:

- `go_generator.go` — generates `EncodeTo(*Writer)` / `DecodeFrom(*Reader)`,
  `Marshal`/`Unmarshal` wrappers, `Validate()`, optional `ToJSON()`/
  `FromJSON()` (gated by `-json`), polymorphic helpers.
- `typescript_generator.go` — generates plain TS interfaces plus
  `encode<Name>` / `decode<Name>` functions; `toJSON_<Name>` / `fromJSON_<Name>`
  when `-json` is enabled. Imports the runtime via `from '@cramberry/runtime'`.
- `rust_generator.go` — generates `pub struct` / `pub enum` definitions
  with `encode_<name>` / `decode_<name>` / `marshal_<name>` /
  `unmarshal_<name>` free functions; `to_json_<name>` / `from_json_<name>`
  when `-json` is enabled. Uses `cramberry::Result<T>` for binary helpers
  and `std::result::Result<...>` for JSON helpers (which return
  `(String, String)`).

Shared helpers (e.g. `ResolveNamedEnum`, `ToPascalCase`, `ToSnakeCase`)
live in `pkg/codegen/generator.go` so each generator's per-type dispatch
stays in sync.

Conformance is verified at multiple layers:

- `testdata/golden/*.bin` — Go-produced golden bytes. Both the reflection
  marshaller and the codegen-emitted `EncodeTo` must reproduce them.
- `make codegen-check` — generates Go, TS, and Rust output for every
  schema in `examples/` and `testdata/` and compiles each in its target
  language. Catches generator regressions that wouldn't show up in
  Go-side unit tests.
- `make codegen-parity-check` — generates Go, Rust, and TypeScript code
  from `scripts/parity_fixture.cram`, encodes the same logical fixture
  through every runtime, and asserts that **Go reflection == Go codegen
  == Rust codegen == TS codegen** for the resulting bytes.

## Schema language

The `.cram` syntax is implemented in `pkg/schema/`:

```cramberry
package example;
option go_package = "github.com/example/types";

enum Status { UNKNOWN = 0; ACTIVE = 1; }

message User {
    required uint64 id = 1;
    string name = 2;
    []string tags = 3;
    map[string]string metadata = 4;
}

interface Principal {
    128 = User;
    129 = Organization;
}
```

Hand-written, recursive-descent parser. Lexer accepts hex literals
(`0xFF`) and rejects malformed exponents (`1e`, `1e-`).

Validator catches:
- Duplicate field numbers, undefined types, broken enum values.
- Illegal map key types (bytes / floats / complex / **bool** are rejected).
- Stacked field modifiers (`required repeated`, `optional repeated`).
- Reserved type-ID range usage: user-declared `@N` IDs must be ≥ 128
  (1-63 builtin reserved, 64-127 stdlib reserved).
- Enums missing a 0-valued variant (cross-language default consistency).
- Imports that escape the importing file's directory and any search-path
  entry (containment check; `import "../../etc/passwd"` is rejected).

`compat.go` checks schema evolution (added/removed fields, type changes).
`schema.WriteToFile` is atomic via `internal/atomicfile` (temp + rename
+ dir fsync) — a crash mid-encode never leaves a half-written `.cram`.

## Cross-language conformance

Conformance is enforced by three independent layers:

**1. Golden-file conformance (`test/integration/`)**

- `gen/interop.go` — code-generated Go types from `testdata/schemas/interop.cram`,
  exercising scalar, repeated, nested, complex, edge, and all-field-numbers
  shapes. Regenerated by `make generate-fixtures`.
- `testdata/golden/*.bin` and `*.hex` — Go-produced expected wire bytes.
- `interop_test.go` — verifies both the reflection path and the
  codegen-emitted `EncodeTo` produce the golden bytes.
- `ts/`, `rust/` — port-specific runners (`make ts-integration-test`,
  `make rust-integration-test`) that decode the golden bytes via each
  language's runtime.

**2. Compile-check (`make codegen-check`)**

For every `.cram` schema in `examples/` and `testdata/`, generate Go, TS,
and Rust output and compile each against its runtime. Catches generator
output that wouldn't compile.

**3. Byte-parity (`make codegen-parity-check`)**

End-to-end probe: generate Go, Rust, and TypeScript code from
`scripts/parity_fixture.cram` (a schema covering all common drift
hotspots — scalars, repeated, nested, maps, optional, enum, recursive),
encode the same logical fixture through each, and assert every byte is
identical:

```
Go reflection == Go codegen == Rust codegen == TS codegen
```

Any byte-stream divergence — even one byte — fails the build.

`make integration-test` runs all three layers.

## Performance characteristics

- Single-byte tag for fields 1–15 → ~1 byte saved per field vs Protobuf.
- `sync.Pool` writers with 256-byte initial buffer.
- Field-info cache keyed by `reflect.Type` — first marshal of a type pays
  introspection cost; subsequent ones do not.
- Zero-copy reads avoid a `[]byte`→`string` copy for hot reads.
