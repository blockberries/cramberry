package cramberry

import (
	"bytes"
	"testing"
)

// TestForwardCompatibility verifies that older decoders can read data
// encoded with newer schemas that have additional fields.
// This is critical for schema evolution - new fields should be silently
// skipped by older decoders that don't know about them.

// V1 schema types (what the "old" decoder knows about)
type UserV1 struct {
	ID   int32  `cramberry:"1"`
	Name string `cramberry:"2"`
}

type OrderV1 struct {
	OrderID int64   `cramberry:"1"`
	Items   []int32 `cramberry:"2"`
}

type NestedV1 struct {
	User UserV1 `cramberry:"1"`
}

// V2 schema types (what the "new" encoder uses)
type UserV2 struct {
	ID       int32   `cramberry:"1"`
	Name     string  `cramberry:"2"`
	Email    string  `cramberry:"3"` // New field
	Age      int32   `cramberry:"4"` // New field
	IsActive bool    `cramberry:"5"` // New field
	Score    float64 `cramberry:"6"` // New field
}

type NestedV2 struct {
	User      UserV2 `cramberry:"1"`
	Timestamp int64  `cramberry:"2"` // New field
}

func TestForwardCompatBasicTypes(t *testing.T) {
	t.Run("string field added", func(t *testing.T) {
		// Encode with V2 (has email field)
		v2 := UserV2{ID: 42, Name: "Alice", Email: "alice@example.com"}
		data, err := Marshal(v2)
		if err != nil {
			t.Fatalf("Marshal V2 error: %v", err)
		}

		// Decode with V1 (doesn't know about email)
		var v1 UserV1
		err = Unmarshal(data, &v1)
		if err != nil {
			t.Fatalf("Unmarshal to V1 error: %v", err)
		}

		// Known fields should be preserved
		if v1.ID != 42 {
			t.Errorf("ID = %d, want 42", v1.ID)
		}
		if v1.Name != "Alice" {
			t.Errorf("Name = %q, want %q", v1.Name, "Alice")
		}
	})

	t.Run("multiple fields added", func(t *testing.T) {
		// Encode with all V2 fields populated
		v2 := UserV2{
			ID:       123,
			Name:     "Bob",
			Email:    "bob@example.com",
			Age:      30,
			IsActive: true,
			Score:    95.5,
		}
		data, err := Marshal(v2)
		if err != nil {
			t.Fatalf("Marshal V2 error: %v", err)
		}

		// Decode with V1
		var v1 UserV1
		err = Unmarshal(data, &v1)
		if err != nil {
			t.Fatalf("Unmarshal to V1 error: %v", err)
		}

		if v1.ID != 123 {
			t.Errorf("ID = %d, want 123", v1.ID)
		}
		if v1.Name != "Bob" {
			t.Errorf("Name = %q, want %q", v1.Name, "Bob")
		}
	})
}

func TestForwardCompatSlicesAndMaps(t *testing.T) {
	t.Run("scalar fields added after slice", func(t *testing.T) {
		// Simpler test: just scalar fields after a slice
		type OrderV2Simple struct {
			OrderID  int64   `cramberry:"1"`
			Items    []int32 `cramberry:"2"`
			Discount float32 `cramberry:"3"` // New field
			Notes    string  `cramberry:"4"` // New field
		}

		v2 := OrderV2Simple{
			OrderID:  999,
			Items:    []int32{1, 2, 3},
			Discount: 10.5,
			Notes:    "Rush order",
		}
		data, err := Marshal(v2)
		if err != nil {
			t.Fatalf("Marshal V2 error: %v", err)
		}

		var v1 OrderV1
		err = Unmarshal(data, &v1)
		if err != nil {
			t.Fatalf("Unmarshal to V1 error: %v", err)
		}

		if v1.OrderID != 999 {
			t.Errorf("OrderID = %d, want 999", v1.OrderID)
		}
		if len(v1.Items) != 3 || v1.Items[0] != 1 || v1.Items[1] != 2 || v1.Items[2] != 3 {
			t.Errorf("Items = %v, want [1, 2, 3]", v1.Items)
		}
	})

	t.Run("new slice field added", func(t *testing.T) {
		// Test with a new slice field that V1 doesn't know about
		type WithExtraSlice struct {
			ID     int32    `cramberry:"1"`
			Tags   []string `cramberry:"2"` // New slice field
			Values []int32  `cramberry:"3"` // Another new slice field
		}

		type OnlyID struct {
			ID int32 `cramberry:"1"`
		}

		v2 := WithExtraSlice{
			ID:     42,
			Tags:   []string{"a", "b", "c"},
			Values: []int32{100, 200, 300},
		}
		data, err := Marshal(v2)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var v1 OnlyID
		err = Unmarshal(data, &v1)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if v1.ID != 42 {
			t.Errorf("ID = %d, want 42", v1.ID)
		}
	})
}

func TestForwardCompatNestedMessages(t *testing.T) {
	t.Run("nested message with new fields", func(t *testing.T) {
		v2 := NestedV2{
			User: UserV2{
				ID:       1,
				Name:     "Charlie",
				Email:    "charlie@example.com",
				Age:      25,
				IsActive: true,
			},
			Timestamp: 1234567890,
		}
		data, err := Marshal(v2)
		if err != nil {
			t.Fatalf("Marshal V2 error: %v", err)
		}

		var v1 NestedV1
		err = Unmarshal(data, &v1)
		if err != nil {
			t.Fatalf("Unmarshal to V1 error: %v", err)
		}

		if v1.User.ID != 1 {
			t.Errorf("User.ID = %d, want 1", v1.User.ID)
		}
		if v1.User.Name != "Charlie" {
			t.Errorf("User.Name = %q, want %q", v1.User.Name, "Charlie")
		}
	})
}

func TestForwardCompatAllWireTypes(t *testing.T) {
	// Test that all wire types can be skipped correctly

	t.Run("skip varint field", func(t *testing.T) {
		w := NewWriter()
		w.WriteCompactTag(1, WireTypeV2SVarint)
		w.WriteSvarint(42) // Known field
		w.WriteCompactTag(99, WireTypeV2Varint)
		w.WriteUvarint(12345) // Unknown varint
		w.WriteEndMarker()

		type KnownOnly struct {
			A int32 `cramberry:"1"`
		}

		var decoded KnownOnly
		err := Unmarshal(w.Bytes(), &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.A != 42 {
			t.Errorf("A = %d, want 42", decoded.A)
		}
	})

	t.Run("skip fixed32 field", func(t *testing.T) {
		w := NewWriter()
		w.WriteCompactTag(1, WireTypeV2SVarint)
		w.WriteSvarint(42) // Known field
		w.WriteCompactTag(99, WireTypeV2Fixed32)
		w.WriteFixed32(0xDEADBEEF) // Unknown fixed32
		w.WriteEndMarker()

		type KnownOnly struct {
			A int32 `cramberry:"1"`
		}

		var decoded KnownOnly
		err := Unmarshal(w.Bytes(), &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.A != 42 {
			t.Errorf("A = %d, want 42", decoded.A)
		}
	})

	t.Run("skip fixed64 field", func(t *testing.T) {
		w := NewWriter()
		w.WriteCompactTag(1, WireTypeV2SVarint)
		w.WriteSvarint(42) // Known field
		w.WriteCompactTag(99, WireTypeV2Fixed64)
		w.WriteFixed64(0xDEADBEEFCAFEBABE) // Unknown fixed64
		w.WriteEndMarker()

		type KnownOnly struct {
			A int32 `cramberry:"1"`
		}

		var decoded KnownOnly
		err := Unmarshal(w.Bytes(), &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.A != 42 {
			t.Errorf("A = %d, want 42", decoded.A)
		}
	})

	t.Run("skip bytes field", func(t *testing.T) {
		w := NewWriter()
		w.WriteCompactTag(1, WireTypeV2SVarint)
		w.WriteSvarint(42) // Known field
		w.WriteCompactTag(99, WireTypeV2Bytes)
		w.WriteBytes([]byte("unknown data that should be skipped")) // Unknown bytes
		w.WriteEndMarker()

		type KnownOnly struct {
			A int32 `cramberry:"1"`
		}

		var decoded KnownOnly
		err := Unmarshal(w.Bytes(), &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.A != 42 {
			t.Errorf("A = %d, want 42", decoded.A)
		}
	})
}

func TestForwardCompatFieldOrder(t *testing.T) {
	t.Run("unknown fields at start", func(t *testing.T) {
		w := NewWriter()
		// Unknown fields first
		w.WriteCompactTag(50, WireTypeV2Bytes)
		w.WriteString("unknown1")
		w.WriteCompactTag(51, WireTypeV2Varint)
		w.WriteUvarint(999)
		// Then known fields
		w.WriteCompactTag(1, WireTypeV2SVarint)
		w.WriteSvarint(42)
		w.WriteCompactTag(2, WireTypeV2Bytes)
		w.WriteString("hello")
		w.WriteEndMarker()

		var decoded UserV1
		err := Unmarshal(w.Bytes(), &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.ID != 42 || decoded.Name != "hello" {
			t.Errorf("Decoded = %+v, want ID=42 Name=hello", decoded)
		}
	})

	t.Run("unknown fields at end", func(t *testing.T) {
		w := NewWriter()
		// Known fields first
		w.WriteCompactTag(1, WireTypeV2SVarint)
		w.WriteSvarint(42)
		w.WriteCompactTag(2, WireTypeV2Bytes)
		w.WriteString("hello")
		// Then unknown fields
		w.WriteCompactTag(50, WireTypeV2Bytes)
		w.WriteString("unknown1")
		w.WriteCompactTag(51, WireTypeV2Varint)
		w.WriteUvarint(999)
		w.WriteEndMarker()

		var decoded UserV1
		err := Unmarshal(w.Bytes(), &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.ID != 42 || decoded.Name != "hello" {
			t.Errorf("Decoded = %+v, want ID=42 Name=hello", decoded)
		}
	})

	t.Run("unknown fields interleaved", func(t *testing.T) {
		w := NewWriter()
		w.WriteCompactTag(50, WireTypeV2Bytes)
		w.WriteString("unknown before")
		w.WriteCompactTag(1, WireTypeV2SVarint)
		w.WriteSvarint(42)
		w.WriteCompactTag(51, WireTypeV2Varint)
		w.WriteUvarint(999)
		w.WriteCompactTag(2, WireTypeV2Bytes)
		w.WriteString("hello")
		w.WriteCompactTag(52, WireTypeV2Fixed64)
		w.WriteFixed64(123456789)
		w.WriteEndMarker()

		var decoded UserV1
		err := Unmarshal(w.Bytes(), &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.ID != 42 || decoded.Name != "hello" {
			t.Errorf("Decoded = %+v, want ID=42 Name=hello", decoded)
		}
	})
}

func TestForwardCompatStrictModeRejectsUnknown(t *testing.T) {
	// Encode with V2
	v2 := UserV2{ID: 42, Name: "Alice", Email: "alice@example.com"}
	data, err := Marshal(v2)
	if err != nil {
		t.Fatalf("Marshal V2 error: %v", err)
	}

	// Decode with V1 in strict mode - should fail
	var v1 UserV1
	err = UnmarshalWithOptions(data, &v1, StrictOptions)
	if err == nil {
		t.Error("Expected error in strict mode for unknown fields")
	}
}

func TestForwardCompatRoundTrip(t *testing.T) {
	// Verify that unknown fields are truly skipped by checking
	// that re-encoding the V1 data doesn't include the V2 fields

	// Encode with V2
	v2 := UserV2{ID: 42, Name: "Alice", Email: "alice@example.com", Age: 30}
	dataV2, err := Marshal(v2)
	if err != nil {
		t.Fatalf("Marshal V2 error: %v", err)
	}

	// Decode to V1
	var v1 UserV1
	err = Unmarshal(dataV2, &v1)
	if err != nil {
		t.Fatalf("Unmarshal to V1 error: %v", err)
	}

	// Re-encode from V1
	dataV1, err := Marshal(v1)
	if err != nil {
		t.Fatalf("Marshal V1 error: %v", err)
	}

	// V1 encoded data should be smaller (no email/age fields)
	if len(dataV1) >= len(dataV2) {
		t.Errorf("V1 encoded size (%d) should be smaller than V2 (%d)", len(dataV1), len(dataV2))
	}

	// Verify V1 data can still be decoded
	var v1Again UserV1
	err = Unmarshal(dataV1, &v1Again)
	if err != nil {
		t.Fatalf("Unmarshal V1 again error: %v", err)
	}
	if v1Again.ID != 42 || v1Again.Name != "Alice" {
		t.Errorf("Round-trip failed: got %+v", v1Again)
	}
}

func TestForwardCompatEmptyAndZeroValues(t *testing.T) {
	t.Run("empty string unknown field", func(t *testing.T) {
		v2 := UserV2{ID: 1, Name: "Test", Email: ""} // Empty email
		data, err := Marshal(v2)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var v1 UserV1
		err = Unmarshal(data, &v1)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if v1.ID != 1 || v1.Name != "Test" {
			t.Errorf("Decoded = %+v", v1)
		}
	})

	t.Run("zero value unknown fields", func(t *testing.T) {
		v2 := UserV2{ID: 1, Name: "Test", Age: 0, IsActive: false, Score: 0.0}
		data, err := Marshal(v2)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var v1 UserV1
		err = Unmarshal(data, &v1)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if v1.ID != 1 || v1.Name != "Test" {
			t.Errorf("Decoded = %+v", v1)
		}
	})
}

func TestForwardCompatLargeUnknownFields(t *testing.T) {
	// Test that large unknown fields are skipped correctly
	type WithLargeField struct {
		ID   int32  `cramberry:"1"`
		Data []byte `cramberry:"2"`
	}

	largeData := bytes.Repeat([]byte("x"), 10000)
	v2 := WithLargeField{ID: 42, Data: largeData}
	data, err := Marshal(v2)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Decode with type that only knows about ID
	type KnownOnly struct {
		ID int32 `cramberry:"1"`
	}

	var v1 KnownOnly
	err = Unmarshal(data, &v1)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if v1.ID != 42 {
		t.Errorf("ID = %d, want 42", v1.ID)
	}
}
