# Cramberry

A high-performance, deterministic binary serialization library for Go with cross-language support.

## Features

- **Compact Binary Format**: Variable-length encoding minimizes wire size
- **Deterministic Encoding**: Same input always produces identical output
- **Full Go Type Support**: All primitives, structs, slices, maps, and pointers
- **Polymorphic Serialization**: Amino-style interface support with type registration
- **Custom Type Registration**: Define optimized encoding for domain-specific types
- **Schema Definition Language**: Language-agnostic schema format for code generation
- **Cross-Language Support**: Code generation for Go, TypeScript, and Rust
- **Streaming Support**: Encode/decode large messages incrementally
- **Type-Safe Decoding**: Decode directly to specific types, not just `interface{}`

## Installation

```bash
go get github.com/cramberry/cramberry-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/cramberry/cramberry-go/pkg/cramberry"
)

type Person struct {
    Name string `cramberry:"1"`
    Age  int    `cramberry:"2"`
}

func main() {
    // Encode
    p := &Person{Name: "Alice", Age: 30}
    data, err := cramberry.Marshal(p)
    if err != nil {
        panic(err)
    }

    // Decode
    var decoded Person
    err = cramberry.Unmarshal(data, &decoded)
    if err != nil {
        panic(err)
    }

    fmt.Printf("%+v\n", decoded)
}
```

## Polymorphic Types

```go
// Define an interface
type Message interface {
    GetContent() string
}

// Register interface and implementations
func init() {
    cramberry.RegisterInterface[Message]("app.Message")
    cramberry.RegisterImplementation[Message, TextMessage](128, "app.TextMessage")
    cramberry.RegisterImplementation[Message, ImageMessage](129, "app.ImageMessage")
}

// Encode interface values
var msg Message = &TextMessage{Text: "Hello"}
data, _ := cramberry.Marshal(msg)

// Decode with type safety
decoded, _ := cramberry.UnmarshalInterface[Message](data)
```

## Schema Definition

```cramberry
package myapp.models;

@typeid(128)
type Person {
    @field(1) name: string;
    @field(2) age: int32;
    @field(3) email: string?;
}
```

## Code Generation

```bash
# Generate Go code from schema
cramberry generate --lang=go --out=./gen schema/*.cramberry

# Generate TypeScript
cramberry generate --lang=typescript --out=./gen/ts schema/*.cramberry
```

## Documentation

- [Architecture Design](ARCHITECTURE.md)
- [Implementation Plan](IMPLEMENTATION_PLAN.md)
- [Wire Format Specification](docs/wire-format.md)
- [Schema Language Guide](docs/schema-language.md)

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
