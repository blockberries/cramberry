package jsontest

import (
	"testing"
)

func TestScalarTypesToJSON(t *testing.T) {
	msg := &ScalarTypes{
		BoolVal:    true,
		Int8Val:    -42,
		Int16Val:   -1000,
		Int32Val:   -100000,
		Int64Val:   -9223372036854775807,
		Uint8Val:   255,
		Uint16Val:  65535,
		Uint32Val:  4294967295,
		Uint64Val:  18446744073709551615,
		Float32Val: 3.14159,
		Float64Val: 2.718281828459045,
		StringVal:  "hello, world!",
		BytesVal:   []byte{0xde, 0xad, 0xbe, 0xef},
	}

	jsonStr, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	t.Logf("JSON output: %s", jsonStr)

	// Verify it's valid JSON and can be decoded
	var decoded ScalarTypes
	if err := decoded.FromJSON(jsonStr); err != nil {
		t.Fatalf("FromJSON() error: %v", err)
	}

	// Verify round-trip
	jsonStr2, err := decoded.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() after FromJSON() error: %v", err)
	}

	if jsonStr != jsonStr2 {
		t.Errorf("Round-trip failed:\nOriginal: %s\nAfter:    %s", jsonStr, jsonStr2)
	}
}

func TestRequiredFieldsValidation(t *testing.T) {
	// Test ToJSON validation
	msg := &RequiredFields{
		OptionalField: "optional",
	}

	_, err := msg.ToJSON()
	if err == nil {
		t.Error("ToJSON() should fail for missing required fields")
	}

	// Test FromJSON validation
	jsonStr := `{"optional_field":"optional"}`
	var decoded RequiredFields
	err = decoded.FromJSON(jsonStr)
	if err == nil {
		t.Error("FromJSON() should fail for missing required fields")
	}
}

func TestEnumJSONEncoding(t *testing.T) {
	msg := &EnumTest{
		Status:   StatusActive,
		Statuses: []Status{StatusActive, StatusPending, StatusInactive},
	}

	jsonStr, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	t.Logf("Enum JSON: %s", jsonStr)

	// Verify enums are encoded as string names
	if string(jsonStr) != `{"status":"ACTIVE","statuses":["ACTIVE","PENDING","INACTIVE"]}` {
		t.Errorf("Enum encoding incorrect: %s", jsonStr)
	}

	// Verify round-trip
	var decoded EnumTest
	if err := decoded.FromJSON(jsonStr); err != nil {
		t.Fatalf("FromJSON() error: %v", err)
	}

	if decoded.Status != msg.Status {
		t.Errorf("Status mismatch: got %v, want %v", decoded.Status, msg.Status)
	}
}

func TestMapKeyOrdering(t *testing.T) {
	msg := &MapTypes{
		StringMap: map[string]string{
			"zebra":  "z",
			"apple":  "a",
			"banana": "b",
		},
		IntMap: map[string]int64{
			"z": 26,
			"a": 1,
			"b": 2,
		},
	}

	jsonStr, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	t.Logf("Map JSON: %s", jsonStr)

	// Verify keys are sorted lexicographically. The two int-keyed fields
	// (int_keyed, uint_keyed) are present-but-empty; the JSON encoder
	// emits them as `{}` because the encoder doesn't omit empty maps.
	expected := `{"string_map":{"apple":"a","banana":"b","zebra":"z"},"int_map":{"a":"1","b":"2","z":"26"},"int_keyed":{},"uint_keyed":{}}`
	if jsonStr != expected {
		t.Errorf("Map key ordering incorrect:\nGot:  %s\nWant: %s", jsonStr, expected)
	}
}

func TestNestedMessage(t *testing.T) {
	msg := &Person{
		Name: "Alice",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Springfield",
			Zip:    "12345",
		},
		Emails: []string{"alice@example.com", "alice@work.com"},
	}

	jsonStr, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	t.Logf("Nested message JSON: %s", jsonStr)

	// Verify round-trip
	var decoded Person
	if err := decoded.FromJSON(jsonStr); err != nil {
		t.Fatalf("FromJSON() error: %v", err)
	}

	if decoded.Name != msg.Name || decoded.Age != msg.Age {
		t.Errorf("Person fields mismatch")
	}

	if decoded.Address.City != msg.Address.City {
		t.Errorf("Nested address mismatch")
	}
}

func TestAllZeroValues(t *testing.T) {
	msg := &AllZeroValues{
		ZeroInt:    0,
		ZeroString: "",
		ZeroBool:   false,
		EmptyArray: []string{},
	}

	jsonStr, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	t.Logf("Zero values JSON: %s", jsonStr)

	// Verify all fields are included even with zero values
	expected := `{"zero_int":"0","zero_string":"","zero_bool":false,"empty_array":[]}`
	if jsonStr != expected {
		t.Errorf("Zero values encoding incorrect:\nGot:  %s\nWant: %s", jsonStr, expected)
	}
}

func TestUnknownFieldsRejected(t *testing.T) {
	jsonStr := `{"zero_int":"0","unknown_field":"should_fail"}`
	var msg AllZeroValues
	err := msg.FromJSON(jsonStr)
	if err == nil {
		t.Error("FromJSON() should reject unknown fields")
	}
	t.Logf("Unknown field correctly rejected: %v", err)
}

// Regression test: FromJSON for an optional pointer field used to emit
// `*m.Value = strVal` directly, which panics when m.Value is nil. The
// generator now allocates a local of the element type, populates it,
// and assigns its address. Trace of the prior bug:
//
//	var m OptionalPointer    // m.Value == nil
//	m.FromJSON(`{"value":"hi"}`)  // takes the non-null branch
//	*m.Value = "hi"          // <- nil pointer dereference panic
//
// This test would have caught the panic on the very first call.
func TestOptionalPointer_FromJSON_NonNullValue(t *testing.T) {
	cases := []struct {
		name       string
		json       string
		wantValue  *string
		wantNumber *int64
	}{
		{
			name:      "string set",
			json:      `{"value":"hello"}`,
			wantValue: stringPtr("hello"),
		},
		{
			name:       "number set",
			json:       `{"number":"42"}`,
			wantNumber: int64Ptr(42),
		},
		{
			name:       "both set",
			json:       `{"value":"hello","number":"42"}`,
			wantValue:  stringPtr("hello"),
			wantNumber: int64Ptr(42),
		},
		{
			name: "both null",
			json: `{"value":null,"number":null}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m OptionalPointer
			if err := m.FromJSON(tc.json); err != nil {
				t.Fatalf("FromJSON(%q) error: %v", tc.json, err)
			}
			if (tc.wantValue == nil) != (m.Value == nil) {
				t.Errorf("Value: got %v, want %v", m.Value, tc.wantValue)
			} else if tc.wantValue != nil && *m.Value != *tc.wantValue {
				t.Errorf("Value: got %q, want %q", *m.Value, *tc.wantValue)
			}
			if (tc.wantNumber == nil) != (m.Number == nil) {
				t.Errorf("Number: got %v, want %v", m.Number, tc.wantNumber)
			} else if tc.wantNumber != nil && *m.Number != *tc.wantNumber {
				t.Errorf("Number: got %d, want %d", *m.Number, *tc.wantNumber)
			}
		})
	}
}

func stringPtr(s string) *string { return &s }
func int64Ptr(n int64) *int64    { return &n }
