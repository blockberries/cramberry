package cramberry

import (
	"reflect"
	"strings"
	"testing"
)

// --- Item 11: parseFieldTag ---------------------------------------------

type goodStructForTag struct {
	A int    `cramberry:"3"`
	B string `cramberry:"4,omitempty"`
}

func TestParseFieldTag_AcceptsValidTag(t *testing.T) {
	// Smoke test: marshaling a well-tagged struct must not panic.
	v := &goodStructForTag{A: 1, B: "x"}
	if _, err := Marshal(v); err != nil {
		t.Fatalf("Marshal of valid struct returned error: %v", err)
	}
}

func TestParseFieldTag_ErrorsOnInvalidNumber(t *testing.T) {
	type badNum struct {
		A int `cramberry:"abc"`
	}
	_, err := Marshal(&badNum{A: 1})
	if err == nil {
		t.Fatal("expected error on non-numeric field tag, got none")
	}
	if !strings.Contains(err.Error(), "invalid field number") {
		t.Errorf("error = %v, want substring 'invalid field number'", err)
	}
}

func TestParseFieldTag_ErrorsOnNonPositiveNumber(t *testing.T) {
	type badNum struct {
		A int `cramberry:"-3"`
	}
	_, err := Marshal(&badNum{A: 1})
	if err == nil {
		t.Fatal("expected error on negative field tag, got none")
	}
}

func TestParseFieldTag_ErrorsOnUnknownOption(t *testing.T) {
	type typoOpt struct {
		A int `cramberry:"3,omit_empty"` // user typo'd "omitempty"
	}
	_, err := Marshal(&typoOpt{A: 1})
	if err == nil {
		t.Fatal("expected error on unknown tag option, got none")
	}
	if !strings.Contains(err.Error(), "unknown tag option") {
		t.Errorf("error = %v, want substring 'unknown tag option'", err)
	}
}

// --- Item 13: isZeroValue ignores unexported fields ---------------------

type structWithHiddenState struct {
	Public  int    `cramberry:"1"`
	private string //nolint:unused // we read it via reflection in the test
}

func TestIsZeroValue_IgnoresUnexportedFields(t *testing.T) {
	// A struct whose only non-zero field is unexported must encode-omit
	// under OmitEmpty (because the encoder also skips unexported fields).
	v := structWithHiddenState{private: "non-empty"}
	rv := reflect.ValueOf(v)
	if !isZeroValue(rv) {
		t.Errorf("isZeroValue returned false for struct with only unexported non-zero field; encoder will emit empty body but OmitEmpty must skip it entirely")
	}
}

// --- Item 14: registry refuses silent ID re-allocation ------------------

type regTypeA struct{ X int }
type regTypeB struct{ Y int }

func TestRegisterOrGetTypeWithID_PanicsOnIDCollision(t *testing.T) {
	r := NewRegistry()
	r.RegisterOrGetTypeWithID(reflect.TypeFor[regTypeA](), 200)
	defer func() {
		if rec := recover(); rec == nil {
			t.Fatal("expected panic on ID collision with a different type, got none")
		}
	}()
	// Try to take id=200 with a different type — must panic, not silently swap.
	r.RegisterOrGetTypeWithID(reflect.TypeFor[regTypeB](), 200)
}

func TestRegisterOrGetTypeWithID_ReturnsExistingIDForRepeatRegistration(t *testing.T) {
	r := NewRegistry()
	id1 := r.RegisterOrGetTypeWithID(reflect.TypeFor[regTypeA](), 200)
	id2 := r.RegisterOrGetTypeWithID(reflect.TypeFor[regTypeA](), 999) // same type
	if id1 != id2 {
		t.Errorf("repeat registration returned different IDs: %d vs %d", id1, id2)
	}
	if id1 != 200 {
		t.Errorf("expected id=200, got %d", id1)
	}
}

// --- nil vs empty slice round-trip contract ----------------------------

// TestSliceNilEmptyRoundTrip locks in the documented contract: nil and
// empty slices both encode as `varint(0)` (a length prefix of zero), and
// both decode back to a non-nil empty slice. This is by design — the wire
// format has no way to distinguish nil from empty without a presence
// bitmap, and OmitEmpty treats both as zero. A change to either branch
// would silently flip semantics in user code, so we test it explicitly.
func TestSliceNilEmptyRoundTrip(t *testing.T) {
	type S struct {
		Items []int32 `cramberry:"1"`
	}

	cases := []struct {
		name  string
		items []int32
	}{
		{"nil", nil},
		{"empty", []int32{}},
	}

	// We disable OmitEmpty so the field is always emitted; otherwise both
	// are omitted and we don't observe the wire format at all.
	opts := DefaultOptions
	opts.OmitEmpty = false

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := MarshalWithOptions(&S{Items: tc.items}, opts)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got S
			if err := UnmarshalWithOptions(data, &got, opts); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			// Both nil and empty round-trip to a non-nil zero-length slice.
			if got.Items == nil {
				t.Errorf("decoded slice is nil; expected empty non-nil")
			}
			if len(got.Items) != 0 {
				t.Errorf("decoded slice = %v, want empty", got.Items)
			}
		})
	}
}
