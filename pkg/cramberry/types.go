package cramberry

import "fmt"

// TypeID uniquely identifies a registered type for polymorphic serialization.
// Type IDs are used in the wire format to identify concrete types when
// encoding/decoding interface values.
type TypeID uint32

// Reserved TypeID ranges.
const (
	// TypeIDNil represents a nil value (no type).
	TypeIDNil TypeID = 0

	// TypeIDBuiltinStart is the start of the built-in type range.
	TypeIDBuiltinStart TypeID = 1

	// TypeIDBuiltinEnd is the end of the built-in type range (inclusive).
	TypeIDBuiltinEnd TypeID = 63

	// TypeIDStdlibStart is the start of the standard library type range.
	TypeIDStdlibStart TypeID = 64

	// TypeIDStdlibEnd is the end of the standard library type range (inclusive).
	TypeIDStdlibEnd TypeID = 127

	// TypeIDUserStart is the start of the user-defined type range.
	TypeIDUserStart TypeID = 128
)

// IsBuiltin returns true if the TypeID is in the built-in range.
func (id TypeID) IsBuiltin() bool {
	return id >= TypeIDBuiltinStart && id <= TypeIDBuiltinEnd
}

// IsStdlib returns true if the TypeID is in the standard library range.
func (id TypeID) IsStdlib() bool {
	return id >= TypeIDStdlibStart && id <= TypeIDStdlibEnd
}

// IsUser returns true if the TypeID is in the user-defined range.
func (id TypeID) IsUser() bool {
	return id >= TypeIDUserStart
}

// IsNil returns true if the TypeID represents nil.
func (id TypeID) IsNil() bool {
	return id == TypeIDNil
}

// IsValid returns true if the TypeID is valid (not nil and > 0).
func (id TypeID) IsValid() bool {
	return id > TypeIDNil
}

// String returns a string representation of the TypeID.
func (id TypeID) String() string {
	return fmt.Sprintf("%d", id)
}

// WireType indicates how a value is encoded on the wire.
// Re-exported from internal/wire for public use.
type WireType uint8

// Wire type constants.
const (
	// WireVarint is used for unsigned integers, booleans, and enums.
	WireVarint WireType = 0

	// WireFixed64 is used for fixed 64-bit values (uint64, int64, float64).
	WireFixed64 WireType = 1

	// WireBytes is used for length-prefixed data (strings, bytes, messages).
	WireBytes WireType = 2

	// WireFixed32 is used for fixed 32-bit values (uint32, int32, float32).
	WireFixed32 WireType = 5

	// WireSVarint is used for signed integers with ZigZag encoding.
	WireSVarint WireType = 6

	// WireTypeRef is used for polymorphic type references.
	WireTypeRef WireType = 7
)

// String returns a human-readable name for the wire type.
func (w WireType) String() string {
	switch w {
	case WireVarint:
		return "Varint"
	case WireFixed64:
		return "Fixed64"
	case WireBytes:
		return "Bytes"
	case WireFixed32:
		return "Fixed32"
	case WireSVarint:
		return "SVarint"
	case WireTypeRef:
		return "TypeRef"
	default:
		return "Unknown"
	}
}

// IsValid returns true if the wire type is known.
func (w WireType) IsValid() bool {
	switch w {
	case WireVarint, WireFixed64, WireBytes, WireFixed32, WireSVarint, WireTypeRef:
		return true
	default:
		return false
	}
}

// Limits defines resource limits for encoding/decoding.
type Limits struct {
	// MaxMessageSize is the maximum total message size in bytes.
	// A value of 0 means no limit.
	MaxMessageSize int64

	// MaxDepth is the maximum nesting depth for structs/slices/maps.
	// A value of 0 means no limit.
	MaxDepth int

	// MaxStringLength is the maximum length of a string in bytes.
	// A value of 0 means no limit.
	MaxStringLength int

	// MaxBytesLength is the maximum length of a byte slice.
	// A value of 0 means no limit.
	MaxBytesLength int

	// MaxArrayLength is the maximum number of elements in a slice/array.
	// A value of 0 means no limit.
	MaxArrayLength int

	// MaxMapSize is the maximum number of entries in a map.
	// A value of 0 means no limit.
	MaxMapSize int
}

// DefaultLimits are the default resource limits.
// These are generous limits suitable for most use cases.
var DefaultLimits = Limits{
	MaxMessageSize:  64 * 1024 * 1024,  // 64 MB
	MaxDepth:        100,
	MaxStringLength: 10 * 1024 * 1024,  // 10 MB
	MaxBytesLength:  100 * 1024 * 1024, // 100 MB
	MaxArrayLength:  1_000_000,
	MaxMapSize:      1_000_000,
}

// SecureLimits are conservative limits for untrusted input.
var SecureLimits = Limits{
	MaxMessageSize:  1 * 1024 * 1024, // 1 MB
	MaxDepth:        32,
	MaxStringLength: 1 * 1024 * 1024, // 1 MB
	MaxBytesLength:  10 * 1024 * 1024, // 10 MB
	MaxArrayLength:  10_000,
	MaxMapSize:      10_000,
}

// NoLimits disables all resource limits.
// Use with caution - only for trusted input.
var NoLimits = Limits{}

// WireVersion specifies the wire format version.
type WireVersion int

const (
	// WireVersionV1 is the original wire format with field count prefix.
	// Deprecated: Use WireVersionV2 for new code.
	WireVersionV1 WireVersion = 1

	// WireVersionV2 is the optimized wire format with compact tags and end markers.
	// Features:
	//   - Single-byte tags for fields 1-15
	//   - End marker instead of field count prefix
	//   - Packed repeated primitives
	//   - Optional deterministic map ordering
	WireVersionV2 WireVersion = 2
)

// Options configures encoding/decoding behavior.
type Options struct {
	// Limits specifies resource limits.
	Limits Limits

	// WireVersion specifies the wire format version.
	// Default is WireVersionV2 for optimal performance.
	WireVersion WireVersion

	// StrictMode rejects unknown fields during decoding.
	StrictMode bool

	// ValidateUTF8 validates that strings are valid UTF-8.
	ValidateUTF8 bool

	// OmitEmpty omits zero-value fields during encoding.
	// This is the default behavior.
	OmitEmpty bool

	// PresenceBitmap uses a presence bitmap for tracking field presence.
	// This allows distinguishing zero values from absent values.
	PresenceBitmap bool

	// Deterministic ensures deterministic output by sorting map keys.
	// This is enabled by default for reproducible encoding.
	// Disable for better performance when determinism is not required.
	Deterministic bool
}

// DefaultOptions are the default encoding/decoding options.
var DefaultOptions = Options{
	Limits:        DefaultLimits,
	WireVersion:   WireVersionV2,
	StrictMode:    false,
	ValidateUTF8:  true,
	OmitEmpty:     true,
	Deterministic: true,
}

// SecureOptions are conservative options for untrusted input.
var SecureOptions = Options{
	Limits:        SecureLimits,
	WireVersion:   WireVersionV2,
	StrictMode:    false,
	ValidateUTF8:  true,
	OmitEmpty:     true,
	Deterministic: true,
}

// StrictOptions reject unknown fields and validate strings.
var StrictOptions = Options{
	Limits:        DefaultLimits,
	WireVersion:   WireVersionV2,
	StrictMode:    true,
	ValidateUTF8:  true,
	OmitEmpty:     true,
	Deterministic: true,
}

// FastOptions prioritize performance over determinism.
// Use when decoding output from the same encoder (map order doesn't matter).
var FastOptions = Options{
	Limits:        DefaultLimits,
	WireVersion:   WireVersionV2,
	StrictMode:    false,
	ValidateUTF8:  false,
	OmitEmpty:     true,
	Deterministic: false,
}

// V1Options use the legacy V1 wire format for compatibility.
// Deprecated: Only use for interop with old encoded data.
var V1Options = Options{
	Limits:        DefaultLimits,
	WireVersion:   WireVersionV1,
	StrictMode:    false,
	ValidateUTF8:  true,
	OmitEmpty:     true,
	Deterministic: true,
}

// Version information, set by ldflags at build time.
var (
	// Version is the semantic version of the library.
	Version = "dev"

	// GitCommit is the git commit hash.
	GitCommit = "unknown"

	// BuildDate is the build timestamp.
	BuildDate = "unknown"
)

// VersionInfo returns a formatted version string.
func VersionInfo() string {
	return Version + " (" + GitCommit + ", " + BuildDate + ")"
}

// Size constants for primitive types.
const (
	// BoolSize is the maximum encoded size of a bool (1 byte varint).
	BoolSize = 1

	// Fixed32Size is the encoded size of a fixed 32-bit value.
	Fixed32Size = 4

	// Fixed64Size is the encoded size of a fixed 64-bit value.
	Fixed64Size = 8

	// Float32Size is the encoded size of a float32.
	Float32Size = 4

	// Float64Size is the encoded size of a float64.
	Float64Size = 8

	// Complex64Size is the encoded size of a complex64 (two float32).
	Complex64Size = 8

	// Complex128Size is the encoded size of a complex128 (two float64).
	Complex128Size = 16

	// MaxVarintLen64 is the maximum encoded size of a varint64.
	MaxVarintLen64 = 10

	// MaxTagSize is the maximum encoded size of a field tag.
	MaxTagSize = MaxVarintLen64
)
