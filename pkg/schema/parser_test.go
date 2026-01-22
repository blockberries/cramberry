package schema

import (
	"testing"
)

func TestParsePackage(t *testing.T) {
	input := `package example;`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if schema.Package == nil {
		t.Fatal("expected package declaration")
	}
	if schema.Package.Name != "example" {
		t.Errorf("expected package name 'example', got %q", schema.Package.Name)
	}
}

func TestParseImport(t *testing.T) {
	input := `
package test;
import "other.cramberry";
import "another.cramberry" as another;
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(schema.Imports))
	}

	if schema.Imports[0].Path != "other.cramberry" {
		t.Errorf("expected import path 'other.cramberry', got %q", schema.Imports[0].Path)
	}
	if schema.Imports[0].Alias != "" {
		t.Errorf("expected no alias, got %q", schema.Imports[0].Alias)
	}

	if schema.Imports[1].Path != "another.cramberry" {
		t.Errorf("expected import path 'another.cramberry', got %q", schema.Imports[1].Path)
	}
	if schema.Imports[1].Alias != "another" {
		t.Errorf("expected alias 'another', got %q", schema.Imports[1].Alias)
	}
}

func TestParseOption(t *testing.T) {
	input := `
package test;
option go_package = "github.com/example/test";
option optimize_for = true;
option max_size = 1024;
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Options) != 3 {
		t.Fatalf("expected 3 options, got %d", len(schema.Options))
	}

	// String option
	if schema.Options[0].Name != "go_package" {
		t.Errorf("expected option name 'go_package', got %q", schema.Options[0].Name)
	}
	if sv, ok := schema.Options[0].Value.(*StringValue); !ok || sv.Value != "github.com/example/test" {
		t.Errorf("expected string value 'github.com/example/test', got %v", schema.Options[0].Value)
	}

	// Bool option
	if schema.Options[1].Name != "optimize_for" {
		t.Errorf("expected option name 'optimize_for', got %q", schema.Options[1].Name)
	}
	if bv, ok := schema.Options[1].Value.(*BoolValue); !ok || !bv.Value {
		t.Errorf("expected bool value true, got %v", schema.Options[1].Value)
	}

	// Number option
	if schema.Options[2].Name != "max_size" {
		t.Errorf("expected option name 'max_size', got %q", schema.Options[2].Name)
	}
	if nv, ok := schema.Options[2].Value.(*NumberValue); !ok || nv.Value != "1024" {
		t.Errorf("expected number value '1024', got %v", schema.Options[2].Value)
	}
}

func TestParseSimpleMessage(t *testing.T) {
	input := `
package test;

message User {
  int32 id = 1;
  string name = 2;
  bool active = 3;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(schema.Messages))
	}

	msg := schema.Messages[0]
	if msg.Name != "User" {
		t.Errorf("expected message name 'User', got %q", msg.Name)
	}

	if len(msg.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(msg.Fields))
	}

	// Check first field
	f := msg.Fields[0]
	if f.Name != "id" || f.Number != 1 {
		t.Errorf("expected field 'id = 1', got '%s = %d'", f.Name, f.Number)
	}
	if st, ok := f.Type.(*ScalarType); !ok || st.Name != "int32" {
		t.Errorf("expected type 'int32', got %v", f.Type)
	}
}

func TestParseMessageWithModifiers(t *testing.T) {
	input := `
package test;

message Request {
  required int32 id = 1;
  optional string name = 2;
  repeated string tags = 3;
  deprecated bool old_field = 4;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	msg := schema.Messages[0]
	if len(msg.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(msg.Fields))
	}

	if !msg.Fields[0].Required {
		t.Error("expected field 0 to be required")
	}
	if !msg.Fields[1].Optional {
		t.Error("expected field 1 to be optional")
	}
	if !msg.Fields[2].Repeated {
		t.Error("expected field 2 to be repeated")
	}
	if !msg.Fields[3].Deprecated {
		t.Error("expected field 3 to be deprecated")
	}
}

func TestParseMessageWithComplexTypes(t *testing.T) {
	input := `
package test;

message Complex {
  []string names = 1;
  [5]byte data = 2;
  map[string]int32 scores = 3;
  *User user = 4;
  other.Package external = 5;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	msg := schema.Messages[0]
	if len(msg.Fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(msg.Fields))
	}

	// Slice type
	if at, ok := msg.Fields[0].Type.(*ArrayType); !ok || at.Size != 0 {
		t.Errorf("expected []string, got %v", msg.Fields[0].Type)
	}

	// Fixed array
	if at, ok := msg.Fields[1].Type.(*ArrayType); !ok || at.Size != 5 {
		t.Errorf("expected [5]byte, got %v", msg.Fields[1].Type)
	}

	// Map type
	if mt, ok := msg.Fields[2].Type.(*MapType); !ok {
		t.Errorf("expected map type, got %v", msg.Fields[2].Type)
	} else {
		if kt, ok := mt.Key.(*ScalarType); !ok || kt.Name != "string" {
			t.Errorf("expected map key 'string', got %v", mt.Key)
		}
		if vt, ok := mt.Value.(*ScalarType); !ok || vt.Name != "int32" {
			t.Errorf("expected map value 'int32', got %v", mt.Value)
		}
	}

	// Pointer type
	if pt, ok := msg.Fields[3].Type.(*PointerType); !ok {
		t.Errorf("expected pointer type, got %v", msg.Fields[3].Type)
	} else {
		if nt, ok := pt.Element.(*NamedType); !ok || nt.Name != "User" {
			t.Errorf("expected *User, got %v", pt.Element)
		}
	}

	// Qualified type
	if nt, ok := msg.Fields[4].Type.(*NamedType); !ok || nt.Package != "other" || nt.Name != "Package" {
		t.Errorf("expected other.Package, got %v", msg.Fields[4].Type)
	}
}

func TestParseMessageWithTypeID(t *testing.T) {
	input := `
package test;

message User @128 {
  int32 id = 1;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	msg := schema.Messages[0]
	if msg.TypeID != 128 {
		t.Errorf("expected type ID 128, got %d", msg.TypeID)
	}
}

func TestParseMessageWithFieldOptions(t *testing.T) {
	input := `
package test;

message User {
  string name = 1 [max_length = 100, min_length = 1];
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	field := schema.Messages[0].Fields[0]
	if len(field.Options) != 2 {
		t.Fatalf("expected 2 field options, got %d", len(field.Options))
	}

	if field.Options[0].Name != "max_length" {
		t.Errorf("expected option 'max_length', got %q", field.Options[0].Name)
	}
	if field.Options[1].Name != "min_length" {
		t.Errorf("expected option 'min_length', got %q", field.Options[1].Name)
	}
}

func TestParseEnum(t *testing.T) {
	input := `
package test;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
  DELETED = 3;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}

	enum := schema.Enums[0]
	if enum.Name != "Status" {
		t.Errorf("expected enum name 'Status', got %q", enum.Name)
	}

	if len(enum.Values) != 4 {
		t.Fatalf("expected 4 enum values, got %d", len(enum.Values))
	}

	expected := []struct {
		name   string
		number int
	}{
		{"UNKNOWN", 0},
		{"ACTIVE", 1},
		{"INACTIVE", 2},
		{"DELETED", 3},
	}

	for i, exp := range expected {
		if enum.Values[i].Name != exp.name {
			t.Errorf("value %d: expected name %q, got %q", i, exp.name, enum.Values[i].Name)
		}
		if enum.Values[i].Number != exp.number {
			t.Errorf("value %d: expected number %d, got %d", i, exp.number, enum.Values[i].Number)
		}
	}
}

func TestParseInterface(t *testing.T) {
	input := `
package test;

interface Animal {
  128 = Dog;
  129 = Cat;
  130 = other.Bird;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(schema.Interfaces))
	}

	iface := schema.Interfaces[0]
	if iface.Name != "Animal" {
		t.Errorf("expected interface name 'Animal', got %q", iface.Name)
	}

	if len(iface.Implementations) != 3 {
		t.Fatalf("expected 3 implementations, got %d", len(iface.Implementations))
	}

	// Check first implementation
	impl := iface.Implementations[0]
	if impl.TypeID != 128 {
		t.Errorf("expected type ID 128, got %d", impl.TypeID)
	}
	if impl.Type.Name != "Dog" {
		t.Errorf("expected type name 'Dog', got %q", impl.Type.Name)
	}

	// Check qualified implementation
	impl3 := iface.Implementations[2]
	if impl3.Type.Package != "other" || impl3.Type.Name != "Bird" {
		t.Errorf("expected 'other.Bird', got '%s.%s'", impl3.Type.Package, impl3.Type.Name)
	}
}

func TestParseCompleteSchema(t *testing.T) {
	input := `
// Complete schema example
package example;

import "common.cramberry";
import "types.cramberry" as types;

option go_package = "github.com/example/test";

/// User represents a user in the system.
message User @128 {
  required int32 id = 1;
  optional string name = 2;
  repeated string tags = 3;
  types.Address address = 4;
}

/// Address represents a physical address.
message Address @129 {
  string street = 1;
  string city = 2;
  string country = 3;
}

enum UserStatus {
  UNKNOWN = 0;
  ACTIVE = 1;
  SUSPENDED = 2;
}

interface Shape {
  130 = Circle;
  131 = Rectangle;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Verify package
	if schema.Package == nil || schema.Package.Name != "example" {
		t.Error("expected package 'example'")
	}

	// Verify imports
	if len(schema.Imports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(schema.Imports))
	}

	// Verify options
	if len(schema.Options) != 1 {
		t.Errorf("expected 1 option, got %d", len(schema.Options))
	}

	// Verify messages
	if len(schema.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(schema.Messages))
	}

	// Verify enums
	if len(schema.Enums) != 1 {
		t.Errorf("expected 1 enum, got %d", len(schema.Enums))
	}

	// Verify interfaces
	if len(schema.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(schema.Interfaces))
	}
}

func TestParseListOption(t *testing.T) {
	input := `
package test;
option allowed_values = [1, 2, 3];
option string_list = ["a", "b", "c"];
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(schema.Options))
	}

	// Check number list
	lv, ok := schema.Options[0].Value.(*ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", schema.Options[0].Value)
	}
	if len(lv.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(lv.Values))
	}

	// Check string list
	lv2, ok := schema.Options[1].Value.(*ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", schema.Options[1].Value)
	}
	if len(lv2.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(lv2.Values))
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing semicolon after package",
			input: `package test`,
		},
		{
			name:  "missing message name",
			input: `message { }`,
		},
		{
			name:  "missing field number",
			input: `message Foo { int32 x = ; }`,
		},
		{
			name:  "invalid type",
			input: `message Foo { 123 x = 1; }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := ParseFile("test.cramberry", tt.input)
			if len(errors) == 0 {
				t.Error("expected parse errors, got none")
			}
		})
	}
}

func TestTypeRefString(t *testing.T) {
	tests := []struct {
		typeRef TypeRef
		str     string
	}{
		{
			&ScalarType{Name: "int32"},
			"int32",
		},
		{
			&NamedType{Name: "User"},
			"User",
		},
		{
			&NamedType{Package: "other", Name: "User"},
			"other.User",
		},
		{
			&ArrayType{Element: &ScalarType{Name: "string"}},
			"[]string",
		},
		{
			&ArrayType{Element: &ScalarType{Name: "byte"}, Size: 5},
			"[5]byte",
		},
		{
			&MapType{
				Key:   &ScalarType{Name: "string"},
				Value: &ScalarType{Name: "int32"},
			},
			"map[string]int32",
		},
		{
			&PointerType{Element: &NamedType{Name: "User"}},
			"*User",
		},
	}

	for _, tt := range tests {
		result := tt.typeRef.String()
		if result != tt.str {
			t.Errorf("expected %q, got %q", tt.str, result)
		}
	}
}

func TestParseNestedTypes(t *testing.T) {
	input := `
package test;

message Outer {
  [][]int32 matrix = 1;
  map[string][]User users_by_group = 2;
  *[]string optional_list = 3;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	msg := schema.Messages[0]

	// [][]int32
	f0 := msg.Fields[0]
	at, ok := f0.Type.(*ArrayType)
	if !ok {
		t.Fatalf("expected ArrayType, got %T", f0.Type)
	}
	inner, ok := at.Element.(*ArrayType)
	if !ok {
		t.Fatalf("expected inner ArrayType, got %T", at.Element)
	}
	if st, ok := inner.Element.(*ScalarType); !ok || st.Name != "int32" {
		t.Errorf("expected int32, got %v", inner.Element)
	}

	// map[string][]User
	f1 := msg.Fields[1]
	mt, ok := f1.Type.(*MapType)
	if !ok {
		t.Fatalf("expected MapType, got %T", f1.Type)
	}
	valArray, ok := mt.Value.(*ArrayType)
	if !ok {
		t.Fatalf("expected ArrayType value, got %T", mt.Value)
	}
	if nt, ok := valArray.Element.(*NamedType); !ok || nt.Name != "User" {
		t.Errorf("expected User, got %v", valArray.Element)
	}

	// *[]string
	f2 := msg.Fields[2]
	pt, ok := f2.Type.(*PointerType)
	if !ok {
		t.Fatalf("expected PointerType, got %T", f2.Type)
	}
	elemArray, ok := pt.Element.(*ArrayType)
	if !ok {
		t.Fatalf("expected ArrayType element, got %T", pt.Element)
	}
	if st, ok := elemArray.Element.(*ScalarType); !ok || st.Name != "string" {
		t.Errorf("expected string, got %v", elemArray.Element)
	}
}

func TestParseAllScalarTypes(t *testing.T) {
	input := `
package test;

message AllScalars {
  bool b = 1;
  int8 i8 = 2;
  int16 i16 = 3;
  int32 i32 = 4;
  int64 i64 = 5;
  int i = 6;
  uint8 u8 = 7;
  uint16 u16 = 8;
  uint32 u32 = 9;
  uint64 u64 = 10;
  uint u = 11;
  float32 f32 = 12;
  float64 f64 = 13;
  complex64 c64 = 14;
  complex128 c128 = 15;
  string s = 16;
  bytes bs = 17;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	msg := schema.Messages[0]
	if len(msg.Fields) != 17 {
		t.Fatalf("expected 17 fields, got %d", len(msg.Fields))
	}

	expectedTypes := []string{
		"bool", "int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint",
		"float32", "float64", "complex64", "complex128",
		"string", "bytes",
	}

	for i, exp := range expectedTypes {
		st, ok := msg.Fields[i].Type.(*ScalarType)
		if !ok {
			t.Errorf("field %d: expected ScalarType, got %T", i, msg.Fields[i].Type)
			continue
		}
		if st.Name != exp {
			t.Errorf("field %d: expected type %q, got %q", i, exp, st.Name)
		}
	}
}

func TestParseEmptyMessage(t *testing.T) {
	input := `
package test;

message Empty {
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(schema.Messages))
	}

	if len(schema.Messages[0].Fields) != 0 {
		t.Errorf("expected 0 fields, got %d", len(schema.Messages[0].Fields))
	}
}

func TestParseEmptyEnum(t *testing.T) {
	input := `
package test;

enum Empty {
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}

	if len(schema.Enums[0].Values) != 0 {
		t.Errorf("expected 0 values, got %d", len(schema.Enums[0].Values))
	}
}

func TestParseEmptyInterface(t *testing.T) {
	input := `
package test;

interface Empty {
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(schema.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(schema.Interfaces))
	}

	if len(schema.Interfaces[0].Implementations) != 0 {
		t.Errorf("expected 0 implementations, got %d", len(schema.Interfaces[0].Implementations))
	}
}

func TestParseErrorRecovery(t *testing.T) {
	// The parser should recover from errors and continue parsing
	input := `
package test;

message Good1 {
  int32 x = 1;
}

message Bad {
  invalid syntax here
}

message Good2 {
  string y = 1;
}
`

	schema, errors := ParseFile("test.cramberry", input)

	// Should have errors
	if len(errors) == 0 {
		t.Error("expected parse errors")
	}

	// Should still parse at least one message (Good1)
	if len(schema.Messages) == 0 {
		t.Error("expected at least one message to be parsed")
	}
}

func TestParsePosition(t *testing.T) {
	input := `package test;

message User {
  int32 id = 1;
}`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Check message position
	msg := schema.Messages[0]
	if msg.Position.Line != 3 {
		t.Errorf("expected message at line 3, got %d", msg.Position.Line)
	}

	// Check field position
	field := msg.Fields[0]
	if field.Position.Line != 4 {
		t.Errorf("expected field at line 4, got %d", field.Position.Line)
	}
}

func TestParseDocComments(t *testing.T) {
	input := `
package test;

/// This is a doc comment for User.
/// It can span multiple lines.
message User {
  /// The user's unique identifier.
  int32 id = 1;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	msg := schema.Messages[0]
	if len(msg.Comments) == 0 {
		t.Error("expected doc comments on message")
	}
}

func TestParseMultipleCombinedModifiers(t *testing.T) {
	input := `
package test;

message Request {
  required repeated deprecated string values = 1;
}
`

	schema, errors := ParseFile("test.cramberry", input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	field := schema.Messages[0].Fields[0]
	if !field.Required {
		t.Error("expected required modifier")
	}
	if !field.Repeated {
		t.Error("expected repeated modifier")
	}
	if !field.Deprecated {
		t.Error("expected deprecated modifier")
	}
}
