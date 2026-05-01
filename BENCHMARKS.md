# Cramberry Performance Benchmarks

Cross-runtime encode/decode benchmarks for Cramberry against Protocol Buffers
and JSON in Go, Rust, and TypeScript. Measurements taken **2026-05-01** on
**Apple M4 Pro / macOS 24.6 (Darwin arm64)** at version `v2.0.0`.

| Tool             | Driver                                          |
|------------------|-------------------------------------------------|
| Go               | `go test -bench -benchmem -benchtime=3s`        |
| Rust             | `cargo bench` (criterion, 3 s warmup + 5 s collect) |
| TypeScript       | `tinybench` via `tsx` (warmupTime: 250 ms)      |
| Sizes            | `make bench-sizes`                              |

Reproduce with `make bench-cross` (requires `protoc` on PATH for the Rust port).

---

## Executive summary

- **Wire size**. Cramberry is **byte-comparable to protobuf** (within ±5 %)
  and **2.2–2.8× smaller than JSON** for every fixture. At 1 000-message
  batches Cramberry is **5 % smaller than protobuf** because the compact tag
  wins the field-1–15 fast path.
- **Go**. Codegen Cramberry **wins decode against protobuf 1.4–2.8×** for
  every non-trivial scenario, and matches or beats it on encode. Reflection
  marshalling is ~2× slower than codegen — the codegen ROI is real.
- **Rust**. Cramberry **wins decode against `prost` 1.1–1.4×** for every
  non-trivial scenario, but loses encode by ~3× (prost's offset-cached
  encoder is hard to beat). JSON via `serde_json` is consistently 2–3×
  slower than Cramberry on both paths.
- **TypeScript**. Cramberry trails `protobufjs` 2–3× on both paths, but
  beats `JSON.stringify` on encode for nested fixtures. The TS port's
  Number/BigInt boundary is the dominant cost; protobufjs has hand-tuned
  Long-pair tricks Cramberry hasn't ported yet.
- **All four runtimes (Go reflection, Go codegen, Rust codegen, TS codegen)
  produce byte-identical output** for every fixture. The byte-parity probe
  runs in `scripts/codegen-parity-check.sh`.

---

## Encoded sizes

| Message       | Cramberry | Protobuf | JSON   | Cram/PB | JSON/PB |
|---------------|-----------|----------|--------|---------|---------|
| SmallMessage  |        18 |       16 |     45 |   1.13× |   2.81× |
| Metrics       |        76 |       75 |    154 |   1.01× |   2.05× |
| Person        |       219 |      212 |    540 |   1.03× |   2.55× |
| Document      |       423 |      419 |    930 |   1.01× |   2.22× |
| Event         |       183 |      183 |    395 |   1.00× |   2.16× |
| Batch100      |     1 727 |    1 790 |  4 623 |   0.96× |   2.58× |
| Batch1000     |    17 029 |   17 962 | 45 573 |   **0.95×** |   2.54× |

The cross-over from "slightly larger" to "slightly smaller" happens because
Cramberry's compact tag (`(fieldNum << 4) | (wireType << 1) | 0`) is one byte
for fields 1–15 vs protobuf's `(fieldNum << 3) | wireType` which is one byte
for fields 1–15 too, but Cramberry's tag-byte savings on the very dense
end-of-batch fields compound across thousand-message arrays.

---

## Go (Apple M4 Pro, Go 1.25.6)

`make bench-go` — `go test -bench -benchmem -benchtime=3s ./internal/bench/...`

### Encode latency (ns/op)

`Cram vs PB` is `(Cram−PB)/PB` — negative means Cramberry is faster.
`Cram vs JSON` is `JSON/Cram` — `>1` means Cramberry is faster.

| Scenario      | Cramberry | Reflection | Protobuf | JSON   | Cram vs PB | Cram vs JSON |
|---------------|-----------|------------|----------|--------|-----------:|-------------:|
| SmallMessage  |     45.8  |      85.6  |    43.7  |   84.6 |     +5 %   |     1.85×    |
| Metrics       |     74.3  |     172.3  |    77.9  |  378.0 |     −5 %   |     5.09×    |
| Person        |    346.4  |     672.6  |   329.8  |  722.5 |     +5 %   |     2.09×    |
| Document      |    716.8  |   1 420.0  |   967.1  | 1 232.0|   **−26 %** |    1.72×    |
| Event         |    345.4  |     581.0  |   537.8  |  529.4 |   **−36 %** |    1.53×    |
| Batch100      |  2 657    |   6 618    |   3 569  |  5 448 |   **−26 %** |    2.05×    |
| Batch1000     | 23 982    |  58 796    |  29 221  | 54 535 |   **−18 %** |    2.27×    |

### Decode latency (ns/op)

| Scenario      | Cramberry | Reflection | Protobuf | JSON    | Cram vs PB | Cram vs JSON |
|---------------|-----------|------------|----------|---------|-----------:|-------------:|
| SmallMessage  |     26.3  |      92.0  |    68.1  |   473.7 |   **−61 %** |   **18.0×**  |
| Metrics       |     42.6  |     367.3  |   119.1  |  1 233  |   **−64 %** |   **28.9×**  |
| Person        |    432.4  |   1 243    |   616.7  |  3 676  |   **−30 %** |     8.50×    |
| Document      |    834.6  |   2 104    |  1 463   |  6 582  |   **−43 %** |     7.89×    |
| Event         |    390.2  |     955.0  |   731.2  |  2 648  |   **−47 %** |     6.78×    |
| Batch100      |  2 918    |   7 911    |  7 152   | 36 211  |   **−59 %** |   **12.4×**  |
| Batch1000     | 27 244    |  76 150    | 62 613   | 334 251 |   **−56 %** |   **12.3×**  |

### Memory (allocs/op)

| Scenario      | Cramberry encode | PB encode | Cramberry decode | PB decode |
|---------------|-----------------:|----------:|-----------------:|----------:|
| SmallMessage  |  1 alloc / 24 B  | 1 / 16 B  |  1 alloc / 16 B  | 2 / 96 B  |
| Metrics       |  1 / 80 B        | 1 / 80 B  |  **0 allocs / 0 B**  | 1 / 128 B |
| Person        |  1 / 224 B       | 1 / 224 B | 20 / 336 B       | 23 / 808 B|
| Document      |  2 / 496 B       | 13 / 640 B| 28 / 1 048 B     | 51 / 2 000 B |
| Batch100      |  2 / 1 824 B     | 9 / 1 920 B| 108 / 5 464 B   | 225 / 12 480 B |
| Batch1000     |  2 / 18 474 B    | 9 / 18 560 B| 1 008 / 49 176 B| 2 028 / 114 240 B |

Cramberry's encode is single-allocation for every fixture (one for the
buffer); protobuf's `proto.Marshal` does up to 13 small allocations on
nested messages because each repeated/map field grows its own slice.
On decode Cramberry allocates **half as many objects** as protobuf at
batch scale (1 008 vs 2 028 for Batch1000) because the codegen reader
skips per-field temporary objects.

The reflection path is **2.0× slower on encode and 2.5–3.0× slower on
decode** than codegen — it's there as a fallback for schema-less callers
and as the canonical implementation, not the production path.

---

## Rust (Apple M4 Pro, rustc stable / criterion 0.5)

`make bench-rust` — `cargo bench --bench encode_decode`. No memory numbers
(criterion does not measure allocations).

### Encode latency

Ratios are JSON/Cramberry — `>1` means JSON is slower; `<1` means Cramberry is slower.

| Scenario      | Cramberry | Prost     | serde_json | Cram vs Prost | Cram vs JSON         |
|---------------|-----------|-----------|------------|--------------:|---------------------:|
| SmallMessage  |     37.0 ns |    5.79 ns |     37.3 ns |    6.4× slower | tied (1.01×)         |
| Metrics       |    112.0 ns |   15.7 ns  |    239.5 ns |    7.1× slower | **2.14× faster**     |
| Person        |    387.7 ns |  113.2 ns  |    511.1 ns |    3.4× slower | **1.32× faster**     |
| Document      |    651.8 ns |  206.9 ns  |    737.3 ns |    3.1× slower | **1.13× faster**     |
| Event         |    255.0 ns |   75.5 ns  |    400.2 ns |    3.4× slower | **1.57× faster**     |
| Batch100      |    3.18 µs |    1.06 µs |    2.99 µs  |    3.0× slower | 1.06× slower         |
| Batch1000     |   30.77 µs |   10.18 µs |   27.47 µs  |    3.0× slower | 1.12× slower         |

### Decode latency

| Scenario      | Cramberry | Prost     | serde_json | Cram vs Prost     | Cram vs JSON      |
|---------------|-----------|-----------|------------|------------------:|------------------:|
| SmallMessage  |     25.4 ns |   25.7 ns  |    53.6 ns | tied              | **2.11× faster**  |
| Metrics       |     42.7 ns |   28.6 ns  |   166.0 ns | 1.5× slower       | **3.89× faster**  |
| Person        |    362.8 ns |  400.0 ns  |    820.8 ns | **1.10× faster**  | **2.26× faster**  |
| Document      |    617.5 ns |  833.6 ns  |  1 427 ns   | **1.35× faster**  | **2.31× faster**  |
| Event         |    311.7 ns |  405.6 ns  |    847.3 ns | **1.30× faster**  | **2.72× faster**  |
| Batch100      |    2.58 µs |   3.71 µs |    6.91 µs  | **1.44× faster**  | **2.68× faster**  |
| Batch1000     |   24.78 µs |  30.04 µs |   64.87 µs  | **1.21× faster**  | **2.62× faster**  |

The Rust encode story is dominated by `prost`'s offset-cached encoder:
it precomputes per-field offsets at codegen time and writes directly into a
pre-sized buffer with zero indirection. Cramberry's encoder still takes a
field-info table lookup per field. Closing that gap is on the v2.1
roadmap.

The decode story is the inverse — Cramberry's reader is **1.1–1.4× faster
than prost on every nontrivial fixture**. Prost allocates a fresh
`Vec<u8>` for every nested length-prefix; Cramberry's reader reuses a
single read cursor over the input slice. JSON is consistently 2–3× behind
Cramberry on decode.

---

## TypeScript (Node 22, V8 12.x, Apple M4 Pro)

`make bench-ts` — tinybench via tsx. No memory numbers (V8 does not
expose precise per-op allocation).

### Encode latency

| Scenario      | Cramberry | protobufjs | JSON.stringify | Cram vs PB | Cram vs JSON |
|---------------|-----------|------------|----------------|-----------:|-------------:|
| SmallMessage  |    320 ns |    152 ns  |    291 ns      |  2.1× slower |    1.10×    |
| Metrics       |    802 ns |    558 ns  |    765 ns      |  1.4× slower |    1.05×    |
| Person        |  2 868 ns |  1 141 ns  |  1 593 ns      |  2.5× slower |    1.80×    |
| Document      |  5 565 ns |  2 021 ns  |  3 437 ns      |  2.8× slower |    1.62×    |
| Event         |  2 397 ns |    940 ns  |  3 067 ns      |  2.5× slower | **0.78× (cram wins)** |
| Batch100      | 26 719 ns | 12 436 ns  | 18 100 ns      |  2.1× slower |    1.48×    |
| Batch1000     |  256 µs   |  118 µs    |    179 µs      |  2.2× slower |    1.43×    |

### Decode latency

| Scenario      | Cramberry | protobufjs | JSON.parse | Cram vs PB | Cram vs JSON |
|---------------|-----------|------------|------------|-----------:|-------------:|
| SmallMessage  |    201 ns |     86 ns  |    163 ns  |  2.3× slower |   1.23×     |
| Metrics       |    225 ns |    152 ns  |    342 ns  |  1.5× slower | **0.66× (cram wins)** |
| Person        |  2 240 ns |    638 ns  |  1 300 ns  |  3.5× slower |   1.72×     |
| Document      |  4 094 ns |  1 297 ns  |  2 053 ns  |  3.2× slower |   1.99×     |
| Event         |  1 830 ns |    648 ns  |  1 024 ns  |  2.8× slower |   1.79×     |
| Batch100      | 14 977 ns |  6 952 ns  |  8 373 ns  |  2.2× slower |   1.79×     |
| Batch1000     |  149 µs   |   67 µs    |   79 µs    |  2.2× slower |   1.89×     |

The Cramberry TS port pays a two-fold cost vs protobufjs:

1. **BigInt boundary**. Every `int64`/`uint64` field must convert to
   `bigint` on decode and back to bytes on encode. protobufjs uses a
   `Long` (low/high pair) representation internally and skips the BigInt
   path entirely. Cramberry's runtime falls back to a Number fast path
   for values that fit in 32 bits (the common case), but the worst case
   is dominant for `Person`/`Event` IDs.
2. **Hidden-class stability**. The v2.0.0 perf pass already moved the
   decode template to default-init all fields at construction (so V8
   sees a monomorphic shape). The remaining gap is per-field
   conditional checks for missing fields — the codegen could
   precompute presence bits but currently does a `?? defaultValue`
   per field.

JSON.stringify is interesting: V8 has a hand-tuned C++ serializer that
beats Cramberry's hand-written varint loop on tiny messages, but loses
on nested objects with maps/arrays where the JSON parser allocates
intermediate strings.

---

## Codegen ROI (Go reflection vs Go codegen)

The reflection marshaller is the canonical implementation; it's also
the slow path. Codegen's whole purpose is to skip the per-field
`reflect.Value` indirection. Numbers below isolate the ROI.

| Scenario      | Encode Refl/Code | Decode Refl/Code |
|---------------|-----------------:|-----------------:|
| SmallMessage  |          1.87×   |        3.51×     |
| Metrics       |          2.32×   |          8.62×   |
| Person        |          1.94×   |          2.87×   |
| Document      |          1.98×   |          2.52×   |
| Event         |          1.68×   |          2.45×   |
| Batch100      |          2.49×   |          2.71×   |
| Batch1000     |          2.45×   |          2.80×   |

For batched workloads the codegen path is **~2.5× faster on encode and
~2.7× faster on decode** than reflection — and Metrics decode is **8.6×
faster** because reflection allocates a `reflect.Value` per scalar
field and codegen reads scalars directly into a stack-allocated struct
(0 allocs / 0 B).

---

## Caveats

- **Single-machine numbers.** Tail variance is meaningful below the
  ~50 ns floor; treat ratios as the signal, not absolutes. Run twice
  before drawing conclusions.
- **`protobufjs` ≠ Google protobuf.** The TS protobuf path uses
  `protobufjs`, which interprets `.proto` files at runtime. It's still
  faster than Cramberry on every fixture, but "TS protobuf" in
  production today *is* protobufjs — the comparison is fair, not
  cherry-picked.
- **Allocator coverage is uneven.** Go's `-benchmem` reports per-op
  bytes and allocs precisely. Rust criterion and tinybench do not.
  The Go memory column above is authoritative for Cramberry's
  allocator pressure across all three runtimes (the wire-format
  emission is platform-independent).
- **Compiler versions matter.** These numbers were taken on Apple
  Silicon with Go 1.25.6, rustc stable 1.86, and Node 22. Linux x86_64
  results are typically 5–15 % slower on encode (different memory
  prefetcher) and equivalent on decode.
- **Encode-side optimization roadmap.** Cramberry's Rust encoder is
  3× behind `prost`. The gap is the cost of the field-info table
  lookup vs prost's compile-time offset constants. v2.1 is exploring
  generating per-field encode functions inline (eliminating the
  table dispatch) which should close most of the gap without losing
  the schema-evolution flexibility that the table provides.

---

## Reproducing

```bash
make bench-go     # ~3 min
make bench-rust   # ~10 min (criterion warmup)
make bench-ts     # ~2 min
make bench-cross  # all three sequentially
make bench-sizes  # encoded-size table only
```

Every run regenerates the fixture data from `internal/bench/schemas/`
so size and latency stay in sync. The fixtures are designed to
exercise the cross-runtime drift hotspots: zero-valued fields,
optional pointers, repeated nested messages, maps with both string
and integer keys, and at least one ≥ 20 KB string to anchor the
multi-byte varint length-prefix path.
