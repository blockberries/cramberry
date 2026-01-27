package cramberry

import (
	"unsafe"

	"github.com/blockberries/cramberry/internal/wire"
)

// Reader provides efficient binary decoding with position tracking.
// Readers are lightweight and can be reused.
//
// The zero value is not ready for use; create with NewReader.
type Reader struct {
	data       []byte
	pos        int
	opts       Options
	depth      int
	err        error
	generation uint64 // Incremented on Reset() to invalidate zero-copy references
}

// ZeroCopyString is a string that references the Reader's buffer directly.
// It validates that the Reader hasn't been reset before allowing access.
//
// Use the String() method to get the string value. If the Reader has been
// reset since this ZeroCopyString was created, String() will panic with
// a clear error message rather than returning corrupted data.
type ZeroCopyString struct {
	s          string
	generation uint64
	reader     *Reader
}

// String returns the string value, panicking if the Reader has been reset.
// This implements fmt.Stringer for convenient use in fmt functions.
func (zcs ZeroCopyString) String() string {
	if zcs.reader != nil && zcs.reader.generation != zcs.generation {
		panic("cramberry: ZeroCopyString accessed after Reader.Reset() - this would cause memory corruption")
	}
	return zcs.s
}

// Valid returns true if the ZeroCopyString is still valid (Reader not reset).
func (zcs ZeroCopyString) Valid() bool {
	return zcs.reader == nil || zcs.reader.generation == zcs.generation
}

// UnsafeString returns the underlying string without validation.
// Only use this if you have externally guaranteed the Reader hasn't been reset.
func (zcs ZeroCopyString) UnsafeString() string {
	return zcs.s
}

// IsEmpty returns true if the string is empty.
func (zcs ZeroCopyString) IsEmpty() bool {
	return len(zcs.s) == 0
}

// Len returns the length of the string.
func (zcs ZeroCopyString) Len() int {
	return len(zcs.s)
}

// MustString returns the string value, panicking if the Reader has been reset.
// This is an alias for String() that makes the panic behavior explicit in the name.
func (zcs ZeroCopyString) MustString() string {
	return zcs.String()
}

// StringOrEmpty returns the string value if valid, or an empty string if the
// Reader has been reset. This is a non-panicking alternative to String().
func (zcs ZeroCopyString) StringOrEmpty() string {
	if !zcs.Valid() {
		return ""
	}
	return zcs.s
}

// TryString returns the string value and a boolean indicating validity.
// Returns ("", false) if the Reader has been reset, otherwise (value, true).
// This allows explicit error checking without panics.
func (zcs ZeroCopyString) TryString() (string, bool) {
	if !zcs.Valid() {
		return "", false
	}
	return zcs.s, true
}

// ZeroCopyBytes is a byte slice that references the Reader's buffer directly.
// It validates that the Reader hasn't been reset before allowing access.
//
// Use the Bytes() method to get the slice. If the Reader has been reset
// since this ZeroCopyBytes was created, Bytes() will panic with a clear
// error message rather than returning corrupted data.
type ZeroCopyBytes struct {
	b          []byte
	generation uint64
	reader     *Reader
}

// Bytes returns the byte slice, panicking if the Reader has been reset.
func (zcb ZeroCopyBytes) Bytes() []byte {
	if zcb.reader != nil && zcb.reader.generation != zcb.generation {
		panic("cramberry: ZeroCopyBytes accessed after Reader.Reset() - this would cause memory corruption")
	}
	return zcb.b
}

// Valid returns true if the ZeroCopyBytes is still valid (Reader not reset).
func (zcb ZeroCopyBytes) Valid() bool {
	return zcb.reader == nil || zcb.reader.generation == zcb.generation
}

// UnsafeBytes returns the underlying slice without validation.
// Only use this if you have externally guaranteed the Reader hasn't been reset.
func (zcb ZeroCopyBytes) UnsafeBytes() []byte {
	return zcb.b
}

// IsEmpty returns true if the slice is empty.
func (zcb ZeroCopyBytes) IsEmpty() bool {
	return len(zcb.b) == 0
}

// Len returns the length of the byte slice.
func (zcb ZeroCopyBytes) Len() int {
	return len(zcb.b)
}

// String returns the bytes as a string, panicking if the Reader has been reset.
// This implements fmt.Stringer for convenient use in fmt functions.
func (zcb ZeroCopyBytes) String() string {
	if zcb.reader != nil && zcb.reader.generation != zcb.generation {
		panic("cramberry: ZeroCopyBytes accessed after Reader.Reset() - this would cause memory corruption")
	}
	return string(zcb.b)
}

// MustBytes returns the byte slice, panicking if the Reader has been reset.
// This is an alias for Bytes() that makes the panic behavior explicit in the name.
func (zcb ZeroCopyBytes) MustBytes() []byte {
	return zcb.Bytes()
}

// BytesOrNil returns the byte slice if valid, or nil if the Reader has been reset.
// This is a non-panicking alternative to Bytes().
func (zcb ZeroCopyBytes) BytesOrNil() []byte {
	if !zcb.Valid() {
		return nil
	}
	return zcb.b
}

// TryBytes returns the byte slice and a boolean indicating validity.
// Returns (nil, false) if the Reader has been reset, otherwise (value, true).
// This allows explicit error checking without panics.
func (zcb ZeroCopyBytes) TryBytes() ([]byte, bool) {
	if !zcb.Valid() {
		return nil, false
	}
	return zcb.b, true
}

// StringOrEmpty returns the bytes as a string if valid, or an empty string if
// the Reader has been reset. This is a non-panicking alternative to String().
func (zcb ZeroCopyBytes) StringOrEmpty() string {
	if !zcb.Valid() {
		return ""
	}
	return string(zcb.b)
}

// TryString returns the bytes as a string and a boolean indicating validity.
// Returns ("", false) if the Reader has been reset, otherwise (value, true).
func (zcb ZeroCopyBytes) TryString() (string, bool) {
	if !zcb.Valid() {
		return "", false
	}
	return string(zcb.b), true
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
// This invalidates all ZeroCopyString and ZeroCopyBytes values obtained
// from this reader - accessing them after Reset will panic.
func (r *Reader) Reset(data []byte) {
	r.data = data
	r.pos = 0
	r.depth = 0
	r.err = nil
	r.generation++ // Invalidate all zero-copy references
}

// Generation returns the current generation counter.
// This is incremented each time Reset() is called.
func (r *Reader) Generation() uint64 {
	return r.generation
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

// ReadUvarintInline reads an unsigned varint with inlined fast path for 1-2 byte values.
// This is faster for small values (< 16384) which are common.
func (r *Reader) ReadUvarintInline() uint64 {
	if r.err != nil || r.pos >= len(r.data) {
		if r.err == nil {
			r.setErrorAt(ErrUnexpectedEOF, "unexpected end of data")
		}
		return 0
	}

	// Fast path: single byte (value < 128)
	b := r.data[r.pos]
	if b < 0x80 {
		r.pos++
		return uint64(b)
	}

	// Fast path: two bytes (value < 16384)
	if r.pos+1 < len(r.data) {
		b2 := r.data[r.pos+1]
		if b2 < 0x80 {
			r.pos += 2
			return uint64(b&0x7f) | uint64(b2)<<7
		}
	}

	// Slow path: delegate to wire package
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

// ReadSvarintInline reads a signed varint with inlined fast path.
func (r *Reader) ReadSvarintInline() int64 {
	u := r.ReadUvarintInline()
	// ZigZag decode: (u >> 1) ^ -(u & 1)
	return int64(u>>1) ^ -int64(u&1)
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
	length := r.ReadUvarintInline()
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

// ReadStringZeroCopy reads a length-prefixed string without allocating.
//
// SAFETY WARNING: The returned string points directly into the Reader's buffer.
// It is only valid under these conditions:
//   - The Reader must NOT be Reset() while the string is in use
//   - The underlying data buffer must NOT be modified or freed
//   - The Reader must remain in scope
//
// Failure to observe these constraints will cause undefined behavior,
// including memory corruption, crashes, or data races.
//
// For safe usage, prefer ReadString() instead. Use ReadStringZeroCopy() only when:
//   - Performance is critical and profiling shows ReadString() as a bottleneck
//   - You can guarantee the Reader outlives all returned strings
//   - You will NOT call Reset() while strings are in use
//   - You will NOT store the returned string beyond the current function scope
//
// Example of UNSAFE usage (DO NOT DO THIS):
//
//	r := cramberry.NewReader(data)
//	s := r.ReadStringZeroCopy()
//	r.Reset(newData)  // UNDEFINED BEHAVIOR: s now points to invalid memory
//	fmt.Println(s)    // CRASH or data corruption
//
// Example of safe usage:
//
//	func processMessage(data []byte) string {
//	    r := cramberry.NewReader(data)
//	    s := r.ReadStringZeroCopy()
//	    result := processString(s)  // Use s immediately, don't store it
//	    return result               // Don't return s itself
//	}
func (r *Reader) ReadStringZeroCopy() ZeroCopyString {
	if !r.checkRead() {
		return ZeroCopyString{}
	}
	length := r.ReadUvarintInline()
	if r.err != nil {
		return ZeroCopyString{}
	}
	if length > uint64(MaxInt) {
		r.setErrorAt(ErrOverflow, "string length overflow")
		return ZeroCopyString{}
	}
	n := int(length)
	// Check limits
	if r.opts.Limits.MaxStringLength > 0 && n > r.opts.Limits.MaxStringLength {
		r.setError(ErrMaxStringLength)
		return ZeroCopyString{}
	}
	if !r.ensure(n) {
		return ZeroCopyString{}
	}
	// Zero-copy: create string header pointing to buffer
	var s string
	if n > 0 {
		s = unsafe.String(&r.data[r.pos], n)
	}
	r.pos += n
	// Skip UTF-8 validation for zero-copy (caller's responsibility)
	return ZeroCopyString{
		s:          s,
		generation: r.generation,
		reader:     r,
	}
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
//
// SAFETY WARNING: The returned slice points directly into the Reader's buffer.
// It is only valid under these conditions:
//   - The Reader must NOT be Reset() while the slice is in use
//   - The underlying data buffer must NOT be modified or freed
//   - The Reader must remain in scope
//   - You must NOT modify the returned slice
//
// Failure to observe these constraints will cause undefined behavior,
// including memory corruption, crashes, or data races.
//
// For safe usage, prefer ReadBytes() instead. Use ReadBytesNoCopy() only when:
//   - Performance is critical and profiling shows ReadBytes() as a bottleneck
//   - You can guarantee the Reader outlives all returned slices
//   - You will NOT call Reset() while slices are in use
//   - You will NOT store the returned slice beyond the current function scope
//   - You need read-only access to the data
//
// Example of UNSAFE usage (DO NOT DO THIS):
//
//	r := cramberry.NewReader(data)
//	b := r.ReadBytesNoCopy()
//	r.Reset(newData)  // UNDEFINED BEHAVIOR: b now points to invalid memory
//	fmt.Println(b)    // CRASH or data corruption
//
// Example of another UNSAFE pattern:
//
//	b := r.ReadBytesNoCopy()
//	b[0] = 'x'  // UNDEFINED BEHAVIOR: modifying shared buffer
func (r *Reader) ReadBytesNoCopy() ZeroCopyBytes {
	if !r.checkRead() {
		return ZeroCopyBytes{}
	}
	length := r.ReadUvarint()
	if r.err != nil {
		return ZeroCopyBytes{}
	}
	if length > uint64(MaxInt) {
		r.setErrorAt(ErrOverflow, "bytes length overflow")
		return ZeroCopyBytes{}
	}
	n := int(length)
	// Check limits
	if r.opts.Limits.MaxBytesLength > 0 && n > r.opts.Limits.MaxBytesLength {
		r.setError(ErrMaxBytesLength)
		return ZeroCopyBytes{}
	}
	if !r.ensure(n) {
		return ZeroCopyBytes{}
	}
	result := r.data[r.pos : r.pos+n]
	r.pos += n
	return ZeroCopyBytes{
		b:          result,
		generation: r.generation,
		reader:     r,
	}
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
//
// SAFETY WARNING: The returned slice points directly into the Reader's buffer.
// See ReadBytesNoCopy documentation for safety requirements.
// For safe usage, prefer ReadRawBytes() instead.
func (r *Reader) ReadRawBytesNoCopy(n int) ZeroCopyBytes {
	if n < 0 {
		r.setError(ErrNegativeLength)
		return ZeroCopyBytes{}
	}
	if !r.ensure(n) {
		return ZeroCopyBytes{}
	}
	result := r.data[r.pos : r.pos+n]
	r.pos += n
	return ZeroCopyBytes{
		b:          result,
		generation: r.generation,
		reader:     r,
	}
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

// ============================================================================
// Fast Packed Array Readers - Direct Memory Copy for Fixed-Size Types
// ============================================================================

// ReadPackedFloat32 reads a packed array of float32 values directly.
// This is faster than reading element by element for large arrays.
func (r *Reader) ReadPackedFloat32(count int) []float32 {
	if count <= 0 {
		return nil
	}
	// Overflow protection: check count before multiplication
	if count > MaxPackedFloat32Length {
		r.setError(ErrMaxArrayLength)
		return nil
	}
	byteSize := count * 4
	if !r.ensure(byteSize) {
		return nil
	}

	result := make([]float32, count)
	for i := 0; i < count; i++ {
		// Little-endian decode
		v := uint32(r.data[r.pos]) |
			uint32(r.data[r.pos+1])<<8 |
			uint32(r.data[r.pos+2])<<16 |
			uint32(r.data[r.pos+3])<<24
		result[i] = *(*float32)(unsafe.Pointer(&v))
		r.pos += 4
	}
	return result
}

// ReadPackedFloat64 reads a packed array of float64 values directly.
func (r *Reader) ReadPackedFloat64(count int) []float64 {
	if count <= 0 {
		return nil
	}
	// Overflow protection: check count before multiplication
	if count > MaxPackedFloat64Length {
		r.setError(ErrMaxArrayLength)
		return nil
	}
	byteSize := count * 8
	if !r.ensure(byteSize) {
		return nil
	}

	result := make([]float64, count)
	for i := 0; i < count; i++ {
		// Little-endian decode
		v := uint64(r.data[r.pos]) |
			uint64(r.data[r.pos+1])<<8 |
			uint64(r.data[r.pos+2])<<16 |
			uint64(r.data[r.pos+3])<<24 |
			uint64(r.data[r.pos+4])<<32 |
			uint64(r.data[r.pos+5])<<40 |
			uint64(r.data[r.pos+6])<<48 |
			uint64(r.data[r.pos+7])<<56
		result[i] = *(*float64)(unsafe.Pointer(&v))
		r.pos += 8
	}
	return result
}

// ReadPackedFixed32 reads a packed array of fixed 32-bit values directly.
func (r *Reader) ReadPackedFixed32(count int) []uint32 {
	if count <= 0 {
		return nil
	}
	// Overflow protection: check count before multiplication
	if count > MaxPackedFixed32Length {
		r.setError(ErrMaxArrayLength)
		return nil
	}
	byteSize := count * 4
	if !r.ensure(byteSize) {
		return nil
	}

	result := make([]uint32, count)
	for i := 0; i < count; i++ {
		// Little-endian decode
		result[i] = uint32(r.data[r.pos]) |
			uint32(r.data[r.pos+1])<<8 |
			uint32(r.data[r.pos+2])<<16 |
			uint32(r.data[r.pos+3])<<24
		r.pos += 4
	}
	return result
}

// ReadPackedFixed64 reads a packed array of fixed 64-bit values directly.
func (r *Reader) ReadPackedFixed64(count int) []uint64 {
	if count <= 0 {
		return nil
	}
	// Overflow protection: check count before multiplication
	if count > MaxPackedFixed64Length {
		r.setError(ErrMaxArrayLength)
		return nil
	}
	byteSize := count * 8
	if !r.ensure(byteSize) {
		return nil
	}

	result := make([]uint64, count)
	for i := 0; i < count; i++ {
		// Little-endian decode
		result[i] = uint64(r.data[r.pos]) |
			uint64(r.data[r.pos+1])<<8 |
			uint64(r.data[r.pos+2])<<16 |
			uint64(r.data[r.pos+3])<<24 |
			uint64(r.data[r.pos+4])<<32 |
			uint64(r.data[r.pos+5])<<40 |
			uint64(r.data[r.pos+6])<<48 |
			uint64(r.data[r.pos+7])<<56
		r.pos += 8
	}
	return result
}
