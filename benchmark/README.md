# Cramberry Performance Benchmarks

This directory contains comprehensive benchmarks comparing Cramberry serialization
against Protocol Buffers and JSON.

## Structure

```
benchmark/
├── schemas/
│   ├── messages.cramberry    # Cramberry schema definitions
│   └── messages.proto        # Equivalent Protobuf definitions
├── gen/
│   ├── cramberry/           # Generated Cramberry Go code
│   └── protobuf/            # Generated Protobuf Go code
├── benchmark_test.go         # Benchmark tests
└── README.md
```

## Running Benchmarks

### Full Benchmark Suite

```bash
go test ./benchmark/... -bench=. -benchmem
```

### Specific Message Type

```bash
# Small message (baseline)
go test ./benchmark/... -bench=SmallMessage -benchmem

# Scalar-heavy message
go test ./benchmark/... -bench=Metrics -benchmem

# Nested messages
go test ./benchmark/... -bench=Person -benchmem

# Complex with arrays/maps
go test ./benchmark/... -bench=Document -benchmem

# Event messages
go test ./benchmark/... -bench=Event -benchmem

# Batch operations
go test ./benchmark/... -bench=Batch -benchmem
```

### Size Comparison

```bash
go test ./benchmark/... -run TestEncodedSizes -v
```

## Regenerating Code

If you modify the schemas, regenerate the code:

```bash
# Cramberry
go run ./cmd/cramberry generate -out benchmark/gen/cramberry -package cramgen benchmark/schemas/messages.cramberry

# Protobuf
protoc --go_out=. --go_opt=paths=source_relative benchmark/schemas/messages.proto
mv benchmark/schemas/messages.pb.go benchmark/gen/protobuf/
```

## Test Scenarios

| Scenario | Description |
|----------|-------------|
| SmallMessage | Minimal 3-field message for baseline |
| Metrics | 10 scalar fields (integers and floats) |
| Person | Deeply nested with optional fields |
| Document | Arrays, maps, and nested messages |
| Event | Maps with string keys, bytes payload |
| Batch100 | 100 repeated messages |
| Batch1000 | 1000 repeated messages |

## Typical Results

On Apple M4 Pro (example):

### Encoded Sizes

| Message       | Cramberry | Protobuf | JSON    | Cram/PB | JSON/PB |
|---------------|-----------|----------|---------|---------|---------|
| SmallMessage  |        18 |       16 |      45 |   1.12x |   2.81x |
| Metrics       |        76 |       75 |     154 |   1.01x |   2.05x |
| Person        |       212 |      212 |     540 |   1.00x |   2.55x |
| Document      |       412 |      419 |     930 |   0.98x |   2.22x |
| Event         |       178 |      183 |     395 |   0.97x |   2.16x |
| Batch100      |      1723 |     1790 |    4623 |   0.96x |   2.58x |
| Batch1000     |     17024 |    17962 |   45573 |   0.95x |   2.54x |

### Performance Summary

- **Cramberry vs Protobuf**: Cramberry uses reflection-based encoding while Protobuf
  uses generated code. Protobuf encoding is typically 2-3x faster for simple messages,
  but the gap narrows for complex messages with many allocations.

- **Cramberry vs JSON**: Cramberry is significantly faster than JSON for decoding
  (3-5x) and produces ~2-3x smaller encoded output.

- **Memory**: Cramberry has competitive allocation patterns. For batch operations,
  Cramberry often allocates less memory than Protobuf despite being reflection-based.

## Key Takeaways

1. **Size efficiency**: Cramberry produces encoded output comparable to Protobuf
   and 2-3x smaller than JSON.

2. **Decode performance**: Cramberry decoding is much faster than JSON and
   competitive with Protobuf for complex messages.

3. **Allocation efficiency**: Cramberry's allocation patterns are efficient,
   especially for batch operations.

4. **Use cases**:
   - Choose Cramberry for schema-first development with good performance
   - Choose Protobuf for maximum raw speed when codegen is acceptable
   - Avoid JSON for performance-critical binary data exchange
