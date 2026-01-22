package cramberry

import (
	"bytes"
	"math"
	"testing"
)

func TestWriterBasic(t *testing.T) {
	w := NewWriter()
	if w.Len() != 0 {
		t.Errorf("Len() = %d, want 0", w.Len())
	}
	if w.Cap() < 256 {
		t.Errorf("Cap() = %d, want >= 256", w.Cap())
	}
	if w.Err() != nil {
		t.Errorf("Err() = %v, want nil", w.Err())
	}
}

func TestWriterWithOptions(t *testing.T) {
	w := NewWriterWithOptions(SecureOptions)
	opts := w.Options()
	if opts.Limits.MaxMessageSize != SecureLimits.MaxMessageSize {
		t.Error("Options not set correctly")
	}
}

func TestWriterWithBuffer(t *testing.T) {
	buf := make([]byte, 100, 1024)
	w := NewWriterWithBuffer(buf, DefaultOptions)
	// Buffer should be reset but capacity preserved
	if w.Len() != 0 {
		t.Errorf("Len() = %d, want 0", w.Len())
	}
	if w.Cap() != 1024 {
		t.Errorf("Cap() = %d, want 1024", w.Cap())
	}
}

func TestWriterPool(t *testing.T) {
	w := GetWriter()
	if w == nil {
		t.Fatal("GetWriter() returned nil")
	}
	w.WriteBool(true)
	data := w.BytesCopy()
	PutWriter(w)

	if len(data) != 1 || data[0] != 1 {
		t.Errorf("data = %v, want [1]", data)
	}

	// Get another writer from pool
	w2 := GetWriter()
	if w2.Len() != 0 {
		t.Errorf("Pooled writer not reset, Len() = %d", w2.Len())
	}
	PutWriter(w2)

	// PutWriter with nil should not panic
	PutWriter(nil)
}

func TestWriterReset(t *testing.T) {
	w := NewWriter()
	w.WriteBool(true)
	w.WriteInt32(42)
	_ = w.Bytes()

	w.Reset()

	if w.Len() != 0 {
		t.Errorf("Len() after Reset = %d, want 0", w.Len())
	}
	if w.Err() != nil {
		t.Errorf("Err() after Reset = %v, want nil", w.Err())
	}

	// Should be able to write again after reset
	w.WriteBool(false)
	if w.Err() != nil {
		t.Errorf("Write after Reset failed: %v", w.Err())
	}
}

func TestWriteBool(t *testing.T) {
	tests := []struct {
		value    bool
		expected byte
	}{
		{false, 0},
		{true, 1},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteBool(tc.value)
		data := w.Bytes()
		if len(data) != 1 || data[0] != tc.expected {
			t.Errorf("WriteBool(%v) = %v, want [%d]", tc.value, data, tc.expected)
		}
	}
}

func TestWriteUint8(t *testing.T) {
	tests := []uint8{0, 1, 127, 128, 255}
	for _, v := range tests {
		w := NewWriter()
		w.WriteUint8(v)
		data := w.Bytes()
		if len(data) != 1 || data[0] != v {
			t.Errorf("WriteUint8(%d) = %v, want [%d]", v, data, v)
		}
	}
}

func TestWriteInt8(t *testing.T) {
	tests := []int8{-128, -1, 0, 1, 127}
	for _, v := range tests {
		w := NewWriter()
		w.WriteInt8(v)
		data := w.Bytes()
		if len(data) != 1 || int8(data[0]) != v {
			t.Errorf("WriteInt8(%d) = %v, want [%d]", v, data, byte(v))
		}
	}
}

func TestWriteUvarint(t *testing.T) {
	tests := []struct {
		value    uint64
		expected []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{127, []byte{0x7F}},
		{128, []byte{0x80, 0x01}},
		{255, []byte{0xFF, 0x01}},
		{300, []byte{0xAC, 0x02}},
		{16383, []byte{0xFF, 0x7F}},
		{16384, []byte{0x80, 0x80, 0x01}},
		{math.MaxUint64, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteUvarint(tc.value)
		data := w.Bytes()
		if !bytes.Equal(data, tc.expected) {
			t.Errorf("WriteUvarint(%d) = %v, want %v", tc.value, data, tc.expected)
		}
	}
}

func TestWriteSvarint(t *testing.T) {
	tests := []struct {
		value    int64
		expected []byte
	}{
		{0, []byte{0x00}},
		{-1, []byte{0x01}},
		{1, []byte{0x02}},
		{-2, []byte{0x03}},
		{2, []byte{0x04}},
		{63, []byte{0x7E}},
		{-64, []byte{0x7F}},
		{64, []byte{0x80, 0x01}},
		{-65, []byte{0x81, 0x01}},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteSvarint(tc.value)
		data := w.Bytes()
		if !bytes.Equal(data, tc.expected) {
			t.Errorf("WriteSvarint(%d) = %v, want %v", tc.value, data, tc.expected)
		}
	}
}

func TestWriteUint(t *testing.T) {
	w := NewWriter()
	w.WriteUint16(300)
	w.WriteUint32(70000)
	w.WriteUint64(1000000)
	w.WriteUint(42)
	if w.Err() != nil {
		t.Errorf("Write failed: %v", w.Err())
	}
}

func TestWriteInt(t *testing.T) {
	w := NewWriter()
	w.WriteInt16(-300)
	w.WriteInt32(-70000)
	w.WriteInt64(-1000000)
	w.WriteInt(-42)
	if w.Err() != nil {
		t.Errorf("Write failed: %v", w.Err())
	}
}

func TestWriteFixed32(t *testing.T) {
	tests := []struct {
		value    uint32
		expected []byte
	}{
		{0, []byte{0, 0, 0, 0}},
		{1, []byte{1, 0, 0, 0}},
		{0x12345678, []byte{0x78, 0x56, 0x34, 0x12}},
		{math.MaxUint32, []byte{0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteFixed32(tc.value)
		data := w.Bytes()
		if !bytes.Equal(data, tc.expected) {
			t.Errorf("WriteFixed32(%d) = %v, want %v", tc.value, data, tc.expected)
		}
	}
}

func TestWriteFixed64(t *testing.T) {
	tests := []struct {
		value    uint64
		expected []byte
	}{
		{0, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{1, []byte{1, 0, 0, 0, 0, 0, 0, 0}},
		{0x123456789ABCDEF0, []byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteFixed64(tc.value)
		data := w.Bytes()
		if !bytes.Equal(data, tc.expected) {
			t.Errorf("WriteFixed64(%d) = %v, want %v", tc.value, data, tc.expected)
		}
	}
}

func TestWriteSFixed32(t *testing.T) {
	tests := []struct {
		value    int32
		expected []byte
	}{
		{0, []byte{0, 0, 0, 0}},
		{1, []byte{1, 0, 0, 0}},
		{-1, []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{-2147483648, []byte{0x00, 0x00, 0x00, 0x80}}, // math.MinInt32
		{2147483647, []byte{0xFF, 0xFF, 0xFF, 0x7F}},  // math.MaxInt32
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteSFixed32(tc.value)
		data := w.Bytes()
		if !bytes.Equal(data, tc.expected) {
			t.Errorf("WriteSFixed32(%d) = %v, want %v", tc.value, data, tc.expected)
		}
	}
}

func TestWriteSFixed64(t *testing.T) {
	tests := []struct {
		value    int64
		expected []byte
	}{
		{0, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{1, []byte{1, 0, 0, 0, 0, 0, 0, 0}},
		{-1, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
		{-9223372036854775808, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80}}, // math.MinInt64
		{9223372036854775807, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x7F}},  // math.MaxInt64
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteSFixed64(tc.value)
		data := w.Bytes()
		if !bytes.Equal(data, tc.expected) {
			t.Errorf("WriteSFixed64(%d) = %v, want %v", tc.value, data, tc.expected)
		}
	}
}

func TestWriteFloat32(t *testing.T) {
	w := NewWriter()
	w.WriteFloat32(3.14)
	data := w.Bytes()
	if len(data) != 4 {
		t.Errorf("WriteFloat32 produced %d bytes, want 4", len(data))
	}
}

func TestWriteFloat64(t *testing.T) {
	w := NewWriter()
	w.WriteFloat64(3.14159265359)
	data := w.Bytes()
	if len(data) != 8 {
		t.Errorf("WriteFloat64 produced %d bytes, want 8", len(data))
	}
}

func TestWriteFloatDeterminism(t *testing.T) {
	// Test that -0 and +0 produce the same encoding
	w1 := NewWriter()
	w1.WriteFloat64(0.0)
	w2 := NewWriter()
	w2.WriteFloat64(math.Copysign(0, -1))
	if !bytes.Equal(w1.Bytes(), w2.Bytes()) {
		t.Error("Float64 -0 and +0 should produce same encoding")
	}

	// Same for float32
	w3 := NewWriter()
	w3.WriteFloat32(0.0)
	w4 := NewWriter()
	w4.WriteFloat32(float32(math.Copysign(0, -1)))
	if !bytes.Equal(w3.Bytes(), w4.Bytes()) {
		t.Error("Float32 -0 and +0 should produce same encoding")
	}
}

func TestWriteComplex(t *testing.T) {
	w := NewWriter()
	w.WriteComplex64(1 + 2i)
	if len(w.Bytes()) != 8 {
		t.Errorf("WriteComplex64 produced %d bytes, want 8", len(w.Bytes()))
	}

	w.Reset()
	w.WriteComplex128(1 + 2i)
	if len(w.Bytes()) != 16 {
		t.Errorf("WriteComplex128 produced %d bytes, want 16", len(w.Bytes()))
	}
}

func TestWriteString(t *testing.T) {
	tests := []struct {
		value string
	}{
		{""},
		{"hello"},
		{"hello, world!"},
		{"日本語"},
		{"emoji: 🎉"},
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteString(tc.value)
		if w.Err() != nil {
			t.Errorf("WriteString(%q) failed: %v", tc.value, w.Err())
		}
	}
}

func TestWriteStringInvalidUTF8(t *testing.T) {
	w := NewWriterWithOptions(Options{ValidateUTF8: true, Limits: DefaultLimits})
	w.WriteString("\xff\xfe")
	if w.Err() == nil {
		t.Error("WriteString with invalid UTF-8 should fail when ValidateUTF8 is true")
	}

	// Without validation, should succeed
	w2 := NewWriterWithOptions(Options{ValidateUTF8: false, Limits: DefaultLimits})
	w2.WriteString("\xff\xfe")
	if w2.Err() != nil {
		t.Errorf("WriteString with ValidateUTF8=false should not fail: %v", w2.Err())
	}
}

func TestWriteBytes(t *testing.T) {
	tests := [][]byte{
		{},
		{1, 2, 3},
		make([]byte, 1000),
	}

	for _, tc := range tests {
		w := NewWriter()
		w.WriteBytes(tc)
		if w.Err() != nil {
			t.Errorf("WriteBytes(%v) failed: %v", tc, w.Err())
		}
	}
}

func TestWriteRawBytes(t *testing.T) {
	w := NewWriter()
	w.WriteRawBytes([]byte{1, 2, 3, 4})
	data := w.Bytes()
	if !bytes.Equal(data, []byte{1, 2, 3, 4}) {
		t.Errorf("WriteRawBytes = %v, want [1,2,3,4]", data)
	}
}

func TestWriteTag(t *testing.T) {
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
		if w.Err() != nil {
			t.Errorf("WriteTag(%d, %d) failed: %v", tc.fieldNum, tc.wireType, w.Err())
		}
	}
}

func TestWriteTagInvalidFieldNumber(t *testing.T) {
	w := NewWriter()
	w.WriteTag(0, WireVarint)
	if w.Err() == nil {
		t.Error("WriteTag with field number 0 should fail")
	}

	w2 := NewWriter()
	w2.WriteTag(-1, WireVarint)
	if w2.Err() == nil {
		t.Error("WriteTag with negative field number should fail")
	}
}

func TestWriteNilAndTypeID(t *testing.T) {
	w := NewWriter()
	w.WriteNil()
	if w.Err() != nil || len(w.Bytes()) != 1 || w.Bytes()[0] != 0 {
		t.Errorf("WriteNil failed: err=%v, data=%v", w.Err(), w.Bytes())
	}

	w.Reset()
	w.WriteTypeID(TypeID(128))
	if w.Err() != nil {
		t.Errorf("WriteTypeID failed: %v", w.Err())
	}
}

func TestBeginEndMessage(t *testing.T) {
	w := NewWriter()
	checkpoint := w.BeginMessage()
	w.WriteInt32(42)
	w.WriteString("hello")
	w.EndMessage(checkpoint)

	if w.Err() != nil {
		t.Fatalf("Message writing failed: %v", w.Err())
	}

	data := w.Bytes()
	// First byte(s) should be the length prefix
	if len(data) < 2 {
		t.Fatalf("Message too short: %v", data)
	}
}

func TestNestedMessages(t *testing.T) {
	w := NewWriter()

	outer := w.BeginMessage()
	w.WriteInt32(1)

	inner := w.BeginMessage()
	w.WriteString("nested")
	w.EndMessage(inner)

	w.WriteInt32(2)
	w.EndMessage(outer)

	if w.Err() != nil {
		t.Fatalf("Nested message writing failed: %v", w.Err())
	}
}

func TestWriteArrayHeader(t *testing.T) {
	w := NewWriter()
	w.WriteArrayHeader(10)
	if w.Err() != nil {
		t.Errorf("WriteArrayHeader failed: %v", w.Err())
	}

	w2 := NewWriter()
	w2.WriteArrayHeader(-1)
	if w2.Err() == nil {
		t.Error("WriteArrayHeader with negative length should fail")
	}
}

func TestWriteMapHeader(t *testing.T) {
	w := NewWriter()
	w.WriteMapHeader(10)
	if w.Err() != nil {
		t.Errorf("WriteMapHeader failed: %v", w.Err())
	}

	w2 := NewWriter()
	w2.WriteMapHeader(-1)
	if w2.Err() == nil {
		t.Error("WriteMapHeader with negative size should fail")
	}
}

func TestWriterLimits(t *testing.T) {
	t.Run("MaxStringLength", func(t *testing.T) {
		w := NewWriterWithOptions(Options{
			Limits:       Limits{MaxStringLength: 5},
			ValidateUTF8: false,
		})
		w.WriteString("hello")
		if w.Err() != nil {
			t.Error("String at limit should succeed")
		}

		w2 := NewWriterWithOptions(Options{
			Limits:       Limits{MaxStringLength: 5},
			ValidateUTF8: false,
		})
		w2.WriteString("hello!")
		if w2.Err() == nil {
			t.Error("String over limit should fail")
		}
	})

	t.Run("MaxBytesLength", func(t *testing.T) {
		w := NewWriterWithOptions(Options{
			Limits: Limits{MaxBytesLength: 5},
		})
		w.WriteBytes([]byte{1, 2, 3, 4, 5})
		if w.Err() != nil {
			t.Error("Bytes at limit should succeed")
		}

		w2 := NewWriterWithOptions(Options{
			Limits: Limits{MaxBytesLength: 5},
		})
		w2.WriteBytes([]byte{1, 2, 3, 4, 5, 6})
		if w2.Err() == nil {
			t.Error("Bytes over limit should fail")
		}
	})

	t.Run("MaxArrayLength", func(t *testing.T) {
		w := NewWriterWithOptions(Options{
			Limits: Limits{MaxArrayLength: 10},
		})
		w.WriteArrayHeader(10)
		if w.Err() != nil {
			t.Error("Array at limit should succeed")
		}

		w2 := NewWriterWithOptions(Options{
			Limits: Limits{MaxArrayLength: 10},
		})
		w2.WriteArrayHeader(11)
		if w2.Err() == nil {
			t.Error("Array over limit should fail")
		}
	})

	t.Run("MaxMapSize", func(t *testing.T) {
		w := NewWriterWithOptions(Options{
			Limits: Limits{MaxMapSize: 10},
		})
		w.WriteMapHeader(10)
		if w.Err() != nil {
			t.Error("Map at limit should succeed")
		}

		w2 := NewWriterWithOptions(Options{
			Limits: Limits{MaxMapSize: 10},
		})
		w2.WriteMapHeader(11)
		if w2.Err() == nil {
			t.Error("Map over limit should fail")
		}
	})

	t.Run("MaxDepth", func(t *testing.T) {
		w := NewWriterWithOptions(Options{
			Limits: Limits{MaxDepth: 2},
		})
		c1 := w.BeginMessage()
		c2 := w.BeginMessage()
		c3 := w.BeginMessage() // Should fail
		if c3 != -1 {
			t.Error("Third nesting level should fail")
		}
		if w.Err() == nil {
			t.Error("Exceeding max depth should set error")
		}
		w.EndMessage(c2)
		w.EndMessage(c1)
	})
}

func TestWriterFrozen(t *testing.T) {
	w := NewWriter()
	w.WriteBool(true)
	_ = w.Bytes() // Freeze the writer

	w.WriteBool(false) // Should fail
	if w.Err() == nil {
		t.Error("Writing to frozen writer should fail")
	}
}

func TestWriterBytesCopy(t *testing.T) {
	w := NewWriter()
	w.WriteBool(true)
	copy1 := w.BytesCopy()

	w.Reset()
	w.WriteBool(false)
	copy2 := w.BytesCopy()

	if copy1[0] != 1 {
		t.Error("First copy was modified")
	}
	if copy2[0] != 0 {
		t.Error("Second copy incorrect")
	}
}

func TestWriterSetOptions(t *testing.T) {
	w := NewWriter()
	w.SetOptions(SecureOptions)
	opts := w.Options()
	if opts.Limits.MaxMessageSize != SecureLimits.MaxMessageSize {
		t.Error("SetOptions did not update options")
	}
}

func TestWriterErrorSticky(t *testing.T) {
	w := NewWriterWithOptions(Options{
		Limits: Limits{MaxStringLength: 3},
	})
	w.WriteString("toolong")
	err1 := w.Err()
	w.WriteBool(true) // Should be ignored
	err2 := w.Err()

	if err1 != err2 {
		t.Error("Error should be sticky")
	}
	if len(w.Bytes()) != 0 {
		t.Error("Nothing should be written after error")
	}
}

func TestSizeOf(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		expected int
	}{
		{"Bool", SizeOfBool(true), 1},
		{"Uint8", SizeOfUint8(255), 1},
		{"Int8", SizeOfInt8(-1), 1},
		{"Uint16_small", SizeOfUint16(127), 1},
		{"Uint16_large", SizeOfUint16(300), 2},
		{"Uint32_small", SizeOfUint32(127), 1},
		{"Uint32_large", SizeOfUint32(16384), 3},
		{"Int16_small", SizeOfInt16(63), 1},
		{"Int16_large", SizeOfInt16(-65), 2},
		{"Int32_small", SizeOfInt32(63), 1},
		{"Float32", SizeOfFloat32(3.14), 4},
		{"Float64", SizeOfFloat64(3.14), 8},
		{"Complex64", SizeOfComplex64(1 + 2i), 8},
		{"Complex128", SizeOfComplex128(1 + 2i), 16},
		{"String_empty", SizeOfString(""), 1},
		{"String_hello", SizeOfString("hello"), 6},
		{"Bytes_empty", SizeOfBytes(nil), 1},
		{"Bytes_5", SizeOfBytes(make([]byte, 5)), 6},
		{"Tag_1", SizeOfTag(1), 1},
		{"Tag_large", SizeOfTag(2000), 2},
		{"Uvarint_0", SizeOfUvarint(0), 1},
		{"Uvarint_max", SizeOfUvarint(math.MaxUint64), 10},
		{"Svarint_0", SizeOfSvarint(0), 1},
		{"Svarint_neg", SizeOfSvarint(-1000000), 3},
		{"SFixed32", SizeOfSFixed32(-1), 4},
		{"SFixed64", SizeOfSFixed64(-1), 8},
	}

	for _, tc := range tests {
		if tc.size != tc.expected {
			t.Errorf("SizeOf%s = %d, want %d", tc.name, tc.size, tc.expected)
		}
	}
}

func TestWriterLargeMessage(t *testing.T) {
	w := NewWriter()
	// Write a large amount of data to test buffer growth
	for i := 0; i < 10000; i++ {
		w.WriteInt32(int32(i))
	}
	if w.Err() != nil {
		t.Errorf("Large message failed: %v", w.Err())
	}
	if w.Len() < 10000 {
		t.Errorf("Large message too short: %d bytes", w.Len())
	}
}

func TestIsValidUTF8(t *testing.T) {
	tests := []struct {
		s     string
		valid bool
	}{
		{"", true},
		{"hello", true},
		{"日本語", true},
		{"emoji: 🎉", true},
		{"\x80", false},                          // Invalid leading byte
		{"\xC0\x80", false},                      // Overlong encoding
		{"\xE0\x80\x80", false},                  // Overlong encoding
		{"\xED\xA0\x80", false},                  // Surrogate half
		{"\xF0\x80\x80\x80", false},              // Overlong encoding
		{"\xF4\x90\x80\x80", false},              // Out of range
		{"\xC2", false},                          // Truncated sequence
		{"\xE0\x80", false},                      // Truncated sequence
		{"\xC2\x00", false},                      // Invalid continuation
		{string([]byte{0xC2, 0xA0}), true},       // Valid 2-byte
		{string([]byte{0xE2, 0x82, 0xAC}), true}, // Euro sign
	}

	for _, tc := range tests {
		got := isValidUTF8(tc.s)
		if got != tc.valid {
			t.Errorf("isValidUTF8(%q) = %v, want %v", tc.s, got, tc.valid)
		}
	}
}

func BenchmarkWriter(b *testing.B) {
	b.Run("Primitives", func(b *testing.B) {
		w := NewWriter()
		for i := 0; i < b.N; i++ {
			w.Reset()
			w.WriteBool(true)
			w.WriteInt32(42)
			w.WriteInt64(1000000)
			w.WriteFloat64(3.14159)
			w.WriteString("hello")
			_ = w.Bytes()
		}
	})

	b.Run("Pooled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			w := GetWriter()
			w.WriteBool(true)
			w.WriteInt32(42)
			w.WriteInt64(1000000)
			w.WriteFloat64(3.14159)
			w.WriteString("hello")
			_ = w.BytesCopy()
			PutWriter(w)
		}
	})

	b.Run("Message", func(b *testing.B) {
		w := NewWriter()
		for i := 0; i < b.N; i++ {
			w.Reset()
			cp := w.BeginMessage()
			w.WriteInt32(42)
			w.WriteString("hello world")
			w.EndMessage(cp)
			_ = w.Bytes()
		}
	})
}
