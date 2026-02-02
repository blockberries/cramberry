# Cramberry

**High-Performance Binary Serialization for Go, TypeScript, and Rust**

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://go.dev)
[![GoDoc](https://pkg.go.dev/badge/github.com/blockberries/cramberry)](https://pkg.go.dev/github.com/blockberries/cramberry)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![CI Status](https://github.com/blockberries/cramberry/workflows/CI/badge.svg)](https://github.com/blockberries/cramberry/actions)

Cramberry is a **deterministic binary serialization library** designed for consensus systems, blockchain applications, and performance-critical distributed systems. It provides compact encoding (37-65% smaller than JSON), fast deserialization (2.7-3x faster), and cross-language support with byte-for-byte compatibility.

---

## Features

- **Deterministic Encoding**: Reproducible binary output for consensus and cryptographic operations
- **Compact Wire Format**: 37-65% smaller than JSON, comparable to Protocol Buffers
- **High Performance**: 2.7-3x faster deserialization than JSON, competitive with Protobuf
- **Cross-Language Support**: Native runtimes for Go, TypeScript, and Rust
- **Schema Language**: Protocol Buffer-like `.cram` schema files
- **Polymorphic Types**: Amino-style type registry for interface serialization
- **Streaming Support**: Delimited messages for large datasets
- **Deterministic JSON**: Human-readable JSON for blockchain SignDoc generation
- **Code Generation**: Generate optimized encoding/decoding code from schemas

---

## Quick Start

### Installation

**Go:**

```bash
go get github.com/blockberries/cramberry
```

**TypeScript:**

```bash
npm install @cramberry/runtime
```

**Rust:**

```bash
cargo add cramberry
```

**CLI Tool:**

```bash
go install github.com/blockberries/cramberry/cmd/cramberry@latest
```

### Basic Usage (Go)

```go
package main

import (
    "fmt"
    "github.com/blockberries/cramberry/pkg/cramberry"
)

type User struct {
    ID    int64  `cramberry:"1,required"`
    Name  string `cramberry:"2"`
    Email string `cramberry:"3"`
}

func main() {
    // Marshal
    user := User{ID: 123, Name: "Alice", Email: "alice@example.com"}
    data, err := cramberry.Marshal(user)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Encoded: %d bytes\n", len(data))

    // Unmarshal
    var decoded User
    err = cramberry.Unmarshal(data, &decoded)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Decoded: %+v\n", decoded)

    // Deterministic JSON (for SignDocs)
    jsonStr, _ := cramberry.ToJSON(user)
    fmt.Printf("JSON: %s\n", jsonStr)
    // Output: {"id":"123","name":"Alice","email":"alice@example.com"}
}
```

### Schema-Driven Development

**1. Define Schema (`user.cram`):**

```cramberry
package user;

enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    INACTIVE = 2;
}

message User {
    id: int64 = 1 [required];
    name: string = 2;
    email: string = 3;
    status: Status = 4;
    tags: []string = 5;
    metadata: map[string]string = 6;
}
```

**2. Generate Code:**

```bash
cramberry generate -lang go -out ./gen user.cram
cramberry generate -lang typescript -out ./gen user.cram
cramberry generate -lang rust -out ./gen user.cram
```

**3. Use Generated Code:**

```go
// Go
user := &gen.User{ID: 123, Name: "Alice"}
data, _ := user.EncodeTo(writer)

// TypeScript
const user = { id: 123n, name: "Alice" };
const data = encode_User(writer, user);

// Rust
let user = User { id: 123, name: "Alice".to_string() };
let data = user.encode_to(&mut writer)?;
```

---

## Use Cases

### 1. Blockchain Transaction Signing

Generate human-readable, deterministic JSON for SignDocs:

```go
tx := Transaction{
    Amount:    1000,
    Recipient: "cosmos1...",
    Fee:       100,
}

signDoc, _ := tx.ToJSON()
// signDoc = `{"amount":"1000","fee":"100","recipient":"cosmos1..."}`

signature := sign([]byte(signDoc), privateKey)
```

**Why Cramberry?**

- **Deterministic**: Identical JSON across all implementations
- **Human-Readable**: Users can review what they're signing
- **Secure**: Prevents blind signing attacks

### 2. Consensus Protocols

Achieve byte-for-byte encoding reproducibility:

```go
// Node 1 (Go)
block := Block{Height: 100, Hash: "abc123"}
encoded, _ := cramberry.Marshal(block)
hash1 := sha256.Sum256(encoded)

// Node 2 (Rust)
let block = Block { height: 100, hash: "abc123".to_string() };
let encoded = cramberry::marshal(&block)?;
let hash2 = sha256(encoded);

// hash1 == hash2 (guaranteed)
```

### 3. Microservices Communication

Compact, cross-language RPC:

```go
// Service A (Go)
request := GetUserRequest{ID: 123}
data, _ := cramberry.Marshal(request)
// Send to Service B...

// Service B (TypeScript)
const request = decode_GetUserRequest(reader, data);
// Process request...
```

**Advantages:**

- **37-65% smaller** than JSON (reduced bandwidth)
- **2.7-3x faster** deserialization (lower latency)
- **Type-safe** across languages

### 4. Data Persistence

Efficient storage with forward compatibility:

```go
// Write
data, _ := cramberry.Marshal(record)
db.Put(key, data)

// Read (even after schema evolution)
var record Record
cramberry.Unmarshal(db.Get(key), &record)
```

---

## Performance

### Benchmark Results (Apple M4 Pro)

**Encoded Sizes:**

| Message | Cramberry | Protobuf | JSON | Cram/PB | JSON/PB |
|---------|-----------|----------|------|---------|---------|
| Small   | 18 bytes  | 16 bytes | 45 bytes | 1.12x | 2.81x |
| Metrics | 76 bytes  | 75 bytes | 154 bytes | 1.01x | 2.05x |
| Complex | 412 bytes | 419 bytes | 930 bytes | 0.98x | 2.22x |

**Performance (ns/op):**

| Operation | Cramberry | Protobuf | JSON |
|-----------|-----------|----------|------|
| Encode Small | 48 ns | 45 ns | 84 ns |
| Decode Small | 27 ns | 68 ns | 403 ns |
| Encode Complex | 305 ns | 331 ns | 1,200 ns |
| Decode Complex | 392 ns | 615 ns | 2,800 ns |

**Key Takeaways:**

- ✅ **Size**: Comparable to Protobuf, 2-3x smaller than JSON
- ✅ **Decode Speed**: Faster than Protobuf, 3-5x faster than JSON
- ✅ **Encode Speed**: Competitive with Protobuf (reflection-based)
- ✅ **Memory**: Efficient allocation patterns

---

## Documentation

- **[Architecture](ARCHITECTURE.md)**: Comprehensive system design and implementation details
- **[API Reference](https://pkg.go.dev/github.com/blockberries/cramberry)**: Go package documentation
- **[Changelog](CHANGELOG.md)**: Version history and release notes
- **[Development Guide](CLAUDE.md)**: Contributing and development workflow

### Key Concepts

- **[Wire Protocol](ARCHITECTURE.md#wire-protocol-specification)**: Binary encoding format
- **[Schema Language](ARCHITECTURE.md#schema-language-pkgschema)**: `.cram` file syntax
- **[Code Generation](ARCHITECTURE.md#code-generation-system)**: Multi-language codegen
- **[Type Registry](ARCHITECTURE.md#registry-implementation-polymorphic-types)**: Polymorphic serialization
- **[JSON Serialization](ARCHITECTURE.md#deterministic-json-serialization)**: Deterministic JSON for SignDocs

---

## CLI Tool

### Commands

**Generate Code:**

```bash
cramberry generate -lang go -out ./gen schemas/*.cram
cramberry generate -lang typescript -out ./gen schemas/*.cram
cramberry generate -lang rust -out ./gen schemas/*.cram
```

**Extract Schema from Go Code:**

```bash
cramberry schema -out user.cram ./pkg/models
```

**Validate Schemas:**

```bash
cramberry validate schemas/*.cram
```

**Format Schemas:**

```bash
cramberry format schemas/*.cram
```

**Version Info:**

```bash
cramberry version
```

---

## Schema Language

### Syntax

```cramberry
package example;

// Enums (explicit values required)
enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    INACTIVE = 2;
}

// Messages (struct-like types)
message User {
    id: int64 = 1 [required];          // Required field
    name: string = 2;                  // Optional field
    email: string = 3;
    status: Status = 4;                // Enum field
    tags: []string = 5;                // Repeated field (slice)
    metadata: map[string]string = 6;   // Map field
    address: *Address = 7;             // Optional pointer
}

message Address {
    street: string = 1;
    city: string = 2;
    country: string = 3;
}

// Interfaces (polymorphic types)
interface Principal {
    User = 128;          // Type ID for User
    Organization = 129;  // Type ID for Organization
}
```

### Supported Types

| Category | Types |
|----------|-------|
| **Integers** | `int8`, `int16`, `int32`, `int64`, `int` |
| **Unsigned** | `uint8`, `uint16`, `uint32`, `uint64`, `uint`, `byte` |
| **Floats** | `float32`, `float64` |
| **Complex** | `complex64`, `complex128` (Go only) |
| **Boolean** | `bool` |
| **Strings** | `string`, `bytes` |
| **Collections** | `[]T` (slice), `map[K]V` (map) |
| **Pointers** | `*T` (optional/nullable) |
| **Custom** | `User`, `pkg.User` (qualified names) |

---

## Cross-Language Support

### Runtime Feature Parity

| Feature | Go | TypeScript | Rust |
|---------|----|-----------|----|
| Binary Encoding | ✅ | ✅ | ✅ |
| Binary Decoding | ✅ | ✅ | ✅ |
| Streaming | ✅ | ✅ | ✅ |
| Type Registry | ✅ | ✅ | ✅ |
| Deterministic JSON | ✅ | ✅ | ✅ |
| Zero-Copy Strings | ✅ | ❌ | ❌ |
| Writer Pooling | ✅ | ❌ | ❌ |

### Integration Testing

**Golden File Strategy:**

1. Go generates canonical binaries (authoritative)
2. TypeScript/Rust verify byte-for-byte compatibility
3. All runtimes produce identical JSON output

**Test Coverage:**

- ✅ Scalar types (int, float, bool, string, bytes)
- ✅ Repeated types (arrays)
- ✅ Maps
- ✅ Nested messages (10+ levels deep)
- ✅ Enums
- ✅ Edge cases (nil, zero, max/min values)

---

## Development

### Build from Source

**Prerequisites:**

- Go 1.25+
- TypeScript 5.0+ (for TS runtime)
- Rust 1.70+ (for Rust runtime)

**Clone and Build:**

```bash
git clone https://github.com/blockberries/cramberry.git
cd cramberry
make build
```

**Run Tests:**

```bash
make test           # Go tests with race detection
make test-short     # Fast tests
make ts-test        # TypeScript tests
make rust-test      # Rust tests
```

**Run Benchmarks:**

```bash
make bench
```

**Linting:**

```bash
make lint           # golangci-lint
make check          # All checks (fmt, vet, lint, test)
```

### Project Structure

```
cramberry/
├── cmd/cramberry/          # CLI application
├── pkg/                    # Public APIs
│   ├── cramberry/          # Main runtime
│   ├── schema/             # Schema parser
│   ├── codegen/            # Code generators
│   └── extract/            # Schema extraction
├── internal/               # Private implementation
│   ├── wire/               # Wire protocol
│   └── bench/              # Benchmarks
├── typescript/             # TypeScript runtime
├── rust/                   # Rust runtime
├── test/integration/       # Cross-language tests
├── testdata/               # Test fixtures
├── examples/               # Example applications
├── Makefile                # Build automation
├── ARCHITECTURE.md         # System design docs
├── README.md               # This file
└── CLAUDE.md               # Development guide
```

---

## Contributing

We welcome contributions! Please follow these guidelines:

### Code Style

- **Go**: Follow [Effective Go](https://go.dev/doc/effective_go) and use `gofmt`
- **TypeScript**: Use prettier and ESLint
- **Rust**: Use `rustfmt` and `clippy`

### Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`make check`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Testing Requirements

- **Unit tests**: All new code must have tests
- **Integration tests**: Cross-language changes require integration tests
- **Coverage**: Maintain >70% coverage
- **Benchmarks**: Performance-critical changes require benchmarks

---

## Testing

### Run Tests

```bash
# All Go tests
make test

# Fast tests (no race detection)
make test-short

# Single package
go test -v ./pkg/cramberry

# Single test function
go test -v ./pkg/cramberry -run TestMarshal

# With coverage
go test -cover ./...
make coverage  # Generate HTML report
```

### Run Benchmarks

```bash
# All benchmarks
make bench

# Specific benchmark
go test -bench=BenchmarkMarshal ./pkg/cramberry

# With memory profiling
go test -bench=. -benchmem ./pkg/cramberry

# Compare with baseline
go test -bench=. ./pkg/cramberry > new.txt
benchstat old.txt new.txt
```

### Cross-Language Tests

```bash
# All runtimes
make test
make ts-test
make rust-test

# Integration tests
go test -v ./test/integration
```

---

## Examples

### Example 1: Basic Marshaling

```go
package main

import (
    "fmt"
    "github.com/blockberries/cramberry/pkg/cramberry"
)

type Message struct {
    ID      int64  `cramberry:"1,required"`
    Content string `cramberry:"2"`
}

func main() {
    msg := Message{ID: 42, Content: "Hello, World!"}

    // Marshal
    data, _ := cramberry.Marshal(msg)
    fmt.Printf("Encoded: %x\n", data)

    // Unmarshal
    var decoded Message
    cramberry.Unmarshal(data, &decoded)
    fmt.Printf("Decoded: %+v\n", decoded)
}
```

### Example 2: Streaming

```go
package main

import (
    "os"
    "github.com/blockberries/cramberry/pkg/cramberry"
)

type LogEntry struct {
    Timestamp int64  `cramberry:"1"`
    Message   string `cramberry:"2"`
}

func main() {
    // Write delimited messages
    file, _ := os.Create("log.bin")
    defer file.Close()

    writer := cramberry.NewStreamWriter(file)
    for i := 0; i < 1000; i++ {
        entry := LogEntry{Timestamp: int64(i), Message: "Log entry"}
        writer.WriteMessage(entry)
    }

    // Read delimited messages
    file2, _ := os.Open("log.bin")
    defer file2.Close()

    reader := cramberry.NewStreamReader(file2)
    for {
        var entry LogEntry
        err := reader.ReadMessage(&entry)
        if err != nil {
            break  // EOF
        }
        fmt.Printf("Entry: %+v\n", entry)
    }
}
```

### Example 3: Polymorphic Types

```go
package main

import (
    "fmt"
    "github.com/blockberries/cramberry/pkg/cramberry"
)

type Animal interface {
    Speak() string
}

type Dog struct {
    Name string `cramberry:"1"`
}

func (d Dog) Speak() string { return "Woof!" }

type Cat struct {
    Name string `cramberry:"1"`
}

func (c Cat) Speak() string { return "Meow!" }

func init() {
    // Register types with explicit IDs
    cramberry.RegisterWithID[Dog](128)
    cramberry.RegisterWithID[Cat](129)
}

func main() {
    animals := []Animal{
        Dog{Name: "Buddy"},
        Cat{Name: "Whiskers"},
    }

    // Marshal slice of interfaces
    data, _ := cramberry.Marshal(animals)

    // Unmarshal
    var decoded []Animal
    cramberry.Unmarshal(data, &decoded)

    for _, animal := range decoded {
        fmt.Printf("%T: %s\n", animal, animal.Speak())
    }
    // Output:
    // Dog: Woof!
    // Cat: Meow!
}
```

### Example 4: Deterministic JSON

```go
package main

import (
    "fmt"
    "github.com/blockberries/cramberry/pkg/cramberry"
)

type Transaction struct {
    From   string            `cramberry:"1,required"`
    To     string            `cramberry:"2,required"`
    Amount int64             `cramberry:"3,required"`
    Memo   string            `cramberry:"4"`
    Fees   map[string]int64  `cramberry:"5"`
}

func main() {
    tx := Transaction{
        From:   "alice",
        To:     "bob",
        Amount: 1000,
        Memo:   "Payment for services",
        Fees:   map[string]int64{"network": 10, "gas": 5},
    }

    // Generate deterministic JSON (for SignDoc)
    jsonStr, _ := cramberry.ToJSON(tx)
    fmt.Println(jsonStr)

    // Output (deterministic, no whitespace, sorted maps):
    // {"amount":"1000","fees":{"gas":"5","network":"10"},"from":"alice","memo":"Payment for services","to":"bob"}

    // User reviews and signs this JSON
    signature := sign([]byte(jsonStr), privateKey)

    // Verify signature
    valid := verify([]byte(jsonStr), signature, publicKey)
    fmt.Printf("Signature valid: %v\n", valid)
}
```

---

## Roadmap

### v1.6 (Short-Term)

- [ ] Performance optimizations (SIMD varint, generated code inlining)
- [ ] Schema versioning and migration tools
- [ ] VS Code extension (syntax highlighting, validation)

### v1.7-v1.8 (Mid-Term)

- [ ] Python, Java, C++ runtimes
- [ ] Compression (zstd/gzip) at wire level
- [ ] Schema registry server

### v2.0+ (Long-Term)

- [ ] Wire protocol v3
- [ ] RPC framework (Cramberry-based gRPC alternative)
- [ ] Public package registry

---

## FAQ

**Q: How does Cramberry compare to Protocol Buffers?**

A: Cramberry is designed specifically for **deterministic encoding** (essential for consensus systems), while Protobuf focuses on **performance**. Cramberry is slightly slower at encoding (reflection-based) but offers comparable or faster decoding, similar size efficiency, and cross-language JSON compatibility.

**Q: Should I use Cramberry or JSON?**

A: Use Cramberry when you need **compact encoding** (37-65% smaller), **fast deserialization** (2.7-3x faster), or **deterministic JSON** (blockchain SignDocs). Use JSON when you need **human readability** everywhere or **broad tooling support**.

**Q: Is Cramberry production-ready?**

A: Yes. Cramberry is extensively tested (>70% coverage), fuzz-tested, and used in blockchain applications. However, it's currently at v1.5 (pre-v2.0 stability).

**Q: Can I use Cramberry with existing Go structs?**

A: Yes. Add `cramberry` struct tags to your existing types:

```go
type User struct {
    ID   int64  `json:"id" cramberry:"1,required"`
    Name string `json:"name" cramberry:"2"`
}
```

**Q: Does Cramberry support schema evolution?**

A: Yes. Unknown fields are skipped during decoding (forward compatibility), and optional fields (pointers, `omitempty`) enable backward compatibility.

**Q: How do I handle breaking changes in schemas?**

A: Use **interface versioning** with separate type IDs:

```cramberry
interface UserV1 { User = 128; }
interface UserV2 { UserV2 = 129; }
```

**Q: Is Cramberry thread-safe?**

A: The **Registry** is thread-safe (concurrent registration/lookup). **Writer/Reader** objects are **not** thread-safe (single-goroutine use, but can be pooled across goroutines).

---

## License

Cramberry is licensed under the **Apache License 2.0**. See [LICENSE](LICENSE) for details.

---

## Community

- **GitHub**: [github.com/blockberries/cramberry](https://github.com/blockberries/cramberry)
- **Issues**: [Report bugs or request features](https://github.com/blockberries/cramberry/issues)
- **Discussions**: [Ask questions or share ideas](https://github.com/blockberries/cramberry/discussions)
- **Documentation**: [pkg.go.dev](https://pkg.go.dev/github.com/blockberries/cramberry)

---

## Acknowledgments

Cramberry is inspired by:

- **Protocol Buffers**: Wire format and schema language design
- **Amino (Cosmos SDK)**: Polymorphic type registry approach
- **Cap'n Proto**: Zero-copy serialization concepts
- **FlatBuffers**: Performance optimization techniques

Special thanks to the Go, TypeScript, and Rust communities for their excellent tooling and libraries.

---

**Built with ❤️ by the Blockberries Team**
