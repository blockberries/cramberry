# Cramberry Performance and Size Benchmark Report

**Test Environment:**
- Platform: darwin/arm64 (Apple M4 Pro)
- Go version: 1.21+
- Comparison formats: Cramberry, Protocol Buffers, JSON

---

## 1. Encoding Performance

| Message Type | Cramberry | Protobuf | JSON | Cram vs PB | Cram vs JSON |
|--------------|-----------|----------|------|------------|--------------|
| SmallMessage | 47.0 ns | 44.6 ns | 83.9 ns | 1.05x slower | 1.79x faster |
| Metrics | 77.0 ns | 80.6 ns | 368.5 ns | 1.05x faster | 4.79x faster |
| Person | 300.3 ns | 332.9 ns | 706.5 ns | 1.11x faster | 2.35x faster |
| Document | 590.1 ns | 985.4 ns | 1278 ns | 1.67x faster | 2.17x faster |
| Event | 275.4 ns | 535.9 ns | 518.0 ns | 1.95x faster | 1.88x faster |
| Batch100 | 2.67 μs | 3.50 μs | 5.47 μs | 1.31x faster | 2.05x faster |
| Batch1000 | 24.4 μs | 30.0 μs | 54.9 μs | 1.23x faster | 2.25x faster |

**Observations:**
- Cramberry encoding is slower than Protobuf only for SmallMessage (5% difference)
- For complex messages (Document, Event), Cramberry encoding is 1.67-1.95x faster than Protobuf
- Cramberry encoding is consistently 1.8-4.8x faster than JSON

---

## 2. Decoding Performance

| Message Type | Cramberry | Protobuf | JSON | Cram vs PB | Cram vs JSON |
|--------------|-----------|----------|------|------------|--------------|
| SmallMessage | 26.8 ns | 68.2 ns | 389.8 ns | 2.55x faster | 14.5x faster |
| Metrics | 42.9 ns | 111.7 ns | 1189 ns | 2.60x faster | 27.7x faster |
| Person | 386.5 ns | 596.1 ns | 3657 ns | 1.54x faster | 9.46x faster |
| Document | 749.6 ns | 1392 ns | 6568 ns | 1.86x faster | 8.76x faster |
| Event | 373.2 ns | 677.0 ns | 2567 ns | 1.81x faster | 6.88x faster |
| Batch100 | 2.93 μs | 6.99 μs | 36.3 μs | 2.39x faster | 12.4x faster |
| Batch1000 | 26.7 μs | 61.3 μs | 337.9 μs | 2.30x faster | 12.7x faster |

**Observations:**
- Cramberry decoding outperforms Protobuf across all message types (1.54-2.60x faster)
- The largest decode performance gains are for simple messages (SmallMessage, Metrics)
- Cramberry decoding is 6.9-27.7x faster than JSON

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
| Metrics | 0 B / 0 allocs | 128 B / 1 alloc | 296 B / 5 allocs |
| Person | 336 B / 20 allocs | 808 B / 23 allocs | 848 B / 30 allocs |
| Document | 1048 B / 28 allocs | 2000 B / 51 allocs | 1880 B / 47 allocs |
| Batch1000 | 49.2 KB / 1008 allocs | 114.2 KB / 2028 allocs | 87.1 KB / 1029 allocs |

**Observations:**
- Metrics decoding achieves zero allocations in Cramberry
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
| Encode speed | 0.95x - 1.95x | Slower only for SmallMessage |
| Decode speed | 1.54x - 2.60x | Faster across all types |
| Encode memory | 0.35x - 1.50x | Single allocation pattern |
| Decode memory | 0.42x - 0.58x | Fewer allocations |
| Encoded size | 0.95x - 1.12x | Within 12% of Protobuf |

### Strengths

- Decode performance is consistently superior (1.5-2.6x faster than Protobuf)
- Single-allocation encoding reduces GC pressure
- Competitive encoded sizes, smaller for complex messages

### Weaknesses

- SmallMessage encoding is 5% slower than Protobuf
- SmallMessage encoded size is 12% larger (18 vs 16 bytes)
- Decode still requires multiple allocations for messages with strings/slices

### Use Case Fit

- Read-heavy workloads benefit most from Cramberry's decode performance
- Batch processing shows strong scaling characteristics
- Applications sensitive to GC pauses benefit from reduced allocations

---

## Running Benchmarks

```bash
# Performance benchmarks
go test -bench=. -benchmem ./benchmark/...

# Size comparison
go test ./benchmark/... -run TestEncodedSizes -v
```
