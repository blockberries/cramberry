package wire

import (
	"bytes"
	"testing"
)

func TestWireTypeString(t *testing.T) {
	tests := []struct {
		wireType WireType
		expected string
	}{
		{WireVarint, "Varint"},
		{WireFixed64, "Fixed64"},
		{WireBytes, "Bytes"},
		{WireFixed32, "Fixed32"},
		{WireSVarint, "SVarint"},
		{WireTypeRef, "TypeRef"},
		{WireType(3), "Unknown"},
		{WireType(4), "Unknown"},
		{WireType(100), "Unknown"},
	}

	for _, tc := range tests {
		if tc.wireType.String() != tc.expected {
			t.Errorf("WireType(%d).String() = %q, want %q", tc.wireType, tc.wireType.String(), tc.expected)
		}
	}
}

func TestWireTypeIsValid(t *testing.T) {
	validTypes := []WireType{WireVarint, WireFixed64, WireBytes, WireFixed32, WireSVarint, WireTypeRef}
	for _, wt := range validTypes {
		if !wt.IsValid() {
			t.Errorf("WireType(%d).IsValid() = false, want true", wt)
		}
	}

	invalidTypes := []WireType{3, 4, 8, 100}
	for _, wt := range invalidTypes {
		if wt.IsValid() {
			t.Errorf("WireType(%d).IsValid() = true, want false", wt)
		}
	}
}

func TestNewTag(t *testing.T) {
	tests := []struct {
		fieldNum int
		wireType WireType
		expected Tag
	}{
		{1, WireVarint, Tag(0x08)},   // (1 << 3) | 0 = 8
		{1, WireFixed64, Tag(0x09)},  // (1 << 3) | 1 = 9
		{1, WireBytes, Tag(0x0A)},    // (1 << 3) | 2 = 10
		{1, WireFixed32, Tag(0x0D)},  // (1 << 3) | 5 = 13
		{1, WireSVarint, Tag(0x0E)},  // (1 << 3) | 6 = 14
		{1, WireTypeRef, Tag(0x0F)},  // (1 << 3) | 7 = 15
		{2, WireVarint, Tag(0x10)},   // (2 << 3) | 0 = 16
		{15, WireVarint, Tag(0x78)},  // (15 << 3) | 0 = 120
		{16, WireVarint, Tag(0x80)},  // (16 << 3) | 0 = 128
		{100, WireBytes, Tag(0x322)}, // (100 << 3) | 2 = 802
	}

	for _, tc := range tests {
		tag := NewTag(tc.fieldNum, tc.wireType)
		if tag != tc.expected {
			t.Errorf("NewTag(%d, %d) = %d, want %d", tc.fieldNum, tc.wireType, tag, tc.expected)
		}
	}
}

func TestTagFieldNumber(t *testing.T) {
	tests := []struct {
		tag      Tag
		expected int
	}{
		{Tag(0x08), 1},
		{Tag(0x10), 2},
		{Tag(0x78), 15},
		{Tag(0x80), 16},
		{Tag(0x322), 100},
	}

	for _, tc := range tests {
		fieldNum := tc.tag.FieldNumber()
		if fieldNum != tc.expected {
			t.Errorf("Tag(%d).FieldNumber() = %d, want %d", tc.tag, fieldNum, tc.expected)
		}
	}
}

func TestTagWireType(t *testing.T) {
	tests := []struct {
		tag      Tag
		expected WireType
	}{
		{Tag(0x08), WireVarint},
		{Tag(0x09), WireFixed64},
		{Tag(0x0A), WireBytes},
		{Tag(0x0D), WireFixed32},
		{Tag(0x0E), WireSVarint},
		{Tag(0x0F), WireTypeRef},
	}

	for _, tc := range tests {
		wireType := tc.tag.WireType()
		if wireType != tc.expected {
			t.Errorf("Tag(%d).WireType() = %d, want %d", tc.tag, wireType, tc.expected)
		}
	}
}

func TestAppendTag(t *testing.T) {
	tests := []struct {
		fieldNum int
		wireType WireType
		expected []byte
	}{
		{1, WireVarint, []byte{0x08}},
		{1, WireBytes, []byte{0x0A}},
		{2, WireVarint, []byte{0x10}},
		{15, WireVarint, []byte{0x78}},
		{16, WireVarint, []byte{0x80, 0x01}},
		{100, WireBytes, []byte{0xa2, 0x06}},
		{1000, WireVarint, []byte{0xc0, 0x3e}},
	}

	for _, tc := range tests {
		result := AppendTag(nil, tc.fieldNum, tc.wireType)
		if !bytes.Equal(result, tc.expected) {
			t.Errorf("AppendTag(nil, %d, %d) = %v, want %v", tc.fieldNum, tc.wireType, result, tc.expected)
		}
	}
}

func TestDecodeTag(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		fieldNum     int
		wireType     WireType
		bytesRead    int
		expectError  bool
	}{
		{"field1_varint", []byte{0x08}, 1, WireVarint, 1, false},
		{"field1_bytes", []byte{0x0A}, 1, WireBytes, 1, false},
		{"field2_varint", []byte{0x10}, 2, WireVarint, 1, false},
		{"field15_varint", []byte{0x78}, 15, WireVarint, 1, false},
		{"field16_varint", []byte{0x80, 0x01}, 16, WireVarint, 2, false},
		{"field100_bytes", []byte{0xa2, 0x06}, 100, WireBytes, 2, false},
		{"field1000_varint", []byte{0xc0, 0x3e}, 1000, WireVarint, 2, false},
		{"with_trailing", []byte{0x08, 0xff, 0xff}, 1, WireVarint, 1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fieldNum, wireType, n, err := DecodeTag(tc.data)
			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fieldNum != tc.fieldNum {
				t.Errorf("fieldNum = %d, want %d", fieldNum, tc.fieldNum)
			}
			if wireType != tc.wireType {
				t.Errorf("wireType = %d, want %d", wireType, tc.wireType)
			}
			if n != tc.bytesRead {
				t.Errorf("bytesRead = %d, want %d", n, tc.bytesRead)
			}
		})
	}
}

func TestDecodeTagErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		err  error
	}{
		{"empty", []byte{}, ErrVarintTruncated},
		{"truncated", []byte{0x80}, ErrVarintTruncated},
		{"field_zero", []byte{0x00}, ErrInvalidFieldNumber},          // field 0
		{"field_zero_wire2", []byte{0x02}, ErrInvalidFieldNumber},    // field 0, wire 2
		{"invalid_wire_3", []byte{0x0B}, ErrInvalidWireType},         // field 1, wire 3
		{"invalid_wire_4", []byte{0x0C}, ErrInvalidWireType},         // field 1, wire 4
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := DecodeTag(tc.data)
			if err != tc.err {
				t.Errorf("DecodeTag(%v) error = %v, want %v", tc.data, err, tc.err)
			}
		})
	}
}

func TestDecodeTagRelaxed(t *testing.T) {
	// Unknown wire types should not cause errors in relaxed mode
	data := []byte{0x0B} // field 1, wire 3 (unknown)
	fieldNum, wireType, n, err := DecodeTagRelaxed(data)

	if err != nil {
		t.Errorf("DecodeTagRelaxed should not error on unknown wire type: %v", err)
	}
	if fieldNum != 1 {
		t.Errorf("fieldNum = %d, want 1", fieldNum)
	}
	if wireType != 3 {
		t.Errorf("wireType = %d, want 3", wireType)
	}
	if n != 1 {
		t.Errorf("n = %d, want 1", n)
	}

	// But field number 0 should still error
	_, _, _, err = DecodeTagRelaxed([]byte{0x00})
	if err != ErrInvalidFieldNumber {
		t.Errorf("DecodeTagRelaxed with field 0 should error: %v", err)
	}
}

func TestTagSize(t *testing.T) {
	tests := []struct {
		fieldNum int
		expected int
	}{
		{1, 1},     // (1 << 3) = 8, fits in 1 byte
		{15, 1},    // (15 << 3) = 120, fits in 1 byte
		{16, 2},    // (16 << 3) = 128, needs 2 bytes
		{2047, 2},  // (2047 << 3) = 16376, fits in 2 bytes
		{2048, 3},  // (2048 << 3) = 16384, needs 3 bytes
		{1000000, 4}, // Large field number
	}

	for _, tc := range tests {
		size := TagSize(tc.fieldNum)
		if size != tc.expected {
			t.Errorf("TagSize(%d) = %d, want %d", tc.fieldNum, size, tc.expected)
		}

		// Verify by actually encoding
		encoded := AppendTag(nil, tc.fieldNum, WireVarint)
		if len(encoded) != tc.expected {
			t.Errorf("TagSize(%d) = %d, but actual encoding is %d bytes", tc.fieldNum, tc.expected, len(encoded))
		}
	}
}

func TestPutTag(t *testing.T) {
	buf := make([]byte, 10)
	n := PutTag(buf, 100, WireBytes)

	expected := []byte{0xa2, 0x06}
	if !bytes.Equal(buf[:n], expected) {
		t.Errorf("PutTag(100, WireBytes) = %v, want %v", buf[:n], expected)
	}
}

func TestValidateFieldNumber(t *testing.T) {
	// Valid field numbers
	validNums := []int{1, 2, 100, 1000, MaxFieldNumber}
	for _, n := range validNums {
		if err := ValidateFieldNumber(n); err != nil {
			t.Errorf("ValidateFieldNumber(%d) = %v, want nil", n, err)
		}
	}

	// Invalid field numbers
	invalidNums := []int{0, -1, -100, MaxFieldNumber + 1}
	for _, n := range invalidNums {
		if err := ValidateFieldNumber(n); err == nil {
			t.Errorf("ValidateFieldNumber(%d) = nil, want error", n)
		}
	}
}

func TestWireTypeForKind(t *testing.T) {
	tests := []struct {
		kind     string
		expected WireType
	}{
		{"bool", WireVarint},
		{"uint8", WireVarint},
		{"uint16", WireVarint},
		{"uint32", WireVarint},
		{"uint64", WireVarint},
		{"uint", WireVarint},
		{"uintptr", WireVarint},
		{"int8", WireSVarint},
		{"int16", WireSVarint},
		{"int32", WireSVarint},
		{"int64", WireSVarint},
		{"int", WireSVarint},
		{"float32", WireFixed32},
		{"float64", WireFixed64},
		{"string", WireBytes},
		{"slice", WireBytes},
		{"array", WireBytes},
		{"map", WireBytes},
		{"struct", WireBytes},
		{"ptr", WireBytes},
		{"interface", WireTypeRef},
		{"unknown", WireBytes},
	}

	for _, tc := range tests {
		result := WireTypeForKind(tc.kind)
		if result != tc.expected {
			t.Errorf("WireTypeForKind(%q) = %v, want %v", tc.kind, result, tc.expected)
		}
	}
}

func TestTagRoundTrip(t *testing.T) {
	wireTypes := []WireType{WireVarint, WireFixed64, WireBytes, WireFixed32, WireSVarint, WireTypeRef}
	fieldNums := []int{1, 2, 15, 16, 127, 128, 1000, 10000, 100000, MaxFieldNumber}

	for _, fieldNum := range fieldNums {
		for _, wireType := range wireTypes {
			encoded := AppendTag(nil, fieldNum, wireType)
			decodedField, decodedWire, n, err := DecodeTag(encoded)

			if err != nil {
				t.Errorf("round trip error for field %d, wire %d: %v", fieldNum, wireType, err)
				continue
			}
			if n != len(encoded) {
				t.Errorf("round trip bytes mismatch: encoded %d, decoded %d", len(encoded), n)
			}
			if decodedField != fieldNum {
				t.Errorf("round trip field mismatch: %d -> %d", fieldNum, decodedField)
			}
			if decodedWire != wireType {
				t.Errorf("round trip wire mismatch: %d -> %d", wireType, decodedWire)
			}
		}
	}
}

// Benchmarks

func BenchmarkAppendTag_Small(b *testing.B) {
	buf := make([]byte, 0, 8)
	for i := 0; i < b.N; i++ {
		buf = AppendTag(buf[:0], 1, WireVarint)
	}
}

func BenchmarkAppendTag_Large(b *testing.B) {
	buf := make([]byte, 0, 8)
	for i := 0; i < b.N; i++ {
		buf = AppendTag(buf[:0], 10000, WireBytes)
	}
}

func BenchmarkDecodeTag_Small(b *testing.B) {
	data := []byte{0x08}
	for i := 0; i < b.N; i++ {
		_, _, _, _ = DecodeTag(data)
	}
}

func BenchmarkDecodeTag_Large(b *testing.B) {
	data := AppendTag(nil, 10000, WireBytes)
	for i := 0; i < b.N; i++ {
		_, _, _, _ = DecodeTag(data)
	}
}

func BenchmarkTagSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = TagSize(1000)
	}
}

// Fuzz test

func FuzzTagRoundTrip(f *testing.F) {
	f.Add(1, uint8(0))
	f.Add(15, uint8(2))
	f.Add(16, uint8(5))
	f.Add(1000, uint8(7))
	f.Add(MaxFieldNumber, uint8(1))

	f.Fuzz(func(t *testing.T, fieldNum int, wireTypeByte uint8) {
		// Skip invalid inputs
		if fieldNum <= 0 || fieldNum > MaxFieldNumber {
			return
		}
		wireType := WireType(wireTypeByte & 0x7) // Only lower 3 bits
		if !wireType.IsValid() {
			return
		}

		encoded := AppendTag(nil, fieldNum, wireType)
		decodedField, decodedWire, n, err := DecodeTag(encoded)

		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if n != len(encoded) {
			t.Fatalf("bytes mismatch: %d vs %d", n, len(encoded))
		}
		if decodedField != fieldNum {
			t.Fatalf("field mismatch: %d vs %d", decodedField, fieldNum)
		}
		if decodedWire != wireType {
			t.Fatalf("wire mismatch: %d vs %d", decodedWire, wireType)
		}
	})
}
