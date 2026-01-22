package wire

import (
	"bytes"
	"math"
	"testing"
)

// Test cases for unsigned varint encoding
var uvarintTestCases = []struct {
	name     string
	value    uint64
	expected []byte
}{
	{"zero", 0, []byte{0x00}},
	{"one", 1, []byte{0x01}},
	{"max_1_byte", 127, []byte{0x7f}},
	{"min_2_byte", 128, []byte{0x80, 0x01}},
	{"300", 300, []byte{0xac, 0x02}},
	{"max_2_byte", 16383, []byte{0xff, 0x7f}},
	{"min_3_byte", 16384, []byte{0x80, 0x80, 0x01}},
	{"max_uint32", math.MaxUint32, []byte{0xff, 0xff, 0xff, 0xff, 0x0f}},
	{"max_uint64", math.MaxUint64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
	{"power_of_2_7", 1 << 7, []byte{0x80, 0x01}},
	{"power_of_2_14", 1 << 14, []byte{0x80, 0x80, 0x01}},
	{"power_of_2_21", 1 << 21, []byte{0x80, 0x80, 0x80, 0x01}},
	{"power_of_2_28", 1 << 28, []byte{0x80, 0x80, 0x80, 0x80, 0x01}},
	{"power_of_2_35", 1 << 35, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x01}},
	{"power_of_2_42", 1 << 42, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}},
	{"power_of_2_49", 1 << 49, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}},
	{"power_of_2_56", 1 << 56, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}},
	{"power_of_2_63", 1 << 63, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}},
}

// Test cases for signed varint encoding (zigzag)
var svarintTestCases = []struct {
	name     string
	value    int64
	expected []byte
}{
	{"zero", 0, []byte{0x00}},
	{"minus_one", -1, []byte{0x01}},
	{"one", 1, []byte{0x02}},
	{"minus_two", -2, []byte{0x03}},
	{"two", 2, []byte{0x04}},
	{"minus_64", -64, []byte{0x7f}},
	{"63", 63, []byte{0x7e}},
	{"64", 64, []byte{0x80, 0x01}},
	{"minus_65", -65, []byte{0x81, 0x01}},
	{"max_int32", math.MaxInt32, []byte{0xfe, 0xff, 0xff, 0xff, 0x0f}},
	{"min_int32", math.MinInt32, []byte{0xff, 0xff, 0xff, 0xff, 0x0f}},
	{"max_int64", math.MaxInt64, []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
	{"min_int64", math.MinInt64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
}

func TestAppendUvarint(t *testing.T) {
	for _, tc := range uvarintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendUvarint(nil, tc.value)
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("AppendUvarint(%d) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

func TestAppendSvarint(t *testing.T) {
	for _, tc := range svarintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendSvarint(nil, tc.value)
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("AppendSvarint(%d) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

func TestDecodeUvarint(t *testing.T) {
	for _, tc := range uvarintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			value, n, err := DecodeUvarint(tc.expected)
			if err != nil {
				t.Fatalf("DecodeUvarint(%v) error: %v", tc.expected, err)
			}
			if value != tc.value {
				t.Errorf("DecodeUvarint(%v) value = %d, want %d", tc.expected, value, tc.value)
			}
			if n != len(tc.expected) {
				t.Errorf("DecodeUvarint(%v) n = %d, want %d", tc.expected, n, len(tc.expected))
			}
		})
	}
}

func TestDecodeSvarint(t *testing.T) {
	for _, tc := range svarintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			value, n, err := DecodeSvarint(tc.expected)
			if err != nil {
				t.Fatalf("DecodeSvarint(%v) error: %v", tc.expected, err)
			}
			if value != tc.value {
				t.Errorf("DecodeSvarint(%v) value = %d, want %d", tc.expected, value, tc.value)
			}
			if n != len(tc.expected) {
				t.Errorf("DecodeSvarint(%v) n = %d, want %d", tc.expected, n, len(tc.expected))
			}
		})
	}
}

func TestUvarintRoundTrip(t *testing.T) {
	// Test many values for round-trip correctness
	testValues := []uint64{
		0, 1, 2, 126, 127, 128, 129, 255, 256,
		16382, 16383, 16384, 16385,
		1<<21 - 1, 1 << 21, 1<<21 + 1,
		1<<28 - 1, 1 << 28, 1<<28 + 1,
		1<<35 - 1, 1 << 35, 1<<35 + 1,
		1<<42 - 1, 1 << 42, 1<<42 + 1,
		1<<49 - 1, 1 << 49, 1<<49 + 1,
		1<<56 - 1, 1 << 56, 1<<56 + 1,
		1<<63 - 1, 1 << 63, 1<<63 + 1,
		math.MaxUint64 - 1, math.MaxUint64,
	}

	for _, v := range testValues {
		encoded := AppendUvarint(nil, v)
		decoded, n, err := DecodeUvarint(encoded)
		if err != nil {
			t.Errorf("round trip failed for %d: encode then decode error: %v", v, err)
			continue
		}
		if decoded != v {
			t.Errorf("round trip failed for %d: got %d", v, decoded)
		}
		if n != len(encoded) {
			t.Errorf("round trip for %d: n=%d, len(encoded)=%d", v, n, len(encoded))
		}
	}
}

func TestSvarintRoundTrip(t *testing.T) {
	testValues := []int64{
		0, 1, -1, 2, -2, 63, -64, 64, -65,
		127, -128, 128, -129,
		math.MaxInt16, math.MinInt16,
		math.MaxInt32, math.MinInt32,
		math.MaxInt64, math.MinInt64,
		math.MaxInt64 - 1, math.MinInt64 + 1,
	}

	for _, v := range testValues {
		encoded := AppendSvarint(nil, v)
		decoded, n, err := DecodeSvarint(encoded)
		if err != nil {
			t.Errorf("round trip failed for %d: encode then decode error: %v", v, err)
			continue
		}
		if decoded != v {
			t.Errorf("round trip failed for %d: got %d", v, decoded)
		}
		if n != len(encoded) {
			t.Errorf("round trip for %d: n=%d, len(encoded)=%d", v, n, len(encoded))
		}
	}
}

func TestDecodeUvarintErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		err  error
	}{
		{"empty", []byte{}, ErrVarintTruncated},
		{"truncated_2byte", []byte{0x80}, ErrVarintTruncated},
		{"truncated_3byte", []byte{0x80, 0x80}, ErrVarintTruncated},
		{"truncated_10byte", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}, ErrVarintTruncated},
		{"overflow", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02}, ErrVarintOverflow},
		{"too_long", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}, ErrVarintTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := DecodeUvarint(tc.data)
			if err != tc.err {
				t.Errorf("DecodeUvarint(%v) error = %v, want %v", tc.data, err, tc.err)
			}
		})
	}
}

func TestDecodeSvarintErrors(t *testing.T) {
	// Same error cases apply since DecodeSvarint wraps DecodeUvarint
	tests := []struct {
		name string
		data []byte
		err  error
	}{
		{"empty", []byte{}, ErrVarintTruncated},
		{"truncated", []byte{0x80}, ErrVarintTruncated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := DecodeSvarint(tc.data)
			if err != tc.err {
				t.Errorf("DecodeSvarint(%v) error = %v, want %v", tc.data, err, tc.err)
			}
		})
	}
}

func TestUvarintSize(t *testing.T) {
	for _, tc := range uvarintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			size := UvarintSize(tc.value)
			if size != len(tc.expected) {
				t.Errorf("UvarintSize(%d) = %d, want %d", tc.value, size, len(tc.expected))
			}
		})
	}
}

func TestSvarintSize(t *testing.T) {
	for _, tc := range svarintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			size := SvarintSize(tc.value)
			if size != len(tc.expected) {
				t.Errorf("SvarintSize(%d) = %d, want %d", tc.value, size, len(tc.expected))
			}
		})
	}
}

func TestPutUvarint(t *testing.T) {
	for _, tc := range uvarintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, MaxVarintLen64)
			n := PutUvarint(buf, tc.value)
			if n != len(tc.expected) {
				t.Errorf("PutUvarint(%d) returned %d, want %d", tc.value, n, len(tc.expected))
			}
			if !bytes.Equal(buf[:n], tc.expected) {
				t.Errorf("PutUvarint(%d) = %v, want %v", tc.value, buf[:n], tc.expected)
			}
		})
	}
}

func TestPutSvarint(t *testing.T) {
	for _, tc := range svarintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, MaxVarintLen64)
			n := PutSvarint(buf, tc.value)
			if n != len(tc.expected) {
				t.Errorf("PutSvarint(%d) returned %d, want %d", tc.value, n, len(tc.expected))
			}
			if !bytes.Equal(buf[:n], tc.expected) {
				t.Errorf("PutSvarint(%d) = %v, want %v", tc.value, buf[:n], tc.expected)
			}
		})
	}
}

func TestDecodeUvarintWithTrailingData(t *testing.T) {
	// Ensure we correctly return bytes consumed when there's trailing data
	data := []byte{0xac, 0x02, 0xff, 0xff} // 300 followed by garbage
	value, n, err := DecodeUvarint(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 300 {
		t.Errorf("value = %d, want 300", value)
	}
	if n != 2 {
		t.Errorf("n = %d, want 2", n)
	}
}

func TestAppendToExistingBuffer(t *testing.T) {
	// Test that Append functions correctly extend existing buffers
	buf := []byte{0x01, 0x02, 0x03}
	buf = AppendUvarint(buf, 300)

	expected := []byte{0x01, 0x02, 0x03, 0xac, 0x02}
	if !bytes.Equal(buf, expected) {
		t.Errorf("AppendUvarint to existing buffer = %v, want %v", buf, expected)
	}
}

// Benchmarks

func BenchmarkAppendUvarint_Small(b *testing.B) {
	buf := make([]byte, 0, 16)
	for i := 0; i < b.N; i++ {
		buf = AppendUvarint(buf[:0], 127)
	}
}

func BenchmarkAppendUvarint_Medium(b *testing.B) {
	buf := make([]byte, 0, 16)
	for i := 0; i < b.N; i++ {
		buf = AppendUvarint(buf[:0], 16384)
	}
}

func BenchmarkAppendUvarint_Large(b *testing.B) {
	buf := make([]byte, 0, 16)
	for i := 0; i < b.N; i++ {
		buf = AppendUvarint(buf[:0], math.MaxUint64)
	}
}

func BenchmarkDecodeUvarint_Small(b *testing.B) {
	data := []byte{0x7f}
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeUvarint(data)
	}
}

func BenchmarkDecodeUvarint_Medium(b *testing.B) {
	data := []byte{0x80, 0x80, 0x01}
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeUvarint(data)
	}
}

func BenchmarkDecodeUvarint_Large(b *testing.B) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeUvarint(data)
	}
}

func BenchmarkAppendSvarint_Small(b *testing.B) {
	buf := make([]byte, 0, 16)
	for i := 0; i < b.N; i++ {
		buf = AppendSvarint(buf[:0], -64)
	}
}

func BenchmarkDecodeSvarint_Small(b *testing.B) {
	data := []byte{0x7f}
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeSvarint(data)
	}
}

func BenchmarkUvarintSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = UvarintSize(uint64(i))
	}
}

// Fuzz test
func FuzzUvarintRoundTrip(f *testing.F) {
	// Seed corpus
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(127))
	f.Add(uint64(128))
	f.Add(uint64(math.MaxUint32))
	f.Add(uint64(math.MaxUint64))

	f.Fuzz(func(t *testing.T, v uint64) {
		encoded := AppendUvarint(nil, v)
		decoded, n, err := DecodeUvarint(encoded)
		if err != nil {
			t.Fatalf("decode error for %d: %v", v, err)
		}
		if decoded != v {
			t.Fatalf("round trip failed: %d -> %v -> %d", v, encoded, decoded)
		}
		if n != len(encoded) {
			t.Fatalf("bytes consumed mismatch: %d vs %d", n, len(encoded))
		}
		if UvarintSize(v) != len(encoded) {
			t.Fatalf("size mismatch: %d vs %d", UvarintSize(v), len(encoded))
		}
	})
}

func FuzzSvarintRoundTrip(f *testing.F) {
	// Seed corpus
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(-1))
	f.Add(int64(math.MaxInt64))
	f.Add(int64(math.MinInt64))

	f.Fuzz(func(t *testing.T, v int64) {
		encoded := AppendSvarint(nil, v)
		decoded, n, err := DecodeSvarint(encoded)
		if err != nil {
			t.Fatalf("decode error for %d: %v", v, err)
		}
		if decoded != v {
			t.Fatalf("round trip failed: %d -> %v -> %d", v, encoded, decoded)
		}
		if n != len(encoded) {
			t.Fatalf("bytes consumed mismatch: %d vs %d", n, len(encoded))
		}
		if SvarintSize(v) != len(encoded) {
			t.Fatalf("size mismatch: %d vs %d", SvarintSize(v), len(encoded))
		}
	})
}
