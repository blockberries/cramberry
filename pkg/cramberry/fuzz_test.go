//go:build go1.18

package cramberry

import (
	"math"
	"reflect"
	"testing"
)

// FuzzUnmarshalBytes tests that Unmarshal never panics on arbitrary input.
func FuzzUnmarshalBytes(f *testing.F) {
	// Seed corpus with valid messages
	f.Add([]byte{}) // Empty
	f.Add([]byte{0x00})
	f.Add([]byte{0x08, 0x01})                                      // Field 1, varint 1
	f.Add([]byte{0x12, 0x05, 'h', 'e', 'l', 'l', 'o'})             // Field 2, string "hello"
	f.Add([]byte{0x08, 0x01, 0x12, 0x05, 'h', 'e', 'l', 'l', 'o'}) // Multiple fields

	// Add edge cases
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}) // Max varint

	f.Fuzz(func(t *testing.T, data []byte) {
		type TestMessage struct {
			ID      int64   `cramberry:"1"`
			Name    string  `cramberry:"2"`
			Value   float64 `cramberry:"3"`
			Enabled bool    `cramberry:"4"`
		}

		var msg TestMessage
		// Should never panic, only return error
		_ = Unmarshal(data, &msg)
	})
}

// FuzzReaderVarint tests that ReadUvarint never panics.
func FuzzReaderVarint(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x00})
	f.Add([]byte{0x7F})
	f.Add([]byte{0x80, 0x01})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01})
	f.Add([]byte{0x80}) // Truncated

	f.Fuzz(func(t *testing.T, data []byte) {
		r := NewReader(data)
		// Should never panic
		_ = r.ReadUvarint()
	})
}

// FuzzReaderString tests that ReadString never panics.
func FuzzReaderString(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x00})                          // Empty string
	f.Add([]byte{0x05, 'h', 'e', 'l', 'l', 'o'}) // Valid string
	f.Add([]byte{0x80, 0x01})                    // Length but no data
	f.Add([]byte{0xFF})                          // Truncated length

	f.Fuzz(func(t *testing.T, data []byte) {
		r := NewReader(data)
		// Should never panic
		_ = r.ReadString()
	})
}

// FuzzMarshalRoundTrip tests that Marshal/Unmarshal round-trips correctly.
func FuzzMarshalRoundTrip(f *testing.F) {
	// Seed corpus with various integers
	f.Add(int64(0), "")
	f.Add(int64(1), "hello")
	f.Add(int64(-1), "world")
	f.Add(int64(math.MaxInt64), "max")
	f.Add(int64(math.MinInt64), "min")

	f.Fuzz(func(t *testing.T, id int64, name string) {
		type TestMessage struct {
			ID   int64  `cramberry:"1"`
			Name string `cramberry:"2"`
		}

		original := TestMessage{ID: id, Name: name}

		// Marshal
		data, err := Marshal(&original)
		if err != nil {
			return // Skip invalid inputs
		}

		// Unmarshal
		var decoded TestMessage
		err = Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		// Compare
		if !reflect.DeepEqual(original, decoded) {
			t.Fatalf("Round-trip failed: got %+v, want %+v", decoded, original)
		}
	})
}

// FuzzWriterReader tests Writer/Reader round-trip.
func FuzzWriterReader(f *testing.F) {
	f.Add(int64(0), int64(0), uint64(0), uint64(0))
	f.Add(int64(1), int64(1), uint64(1), uint64(1))
	f.Add(int64(-1), int64(-1), uint64(math.MaxUint32), uint64(math.MaxUint64))

	f.Fuzz(func(t *testing.T, i32 int64, i64 int64, u32 uint64, u64 uint64) {
		w := NewWriter()
		w.WriteSvarint(i32)
		w.WriteSvarint(i64)
		w.WriteUvarint(u32)
		w.WriteUvarint(u64)

		if w.Err() != nil {
			t.Fatalf("Writer error: %v", w.Err())
		}

		r := NewReader(w.Bytes())

		gotI32 := r.ReadSvarint()
		gotI64 := r.ReadSvarint()
		gotU32 := r.ReadUvarint()
		gotU64 := r.ReadUvarint()

		if r.Err() != nil {
			t.Fatalf("Reader error: %v", r.Err())
		}

		if gotI32 != i32 {
			t.Errorf("int32: got %d, want %d", gotI32, i32)
		}
		if gotI64 != i64 {
			t.Errorf("int64: got %d, want %d", gotI64, i64)
		}
		if gotU32 != u32 {
			t.Errorf("uint32: got %d, want %d", gotU32, u32)
		}
		if gotU64 != u64 {
			t.Errorf("uint64: got %d, want %d", gotU64, u64)
		}
	})
}

// FuzzFloatRoundTrip tests float encoding round-trip.
func FuzzFloatRoundTrip(f *testing.F) {
	f.Add(float32(0), float64(0))
	f.Add(float32(1.5), float64(1.5))
	f.Add(float32(-1.5), float64(-1.5))
	f.Add(float32(math.MaxFloat32), float64(math.MaxFloat64))
	f.Add(float32(math.SmallestNonzeroFloat32), float64(math.SmallestNonzeroFloat64))

	f.Fuzz(func(t *testing.T, f32 float32, f64 float64) {
		w := NewWriter()
		w.WriteFloat32(f32)
		w.WriteFloat64(f64)

		if w.Err() != nil {
			t.Fatalf("Writer error: %v", w.Err())
		}

		r := NewReader(w.Bytes())
		gotF32 := r.ReadFloat32()
		gotF64 := r.ReadFloat64()

		if r.Err() != nil {
			t.Fatalf("Reader error: %v", r.Err())
		}

		// Handle NaN specially - NaN != NaN
		if math.IsNaN(float64(f32)) {
			if !math.IsNaN(float64(gotF32)) {
				t.Errorf("float32 NaN: got %v, want NaN", gotF32)
			}
		} else if gotF32 != f32 {
			t.Errorf("float32: got %v, want %v", gotF32, f32)
		}

		if math.IsNaN(f64) {
			if !math.IsNaN(gotF64) {
				t.Errorf("float64 NaN: got %v, want NaN", gotF64)
			}
		} else if gotF64 != f64 {
			t.Errorf("float64: got %v, want %v", gotF64, f64)
		}
	})
}
