# Cross-Runtime Integration Tests

This directory contains integration tests that verify Cramberry's binary encoding
compatibility across Go, TypeScript, and Rust runtimes.

## Test Strategy

The tests use a "golden file" approach where Go generates canonical binary files
that all runtimes must be able to produce and consume identically.

### How It Works

1. **Golden File Generation** (Go)
   - Go generates binary files for each test case
   - Files are stored in `../golden/` with `.bin` and `.hex` extensions
   - Run with `GENERATE_GOLDEN=1 go test ./tests/integration/...`

2. **Go Tests** (`interop_test.go`)
   - Verifies encode/decode roundtrips
   - Verifies current encoding matches golden files
   - Tests all data types: scalars, arrays, maps, nested messages, enums

3. **TypeScript Tests** (`ts/interop.test.ts`)
   - Encodes test data and compares byte-for-byte with Go encoding
   - Decodes golden files and verifies correct values
   - Uses the same test data as Go

4. **Rust Tests** (`rust/src/interop_test.rs`)
   - Encodes test data and compares byte-for-byte with Go encoding
   - Decodes golden files and verifies correct values
   - Uses the same test data as Go

### Cross-Runtime Compatibility Guarantee

Since all runtimes produce **identical byte sequences** for the same input data,
cross-runtime compatibility is guaranteed:

- Go encode → TypeScript decode ✓ (verified via golden files)
- Go encode → Rust decode ✓ (verified via golden files)
- TypeScript encode → Go decode ✓ (TypeScript encoding == Go encoding)
- TypeScript encode → Rust decode ✓ (TypeScript encoding == Go encoding)
- Rust encode → Go decode ✓ (Rust encoding == Go encoding)
- Rust encode → TypeScript decode ✓ (Rust encoding == Go encoding)

## Test Cases

### Scalar Types
- bool, int32, int64, uint32, uint64
- float32, float64
- string, bytes

### Repeated Types
- int32 arrays
- string arrays
- bytes arrays

### Complex Types
- Nested messages (optional and required)
- Enums
- Maps (string→int, int→string)
- Arrays of nested messages

### Edge Cases
- Zero values
- Min/max integer values (±2³¹, ±2⁶³, 2³², 2⁶⁴)
- Empty strings and bytes
- Unicode strings (Chinese, emoji)

### Field Numbers
- Small (1, 15)
- Boundary (16, 127, 128)
- Large (1000)

## Running Tests

```bash
# Go
go test ./tests/integration/... -v

# TypeScript
cd tests/integration/ts && npm test

# Rust
cd tests/integration/rust && cargo test
```

## Regenerating Golden Files

If the encoding format changes, regenerate golden files:

```bash
GENERATE_GOLDEN=1 go test ./tests/integration/... -v
```

Then verify all runtimes still pass.
