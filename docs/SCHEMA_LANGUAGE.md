# Cramberry Schema Language Reference

This document provides a complete reference for the Cramberry schema language (`.cram` files).

## Table of Contents

- [Overview](#overview)
- [Syntax Elements](#syntax-elements)
- [Packages](#packages)
- [Imports](#imports)
- [Options](#options)
- [Enums](#enums)
- [Messages](#messages)
- [Interfaces](#interfaces)
- [Types](#types)
- [Comments](#comments)
- [Complete Example](#complete-example)
- [Best Practices](#best-practices)

## Overview

Cramberry schema files use the `.cram` extension and define types for code generation. The schema language is designed to be:

- **Familiar** - Similar syntax to Protocol Buffers and Go
- **Type-safe** - Strong typing with validation
- **Cross-language** - Generates code for Go, TypeScript, and Rust

## Syntax Elements

### File Structure

```cramberry
// Package declaration (required, first statement)
package mypackage;

// Options (optional)
option go_package = "github.com/myorg/myapp/gen";

// Imports (optional)
import "other.cram";

// Type definitions
enum MyEnum { ... }
message MyMessage { ... }
interface MyInterface { ... }
```

### Identifiers

- Must start with a letter (a-z, A-Z)
- May contain letters, digits, and underscores
- Are case-sensitive
- Convention: PascalCase for types, snake_case for fields

### Literals

```cramberry
// Integers
42
-17
0

// Strings (for options)
"path/to/package"
```

## Packages

Every schema file must declare a package:

```cramberry
package mypackage;
```

The package name:
- Used as the namespace for generated code
- Should match the directory structure
- Is required (must be the first statement)

## Imports

Import other schema files:

```cramberry
import "common/types.cram";
import "models/user.cram";
```

Imported types can be referenced by their full name:

```cramberry
message Order {
    user: common.User = 1;  // Type from imported schema
}
```

## Options

Configure code generation behavior:

```cramberry
// Go package path
option go_package = "github.com/myorg/myapp/gen/models";

// TypeScript module
option ts_module = "@myapp/models";

// Rust crate
option rust_crate = "myapp_models";
```

### Available Options

| Option | Description |
|--------|-------------|
| `go_package` | Go import path for generated code |
| `ts_module` | TypeScript/JavaScript module name |
| `rust_crate` | Rust crate name |

## Enums

Define enumerated types:

```cramberry
enum Status {
    UNKNOWN = 0;    // First value should be 0 (zero value)
    ACTIVE = 1;
    SUSPENDED = 2;
    DELETED = 3;
}
```

### Enum Rules

1. Values must be unique within the enum
2. First value should be 0 (represents the zero value)
3. Values must be non-negative integers
4. Names must be unique within the enum

### Enum with Documentation

```cramberry
/// Status represents the account state.
enum Status {
    /// Default/unknown status.
    UNKNOWN = 0;

    /// Account is active and operational.
    ACTIVE = 1;

    /// Account has been temporarily suspended.
    SUSPENDED = 2;

    /// Account has been permanently deleted.
    DELETED = 3;
}
```

## Messages

Define structured types:

```cramberry
message User {
    id: int64 = 1;
    name: string = 2;
    email: string = 3;
}
```

### Field Syntax

```cramberry
field_name: type = field_number [options];
```

- **field_name** - Identifier using snake_case convention
- **type** - Scalar, enum, message, or collection type
- **field_number** - Positive integer (1-536870911), must be unique per message
- **options** - Optional field modifiers in brackets

### Field Options

```cramberry
message User {
    // Required field - must be present when decoding
    id: int64 = 1 [required];

    // Optional field (explicit) - may be absent
    nickname: optional string = 2;

    // Repeated field - zero or more values
    tags: repeated string = 3;
    // Or using slice syntax:
    tags: []string = 3;
}
```

### Field Modifiers

| Modifier | Description |
|----------|-------------|
| `required` | Field must be present |
| `optional` | Field may be absent (default for pointers) |
| `repeated` | Zero or more values (slice/array) |

### Nested Messages

```cramberry
message Order {
    id: int64 = 1;

    // Nested message definition
    message Item {
        product_id: int64 = 1;
        quantity: int32 = 2;
        price: float64 = 3;
    }

    items: []Item = 2;
}
```

### Pointer Fields

Use `*` prefix for optional single values:

```cramberry
message Profile {
    user: User = 1;           // Embedded, always present
    address: *Address = 2;    // Pointer, may be nil
}
```

## Interfaces

Define polymorphic types:

```cramberry
interface Principal {
    User = 128;
    Organization = 129;
    ServiceAccount = 130;
}
```

### Interface Syntax

```cramberry
interface InterfaceName {
    ConcreteType1 = type_id_1;
    ConcreteType2 = type_id_2;
}
```

- **InterfaceName** - The interface type name
- **ConcreteType** - A message type that implements this interface
- **type_id** - Unique TypeID (must be >= 128 for user types)

### Interface Rules

1. TypeIDs must be unique across all interfaces
2. User-defined types must use IDs >= 128
3. Referenced concrete types must be defined messages
4. One message can implement multiple interfaces

### Example with Interface

```cramberry
message User {
    id: int64 = 1;
    name: string = 2;
}

message Organization {
    id: int64 = 1;
    name: string = 2;
    members: []User = 3;
}

// Both User and Organization can be used where Principal is expected
interface Principal {
    User = 128;
    Organization = 129;
}

message AuditLog {
    actor: Principal = 1;     // Can be User or Organization
    action: string = 2;
    timestamp: int64 = 3;
}
```

## Types

### Scalar Types

| Type | Description | Go Type | Wire Type |
|------|-------------|---------|-----------|
| `bool` | Boolean | `bool` | Varint |
| `int8` | 8-bit signed | `int8` | SVarint |
| `int16` | 16-bit signed | `int16` | SVarint |
| `int32` | 32-bit signed | `int32` | SVarint |
| `int64` | 64-bit signed | `int64` | SVarint |
| `uint8` | 8-bit unsigned | `uint8` | Varint |
| `uint16` | 16-bit unsigned | `uint16` | Varint |
| `uint32` | 32-bit unsigned | `uint32` | Varint |
| `uint64` | 64-bit unsigned | `uint64` | Varint |
| `float32` | 32-bit float | `float32` | Fixed32 |
| `float64` | 64-bit float | `float64` | Fixed64 |
| `string` | UTF-8 string | `string` | Bytes |
| `bytes` | Byte slice | `[]byte` | Bytes |

### Collection Types

```cramberry
// Slice/repeated
tags: []string = 1;
scores: []int32 = 2;

// Map (primitive keys only)
metadata: map[string]string = 3;
counts: map[int64]int32 = 4;

// Array (fixed size) - less common
coordinates: [3]float64 = 5;
```

### Map Key Restrictions

Map keys must be primitive types:
- `string`
- `bool`
- Integer types: `int8`, `int16`, `int32`, `int64`, `uint8`, `uint16`, `uint32`, `uint64`
- Float types: `float32`, `float64`

**Not allowed as map keys:**
- Messages/structs
- Slices/arrays
- Other maps
- Interfaces

### Complex Types

```cramberry
message Document {
    // Embedded message
    author: User = 1;

    // Pointer to message (optional)
    reviewer: *User = 2;

    // Slice of messages
    comments: []Comment = 3;

    // Map with message values
    versions: map[string]DocumentVersion = 4;

    // Enum
    status: Status = 5;

    // Interface (polymorphic)
    owner: Principal = 6;
}
```

## Comments

### Single-line Comments

```cramberry
// This is a single-line comment
message User {
    id: int64 = 1;  // Inline comment
}
```

### Documentation Comments

Use `///` for documentation that gets included in generated code:

```cramberry
/// User represents a registered user in the system.
///
/// Users can have multiple roles and belong to organizations.
message User {
    /// Unique identifier for the user.
    id: int64 = 1 [required];

    /// User's display name.
    name: string = 2;

    /// Email address (must be unique).
    email: string = 3;
}
```

## Complete Example

```cramberry
// user.cram - User management types

package example.users;

option go_package = "github.com/myorg/myapp/gen/users";

/// Status represents account status.
enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    SUSPENDED = 2;
    DELETED = 3;
}

/// Role defines permission levels.
enum Role {
    GUEST = 0;
    USER = 1;
    MODERATOR = 2;
    ADMIN = 3;
}

/// Address represents a physical or mailing address.
message Address {
    street: string = 1;
    city: string = 2;
    state: string = 3;
    country: string = 4;
    zip_code: string = 5;
}

/// User represents a registered user.
message User {
    /// Unique user identifier.
    id: int64 = 1 [required];

    /// User's display name.
    name: string = 2;

    /// Email address (must be unique).
    email: string = 3;

    /// Account status.
    status: Status = 4;

    /// User's permission role.
    role: Role = 5;

    /// Optional profile picture URL.
    avatar_url: optional string = 6;

    /// User's addresses (home, work, etc).
    addresses: []Address = 7;

    /// Custom metadata.
    metadata: map[string]string = 8;

    /// Account creation timestamp (Unix seconds).
    created_at: int64 = 9;

    /// Last update timestamp (Unix seconds).
    updated_at: int64 = 10;
}

/// Organization represents a company or group.
message Organization {
    id: int64 = 1 [required];
    name: string = 2;
    description: string = 3;
    members: []User = 4;
    owner: *User = 5;
    created_at: int64 = 6;
}

/// Principal represents any authenticatable entity.
interface Principal {
    User = 128;
    Organization = 129;
}
```

## Best Practices

### Field Numbers

1. **Use 1-15 for frequent fields** - These use single-byte tags in V2 format
2. **Reserve ranges for extensions** - e.g., 100-199 for future use
3. **Never reuse field numbers** - Even after removing a field
4. **Document removed fields** - Add comment about retired numbers

```cramberry
message User {
    // Common fields (1-15, single-byte tag)
    id: int64 = 1;
    name: string = 2;
    email: string = 3;

    // Less common fields (16+, multi-byte tag)
    bio: string = 16;
    website: string = 17;

    // Field 4 was 'age', removed in v2.0
    // Field 5 was 'phone', removed in v2.1

    // Reserved for future use: 100-199
}
```

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Package | lowercase, dots | `example.users` |
| Message | PascalCase | `UserProfile` |
| Field | snake_case | `created_at` |
| Enum | PascalCase | `Status` |
| Enum Value | SCREAMING_SNAKE | `ACTIVE_STATUS` |
| Interface | PascalCase | `Principal` |

### Enum Best Practices

1. Always include a zero value (UNKNOWN, UNSPECIFIED, DEFAULT)
2. Use meaningful prefixes if values might conflict
3. Document the meaning of each value

```cramberry
enum OrderStatus {
    ORDER_STATUS_UNKNOWN = 0;   // Prefix prevents conflicts
    ORDER_STATUS_PENDING = 1;
    ORDER_STATUS_CONFIRMED = 2;
    ORDER_STATUS_SHIPPED = 3;
    ORDER_STATUS_DELIVERED = 4;
    ORDER_STATUS_CANCELLED = 5;
}
```

### Interface TypeIDs

1. Start user types at 128
2. Group related types in ranges
3. Document TypeID assignments
4. Never change TypeIDs after deployment

```cramberry
// TypeID assignments:
// 128-199: User types
// 200-299: Order types
// 300-399: Payment types

interface Principal {
    User = 128;
    Organization = 129;
    ServiceAccount = 130;
}

interface PaymentMethod {
    CreditCard = 300;
    BankTransfer = 301;
    Cryptocurrency = 302;
}
```

### Documentation

1. Document every public type
2. Document non-obvious fields
3. Include units for numeric fields
4. Note constraints and validation rules

```cramberry
/// Order represents a customer purchase order.
///
/// Orders progress through states: PENDING -> CONFIRMED -> SHIPPED -> DELIVERED
/// or may be CANCELLED at any point before SHIPPED.
message Order {
    /// Unique order identifier (UUID format).
    id: string = 1 [required];

    /// Order total in cents (USD).
    total_cents: int64 = 2;

    /// Maximum items per order: 100.
    items: []OrderItem = 3;

    /// Estimated delivery time (Unix timestamp, seconds).
    estimated_delivery_at: int64 = 4;
}
```
