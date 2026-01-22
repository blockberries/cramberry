package cramberry

import (
	"bytes"
	"math"
	"testing"
)

func TestReaderBasic(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewReader(data)

	if r.Len() != 5 {
		t.Errorf("Len() = %d, want 5", r.Len())
	}
	if r.Pos() != 0 {
		t.Errorf("Pos() = %d, want 0", r.Pos())
	}
	if !bytes.Equal(r.Data(), data) {
		t.Error("Data() mismatch")
	}
	if r.EOF() {
		t.Error("EOF() should be false initially")
	}
	if r.Err() != nil {
		t.Errorf("Err() = %v, want nil", r.Err())
	}
}

func TestReaderWithOptions(t *testing.T) {
	r := NewReaderWithOptions([]byte{1, 2, 3}, SecureOptions)
	opts := r.Options()
	if opts.Limits.MaxMessageSize != SecureLimits.MaxMessageSize {
		t.Error("Options not set correctly")
	}
}

func TestReaderReset(t *testing.T) {
	r := NewReader([]byte{1, 2, 3})
	r.ReadUint8()
	r.Reset([]byte{4, 5, 6, 7})

	if r.Pos() != 0 {
		t.Errorf("Pos() after Reset = %d, want 0", r.Pos())
	}
	if r.Len() != 4 {
		t.Errorf("Len() after Reset = %d, want 4", r.Len())
	}
}

func TestReaderSetOptions(t *testing.T) {
	r := NewReader([]byte{})
	r.SetOptions(SecureOptions)
	if r.Options().Limits.MaxMessageSize != SecureLimits.MaxMessageSize {
		t.Error("SetOptions did not update options")
	}
}

func TestReadBool(t *testing.T) {
	tests := []struct {
		data     []byte
		expected bool
	}{
		{[]byte{0}, false},
		{[]byte{1}, true},
		{[]byte{2}, true},  // Non-zero is true
		{[]byte{255}, true},
	}

	for _, tc := range tests {
		r := NewReader(tc.data)
		got := r.ReadBool()
		if r.Err() != nil {
			t.Errorf("ReadBool(%v) error: %v", tc.data, r.Err())
		}
		if got != tc.expected {
			t.Errorf("ReadBool(%v) = %v, want %v", tc.data, got, tc.expected)
		}
	}
}

func TestReadUint8(t *testing.T) {
	tests := []uint8{0, 1, 127, 128, 255}
	for _, v := range tests {
		r := NewReader([]byte{v})
		got := r.ReadUint8()
		if r.Err() != nil {
			t.Errorf("ReadUint8 error: %v", r.Err())
		}
		if got != v {
			t.Errorf("ReadUint8 = %d, want %d", got, v)
		}
	}
}

func TestReadInt8(t *testing.T) {
	tests := []int8{-128, -1, 0, 1, 127}
	for _, v := range tests {
		r := NewReader([]byte{byte(v)})
		got := r.ReadInt8()
		if r.Err() != nil {
			t.Errorf("ReadInt8 error: %v", r.Err())
		}
		if got != v {
			t.Errorf("ReadInt8 = %d, want %d", got, v)
		}
	}
}

func TestReadUvarint(t *testing.T) {
	tests := []struct {
		data     []byte
		expected uint64
	}{
		{[]byte{0x00}, 0},
		{[]byte{0x01}, 1},
		{[]byte{0x7F}, 127},
		{[]byte{0x80, 0x01}, 128},
		{[]byte{0xFF, 0x01}, 255},
		{[]byte{0xAC, 0x02}, 300},
		{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}, math.MaxUint64},
	}

	for _, tc := range tests {
		r := NewReader(tc.data)
		got := r.ReadUvarint()
		if r.Err() != nil {
			t.Errorf("ReadUvarint(%v) error: %v", tc.data, r.Err())
		}
		if got != tc.expected {
			t.Errorf("ReadUvarint(%v) = %d, want %d", tc.data, got, tc.expected)
		}
	}
}

func TestReadSvarint(t *testing.T) {
	tests := []struct {
		data     []byte
		expected int64
	}{
		{[]byte{0x00}, 0},
		{[]byte{0x01}, -1},
		{[]byte{0x02}, 1},
		{[]byte{0x03}, -2},
		{[]byte{0x04}, 2},
		{[]byte{0x7E}, 63},
		{[]byte{0x7F}, -64},
	}

	for _, tc := range tests {
		r := NewReader(tc.data)
		got := r.ReadSvarint()
		if r.Err() != nil {
			t.Errorf("ReadSvarint(%v) error: %v", tc.data, r.Err())
		}
		if got != tc.expected {
			t.Errorf("ReadSvarint(%v) = %d, want %d", tc.data, got, tc.expected)
		}
	}
}

func TestReadUint(t *testing.T) {
	// Write values and read them back
	w := NewWriter()
	w.WriteUint16(300)
	w.WriteUint32(70000)
	w.WriteUint64(1000000)
	w.WriteUint(42)

	r := NewReader(w.Bytes())
	if r.ReadUint16() != 300 {
		t.Error("ReadUint16 mismatch")
	}
	if r.ReadUint32() != 70000 {
		t.Error("ReadUint32 mismatch")
	}
	if r.ReadUint64() != 1000000 {
		t.Error("ReadUint64 mismatch")
	}
	if r.ReadUint() != 42 {
		t.Error("ReadUint mismatch")
	}
	if r.Err() != nil {
		t.Errorf("Unexpected error: %v", r.Err())
	}
}

func TestReadInt(t *testing.T) {
	w := NewWriter()
	w.WriteInt16(-300)
	w.WriteInt32(-70000)
	w.WriteInt64(-1000000)
	w.WriteInt(-42)

	r := NewReader(w.Bytes())
	if r.ReadInt16() != -300 {
		t.Error("ReadInt16 mismatch")
	}
	if r.ReadInt32() != -70000 {
		t.Error("ReadInt32 mismatch")
	}
	if r.ReadInt64() != -1000000 {
		t.Error("ReadInt64 mismatch")
	}
	if r.ReadInt() != -42 {
		t.Error("ReadInt mismatch")
	}
	if r.Err() != nil {
		t.Errorf("Unexpected error: %v", r.Err())
	}
}

func TestReadFixed32(t *testing.T) {
	tests := []struct {
		data     []byte
		expected uint32
	}{
		{[]byte{0, 0, 0, 0}, 0},
		{[]byte{1, 0, 0, 0}, 1},
		{[]byte{0x78, 0x56, 0x34, 0x12}, 0x12345678},
		{[]byte{0xFF, 0xFF, 0xFF, 0xFF}, math.MaxUint32},
	}

	for _, tc := range tests {
		r := NewReader(tc.data)
		got := r.ReadFixed32()
		if r.Err() != nil {
			t.Errorf("ReadFixed32 error: %v", r.Err())
		}
		if got != tc.expected {
			t.Errorf("ReadFixed32(%v) = %d, want %d", tc.data, got, tc.expected)
		}
	}
}

func TestReadFixed64(t *testing.T) {
	tests := []struct {
		data     []byte
		expected uint64
	}{
		{[]byte{0, 0, 0, 0, 0, 0, 0, 0}, 0},
		{[]byte{1, 0, 0, 0, 0, 0, 0, 0}, 1},
		{[]byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}, 0x123456789ABCDEF0},
	}

	for _, tc := range tests {
		r := NewReader(tc.data)
		got := r.ReadFixed64()
		if r.Err() != nil {
			t.Errorf("ReadFixed64 error: %v", r.Err())
		}
		if got != tc.expected {
			t.Errorf("ReadFixed64(%v) = %d, want %d", tc.data, got, tc.expected)
		}
	}
}

func TestReadSFixed32(t *testing.T) {
	tests := []struct {
		value int32
	}{
		{0},
		{1},
		{-1},
		{math.MinInt32},
		{math.MaxInt32},
		{-12345},
		{12345},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteSFixed32(tc.value)

		r := NewReader(w.Bytes())
		got := r.ReadSFixed32()
		if r.Err() != nil {
			t.Errorf("ReadSFixed32 error: %v", r.Err())
		}
		if got != tc.value {
			t.Errorf("ReadSFixed32 = %d, want %d", got, tc.value)
		}
	}
}

func TestReadSFixed64(t *testing.T) {
	tests := []struct {
		value int64
	}{
		{0},
		{1},
		{-1},
		{math.MinInt64},
		{math.MaxInt64},
		{-123456789012345},
		{123456789012345},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteSFixed64(tc.value)

		r := NewReader(w.Bytes())
		got := r.ReadSFixed64()
		if r.Err() != nil {
			t.Errorf("ReadSFixed64 error: %v", r.Err())
		}
		if got != tc.value {
			t.Errorf("ReadSFixed64 = %d, want %d", got, tc.value)
		}
	}
}

func TestReadFloat(t *testing.T) {
	// Test round-trip
	w := NewWriter()
	w.WriteFloat32(3.14)
	w.WriteFloat64(3.14159265359)

	r := NewReader(w.Bytes())
	f32 := r.ReadFloat32()
	f64 := r.ReadFloat64()

	if r.Err() != nil {
		t.Errorf("ReadFloat error: %v", r.Err())
	}
	if f32 != 3.14 {
		t.Errorf("ReadFloat32 = %v, want 3.14", f32)
	}
	if f64 != 3.14159265359 {
		t.Errorf("ReadFloat64 = %v, want 3.14159265359", f64)
	}
}

func TestReadComplex(t *testing.T) {
	w := NewWriter()
	w.WriteComplex64(1 + 2i)
	w.WriteComplex128(3 + 4i)

	r := NewReader(w.Bytes())
	c64 := r.ReadComplex64()
	c128 := r.ReadComplex128()

	if r.Err() != nil {
		t.Errorf("ReadComplex error: %v", r.Err())
	}
	if c64 != 1+2i {
		t.Errorf("ReadComplex64 = %v, want 1+2i", c64)
	}
	if c128 != 3+4i {
		t.Errorf("ReadComplex128 = %v, want 3+4i", c128)
	}
}

func TestReadString(t *testing.T) {
	tests := []string{
		"",
		"hello",
		"hello, world!",
		"日本語",
		"emoji: 🎉",
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteString(tc)

		r := NewReader(w.Bytes())
		got := r.ReadString()
		if r.Err() != nil {
			t.Errorf("ReadString(%q) error: %v", tc, r.Err())
		}
		if got != tc {
			t.Errorf("ReadString = %q, want %q", got, tc)
		}
	}
}

func TestReadStringInvalidUTF8(t *testing.T) {
	// Create a reader with invalid UTF-8 string
	w := NewWriterWithOptions(Options{ValidateUTF8: false})
	w.WriteString("\xff\xfe")

	r := NewReaderWithOptions(w.Bytes(), Options{ValidateUTF8: true})
	_ = r.ReadString()
	if r.Err() == nil {
		t.Error("ReadString with invalid UTF-8 should fail when ValidateUTF8 is true")
	}

	// Without validation, should succeed
	r2 := NewReaderWithOptions(w.Bytes(), Options{ValidateUTF8: false})
	s := r2.ReadString()
	if r2.Err() != nil {
		t.Errorf("ReadString with ValidateUTF8=false should not fail: %v", r2.Err())
	}
	if s != "\xff\xfe" {
		t.Errorf("ReadString = %q, want %q", s, "\xff\xfe")
	}
}

func TestReadBytes(t *testing.T) {
	tests := [][]byte{
		{},
		{1, 2, 3},
		make([]byte, 100),
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteBytes(tc)

		r := NewReader(w.Bytes())
		got := r.ReadBytes()
		if r.Err() != nil {
			t.Errorf("ReadBytes error: %v", r.Err())
		}
		if !bytes.Equal(got, tc) {
			t.Errorf("ReadBytes mismatch")
		}
	}
}

func TestReadBytesNoCopy(t *testing.T) {
	w := NewWriter()
	w.WriteBytes([]byte{1, 2, 3, 4, 5})
	data := w.BytesCopy()

	r := NewReader(data)
	got := r.ReadBytesNoCopy()
	if r.Err() != nil {
		t.Errorf("ReadBytesNoCopy error: %v", r.Err())
	}
	if !bytes.Equal(got, []byte{1, 2, 3, 4, 5}) {
		t.Error("ReadBytesNoCopy mismatch")
	}
}

func TestReadRawBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewReader(data)
	got := r.ReadRawBytes(3)
	if r.Err() != nil {
		t.Errorf("ReadRawBytes error: %v", r.Err())
	}
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Errorf("ReadRawBytes = %v, want [1,2,3]", got)
	}
	if r.Pos() != 3 {
		t.Errorf("Pos() = %d, want 3", r.Pos())
	}
}

func TestReadRawBytesNoCopy(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewReader(data)
	got := r.ReadRawBytesNoCopy(3)
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Error("ReadRawBytesNoCopy mismatch")
	}
}

func TestReadRawBytesNegative(t *testing.T) {
	r := NewReader([]byte{1, 2, 3})
	_ = r.ReadRawBytes(-1)
	if r.Err() == nil {
		t.Error("ReadRawBytes with negative length should fail")
	}
}

func TestReadTag(t *testing.T) {
	tests := []struct {
		fieldNum int
		wireType WireType
	}{
		{1, WireVarint},
		{2, WireFixed64},
		{15, WireBytes},
		{16, WireFixed32},
		{100, WireSVarint},
		{1000, WireTypeRef},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteTag(tc.fieldNum, tc.wireType)

		r := NewReader(w.Bytes())
		fn, wt := r.ReadTag()
		if r.Err() != nil {
			t.Errorf("ReadTag error: %v", r.Err())
		}
		if fn != tc.fieldNum || wt != tc.wireType {
			t.Errorf("ReadTag = (%d, %d), want (%d, %d)", fn, wt, tc.fieldNum, tc.wireType)
		}
	}
}

func TestReadTypeID(t *testing.T) {
	w := NewWriter()
	w.WriteTypeID(TypeID(128))

	r := NewReader(w.Bytes())
	id := r.ReadTypeID()
	if r.Err() != nil {
		t.Errorf("ReadTypeID error: %v", r.Err())
	}
	if id != 128 {
		t.Errorf("ReadTypeID = %d, want 128", id)
	}
}

func TestReaderBeginEndMessage(t *testing.T) {
	// Create a message
	w := NewWriter()
	cp := w.BeginMessage()
	w.WriteInt32(42)
	w.WriteString("hello")
	w.EndMessage(cp)

	// Read it back
	r := NewReader(w.Bytes())
	endPos := r.BeginMessage()
	if r.Err() != nil {
		t.Fatalf("BeginMessage error: %v", r.Err())
	}

	v := r.ReadInt32()
	s := r.ReadString()
	r.EndMessage(endPos)

	if r.Err() != nil {
		t.Fatalf("Message reading error: %v", r.Err())
	}
	if v != 42 {
		t.Errorf("v = %d, want 42", v)
	}
	if s != "hello" {
		t.Errorf("s = %q, want %q", s, "hello")
	}
}

func TestReaderNestedMessages(t *testing.T) {
	// Create nested messages
	w := NewWriter()
	outer := w.BeginMessage()
	w.WriteInt32(1)
	inner := w.BeginMessage()
	w.WriteString("nested")
	w.EndMessage(inner)
	w.WriteInt32(2)
	w.EndMessage(outer)

	// Read them back
	r := NewReader(w.Bytes())
	outerEnd := r.BeginMessage()
	v1 := r.ReadInt32()
	innerEnd := r.BeginMessage()
	s := r.ReadString()
	r.EndMessage(innerEnd)
	v2 := r.ReadInt32()
	r.EndMessage(outerEnd)

	if r.Err() != nil {
		t.Fatalf("Nested message reading error: %v", r.Err())
	}
	if v1 != 1 || v2 != 2 {
		t.Errorf("v1=%d, v2=%d, want 1, 2", v1, v2)
	}
	if s != "nested" {
		t.Errorf("s = %q, want %q", s, "nested")
	}
}

func TestReadArrayHeader(t *testing.T) {
	w := NewWriter()
	w.WriteArrayHeader(10)

	r := NewReader(w.Bytes())
	n := r.ReadArrayHeader()
	if r.Err() != nil {
		t.Errorf("ReadArrayHeader error: %v", r.Err())
	}
	if n != 10 {
		t.Errorf("ReadArrayHeader = %d, want 10", n)
	}
}

func TestReadMapHeader(t *testing.T) {
	w := NewWriter()
	w.WriteMapHeader(10)

	r := NewReader(w.Bytes())
	n := r.ReadMapHeader()
	if r.Err() != nil {
		t.Errorf("ReadMapHeader error: %v", r.Err())
	}
	if n != 10 {
		t.Errorf("ReadMapHeader = %d, want 10", n)
	}
}

func TestReaderLimits(t *testing.T) {
	t.Run("MaxStringLength", func(t *testing.T) {
		w := NewWriter()
		w.WriteString("hello!")

		r := NewReaderWithOptions(w.Bytes(), Options{
			Limits: Limits{MaxStringLength: 5},
		})
		_ = r.ReadString()
		if r.Err() == nil {
			t.Error("String over limit should fail")
		}
	})

	t.Run("MaxBytesLength", func(t *testing.T) {
		w := NewWriter()
		w.WriteBytes([]byte{1, 2, 3, 4, 5, 6})

		r := NewReaderWithOptions(w.Bytes(), Options{
			Limits: Limits{MaxBytesLength: 5},
		})
		_ = r.ReadBytes()
		if r.Err() == nil {
			t.Error("Bytes over limit should fail")
		}
	})

	t.Run("MaxArrayLength", func(t *testing.T) {
		w := NewWriter()
		w.WriteArrayHeader(11)

		r := NewReaderWithOptions(w.Bytes(), Options{
			Limits: Limits{MaxArrayLength: 10},
		})
		_ = r.ReadArrayHeader()
		if r.Err() == nil {
			t.Error("Array over limit should fail")
		}
	})

	t.Run("MaxMapSize", func(t *testing.T) {
		w := NewWriter()
		w.WriteMapHeader(11)

		r := NewReaderWithOptions(w.Bytes(), Options{
			Limits: Limits{MaxMapSize: 10},
		})
		_ = r.ReadMapHeader()
		if r.Err() == nil {
			t.Error("Map over limit should fail")
		}
	})

	t.Run("MaxDepth", func(t *testing.T) {
		// Create deeply nested messages
		w := NewWriter()
		c1 := w.BeginMessage()
		c2 := w.BeginMessage()
		w.WriteInt32(42)
		w.EndMessage(c2)
		w.EndMessage(c1)

		r := NewReaderWithOptions(w.Bytes(), Options{
			Limits: Limits{MaxDepth: 1},
		})
		_ = r.BeginMessage()
		_ = r.BeginMessage() // Should fail
		if r.Err() == nil {
			t.Error("Exceeding max depth should fail")
		}
	})

	t.Run("MaxMessageSize", func(t *testing.T) {
		// Create a message claiming to be very large
		w := NewWriter()
		// Write a length that exceeds the limit
		w.WriteUvarint(1000000)

		r := NewReaderWithOptions(w.Bytes(), Options{
			Limits: Limits{MaxMessageSize: 1000},
		})
		_ = r.BeginMessage()
		if r.Err() == nil {
			t.Error("Message over size limit should fail")
		}
	})
}

func TestReaderUnexpectedEOF(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*Reader)
	}{
		{"ReadBool", func(r *Reader) { r.ReadBool() }},
		{"ReadUint8", func(r *Reader) { r.ReadUint8() }},
		{"ReadFixed32", func(r *Reader) { r.ReadFixed32() }},
		{"ReadFixed64", func(r *Reader) { r.ReadFixed64() }},
		{"ReadFloat32", func(r *Reader) { r.ReadFloat32() }},
		{"ReadFloat64", func(r *Reader) { r.ReadFloat64() }},
		{"ReadComplex64", func(r *Reader) { r.ReadComplex64() }},
		{"ReadComplex128", func(r *Reader) { r.ReadComplex128() }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewReader([]byte{})
			tc.fn(r)
			if r.Err() == nil {
				t.Errorf("%s on empty data should fail", tc.name)
			}
		})
	}
}

func TestReaderSkip(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewReader(data)
	r.Skip(2)
	if r.Pos() != 2 {
		t.Errorf("Pos() after Skip(2) = %d, want 2", r.Pos())
	}
	v := r.ReadUint8()
	if v != 3 {
		t.Errorf("ReadUint8 after Skip = %d, want 3", v)
	}
}

func TestReaderSkipPastEOF(t *testing.T) {
	r := NewReader([]byte{1, 2, 3})
	r.Skip(10)
	if r.Err() == nil {
		t.Error("Skip past EOF should fail")
	}
}

func TestSkipValue(t *testing.T) {
	tests := []struct {
		name     string
		wireType WireType
		setup    func(*Writer)
	}{
		{"Varint", WireVarint, func(w *Writer) { w.WriteUvarint(12345) }},
		{"SVarint", WireSVarint, func(w *Writer) { w.WriteSvarint(-12345) }},
		{"Fixed32", WireFixed32, func(w *Writer) { w.WriteFixed32(12345) }},
		{"Fixed64", WireFixed64, func(w *Writer) { w.WriteFixed64(12345) }},
		{"Bytes", WireBytes, func(w *Writer) { w.WriteBytes([]byte{1, 2, 3, 4, 5}) }},
		{"TypeRef", WireTypeRef, func(w *Writer) { w.WriteTypeID(TypeID(128)) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWriter()
			tc.setup(w)
			w.WriteInt32(42) // Marker after the value

			r := NewReader(w.Bytes())
			r.SkipValue(tc.wireType)
			if r.Err() != nil {
				t.Fatalf("SkipValue error: %v", r.Err())
			}
			// Should be able to read the marker
			v := r.ReadInt32()
			if v != 42 {
				t.Errorf("After SkipValue, ReadInt32 = %d, want 42", v)
			}
		})
	}
}

func TestSkipValueUnknownWireType(t *testing.T) {
	r := NewReader([]byte{1, 2, 3})
	r.SkipValue(WireType(99))
	if r.Err() == nil {
		t.Error("SkipValue with unknown wire type should fail")
	}
}

func TestSubReader(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	r := NewReader(data)
	r.Skip(2)

	sub := r.SubReader(4)
	if sub == nil {
		t.Fatal("SubReader returned nil")
	}
	if sub.Len() != 4 {
		t.Errorf("SubReader Len() = %d, want 4", sub.Len())
	}
	if r.Pos() != 6 {
		t.Errorf("Parent Pos() = %d, want 6", r.Pos())
	}

	// Read from sub-reader
	v := sub.ReadUint8()
	if v != 3 {
		t.Errorf("SubReader ReadUint8 = %d, want 3", v)
	}
}

func TestReaderRemaining(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewReader(data)
	r.Skip(2)
	rem := r.Remaining()
	if !bytes.Equal(rem, []byte{3, 4, 5}) {
		t.Errorf("Remaining = %v, want [3,4,5]", rem)
	}
}

func TestReaderEOF(t *testing.T) {
	r := NewReader([]byte{1, 2})
	if r.EOF() {
		t.Error("EOF() should be false initially")
	}
	r.Skip(2)
	if !r.EOF() {
		t.Error("EOF() should be true after reading all data")
	}
}

func TestReaderRemainingEmpty(t *testing.T) {
	r := NewReader([]byte{1})
	r.Skip(1)
	rem := r.Remaining()
	if rem != nil {
		t.Errorf("Remaining at EOF = %v, want nil", rem)
	}
}

func TestReaderErrorSticky(t *testing.T) {
	r := NewReader([]byte{})
	_ = r.ReadBool()
	err1 := r.Err()
	_ = r.ReadInt32()
	err2 := r.Err()

	if err1 != err2 {
		t.Error("Error should be sticky")
	}
}

func TestReadWriteRoundTrip(t *testing.T) {
	// Write a complex structure
	w := NewWriter()
	w.WriteBool(true)
	w.WriteInt8(-42)
	w.WriteUint16(300)
	w.WriteInt32(-100000)
	w.WriteUint64(math.MaxUint64)
	w.WriteFloat32(3.14)
	w.WriteFloat64(2.71828)
	w.WriteComplex64(1 + 2i)
	w.WriteComplex128(3 + 4i)
	w.WriteString("hello, world!")
	w.WriteBytes([]byte{0xDE, 0xAD, 0xBE, 0xEF})

	if w.Err() != nil {
		t.Fatalf("Write error: %v", w.Err())
	}

	// Read it back
	r := NewReader(w.Bytes())
	if r.ReadBool() != true {
		t.Error("Bool mismatch")
	}
	if r.ReadInt8() != -42 {
		t.Error("Int8 mismatch")
	}
	if r.ReadUint16() != 300 {
		t.Error("Uint16 mismatch")
	}
	if r.ReadInt32() != -100000 {
		t.Error("Int32 mismatch")
	}
	if r.ReadUint64() != math.MaxUint64 {
		t.Error("Uint64 mismatch")
	}
	if r.ReadFloat32() != 3.14 {
		t.Error("Float32 mismatch")
	}
	if r.ReadFloat64() != 2.71828 {
		t.Error("Float64 mismatch")
	}
	if r.ReadComplex64() != 1+2i {
		t.Error("Complex64 mismatch")
	}
	if r.ReadComplex128() != 3+4i {
		t.Error("Complex128 mismatch")
	}
	if r.ReadString() != "hello, world!" {
		t.Error("String mismatch")
	}
	if !bytes.Equal(r.ReadBytes(), []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
		t.Error("Bytes mismatch")
	}

	if r.Err() != nil {
		t.Errorf("Read error: %v", r.Err())
	}
	if !r.EOF() {
		t.Error("Should be at EOF")
	}
}

func TestReadUint16Overflow(t *testing.T) {
	w := NewWriter()
	w.WriteUvarint(0x10000) // 65536, too large for uint16

	r := NewReader(w.Bytes())
	_ = r.ReadUint16()
	if r.Err() == nil {
		t.Error("ReadUint16 with overflow should fail")
	}
}

func TestReadUint32Overflow(t *testing.T) {
	w := NewWriter()
	w.WriteUvarint(0x100000000) // Too large for uint32

	r := NewReader(w.Bytes())
	_ = r.ReadUint32()
	if r.Err() == nil {
		t.Error("ReadUint32 with overflow should fail")
	}
}

func TestReadInt16Overflow(t *testing.T) {
	w := NewWriter()
	w.WriteSvarint(40000) // Too large for int16

	r := NewReader(w.Bytes())
	_ = r.ReadInt16()
	if r.Err() == nil {
		t.Error("ReadInt16 with overflow should fail")
	}
}

func TestReadInt32Overflow(t *testing.T) {
	w := NewWriter()
	w.WriteSvarint(3000000000) // Too large for int32

	r := NewReader(w.Bytes())
	_ = r.ReadInt32()
	if r.Err() == nil {
		t.Error("ReadInt32 with overflow should fail")
	}
}

func BenchmarkReader(b *testing.B) {
	// Prepare data
	w := NewWriter()
	w.WriteBool(true)
	w.WriteInt32(42)
	w.WriteInt64(1000000)
	w.WriteFloat64(3.14159)
	w.WriteString("hello")
	data := w.BytesCopy()

	b.Run("Primitives", func(b *testing.B) {
		r := NewReader(data)
		for i := 0; i < b.N; i++ {
			r.Reset(data)
			_ = r.ReadBool()
			_ = r.ReadInt32()
			_ = r.ReadInt64()
			_ = r.ReadFloat64()
			_ = r.ReadString()
		}
	})
}
