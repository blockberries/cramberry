// Package integration provides cross-runtime interoperability tests.
//
// These tests verify that Go, TypeScript, and Rust runtimes produce
// identical binary encodings and can decode each other's output.
package integration

import (
	"bytes"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/cramberry/cramberry-go/pkg/cramberry"
	"github.com/cramberry/cramberry-go/tests/integration/gen"
)

// TestData contains all the test cases used for cross-runtime verification.
var TestData = struct {
	ScalarTypes     *interop.ScalarTypes
	RepeatedTypes   *interop.RepeatedTypes
	NestedMessage   *interop.NestedMessage
	ComplexTypes    *interop.ComplexTypes
	EdgeCases       *interop.EdgeCases
	AllFieldNumbers *interop.AllFieldNumbers
}{
	ScalarTypes: &interop.ScalarTypes{
		BoolVal:    true,
		Int32Val:   -42,
		Int64Val:   -9223372036854775807,
		Uint32Val:  4294967295,
		Uint64Val:  18446744073709551615,
		Float32Val: 3.14159,
		Float64Val: 2.718281828459045,
		StringVal:  "hello, cramberry!",
		BytesVal:   []byte{0xde, 0xad, 0xbe, 0xef},
	},
	RepeatedTypes: &interop.RepeatedTypes{
		Int32List:  []int32{1, -2, 3, -4, 5},
		StringList: []string{"alpha", "beta", "gamma"},
		BytesList:  [][]byte{{0x01, 0x02}, {0x03, 0x04, 0x05}},
	},
	NestedMessage: &interop.NestedMessage{
		Name:  "nested",
		Value: 123,
	},
	ComplexTypes: &interop.ComplexTypes{
		Status: interop.StatusActive,
		OptionalNested: &interop.NestedMessage{
			Name:  "optional",
			Value: 456,
		},
		RequiredNested: interop.NestedMessage{
			Name:  "required",
			Value: 789,
		},
		NestedList: []interop.NestedMessage{
			{Name: "first", Value: 1},
			{Name: "second", Value: 2},
		},
		StringIntMap: map[string]int32{
			"one":   1,
			"two":   2,
			"three": 3,
		},
		IntStringMap: map[int32]string{
			1: "one",
			2: "two",
			3: "three",
		},
	},
	EdgeCases: &interop.EdgeCases{
		ZeroInt:       0,
		NegativeOne:   -1,
		MaxInt32:      math.MaxInt32,
		MinInt32:      math.MinInt32,
		MaxInt64:      math.MaxInt64,
		MinInt64:      math.MinInt64,
		MaxUint32:     math.MaxUint32,
		MaxUint64:     math.MaxUint64,
		EmptyString:   "",
		UnicodeString: "Hello, 世界! 🎉",
		EmptyBytes:    []byte{},
	},
	AllFieldNumbers: &interop.AllFieldNumbers{
		Field1:    100,
		Field15:   1500,
		Field16:   1600,
		Field127:  12700,
		Field128:  12800,
		Field1000: 100000,
	},
}

const goldenDir = "../golden"

// TestScalarTypesEncodeDecode tests encoding and decoding of scalar types.
func TestScalarTypesEncodeDecode(t *testing.T) {
	data, err := cramberry.Marshal(TestData.ScalarTypes)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("ScalarTypes encoded size: %d bytes", len(data))
	t.Logf("ScalarTypes hex: %s", hex.EncodeToString(data))

	var decoded interop.ScalarTypes
	if err := cramberry.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify all fields
	if decoded.BoolVal != TestData.ScalarTypes.BoolVal {
		t.Errorf("BoolVal mismatch: got %v, want %v", decoded.BoolVal, TestData.ScalarTypes.BoolVal)
	}
	if decoded.Int32Val != TestData.ScalarTypes.Int32Val {
		t.Errorf("Int32Val mismatch: got %v, want %v", decoded.Int32Val, TestData.ScalarTypes.Int32Val)
	}
	if decoded.Int64Val != TestData.ScalarTypes.Int64Val {
		t.Errorf("Int64Val mismatch: got %v, want %v", decoded.Int64Val, TestData.ScalarTypes.Int64Val)
	}
	if decoded.Uint32Val != TestData.ScalarTypes.Uint32Val {
		t.Errorf("Uint32Val mismatch: got %v, want %v", decoded.Uint32Val, TestData.ScalarTypes.Uint32Val)
	}
	if decoded.Uint64Val != TestData.ScalarTypes.Uint64Val {
		t.Errorf("Uint64Val mismatch: got %v, want %v", decoded.Uint64Val, TestData.ScalarTypes.Uint64Val)
	}
	if decoded.Float32Val != TestData.ScalarTypes.Float32Val {
		t.Errorf("Float32Val mismatch: got %v, want %v", decoded.Float32Val, TestData.ScalarTypes.Float32Val)
	}
	if decoded.Float64Val != TestData.ScalarTypes.Float64Val {
		t.Errorf("Float64Val mismatch: got %v, want %v", decoded.Float64Val, TestData.ScalarTypes.Float64Val)
	}
	if decoded.StringVal != TestData.ScalarTypes.StringVal {
		t.Errorf("StringVal mismatch: got %v, want %v", decoded.StringVal, TestData.ScalarTypes.StringVal)
	}
	if !bytes.Equal(decoded.BytesVal, TestData.ScalarTypes.BytesVal) {
		t.Errorf("BytesVal mismatch: got %v, want %v", decoded.BytesVal, TestData.ScalarTypes.BytesVal)
	}
}

// TestRepeatedTypesEncodeDecode tests encoding and decoding of repeated types.
func TestRepeatedTypesEncodeDecode(t *testing.T) {
	data, err := cramberry.Marshal(TestData.RepeatedTypes)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("RepeatedTypes encoded size: %d bytes", len(data))
	t.Logf("RepeatedTypes hex: %s", hex.EncodeToString(data))

	var decoded interop.RepeatedTypes
	if err := cramberry.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify int32 list
	if len(decoded.Int32List) != len(TestData.RepeatedTypes.Int32List) {
		t.Errorf("Int32List length mismatch: got %d, want %d", len(decoded.Int32List), len(TestData.RepeatedTypes.Int32List))
	}
	for i, v := range TestData.RepeatedTypes.Int32List {
		if decoded.Int32List[i] != v {
			t.Errorf("Int32List[%d] mismatch: got %d, want %d", i, decoded.Int32List[i], v)
		}
	}

	// Verify string list
	if len(decoded.StringList) != len(TestData.RepeatedTypes.StringList) {
		t.Errorf("StringList length mismatch: got %d, want %d", len(decoded.StringList), len(TestData.RepeatedTypes.StringList))
	}
	for i, v := range TestData.RepeatedTypes.StringList {
		if decoded.StringList[i] != v {
			t.Errorf("StringList[%d] mismatch: got %q, want %q", i, decoded.StringList[i], v)
		}
	}
}

// TestNestedMessageEncodeDecode tests encoding and decoding of nested messages.
func TestNestedMessageEncodeDecode(t *testing.T) {
	data, err := cramberry.Marshal(TestData.NestedMessage)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("NestedMessage encoded size: %d bytes", len(data))
	t.Logf("NestedMessage hex: %s", hex.EncodeToString(data))

	var decoded interop.NestedMessage
	if err := cramberry.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Name != TestData.NestedMessage.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, TestData.NestedMessage.Name)
	}
	if decoded.Value != TestData.NestedMessage.Value {
		t.Errorf("Value mismatch: got %d, want %d", decoded.Value, TestData.NestedMessage.Value)
	}
}

// TestComplexTypesEncodeDecode tests encoding and decoding of complex types.
func TestComplexTypesEncodeDecode(t *testing.T) {
	data, err := cramberry.Marshal(TestData.ComplexTypes)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("ComplexTypes encoded size: %d bytes", len(data))
	t.Logf("ComplexTypes hex: %s", hex.EncodeToString(data))

	var decoded interop.ComplexTypes
	if err := cramberry.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Status != TestData.ComplexTypes.Status {
		t.Errorf("Status mismatch: got %v, want %v", decoded.Status, TestData.ComplexTypes.Status)
	}

	if decoded.OptionalNested == nil {
		t.Error("OptionalNested is nil, expected value")
	} else {
		if decoded.OptionalNested.Name != TestData.ComplexTypes.OptionalNested.Name {
			t.Errorf("OptionalNested.Name mismatch")
		}
	}

	if decoded.RequiredNested.Name != TestData.ComplexTypes.RequiredNested.Name {
		t.Errorf("RequiredNested.Name mismatch")
	}

	if len(decoded.NestedList) != len(TestData.ComplexTypes.NestedList) {
		t.Errorf("NestedList length mismatch")
	}

	// Verify maps (note: map ordering may differ, but values should match)
	if len(decoded.StringIntMap) != len(TestData.ComplexTypes.StringIntMap) {
		t.Errorf("StringIntMap length mismatch")
	}
	for k, v := range TestData.ComplexTypes.StringIntMap {
		if decoded.StringIntMap[k] != v {
			t.Errorf("StringIntMap[%q] mismatch: got %d, want %d", k, decoded.StringIntMap[k], v)
		}
	}
}

// TestEdgeCasesEncodeDecode tests encoding and decoding of edge case values.
func TestEdgeCasesEncodeDecode(t *testing.T) {
	data, err := cramberry.Marshal(TestData.EdgeCases)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("EdgeCases encoded size: %d bytes", len(data))
	t.Logf("EdgeCases hex: %s", hex.EncodeToString(data))

	var decoded interop.EdgeCases
	if err := cramberry.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ZeroInt != 0 {
		t.Errorf("ZeroInt mismatch: got %d, want 0", decoded.ZeroInt)
	}
	if decoded.NegativeOne != -1 {
		t.Errorf("NegativeOne mismatch: got %d, want -1", decoded.NegativeOne)
	}
	if decoded.MaxInt32 != math.MaxInt32 {
		t.Errorf("MaxInt32 mismatch: got %d, want %d", decoded.MaxInt32, math.MaxInt32)
	}
	if decoded.MinInt32 != math.MinInt32 {
		t.Errorf("MinInt32 mismatch: got %d, want %d", decoded.MinInt32, math.MinInt32)
	}
	if decoded.MaxInt64 != math.MaxInt64 {
		t.Errorf("MaxInt64 mismatch: got %d, want %d", decoded.MaxInt64, math.MaxInt64)
	}
	if decoded.MinInt64 != math.MinInt64 {
		t.Errorf("MinInt64 mismatch: got %d, want %d", decoded.MinInt64, math.MinInt64)
	}
	if decoded.MaxUint32 != math.MaxUint32 {
		t.Errorf("MaxUint32 mismatch")
	}
	if decoded.MaxUint64 != math.MaxUint64 {
		t.Errorf("MaxUint64 mismatch")
	}
	if decoded.UnicodeString != TestData.EdgeCases.UnicodeString {
		t.Errorf("UnicodeString mismatch: got %q, want %q", decoded.UnicodeString, TestData.EdgeCases.UnicodeString)
	}
}

// TestAllFieldNumbersEncodeDecode tests encoding and decoding with various field numbers.
func TestAllFieldNumbersEncodeDecode(t *testing.T) {
	data, err := cramberry.Marshal(TestData.AllFieldNumbers)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("AllFieldNumbers encoded size: %d bytes", len(data))
	t.Logf("AllFieldNumbers hex: %s", hex.EncodeToString(data))

	var decoded interop.AllFieldNumbers
	if err := cramberry.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Field1 != TestData.AllFieldNumbers.Field1 {
		t.Errorf("Field1 mismatch")
	}
	if decoded.Field15 != TestData.AllFieldNumbers.Field15 {
		t.Errorf("Field15 mismatch")
	}
	if decoded.Field16 != TestData.AllFieldNumbers.Field16 {
		t.Errorf("Field16 mismatch")
	}
	if decoded.Field127 != TestData.AllFieldNumbers.Field127 {
		t.Errorf("Field127 mismatch")
	}
	if decoded.Field128 != TestData.AllFieldNumbers.Field128 {
		t.Errorf("Field128 mismatch")
	}
	if decoded.Field1000 != TestData.AllFieldNumbers.Field1000 {
		t.Errorf("Field1000 mismatch")
	}
}

// TestGenerateGoldenFiles generates golden byte files for cross-runtime testing.
// Run with: go test -v -run TestGenerateGoldenFiles -generate-golden
func TestGenerateGoldenFiles(t *testing.T) {
	if os.Getenv("GENERATE_GOLDEN") != "1" {
		t.Skip("Set GENERATE_GOLDEN=1 to regenerate golden files")
	}

	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("Failed to create golden dir: %v", err)
	}

	testCases := []struct {
		name string
		data interface{}
	}{
		{"scalar_types", TestData.ScalarTypes},
		{"repeated_types", TestData.RepeatedTypes},
		{"nested_message", TestData.NestedMessage},
		{"complex_types", TestData.ComplexTypes},
		{"edge_cases", TestData.EdgeCases},
		{"all_field_numbers", TestData.AllFieldNumbers},
	}

	for _, tc := range testCases {
		data, err := cramberry.Marshal(tc.data)
		if err != nil {
			t.Errorf("Failed to marshal %s: %v", tc.name, err)
			continue
		}

		path := filepath.Join(goldenDir, tc.name+".bin")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Errorf("Failed to write %s: %v", path, err)
			continue
		}

		// Also write hex for easier inspection
		hexPath := filepath.Join(goldenDir, tc.name+".hex")
		if err := os.WriteFile(hexPath, []byte(hex.EncodeToString(data)), 0644); err != nil {
			t.Errorf("Failed to write %s: %v", hexPath, err)
		}

		t.Logf("Generated %s (%d bytes)", path, len(data))
	}
}

// TestVerifyGoldenFiles verifies that current encoding matches golden files.
func TestVerifyGoldenFiles(t *testing.T) {
	testCases := []struct {
		name string
		data interface{}
	}{
		{"scalar_types", TestData.ScalarTypes},
		{"repeated_types", TestData.RepeatedTypes},
		{"nested_message", TestData.NestedMessage},
		{"complex_types", TestData.ComplexTypes},
		{"edge_cases", TestData.EdgeCases},
		{"all_field_numbers", TestData.AllFieldNumbers},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(goldenDir, tc.name+".bin")
			golden, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s (run with GENERATE_GOLDEN=1 to create)", path)
				return
			}
			if err != nil {
				t.Fatalf("Failed to read golden file: %v", err)
			}

			encoded, err := cramberry.Marshal(tc.data)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			if !bytes.Equal(encoded, golden) {
				t.Errorf("Encoding mismatch for %s\nGot:  %s\nWant: %s",
					tc.name, hex.EncodeToString(encoded), hex.EncodeToString(golden))
			}
		})
	}
}
