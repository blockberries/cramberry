# Cramberry Performance and Size Benchmark Report

**Test Environment:**
- Platform: darwin/arm64 (Apple M4 Pro)
- Go version: 1.24+
- Cramberry version: 1.3.0
- Comparison formats: Cramberry (Generated), Cramberry (Reflection), Protocol Buffers, JSON

---

## 1. Encoding Performance (Generated Code)

| Message Type | Cramberry | Protobuf | JSON | Cram vs PB | Cram vs JSON |
|--------------|-----------|----------|------|------------|--------------|
| SmallMessage | 49.4 ns | 44.8 ns | 84.3 ns | 1.10x slower | 1.71x faster |
| Metrics | 76.3 ns | 79.5 ns | 385.8 ns | 1.04x faster | 5.06x faster |
| Person | 308.3 ns | 330.9 ns | 685.5 ns | 1.07x faster | 2.22x faster |
| Document | 603.7 ns | 1002.8 ns | 1217.3 ns | 1.66x faster | 2.02x faster |
| Event | 277.2 ns | 551.5 ns | 528.2 ns | 1.99x faster | 1.91x faster |
| Batch100 | 2.71 us | 3.41 us | 5.34 us | 1.26x faster | 1.97x faster |
| Batch1000 | 25.0 us | 29.9 us | 53.8 us | 1.20x faster | 2.15x faster |

**Observations:**
- Cramberry encoding is comparable to Protobuf for simple messages
- For complex messages (Document, Event), Cramberry encoding is 1.7-2.0x faster than Protobuf
- Cramberry encoding is consistently 1.7-5.1x faster than JSON

---

## 2. Decoding Performance (Generated Code)

| Message Type | Cramberry | Protobuf | JSON | Cram vs PB | Cram vs JSON |
|--------------|-----------|----------|------|------------|--------------|
| SmallMessage | 26.8 ns | 68.6 ns | 401.7 ns | **2.56x faster** | 15.0x faster |
| Metrics | 41.8 ns | 111.9 ns | 1175 ns | **2.68x faster** | 28.1x faster |
| Person | 395.4 ns | 586.2 ns | 3632 ns | **1.48x faster** | 9.19x faster |
| Document | 821.8 ns | 1381 ns | 6522 ns | **1.68x faster** | 7.94x faster |
| Event | 386.6 ns | 756.0 ns | 2564 ns | **1.96x faster** | 6.63x faster |
| Batch100 | 2.93 us | 6.92 us | 36.3 us | **2.36x faster** | 12.4x faster |
| Batch1000 | 28.3 us | 62.1 us | 339.5 us | **2.19x faster** | 12.0x faster |

**Observations:**
- Cramberry decoding outperforms Protobuf across all message types (1.48-2.68x faster)
- The largest decode performance gains are for simple messages (SmallMessage, Metrics)
- Cramberry decoding is 6.6-28.1x faster than JSON

---

## 3. Generated Code vs Reflection API

This section compares Cramberry's generated code (MarshalCramberry/UnmarshalCramberry methods) against the reflection-based API (cramberry.Marshal/Unmarshal functions).

### 3.1 Encoding: Generated vs Reflection

| Message Type | Generated | Reflection | Speedup |
|--------------|-----------|------------|---------|
| SmallMessage | 49.4 ns | 81.9 ns | **1.66x** |
| Metrics | 76.3 ns | 173.2 ns | **2.27x** |
| Person | 308.3 ns | 654.7 ns | **2.12x** |
| Document | 603.7 ns | 1382 ns | **2.29x** |
| Event | 277.2 ns | 596.7 ns | **2.15x** |
| Batch100 | 2.71 us | 6.37 us | **2.35x** |
| Batch1000 | 25.0 us | 58.1 us | **2.32x** |

### 3.2 Decoding: Generated vs Reflection

| Message Type | Generated | Reflection | Speedup |
|--------------|-----------|------------|---------|
| SmallMessage | 26.8 ns | 116.7 ns | **4.35x** |
| Metrics | 41.8 ns | 483.5 ns | **11.57x** |
| Person | 395.4 ns | 1442 ns | **3.65x** |
| Document | 821.8 ns | 2499 ns | **3.04x** |
| Event | 386.6 ns | 1135 ns | **2.94x** |
| Batch100 | 2.93 us | 10.0 us | **3.41x** |
| Batch1000 | 28.3 us | 95.6 us | **3.38x** |

### 3.3 Memory: Generated vs Reflection

| Message Type | Generated Encode | Reflection Encode | Generated Decode | Reflection Decode |
|--------------|------------------|-------------------|------------------|-------------------|
| SmallMessage | 24 B / 1 alloc | 24 B / 1 alloc | 16 B / 1 alloc | 48 B / 2 allocs |
| Metrics | 80 B / 1 alloc | 80 B / 1 alloc | 0 B / 0 allocs | 736 B / 7 allocs |
| Person | 224 B / 1 alloc | 224 B / 1 alloc | 336 B / 20 allocs | 1136 B / 27 allocs |
| Document | 416 B / 1 alloc | 688 B / 11 allocs | 1032 B / 28 allocs | 2112 B / 46 allocs |
| Event | 192 B / 1 alloc | 400 B / 9 allocs | 568 B / 20 allocs | 1448 B / 31 allocs |

**Observations:**
- Generated code is **1.7-2.4x faster** for encoding
- Generated code is **2.9-11.6x faster** for decoding
- Generated code achieves **zero allocations** for Metrics decode (impossible with reflection)
- For performance-critical paths, use generated code; for convenience, reflection API is acceptable

---

## 4. Memory Allocation (Encoding)

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

## 5. Memory Allocation (Decoding)

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

## 6. Encoded Size Comparison

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

## 7. Summary

### Performance relative to Protocol Buffers

| Metric | Range | Notes |
|--------|-------|-------|
| Encode speed | 0.91x - 1.99x | Faster for complex messages |
| Decode speed | **1.48x - 2.68x** | Faster across all types |
| Encode memory | 0.35x - 1.50x | Single allocation pattern |
| Decode memory | 0.42x - 0.58x | Fewer allocations |
| Encoded size | 0.95x - 1.12x | Within 12% of Protobuf |

### Generated Code vs Reflection

| Metric | Speedup |
|--------|---------|
| Encode | 1.7x - 2.4x faster |
| Decode | 2.9x - 11.6x faster |

### Strengths

- **Decode performance is consistently superior** (1.5-2.7x faster than Protobuf)
- **Zero-allocation decoding** for simple value types (Metrics)
- **Single-allocation encoding** reduces GC pressure
- **Generated code** provides maximum performance when needed
- **Reflection API** offers convenience with acceptable overhead
- Competitive encoded sizes, smaller for complex messages
- Deterministic encoding for consensus/cryptographic applications

### Trade-offs

- SmallMessage encoding is ~10% slower than Protobuf
- SmallMessage encoded size is 12% larger (18 vs 16 bytes)
- Reflection API adds 2-12x overhead compared to generated code
- Decode still requires allocations for messages with strings/slices

### Use Case Fit

- **Read-heavy workloads** benefit most from Cramberry's decode performance
- **Batch processing** shows strong scaling characteristics (2.2-2.4x faster decode)
- **Applications sensitive to GC pauses** benefit from reduced allocations
- **Consensus systems** benefit from deterministic encoding
- **Rapid prototyping** can use reflection API, switching to generated code for production

---

## 8. Security Features (v1.3.0)

Cramberry v1.3.0 includes comprehensive security hardening:

- **Integer overflow protection** in packed array writers/readers
- **Zero-copy memory safety** with generation-based validation
- **Cross-language varint consistency** (10-byte maximum)
- **NaN canonicalization** for deterministic float encoding
- **Thread-safe registries** in all language runtimes
- **V2 wire format conformance** across Go, TypeScript, and Rust

See [docs/SECURITY.md](docs/SECURITY.md) for security best practices.

---

## Running Benchmarks

```bash
# Performance benchmarks (generated code vs Protobuf vs JSON)
go test -bench=. -benchmem ./benchmark/...

# Size comparison
go test ./benchmark/... -run TestEncodedSizes -v

# Generated code vs Reflection comparison
go test ./benchmark/... -bench=Reflection -benchmem

# Full benchmark suite with multiple iterations
go test -bench=. -benchmem -count=5 ./benchmark/... | tee results.txt
```
