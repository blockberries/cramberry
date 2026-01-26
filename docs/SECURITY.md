# Security Guide

This document describes security considerations when using Cramberry, including protection mechanisms, best practices, and how to handle untrusted input.

## Overview

Cramberry is designed with security in mind, particularly for systems processing untrusted input. The library includes configurable resource limits, input validation, and error handling to prevent common attack vectors.

## Security Hardening (v1.1.0+)

Version 1.1.0+ includes significant security improvements:

| Feature | Protection |
|---------|------------|
| Integer overflow protection | Bounds checking in packed array writers/readers |
| Zero-copy memory safety | Generation-based use-after-free detection (v1.2.0) |
| Cross-language varint consistency | 10-byte maximum across Go, TypeScript, Rust |
| NaN canonicalization | Deterministic float encoding for all NaN values |
| Thread-safe registries | Safe concurrent type registration |

## Resource Limits

### Default Limits

Cramberry enforces resource limits to prevent denial-of-service attacks:

```go
var DefaultLimits = Limits{
    MaxMessageSize:  64 * 1024 * 1024, // 64 MB
    MaxDepth:        100,               // Nesting depth
    MaxStringLength: 10 * 1024 * 1024,  // 10 MB
    MaxBytesLength:  100 * 1024 * 1024, // 100 MB
    MaxArrayLength:  1_000_000,         // 1M elements
    MaxMapSize:      1_000_000,         // 1M entries
}
```

### Secure Limits for Untrusted Input

When processing untrusted input, use `SecureOptions`:

```go
var SecureOptions = Options{
    Limits: Limits{
        MaxMessageSize:  1 * 1024 * 1024, // 1 MB
        MaxDepth:        32,
        MaxStringLength: 1 * 1024 * 1024,  // 1 MB
        MaxBytesLength:  10 * 1024 * 1024, // 10 MB
        MaxArrayLength:  10_000,
        MaxMapSize:      10_000,
    },
    ValidateUTF8:  true,
    Deterministic: true,
}

// Usage
err := cramberry.UnmarshalWithOptions(untrustedData, &msg, cramberry.SecureOptions)
```

### Custom Limits

Configure limits based on your application's requirements:

```go
customOpts := cramberry.Options{
    Limits: cramberry.Limits{
        MaxMessageSize:  100 * 1024, // 100 KB
        MaxDepth:        10,
        MaxStringLength: 1024,
        MaxBytesLength:  10 * 1024,
        MaxArrayLength:  100,
        MaxMapSize:      50,
    },
    ValidateUTF8:  true,
    StrictMode:    true,
    Deterministic: true,
}
```

## Attack Vectors and Mitigations

### 1. Memory Exhaustion

**Attack**: Malicious input specifies extremely large arrays, maps, or strings.

**Mitigation**: Resource limits prevent allocation of oversized structures:

```go
// These limits prevent memory exhaustion
MaxArrayLength: 10_000,  // Max elements in arrays
MaxMapSize:     10_000,  // Max entries in maps
MaxStringLength: 1 * 1024 * 1024,  // Max string size
```

### 2. Stack Overflow (Deep Nesting)

**Attack**: Deeply nested structures cause stack overflow during recursive decoding.

**Mitigation**: Depth limit prevents infinite recursion:

```go
MaxDepth: 32,  // Maximum nesting depth for SecureOptions
```

### 3. CPU Exhaustion (Billion Laughs)

**Attack**: Small encoded data expands to huge decoded structures.

**Mitigation**: Combined limits on total message size and element counts:

```go
MaxMessageSize: 1 * 1024 * 1024,  // Total encoded size
MaxArrayLength: 10_000,           // Decoded element limit
```

### 4. Invalid UTF-8 Strings

**Attack**: Invalid UTF-8 sequences cause downstream issues in string processing.

**Mitigation**: UTF-8 validation is enabled by default:

```go
ValidateUTF8: true,  // Reject invalid UTF-8
```

### 5. Unknown Field Injection

**Attack**: Extra fields smuggled into messages.

**Mitigation**: Enable strict mode to reject unknown fields:

```go
StrictMode: true,  // Reject messages with unknown fields
```

### 6. Type Confusion

**Attack**: Encoded type ID doesn't match expected type.

**Mitigation**: Type registry validates concrete types during decode:

```go
// Only registered types can be decoded
cramberry.RegisterOrGet[AllowedType1]()
cramberry.RegisterOrGet[AllowedType2]()
```

### 7. Zero-Copy Use-After-Free (v1.2.0+)

**Attack**: Zero-copy references used after Reader.Reset() causes memory corruption.

**Mitigation**: Generation-based validation detects invalid references:

```go
r := cramberry.NewReader(data)
zcs := r.ReadStringZeroCopy()  // Returns ZeroCopyString

r.Reset(newData)  // Increments generation counter

// Accessing the string now panics instead of returning corrupted data
_ = zcs.String()  // PANICS: "cramberry: ZeroCopyString accessed after Reader.Reset()"

// Can check validity without panicking
if zcs.Valid() {
    // Safe to use
}
```

### 8. Integer Overflow in Packed Arrays (v1.1.0+)

**Attack**: Malicious array length causes integer overflow in size calculation.

**Mitigation**: Bounds checking before multiplication:

```go
// Protected by overflow constants
const MaxPackedFloat32Length = (1 << 30) - 1  // ~1 billion elements

// Writers/readers check bounds before allocation
if len(values) > MaxPackedFloat32Length {
    return ErrMaxArrayLength
}
```

## Best Practices

### For Network Services

1. **Always use SecureOptions for untrusted input**:
   ```go
   func handleRequest(data []byte) error {
       var req Request
       if err := cramberry.UnmarshalWithOptions(data, &req, cramberry.SecureOptions); err != nil {
           return fmt.Errorf("invalid request: %w", err)
       }
       // Process request...
   }
   ```

2. **Set appropriate timeouts**:
   ```go
   ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
   defer cancel()
   // Process within timeout
   ```

3. **Log security-relevant errors**:
   ```go
   if cramberry.IsLimitExceeded(err) {
       log.Warn("Request exceeded limits", "error", err, "remote", remoteAddr)
   }
   ```

### For File Processing

1. **Check file size before reading**:
   ```go
   info, _ := file.Stat()
   if info.Size() > maxFileSize {
       return errors.New("file too large")
   }
   ```

2. **Use streaming for large files**:
   ```go
   it := cramberry.NewMessageIterator(file)
   var msg Message
   for it.Next(&msg) {
       // Process incrementally
   }
   ```

### For Type Registration

1. **Register types at startup (idempotent)**:
   ```go
   func init() {
       // RegisterOrGet is safe to call multiple times
       cramberry.RegisterOrGet[SafeType1]()
       cramberry.RegisterOrGet[SafeType2]()
   }
   ```

2. **Use explicit type IDs for stability**:
   ```go
   cramberry.RegisterOrGetWithID[User](128)
   cramberry.RegisterOrGetWithID[Order](129)
   ```

3. **Don't register sensitive internal types**:
   ```go
   // DON'T expose internal types
   // cramberry.RegisterOrGet[internalConfig]()  // Bad!

   // DO use separate DTOs
   cramberry.RegisterOrGet[PublicUserDTO]()  // Good
   ```

4. **Avoid deprecated MustRegister functions**:
   ```go
   // Deprecated: can panic on duplicate registration
   // cramberry.MustRegister[Type]()

   // Recommended: idempotent registration
   cramberry.RegisterOrGet[Type]()
   ```

## Error Handling

### Checking Error Types

```go
err := cramberry.Unmarshal(data, &msg)
if err != nil {
    switch {
    case cramberry.IsLimitExceeded(err):
        // Resource limit violation - possible attack
        log.Warn("Limit exceeded", "error", err)
        return ErrBadRequest

    case cramberry.IsFatal(err):
        // Programming error - should not happen in production
        log.Error("Fatal decode error", "error", err)
        return ErrInternalError

    case errors.Is(err, cramberry.ErrInvalidUTF8):
        // Invalid string encoding
        return ErrInvalidInput

    case errors.Is(err, cramberry.ErrUnknownType):
        // Unknown polymorphic type
        return ErrUnsupportedType

    default:
        // Generic decode error
        return ErrMalformedData
    }
}
```

### Error Types

| Error | Security Implication |
|-------|---------------------|
| `ErrMaxDepthExceeded` | Possible stack overflow attempt |
| `ErrMaxSizeExceeded` | Possible memory exhaustion attempt |
| `ErrMaxArrayLength` | Possible billion laughs attack or overflow attempt |
| `ErrMaxMapSize` | Possible memory exhaustion attempt |
| `ErrMaxStringLength` | Oversized string allocation attempt |
| `ErrMaxBytesLength` | Oversized bytes allocation attempt |
| `ErrInvalidUTF8` | Malformed or malicious string |
| `ErrUnknownType` | Type ID not in registry |
| `ErrUnregisteredType` | Attempt to encode unknown type |
| `ErrOverflow` | Integer overflow detected (v1.1.0+) |
| `ErrVarintOverflow` | Varint exceeds 10-byte maximum |

### Panic Conditions (v1.2.0+)

Zero-copy wrapper types panic instead of returning corrupted data:

| Panic Message | Cause |
|--------------|-------|
| `ZeroCopyString accessed after Reader.Reset()` | String reference used after Reader reset |
| `ZeroCopyBytes accessed after Reader.Reset()` | Bytes reference used after Reader reset |

Use `.Valid()` to check validity before access, or `.UnsafeString()`/`.UnsafeBytes()` to bypass validation.

## Deterministic Encoding

Cramberry's deterministic encoding is critical for security in certain applications:

### Cryptographic Hashing

```go
// Deterministic encoding ensures consistent hashes
data, _ := cramberry.Marshal(tx)
hash := sha256.Sum256(data)
```

### Digital Signatures

```go
// Same data always produces same encoding
data, _ := cramberry.Marshal(document)
signature := ed25519.Sign(privateKey, data)
```

### Consensus Systems

```go
// All nodes must encode identically for consensus
data, _ := cramberry.Marshal(block)
// Nodes can verify they're working on the same data
```

**Important**: Always use `DefaultOptions` or `SecureOptions` (both have `Deterministic: true`) when encoding data that will be hashed or signed. `FastOptions` disables determinism.

## Threat Model

### In Scope

Cramberry protects against:
- Memory exhaustion from oversized allocations
- Stack overflow from deep nesting
- CPU exhaustion from expansion attacks
- Invalid UTF-8 injection
- Unknown field smuggling (with StrictMode)

### Out of Scope

Cramberry does NOT protect against:
- Application-level logic vulnerabilities
- Side-channel attacks
- Timing attacks (decode time varies with input)
- Denial of service via valid but large messages

## Security Checklist

- [ ] Use `SecureOptions` for untrusted input
- [ ] Set appropriate `Limits` for your use case
- [ ] Enable `ValidateUTF8` for string validation
- [ ] Enable `StrictMode` if schema evolution isn't needed
- [ ] Register only necessary types for polymorphic decoding
- [ ] Use explicit type IDs for wire format stability
- [ ] Use `RegisterOrGet` instead of deprecated `MustRegister`
- [ ] Log security-relevant errors for monitoring
- [ ] Set request timeouts at the application level
- [ ] Validate file sizes before processing
- [ ] Use streaming for large datasets
- [ ] Check `Valid()` on zero-copy references before use (v1.2.0+)
- [ ] Upgrade to v1.2.0+ for zero-copy safety features

## Reporting Security Issues

If you discover a security vulnerability in Cramberry, please report it privately by emailing security@blockberries.com. Do not open a public issue.

We will:
1. Confirm receipt within 48 hours
2. Provide an initial assessment within 7 days
3. Work with you on coordinated disclosure
4. Credit you in the security advisory (if desired)
