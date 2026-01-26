package wire

import (
	"encoding/binary"
	"math"
)

// Canonical NaN bit patterns for deterministic encoding.
// We use quiet NaN with no payload (all significand bits zero except the quiet bit).
const (
	// canonicalNaN32 is the canonical 32-bit quiet NaN: 0x7FC00000
	// Sign=0, Exponent=0xFF (all 1s), Quiet bit=1, Significand=0
	canonicalNaN32 = 0x7FC00000

	// canonicalNaN64 is the canonical 64-bit quiet NaN: 0x7FF8000000000000
	// Sign=0, Exponent=0x7FF (all 1s), Quiet bit=1, Significand=0
	canonicalNaN64 = 0x7FF8000000000000
)

// AppendFixed32 appends a 32-bit value in little-endian format.
func AppendFixed32(buf []byte, v uint32) []byte {
	return append(buf,
		byte(v),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
	)
}

// AppendFixed64 appends a 64-bit value in little-endian format.
func AppendFixed64(buf []byte, v uint64) []byte {
	return append(buf,
		byte(v),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
		byte(v>>32),
		byte(v>>40),
		byte(v>>48),
		byte(v>>56),
	)
}

// DecodeFixed32 decodes a little-endian 32-bit value.
// Returns the value and an error if the input is too short.
func DecodeFixed32(data []byte) (uint32, error) {
	if len(data) < 4 {
		return 0, ErrVarintTruncated // Reuse error, conceptually "data truncated"
	}
	return binary.LittleEndian.Uint32(data), nil
}

// DecodeFixed64 decodes a little-endian 64-bit value.
// Returns the value and an error if the input is too short.
func DecodeFixed64(data []byte) (uint64, error) {
	if len(data) < 8 {
		return 0, ErrVarintTruncated
	}
	return binary.LittleEndian.Uint64(data), nil
}

// PutFixed32 writes a 32-bit value to buf in little-endian format.
// The buffer must have at least 4 bytes available.
func PutFixed32(buf []byte, v uint32) {
	buf[0] = byte(v)
	buf[1] = byte(v >> 8)
	buf[2] = byte(v >> 16)
	buf[3] = byte(v >> 24)
}

// PutFixed64 writes a 64-bit value to buf in little-endian format.
// The buffer must have at least 8 bytes available.
func PutFixed64(buf []byte, v uint64) {
	buf[0] = byte(v)
	buf[1] = byte(v >> 8)
	buf[2] = byte(v >> 16)
	buf[3] = byte(v >> 24)
	buf[4] = byte(v >> 32)
	buf[5] = byte(v >> 40)
	buf[6] = byte(v >> 48)
	buf[7] = byte(v >> 56)
}

// Float32 encoding with canonicalization for deterministic output.

// AppendFloat32 appends a float32 in canonicalized little-endian format.
//
// Canonicalization rules:
//   - Negative zero (-0.0) is converted to positive zero (+0.0)
//   - All NaN values are converted to canonical quiet NaN (0x7FC00000)
//   - All other values (including +Inf, -Inf, subnormals) are preserved
func AppendFloat32(buf []byte, v float32) []byte {
	bits := canonicalizeFloat32(v)
	return AppendFixed32(buf, bits)
}

// DecodeFloat32 decodes a canonicalized float32 from little-endian bytes.
func DecodeFloat32(data []byte) (float32, error) {
	bits, err := DecodeFixed32(data)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(bits), nil
}

// PutFloat32 writes a canonicalized float32 to buf in little-endian format.
func PutFloat32(buf []byte, v float32) {
	bits := canonicalizeFloat32(v)
	PutFixed32(buf, bits)
}

// canonicalizeFloat32 returns the canonical bit representation of a float32.
func canonicalizeFloat32(v float32) uint32 {
	bits := math.Float32bits(v)

	// Check for NaN: exponent all 1s and significand non-zero
	if bits&0x7F800000 == 0x7F800000 && bits&0x007FFFFF != 0 {
		return canonicalNaN32
	}

	// Check for negative zero
	if bits == 0x80000000 {
		return 0
	}

	return bits
}

// Float64 encoding with canonicalization for deterministic output.

// AppendFloat64 appends a float64 in canonicalized little-endian format.
//
// Canonicalization rules:
//   - Negative zero (-0.0) is converted to positive zero (+0.0)
//   - All NaN values are converted to canonical quiet NaN (0x7FF8000000000000)
//   - All other values (including +Inf, -Inf, subnormals) are preserved
func AppendFloat64(buf []byte, v float64) []byte {
	bits := canonicalizeFloat64(v)
	return AppendFixed64(buf, bits)
}

// DecodeFloat64 decodes a canonicalized float64 from little-endian bytes.
func DecodeFloat64(data []byte) (float64, error) {
	bits, err := DecodeFixed64(data)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(bits), nil
}

// PutFloat64 writes a canonicalized float64 to buf in little-endian format.
func PutFloat64(buf []byte, v float64) {
	bits := canonicalizeFloat64(v)
	PutFixed64(buf, bits)
}

// canonicalizeFloat64 returns the canonical bit representation of a float64.
func canonicalizeFloat64(v float64) uint64 {
	bits := math.Float64bits(v)

	// Check for NaN: exponent all 1s and significand non-zero
	if bits&0x7FF0000000000000 == 0x7FF0000000000000 && bits&0x000FFFFFFFFFFFFF != 0 {
		return canonicalNaN64
	}

	// Check for negative zero
	if bits == 0x8000000000000000 {
		return 0
	}

	return bits
}

// Complex number encoding.
// Complex numbers are encoded as two consecutive floats: real part, then imaginary part.

// AppendComplex64 appends a complex64 as two canonicalized float32 values.
func AppendComplex64(buf []byte, v complex64) []byte {
	buf = AppendFloat32(buf, real(v))
	buf = AppendFloat32(buf, imag(v))
	return buf
}

// DecodeComplex64 decodes a complex64 from 8 bytes (two float32 values).
func DecodeComplex64(data []byte) (complex64, error) {
	if len(data) < 8 {
		return 0, ErrVarintTruncated
	}
	r, _ := DecodeFloat32(data[0:4])
	i, _ := DecodeFloat32(data[4:8])
	return complex(r, i), nil
}

// AppendComplex128 appends a complex128 as two canonicalized float64 values.
func AppendComplex128(buf []byte, v complex128) []byte {
	buf = AppendFloat64(buf, real(v))
	buf = AppendFloat64(buf, imag(v))
	return buf
}

// DecodeComplex128 decodes a complex128 from 16 bytes (two float64 values).
func DecodeComplex128(data []byte) (complex128, error) {
	if len(data) < 16 {
		return 0, ErrVarintTruncated
	}
	r, _ := DecodeFloat64(data[0:8])
	i, _ := DecodeFloat64(data[8:16])
	return complex(r, i), nil
}

// Size constants for fixed-width types.
const (
	Fixed32Size    = 4
	Fixed64Size    = 8
	Float32Size    = 4
	Float64Size    = 8
	Complex64Size  = 8
	Complex128Size = 16
)

// IsNaN32 returns true if the float32 is any NaN value.
func IsNaN32(v float32) bool {
	bits := math.Float32bits(v)
	return bits&0x7F800000 == 0x7F800000 && bits&0x007FFFFF != 0
}

// IsNaN64 returns true if the float64 is any NaN value.
func IsNaN64(v float64) bool {
	bits := math.Float64bits(v)
	return bits&0x7FF0000000000000 == 0x7FF0000000000000 && bits&0x000FFFFFFFFFFFFF != 0
}

// IsNegativeZero32 returns true if the float32 is negative zero.
func IsNegativeZero32(v float32) bool {
	return math.Float32bits(v) == 0x80000000
}

// IsNegativeZero64 returns true if the float64 is negative zero.
func IsNegativeZero64(v float64) bool {
	return math.Float64bits(v) == 0x8000000000000000
}

// CanonicalFloat32Bits returns the canonical bit representation of a float32.
// NaN values are converted to the canonical NaN (0x7FC00000).
// Negative zero is converted to positive zero.
func CanonicalFloat32Bits(v float32) uint32 {
	return canonicalizeFloat32(v)
}

// CanonicalFloat64Bits returns the canonical bit representation of a float64.
// NaN values are converted to the canonical NaN (0x7FF8000000000000).
// Negative zero is converted to positive zero.
func CanonicalFloat64Bits(v float64) uint64 {
	return canonicalizeFloat64(v)
}
