package cramberry

import (
	"github.com/cramberry/cramberry-go/internal/wire"
)

// Reader provides efficient binary decoding with position tracking.
// Readers are lightweight and can be reused.
//
// The zero value is not ready for use; create with NewReader.
type Reader struct {
	data  []byte
	pos   int
	opts  Options
	depth int
	err   error
}

// NewReader creates a new Reader for the given data.
func NewReader(data []byte) *Reader {
	return &Reader{
		data: data,
		opts: DefaultOptions,
	}
}

// NewReaderWithOptions creates a new Reader with the specified options.
func NewReaderWithOptions(data []byte, opts Options) *Reader {
	return &Reader{
		data: data,
		opts: opts,
	}
}

// Reset resets the reader to read from new data.
func (r *Reader) Reset(data []byte) {
	r.data = data
	r.pos = 0
	r.depth = 0
	r.err = nil
}

// SetOptions updates the reader's options.
func (r *Reader) SetOptions(opts Options) {
	r.opts = opts
}

// Options returns the reader's current options.
func (r *Reader) Options() Options {
	return r.opts
}

// Len returns the number of unread bytes.
func (r *Reader) Len() int {
	if r.pos >= len(r.data) {
		return 0
	}
	return len(r.data) - r.pos
}

// Pos returns the current read position.
func (r *Reader) Pos() int {
	return r.pos
}

// Data returns the underlying data slice.
func (r *Reader) Data() []byte {
	return r.data
}

// Remaining returns the unread portion of the data.
func (r *Reader) Remaining() []byte {
	if r.pos >= len(r.data) {
		return nil
	}
	return r.data[r.pos:]
}

// EOF returns true if all data has been read.
func (r *Reader) EOF() bool {
	return r.pos >= len(r.data)
}

// Err returns the first error that occurred during reading, if any.
func (r *Reader) Err() error {
	return r.err
}

// setError records the first error that occurs.
func (r *Reader) setError(err error) {
	if r.err == nil {
		r.err = err
	}
}

// setErrorAt records an error with position information.
func (r *Reader) setErrorAt(err error, message string) {
	if r.err == nil {
		r.err = NewDecodeErrorAt(r.pos, message, err)
	}
}

// checkRead ensures we can read from the buffer.
func (r *Reader) checkRead() bool {
	return r.err == nil
}

// ensure checks that n bytes are available.
func (r *Reader) ensure(n int) bool {
	if r.err != nil {
		return false
	}
	if r.pos+n > len(r.data) {
		r.setErrorAt(ErrUnexpectedEOF, "unexpected end of data")
		return false
	}
	return true
}

// enterNested increases the nesting depth and checks limits.
func (r *Reader) enterNested() bool {
	if r.opts.Limits.MaxDepth > 0 && r.depth >= r.opts.Limits.MaxDepth {
		r.setError(ErrMaxDepthExceeded)
		return false
	}
	r.depth++
	return true
}

// exitNested decreases the nesting depth.
func (r *Reader) exitNested() {
	if r.depth > 0 {
		r.depth--
	}
}

// Skip skips n bytes.
func (r *Reader) Skip(n int) {
	if !r.ensure(n) {
		return
	}
	r.pos += n
}

// ReadBool reads a boolean value.
func (r *Reader) ReadBool() bool {
	if !r.ensure(1) {
		return false
	}
	b := r.data[r.pos]
	r.pos++
	return b != 0
}

// ReadUint8 reads an unsigned 8-bit integer.
func (r *Reader) ReadUint8() uint8 {
	if !r.ensure(1) {
		return 0
	}
	v := r.data[r.pos]
	r.pos++
	return v
}

// ReadInt8 reads a signed 8-bit integer.
func (r *Reader) ReadInt8() int8 {
	if !r.ensure(1) {
		return 0
	}
	v := int8(r.data[r.pos])
	r.pos++
	return v
}

// ReadUint16 reads an unsigned 16-bit integer (varint encoded).
func (r *Reader) ReadUint16() uint16 {
	v := r.ReadUvarint()
	if r.err != nil {
		return 0
	}
	if v > 0xFFFF {
		r.setErrorAt(ErrOverflow, "uint16 overflow")
		return 0
	}
	return uint16(v)
}

// ReadUint32 reads an unsigned 32-bit integer (varint encoded).
func (r *Reader) ReadUint32() uint32 {
	v := r.ReadUvarint()
	if r.err != nil {
		return 0
	}
	if v > 0xFFFFFFFF {
		r.setErrorAt(ErrOverflow, "uint32 overflow")
		return 0
	}
	return uint32(v)
}

// ReadUint64 reads an unsigned 64-bit integer (varint encoded).
func (r *Reader) ReadUint64() uint64 {
	return r.ReadUvarint()
}

// ReadUint reads an unsigned integer (varint encoded).
func (r *Reader) ReadUint() uint {
	return uint(r.ReadUvarint())
}

// ReadInt16 reads a signed 16-bit integer (signed varint encoded).
func (r *Reader) ReadInt16() int16 {
	v := r.ReadSvarint()
	if r.err != nil {
		return 0
	}
	if v < -32768 || v > 32767 {
		r.setErrorAt(ErrOverflow, "int16 overflow")
		return 0
	}
	return int16(v)
}

// ReadInt32 reads a signed 32-bit integer (signed varint encoded).
func (r *Reader) ReadInt32() int32 {
	v := r.ReadSvarint()
	if r.err != nil {
		return 0
	}
	if v < -2147483648 || v > 2147483647 {
		r.setErrorAt(ErrOverflow, "int32 overflow")
		return 0
	}
	return int32(v)
}

// ReadInt64 reads a signed 64-bit integer (signed varint encoded).
func (r *Reader) ReadInt64() int64 {
	return r.ReadSvarint()
}

// ReadInt reads a signed integer (signed varint encoded).
func (r *Reader) ReadInt() int {
	return int(r.ReadSvarint())
}

// ReadUvarint reads an unsigned varint.
func (r *Reader) ReadUvarint() uint64 {
	if !r.checkRead() {
		return 0
	}
	v, n, err := wire.DecodeUvarint(r.data[r.pos:])
	if err != nil {
		r.setErrorAt(err, "invalid varint")
		return 0
	}
	r.pos += n
	return v
}

// ReadSvarint reads a signed varint (ZigZag encoded).
func (r *Reader) ReadSvarint() int64 {
	if !r.checkRead() {
		return 0
	}
	v, n, err := wire.DecodeSvarint(r.data[r.pos:])
	if err != nil {
		r.setErrorAt(err, "invalid signed varint")
		return 0
	}
	r.pos += n
	return v
}

// ReadFixed32 reads a fixed 32-bit value (little-endian).
func (r *Reader) ReadFixed32() uint32 {
	if !r.ensure(Fixed32Size) {
		return 0
	}
	v, _ := wire.DecodeFixed32(r.data[r.pos:])
	r.pos += Fixed32Size
	return v
}

// ReadFixed64 reads a fixed 64-bit value (little-endian).
func (r *Reader) ReadFixed64() uint64 {
	if !r.ensure(Fixed64Size) {
		return 0
	}
	v, _ := wire.DecodeFixed64(r.data[r.pos:])
	r.pos += Fixed64Size
	return v
}

// ReadSFixed32 reads a signed fixed 32-bit value (little-endian).
func (r *Reader) ReadSFixed32() int32 {
	return int32(r.ReadFixed32())
}

// ReadSFixed64 reads a signed fixed 64-bit value (little-endian).
func (r *Reader) ReadSFixed64() int64 {
	return int64(r.ReadFixed64())
}

// ReadFloat32 reads a 32-bit floating point number.
func (r *Reader) ReadFloat32() float32 {
	if !r.ensure(Float32Size) {
		return 0
	}
	v, _ := wire.DecodeFloat32(r.data[r.pos:])
	r.pos += Float32Size
	return v
}

// ReadFloat64 reads a 64-bit floating point number.
func (r *Reader) ReadFloat64() float64 {
	if !r.ensure(Float64Size) {
		return 0
	}
	v, _ := wire.DecodeFloat64(r.data[r.pos:])
	r.pos += Float64Size
	return v
}

// ReadComplex64 reads a complex64 value.
func (r *Reader) ReadComplex64() complex64 {
	if !r.ensure(Complex64Size) {
		return 0
	}
	v, _ := wire.DecodeComplex64(r.data[r.pos:])
	r.pos += Complex64Size
	return v
}

// ReadComplex128 reads a complex128 value.
func (r *Reader) ReadComplex128() complex128 {
	if !r.ensure(Complex128Size) {
		return 0
	}
	v, _ := wire.DecodeComplex128(r.data[r.pos:])
	r.pos += Complex128Size
	return v
}

// ReadString reads a length-prefixed string.
func (r *Reader) ReadString() string {
	if !r.checkRead() {
		return ""
	}
	length := r.ReadUvarint()
	if r.err != nil {
		return ""
	}
	if length > uint64(MaxInt) {
		r.setErrorAt(ErrOverflow, "string length overflow")
		return ""
	}
	n := int(length)
	// Check limits
	if r.opts.Limits.MaxStringLength > 0 && n > r.opts.Limits.MaxStringLength {
		r.setError(ErrMaxStringLength)
		return ""
	}
	if !r.ensure(n) {
		return ""
	}
	s := string(r.data[r.pos : r.pos+n])
	r.pos += n
	// Validate UTF-8 if required
	if r.opts.ValidateUTF8 && !isValidUTF8(s) {
		r.setError(ErrInvalidUTF8)
		return ""
	}
	return s
}

// ReadBytes reads a length-prefixed byte slice.
func (r *Reader) ReadBytes() []byte {
	if !r.checkRead() {
		return nil
	}
	length := r.ReadUvarint()
	if r.err != nil {
		return nil
	}
	if length > uint64(MaxInt) {
		r.setErrorAt(ErrOverflow, "bytes length overflow")
		return nil
	}
	n := int(length)
	// Check limits
	if r.opts.Limits.MaxBytesLength > 0 && n > r.opts.Limits.MaxBytesLength {
		r.setError(ErrMaxBytesLength)
		return nil
	}
	if !r.ensure(n) {
		return nil
	}
	// Return a copy to avoid aliasing
	result := make([]byte, n)
	copy(result, r.data[r.pos:r.pos+n])
	r.pos += n
	return result
}

// ReadBytesNoCopy reads a length-prefixed byte slice without copying.
// The returned slice is only valid until the next operation on the reader's data.
func (r *Reader) ReadBytesNoCopy() []byte {
	if !r.checkRead() {
		return nil
	}
	length := r.ReadUvarint()
	if r.err != nil {
		return nil
	}
	if length > uint64(MaxInt) {
		r.setErrorAt(ErrOverflow, "bytes length overflow")
		return nil
	}
	n := int(length)
	// Check limits
	if r.opts.Limits.MaxBytesLength > 0 && n > r.opts.Limits.MaxBytesLength {
		r.setError(ErrMaxBytesLength)
		return nil
	}
	if !r.ensure(n) {
		return nil
	}
	result := r.data[r.pos : r.pos+n]
	r.pos += n
	return result
}

// ReadRawBytes reads exactly n bytes without a length prefix.
func (r *Reader) ReadRawBytes(n int) []byte {
	if n < 0 {
		r.setError(ErrNegativeLength)
		return nil
	}
	if !r.ensure(n) {
		return nil
	}
	result := make([]byte, n)
	copy(result, r.data[r.pos:r.pos+n])
	r.pos += n
	return result
}

// ReadRawBytesNoCopy reads exactly n bytes without copying.
func (r *Reader) ReadRawBytesNoCopy(n int) []byte {
	if n < 0 {
		r.setError(ErrNegativeLength)
		return nil
	}
	if !r.ensure(n) {
		return nil
	}
	result := r.data[r.pos : r.pos+n]
	r.pos += n
	return result
}

// ReadTag reads a field tag (field number + wire type).
func (r *Reader) ReadTag() (fieldNum int, wireType WireType) {
	if !r.checkRead() {
		return 0, 0
	}
	fn, wt, n, err := wire.DecodeTag(r.data[r.pos:])
	if err != nil {
		r.setErrorAt(err, "invalid field tag")
		return 0, 0
	}
	r.pos += n
	return fn, WireType(wt)
}

// ReadTypeID reads a type ID for polymorphic decoding.
func (r *Reader) ReadTypeID() TypeID {
	v := r.ReadUvarint()
	if r.err != nil {
		return TypeIDNil
	}
	return TypeID(v)
}

// BeginMessage starts reading a length-prefixed message.
// Returns the end position that should be passed to EndMessage.
// The reader will be limited to reading within the message bounds.
func (r *Reader) BeginMessage() int {
	if !r.checkRead() {
		return -1
	}
	if !r.enterNested() {
		return -1
	}
	length := r.ReadUvarint()
	if r.err != nil {
		return -1
	}
	if length > uint64(MaxInt) {
		r.setErrorAt(ErrOverflow, "message length overflow")
		return -1
	}
	msgLen := int(length)
	// Check message size limit
	if r.opts.Limits.MaxMessageSize > 0 && int64(msgLen) > r.opts.Limits.MaxMessageSize {
		r.setError(ErrMaxSizeExceeded)
		return -1
	}
	if !r.ensure(msgLen) {
		return -1
	}
	return r.pos + msgLen
}

// EndMessage finishes reading a length-prefixed message.
// The endPos should be the value returned by BeginMessage.
// If not all bytes were read, the position is advanced to the end.
func (r *Reader) EndMessage(endPos int) {
	if endPos < 0 || r.err != nil {
		return
	}
	r.exitNested()
	if r.pos < endPos {
		// Skip any unread bytes in the message
		r.pos = endPos
	} else if r.pos > endPos {
		r.setErrorAt(ErrOverflow, "read past message boundary")
	}
}

// ReadArrayHeader reads the length of an array/slice.
func (r *Reader) ReadArrayHeader() int {
	if !r.checkRead() {
		return 0
	}
	length := r.ReadUvarint()
	if r.err != nil {
		return 0
	}
	if length > uint64(MaxInt) {
		r.setErrorAt(ErrOverflow, "array length overflow")
		return 0
	}
	n := int(length)
	// Check limits
	if r.opts.Limits.MaxArrayLength > 0 && n > r.opts.Limits.MaxArrayLength {
		r.setError(ErrMaxArrayLength)
		return 0
	}
	return n
}

// ReadMapHeader reads the size of a map.
func (r *Reader) ReadMapHeader() int {
	if !r.checkRead() {
		return 0
	}
	size := r.ReadUvarint()
	if r.err != nil {
		return 0
	}
	if size > uint64(MaxInt) {
		r.setErrorAt(ErrOverflow, "map size overflow")
		return 0
	}
	n := int(size)
	// Check limits
	if r.opts.Limits.MaxMapSize > 0 && n > r.opts.Limits.MaxMapSize {
		r.setError(ErrMaxMapSize)
		return 0
	}
	return n
}

// SkipValue skips a value based on its wire type.
func (r *Reader) SkipValue(wireType WireType) {
	if !r.checkRead() {
		return
	}
	switch wireType {
	case WireVarint, WireSVarint:
		_ = r.ReadUvarint()
	case WireFixed64:
		r.Skip(Fixed64Size)
	case WireFixed32:
		r.Skip(Fixed32Size)
	case WireBytes:
		length := r.ReadUvarint()
		if r.err != nil {
			return
		}
		if length > uint64(MaxInt) {
			r.setErrorAt(ErrOverflow, "skip length overflow")
			return
		}
		r.Skip(int(length))
	case WireTypeRef:
		// TypeRef is a varint type ID followed by the actual value
		// We can't fully skip without knowing the type, so just skip the type ID
		_ = r.ReadUvarint()
	default:
		r.setErrorAt(ErrInvalidWireType, "unknown wire type")
	}
}

// SubReader creates a sub-reader for a portion of the data.
// The sub-reader has independent position tracking but shares the underlying data.
func (r *Reader) SubReader(length int) *Reader {
	if !r.ensure(length) {
		return nil
	}
	sub := &Reader{
		data: r.data[r.pos : r.pos+length],
		opts: r.opts,
	}
	r.pos += length
	return sub
}

// MaxInt is the maximum value of int (platform dependent).
const MaxInt = int(^uint(0) >> 1)
