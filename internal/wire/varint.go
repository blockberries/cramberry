// Package wire provides low-level encoding primitives for the Cramberry wire format.
package wire

import "errors"

// Maximum number of bytes for a varint-encoded uint64.
// A uint64 has 64 bits, and each varint byte encodes 7 bits,
// so we need ceil(64/7) = 10 bytes maximum.
const MaxVarintLen64 = 10

// Errors for varint decoding.
var (
	// ErrVarintOverflow indicates the varint overflows a 64-bit integer.
	ErrVarintOverflow = errors.New("cramberry: varint overflows uint64")

	// ErrVarintTruncated indicates the input data was truncated.
	ErrVarintTruncated = errors.New("cramberry: varint truncated")

	// ErrVarintTooLong indicates the varint encoding exceeds maximum length.
	ErrVarintTooLong = errors.New("cramberry: varint exceeds maximum length")
)

// AppendUvarint appends the varint encoding of v to buf and returns the extended buffer.
//
// The encoding uses 7 bits per byte, with the MSB as a continuation flag.
// Bytes are ordered from least significant to most significant (little-endian varint).
//
// Example encodings:
//   - 0 → [0x00]
//   - 1 → [0x01]
//   - 127 → [0x7f]
//   - 128 → [0x80, 0x01]
//   - 300 → [0xac, 0x02]
func AppendUvarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

// AppendSvarint appends the zigzag-encoded varint of v to buf and returns the extended buffer.
//
// ZigZag encoding maps signed integers to unsigned integers so that numbers with
// small absolute values have small varint encodings:
//
//	0 → 0, -1 → 1, 1 → 2, -2 → 3, 2 → 4, ...
//
// This ensures that -1 encodes to 1 byte, not 10 bytes.
func AppendSvarint(buf []byte, v int64) []byte {
	// ZigZag encoding: (v << 1) ^ (v >> 63)
	// The arithmetic right shift (v >> 63) produces all 1s for negative, all 0s for positive.
	// XOR flips all bits for negative numbers after the left shift.
	uv := uint64(v<<1) ^ uint64(v>>63)
	return AppendUvarint(buf, uv)
}

// DecodeUvarint decodes a varint from data and returns the value and the number of bytes consumed.
// If the data is truncated or the varint overflows, an error is returned.
//
// The function is optimized for the common case of small values (1-2 bytes).
func DecodeUvarint(data []byte) (uint64, int, error) {
	if len(data) == 0 {
		return 0, 0, ErrVarintTruncated
	}

	// Fast path for single-byte varints (values 0-127)
	if data[0] < 0x80 {
		return uint64(data[0]), 1, nil
	}

	// General case: multi-byte varint
	var v uint64
	var shift uint

	for i := 0; i < len(data); i++ {
		if i >= MaxVarintLen64 {
			return 0, 0, ErrVarintTooLong
		}

		b := data[i]
		// At the 10th byte (index 9), we've already consumed 63 bits.
		// The 10th byte can only contribute 1 more bit (bit 63 of uint64).
		if i == 9 {
			// If continuation bit is set, we'd need 11+ bytes
			if b >= 0x80 {
				return 0, 0, ErrVarintTooLong
			}
			// If data portion is > 1, value would overflow uint64
			if b > 1 {
				return 0, 0, ErrVarintOverflow
			}
		}

		v |= uint64(b&0x7f) << shift

		if b < 0x80 {
			// This is the last byte
			return v, i + 1, nil
		}

		shift += 7
	}

	// Reached end of data without finding a terminating byte
	return 0, 0, ErrVarintTruncated
}

// DecodeSvarint decodes a zigzag-encoded signed varint from data.
// Returns the value and the number of bytes consumed, or an error.
func DecodeSvarint(data []byte) (int64, int, error) {
	uv, n, err := DecodeUvarint(data)
	if err != nil {
		return 0, n, err
	}
	// Unzigzag: (uv >> 1) ^ -(uv & 1)
	// If LSB is 0: result is uv >> 1 (positive number)
	// If LSB is 1: result is (uv >> 1) ^ -1 = -(uv >> 1) - 1 (negative number)
	return int64(uv>>1) ^ -int64(uv&1), n, nil
}

// UvarintSize returns the number of bytes required to encode v as a varint.
//
// This is useful for pre-allocating buffers.
func UvarintSize(v uint64) int {
	// Count the number of 7-bit groups needed
	switch {
	case v < 1<<7:
		return 1
	case v < 1<<14:
		return 2
	case v < 1<<21:
		return 3
	case v < 1<<28:
		return 4
	case v < 1<<35:
		return 5
	case v < 1<<42:
		return 6
	case v < 1<<49:
		return 7
	case v < 1<<56:
		return 8
	case v < 1<<63:
		return 9
	default:
		return 10
	}
}

// SvarintSize returns the number of bytes required to encode v as a zigzag varint.
func SvarintSize(v int64) int {
	// Apply zigzag encoding and get the size
	uv := uint64(v<<1) ^ uint64(v>>63)
	return UvarintSize(uv)
}

// PutUvarint encodes v into buf and returns the number of bytes written.
// The buffer must be large enough to hold the encoded value; use UvarintSize
// to determine the required size.
//
// This is a lower-level function than AppendUvarint, useful when the buffer
// is already allocated.
func PutUvarint(buf []byte, v uint64) int {
	i := 0
	for v >= 0x80 {
		buf[i] = byte(v) | 0x80
		v >>= 7
		i++
	}
	buf[i] = byte(v)
	return i + 1
}

// PutSvarint encodes v into buf using zigzag encoding and returns bytes written.
func PutSvarint(buf []byte, v int64) int {
	uv := uint64(v<<1) ^ uint64(v>>63)
	return PutUvarint(buf, uv)
}
