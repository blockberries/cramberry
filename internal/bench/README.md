# Cramberry Performance Benchmarks

Cross-language benchmark suite for Cramberry serialization. Compares the
Cramberry codegen and reflection paths against Protocol Buffers and JSON
in **Go, Rust, and TypeScript** so the runtime cost of an end-to-end
encode/decode is directly comparable across every supported port.

## Layout

```
internal/bench/
├── schemas/
│   ├── messages.cram         # Cramberry schema (one file, three runtimes)
│   └── messages.proto        # Equivalent protobuf definitions
├── gen/                      # Go: pre-generated cramberry + protobuf code
│   ├── cramberry/messages.go
│   └── protobuf/messages.pb.go
├── benchmark_test.go         # Go benchmarks + size comparison test
│
├── rust/                     # Rust criterion benches
│   ├── Cargo.toml
│   ├── build.rs              # Invokes prost-build at compile time
│   ├── src/
│   │   ├── lib.rs
│   │   ├── messages.rs       # cramberry-generated rust codecs
│   │   └── fixtures.rs
│   └── benches/encode_decode.rs
│
└── ts/                       # TypeScript tinybench harness
    ├── package.json
    ├── tsconfig.json
    └── src/
        ├── messages.ts       # cramberry-generated TS codecs
        ├── fixtures.ts
        └── bench.ts
```

## Running

| Command            | What it runs                                             |
|--------------------|----------------------------------------------------------|
| `make bench`       | Go benchmarks (alias for `bench-go`)                     |
| `make bench-go`    | Go: cramberry codegen + reflection vs protobuf vs JSON   |
| `make bench-rust`  | Rust: cramberry codegen vs prost vs serde_json           |
| `make bench-ts`    | TypeScript: cramberry codegen vs protobufjs vs JSON      |
| `make bench-cross` | All three languages, sequentially                        |
| `make bench-sizes` | Just the encoded-size comparison table (Go runs it)      |

`make bench-rust` requires `protoc` on `PATH` (used by `prost-build` to
compile `messages.proto` at build time):

```bash
brew install protobuf       # macOS
apt install protobuf-compiler   # Debian/Ubuntu
```

`make bench-ts` shells out to `npm install` in `internal/bench/ts/` on
first run; subsequent invocations reuse the local `node_modules`.

## Filtering benchmarks

```bash
# Go: standard go test -bench filter
go test ./internal/bench/... -bench=Person -benchmem

# Rust: criterion accepts a regex filter after `--`
cd internal/bench/rust && cargo bench --bench encode_decode -- Document/cramberry

# TypeScript: a single positional substring
cd internal/bench/ts && npx tsx src/bench.ts SmallMessage
```

## Test scenarios

Every scenario runs encode + decode for every codec:

| Scenario      | Description                               | Cramberry size |
|---------------|-------------------------------------------|----------------|
| SmallMessage  | 3 fields — minimal baseline               | 18 B           |
| Metrics       | 10 scalar fields (ints + floats)          | 76 B           |
| Person        | Deeply nested with optional fields        | 212 B          |
| Document      | Arrays, maps, nested messages             | 412 B          |
| Event         | Maps with string keys + bytes payload     | 180 B          |
| Batch100      | 100 repeated messages                     | 1 723 B        |
| Batch1000     | 1 000 repeated messages                   | 17 024 B       |

The Go side additionally exposes `*_Reflection_*` variants that exercise
`cramberry.Marshal` / `Unmarshal` (the reflection path) instead of the
generated `MarshalCramberry` methods. Reflection is observably 3× slower
on encode and 3–12× slower on decode — useful for measuring codegen ROI.

## Encoded sizes

| Message      | Cramberry | Protobuf | JSON   | Cram/PB | JSON/PB |
|--------------|-----------|----------|--------|---------|---------|
| SmallMessage |        18 |       16 |     45 |   1.12x |   2.81x |
| Metrics      |        76 |       75 |    154 |   1.01x |   2.05x |
| Person       |       212 |      212 |    540 |   1.00x |   2.55x |
| Document     |       412 |      419 |    930 |   0.98x |   2.22x |
| Event        |       180 |      183 |    395 |   0.98x |   2.16x |
| Batch100     |     1 723 |    1 790 |  4 623 |   0.96x |   2.58x |
| Batch1000    |    17 024 |   17 962 | 45 573 |   0.95x |   2.54x |

## Reading the latency results

Each language emits results in its native bench tool's format:

- **Go** — `go test -bench` output. `ns/op`, `B/op`, `allocs/op`. Memory
  numbers via `-benchmem` are authoritative for the Cramberry runtime
  because Go's allocator is the only one that reports them precisely.
- **Rust** — criterion's three-line summary per bench, plus an HTML
  report under `internal/bench/rust/target/criterion/`. No memory
  numbers — `cargo bench` does not measure allocations.
- **TypeScript** — a console.table with `ns/op`, `ops/sec`, and
  `samples`. V8 doesn't expose precise per-op allocation either; treat
  these as latency only.

Treat single-run numbers as indicative, not authoritative. Tail variance
between runs is significant on a noisy laptop. Run twice and look for
agreement before drawing conclusions.

## Regenerating fixtures

When you modify `messages.cram` or `messages.proto`:

```bash
# Go (cramberry)
make build
./bin/cramberry generate -lang go -package benchmark \
    -out internal/bench/gen/cramberry \
    internal/bench/schemas/messages.cram

# Go (protobuf)
protoc --go_out=. --go_opt=paths=source_relative \
    internal/bench/schemas/messages.proto
mv internal/bench/schemas/messages.pb.go internal/bench/gen/protobuf/

# Rust (cramberry — prost regenerates automatically via build.rs)
./bin/cramberry generate -lang rust \
    -out internal/bench/rust/src \
    internal/bench/schemas/messages.cram

# TypeScript (cramberry)
./bin/cramberry generate -lang ts \
    -out internal/bench/ts/src \
    internal/bench/schemas/messages.cram
```

Update the fixture builders (`fixtures.rs`, `fixtures.ts`,
`benchmark_test.go::makeCramberry*`) so all three languages keep
constructing semantically identical inputs — otherwise the size and
latency numbers stop being directly comparable.

## Caveats

- **Single-machine noise.** Numbers below the ~50ns floor track
  uncertainty more than algorithmic difference. Compare ratios, not
  absolutes.
- **JIT warmup.** TypeScript needs warmup; tinybench does this
  automatically (`warmupTime: 250ms`). Rust criterion warms up by
  default. Go's `testing.B` warms up implicitly via `b.N`.
- **Allocator differences.** Go's `-benchmem` reports per-op B and
  allocs; Rust criterion and tinybench do not. To compare allocation
  pressure across languages, use a profiler (instruments / dhat /
  Chrome DevTools heap profile) — not these benches.
- **protobuf in TS goes through protobufjs.** `protobufjs` interprets
  `.proto` files at runtime; it caches the parsed root, but the per-op
  cost is materially higher than `prost` (Rust) or `google.golang.org/
  protobuf` (Go). The TS protobuf numbers are a fair representation of
  what a TS application would actually pay — not an upper bound on
  Google-protobuf throughput.

## Cross-language byte parity

Encoded bytes are byte-identical across Go reflection, Go codegen, Rust
codegen, and TS codegen for every common type. The byte-parity probe is
under `scripts/codegen-parity-check.sh` (run with
`make codegen-parity-check`) and is independent of these benchmarks.
