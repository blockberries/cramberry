# Cramberry Performance and Size Benchmark Report

**Test Environment:**
- Platform: darwin/arm64 (Apple M4 Pro)
- Go version: 1.24+
- Cramberry version: 1.2.0
- Comparison formats: Cramberry, Protocol Buffers, JSON

---

## 1. Encoding Performance

| Message Type | Cramberry | Protobuf | JSON | Cram vs PB | Cram vs JSON |
|--------------|-----------|----------|------|------------|--------------|
| SmallMessage | 49.8 ns | 45.9 ns | 86.2 ns | 1.09x slower | 1.73x faster |
| Metrics | 80.2 ns | 79.5 ns | 375.6 ns | 1.01x slower | 4.68x faster |
| Person | 304.2 ns | 368.3 ns | 692.1 ns | 1.21x faster | 2.28x faster |
| Document | 607.5 ns | 1037 ns | 1244 ns | 1.71x faster | 2.05x faster |
| Event | 279.2 ns | 531.2 ns | 547.1 ns | 1.90x faster | 1.96x faster |
| Batch100 | 2.69 us | 3.42 us | 5.61 us | 1.27x faster | 2.09x faster |
| Batch1000 | 25.8 us | 30.6 us | 55.3 us | 1.19x faster | 2.14x faster |

**Observations:**
- Cramberry encoding is comparable to Protobuf for simple messages
- For complex messages (Document, Event), Cramberry encoding is 1.7-1.9x faster than Protobuf
- Cramberry encoding is consistently 1.7-4.7x faster than JSON

---

## 2. Decoding Performance

| Message Type | Cramberry | Protobuf | JSON | Cram vs PB | Cram vs JSON |
|--------------|-----------|----------|------|------------|--------------|
| SmallMessage | 27.8 ns | 67.5 ns | 402.6 ns | **2.43x faster** | 14.5x faster |
| Metrics | 41.4 ns | 119.3 ns | 1170 ns | **2.88x faster** | 28.3x faster |
| Person | 388.0 ns | 592.4 ns | 3641 ns | **1.53x faster** | 9.38x faster |
| Document | 741.7 ns | 1394 ns | 6462 ns | **1.88x faster** | 8.71x faster |
| Event | 382.6 ns | 683.3 ns | 2525 ns | **1.79x faster** | 6.60x faster |
| Batch100 | 2.92 us | 6.76 us | 36.3 us | **2.32x faster** | 12.4x faster |
| Batch1000 | 27.0 us | 61.4 us | 339.5 us | **2.27x faster** | 12.6x faster |

**Observations:**
- Cramberry decoding outperforms Protobuf across all message types (1.53-2.88x faster)
- The largest decode performance gains are for simple messages (SmallMessage, Metrics)
- Cramberry decoding is 6.6-28.3x faster than JSON

---

## 3. Memory Allocation (Encoding)

| Message Type | Cramberry | Protobuf | JSON |
|--------------|-----------|----------|------|
| SmallMessage | 24 B / 1 alloc | 16 B / 1 alloc | 48 B / 1 alloc |
| Metrics | 80 B / 1 alloc | 80 B / 1 alloc | 160 B / 1 alloc |
| Person | 224 B / 1 alloc | 224 B / 1 alloc | 576 B / 1 alloc |
| Document | 416 B / 1 alloc | 640 B / 13 allocs | 1248 B / 8 allocs |
| Event | 192 B / 1 alloc | 320 B / 9 allocs | 560 B / 6 allocs |
| Batch1000 | 18.4 KB / 1 alloc | 18.6 KB / 9 allocs | 49.3 KB / 6 allocs |

**Observations:**
- Cramberry consistently achieves single-allocation encoding
- Protobuf requires multiple allocations for complex types (Document: 13, Event: 9)
- Cramberry uses 35-65% less memory than Protobuf for Document and Event encoding

---

## 4. Memory Allocation (Decoding)

| Message Type | Cramberry | Protobuf | JSON |
|--------------|-----------|----------|------|
| SmallMessage | 16 B / 1 alloc | 96 B / 2 allocs | 264 B / 6 allocs |
| Metrics | **0 B / 0 allocs** | 128 B / 1 alloc | 296 B / 5 allocs |
| Person | 336 B / 20 allocs | 808 B / 23 allocs | 848 B / 30 allocs |
| Document | 1032 B / 28 allocs | 2000 B / 51 allocs | 1880 B / 47 allocs |
| Batch1000 | 49.2 KB / 1008 allocs | 114.2 KB / 2028 allocs | 87.1 KB / 1029 allocs |

**Observations:**
- Metrics decoding achieves **zero allocations** in Cramberry
- Cramberry decode allocations are typically 45-58% fewer than Protobuf
- Memory usage during decode is 42-57% lower than Protobuf for most message types

---

## 5. Encoded Size Comparison

| Message Type | Cramberry | Protobuf | JSON | Cram/PB Ratio |
|--------------|-----------|----------|------|---------------|
| SmallMessage | 18 bytes | 16 bytes | 45 bytes | 1.12x larger |
| Metrics | 76 bytes | 75 bytes | 154 bytes | 1.01x larger |
| Person | 212 bytes | 212 bytes | 540 bytes | 1.00x (equal) |
| Document | 412 bytes | 419 bytes | 930 bytes | 0.98x smaller |
| Event | 180 bytes | 183 bytes | 395 bytes | 0.98x smaller |
| Batch100 | 1723 bytes | 1790 bytes | 4623 bytes | 0.96x smaller |
| Batch1000 | 17024 bytes | 17962 bytes | 45573 bytes | 0.95x smaller |

**Observations:**
- SmallMessage has 12% size overhead compared to Protobuf (2 bytes)
- For larger messages, Cramberry produces 2-5% smaller output than Protobuf
- Cramberry output is 2.1-2.8x smaller than JSON across all message types

---

## 6. Summary

### Performance relative to Protocol Buffers

| Metric | Range | Notes |
|--------|-------|-------|
| Encode speed | 0.92x - 1.90x | Faster for complex messages |
| Decode speed | **1.53x - 2.88x** | Faster across all types |
| Encode memory | 0.35x - 1.50x | Single allocation pattern |
| Decode memory | 0.42x - 0.58x | Fewer allocations |
| Encoded size | 0.95x - 1.12x | Within 12% of Protobuf |

### Strengths

- **Decode performance is consistently superior** (1.5-2.9x faster than Protobuf)
- **Zero-allocation decoding** for simple value types (Metrics)
- **Single-allocation encoding** reduces GC pressure
- Competitive encoded sizes, smaller for complex messages
- Deterministic encoding for consensus/cryptographic applications

### Trade-offs

- SmallMessage encoding is ~9% slower than Protobuf
- SmallMessage encoded size is 12% larger (18 vs 16 bytes)
- Decode still requires allocations for messages with strings/slices

### Use Case Fit

- **Read-heavy workloads** benefit most from Cramberry's decode performance
- **Batch processing** shows strong scaling characteristics (2.3x faster decode)
- **Applications sensitive to GC pauses** benefit from reduced allocations
- **Consensus systems** benefit from deterministic encoding

---

## 7. Security Features (v1.2.0)

Cramberry v1.2.0 includes important security hardening:

- **Integer overflow protection** in packed array writers/readers
- **Zero-copy memory safety** with generation-based validation
- **Cross-language varint consistency** (10-byte maximum)
- **NaN canonicalization** for deterministic float encoding
- **Thread-safe registries** in all language runtimes

See [docs/SECURITY.md](docs/SECURITY.md) for security best practices.

---

## Running Benchmarks

```bash
# Performance benchmarks
go test -bench=. -benchmem ./benchmark/...

# Size comparison
go test ./benchmark/... -run TestEncodedSizes -v

# Full benchmark suite with multiple iterations
go test -bench=. -benchmem -count=5 ./benchmark/... | tee results.txt
```
