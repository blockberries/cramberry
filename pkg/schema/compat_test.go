package schema

import (
	"testing"
)

func TestCheckCompatibility_NoChanges(t *testing.T) {
	schema := &Schema{
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &NamedType{Name: "int64"}},
					{Name: "name", Number: 2, Type: &NamedType{Name: "string"}},
				},
			},
		},
	}

	report := CheckCompatibility(schema, schema)
	if !report.IsCompatible() {
		t.Errorf("identical schemas should be compatible, got %d breaking changes", len(report.Breaking))
	}
}

func TestCheckCompatibility_FieldTypeChanged(t *testing.T) {
	old := &Schema{
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &NamedType{Name: "int32"}},
				},
			},
		},
	}

	new := &Schema{
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &NamedType{Name: "string"}}, // Changed type
				},
			},
		},
	}

	report := CheckCompatibility(old, new)
	if report.IsCompatible() {
		t.Error("field type change should be breaking")
	}

	found := false
	for _, b := range report.Breaking {
		if b.Type == FieldTypeChanged {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected FieldTypeChanged breaking change")
	}
}

func TestCheckCompatibility_RequiredFieldAdded(t *testing.T) {
	old := &Schema{
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &NamedType{Name: "int64"}},
				},
			},
		},
	}

	new := &Schema{
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &NamedType{Name: "int64"}},
					{Name: "email", Number: 2, Type: &NamedType{Name: "string"}, Required: true}, // New required
				},
			},
		},
	}

	report := CheckCompatibility(old, new)
	if report.IsCompatible() {
		t.Error("adding required field should be breaking")
	}

	found := false
	for _, b := range report.Breaking {
		if b.Type == RequiredFieldAdded {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected RequiredFieldAdded breaking change")
	}
}

func TestCheckCompatibility_RequiredFieldRemoved(t *testing.T) {
	old := &Schema{
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &NamedType{Name: "int64"}, Required: true},
				},
			},
		},
	}

	new := &Schema{
		Messages: []*Message{
			{
				Name:   "User",
				Fields: []*Field{}, // Removed required field
			},
		},
	}

	report := CheckCompatibility(old, new)
	if report.IsCompatible() {
		t.Error("removing required field should be breaking")
	}

	found := false
	for _, b := range report.Breaking {
		if b.Type == RequiredFieldRemoved {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected RequiredFieldRemoved breaking change")
	}
}

func TestCheckCompatibility_MessageRemoved(t *testing.T) {
	old := &Schema{
		Messages: []*Message{
			{Name: "User"},
		},
	}

	new := &Schema{
		Messages: []*Message{}, // User removed
	}

	report := CheckCompatibility(old, new)
	if report.IsCompatible() {
		t.Error("removing message should be breaking")
	}

	found := false
	for _, b := range report.Breaking {
		if b.Type == MessageRemoved {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MessageRemoved breaking change")
	}
}

func TestCheckCompatibility_EnumValueRemoved(t *testing.T) {
	old := &Schema{
		Enums: []*Enum{
			{
				Name: "Status",
				Values: []*EnumValue{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ACTIVE", Number: 1},
					{Name: "INACTIVE", Number: 2},
				},
			},
		},
	}

	new := &Schema{
		Enums: []*Enum{
			{
				Name: "Status",
				Values: []*EnumValue{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ACTIVE", Number: 1},
					// INACTIVE removed
				},
			},
		},
	}

	report := CheckCompatibility(old, new)
	if report.IsCompatible() {
		t.Error("removing enum value should be breaking")
	}

	found := false
	for _, b := range report.Breaking {
		if b.Type == EnumValueRemoved {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected EnumValueRemoved breaking change")
	}
}

func TestCheckCompatibility_EnumValueReused(t *testing.T) {
	old := &Schema{
		Enums: []*Enum{
			{
				Name: "Status",
				Values: []*EnumValue{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ACTIVE", Number: 1},
				},
			},
		},
	}

	new := &Schema{
		Enums: []*Enum{
			{
				Name: "Status",
				Values: []*EnumValue{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ENABLED", Number: 1}, // Reused number 1 with different name
				},
			},
		},
	}

	report := CheckCompatibility(old, new)
	if report.IsCompatible() {
		t.Error("reusing enum value number should be breaking")
	}

	found := false
	for _, b := range report.Breaking {
		if b.Type == EnumValueReused {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected EnumValueReused breaking change")
	}
}

func TestCheckCompatibility_InterfaceTypeRemoved(t *testing.T) {
	old := &Schema{
		Interfaces: []*Interface{
			{
				Name: "Animal",
				Implementations: []*Implementation{
					{TypeID: 128, Type: &NamedType{Name: "Dog"}},
					{TypeID: 129, Type: &NamedType{Name: "Cat"}},
				},
			},
		},
	}

	new := &Schema{
		Interfaces: []*Interface{
			{
				Name: "Animal",
				Implementations: []*Implementation{
					{TypeID: 128, Type: &NamedType{Name: "Dog"}},
					// Cat removed
				},
			},
		},
	}

	report := CheckCompatibility(old, new)
	if report.IsCompatible() {
		t.Error("removing interface type should be breaking")
	}

	found := false
	for _, b := range report.Breaking {
		if b.Type == InterfaceTypeRemoved {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected InterfaceTypeRemoved breaking change")
	}
}

func TestCheckCompatibility_OptionalFieldAdded(t *testing.T) {
	old := &Schema{
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &NamedType{Name: "int64"}},
				},
			},
		},
	}

	new := &Schema{
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &NamedType{Name: "int64"}},
					{Name: "email", Number: 2, Type: &NamedType{Name: "string"}, Required: false}, // Optional
				},
			},
		},
	}

	report := CheckCompatibility(old, new)
	if !report.IsCompatible() {
		t.Errorf("adding optional field should be compatible, got %d breaking changes", len(report.Breaking))
	}
}

func TestCheckCompatibility_IntWidening(t *testing.T) {
	old := &Schema{
		Messages: []*Message{
			{
				Name: "Counter",
				Fields: []*Field{
					{Name: "value", Number: 1, Type: &NamedType{Name: "int32"}},
				},
			},
		},
	}

	new := &Schema{
		Messages: []*Message{
			{
				Name: "Counter",
				Fields: []*Field{
					{Name: "value", Number: 1, Type: &NamedType{Name: "int64"}}, // Widened
				},
			},
		},
	}

	report := CheckCompatibility(old, new)
	if !report.IsCompatible() {
		t.Errorf("int32->int64 should be compatible, got breaking changes: %v", report.Breaking)
	}
}

func TestBreakingChangeType_String(t *testing.T) {
	tests := []struct {
		changeType BreakingChangeType
		expected   string
	}{
		{FieldNumberReused, "field number reused"},
		{FieldTypeChanged, "field type changed"},
		{RequiredFieldAdded, "required field added"},
		{RequiredFieldRemoved, "required field removed"},
		{EnumValueReused, "enum value number reused"},
		{EnumValueRemoved, "enum value removed"},
		{MessageRemoved, "message removed"},
		{EnumRemoved, "enum removed"},
		{InterfaceTypeRemoved, "interface type removed"},
		{InterfaceTypeIDReused, "interface type ID reused"},
	}

	for _, tc := range tests {
		if got := tc.changeType.String(); got != tc.expected {
			t.Errorf("%d.String() = %q, want %q", tc.changeType, got, tc.expected)
		}
	}
}

func TestBreakingChange_Error(t *testing.T) {
	b := BreakingChange{
		Type:     FieldTypeChanged,
		Message:  "field 'id' type changed from int32 to string",
		Location: "User.id",
	}

	got := b.Error()
	if got == "" {
		t.Error("Error() should not return empty string")
	}
}
