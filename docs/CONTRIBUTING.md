# Contributing to Cramberry

Thank you for your interest in contributing to Cramberry! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Code Style](#code-style)
- [Pull Request Process](#pull-request-process)
- [Issue Guidelines](#issue-guidelines)
- [Release Process](#release-process)

## Code of Conduct

Please be respectful and constructive in all interactions. We welcome contributors of all backgrounds and experience levels.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Node.js 18+ (for TypeScript runtime)
- Rust 1.70+ (for Rust runtime)
- Make
- Git

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR-USERNAME/cramberry.git
   cd cramberry
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/blockberries/cramberry.git
   ```

## Development Setup

### Initial Setup

```bash
# Install Go dependencies
go mod download

# Install development tools
make tools

# Verify setup
make check
```

### Project Structure

```
cramberry/
├── cmd/cramberry/         # CLI tool
├── pkg/
│   ├── cramberry/         # Core runtime library
│   ├── schema/            # Schema language parser
│   ├── codegen/           # Code generators
│   └── extract/           # Schema extraction
├── internal/wire/         # Low-level wire primitives
├── typescript/            # TypeScript runtime
├── rust/                  # Rust runtime
├── benchmark/             # Performance benchmarks
├── examples/              # Example applications
└── tests/integration/     # Cross-language tests
```

### Available Make Targets

```bash
make help        # Show all available targets
make build       # Build CLI to bin/cramberry
make test        # Run tests with race detection
make test-short  # Run tests without race detection (faster)
make lint        # Run golangci-lint
make check       # Run all checks (format, vet, lint, test)
make bench       # Run benchmarks
make ts-test     # Run TypeScript tests
make rust-test   # Run Rust tests
make clean       # Clean build artifacts
```

## Making Changes

### Branch Naming

Use descriptive branch names:
- `feature/add-streaming-support`
- `fix/decode-empty-map`
- `docs/update-migration-guide`
- `refactor/simplify-writer`

### Commit Messages

Follow conventional commit format:

```
type(scope): short description

Longer description if needed. Explain the motivation
for the change and contrast with previous behavior.

Fixes #123
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Code style (formatting, no logic change)
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(streaming): add MessageIterator for stream reading

Add an iterator pattern for reading delimited messages from streams.
This provides a more ergonomic API than manual ReadDelimited calls.

Fixes #45
```

```
fix(decoder): handle empty maps correctly

Empty maps were being decoded as nil instead of empty maps.
This caused issues when the caller expected a non-nil map.

Fixes #78
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test -v ./pkg/cramberry/...

# Run specific test
go test -v ./pkg/cramberry -run TestMarshal

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Writing Tests

1. **Test file naming**: `*_test.go` in the same package
2. **Test function naming**: `TestFunctionName`, `TestType_Method`
3. **Table-driven tests** for multiple cases
4. **Benchmark naming**: `BenchmarkFunctionName`

```go
func TestMarshal_Struct(t *testing.T) {
    tests := []struct {
        name    string
        input   any
        want    []byte
        wantErr bool
    }{
        {
            name:  "simple struct",
            input: User{ID: 1, Name: "Alice"},
            want:  []byte{...},
        },
        {
            name:    "nil pointer",
            input:   (*User)(nil),
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := cramberry.Marshal(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !bytes.Equal(got, tt.want) {
                t.Errorf("Marshal() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Coverage

Aim for high coverage on:
- Public APIs (Marshal, Unmarshal, etc.)
- Edge cases (nil, empty, max values)
- Error paths
- Wire format encoding/decoding

### Cross-Language Tests

Integration tests verify compatibility:

```bash
# Run all cross-language tests
make ts-test
make rust-test

# Run interoperability test
cd tests/integration && go test -v
```

## Code Style

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` (enforced by CI)
- Use `golangci-lint` (run `make lint`)

**Key points:**
- Export only necessary symbols
- Document all exported types/functions
- Handle all errors
- Avoid global state
- Use context for cancellation

### Naming Conventions

```go
// Public types: PascalCase
type StreamWriter struct {}

// Public functions: PascalCase
func Marshal(v any) ([]byte, error)

// Private types/functions: camelCase
type fieldInfo struct {}
func encodeStruct(w *Writer, v reflect.Value) error

// Constants: PascalCase for public, camelCase for private
const MaxVarintLen64 = 10
const defaultBufferSize = 4096
```

### Error Handling

```go
// Use sentinel errors for known conditions
var ErrInvalidVarint = errors.New("cramberry: invalid varint")

// Wrap errors with context
return fmt.Errorf("decode field %s: %w", name, err)

// Use typed errors for rich context
return &DecodeError{
    Type:   typeName,
    Field:  fieldName,
    Offset: offset,
    Cause:  err,
}
```

### Documentation

```go
// Package cramberry provides high-performance, deterministic binary serialization.
package cramberry

// Marshal encodes a Go value into cramberry binary format.
// The value must be a supported type (see package documentation).
//
// For struct types, fields are encoded in field number order.
// Field numbers are assigned based on the "cramberry" struct tag,
// or sequentially if no tag is present.
func Marshal(v any) ([]byte, error)
```

## Pull Request Process

### Before Submitting

1. **Sync with upstream**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run all checks**:
   ```bash
   make check
   ```

3. **Update documentation** if needed

4. **Add tests** for new functionality

### PR Description Template

```markdown
## Summary
Brief description of changes.

## Motivation
Why is this change needed?

## Changes
- Change 1
- Change 2

## Testing
How was this tested?

## Breaking Changes
List any breaking changes (if applicable).

## Related Issues
Fixes #123
Related to #456
```

### Review Process

1. PRs require at least one approval
2. CI must pass (tests, lint, build)
3. Address all review comments
4. Squash commits if requested
5. Maintainer will merge when ready

### After Merge

- Delete your feature branch
- Update your local main:
  ```bash
  git checkout main
  git pull upstream main
  ```

## Issue Guidelines

### Bug Reports

Include:
- Cramberry version (`cramberry version`)
- Go version (`go version`)
- Operating system
- Minimal reproduction case
- Expected vs actual behavior
- Stack trace (if applicable)

### Feature Requests

Include:
- Use case description
- Proposed API (if applicable)
- Alternative approaches considered
- Impact on existing functionality

### Labels

- `bug`: Something isn't working
- `enhancement`: New feature or request
- `documentation`: Documentation improvements
- `good first issue`: Good for newcomers
- `help wanted`: Extra attention needed
- `priority:high/medium/low`: Priority level
- `breaking`: Breaking change

## Release Process

### Version Numbering

We use [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

1. Update CHANGELOG.md
2. Update version in code
3. Run full test suite
4. Create release tag
5. Build and publish binaries
6. Update documentation

### Changelog Format

```markdown
## [1.2.0] - 2024-01-15

### Added
- New streaming MessageIterator for reading delimited messages

### Changed
- Improved decode performance for nested structs

### Fixed
- Empty map decoded as nil instead of empty map (#78)

### Deprecated
- V1Options will be removed in v2.0

### Security
- Updated dependency X to fix CVE-YYYY-XXXX
```

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Open a GitHub Issue
- **Security**: Email security@blockberries.com

## Recognition

Contributors are recognized in:
- CHANGELOG.md (for the release)
- GitHub contributors list

Thank you for contributing to Cramberry!
