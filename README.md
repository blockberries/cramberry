# Cramberry

Deterministic binary serialization for Go, TypeScript, and Rust, with a
`.cram` schema language and code generators.

Designed for consensus-critical encoding: every implementation produces
byte-identical output for the same input. Used as the wire format and SignDoc
codec across the [Stealth blockchain stack](../README.md).

## Features

- **Deterministic** — sorted map keys, fixed field ordering, NaN
  canonicalization, `-0.0` → `+0.0` normalization, canonical varints
  (non-canonical encodings rejected on decode).
- **Compact** — comparable size to Protobuf, 2–3× smaller than JSON.
- **Schema-first** — `.cram` files generate type-safe Go / TS / Rust code.
- **Reflection fallback** — `Marshal`/`Unmarshal` on arbitrary structs via
  `cramberry:` struct tags.
- **Zero-copy reads** in Go via generation-tracked `unsafe.String`.
- **Cross-language byte parity** — `make codegen-parity-check` proves
  Go reflection, Go codegen, Rust codegen, and TS codegen all produce
  byte-identical output for the same logical input across every common
  type (scalars, repeated, nested, maps, optional, enums, recursive).

## Install

```bash
cd cramberry && make install   # injects version / commit / build-date ldflags
# or:
cd cramberry && make build     # produces ./bin/cramberry
```

`go install ./cmd/cramberry` also works but produces a binary that
reports `cramberry version dev (unknown, unknown)` — `make` is what
plumbs the version metadata through `-ldflags`.

## Usage

```go
import "github.com/blockberries/cramberry/pkg/cramberry"

type User struct {
    ID   uint64 `cramberry:"1"`
    Name string `cramberry:"2"`
    Tags []string `cramberry:"3"`
}

bytes, err := cramberry.Marshal(&User{ID: 1, Name: "alice", Tags: []string{"x"}})

var got User
err = cramberry.Unmarshal(bytes, &got)
```

For schema-driven types, write `user.cram`:

```cramberry
package example;

message User {
    uint64 id = 1;
    string name = 2;
    []string tags = 3;
}
```

Then generate Go/TS/Rust:

```bash
cramberry generate -lang go   -out ./types  ./schema/*.cram
cramberry generate -lang ts   -out ./ts     ./schema/*.cram
cramberry generate -lang rust -out ./rust   ./schema/*.cram
```

Short aliases (`-lang ts` / `-lang rs` / `-lang golang` / `-lang js`)
are accepted alongside the canonical names.

The generated TypeScript imports `from '@cramberry/runtime'` —
`npm install @cramberry/runtime` (or `npm install file:.../typescript`
for local development) provides the runtime.

## Wire format

A message is `[length:varint][field…][0x00 end marker]`.

Fields are tagged `[fieldNum:1-or-2 bytes][value]`:

- Fields 1–15 use a single-byte tag: `(fieldNum << 4) | (wireType << 1) | 0`.
- Fields 16+ use an extended marker: `(wireType << 1) | 1`, then varint fieldNum.

Wire types:

| Code | Name      | Encoding                                                 |
|------|-----------|----------------------------------------------------------|
| 0    | Varint    | LEB128 (integers, bools, enums)                          |
| 1    | Fixed64   | Little-endian 8 bytes (float64, fixed64)                 |
| 2    | Bytes     | Length-prefixed (string, bytes, message, packed array)   |
| 3    | Fixed32   | Little-endian 4 bytes (float32, fixed32)                 |
| 4    | SVarint   | ZigZag-encoded LEB128 (signed ints)                      |

## Layout

```
cramberry/
├── cmd/cramberry/         CLI: generate, validate, format, schema, version
├── pkg/cramberry/         Runtime: Writer/Reader, Marshal/Unmarshal, registry, JSON, stream
├── pkg/schema/            .cram lexer, parser, validator, AST, formatter
├── pkg/codegen/           Generators for Go, TS, Rust
├── pkg/extract/           Reverse: Go AST → .cram
├── internal/wire/         LEB128, ZigZag, fixed-width
├── internal/atomicfile/   Crash-safe file writes (temp + rename + dir fsync)
├── typescript/src/        TS port (uses identical wire format)
├── rust/src/              Rust port (uses identical wire format)
├── scripts/               codegen-check + codegen-parity-check shell harnesses
└── test/integration/      Cross-language conformance: TS + Rust runners,
                           golden files, generated-code byte-parity probe
```

## Development

See [`CLAUDE.md`](./CLAUDE.md) for development guidelines.
[`ARCHITECTURE.md`](./ARCHITECTURE.md) for design details.
[`CHANGELOG.md`](./CHANGELOG.md) for release history.

## License

Apache-2.0.
