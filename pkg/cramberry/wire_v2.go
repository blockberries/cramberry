package cramberry

// Wire Format V2 - Optimized encoding format
//
// Key changes from V1:
// 1. End marker instead of field count prefix
// 2. Compact tags for fields 1-15 (single byte)
// 3. Packed repeated primitives
// 4. Optional deterministic mode (map sorting opt-in)
//
// Tag encoding:
//   Fields 1-15:  [fieldNum:4][wireType:3][0:1] = single byte
//   Fields 16+:   [0:4][wireType:3][1:1] followed by varint fieldNum
//   End marker:   0x00 (fieldNum=0, wireType=0, extended=0)

const (
	// EndMarker signals the end of a struct's fields
	EndMarker byte = 0x00

	// tagExtendedBit indicates the field number is in the following varint
	tagExtendedBit byte = 0x01

	// tagWireTypeMask extracts the wire type from a compact tag
	tagWireTypeMask byte = 0x0E

	// tagWireTypeShift is the bit shift for wire type in compact tag
	tagWireTypeShift = 1

	// tagFieldNumShift is the bit shift for field number in compact tag
	tagFieldNumShift = 4

	// maxCompactFieldNum is the maximum field number that fits in a compact tag
	maxCompactFieldNum = 15
)

// V2 Wire Types (simplified from V1)
const (
	// WireTypeV2Varint is for integers, bools, enums (unsigned varint)
	WireTypeV2Varint byte = 0

	// WireTypeV2Fixed64 is for fixed 64-bit values (float64, fixed64)
	WireTypeV2Fixed64 byte = 1

	// WireTypeV2Bytes is for length-prefixed data (string, bytes, messages, packed arrays)
	WireTypeV2Bytes byte = 2

	// WireTypeV2Fixed32 is for fixed 32-bit values (float32, fixed32)
	WireTypeV2Fixed32 byte = 3

	// WireTypeV2SVarint is for signed integers (zigzag encoded)
	WireTypeV2SVarint byte = 4
)

// EncodeCompactTag encodes a field tag in compact format.
// Returns the encoded bytes.
func EncodeCompactTag(fieldNum int, wireType byte) []byte {
	if fieldNum <= 0 {
		return nil // Invalid field number
	}

	if fieldNum <= maxCompactFieldNum {
		// Compact format: single byte
		tag := byte(fieldNum<<tagFieldNumShift) | (wireType << tagWireTypeShift)
		return []byte{tag}
	}

	// Extended format: marker byte + varint field number
	marker := (wireType << tagWireTypeShift) | tagExtendedBit
	result := []byte{marker}
	// Append varint-encoded field number
	for fieldNum >= 0x80 {
		result = append(result, byte(fieldNum)|0x80)
		fieldNum >>= 7
	}
	result = append(result, byte(fieldNum))
	return result
}

// DecodeCompactTag decodes a field tag from the compact format.
// Returns fieldNum (0 for end marker), wireType, and bytes consumed.
func DecodeCompactTag(data []byte) (fieldNum int, wireType byte, n int) {
	if len(data) == 0 {
		return 0, 0, 0
	}

	tag := data[0]

	// Check for end marker
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

	// Decode varint
	var shift uint
	n = 1
	for i := 1; i < len(data); i++ {
		b := data[i]
		fieldNum |= int(b&0x7F) << shift
		n++
		if b < 0x80 {
			break
		}
		shift += 7
	}

	return fieldNum, wireType, n
}

// CompactTagSize returns the encoded size of a compact tag.
func CompactTagSize(fieldNum int) int {
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

// WriteCompactTag writes a compact tag to the writer.
func (w *Writer) WriteCompactTag(fieldNum int, wireType byte) {
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

// ReadCompactTag reads a compact tag from the reader.
// Returns fieldNum=0 for end marker.
func (r *Reader) ReadCompactTag() (fieldNum int, wireType byte) {
	if r.err != nil || r.pos >= len(r.data) {
		return 0, 0
	}

	tag := r.data[r.pos]
	r.pos++

	// Check for end marker
	if tag == EndMarker {
		return 0, 0
	}

	wireType = (tag & tagWireTypeMask) >> tagWireTypeShift

	if tag&tagExtendedBit == 0 {
		// Compact format
		fieldNum = int(tag >> tagFieldNumShift)
		return fieldNum, wireType
	}

	// Extended format: read varint field number
	var shift uint
	for r.pos < len(r.data) {
		b := r.data[r.pos]
		r.pos++
		fieldNum |= int(b&0x7F) << shift
		if b < 0x80 {
			break
		}
		shift += 7
	}

	return fieldNum, wireType
}

// SkipValueV2 skips a value based on V2 wire type.
func (r *Reader) SkipValueV2(wireType byte) {
	if r.err != nil {
		return
	}

	switch wireType {
	case WireTypeV2Varint, WireTypeV2SVarint:
		// Skip varint
		for r.pos < len(r.data) {
			b := r.data[r.pos]
			r.pos++
			if b < 0x80 {
				break
			}
		}

	case WireTypeV2Fixed32:
		r.pos += 4

	case WireTypeV2Fixed64:
		r.pos += 8

	case WireTypeV2Bytes:
		// Read length, then skip that many bytes
		length := r.ReadUvarint()
		r.pos += int(length)

	default:
		r.setError(NewDecodeError("unknown wire type", nil))
	}

	if r.pos > len(r.data) {
		r.pos = len(r.data)
		r.setError(ErrUnexpectedEOF)
	}
}
