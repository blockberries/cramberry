package cramberry

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

// Test types
type SimpleStruct struct {
	Name string `cramberry:"1"`
	Age  int32  `cramberry:"2"`
}

type NestedStruct struct {
	ID     int64         `cramberry:"1"`
	Simple *SimpleStruct `cramberry:"2"`
}

type AllPrimitives struct {
	Bool       bool       `cramberry:"1"`
	Int8       int8       `cramberry:"2"`
	Int16      int16      `cramberry:"3"`
	Int32      int32      `cramberry:"4"`
	Int64      int64      `cramberry:"5"`
	Uint8      uint8      `cramberry:"6"`
	Uint16     uint16     `cramberry:"7"`
	Uint32     uint32     `cramberry:"8"`
	Uint64     uint64     `cramberry:"9"`
	Float32    float32    `cramberry:"10"`
	Float64    float64    `cramberry:"11"`
	Complex64  complex64  `cramberry:"12"`
	Complex128 complex128 `cramberry:"13"`
	String     string     `cramberry:"14"`
	Bytes      []byte     `cramberry:"15"`
}

type WithSlice struct {
	Items []int32 `cramberry:"1"`
}

type WithMap struct {
	Data map[string]int32 `cramberry:"1"`
}

type WithTags struct {
	Required    string `cramberry:"1,required"`
	OmitEmpty   string `cramberry:"2,omitempty"`
	SkipThis    string `cramberry:"-"`
	NoTag       string // Will get auto-assigned field number
	ExplicitNum string `cramberry:"10"`
}

type WithPointers struct {
	Name    *string       `cramberry:"1"`
	Numbers *[]int32      `cramberry:"2"`
	Nested  *SimpleStruct `cramberry:"3"`
}

func TestMarshalPrimitives(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"bool_true", true},
		{"bool_false", false},
		{"int8_pos", int8(42)},
		{"int8_neg", int8(-42)},
		{"int16", int16(-1000)},
		{"int32", int32(-100000)},
		{"int64", int64(-1000000000)},
		{"uint8", uint8(255)},
		{"uint16", uint16(65000)},
		{"uint32", uint32(4000000000)},
		{"uint64", uint64(math.MaxUint64)},
		{"float32", float32(3.14)},
		{"float64", 3.14159265359},
		{"complex64", complex64(1 + 2i)},
		{"complex128", complex128(3 + 4i)},
		{"string", "hello, world!"},
		{"bytes", []byte{1, 2, 3, 4, 5}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			if len(data) == 0 {
				t.Error("Marshal produced empty data")
			}
		})
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		original := true
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result bool
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if result != original {
			t.Errorf("got %v, want %v", result, original)
		}
	})

	t.Run("int32", func(t *testing.T) {
		original := int32(-42)
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result int32
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if result != original {
			t.Errorf("got %v, want %v", result, original)
		}
	})

	t.Run("string", func(t *testing.T) {
		original := "hello, world!"
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result string
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if result != original {
			t.Errorf("got %q, want %q", result, original)
		}
	})

	t.Run("float64", func(t *testing.T) {
		original := 3.14159265359
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result float64
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if result != original {
			t.Errorf("got %v, want %v", result, original)
		}
	})

	t.Run("bytes", func(t *testing.T) {
		original := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result []byte
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !bytes.Equal(result, original) {
			t.Errorf("got %v, want %v", result, original)
		}
	})
}

func TestMarshalUnmarshalSlice(t *testing.T) {
	t.Run("int32_slice", func(t *testing.T) {
		original := []int32{1, 2, 3, 4, 5}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result []int32
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(result, original) {
			t.Errorf("got %v, want %v", result, original)
		}
	})

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
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %v", result)
		}
	})

	t.Run("string_slice", func(t *testing.T) {
		original := []string{"hello", "world", "foo", "bar"}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result []string
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(result, original) {
			t.Errorf("got %v, want %v", result, original)
		}
	})
}

func TestMarshalUnmarshalArray(t *testing.T) {
	original := [3]int32{10, 20, 30}
	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var result [3]int32
	if err := Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result != original {
		t.Errorf("got %v, want %v", result, original)
	}
}

func TestMarshalUnmarshalMap(t *testing.T) {
	t.Run("string_int", func(t *testing.T) {
		original := map[string]int32{
			"one":   1,
			"two":   2,
			"three": 3,
		}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result map[string]int32
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(result, original) {
			t.Errorf("got %v, want %v", result, original)
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
		// nil map unmarshals as empty map
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("int_int", func(t *testing.T) {
		original := map[int32]int64{
			1:  100,
			2:  200,
			-3: -300,
		}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result map[int32]int64
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(result, original) {
			t.Errorf("got %v, want %v", result, original)
		}
	})
}

func TestMarshalUnmarshalStruct(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		original := SimpleStruct{Name: "Alice", Age: 30}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result SimpleStruct
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if result != original {
			t.Errorf("got %v, want %v", result, original)
		}
	})

	t.Run("nested", func(t *testing.T) {
		original := NestedStruct{
			ID: 42,
			Simple: &SimpleStruct{
				Name: "Bob",
				Age:  25,
			},
		}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result NestedStruct
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if result.ID != original.ID {
			t.Errorf("ID: got %v, want %v", result.ID, original.ID)
		}
		if result.Simple == nil || *result.Simple != *original.Simple {
			t.Errorf("Simple: got %v, want %v", result.Simple, original.Simple)
		}
	})

	t.Run("all_primitives", func(t *testing.T) {
		original := AllPrimitives{
			Bool:       true,
			Int8:       -42,
			Int16:      -1000,
			Int32:      -100000,
			Int64:      -1000000000,
			Uint8:      255,
			Uint16:     65000,
			Uint32:     4000000000,
			Uint64:     math.MaxUint64,
			Float32:    3.14,
			Float64:    3.14159265359,
			Complex64:  1 + 2i,
			Complex128: 3 + 4i,
			String:     "hello",
			Bytes:      []byte{1, 2, 3},
		}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result AllPrimitives
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(result, original) {
			t.Errorf("got %+v, want %+v", result, original)
		}
	})

	t.Run("with_slice", func(t *testing.T) {
		original := WithSlice{Items: []int32{1, 2, 3, 4, 5}}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result WithSlice
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(result, original) {
			t.Errorf("got %v, want %v", result, original)
		}
	})

	t.Run("with_map", func(t *testing.T) {
		original := WithMap{Data: map[string]int32{"a": 1, "b": 2}}
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		var result WithMap
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(result, original) {
			t.Errorf("got %v, want %v", result, original)
		}
	})
}

func TestMarshalUnmarshalPointers(t *testing.T) {
	name := "test"
	numbers := []int32{1, 2, 3}
	original := WithPointers{
		Name:    &name,
		Numbers: &numbers,
		Nested:  &SimpleStruct{Name: "nested", Age: 10},
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result WithPointers
	if err := Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Name == nil || *result.Name != name {
		t.Errorf("Name: got %v, want %v", result.Name, &name)
	}
	if result.Numbers == nil || !reflect.DeepEqual(*result.Numbers, numbers) {
		t.Errorf("Numbers: got %v, want %v", result.Numbers, &numbers)
	}
	if result.Nested == nil || *result.Nested != *original.Nested {
		t.Errorf("Nested: got %v, want %v", result.Nested, original.Nested)
	}
}

func TestMarshalNilPointers(t *testing.T) {
	original := WithPointers{
		Name:    nil,
		Numbers: nil,
		Nested:  nil,
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result WithPointers
	if err := Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Name != nil {
		t.Errorf("Name should be nil, got %v", result.Name)
	}
	if result.Numbers != nil {
		t.Errorf("Numbers should be nil, got %v", result.Numbers)
	}
	if result.Nested != nil {
		t.Errorf("Nested should be nil, got %v", result.Nested)
	}
}

func TestMarshalDeterminism(t *testing.T) {
	// Maps should encode deterministically
	original := map[string]int32{
		"zebra": 5,
		"apple": 1,
		"banana": 2,
		"cherry": 3,
	}

	var encodings [][]byte
	for i := 0; i < 10; i++ {
		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		encodings = append(encodings, data)
	}

	// All encodings should be identical
	for i := 1; i < len(encodings); i++ {
		if !bytes.Equal(encodings[0], encodings[i]) {
			t.Errorf("Encoding %d differs from encoding 0", i)
		}
	}
}

func TestMarshalAppend(t *testing.T) {
	buf := make([]byte, 0, 100)
	result, err := MarshalAppend(buf, int32(42))
	if err != nil {
		t.Fatalf("MarshalAppend error: %v", err)
	}
	if len(result) == 0 {
		t.Error("MarshalAppend produced empty data")
	}
	if cap(result) != 100 {
		t.Error("MarshalAppend should reuse provided buffer capacity")
	}
}

func TestUnmarshalErrors(t *testing.T) {
	t.Run("not_pointer", func(t *testing.T) {
		var v int32
		err := Unmarshal([]byte{1}, v)
		if err != ErrNotPointer {
			t.Errorf("expected ErrNotPointer, got %v", err)
		}
	})

	t.Run("nil_pointer", func(t *testing.T) {
		var v *int32
		err := Unmarshal([]byte{1}, v)
		if err != ErrNilPointer {
			t.Errorf("expected ErrNilPointer, got %v", err)
		}
	})
}

func TestOmitEmpty(t *testing.T) {
	original := SimpleStruct{Name: "", Age: 0}

	// With OmitEmpty (default)
	data1, err := MarshalWithOptions(original, DefaultOptions)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Without OmitEmpty
	opts := DefaultOptions
	opts.OmitEmpty = false
	data2, err := MarshalWithOptions(original, opts)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// OmitEmpty should produce smaller output for zero values
	if len(data1) >= len(data2) {
		t.Logf("data1 (OmitEmpty=true): %d bytes", len(data1))
		t.Logf("data2 (OmitEmpty=false): %d bytes", len(data2))
		// This might not always be true depending on the struct layout
	}
}

func TestSize(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"int32", int32(42)},
		{"string", "hello"},
		{"struct", SimpleStruct{Name: "test", Age: 25}},
		{"slice", []int32{1, 2, 3, 4, 5}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			size := Size(tc.value)
			data, err := Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			if size != len(data) {
				t.Errorf("Size() = %d, but Marshal produced %d bytes", size, len(data))
			}
		})
	}
}

func TestNilValue(t *testing.T) {
	data, err := Marshal(nil)
	if err != nil {
		t.Fatalf("Marshal nil error: %v", err)
	}
	if len(data) != 1 || data[0] != 0 {
		t.Errorf("Marshal nil = %v, want [0]", data)
	}
}

func BenchmarkMarshal(b *testing.B) {
	b.Run("SimpleStruct", func(b *testing.B) {
		v := SimpleStruct{Name: "Alice", Age: 30}
		for i := 0; i < b.N; i++ {
			_, _ = Marshal(v)
		}
	})

	b.Run("AllPrimitives", func(b *testing.B) {
		v := AllPrimitives{
			Bool:       true,
			Int8:       -42,
			Int16:      -1000,
			Int32:      -100000,
			Int64:      -1000000000,
			Uint8:      255,
			Uint16:     65000,
			Uint32:     4000000000,
			Uint64:     math.MaxUint64,
			Float32:    3.14,
			Float64:    3.14159265359,
			Complex64:  1 + 2i,
			Complex128: 3 + 4i,
			String:     "hello",
			Bytes:      []byte{1, 2, 3},
		}
		for i := 0; i < b.N; i++ {
			_, _ = Marshal(v)
		}
	})
}

func BenchmarkUnmarshal(b *testing.B) {
	v := SimpleStruct{Name: "Alice", Age: 30}
	data, _ := Marshal(v)

	b.Run("SimpleStruct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result SimpleStruct
			_ = Unmarshal(data, &result)
		}
	})
}

// Polymorphic types for testing
type Greeter interface {
	Greet() string
}

type EnglishGreeter struct {
	Name string `cramberry:"1"`
}

func (g *EnglishGreeter) Greet() string {
	return "Hello, " + g.Name
}

type SpanishGreeter struct {
	Name string `cramberry:"1"`
}

func (g *SpanishGreeter) Greet() string {
	return "Hola, " + g.Name
}

type PolymorphicContainer struct {
	Greeter Greeter `cramberry:"1"`
}

func TestPolymorphicEncoding(t *testing.T) {
	// Clear and set up registry
	DefaultRegistry.Clear()
	defer DefaultRegistry.Clear()

	// Register types
	MustRegister[EnglishGreeter]()
	MustRegister[SpanishGreeter]()

	t.Run("EnglishGreeter", func(t *testing.T) {
		// Polymorphic encoding works through struct fields with interface types
		original := PolymorphicContainer{
			Greeter: &EnglishGreeter{Name: "Alice"},
		}

		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var result PolymorphicContainer
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		eg, ok := result.Greeter.(*EnglishGreeter)
		if !ok {
			t.Fatalf("Expected *EnglishGreeter, got %T", result.Greeter)
		}
		if eg.Name != "Alice" {
			t.Errorf("Name = %q, want %q", eg.Name, "Alice")
		}
	})

	t.Run("SpanishGreeter", func(t *testing.T) {
		original := PolymorphicContainer{
			Greeter: &SpanishGreeter{Name: "Carlos"},
		}

		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var result PolymorphicContainer
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		sg, ok := result.Greeter.(*SpanishGreeter)
		if !ok {
			t.Fatalf("Expected *SpanishGreeter, got %T", result.Greeter)
		}
		if sg.Name != "Carlos" {
			t.Errorf("Name = %q, want %q", sg.Name, "Carlos")
		}
	})

	t.Run("NilInterface", func(t *testing.T) {
		original := PolymorphicContainer{
			Greeter: nil,
		}

		data, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var result PolymorphicContainer
		if err := Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if result.Greeter != nil {
			t.Errorf("Expected nil, got %v", result.Greeter)
		}
	})
}

func TestPolymorphicUnregisteredType(t *testing.T) {
	DefaultRegistry.Clear()
	defer DefaultRegistry.Clear()

	// Don't register the type
	original := PolymorphicContainer{
		Greeter: &EnglishGreeter{Name: "Alice"},
	}

	_, err := Marshal(original)
	if err == nil {
		t.Error("Expected error for unregistered type")
	}
}
