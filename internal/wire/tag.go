package wire

import "errors"

// WireType indicates how a value is encoded on the wire.
type WireType uint8

const (
	// WireVarint is used for unsigned integers, booleans, and enums.
	// Values are encoded as variable-length integers (LEB128).
	WireVarint WireType = 0

	// WireFixed64 is used for fixed 64-bit values (uint64, int64, float64).
	// Values are encoded as 8 bytes in little-endian order.
	WireFixed64 WireType = 1

	// WireBytes is used for length-prefixed data: strings, byte slices,
	// embedded messages, and packed repeated fields.
	// Format: [length: varint] [data: length bytes]
	WireBytes WireType = 2

	// WireFixed32 is used for fixed 32-bit values (uint32, int32, float32).
	// Values are encoded as 4 bytes in little-endian order.
	WireFixed32 WireType = 5

	// WireSVarint is used for signed integers with ZigZag encoding.
	// This ensures small absolute values use fewer bytes regardless of sign.
	WireSVarint WireType = 6

	// WireTypeRef is used for polymorphic types (interface values).
	// Format: [type_id: varint] [value_length: varint] [value: bytes]
	WireTypeRef WireType = 7
)

// Note on wire type values:
// Wire types 3 and 4 are intentionally skipped and reserved. In Protocol Buffers,
// these values were used for the deprecated "start group" (3) and "end group" (4)
// wire types. Cramberry skips these values to maintain partial compatibility with
// protobuf tooling and to reserve them for potential future use. Any data encoded
// with wire types 3 or 4 should be treated as invalid.

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

// IsValid returns true if the wire type is a known type.
func (w WireType) IsValid() bool {
	switch w {
	case WireVarint, WireFixed64, WireBytes, WireFixed32, WireSVarint, WireTypeRef:
		return true
	default:
		return false
	}
}

// Errors for tag decoding.
var (
	// ErrInvalidWireType indicates an unknown or invalid wire type.
	ErrInvalidWireType = errors.New("cramberry: invalid wire type")

	// ErrInvalidFieldNumber indicates an invalid field number (must be > 0).
	ErrInvalidFieldNumber = errors.New("cramberry: invalid field number")
)

// Tag represents a field tag combining field number and wire type.
// The tag is encoded as a varint: (field_number << 3) | wire_type
type Tag uint64

// NewTag creates a new tag from a field number and wire type.
func NewTag(fieldNum int, wireType WireType) Tag {
	return Tag(uint64(fieldNum)<<3 | uint64(wireType))
}

// FieldNumber returns the field number from the tag.
func (t Tag) FieldNumber() int {
	return int(t >> 3)
}

// WireType returns the wire type from the tag.
func (t Tag) WireType() WireType {
	return WireType(t & 0x7)
}

// AppendTag appends a field tag to buf and returns the extended buffer.
// The tag is encoded as: (field_number << 3) | wire_type
func AppendTag(buf []byte, fieldNum int, wireType WireType) []byte {
	tag := uint64(fieldNum)<<3 | uint64(wireType)
	return AppendUvarint(buf, tag)
}

// DecodeTag decodes a field tag from data.
// Returns the field number, wire type, bytes consumed, and any error.
//
// A field number of 0 is invalid and will return an error.
// Unknown wire types will return ErrInvalidWireType.
func DecodeTag(data []byte) (fieldNum int, wireType WireType, n int, err error) {
	tag, n, err := DecodeUvarint(data)
	if err != nil {
		return 0, 0, 0, err
	}

	fieldNum = int(tag >> 3)
	wireType = WireType(tag & 0x7)

	// Validate field number (must be positive)
	if fieldNum <= 0 {
		return 0, 0, 0, ErrInvalidFieldNumber
	}

	// Validate wire type
	if !wireType.IsValid() {
		return 0, 0, n, ErrInvalidWireType
	}

	return fieldNum, wireType, n, nil
}

// DecodeTagRelaxed decodes a field tag without validating the wire type.
// This is useful for forward compatibility when new wire types may be added.
func DecodeTagRelaxed(data []byte) (fieldNum int, wireType WireType, n int, err error) {
	tag, n, err := DecodeUvarint(data)
	if err != nil {
		return 0, 0, 0, err
	}

	fieldNum = int(tag >> 3)
	wireType = WireType(tag & 0x7)

	if fieldNum <= 0 {
		return 0, 0, 0, ErrInvalidFieldNumber
	}

	return fieldNum, wireType, n, nil
}

// TagSize returns the number of bytes required to encode a tag.
func TagSize(fieldNum int) int {
	// Tag is encoded as (fieldNum << 3) | wireType
	// The wire type adds at most 7 to the value, which doesn't change the size
	// because we're shifting by 3 bits.
	return UvarintSize(uint64(fieldNum) << 3)
}

// PutTag encodes a tag into buf and returns the number of bytes written.
// The buffer must be large enough (use TagSize to determine).
func PutTag(buf []byte, fieldNum int, wireType WireType) int {
	tag := uint64(fieldNum)<<3 | uint64(wireType)
	return PutUvarint(buf, tag)
}

// MaxFieldNumber is the maximum allowed field number.
// Field numbers are encoded as part of a varint, so technically they can be
// very large, but we impose a practical limit for safety.
const MaxFieldNumber = 1<<29 - 1 // ~536 million

// ValidateFieldNumber returns an error if the field number is invalid.
func ValidateFieldNumber(fieldNum int) error {
	if fieldNum <= 0 {
		return ErrInvalidFieldNumber
	}
	if fieldNum > MaxFieldNumber {
		return ErrInvalidFieldNumber
	}
	return nil
}

// WireTypeForKind returns the typical wire type for a Go kind.
// This is a helper for reflection-based encoding.
func WireTypeForKind(kind string) WireType {
	switch kind {
	case "bool", "uint8", "uint16", "uint32", "uint64", "uint", "uintptr":
		return WireVarint
	case "int8", "int16", "int32", "int64", "int":
		return WireSVarint
	case "float32":
		return WireFixed32
	case "float64":
		return WireFixed64
	case "string", "slice", "array", "map", "struct", "ptr":
		return WireBytes
	case "interface":
		return WireTypeRef
	default:
		return WireBytes
	}
}
