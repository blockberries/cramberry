package cramberry

import (
	"github.com/blockberries/cramberry/internal/wire"
)

// Wire format
//
// A message is a sequence of (tag, value) pairs terminated by a single 0x00
// end marker. Tags use one of two encodings:
//
//	Fields 1-15:  [fieldNum:4][wireType:3][0:1]  — single byte
//	Fields 16+:   [0:4][wireType:3][1:1] followed by varint fieldNum
//	End marker:   0x00 (fieldNum=0, wireType=0, extended=0)

const (
	// EndMarker signals the end of a struct's fields.
	EndMarker byte = 0x00

	// tagExtendedBit indicates the field number is in the following varint.
	tagExtendedBit byte = 0x01

	// tagWireTypeMask extracts the wire type from a tag byte.
	tagWireTypeMask byte = 0x0E

	// tagWireTypeShift is the bit shift for wire type in a tag byte.
	tagWireTypeShift = 1

	// tagFieldNumShift is the bit shift for field number in a compact tag byte.
	tagFieldNumShift = 4

	// maxCompactFieldNum is the maximum field number that fits in a compact tag.
	maxCompactFieldNum = 15
)

// Wire types.
const (
	// WireVarint is for unsigned integers, bools, and enums (LEB128).
	WireVarint byte = 0

	// WireFixed64 is for fixed 64-bit values (float64, fixed64).
	WireFixed64 byte = 1

	// WireBytes is for length-prefixed payloads (string, bytes, message,
	// packed array).
	WireBytes byte = 2

	// WireFixed32 is for fixed 32-bit values (float32, fixed32).
	WireFixed32 byte = 3

	// WireSVarint is for signed integers (ZigZag-LEB128).
	WireSVarint byte = 4
)

// EncodeTag encodes a field tag and returns the encoded bytes.
func EncodeTag(fieldNum int, wireType byte) []byte {
	if fieldNum <= 0 {
		return nil
	}

	if fieldNum <= maxCompactFieldNum {
		// Compact format: single byte
		tag := byte(fieldNum<<tagFieldNumShift) | (wireType << tagWireTypeShift)
		return []byte{tag}
	}

	// Extended format: marker byte + varint field number
	marker := (wireType << tagWireTypeShift) | tagExtendedBit
	result := []byte{marker}
	for fieldNum >= 0x80 {
		result = append(result, byte(fieldNum)|0x80)
		fieldNum >>= 7
	}
	result = append(result, byte(fieldNum))
	return result
}

// DecodeTag decodes a field tag from data.
// Returns fieldNum (0 for end marker), wireType, and bytes consumed.
func DecodeTag(data []byte) (fieldNum int, wireType byte, n int) {
	if len(data) == 0 {
		return 0, 0, 0
	}

	tag := data[0]

	if tag == EndMarker {
		return 0, 0, 1
	}

	wireType = (tag & tagWireTypeMask) >> tagWireTypeShift

	if tag&tagExtendedBit == 0 {
		// Compact format: field number in upper 4 bits
		fieldNum = int(tag >> tagFieldNumShift)
		return fieldNum, wireType, 1
	}

	// Extended format: field number in following varint
	if len(data) < 2 {
		return 0, 0, 0 // Need more data
	}

	var fn uint64
	var shift uint
	n = 1
	for i := 1; i < len(data) && i <= wire.MaxVarintLen64; i++ {
		b := data[i]

		// At the 10th varint byte (index 10 in data), check for overflow.
		// We've consumed 63 bits (9 bytes * 7 bits); only 1 more bit allowed.
		if i == 10 {
			if b >= 0x80 {
				return 0, 0, 0 // Varint too long
			}
			if b > 1 {
				return 0, 0, 0 // Would overflow uint64
			}
		}

		fn |= uint64(b&0x7F) << shift
		n++
		if b < 0x80 {
			// Canonical-encoding check: a multi-byte field-number varint
			// whose terminating byte is zero could have been encoded in
			// one fewer byte. (i==1 is the first varint byte; i>1 means
			// at least two bytes have been consumed.)
			if i > 1 && b == 0 {
				return 0, 0, 0
			}
			if fn > uint64(wire.MaxFieldNumber) {
				return 0, 0, 0 // Field number too large
			}
			return int(fn), wireType, n
		}
		shift += 7
	}

	// Reached max length without termination
	return 0, 0, 0
}

// TagSize returns the encoded size of a tag.
func TagSize(fieldNum int) int {
	if fieldNum <= maxCompactFieldNum {
		return 1
	}
	// 1 byte marker + varint size
	size := 1
	for fieldNum >= 0x80 {
		size++
		fieldNum >>= 7
	}
	return size + 1
}

// WriteTag writes a field tag to the writer.
func (w *Writer) WriteTag(fieldNum int, wireType byte) {
	if !w.checkWrite() {
		return
	}
	if fieldNum <= 0 {
		w.setError(ErrInvalidFieldNumber)
		return
	}

	if fieldNum <= maxCompactFieldNum {
		// Compact format: single byte
		tag := byte(fieldNum<<tagFieldNumShift) | (wireType << tagWireTypeShift)
		w.grow(1)
		w.buf = append(w.buf, tag)
		return
	}

	// Extended format: marker byte + varint field number
	marker := (wireType << tagWireTypeShift) | tagExtendedBit
	w.grow(1 + MaxVarintLen64)
	w.buf = append(w.buf, marker)

	// Write varint field number
	fn := uint64(fieldNum)
	for fn >= 0x80 {
		w.buf = append(w.buf, byte(fn)|0x80)
		fn >>= 7
	}
	w.buf = append(w.buf, byte(fn))
}

// WriteEndMarker writes the struct end marker.
func (w *Writer) WriteEndMarker() {
	if !w.checkWrite() {
		return
	}
	w.grow(1)
	w.buf = append(w.buf, EndMarker)
}

// ReadTag reads a field tag from the reader.
// Returns fieldNum=0 for end marker or on error.
func (r *Reader) ReadTag() (fieldNum int, wireType byte) {
	if r.err != nil || r.pos >= len(r.data) {
		return 0, 0
	}

	tag := r.data[r.pos]
	r.pos++

	if tag == EndMarker {
		return 0, 0
	}

	wireType = (tag & tagWireTypeMask) >> tagWireTypeShift

	if tag&tagExtendedBit == 0 {
		// Compact format
		fieldNum = int(tag >> tagFieldNumShift)
		return fieldNum, wireType
	}

	// Extended format: read varint field number with safety limits
	var fn uint64
	var shift uint
	for i := 0; i < wire.MaxVarintLen64 && r.pos < len(r.data); i++ {
		b := r.data[r.pos]
		r.pos++

		// At the 10th byte (index 9), check for overflow.
		// We've consumed 63 bits; only 1 more bit allowed.
		if i == 9 {
			if b >= 0x80 {
				r.setError(ErrInvalidVarint)
				return 0, 0
			}
			if b > 1 {
				r.setError(ErrOverflow)
				return 0, 0
			}
		}

		fn |= uint64(b&0x7F) << shift
		if b < 0x80 {
			// Canonical-encoding check: a multi-byte field-number
			// varint whose terminating byte is zero could have been
			// encoded in one fewer byte (i==0 is single-byte path, so
			// i>0 means we've read at least two bytes).
			if i > 0 && b == 0 {
				r.setError(wire.ErrVarintNonCanonical)
				return 0, 0
			}
			if fn > uint64(wire.MaxFieldNumber) {
				r.setError(ErrOverflow)
				return 0, 0
			}
			return int(fn), wireType
		}
		shift += 7
	}

	// Reached max length without termination or ran out of data
	r.setError(ErrInvalidVarint)
	return 0, 0
}

// SkipValue skips a value based on its wire type.
func (r *Reader) SkipValue(wireType byte) {
	if r.err != nil {
		return
	}

	switch wireType {
	case WireVarint, WireSVarint:
		// Skip varint with max length limit
		for i := 0; i < wire.MaxVarintLen64 && r.pos < len(r.data); i++ {
			b := r.data[r.pos]
			r.pos++
			if b < 0x80 {
				return
			}
		}
		// Either ran out of data or exceeded max varint length
		r.setError(ErrInvalidVarint)

	case WireFixed32:
		if !r.ensure(4) {
			return
		}
		r.pos += 4

	case WireFixed64:
		if !r.ensure(8) {
			return
		}
		r.pos += 8

	case WireBytes:
		// Read length, then skip that many bytes
		length := r.ReadUvarint()
		if r.err != nil {
			return
		}
		// Check for overflow before converting to int
		remaining := len(r.data) - r.pos
		if length > uint64(remaining) {
			r.setError(ErrUnexpectedEOF)
			return
		}
		r.pos += int(length)

	default:
		r.setError(NewDecodeError("unknown wire type", nil))
	}

	if r.pos > len(r.data) {
		r.pos = len(r.data)
		r.setError(ErrUnexpectedEOF)
	}
}
