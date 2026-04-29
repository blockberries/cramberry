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
- Optional fields: omitted when zero **only if** `Options.OmitEmpty=true`.
  (Per-field `omitempty` tag is currently ignored — see PLAN T1-10.)

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
`enterNested` all enforce limits. **Note**: `encodePackedSlice` writes
varint length without going through `WriteArrayHeader`; the encode path can
exceed `MaxArrayLength` (decode side enforces).

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
  `Validate()`, `ToJSON()`/`FromJSON()`, polymorphic helpers.
- `typescript_generator.go` — generates classes with `encode`/`decode`/
  `toJSON`/`fromJSON`. One known incomplete spot at line 484 around
  polymorphic JSON dispatch.
- `rust_generator.go` — generates structs with derived `Cramberry`-style
  trait impls.

Conformance is verified by golden files in `testdata/golden/`.

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
    User = 128;
    Organization = 129;
}
```

Hand-written, recursive-descent parser. Validator catches duplicate field
numbers, undefined types, broken enum values, illegal map key types.
`compat.go` checks schema evolution (added/removed fields, type changes).

## Cross-language conformance

`test/integration/` contains:

- `gen/interop.go` — hand-written test programs that exercise scalar,
  repeated, nested, complex, edge, and all-field-numbers shapes.
- `testdata/golden/*.bin` and `*.hex` — expected wire-format output.
- `interop_test.go` — Go side: produces and verifies golden files.
- `ts/`, `rust/` — separate test runners (`make ts-test`, `make rust-test`).

Conformance fails if any port produces non-identical bytes.

## Performance characteristics

- Single-byte tag for fields 1–15 → ~1 byte saved per field vs Protobuf.
- `sync.Pool` writers with 256-byte initial buffer.
- Field-info cache keyed by `reflect.Type` — first marshal of a type pays
  introspection cost; subsequent ones do not.
- Zero-copy reads avoid a `[]byte`→`string` copy for hot reads.
