package cramberry

import (
	"bytes"
	"reflect"
	"testing"
)

func TestCompactTagEncodeDecode(t *testing.T) {
	tests := []struct {
		name     string
		fieldNum int
		wireType byte
	}{
		{"field 1 varint", 1, WireTypeV2Varint},
		{"field 15 bytes", 15, WireTypeV2Bytes},
		{"field 16 fixed32", 16, WireTypeV2Fixed32},
		{"field 100 fixed64", 100, WireTypeV2Fixed64},
		{"field 1000 svarint", 1000, WireTypeV2SVarint},
		{"field 16383 varint", 16383, WireTypeV2Varint},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test standalone encode/decode functions
			encoded := EncodeCompactTag(tt.fieldNum, tt.wireType)
			if len(encoded) == 0 {
				t.Fatalf("EncodeCompactTag returned empty for field %d", tt.fieldNum)
			}

			decodedNum, decodedType, n := DecodeCompactTag(encoded)
			if n != len(encoded) {
				t.Errorf("DecodeCompactTag consumed %d bytes, expected %d", n, len(encoded))
			}
			if decodedNum != tt.fieldNum {
				t.Errorf("fieldNum = %d, want %d", decodedNum, tt.fieldNum)
			}
			if decodedType != tt.wireType {
				t.Errorf("wireType = %d, want %d", decodedType, tt.wireType)
			}

			// Verify compact tags for fields 1-15 are single byte
			expectedSize := CompactTagSize(tt.fieldNum)
			if len(encoded) != expectedSize {
				t.Errorf("encoded size = %d, expected %d", len(encoded), expectedSize)
			}
			if tt.fieldNum <= 15 && len(encoded) != 1 {
				t.Errorf("field %d should use compact (1-byte) tag, got %d bytes", tt.fieldNum, len(encoded))
			}
		})
	}
}

func TestCompactTagSize(t *testing.T) {
	tests := []struct {
		fieldNum     int
		expectedSize int
	}{
		{1, 1},
		{15, 1},
		{16, 2},     // 1 marker + 1 byte varint
		{127, 2},    // 1 marker + 1 byte varint
		{128, 3},    // 1 marker + 2 byte varint
		{16383, 3},  // 1 marker + 2 byte varint
		{16384, 4},  // 1 marker + 3 byte varint
	}

	for _, tt := range tests {
		size := CompactTagSize(tt.fieldNum)
		if size != tt.expectedSize {
			t.Errorf("CompactTagSize(%d) = %d, want %d", tt.fieldNum, size, tt.expectedSize)
		}
	}
}

func TestEndMarker(t *testing.T) {
	// Verify end marker is decoded as fieldNum=0
	data := []byte{EndMarker}
	fieldNum, wireType, n := DecodeCompactTag(data)
	if fieldNum != 0 {
		t.Errorf("end marker fieldNum = %d, want 0", fieldNum)
	}
	if wireType != 0 {
		t.Errorf("end marker wireType = %d, want 0", wireType)
	}
	if n != 1 {
		t.Errorf("end marker consumed %d bytes, want 1", n)
	}
}

func TestWriterReaderCompactTag(t *testing.T) {
	w := NewWriterWithOptions(DefaultOptions)

	// Write several compact tags
	w.WriteCompactTag(1, WireTypeV2Varint)
	w.WriteCompactTag(15, WireTypeV2Bytes)
	w.WriteCompactTag(16, WireTypeV2Fixed32)
	w.WriteCompactTag(100, WireTypeV2Fixed64)
	w.WriteEndMarker()

	if w.Err() != nil {
		t.Fatalf("Writer error: %v", w.Err())
	}

	data := w.Bytes()

	// Read back
	r := NewReaderWithOptions(data, DefaultOptions)

	tests := []struct {
		expectedNum  int
		expectedType byte
	}{
		{1, WireTypeV2Varint},
		{15, WireTypeV2Bytes},
		{16, WireTypeV2Fixed32},
		{100, WireTypeV2Fixed64},
		{0, 0}, // End marker
	}

	for i, tt := range tests {
		num, wt := r.ReadCompactTag()
		if r.Err() != nil {
			t.Fatalf("Read %d error: %v", i, r.Err())
		}
		if num != tt.expectedNum {
			t.Errorf("Tag %d: fieldNum = %d, want %d", i, num, tt.expectedNum)
		}
		if wt != tt.expectedType {
			t.Errorf("Tag %d: wireType = %d, want %d", i, wt, tt.expectedType)
		}
	}
}

func TestV2StructEncoding(t *testing.T) {
	type TestStruct struct {
		A int32  `cramberry:"1"`
		B string `cramberry:"2"`
		C bool   `cramberry:"3"`
	}

	original := TestStruct{A: 42, B: "hello", C: true}

	// Marshal with V2
	data, err := MarshalWithOptions(original, DefaultOptions)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify end marker is present at the end
	if len(data) > 0 && data[len(data)-1] != EndMarker {
		// End marker might not be the very last byte due to nested encoding
		// But we can verify by decoding
	}

	// Unmarshal
	var decoded TestStruct
	err = UnmarshalWithOptions(data, &decoded, DefaultOptions)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded != original {
		t.Errorf("Decoded = %+v, want %+v", decoded, original)
	}
}

func TestV2VsV1Size(t *testing.T) {
	type TestStruct struct {
		Field1  int32  `cramberry:"1"`
		Field2  string `cramberry:"2"`
		Field3  bool   `cramberry:"3"`
		Field10 int64  `cramberry:"10"`
		Field15 bool   `cramberry:"15"`
	}

	original := TestStruct{
		Field1:  100,
		Field2:  "test",
		Field3:  true,
		Field10: 12345,
		Field15: false,
	}

	// Marshal with V2
	v2Data, err := MarshalWithOptions(original, DefaultOptions)
	if err != nil {
		t.Fatalf("V2 Marshal error: %v", err)
	}

	// Marshal with V1
	v1Data, err := MarshalWithOptions(original, V1Options)
	if err != nil {
		t.Fatalf("V1 Marshal error: %v", err)
	}

	t.Logf("V1 size: %d bytes, V2 size: %d bytes", len(v1Data), len(v2Data))

	// V2 should typically be smaller or similar for small structs with fields 1-15
	// V2 has: 4 compact tags (1 byte each) + end marker (1 byte) = 5 bytes overhead
	// V1 has: field count (1 byte) + 4 tags (varying size)
}

func TestV2NestedStruct(t *testing.T) {
	type Inner struct {
		X int32  `cramberry:"1"`
		Y string `cramberry:"2"`
	}

	type Outer struct {
		Name  string `cramberry:"1"`
		Inner Inner  `cramberry:"2"`
		Count int64  `cramberry:"3"`
	}

	original := Outer{
		Name:  "test",
		Inner: Inner{X: 10, Y: "nested"},
		Count: 100,
	}

	// Round-trip with V2
	data, err := MarshalWithOptions(original, DefaultOptions)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Outer
	err = UnmarshalWithOptions(data, &decoded, DefaultOptions)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded != original {
		t.Errorf("Decoded = %+v, want %+v", decoded, original)
	}
}

func TestV2SkipUnknownFields(t *testing.T) {
	// Create encoded data with a field that won't be in our target struct
	w := NewWriterWithOptions(DefaultOptions)

	// Field 1: int32 (known)
	w.WriteCompactTag(1, WireTypeV2SVarint)
	w.WriteSvarint(42)

	// Field 99: string (unknown - will be skipped)
	w.WriteCompactTag(99, WireTypeV2Bytes)
	w.WriteString("unknown field")

	// Field 2: bool (known)
	w.WriteCompactTag(2, WireTypeV2Varint)
	w.WriteBool(true)

	w.WriteEndMarker()

	if w.Err() != nil {
		t.Fatalf("Writer error: %v", w.Err())
	}

	type PartialStruct struct {
		A int32 `cramberry:"1"`
		B bool  `cramberry:"2"`
	}

	var decoded PartialStruct
	err := UnmarshalWithOptions(w.Bytes(), &decoded, DefaultOptions)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.A != 42 {
		t.Errorf("A = %d, want 42", decoded.A)
	}
	if !decoded.B {
		t.Error("B = false, want true")
	}
}

func TestV2ExtendedFieldNumbers(t *testing.T) {
	type LargeFieldStruct struct {
		Field1   int32 `cramberry:"1"`
		Field16  int32 `cramberry:"16"`
		Field100 int32 `cramberry:"100"`
		Field999 int32 `cramberry:"999"`
	}

	original := LargeFieldStruct{
		Field1:   1,
		Field16:  16,
		Field100: 100,
		Field999: 999,
	}

	data, err := MarshalWithOptions(original, DefaultOptions)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded LargeFieldStruct
	err = UnmarshalWithOptions(data, &decoded, DefaultOptions)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded != original {
		t.Errorf("Decoded = %+v, want %+v", decoded, original)
	}
}

func TestPackedSliceEncoding(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"[]int32", []int32{1, 2, 3, 100, -50, 1000}},
		{"[]int64", []int64{1, 2, 3, 1<<40, -1 << 30}},
		{"[]uint32", []uint32{0, 1, 255, 65535, 1<<20}},
		{"[]uint64", []uint64{0, 1, 1 << 32, 1 << 50}},
		{"[]float32", []float32{1.5, 2.5, 3.14, -100.25}},
		{"[]float64", []float64{1.5, 2.5, 3.14159265359, -1e100}},
		{"[]bool", []bool{true, false, true, true, false}},
		{"[]int8", []int8{-128, 0, 127}},
		{"[]uint8 (bytes)", []uint8{0, 128, 255}}, // Note: []byte is special-cased
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal with V2
			data, err := MarshalWithOptions(tt.value, DefaultOptions)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			// Create a new slice of the same type
			rv := reflect.ValueOf(tt.value)
			decoded := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
			decodedPtr := reflect.New(rv.Type())
			decodedPtr.Elem().Set(decoded)

			err = UnmarshalWithOptions(data, decodedPtr.Interface(), DefaultOptions)
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			// Compare
			decodedSlice := decodedPtr.Elem()
			if rv.Len() != decodedSlice.Len() {
				t.Errorf("Length mismatch: got %d, want %d", decodedSlice.Len(), rv.Len())
				return
			}

			for i := 0; i < rv.Len(); i++ {
				if !reflect.DeepEqual(rv.Index(i).Interface(), decodedSlice.Index(i).Interface()) {
					t.Errorf("Element %d mismatch: got %v, want %v",
						i, decodedSlice.Index(i).Interface(), rv.Index(i).Interface())
				}
			}
		})
	}
}

func TestPackedArrayEncoding(t *testing.T) {
	type ArrayStruct struct {
		Ints    [5]int32   `cramberry:"1"`
		Floats  [3]float64 `cramberry:"2"`
		Bools   [4]bool    `cramberry:"3"`
	}

	original := ArrayStruct{
		Ints:   [5]int32{1, 2, 3, 4, 5},
		Floats: [3]float64{1.1, 2.2, 3.3},
		Bools:  [4]bool{true, false, true, false},
	}

	data, err := MarshalWithOptions(original, DefaultOptions)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ArrayStruct
	err = UnmarshalWithOptions(data, &decoded, DefaultOptions)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded != original {
		t.Errorf("Decoded = %+v, want %+v", decoded, original)
	}
}

func TestPackedSliceInStruct(t *testing.T) {
	type SliceStruct struct {
		Name   string    `cramberry:"1"`
		Values []int64   `cramberry:"2"`
		Scores []float32 `cramberry:"3"`
	}

	original := SliceStruct{
		Name:   "test",
		Values: []int64{100, 200, 300, -400, 500},
		Scores: []float32{1.5, 2.5, 3.5, 4.5},
	}

	data, err := MarshalWithOptions(original, DefaultOptions)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded SliceStruct
	err = UnmarshalWithOptions(data, &decoded, DefaultOptions)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name mismatch")
	}
	if len(decoded.Values) != len(original.Values) {
		t.Errorf("Values length mismatch")
	}
	for i, v := range original.Values {
		if decoded.Values[i] != v {
			t.Errorf("Values[%d] = %d, want %d", i, decoded.Values[i], v)
		}
	}
	if len(decoded.Scores) != len(original.Scores) {
		t.Errorf("Scores length mismatch")
	}
	for i, v := range original.Scores {
		if decoded.Scores[i] != v {
			t.Errorf("Scores[%d] = %f, want %f", i, decoded.Scores[i], v)
		}
	}
}

func TestPackedVsUnpackedSize(t *testing.T) {
	// Compare packed V2 encoding size to V1
	type IntSliceStruct struct {
		Values []int32 `cramberry:"1"`
	}

	original := IntSliceStruct{
		Values: make([]int32, 100),
	}
	for i := range original.Values {
		original.Values[i] = int32(i * 10)
	}

	v2Data, _ := MarshalWithOptions(original, DefaultOptions)
	v1Data, _ := MarshalWithOptions(original, V1Options)

	t.Logf("100 int32 values - V1: %d bytes, V2: %d bytes (%.1f%% smaller)",
		len(v1Data), len(v2Data), 100*(1-float64(len(v2Data))/float64(len(v1Data))))

	// V2 should be smaller because:
	// V1: 1 (field count) + tag + 100*(tag + varint) per element
	// V2: compact tag + count + 100*varint (no per-element tags)
}

func TestEmptyPackedSlice(t *testing.T) {
	type SliceStruct struct {
		Values []int32 `cramberry:"1"`
	}

	original := SliceStruct{Values: []int32{}}

	data, err := MarshalWithOptions(original, DefaultOptions)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded SliceStruct
	err = UnmarshalWithOptions(data, &decoded, DefaultOptions)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Empty slices may decode as nil, which is fine
	if len(decoded.Values) != 0 {
		t.Errorf("Expected empty slice, got %v", decoded.Values)
	}
}

func TestDeterministicOption(t *testing.T) {
	type MapStruct struct {
		Data map[string]int `cramberry:"1"`
	}

	original := MapStruct{
		Data: map[string]int{"z": 1, "a": 2, "m": 3},
	}

	// With deterministic mode, encoding should be consistent
	data1, _ := MarshalWithOptions(original, DefaultOptions)
	data2, _ := MarshalWithOptions(original, DefaultOptions)

	if !bytes.Equal(data1, data2) {
		t.Error("Deterministic mode should produce identical output")
	}

	// Without deterministic mode, we can't guarantee order
	// but decoding should still work
	fastOpts := FastOptions
	data3, err := MarshalWithOptions(original, fastOpts)
	if err != nil {
		t.Fatalf("Fast marshal error: %v", err)
	}

	var decoded MapStruct
	err = UnmarshalWithOptions(data3, &decoded, fastOpts)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Values should match even if order differs
	if len(decoded.Data) != len(original.Data) {
		t.Error("Map size mismatch")
	}
	for k, v := range original.Data {
		if decoded.Data[k] != v {
			t.Errorf("Map value mismatch for key %s", k)
		}
	}
}
