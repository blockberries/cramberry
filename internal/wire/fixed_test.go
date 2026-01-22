package wire

import (
	"bytes"
	"math"
	"testing"
)

func TestAppendFixed32(t *testing.T) {
	tests := []struct {
		name     string
		value    uint32
		expected []byte
	}{
		{"zero", 0, []byte{0x00, 0x00, 0x00, 0x00}},
		{"one", 1, []byte{0x01, 0x00, 0x00, 0x00}},
		{"256", 256, []byte{0x00, 0x01, 0x00, 0x00}},
		{"0x12345678", 0x12345678, []byte{0x78, 0x56, 0x34, 0x12}},
		{"max_uint32", math.MaxUint32, []byte{0xff, 0xff, 0xff, 0xff}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendFixed32(nil, tc.value)
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("AppendFixed32(%d) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

func TestAppendFixed64(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		expected []byte
	}{
		{"zero", 0, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"one", 1, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"0x123456789ABCDEF0", 0x123456789ABCDEF0, []byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}},
		{"max_uint64", math.MaxUint64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendFixed64(nil, tc.value)
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("AppendFixed64(%d) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

func TestDecodeFixed32(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint32
	}{
		{"zero", []byte{0x00, 0x00, 0x00, 0x00}, 0},
		{"one", []byte{0x01, 0x00, 0x00, 0x00}, 1},
		{"0x12345678", []byte{0x78, 0x56, 0x34, 0x12}, 0x12345678},
		{"max_uint32", []byte{0xff, 0xff, 0xff, 0xff}, math.MaxUint32},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeFixed32(tc.data)
			if err != nil {
				t.Fatalf("DecodeFixed32(%v) error: %v", tc.data, err)
			}
			if result != tc.expected {
				t.Errorf("DecodeFixed32(%v) = %d, want %d", tc.data, result, tc.expected)
			}
		})
	}
}

func TestDecodeFixed64(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint64
	}{
		{"zero", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 0},
		{"one", []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 1},
		{"max_uint64", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, math.MaxUint64},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeFixed64(tc.data)
			if err != nil {
				t.Fatalf("DecodeFixed64(%v) error: %v", tc.data, err)
			}
			if result != tc.expected {
				t.Errorf("DecodeFixed64(%v) = %d, want %d", tc.data, result, tc.expected)
			}
		})
	}
}

func TestDecodeFixed32Error(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"one_byte", []byte{0x01}},
		{"three_bytes", []byte{0x01, 0x02, 0x03}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeFixed32(tc.data)
			if err == nil {
				t.Errorf("DecodeFixed32(%v) should return error", tc.data)
			}
		})
	}
}

func TestDecodeFixed64Error(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"one_byte", []byte{0x01}},
		{"seven_bytes", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeFixed64(tc.data)
			if err == nil {
				t.Errorf("DecodeFixed64(%v) should return error", tc.data)
			}
		})
	}
}

func TestPutFixed32(t *testing.T) {
	buf := make([]byte, 4)
	PutFixed32(buf, 0x12345678)
	expected := []byte{0x78, 0x56, 0x34, 0x12}
	if !bytes.Equal(buf, expected) {
		t.Errorf("PutFixed32 = %v, want %v", buf, expected)
	}
}

func TestPutFixed64(t *testing.T) {
	buf := make([]byte, 8)
	PutFixed64(buf, 0x123456789ABCDEF0)
	expected := []byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}
	if !bytes.Equal(buf, expected) {
		t.Errorf("PutFixed64 = %v, want %v", buf, expected)
	}
}

// Float32 tests

func TestAppendFloat32(t *testing.T) {
	tests := []struct {
		name     string
		value    float32
		expected []byte
	}{
		{"zero", 0.0, []byte{0x00, 0x00, 0x00, 0x00}},
		{"one", 1.0, []byte{0x00, 0x00, 0x80, 0x3f}},              // IEEE 754: 0x3F800000
		{"minus_one", -1.0, []byte{0x00, 0x00, 0x80, 0xbf}},       // IEEE 754: 0xBF800000
		{"pi_approx", float32(3.14), []byte{0xc3, 0xf5, 0x48, 0x40}}, // Approximate
		{"pos_inf", float32(math.Inf(1)), []byte{0x00, 0x00, 0x80, 0x7f}},  // 0x7F800000
		{"neg_inf", float32(math.Inf(-1)), []byte{0x00, 0x00, 0x80, 0xff}}, // 0xFF800000
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendFloat32(nil, tc.value)
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("AppendFloat32(%v) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

func TestFloat32NegativeZeroCanonicalization(t *testing.T) {
	// Negative zero should be encoded as positive zero
	negZero := math.Float32frombits(0x80000000)
	encoded := AppendFloat32(nil, negZero)
	expected := []byte{0x00, 0x00, 0x00, 0x00}

	if !bytes.Equal(encoded, expected) {
		t.Errorf("Negative zero encoded as %v, want %v", encoded, expected)
	}

	// Verify it decodes to positive zero
	decoded, err := DecodeFloat32(encoded)
	if err != nil {
		t.Fatalf("DecodeFloat32 error: %v", err)
	}
	if math.Float32bits(decoded) != 0 {
		t.Errorf("Decoded negative zero has bits %x, want 0", math.Float32bits(decoded))
	}
}

func TestFloat32NaNCanonicalization(t *testing.T) {
	// Various NaN bit patterns should all encode to canonical NaN
	nanPatterns := []uint32{
		0x7FC00000, // Quiet NaN (canonical)
		0x7FC00001, // Quiet NaN with payload
		0x7FFFFFFF, // Quiet NaN with max payload
		0x7F800001, // Signaling NaN
		0x7FBFFFFF, // Signaling NaN with max payload
		0xFFC00000, // Negative quiet NaN
		0xFFFFFFFF, // Negative NaN with payload
	}

	canonicalEncoded := []byte{0x00, 0x00, 0xC0, 0x7F} // 0x7FC00000 in little-endian

	for _, bits := range nanPatterns {
		nan := math.Float32frombits(bits)
		encoded := AppendFloat32(nil, nan)

		if !bytes.Equal(encoded, canonicalEncoded) {
			t.Errorf("NaN(0x%08X) encoded as %v, want %v (canonical)", bits, encoded, canonicalEncoded)
		}
	}
}

func TestFloat32RoundTrip(t *testing.T) {
	values := []float32{
		0, 1, -1, 0.5, -0.5,
		float32(math.Pi), float32(-math.Pi),
		float32(math.MaxFloat32), float32(-math.MaxFloat32),
		float32(math.SmallestNonzeroFloat32), float32(-math.SmallestNonzeroFloat32),
		float32(math.Inf(1)), float32(math.Inf(-1)),
	}

	for _, v := range values {
		encoded := AppendFloat32(nil, v)
		decoded, err := DecodeFloat32(encoded)
		if err != nil {
			t.Errorf("Float32 round trip error for %v: %v", v, err)
			continue
		}
		if decoded != v {
			t.Errorf("Float32 round trip: %v -> %v -> %v", v, encoded, decoded)
		}
	}
}

// Float64 tests

func TestAppendFloat64(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected []byte
	}{
		{"zero", 0.0, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"one", 1.0, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f}},  // 0x3FF0000000000000
		{"minus_one", -1.0, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0xbf}},
		{"pos_inf", math.Inf(1), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x7f}},
		{"neg_inf", math.Inf(-1), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0xff}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendFloat64(nil, tc.value)
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("AppendFloat64(%v) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

func TestFloat64NegativeZeroCanonicalization(t *testing.T) {
	negZero := math.Float64frombits(0x8000000000000000)
	encoded := AppendFloat64(nil, negZero)
	expected := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	if !bytes.Equal(encoded, expected) {
		t.Errorf("Negative zero encoded as %v, want %v", encoded, expected)
	}

	decoded, err := DecodeFloat64(encoded)
	if err != nil {
		t.Fatalf("DecodeFloat64 error: %v", err)
	}
	if math.Float64bits(decoded) != 0 {
		t.Errorf("Decoded negative zero has bits %x, want 0", math.Float64bits(decoded))
	}
}

func TestFloat64NaNCanonicalization(t *testing.T) {
	nanPatterns := []uint64{
		0x7FF8000000000000, // Quiet NaN (canonical)
		0x7FF8000000000001, // Quiet NaN with payload
		0x7FFFFFFFFFFFFFFF, // Quiet NaN with max payload
		0x7FF0000000000001, // Signaling NaN
		0xFFF8000000000000, // Negative quiet NaN
	}

	canonicalEncoded := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF8, 0x7F}

	for _, bits := range nanPatterns {
		nan := math.Float64frombits(bits)
		encoded := AppendFloat64(nil, nan)

		if !bytes.Equal(encoded, canonicalEncoded) {
			t.Errorf("NaN(0x%016X) encoded as %v, want %v (canonical)", bits, encoded, canonicalEncoded)
		}
	}
}

func TestFloat64RoundTrip(t *testing.T) {
	values := []float64{
		0, 1, -1, 0.5, -0.5,
		math.Pi, -math.Pi,
		math.MaxFloat64, -math.MaxFloat64,
		math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64,
		math.Inf(1), math.Inf(-1),
	}

	for _, v := range values {
		encoded := AppendFloat64(nil, v)
		decoded, err := DecodeFloat64(encoded)
		if err != nil {
			t.Errorf("Float64 round trip error for %v: %v", v, err)
			continue
		}
		if decoded != v {
			t.Errorf("Float64 round trip: %v -> %v -> %v", v, encoded, decoded)
		}
	}
}

// Complex number tests

func TestAppendComplex64(t *testing.T) {
	c := complex(float32(1.0), float32(2.0))
	encoded := AppendComplex64(nil, c)

	// Should be 8 bytes: real(4) + imag(4)
	if len(encoded) != 8 {
		t.Errorf("AppendComplex64 length = %d, want 8", len(encoded))
	}

	// First 4 bytes should be 1.0
	realBytes := encoded[:4]
	expectedReal := []byte{0x00, 0x00, 0x80, 0x3f} // 1.0 as float32
	if !bytes.Equal(realBytes, expectedReal) {
		t.Errorf("Complex64 real part = %v, want %v", realBytes, expectedReal)
	}

	// Next 4 bytes should be 2.0
	imagBytes := encoded[4:]
	expectedImag := []byte{0x00, 0x00, 0x00, 0x40} // 2.0 as float32
	if !bytes.Equal(imagBytes, expectedImag) {
		t.Errorf("Complex64 imag part = %v, want %v", imagBytes, expectedImag)
	}
}

func TestComplex64RoundTrip(t *testing.T) {
	values := []complex64{
		complex(0, 0),
		complex(1, 0),
		complex(0, 1),
		complex(1, 2),
		complex(-1, -2),
		complex(float32(math.Pi), float32(-math.E)),
	}

	for _, v := range values {
		encoded := AppendComplex64(nil, v)
		decoded, err := DecodeComplex64(encoded)
		if err != nil {
			t.Errorf("Complex64 round trip error for %v: %v", v, err)
			continue
		}
		if decoded != v {
			t.Errorf("Complex64 round trip: %v -> %v", v, decoded)
		}
	}
}

func TestAppendComplex128(t *testing.T) {
	c := complex(1.0, 2.0)
	encoded := AppendComplex128(nil, c)

	// Should be 16 bytes: real(8) + imag(8)
	if len(encoded) != 16 {
		t.Errorf("AppendComplex128 length = %d, want 16", len(encoded))
	}
}

func TestComplex128RoundTrip(t *testing.T) {
	values := []complex128{
		complex(0, 0),
		complex(1, 0),
		complex(0, 1),
		complex(1, 2),
		complex(-1, -2),
		complex(math.Pi, -math.E),
		complex(math.MaxFloat64, math.SmallestNonzeroFloat64),
	}

	for _, v := range values {
		encoded := AppendComplex128(nil, v)
		decoded, err := DecodeComplex128(encoded)
		if err != nil {
			t.Errorf("Complex128 round trip error for %v: %v", v, err)
			continue
		}
		if decoded != v {
			t.Errorf("Complex128 round trip: %v -> %v", v, decoded)
		}
	}
}

func TestDecodeComplex64Error(t *testing.T) {
	_, err := DecodeComplex64([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err == nil {
		t.Error("DecodeComplex64 with 7 bytes should error")
	}
}

func TestDecodeComplex128Error(t *testing.T) {
	_, err := DecodeComplex128(make([]byte, 15))
	if err == nil {
		t.Error("DecodeComplex128 with 15 bytes should error")
	}
}

// Helper function tests

func TestIsNaN32(t *testing.T) {
	if !IsNaN32(float32(math.NaN())) {
		t.Error("IsNaN32(NaN) should be true")
	}
	if IsNaN32(0) {
		t.Error("IsNaN32(0) should be false")
	}
	if IsNaN32(float32(math.Inf(1))) {
		t.Error("IsNaN32(+Inf) should be false")
	}
}

func TestIsNaN64(t *testing.T) {
	if !IsNaN64(math.NaN()) {
		t.Error("IsNaN64(NaN) should be true")
	}
	if IsNaN64(0) {
		t.Error("IsNaN64(0) should be false")
	}
	if IsNaN64(math.Inf(1)) {
		t.Error("IsNaN64(+Inf) should be false")
	}
}

func TestIsNegativeZero32(t *testing.T) {
	negZero := math.Float32frombits(0x80000000)
	if !IsNegativeZero32(negZero) {
		t.Error("IsNegativeZero32(-0) should be true")
	}
	if IsNegativeZero32(0) {
		t.Error("IsNegativeZero32(+0) should be false")
	}
}

func TestIsNegativeZero64(t *testing.T) {
	negZero := math.Float64frombits(0x8000000000000000)
	if !IsNegativeZero64(negZero) {
		t.Error("IsNegativeZero64(-0) should be true")
	}
	if IsNegativeZero64(0) {
		t.Error("IsNegativeZero64(+0) should be false")
	}
}

// Determinism test
func TestEncodingDeterminism(t *testing.T) {
	// Same value should always produce identical encoding
	values := []float64{0, 1, -1, math.Pi, math.Inf(1), math.Inf(-1)}

	for _, v := range values {
		first := AppendFloat64(nil, v)
		for i := 0; i < 100; i++ {
			second := AppendFloat64(nil, v)
			if !bytes.Equal(first, second) {
				t.Errorf("Non-deterministic encoding for %v: %v != %v", v, first, second)
				break
			}
		}
	}

	// NaN should always produce canonical encoding
	nan := math.NaN()
	firstNaN := AppendFloat64(nil, nan)
	for i := 0; i < 100; i++ {
		secondNaN := AppendFloat64(nil, nan)
		if !bytes.Equal(firstNaN, secondNaN) {
			t.Errorf("Non-deterministic NaN encoding: %v != %v", firstNaN, secondNaN)
			break
		}
	}
}

// Benchmarks

func BenchmarkAppendFixed32(b *testing.B) {
	buf := make([]byte, 0, 8)
	for i := 0; i < b.N; i++ {
		buf = AppendFixed32(buf[:0], 0x12345678)
	}
}

func BenchmarkAppendFixed64(b *testing.B) {
	buf := make([]byte, 0, 16)
	for i := 0; i < b.N; i++ {
		buf = AppendFixed64(buf[:0], 0x123456789ABCDEF0)
	}
}

func BenchmarkDecodeFixed32(b *testing.B) {
	data := []byte{0x78, 0x56, 0x34, 0x12}
	for i := 0; i < b.N; i++ {
		_, _ = DecodeFixed32(data)
	}
}

func BenchmarkDecodeFixed64(b *testing.B) {
	data := []byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}
	for i := 0; i < b.N; i++ {
		_, _ = DecodeFixed64(data)
	}
}

func BenchmarkAppendFloat64(b *testing.B) {
	buf := make([]byte, 0, 16)
	for i := 0; i < b.N; i++ {
		buf = AppendFloat64(buf[:0], math.Pi)
	}
}

func BenchmarkAppendFloat64_NaN(b *testing.B) {
	buf := make([]byte, 0, 16)
	nan := math.NaN()
	for i := 0; i < b.N; i++ {
		buf = AppendFloat64(buf[:0], nan)
	}
}

func BenchmarkDecodeFloat64(b *testing.B) {
	data := AppendFloat64(nil, math.Pi)
	for i := 0; i < b.N; i++ {
		_, _ = DecodeFloat64(data)
	}
}

// Fuzz tests

func FuzzFixed32RoundTrip(f *testing.F) {
	f.Add(uint32(0))
	f.Add(uint32(1))
	f.Add(uint32(math.MaxUint32))

	f.Fuzz(func(t *testing.T, v uint32) {
		encoded := AppendFixed32(nil, v)
		decoded, err := DecodeFixed32(encoded)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if decoded != v {
			t.Fatalf("round trip failed: %d -> %d", v, decoded)
		}
	})
}

func FuzzFixed64RoundTrip(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(math.MaxUint64))

	f.Fuzz(func(t *testing.T, v uint64) {
		encoded := AppendFixed64(nil, v)
		decoded, err := DecodeFixed64(encoded)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if decoded != v {
			t.Fatalf("round trip failed: %d -> %d", v, decoded)
		}
	})
}

func FuzzFloat64RoundTrip(f *testing.F) {
	f.Add(uint64(0))                          // 0.0
	f.Add(uint64(0x3FF0000000000000))          // 1.0
	f.Add(uint64(0x7FF0000000000000))          // +Inf
	f.Add(uint64(0x7FF8000000000000))          // NaN

	f.Fuzz(func(t *testing.T, bits uint64) {
		v := math.Float64frombits(bits)
		encoded := AppendFloat64(nil, v)
		decoded, err := DecodeFloat64(encoded)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		// For NaN, check that we get a NaN back (not necessarily same bits)
		if math.IsNaN(v) {
			if !math.IsNaN(decoded) {
				t.Fatalf("NaN round trip failed: got non-NaN %v", decoded)
			}
			return
		}

		// For -0, check that we get +0 back (canonicalization)
		if IsNegativeZero64(v) {
			if decoded != 0 || math.Float64bits(decoded) != 0 {
				t.Fatalf("negative zero should decode to positive zero")
			}
			return
		}

		// For all other values, should be identical
		if decoded != v {
			t.Fatalf("round trip failed: %v -> %v", v, decoded)
		}
	})
}
