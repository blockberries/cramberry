package cramberry

import (
	"bufio"
	"io"
	"sync"

	"github.com/blockberries/cramberry/internal/wire"
)

// StreamWriter writes Cramberry-encoded data to an io.Writer.
// It buffers writes for efficiency and supports streaming multiple messages.
//
// StreamWriter is safe for concurrent use from a single goroutine,
// but not for use from multiple goroutines.
type StreamWriter struct {
	w      *bufio.Writer
	opts   Options
	depth  int
	err    error
	closed bool
	// scratch is used for encoding varints without allocation
	scratch [MaxVarintLen64]byte
}

// streamWriterPool provides pooled writers for reduced allocations.
var streamWriterPool = sync.Pool{
	New: func() any {
		return &StreamWriter{
			opts: DefaultOptions,
		}
	},
}

// NewStreamWriter creates a new StreamWriter that writes to w.
// The default buffer size is 4096 bytes.
func NewStreamWriter(w io.Writer) *StreamWriter {
	return NewStreamWriterSize(w, 4096)
}

// NewStreamWriterSize creates a new StreamWriter with a specified buffer size.
func NewStreamWriterSize(w io.Writer, bufSize int) *StreamWriter {
	return &StreamWriter{
		w:    bufio.NewWriterSize(w, bufSize),
		opts: DefaultOptions,
	}
}

// NewStreamWriterWithOptions creates a new StreamWriter with options.
func NewStreamWriterWithOptions(w io.Writer, opts Options) *StreamWriter {
	return &StreamWriter{
		w:    bufio.NewWriterSize(w, 4096),
		opts: opts,
	}
}

// GetStreamWriter gets a StreamWriter from the pool.
// Call PutStreamWriter to return it when done.
func GetStreamWriter(w io.Writer) *StreamWriter {
	sw := streamWriterPool.Get().(*StreamWriter)
	sw.Reset(w)
	return sw
}

// PutStreamWriter returns a StreamWriter to the pool.
func PutStreamWriter(sw *StreamWriter) {
	if sw == nil {
		return
	}
	sw.w = nil // Allow GC of the underlying writer
	streamWriterPool.Put(sw)
}

// Reset resets the StreamWriter to write to a new io.Writer.
func (sw *StreamWriter) Reset(w io.Writer) {
	if sw.w == nil {
		sw.w = bufio.NewWriterSize(w, 4096)
	} else {
		sw.w.Reset(w)
	}
	sw.depth = 0
	sw.err = nil
	sw.closed = false
}

// SetOptions updates the writer's options.
func (sw *StreamWriter) SetOptions(opts Options) {
	sw.opts = opts
}

// Options returns the writer's current options.
func (sw *StreamWriter) Options() Options {
	return sw.opts
}

// Flush writes any buffered data to the underlying writer.
func (sw *StreamWriter) Flush() error {
	if sw.err != nil {
		return sw.err
	}
	if err := sw.w.Flush(); err != nil {
		sw.err = NewEncodeError("flush failed", err)
		return sw.err
	}
	return nil
}

// Close flushes and releases resources.
// The underlying io.Writer is not closed.
func (sw *StreamWriter) Close() error {
	if sw.closed {
		return nil
	}
	sw.closed = true
	return sw.Flush()
}

// Err returns any error that occurred during writing.
func (sw *StreamWriter) Err() error {
	return sw.err
}

// setError records the first error.
func (sw *StreamWriter) setError(err error) {
	if sw.err == nil {
		sw.err = err
	}
}

// checkWrite ensures we can write.
func (sw *StreamWriter) checkWrite() bool {
	if sw.closed {
		sw.setError(NewEncodeError("writer is closed", nil))
		return false
	}
	return sw.err == nil
}

// write writes bytes to the buffer.
func (sw *StreamWriter) write(b []byte) {
	if !sw.checkWrite() {
		return
	}
	_, err := sw.w.Write(b)
	if err != nil {
		sw.setError(NewEncodeError("write failed", err))
	}
}

// writeByte writes a single byte.
func (sw *StreamWriter) writeByte(b byte) {
	if !sw.checkWrite() {
		return
	}
	if err := sw.w.WriteByte(b); err != nil {
		sw.setError(NewEncodeError("write failed", err))
	}
}

// WriteBool writes a boolean value.
func (sw *StreamWriter) WriteBool(v bool) {
	if v {
		sw.writeByte(1)
	} else {
		sw.writeByte(0)
	}
}

// WriteUint8 writes an unsigned 8-bit integer.
func (sw *StreamWriter) WriteUint8(v uint8) {
	sw.writeByte(v)
}

// WriteInt8 writes a signed 8-bit integer.
func (sw *StreamWriter) WriteInt8(v int8) {
	sw.writeByte(byte(v))
}

// WriteUint16 writes an unsigned 16-bit integer as a varint.
func (sw *StreamWriter) WriteUint16(v uint16) {
	sw.WriteUvarint(uint64(v))
}

// WriteUint32 writes an unsigned 32-bit integer as a varint.
func (sw *StreamWriter) WriteUint32(v uint32) {
	sw.WriteUvarint(uint64(v))
}

// WriteUint64 writes an unsigned 64-bit integer as a varint.
func (sw *StreamWriter) WriteUint64(v uint64) {
	sw.WriteUvarint(v)
}

// WriteUint writes an unsigned integer as a varint.
func (sw *StreamWriter) WriteUint(v uint) {
	sw.WriteUvarint(uint64(v))
}

// WriteInt16 writes a signed 16-bit integer as a signed varint.
func (sw *StreamWriter) WriteInt16(v int16) {
	sw.WriteSvarint(int64(v))
}

// WriteInt32 writes a signed 32-bit integer as a signed varint.
func (sw *StreamWriter) WriteInt32(v int32) {
	sw.WriteSvarint(int64(v))
}

// WriteInt64 writes a signed 64-bit integer as a signed varint.
func (sw *StreamWriter) WriteInt64(v int64) {
	sw.WriteSvarint(v)
}

// WriteInt writes a signed integer as a signed varint.
func (sw *StreamWriter) WriteInt(v int) {
	sw.WriteSvarint(int64(v))
}

// WriteUvarint writes an unsigned varint.
func (sw *StreamWriter) WriteUvarint(v uint64) {
	if !sw.checkWrite() {
		return
	}
	n := wire.AppendUvarint(sw.scratch[:0], v)
	sw.write(n)
}

// WriteSvarint writes a signed varint using ZigZag encoding.
func (sw *StreamWriter) WriteSvarint(v int64) {
	if !sw.checkWrite() {
		return
	}
	n := wire.AppendSvarint(sw.scratch[:0], v)
	sw.write(n)
}

// WriteFixed32 writes a fixed 32-bit value (little-endian).
func (sw *StreamWriter) WriteFixed32(v uint32) {
	if !sw.checkWrite() {
		return
	}
	n := wire.AppendFixed32(sw.scratch[:0], v)
	sw.write(n)
}

// WriteFixed64 writes a fixed 64-bit value (little-endian).
func (sw *StreamWriter) WriteFixed64(v uint64) {
	if !sw.checkWrite() {
		return
	}
	n := wire.AppendFixed64(sw.scratch[:0], v)
	sw.write(n)
}

// WriteFloat32 writes a 32-bit floating point number.
func (sw *StreamWriter) WriteFloat32(v float32) {
	if !sw.checkWrite() {
		return
	}
	n := wire.AppendFloat32(sw.scratch[:0], v)
	sw.write(n)
}

// WriteFloat64 writes a 64-bit floating point number.
func (sw *StreamWriter) WriteFloat64(v float64) {
	if !sw.checkWrite() {
		return
	}
	n := wire.AppendFloat64(sw.scratch[:0], v)
	sw.write(n)
}

// WriteComplex64 writes a complex64 value.
func (sw *StreamWriter) WriteComplex64(v complex64) {
	if !sw.checkWrite() {
		return
	}
	n := wire.AppendComplex64(sw.scratch[:0], v)
	sw.write(n)
}

// WriteComplex128 writes a complex128 value.
func (sw *StreamWriter) WriteComplex128(v complex128) {
	if !sw.checkWrite() {
		return
	}
	var buf [Complex128Size]byte
	n := wire.AppendComplex128(buf[:0], v)
	sw.write(n)
}

// WriteString writes a length-prefixed string.
func (sw *StreamWriter) WriteString(s string) {
	if !sw.checkWrite() {
		return
	}
	// Check limits
	if sw.opts.Limits.MaxStringLength > 0 && len(s) > sw.opts.Limits.MaxStringLength {
		sw.setError(ErrMaxStringLength)
		return
	}
	// Validate UTF-8 if required
	if sw.opts.ValidateUTF8 && !isValidUTF8(s) {
		sw.setError(ErrInvalidUTF8)
		return
	}
	sw.WriteUvarint(uint64(len(s)))
	if sw.err != nil {
		return
	}
	sw.write([]byte(s))
}

// WriteBytes writes a length-prefixed byte slice.
func (sw *StreamWriter) WriteBytes(b []byte) {
	if !sw.checkWrite() {
		return
	}
	// Check limits
	if sw.opts.Limits.MaxBytesLength > 0 && len(b) > sw.opts.Limits.MaxBytesLength {
		sw.setError(ErrMaxBytesLength)
		return
	}
	sw.WriteUvarint(uint64(len(b)))
	if sw.err != nil {
		return
	}
	sw.write(b)
}

// WriteRawBytes writes raw bytes without a length prefix.
func (sw *StreamWriter) WriteRawBytes(b []byte) {
	sw.write(b)
}

// WriteTag writes a field tag (field number + wire type).
func (sw *StreamWriter) WriteTag(fieldNum int, wireType WireType) {
	if !sw.checkWrite() {
		return
	}
	if fieldNum <= 0 {
		sw.setError(ErrInvalidFieldNumber)
		return
	}
	n := wire.AppendTag(sw.scratch[:0], fieldNum, wire.Type(wireType))
	sw.write(n)
}

// WriteNil writes a nil marker (TypeID 0).
func (sw *StreamWriter) WriteNil() {
	sw.WriteUvarint(uint64(TypeIDNil))
}

// WriteTypeID writes a type ID for polymorphic encoding.
func (sw *StreamWriter) WriteTypeID(id TypeID) {
	sw.WriteUvarint(uint64(id))
}

// WriteArrayHeader writes the length of an array/slice.
func (sw *StreamWriter) WriteArrayHeader(length int) {
	if !sw.checkWrite() {
		return
	}
	if length < 0 {
		sw.setError(ErrNegativeLength)
		return
	}
	if sw.opts.Limits.MaxArrayLength > 0 && length > sw.opts.Limits.MaxArrayLength {
		sw.setError(ErrMaxArrayLength)
		return
	}
	sw.WriteUvarint(uint64(length))
}

// WriteMapHeader writes the size of a map.
func (sw *StreamWriter) WriteMapHeader(size int) {
	if !sw.checkWrite() {
		return
	}
	if size < 0 {
		sw.setError(ErrNegativeLength)
		return
	}
	if sw.opts.Limits.MaxMapSize > 0 && size > sw.opts.Limits.MaxMapSize {
		sw.setError(ErrMaxMapSize)
		return
	}
	sw.WriteUvarint(uint64(size))
}

// WriteMessage writes a complete message with length prefix.
// This is useful for streaming multiple messages to the same writer.
func (sw *StreamWriter) WriteMessage(data []byte) {
	if !sw.checkWrite() {
		return
	}
	if sw.opts.Limits.MaxMessageSize > 0 && int64(len(data)) > sw.opts.Limits.MaxMessageSize {
		sw.setError(ErrMaxSizeExceeded)
		return
	}
	sw.WriteUvarint(uint64(len(data)))
	if sw.err != nil {
		return
	}
	sw.write(data)
}

// WriteDelimited writes a marshaled value with a length prefix.
// This enables streaming multiple messages to the same writer.
func (sw *StreamWriter) WriteDelimited(v any) error {
	if !sw.checkWrite() {
		return sw.err
	}
	data, err := Marshal(v)
	if err != nil {
		sw.setError(err)
		return err
	}
	sw.WriteMessage(data)
	return sw.err
}

// StreamReader reads Cramberry-encoded data from an io.Reader.
// It buffers reads for efficiency and supports streaming multiple messages.
//
// StreamReader is safe for concurrent use from a single goroutine,
// but not for use from multiple goroutines.
type StreamReader struct {
	r       *bufio.Reader
	opts    Options
	depth   int
	err     error
	scratch [MaxVarintLen64]byte
}

// streamReaderPool provides pooled readers for reduced allocations.
var streamReaderPool = sync.Pool{
	New: func() any {
		return &StreamReader{
			opts: DefaultOptions,
		}
	},
}

// NewStreamReader creates a new StreamReader that reads from r.
// The default buffer size is 4096 bytes.
func NewStreamReader(r io.Reader) *StreamReader {
	return NewStreamReaderSize(r, 4096)
}

// NewStreamReaderSize creates a new StreamReader with a specified buffer size.
func NewStreamReaderSize(r io.Reader, bufSize int) *StreamReader {
	return &StreamReader{
		r:    bufio.NewReaderSize(r, bufSize),
		opts: DefaultOptions,
	}
}

// NewStreamReaderWithOptions creates a new StreamReader with options.
func NewStreamReaderWithOptions(r io.Reader, opts Options) *StreamReader {
	return &StreamReader{
		r:    bufio.NewReaderSize(r, 4096),
		opts: opts,
	}
}

// GetStreamReader gets a StreamReader from the pool.
// Call PutStreamReader to return it when done.
func GetStreamReader(r io.Reader) *StreamReader {
	sr := streamReaderPool.Get().(*StreamReader)
	sr.Reset(r)
	return sr
}

// PutStreamReader returns a StreamReader to the pool.
func PutStreamReader(sr *StreamReader) {
	if sr == nil {
		return
	}
	sr.r = nil // Allow GC of the underlying reader
	streamReaderPool.Put(sr)
}

// Reset resets the StreamReader to read from a new io.Reader.
func (sr *StreamReader) Reset(r io.Reader) {
	if sr.r == nil {
		sr.r = bufio.NewReaderSize(r, 4096)
	} else {
		sr.r.Reset(r)
	}
	sr.depth = 0
	sr.err = nil
}

// SetOptions updates the reader's options.
func (sr *StreamReader) SetOptions(opts Options) {
	sr.opts = opts
}

// Options returns the reader's current options.
func (sr *StreamReader) Options() Options {
	return sr.opts
}

// Err returns any error that occurred during reading.
func (sr *StreamReader) Err() error {
	return sr.err
}

// setError records the first error.
func (sr *StreamReader) setError(err error) {
	if sr.err == nil {
		sr.err = err
	}
}

// checkRead ensures we can read.
func (sr *StreamReader) checkRead() bool {
	return sr.err == nil
}

// readFull reads exactly len(b) bytes.
func (sr *StreamReader) readFull(b []byte) bool {
	if !sr.checkRead() {
		return false
	}
	_, err := io.ReadFull(sr.r, b)
	if err != nil {
		if err == io.EOF {
			sr.setError(ErrUnexpectedEOF)
		} else {
			sr.setError(NewDecodeError("read failed", err))
		}
		return false
	}
	return true
}

// readByte reads a single byte.
func (sr *StreamReader) readByte() (byte, bool) {
	if !sr.checkRead() {
		return 0, false
	}
	b, err := sr.r.ReadByte()
	if err != nil {
		if err == io.EOF {
			sr.setError(ErrUnexpectedEOF)
		} else {
			sr.setError(NewDecodeError("read failed", err))
		}
		return 0, false
	}
	return b, true
}

// ReadBool reads a boolean value.
func (sr *StreamReader) ReadBool() bool {
	b, ok := sr.readByte()
	if !ok {
		return false
	}
	return b != 0
}

// ReadUint8 reads an unsigned 8-bit integer.
func (sr *StreamReader) ReadUint8() uint8 {
	b, _ := sr.readByte()
	return b
}

// ReadInt8 reads a signed 8-bit integer.
func (sr *StreamReader) ReadInt8() int8 {
	b, _ := sr.readByte()
	return int8(b)
}

// ReadUvarint reads an unsigned varint.
func (sr *StreamReader) ReadUvarint() uint64 {
	if !sr.checkRead() {
		return 0
	}
	// Read varint byte by byte
	var result uint64
	var shift uint
	for i := 0; i < MaxVarintLen64; i++ {
		b, err := sr.r.ReadByte()
		if err != nil {
			if err == io.EOF {
				sr.setError(ErrUnexpectedEOF)
			} else {
				sr.setError(NewDecodeError("read varint failed", err))
			}
			return 0
		}
		if i == 9 && b > 1 {
			// 10th byte can only be 0 or 1 for valid uint64
			sr.setError(wire.ErrVarintOverflow)
			return 0
		}
		result |= uint64(b&0x7F) << shift
		if b < 0x80 {
			return result
		}
		shift += 7
	}
	sr.setError(wire.ErrVarintTooLong)
	return 0
}

// ReadSvarint reads a signed varint (ZigZag encoded).
func (sr *StreamReader) ReadSvarint() int64 {
	v := sr.ReadUvarint()
	if sr.err != nil {
		return 0
	}
	// ZigZag decode
	return int64((v >> 1) ^ -(v & 1))
}

// ReadUint16 reads an unsigned 16-bit integer.
func (sr *StreamReader) ReadUint16() uint16 {
	v := sr.ReadUvarint()
	if sr.err != nil {
		return 0
	}
	if v > 0xFFFF {
		sr.setError(ErrOverflow)
		return 0
	}
	return uint16(v)
}

// ReadUint32 reads an unsigned 32-bit integer.
func (sr *StreamReader) ReadUint32() uint32 {
	v := sr.ReadUvarint()
	if sr.err != nil {
		return 0
	}
	if v > 0xFFFFFFFF {
		sr.setError(ErrOverflow)
		return 0
	}
	return uint32(v)
}

// ReadUint64 reads an unsigned 64-bit integer.
func (sr *StreamReader) ReadUint64() uint64 {
	return sr.ReadUvarint()
}

// ReadUint reads an unsigned integer.
func (sr *StreamReader) ReadUint() uint {
	return uint(sr.ReadUvarint())
}

// ReadInt16 reads a signed 16-bit integer.
func (sr *StreamReader) ReadInt16() int16 {
	v := sr.ReadSvarint()
	if sr.err != nil {
		return 0
	}
	if v < -32768 || v > 32767 {
		sr.setError(ErrOverflow)
		return 0
	}
	return int16(v)
}

// ReadInt32 reads a signed 32-bit integer.
func (sr *StreamReader) ReadInt32() int32 {
	v := sr.ReadSvarint()
	if sr.err != nil {
		return 0
	}
	if v < -2147483648 || v > 2147483647 {
		sr.setError(ErrOverflow)
		return 0
	}
	return int32(v)
}

// ReadInt64 reads a signed 64-bit integer.
func (sr *StreamReader) ReadInt64() int64 {
	return sr.ReadSvarint()
}

// ReadInt reads a signed integer.
func (sr *StreamReader) ReadInt() int {
	return int(sr.ReadSvarint())
}

// ReadFixed32 reads a fixed 32-bit value (little-endian).
func (sr *StreamReader) ReadFixed32() uint32 {
	if !sr.readFull(sr.scratch[:Fixed32Size]) {
		return 0
	}
	v, _ := wire.DecodeFixed32(sr.scratch[:Fixed32Size])
	return v
}

// ReadFixed64 reads a fixed 64-bit value (little-endian).
func (sr *StreamReader) ReadFixed64() uint64 {
	if !sr.readFull(sr.scratch[:Fixed64Size]) {
		return 0
	}
	v, _ := wire.DecodeFixed64(sr.scratch[:Fixed64Size])
	return v
}

// ReadFloat32 reads a 32-bit floating point number.
func (sr *StreamReader) ReadFloat32() float32 {
	if !sr.readFull(sr.scratch[:Float32Size]) {
		return 0
	}
	v, _ := wire.DecodeFloat32(sr.scratch[:Float32Size])
	return v
}

// ReadFloat64 reads a 64-bit floating point number.
func (sr *StreamReader) ReadFloat64() float64 {
	if !sr.readFull(sr.scratch[:Float64Size]) {
		return 0
	}
	v, _ := wire.DecodeFloat64(sr.scratch[:Float64Size])
	return v
}

// ReadComplex64 reads a complex64 value.
func (sr *StreamReader) ReadComplex64() complex64 {
	if !sr.readFull(sr.scratch[:Complex64Size]) {
		return 0
	}
	v, _ := wire.DecodeComplex64(sr.scratch[:Complex64Size])
	return v
}

// ReadComplex128 reads a complex128 value.
func (sr *StreamReader) ReadComplex128() complex128 {
	var buf [Complex128Size]byte
	if !sr.readFull(buf[:]) {
		return 0
	}
	v, _ := wire.DecodeComplex128(buf[:])
	return v
}

// ReadString reads a length-prefixed string.
func (sr *StreamReader) ReadString() string {
	length := sr.ReadUvarint()
	if sr.err != nil {
		return ""
	}
	if length > uint64(MaxInt) {
		sr.setError(ErrOverflow)
		return ""
	}
	n := int(length)
	// Check limits
	if sr.opts.Limits.MaxStringLength > 0 && n > sr.opts.Limits.MaxStringLength {
		sr.setError(ErrMaxStringLength)
		return ""
	}
	// Read string data
	buf := make([]byte, n)
	if !sr.readFull(buf) {
		return ""
	}
	s := string(buf)
	// Validate UTF-8 if required
	if sr.opts.ValidateUTF8 && !isValidUTF8(s) {
		sr.setError(ErrInvalidUTF8)
		return ""
	}
	return s
}

// ReadBytes reads a length-prefixed byte slice.
func (sr *StreamReader) ReadBytes() []byte {
	length := sr.ReadUvarint()
	if sr.err != nil {
		return nil
	}
	if length > uint64(MaxInt) {
		sr.setError(ErrOverflow)
		return nil
	}
	n := int(length)
	// Check limits
	if sr.opts.Limits.MaxBytesLength > 0 && n > sr.opts.Limits.MaxBytesLength {
		sr.setError(ErrMaxBytesLength)
		return nil
	}
	// Read byte data
	buf := make([]byte, n)
	if !sr.readFull(buf) {
		return nil
	}
	return buf
}

// ReadRawBytes reads exactly n bytes without a length prefix.
func (sr *StreamReader) ReadRawBytes(n int) []byte {
	if n < 0 {
		sr.setError(ErrNegativeLength)
		return nil
	}
	buf := make([]byte, n)
	if !sr.readFull(buf) {
		return nil
	}
	return buf
}

// ReadTag reads a field tag (field number + wire type).
func (sr *StreamReader) ReadTag() (fieldNum int, wireType WireType) {
	v := sr.ReadUvarint()
	if sr.err != nil {
		return 0, 0
	}
	fieldNum = int(v >> 3)
	wireType = WireType(v & 7)
	if fieldNum <= 0 {
		sr.setError(ErrInvalidFieldNumber)
		return 0, 0
	}
	return fieldNum, wireType
}

// ReadTypeID reads a type ID for polymorphic decoding.
func (sr *StreamReader) ReadTypeID() TypeID {
	v := sr.ReadUvarint()
	if sr.err != nil {
		return TypeIDNil
	}
	return TypeID(v)
}

// ReadArrayHeader reads the length of an array/slice.
func (sr *StreamReader) ReadArrayHeader() int {
	length := sr.ReadUvarint()
	if sr.err != nil {
		return 0
	}
	if length > uint64(MaxInt) {
		sr.setError(ErrOverflow)
		return 0
	}
	n := int(length)
	// Check limits
	if sr.opts.Limits.MaxArrayLength > 0 && n > sr.opts.Limits.MaxArrayLength {
		sr.setError(ErrMaxArrayLength)
		return 0
	}
	return n
}

// ReadMapHeader reads the size of a map.
func (sr *StreamReader) ReadMapHeader() int {
	size := sr.ReadUvarint()
	if sr.err != nil {
		return 0
	}
	if size > uint64(MaxInt) {
		sr.setError(ErrOverflow)
		return 0
	}
	n := int(size)
	// Check limits
	if sr.opts.Limits.MaxMapSize > 0 && n > sr.opts.Limits.MaxMapSize {
		sr.setError(ErrMaxMapSize)
		return 0
	}
	return n
}

// ReadMessage reads a length-prefixed message and returns the raw bytes.
// This is useful for streaming multiple messages from the same reader.
func (sr *StreamReader) ReadMessage() []byte {
	length := sr.ReadUvarint()
	if sr.err != nil {
		return nil
	}
	if length > uint64(MaxInt) {
		sr.setError(ErrOverflow)
		return nil
	}
	n := int(length)
	// Check limits
	if sr.opts.Limits.MaxMessageSize > 0 && int64(n) > sr.opts.Limits.MaxMessageSize {
		sr.setError(ErrMaxSizeExceeded)
		return nil
	}
	// Read message data
	buf := make([]byte, n)
	if !sr.readFull(buf) {
		return nil
	}
	return buf
}

// ReadDelimited reads a length-prefixed message and unmarshals it.
// This enables streaming multiple messages from the same reader.
func (sr *StreamReader) ReadDelimited(v any) error {
	data := sr.ReadMessage()
	if sr.err != nil {
		return sr.err
	}
	return Unmarshal(data, v)
}

// SkipMessage skips a length-prefixed message without reading its contents.
func (sr *StreamReader) SkipMessage() {
	length := sr.ReadUvarint()
	if sr.err != nil {
		return
	}
	if length > uint64(MaxInt) {
		sr.setError(ErrOverflow)
		return
	}
	n := int(length)
	// Discard n bytes
	discarded, err := sr.r.Discard(n)
	if err != nil || discarded < n {
		sr.setError(NewDecodeError("skip message failed", err))
	}
}

// Peek returns the next n bytes without advancing the reader.
// The returned bytes are only valid until the next read call.
func (sr *StreamReader) Peek(n int) ([]byte, error) {
	if !sr.checkRead() {
		return nil, sr.err
	}
	return sr.r.Peek(n)
}

// Buffered returns the number of bytes available in the buffer.
func (sr *StreamReader) Buffered() int {
	return sr.r.Buffered()
}

// MessageIterator provides an iterator for reading delimited messages.
type MessageIterator struct {
	reader *StreamReader
	err    error
}

// NewMessageIterator creates an iterator for reading delimited messages.
func NewMessageIterator(r io.Reader) *MessageIterator {
	return &MessageIterator{
		reader: NewStreamReader(r),
	}
}

// Next reads the next message and returns true if successful.
// Returns false on EOF or error.
func (it *MessageIterator) Next(v any) bool {
	// Check for buffered data first
	if it.reader.Buffered() == 0 {
		// Try to peek to detect EOF
		_, err := it.reader.Peek(1)
		if err == io.EOF {
			return false
		}
	}
	err := it.reader.ReadDelimited(v)
	if err != nil {
		if err == ErrUnexpectedEOF && it.reader.Buffered() == 0 {
			// Clean EOF
			return false
		}
		it.err = err
		return false
	}
	return true
}

// Err returns any error that occurred during iteration.
func (it *MessageIterator) Err() error {
	return it.err
}
