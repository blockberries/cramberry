package cramberry

import (
	"math"
	"sync"

	"github.com/cramberry/cramberry-go/internal/wire"
)

// Writer provides efficient binary encoding with buffer management.
// Writers can be reused to reduce allocations.
//
// The zero value is ready to use, but for better performance,
// use NewWriter or a sync.Pool of writers.
type Writer struct {
	buf    []byte
	opts   Options
	depth  int
	err    error
	frozen bool // prevents further writes after Bytes() is called
}

// writerPool provides pooled writers for reduced allocations.
var writerPool = sync.Pool{
	New: func() any {
		return &Writer{
			buf:  make([]byte, 0, 256),
			opts: DefaultOptions,
		}
	},
}

// NewWriter creates a new Writer with default options.
func NewWriter() *Writer {
	return &Writer{
		buf:  make([]byte, 0, 256),
		opts: DefaultOptions,
	}
}

// NewWriterWithOptions creates a new Writer with the specified options.
func NewWriterWithOptions(opts Options) *Writer {
	return &Writer{
		buf:  make([]byte, 0, 256),
		opts: opts,
	}
}

// NewWriterWithBuffer creates a Writer using the provided buffer.
// The buffer will be reused if it has sufficient capacity.
func NewWriterWithBuffer(buf []byte, opts Options) *Writer {
	return &Writer{
		buf:  buf[:0],
		opts: opts,
	}
}

// GetWriter gets a Writer from the pool.
// The Writer should be returned with PutWriter when done.
func GetWriter() *Writer {
	w := writerPool.Get().(*Writer)
	w.Reset()
	return w
}

// PutWriter returns a Writer to the pool.
// The Writer must not be used after calling this.
func PutWriter(w *Writer) {
	if w == nil {
		return
	}
	// Don't pool large buffers to avoid memory bloat
	if cap(w.buf) > 64*1024 {
		return
	}
	w.Reset()
	writerPool.Put(w)
}

// Reset clears the writer for reuse.
func (w *Writer) Reset() {
	w.buf = w.buf[:0]
	w.depth = 0
	w.err = nil
	w.frozen = false
}

// SetOptions updates the writer's options.
func (w *Writer) SetOptions(opts Options) {
	w.opts = opts
}

// Options returns the writer's current options.
func (w *Writer) Options() Options {
	return w.opts
}

// Len returns the current length of the encoded data.
func (w *Writer) Len() int {
	return len(w.buf)
}

// Cap returns the current capacity of the internal buffer.
func (w *Writer) Cap() int {
	return cap(w.buf)
}

// Bytes returns the encoded data.
// The returned slice is only valid until the next call to Reset or any Write method.
// To get a copy, use BytesCopy.
func (w *Writer) Bytes() []byte {
	w.frozen = true
	return w.buf
}

// BytesCopy returns a copy of the encoded data.
// This is safe to use after Reset or further writes.
func (w *Writer) BytesCopy() []byte {
	result := make([]byte, len(w.buf))
	copy(result, w.buf)
	return result
}

// Err returns the first error that occurred during writing, if any.
func (w *Writer) Err() error {
	return w.err
}

// setError records the first error that occurs.
func (w *Writer) setError(err error) {
	if w.err == nil {
		w.err = err
	}
}

// checkWrite ensures we can write to the buffer.
func (w *Writer) checkWrite() bool {
	if w.frozen {
		w.setError(NewEncodeError("writer is frozen after Bytes() call", nil))
		return false
	}
	if w.err != nil {
		return false
	}
	return true
}

// grow ensures the buffer has room for n more bytes.
func (w *Writer) grow(n int) {
	if len(w.buf)+n <= cap(w.buf) {
		return
	}
	// Check size limit
	if w.opts.Limits.MaxMessageSize > 0 && int64(len(w.buf)+n) > w.opts.Limits.MaxMessageSize {
		w.setError(ErrMaxSizeExceeded)
		return
	}
	// Grow by doubling, with a minimum growth
	newCap := cap(w.buf) * 2
	if newCap < len(w.buf)+n {
		newCap = len(w.buf) + n
	}
	// Cap growth to avoid excessive allocation
	if newCap > 256*1024*1024 {
		newCap = len(w.buf) + n
	}
	newBuf := make([]byte, len(w.buf), newCap)
	copy(newBuf, w.buf)
	w.buf = newBuf
}

// enterNested increases the nesting depth and checks limits.
func (w *Writer) enterNested() bool {
	if w.opts.Limits.MaxDepth > 0 && w.depth >= w.opts.Limits.MaxDepth {
		w.setError(ErrMaxDepthExceeded)
		return false
	}
	w.depth++
	return true
}

// exitNested decreases the nesting depth.
func (w *Writer) exitNested() {
	if w.depth > 0 {
		w.depth--
	}
}

// WriteBool writes a boolean value.
func (w *Writer) WriteBool(v bool) {
	if !w.checkWrite() {
		return
	}
	w.grow(1)
	if v {
		w.buf = append(w.buf, 1)
	} else {
		w.buf = append(w.buf, 0)
	}
}

// WriteUint8 writes an unsigned 8-bit integer.
func (w *Writer) WriteUint8(v uint8) {
	if !w.checkWrite() {
		return
	}
	w.grow(1)
	w.buf = append(w.buf, v)
}

// WriteUint16 writes an unsigned 16-bit integer as a varint.
func (w *Writer) WriteUint16(v uint16) {
	w.WriteUvarint(uint64(v))
}

// WriteUint32 writes an unsigned 32-bit integer as a varint.
func (w *Writer) WriteUint32(v uint32) {
	w.WriteUvarint(uint64(v))
}

// WriteUint64 writes an unsigned 64-bit integer as a varint.
func (w *Writer) WriteUint64(v uint64) {
	w.WriteUvarint(v)
}

// WriteUint writes an unsigned integer as a varint.
func (w *Writer) WriteUint(v uint) {
	w.WriteUvarint(uint64(v))
}

// WriteInt8 writes a signed 8-bit integer.
func (w *Writer) WriteInt8(v int8) {
	if !w.checkWrite() {
		return
	}
	w.grow(1)
	w.buf = append(w.buf, byte(v))
}

// WriteInt16 writes a signed 16-bit integer as a signed varint.
func (w *Writer) WriteInt16(v int16) {
	w.WriteSvarint(int64(v))
}

// WriteInt32 writes a signed 32-bit integer as a signed varint.
func (w *Writer) WriteInt32(v int32) {
	w.WriteSvarint(int64(v))
}

// WriteInt64 writes a signed 64-bit integer as a signed varint.
func (w *Writer) WriteInt64(v int64) {
	w.WriteSvarint(v)
}

// WriteInt writes a signed integer as a signed varint.
func (w *Writer) WriteInt(v int) {
	w.WriteSvarint(int64(v))
}

// WriteUvarint writes an unsigned varint.
func (w *Writer) WriteUvarint(v uint64) {
	if !w.checkWrite() {
		return
	}
	w.grow(MaxVarintLen64)
	w.buf = wire.AppendUvarint(w.buf, v)
}

// WriteUvarintInline writes an unsigned varint with inlined fast path for 1-2 byte values.
// This is faster for small values (< 16384) which are common.
func (w *Writer) WriteUvarintInline(v uint64) {
	if w.frozen || w.err != nil {
		if !w.frozen {
			return
		}
		w.setError(NewEncodeError("writer is frozen after Bytes() call", nil))
		return
	}

	// Fast path: single byte (value < 128)
	if v < 0x80 {
		if len(w.buf)+1 > cap(w.buf) {
			w.grow(1)
		}
		w.buf = append(w.buf, byte(v))
		return
	}

	// Fast path: two bytes (value < 16384)
	if v < 0x4000 {
		if len(w.buf)+2 > cap(w.buf) {
			w.grow(2)
		}
		w.buf = append(w.buf, byte(v)|0x80, byte(v>>7))
		return
	}

	// Slow path: delegate to wire package
	w.grow(MaxVarintLen64)
	w.buf = wire.AppendUvarint(w.buf, v)
}

// WriteSvarint writes a signed varint using ZigZag encoding.
func (w *Writer) WriteSvarint(v int64) {
	if !w.checkWrite() {
		return
	}
	w.grow(MaxVarintLen64)
	w.buf = wire.AppendSvarint(w.buf, v)
}

// WriteSvarintInline writes a signed varint with inlined fast path.
func (w *Writer) WriteSvarintInline(v int64) {
	// ZigZag encode: (v << 1) ^ (v >> 63)
	u := uint64(v<<1) ^ uint64(v>>63)
	w.WriteUvarintInline(u)
}

// WriteFloat32 writes a 32-bit floating point number.
// The encoding is deterministic: -0 becomes +0, and NaN is normalized.
func (w *Writer) WriteFloat32(v float32) {
	if !w.checkWrite() {
		return
	}
	w.grow(Float32Size)
	w.buf = wire.AppendFloat32(w.buf, v)
}

// WriteFloat64 writes a 64-bit floating point number.
// The encoding is deterministic: -0 becomes +0, and NaN is normalized.
func (w *Writer) WriteFloat64(v float64) {
	if !w.checkWrite() {
		return
	}
	w.grow(Float64Size)
	w.buf = wire.AppendFloat64(w.buf, v)
}

// WriteComplex64 writes a complex64 value.
func (w *Writer) WriteComplex64(v complex64) {
	if !w.checkWrite() {
		return
	}
	w.grow(Complex64Size)
	w.buf = wire.AppendComplex64(w.buf, v)
}

// WriteComplex128 writes a complex128 value.
func (w *Writer) WriteComplex128(v complex128) {
	if !w.checkWrite() {
		return
	}
	w.grow(Complex128Size)
	w.buf = wire.AppendComplex128(w.buf, v)
}

// WriteFixed32 writes a fixed 32-bit value (little-endian).
func (w *Writer) WriteFixed32(v uint32) {
	if !w.checkWrite() {
		return
	}
	w.grow(Fixed32Size)
	w.buf = wire.AppendFixed32(w.buf, v)
}

// WriteFixed64 writes a fixed 64-bit value (little-endian).
func (w *Writer) WriteFixed64(v uint64) {
	if !w.checkWrite() {
		return
	}
	w.grow(Fixed64Size)
	w.buf = wire.AppendFixed64(w.buf, v)
}

// WriteSFixed32 writes a signed fixed 32-bit value (little-endian).
func (w *Writer) WriteSFixed32(v int32) {
	w.WriteFixed32(uint32(v))
}

// WriteSFixed64 writes a signed fixed 64-bit value (little-endian).
func (w *Writer) WriteSFixed64(v int64) {
	w.WriteFixed64(uint64(v))
}

// WriteString writes a length-prefixed string.
func (w *Writer) WriteString(s string) {
	if !w.checkWrite() {
		return
	}
	// Check string length limit
	if w.opts.Limits.MaxStringLength > 0 && len(s) > w.opts.Limits.MaxStringLength {
		w.setError(ErrMaxStringLength)
		return
	}
	// Validate UTF-8 if required
	if w.opts.ValidateUTF8 && !isValidUTF8(s) {
		w.setError(ErrInvalidUTF8)
		return
	}
	// Write length prefix
	w.WriteUvarint(uint64(len(s)))
	if w.err != nil {
		return
	}
	// Write string data
	w.grow(len(s))
	w.buf = append(w.buf, s...)
}

// WriteBytes writes a length-prefixed byte slice.
func (w *Writer) WriteBytes(b []byte) {
	if !w.checkWrite() {
		return
	}
	// Check bytes length limit
	if w.opts.Limits.MaxBytesLength > 0 && len(b) > w.opts.Limits.MaxBytesLength {
		w.setError(ErrMaxBytesLength)
		return
	}
	// Write length prefix
	w.WriteUvarint(uint64(len(b)))
	if w.err != nil {
		return
	}
	// Write byte data
	w.grow(len(b))
	w.buf = append(w.buf, b...)
}

// WriteRawBytes writes raw bytes without a length prefix.
func (w *Writer) WriteRawBytes(b []byte) {
	if !w.checkWrite() {
		return
	}
	w.grow(len(b))
	w.buf = append(w.buf, b...)
}

// WriteTag writes a field tag (field number + wire type).
func (w *Writer) WriteTag(fieldNum int, wireType WireType) {
	if !w.checkWrite() {
		return
	}
	if fieldNum <= 0 {
		w.setError(ErrInvalidFieldNumber)
		return
	}
	w.grow(MaxTagSize)
	w.buf = wire.AppendTag(w.buf, fieldNum, wire.WireType(wireType))
}

// WriteNil writes a nil marker (TypeID 0).
func (w *Writer) WriteNil() {
	if !w.checkWrite() {
		return
	}
	w.WriteUvarint(uint64(TypeIDNil))
}

// WriteTypeID writes a type ID for polymorphic encoding.
func (w *Writer) WriteTypeID(id TypeID) {
	if !w.checkWrite() {
		return
	}
	w.WriteUvarint(uint64(id))
}

// BeginMessage starts writing a length-prefixed message.
// Returns a checkpoint that must be passed to EndMessage.
func (w *Writer) BeginMessage() int {
	if !w.checkWrite() {
		return -1
	}
	if !w.enterNested() {
		return -1
	}
	// Reserve space for length (we'll fill it in later)
	// We reserve MaxVarintLen64 bytes to handle any message size
	checkpoint := len(w.buf)
	w.grow(MaxVarintLen64)
	w.buf = append(w.buf, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	return checkpoint
}

// EndMessage finishes writing a length-prefixed message.
// The checkpoint should be the value returned by BeginMessage.
func (w *Writer) EndMessage(checkpoint int) {
	if checkpoint < 0 || w.err != nil {
		return
	}
	w.exitNested()

	// Calculate the message length (excluding the length prefix placeholder)
	msgStart := checkpoint + MaxVarintLen64
	msgLen := len(w.buf) - msgStart

	// Encode the length to a temporary buffer
	var lenBuf [MaxVarintLen64]byte
	lenBytes := wire.AppendUvarint(lenBuf[:0], uint64(msgLen))
	lenSize := len(lenBytes)

	// Calculate how many bytes we need to shift
	shift := MaxVarintLen64 - lenSize

	if shift > 0 {
		// Move message content to close the gap
		copy(w.buf[checkpoint+lenSize:], w.buf[msgStart:])
		w.buf = w.buf[:len(w.buf)-shift]
	}

	// Write the length prefix
	copy(w.buf[checkpoint:], lenBytes)
}

// WriteArrayHeader writes the length of an array/slice.
func (w *Writer) WriteArrayHeader(length int) {
	if !w.checkWrite() {
		return
	}
	if length < 0 {
		w.setError(ErrNegativeLength)
		return
	}
	if w.opts.Limits.MaxArrayLength > 0 && length > w.opts.Limits.MaxArrayLength {
		w.setError(ErrMaxArrayLength)
		return
	}
	w.WriteUvarint(uint64(length))
}

// WriteMapHeader writes the size of a map.
func (w *Writer) WriteMapHeader(size int) {
	if !w.checkWrite() {
		return
	}
	if size < 0 {
		w.setError(ErrNegativeLength)
		return
	}
	if w.opts.Limits.MaxMapSize > 0 && size > w.opts.Limits.MaxMapSize {
		w.setError(ErrMaxMapSize)
		return
	}
	w.WriteUvarint(uint64(size))
}

// isValidUTF8 checks if a string is valid UTF-8.
// This is a fast implementation that avoids allocations.
func isValidUTF8(s string) bool {
	for i := 0; i < len(s); {
		if s[i] < 0x80 {
			// ASCII
			i++
			continue
		}
		// Multi-byte sequence
		size := utf8SequenceSize(s[i])
		if size == 0 {
			return false // Invalid leading byte
		}
		if i+size > len(s) {
			return false // Truncated sequence
		}
		// Validate continuation bytes
		for j := 1; j < size; j++ {
			if s[i+j]&0xC0 != 0x80 {
				return false // Invalid continuation byte
			}
		}
		// Validate minimum encoding and range
		var codepoint uint32
		switch size {
		case 2:
			codepoint = uint32(s[i]&0x1F)<<6 | uint32(s[i+1]&0x3F)
			if codepoint < 0x80 {
				return false // Overlong encoding
			}
		case 3:
			codepoint = uint32(s[i]&0x0F)<<12 | uint32(s[i+1]&0x3F)<<6 | uint32(s[i+2]&0x3F)
			if codepoint < 0x800 {
				return false // Overlong encoding
			}
			if codepoint >= 0xD800 && codepoint <= 0xDFFF {
				return false // Surrogate pairs not allowed
			}
		case 4:
			codepoint = uint32(s[i]&0x07)<<18 | uint32(s[i+1]&0x3F)<<12 | uint32(s[i+2]&0x3F)<<6 | uint32(s[i+3]&0x3F)
			if codepoint < 0x10000 || codepoint > 0x10FFFF {
				return false // Overlong or out of range
			}
		}
		i += size
	}
	return true
}

// utf8SequenceSize returns the size of a UTF-8 sequence based on the leading byte.
// Returns 0 for invalid leading bytes.
func utf8SequenceSize(b byte) int {
	if b < 0x80 {
		return 1
	}
	if b < 0xC0 {
		return 0 // Continuation byte, invalid as leading
	}
	if b < 0xE0 {
		return 2
	}
	if b < 0xF0 {
		return 3
	}
	if b < 0xF8 {
		return 4
	}
	return 0 // Invalid
}

// SizeOfBool returns the encoded size of a boolean.
func SizeOfBool(_ bool) int {
	return BoolSize
}

// SizeOfUint8 returns the encoded size of a uint8.
func SizeOfUint8(_ uint8) int {
	return 1
}

// SizeOfUint16 returns the encoded size of a uint16.
func SizeOfUint16(v uint16) int {
	return wire.UvarintSize(uint64(v))
}

// SizeOfUint32 returns the encoded size of a uint32.
func SizeOfUint32(v uint32) int {
	return wire.UvarintSize(uint64(v))
}

// SizeOfUint64 returns the encoded size of a uint64.
func SizeOfUint64(v uint64) int {
	return wire.UvarintSize(v)
}

// SizeOfInt8 returns the encoded size of an int8.
func SizeOfInt8(_ int8) int {
	return 1
}

// SizeOfInt16 returns the encoded size of an int16.
func SizeOfInt16(v int16) int {
	return wire.SvarintSize(int64(v))
}

// SizeOfInt32 returns the encoded size of an int32.
func SizeOfInt32(v int32) int {
	return wire.SvarintSize(int64(v))
}

// SizeOfInt64 returns the encoded size of an int64.
func SizeOfInt64(v int64) int {
	return wire.SvarintSize(v)
}

// SizeOfFloat32 returns the encoded size of a float32.
func SizeOfFloat32(_ float32) int {
	return Float32Size
}

// SizeOfFloat64 returns the encoded size of a float64.
func SizeOfFloat64(_ float64) int {
	return Float64Size
}

// SizeOfComplex64 returns the encoded size of a complex64.
func SizeOfComplex64(_ complex64) int {
	return Complex64Size
}

// SizeOfComplex128 returns the encoded size of a complex128.
func SizeOfComplex128(_ complex128) int {
	return Complex128Size
}

// SizeOfString returns the encoded size of a string (length prefix + data).
func SizeOfString(s string) int {
	return wire.UvarintSize(uint64(len(s))) + len(s)
}

// SizeOfBytes returns the encoded size of a byte slice (length prefix + data).
func SizeOfBytes(b []byte) int {
	return wire.UvarintSize(uint64(len(b))) + len(b)
}

// SizeOfTag returns the encoded size of a field tag.
func SizeOfTag(fieldNum int) int {
	return wire.UvarintSize(uint64(fieldNum) << 3)
}

// SizeOfUvarint returns the encoded size of an unsigned varint.
func SizeOfUvarint(v uint64) int {
	return wire.UvarintSize(v)
}

// SizeOfSvarint returns the encoded size of a signed varint.
func SizeOfSvarint(v int64) int {
	return wire.SvarintSize(v)
}

// SizeOfSFixed32 returns the encoded size of a signed fixed 32-bit value.
func SizeOfSFixed32(_ int32) int {
	return Fixed32Size
}

// SizeOfSFixed64 returns the encoded size of a signed fixed 64-bit value.
func SizeOfSFixed64(_ int64) int {
	return Fixed64Size
}

// Suppress unused import warning
var _ = math.Float32bits

// ============================================================================
// Fast Packed Array Writers - Direct Memory Layout for Fixed-Size Types
// ============================================================================

// WritePackedFloat32 writes a packed array of float32 values.
// This is faster than writing element by element for large arrays.
func (w *Writer) WritePackedFloat32(values []float32) {
	if !w.checkWrite() {
		return
	}
	if len(values) == 0 {
		return
	}
	byteSize := len(values) * 4
	w.grow(byteSize)
	for _, v := range values {
		bits := math.Float32bits(v)
		w.buf = append(w.buf,
			byte(bits),
			byte(bits>>8),
			byte(bits>>16),
			byte(bits>>24))
	}
}

// WritePackedFloat64 writes a packed array of float64 values.
func (w *Writer) WritePackedFloat64(values []float64) {
	if !w.checkWrite() {
		return
	}
	if len(values) == 0 {
		return
	}
	byteSize := len(values) * 8
	w.grow(byteSize)
	for _, v := range values {
		bits := math.Float64bits(v)
		w.buf = append(w.buf,
			byte(bits),
			byte(bits>>8),
			byte(bits>>16),
			byte(bits>>24),
			byte(bits>>32),
			byte(bits>>40),
			byte(bits>>48),
			byte(bits>>56))
	}
}

// WritePackedFixed32 writes a packed array of fixed 32-bit values.
func (w *Writer) WritePackedFixed32(values []uint32) {
	if !w.checkWrite() {
		return
	}
	if len(values) == 0 {
		return
	}
	byteSize := len(values) * 4
	w.grow(byteSize)
	for _, v := range values {
		w.buf = append(w.buf,
			byte(v),
			byte(v>>8),
			byte(v>>16),
			byte(v>>24))
	}
}

// WritePackedFixed64 writes a packed array of fixed 64-bit values.
func (w *Writer) WritePackedFixed64(values []uint64) {
	if !w.checkWrite() {
		return
	}
	if len(values) == 0 {
		return
	}
	byteSize := len(values) * 8
	w.grow(byteSize)
	for _, v := range values {
		w.buf = append(w.buf,
			byte(v),
			byte(v>>8),
			byte(v>>16),
			byte(v>>24),
			byte(v>>32),
			byte(v>>40),
			byte(v>>48),
			byte(v>>56))
	}
}
