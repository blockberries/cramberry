package cramberry

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

// Edge case tests for REFACTORING item 32

// TestLargeStrings tests encoding/decoding of large strings.
func TestLargeStrings(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"64KB", 64 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a string of the specified size
			original := strings.Repeat("x", tc.size)

			data, err := Marshal(original)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var result string
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if result != original {
				t.Errorf("string mismatch: got length %d, want %d", len(result), len(original))
			}
		})
	}
}

// TestLargeBytes tests encoding/decoding of large byte slices.
func TestLargeBytes(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"64KB", 64 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a byte slice of the specified size
			original := make([]byte, tc.size)
			for i := range original {
				original[i] = byte(i % 256)
			}

			data, err := Marshal(original)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var result []byte
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if !bytes.Equal(result, original) {
				t.Errorf("bytes mismatch: got length %d, want %d", len(result), len(original))
			}
		})
	}
}

// DeepStruct is used to test deeply nested structures.
type DeepStruct struct {
	Value  int32       `cramberry:"1"`
	Nested *DeepStruct `cramberry:"2"`
}

// createDeepStruct creates a struct nested to the specified depth.
func createDeepStruct(depth int) *DeepStruct {
	if depth <= 0 {
		return nil
	}
	return &DeepStruct{
		Value:  int32(depth),
		Nested: createDeepStruct(depth - 1),
	}
}

// countDepth counts the depth of a DeepStruct.
func countDepth(s *DeepStruct) int {
	if s == nil {
		return 0
	}
	return 1 + countDepth(s.Nested)
}

// TestDeeplyNestedStructures tests encoding/decoding of deeply nested structures.
func TestDeeplyNestedStructures(t *testing.T) {
	tests := []struct {
		name  string
		depth int
	}{
		{"depth_10", 10},
		{"depth_50", 50},
		{"depth_100", 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original := createDeepStruct(tc.depth)

			data, err := Marshal(original)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var result DeepStruct
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			resultDepth := countDepth(&result)
			if resultDepth != tc.depth {
				t.Errorf("depth mismatch: got %d, want %d", resultDepth, tc.depth)
			}
		})
	}
}

// TestLargeSlices tests encoding/decoding of large slices.
func TestLargeSlices(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"100_elements", 100},
		{"1000_elements", 1000},
		{"10000_elements", 10000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original := make([]int32, tc.count)
			for i := range original {
				original[i] = int32(i)
			}

			data, err := Marshal(original)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var result []int32
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if len(result) != len(original) {
				t.Errorf("slice length mismatch: got %d, want %d", len(result), len(original))
			}
		})
	}
}

// TestLargeMaps tests encoding/decoding of large maps.
func TestLargeMaps(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"100_entries", 100},
		{"1000_entries", 1000},
		{"10000_entries", 10000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original := make(map[string]int32, tc.count)
			for i := 0; i < tc.count; i++ {
				original[strings.Repeat("k", i%100+1)+string(rune('a'+i%26))] = int32(i)
			}

			data, err := Marshal(original)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var result map[string]int32
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if len(result) != len(original) {
				t.Errorf("map size mismatch: got %d, want %d", len(result), len(original))
			}
		})
	}
}

// TestIntegerEdgeCases tests min/max integer values.
func TestIntegerEdgeCases(t *testing.T) {
	t.Run("int8", func(t *testing.T) {
		for _, v := range []int8{math.MinInt8, -1, 0, 1, math.MaxInt8} {
			data, err := Marshal(v)
			if err != nil {
				t.Fatalf("Marshal(%d) error: %v", v, err)
			}
			var result int8
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if result != v {
				t.Errorf("got %d, want %d", result, v)
			}
		}
	})

	t.Run("int16", func(t *testing.T) {
		for _, v := range []int16{math.MinInt16, -1, 0, 1, math.MaxInt16} {
			data, err := Marshal(v)
			if err != nil {
				t.Fatalf("Marshal(%d) error: %v", v, err)
			}
			var result int16
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if result != v {
				t.Errorf("got %d, want %d", result, v)
			}
		}
	})

	t.Run("int32", func(t *testing.T) {
		for _, v := range []int32{math.MinInt32, -1, 0, 1, math.MaxInt32} {
			data, err := Marshal(v)
			if err != nil {
				t.Fatalf("Marshal(%d) error: %v", v, err)
			}
			var result int32
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if result != v {
				t.Errorf("got %d, want %d", result, v)
			}
		}
	})

	t.Run("int64", func(t *testing.T) {
		for _, v := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
			data, err := Marshal(v)
			if err != nil {
				t.Fatalf("Marshal(%d) error: %v", v, err)
			}
			var result int64
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if result != v {
				t.Errorf("got %d, want %d", result, v)
			}
		}
	})

	t.Run("uint64_max", func(t *testing.T) {
		v := uint64(math.MaxUint64)
		data, err := Marshal(v)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result uint64
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if result != v {
			t.Errorf("got %d, want %d", result, v)
		}
	})
}

// TestFloatEdgeCases tests special float values.
func TestFloatEdgeCases(t *testing.T) {
	t.Run("float32", func(t *testing.T) {
		for _, v := range []float32{
			0,
			-0,
			1,
			-1,
			math.MaxFloat32,
			math.SmallestNonzeroFloat32,
			float32(math.Inf(1)),
			float32(math.Inf(-1)),
		} {
			data, err := Marshal(v)
			if err != nil {
				t.Fatalf("Marshal(%v) error: %v", v, err)
			}
			var result float32
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			// For infinity, direct comparison works
			if result != v && !(math.IsInf(float64(result), 0) && math.IsInf(float64(v), 0)) {
				t.Errorf("got %v, want %v", result, v)
			}
		}
	})

	t.Run("float32_nan", func(t *testing.T) {
		v := float32(math.NaN())
		data, err := Marshal(v)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result float32
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !math.IsNaN(float64(result)) {
			t.Errorf("expected NaN, got %v", result)
		}
	})

	t.Run("float64_special", func(t *testing.T) {
		for _, v := range []float64{
			0,
			-0,
			math.MaxFloat64,
			math.SmallestNonzeroFloat64,
			math.Inf(1),
			math.Inf(-1),
		} {
			data, err := Marshal(v)
			if err != nil {
				t.Fatalf("Marshal(%v) error: %v", v, err)
			}
			var result float64
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if result != v && !(math.IsInf(result, 0) && math.IsInf(v, 0)) {
				t.Errorf("got %v, want %v", result, v)
			}
		}
	})
}

// TestEmptyCollections tests empty slices and maps.
func TestEmptyCollections(t *testing.T) {
	t.Run("empty_slice", func(t *testing.T) {
		original := []int32{}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result []int32
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %v", result)
		}
	})

	t.Run("nil_slice", func(t *testing.T) {
		var original []int32 = nil
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result []int32
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		// Nil slice unmarshals to empty slice
		if result == nil || len(result) != 0 {
			t.Errorf("expected empty slice, got %v", result)
		}
	})

	t.Run("empty_map", func(t *testing.T) {
		original := map[string]int32{}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result map[string]int32
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("nil_map", func(t *testing.T) {
		var original map[string]int32 = nil
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result map[string]int32
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		// Nil map unmarshals to empty map
		if result == nil || len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("empty_string", func(t *testing.T) {
		original := ""
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result string
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("empty_bytes", func(t *testing.T) {
		original := []byte{}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result []byte
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty bytes, got %v", result)
		}
	})
}

// TestUnicodeStrings tests various Unicode string edge cases.
func TestUnicodeStrings(t *testing.T) {
	tests := []struct {
		name string
		str  string
	}{
		{"ascii", "hello, world!"},
		{"latin1", "café résumé naïve"},
		{"chinese", "你好世界"},
		{"japanese", "こんにちは"},
		{"korean", "안녕하세요"},
		{"cyrillic", "Привет мир"},
		{"arabic", "مرحبا بالعالم"},
		{"emoji", "Hello 👋 World 🌍 🎉"},
		{"mixed", "Hello, 世界! 🎉 مرحبا"},
		{"combining_chars", "e\u0301"}, // é as e + combining accent
		{"zero_width", "a\u200Bb"},     // zero-width space
		{"null_byte", "a\x00b"},        // embedded null
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := Marshal(tc.str)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			var result string
			if err := Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if result != tc.str {
				t.Errorf("got %q, want %q", result, tc.str)
			}
		})
	}
}

// TestMalformedInput tests handling of malformed input data.
func TestMalformedInput(t *testing.T) {
	t.Run("empty_input", func(t *testing.T) {
		var result int32
		err := Unmarshal([]byte{}, &result)
		// Empty input should fail
		if err == nil {
			t.Error("expected error for empty input")
		}
	})

	t.Run("truncated_varint", func(t *testing.T) {
		// Start of a varint that continues but has no more bytes
		data := []byte{0x80} // High bit set, expects continuation
		var result int32
		err := Unmarshal(data, &result)
		if err == nil {
			t.Error("expected error for truncated varint")
		}
	})

	t.Run("truncated_string", func(t *testing.T) {
		// Length prefix says 10 bytes but only 5 present
		data := []byte{0x0a, 'h', 'e', 'l', 'l', 'o'}
		var result string
		err := Unmarshal(data, &result)
		if err == nil {
			t.Error("expected error for truncated string")
		}
	})

	t.Run("nil_destination", func(t *testing.T) {
		data := []byte{0x01}
		err := Unmarshal(data, nil)
		if err == nil {
			t.Error("expected error for nil destination")
		}
	})

	t.Run("non_pointer_destination", func(t *testing.T) {
		data := []byte{0x01}
		var v int32
		err := Unmarshal(data, v)
		if err == nil {
			t.Error("expected error for non-pointer destination")
		}
	})
}

// FieldNumberStruct tests various field number edge cases.
type FieldNumberStruct struct {
	Field1     int32 `cramberry:"1"`
	Field15    int32 `cramberry:"15"`    // Single byte tag
	Field16    int32 `cramberry:"16"`    // Two byte tag (boundary)
	Field127   int32 `cramberry:"127"`   // Single byte tag max
	Field128   int32 `cramberry:"128"`   // Two byte tag (boundary)
	Field2047  int32 `cramberry:"2047"`  // Two byte tag max (11 bits)
	Field16383 int32 `cramberry:"16383"` // Larger tag
}

// TestFieldNumberBoundaries tests field numbers at boundaries.
func TestFieldNumberBoundaries(t *testing.T) {
	original := FieldNumberStruct{
		Field1:     1,
		Field15:    15,
		Field16:    16,
		Field127:   127,
		Field128:   128,
		Field2047:  2047,
		Field16383: 16383,
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result FieldNumberStruct
	if err := Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result != original {
		t.Errorf("field number roundtrip failed:\ngot:  %+v\nwant: %+v", result, original)
	}
}
