package schema

import (
	"strings"
	"testing"
)

func TestValidateSimpleMessage(t *testing.T) {
	input := `
package test;

message User {
  int32 id = 1;
  string name = 2;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	errors := Validate(schema)
	for _, err := range errors {
		if err.Severity == SeverityError {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestValidateDuplicateFieldNumber(t *testing.T) {
	input := `
package test;

message User {
  int32 id = 1;
  string name = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()

	if !validator.HasErrors() {
		t.Fatal("expected validation errors")
	}

	found := false
	for _, err := range errors {
		if err.Severity == SeverityError && err.Message != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected duplicate field number error")
	}
}

func TestValidateDuplicateFieldName(t *testing.T) {
	input := `
package test;

message User {
  int32 name = 1;
  string name = 2;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	errors := Validate(schema)
	hasError := false
	for _, err := range errors {
		if err.Severity == SeverityError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected duplicate field name error")
	}
}

func TestValidateZeroFieldNumber(t *testing.T) {
	input := `
package test;

message User {
  int32 id = 0;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	errors := Validate(schema)
	hasError := false
	for _, err := range errors {
		if err.Severity == SeverityError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected zero field number error")
	}
}

func TestValidateNegativeFieldNumber(t *testing.T) {
	input := `
package test;

message User {
  int32 id = -1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	errors := Validate(schema)
	hasError := false
	for _, err := range errors {
		if err.Severity == SeverityError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected negative field number error")
	}
}

func TestValidateDuplicateTypeName(t *testing.T) {
	input := `
package test;

message User {
  int32 id = 1;
}

message User {
  string name = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	errors := Validate(schema)
	hasError := false
	for _, err := range errors {
		if err.Severity == SeverityError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected duplicate type name error")
	}
}

func TestValidateUndefinedType(t *testing.T) {
	input := `
package test;

message User {
  Address address = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	errors := Validate(schema)
	hasError := false
	for _, err := range errors {
		if err.Severity == SeverityError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected undefined type error")
	}
}

func TestValidateValidTypeReference(t *testing.T) {
	input := `
package test;

message Address {
  string street = 1;
}

message User {
  Address address = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()
	if validator.HasErrors() {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestValidateEnum(t *testing.T) {
	input := `
package test;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()
	if validator.HasErrors() {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestValidateEnumDuplicateNumber(t *testing.T) {
	input := `
package test;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected duplicate enum value error")
	}
}

func TestValidateEnumDuplicateName(t *testing.T) {
	input := `
package test;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  ACTIVE = 2;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected duplicate enum name error")
	}
}

func TestValidateEnumMissingZero(t *testing.T) {
	input := `
package test;

enum Status {
  ACTIVE = 1;
  INACTIVE = 2;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errs := validator.Validate()

	found := false
	for _, e := range errs {
		if e.Severity == SeverityError && strings.Contains(e.Message, "zero value") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about missing zero value; got %v", errs)
	}
}

func TestValidateInterface(t *testing.T) {
	input := `
package test;

message Dog {
  string name = 1;
}

message Cat {
  string name = 1;
}

interface Animal {
  128 = Dog;
  129 = Cat;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()
	if validator.HasErrors() {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestValidateInterfaceDuplicateTypeID(t *testing.T) {
	input := `
package test;

message Dog {
  string name = 1;
}

message Cat {
  string name = 1;
}

interface Animal {
  128 = Dog;
  128 = Cat;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected duplicate type ID error")
	}
}

func TestValidateInterfaceUndefinedType(t *testing.T) {
	input := `
package test;

interface Animal {
  128 = Dog;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected undefined type error")
	}
}

func TestValidateInterfaceReferenceEnum(t *testing.T) {
	input := `
package test;

enum Status {
  UNKNOWN = 0;
}

interface Animal {
  128 = Status;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected error: interface should reference message, not enum")
	}
}

func TestValidateMapKeyType(t *testing.T) {
	tests := []struct {
		name      string
		keyType   string
		expectErr bool
	}{
		{"string key", "map[string]int32", false},
		{"int32 key", "map[int32]string", false},
		{"bool key", "map[bool]string", true},       // bool keys break codegen
		{"bytes key", "map[bytes]string", true},     // bytes not comparable
		{"float32 key", "map[float32]string", true}, // floats not comparable
		{"float64 key", "map[float64]string", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `
package test;

message Test {
  ` + tt.keyType + ` data = 1;
}
`
			schema, parseErrors := ParseFile("test.cram", input)
			if len(parseErrors) > 0 {
				t.Fatalf("parse errors: %v", parseErrors)
			}

			validator := NewValidator(schema)
			errors := validator.Validate()

			if tt.expectErr && !validator.HasErrors() {
				t.Errorf("expected error for %s", tt.keyType)
			}
			if !tt.expectErr && validator.HasErrors() {
				t.Errorf("unexpected error for %s: %v", tt.keyType, errors)
			}
		})
	}
}

func TestValidateModifierCombinations(t *testing.T) {
	input := `
package test;

message Test {
  required optional int32 x = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected error for conflicting modifiers")
	}
}

func TestValidateWithImports(t *testing.T) {
	// Main schema
	mainInput := `
package main;

import "other.cram" as other;

message User {
  other.Address address = 1;
}
`

	// Imported schema
	otherInput := `
package other;

message Address {
  string street = 1;
}
`

	mainSchema, parseErrors := ParseFile("main.cram", mainInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	otherSchema, parseErrors := ParseFile("other.cram", otherInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(mainSchema)
	validator.AddImport("other.cram", "other", otherSchema)
	errors := validator.Validate()

	if validator.HasErrors() {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestValidateSamePackageImport(t *testing.T) {
	// Main schema imports another schema from the same package
	// Types from same-package imports can be used without qualification
	mainInput := `
package myproject;

import "types.cram";

message User {
  Address address = 1;
}
`

	// Imported schema with same package name
	typesInput := `
package myproject;

message Address {
  string street = 1;
}
`

	mainSchema, parseErrors := ParseFile("main.cram", mainInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	typesSchema, parseErrors := ParseFile("types.cram", typesInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(mainSchema)
	validator.AddImport("types.cram", "types.cram", typesSchema)
	errors := validator.Validate()

	if validator.HasErrors() {
		t.Errorf("unexpected errors (same-package types should be accessible): %v", errors)
	}
}

func TestValidateUnknownPackage(t *testing.T) {
	input := `
package test;

message User {
  unknown.Address address = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected unknown package error")
	}

	found := false
	for _, err := range errors {
		if err.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Error("expected error about unknown package")
	}
}

func TestValidateReservedFieldNumber(t *testing.T) {
	input := `
package test;

message Test {
  int32 x = 19500;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	warnings := validator.Warnings()

	if len(warnings) == 0 {
		t.Error("expected warning about reserved field number")
	}
}

func TestValidateMaxFieldNumber(t *testing.T) {
	input := `
package test;

message Test {
  int32 x = 600000000;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected error for field number exceeding maximum")
	}
}

func TestValidationErrorString(t *testing.T) {
	err := ValidationError{
		Position: Position{
			Filename: "test.cram",
			Line:     10,
			Column:   5,
		},
		Message:  "test error",
		Severity: SeverityError,
	}

	s := err.Error()
	expected := "test.cram:10:5: error: test error"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		str      string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
	}

	for _, tt := range tests {
		if tt.severity.String() != tt.str {
			t.Errorf("expected %q, got %q", tt.str, tt.severity.String())
		}
	}
}

func TestTypeDefKindString(t *testing.T) {
	tests := []struct {
		kind TypeDefKind
		str  string
	}{
		{TypeDefMessage, "message"},
		{TypeDefEnum, "enum"},
		{TypeDefInterface, "interface"},
	}

	for _, tt := range tests {
		if tt.kind.String() != tt.str {
			t.Errorf("expected %q, got %q", tt.str, tt.kind.String())
		}
	}
}

func TestValidateComplexSchema(t *testing.T) {
	input := `
package example;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}

message Address {
  string street = 1;
  string city = 2;
  string zip = 3;
}

message User @128 {
  required int32 id = 1;
  optional string name = 2;
  repeated string tags = 3;
  Status status = 4;
  *Address address = 5;
  map[string]int32 scores = 6;
}

message Admin @129 {
  User user = 1;
  repeated string permissions = 2;
}

interface Person {
  128 = User;
  129 = Admin;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()
	if validator.HasErrors() {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestValidateNestedArrays(t *testing.T) {
	input := `
package test;

message Test {
  [][]int32 matrix = 1;
  [][][]string cube = 2;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()
	if validator.HasErrors() {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestValidatePointerToUndefined(t *testing.T) {
	input := `
package test;

message Test {
  *Unknown ptr = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	validator.Validate()
	if !validator.HasErrors() {
		t.Error("expected error for pointer to undefined type")
	}
}

func TestValidateEnumAsFieldType(t *testing.T) {
	input := `
package test;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
}

message Test {
  Status status = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()
	if validator.HasErrors() {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestValidateEnumAsMapKey(t *testing.T) {
	input := `
package test;

enum Direction {
  UNKNOWN = 0;
  NORTH = 1;
  SOUTH = 2;
}

message Test {
  map[Direction]string labels = 1;
}
`

	schema, parseErrors := ParseFile("test.cram", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	validator := NewValidator(schema)
	errors := validator.Validate()
	if validator.HasErrors() {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestValidateInterfaceWithMultipleSamePackageImports(t *testing.T) {
	// This tests that an interface can reference types from multiple
	// imported files that share the same package name.

	// Main schema with interface
	animalsInput := `
package animals;

import "dog.cram";
import "cat.cram";

interface Animal {
  128 = Dog;
  129 = Cat;
}
`

	// First imported schema
	dogInput := `
package animals;

message Dog {
  string name = 1;
}
`

	// Second imported schema
	catInput := `
package animals;

message Cat {
  string name = 1;
}
`

	animalsSchema, parseErrors := ParseFile("animals.cram", animalsInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors for animals.cram: %v", parseErrors)
	}

	dogSchema, parseErrors := ParseFile("dog.cram", dogInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors for dog.cram: %v", parseErrors)
	}

	catSchema, parseErrors := ParseFile("cat.cram", catInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors for cat.cram: %v", parseErrors)
	}

	validator := NewValidator(animalsSchema)
	validator.AddImport("dog.cram", "dog.cram", dogSchema)
	validator.AddImport("cat.cram", "cat.cram", catSchema)
	errors := validator.Validate()

	if validator.HasErrors() {
		t.Errorf("unexpected errors (interface should be able to reference types from multiple same-package imports): %v", errors)
	}
}

func TestValidateInterfaceWithMixedLocalAndImportedTypes(t *testing.T) {
	// Test that interface can mix local types with same-package imported types

	mainInput := `
package myproject;

import "external.cram";

message LocalType {
  string name = 1;
}

interface Entity {
  128 = LocalType;
  129 = ImportedType;
}
`

	externalInput := `
package myproject;

message ImportedType {
  int32 id = 1;
}
`

	mainSchema, parseErrors := ParseFile("main.cram", mainInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors for main.cram: %v", parseErrors)
	}

	externalSchema, parseErrors := ParseFile("external.cram", externalInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors for external.cram: %v", parseErrors)
	}

	validator := NewValidator(mainSchema)
	validator.AddImport("external.cram", "external.cram", externalSchema)
	errors := validator.Validate()

	if validator.HasErrors() {
		t.Errorf("unexpected errors (interface should work with mixed local and imported types): %v", errors)
	}
}

func TestValidateInterfaceRejectsDifferentPackageUnqualified(t *testing.T) {
	// Types from different packages MUST be qualified

	mainInput := `
package main;

import "other.cram";

interface Entity {
  128 = OtherType;
}
`

	otherInput := `
package other;

message OtherType {
  string name = 1;
}
`

	mainSchema, parseErrors := ParseFile("main.cram", mainInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors for main.cram: %v", parseErrors)
	}

	otherSchema, parseErrors := ParseFile("other.cram", otherInput)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors for other.cram: %v", parseErrors)
	}

	validator := NewValidator(mainSchema)
	validator.AddImport("other.cram", "other.cram", otherSchema)
	validator.Validate()

	if !validator.HasErrors() {
		t.Error("expected error: unqualified type from different package should be rejected")
	}
}

func TestValidator_RejectsDirectSelfRecursion(t *testing.T) {
	s := &Schema{
		Messages: []*Message{
			{
				Name: "Node",
				Fields: []*Field{
					{Name: "parent", Number: 1, Type: &NamedType{Name: "Node"}},
				},
			},
		},
	}
	v := NewValidator(s)
	errs := v.Validate()
	if len(errs) == 0 {
		t.Fatal("expected an error for self-recursive message; got none")
	}
}

func TestValidator_RejectsIndirectRecursion(t *testing.T) {
	s := &Schema{
		Messages: []*Message{
			{Name: "A", Fields: []*Field{{Name: "b", Number: 1, Type: &NamedType{Name: "B"}}}},
			{Name: "B", Fields: []*Field{{Name: "a", Number: 1, Type: &NamedType{Name: "A"}}}},
		},
	}
	v := NewValidator(s)
	errs := v.Validate()
	if len(errs) == 0 {
		t.Fatal("expected an error for A→B→A cycle; got none")
	}
}

func TestValidator_AllowsRecursionThroughPointer(t *testing.T) {
	s := &Schema{
		Messages: []*Message{
			{
				Name: "Tree",
				Fields: []*Field{
					{Name: "left", Number: 1, Type: &PointerType{Element: &NamedType{Name: "Tree"}}},
				},
			},
		},
	}
	v := NewValidator(s)
	errs := v.Validate()
	for _, e := range errs {
		if e.Severity == SeverityError {
			t.Errorf("recursion through pointer should be allowed; got error: %s", e.Message)
		}
	}
}

func TestValidator_AllowsRecursionThroughRepeatedField(t *testing.T) {
	s := &Schema{
		Messages: []*Message{
			{
				Name: "Tree",
				Fields: []*Field{
					{Name: "children", Number: 1, Type: &NamedType{Name: "Tree"}, Repeated: true},
				},
			},
		},
	}
	v := NewValidator(s)
	errs := v.Validate()
	for _, e := range errs {
		if e.Severity == SeverityError {
			t.Errorf("recursion through repeated field should be allowed; got error: %s", e.Message)
		}
	}
}

func TestValidator_RejectsStackedModifiers(t *testing.T) {
	cases := []struct {
		name  string
		field *Field
	}{
		{
			name:  "required+repeated",
			field: &Field{Name: "x", Number: 1, Type: &ScalarType{Name: "int32"}, Required: true, Repeated: true},
		},
		{
			name:  "required+optional",
			field: &Field{Name: "x", Number: 1, Type: &ScalarType{Name: "int32"}, Required: true, Optional: true},
		},
		{
			name:  "optional+repeated",
			field: &Field{Name: "x", Number: 1, Type: &ScalarType{Name: "int32"}, Optional: true, Repeated: true},
		},
		{
			name:  "all three",
			field: &Field{Name: "x", Number: 1, Type: &ScalarType{Name: "int32"}, Required: true, Optional: true, Repeated: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Schema{
				Messages: []*Message{
					{Name: "M", Fields: []*Field{tc.field}},
				},
			}
			v := NewValidator(s)
			errs := v.Validate()
			found := false
			for _, e := range errs {
				if e.Severity == SeverityError && strings.Contains(e.Message, "mutually exclusive") {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected mutually-exclusive-modifier error; got %v", errs)
			}
		})
	}
}

// TestValidator_RejectsReservedTypeID covers the runtime-reserved range:
// type IDs 1-127 are owned by the runtime (1-63 builtin, 64-127 stdlib),
// so user schemas that pick a reserved ID must fail validation rather
// than collide silently with another runtime registration.
func TestValidator_RejectsReservedTypeID(t *testing.T) {
	cases := []struct {
		name   string
		schema string
	}{
		{
			name: "message in builtin range",
			schema: `
package test;
message Foo @5 {
  string x = 1;
}`,
		},
		{
			name: "message in stdlib range",
			schema: `
package test;
message Foo @100 {
  string x = 1;
}`,
		},
		{
			name: "message at upper boundary",
			schema: `
package test;
message Foo @127 {
  string x = 1;
}`,
		},
		{
			name: "interface implementation in reserved range",
			schema: `
package test;
message M { string x = 1; }
interface I {
  5 = M;
}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, parseErrs := ParseFile("test.cram", tc.schema)
			if len(parseErrs) > 0 {
				t.Fatalf("parse errors: %v", parseErrs)
			}
			v := NewValidator(s)
			errs := v.Validate()
			found := false
			for _, e := range errs {
				if e.Severity == SeverityError && strings.Contains(e.Message, "reserved range") {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected reserved-range error, got: %v", errs)
			}
		})
	}
}

func TestValidator_AcceptsTypeIDAtUserBoundary(t *testing.T) {
	input := `
package test;
message Foo @128 {
  string x = 1;
}`
	s, parseErrs := ParseFile("test.cram", input)
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	v := NewValidator(s)
	errs := v.Validate()
	for _, e := range errs {
		if e.Severity == SeverityError {
			t.Errorf("type ID 128 should validate cleanly; got %v", e)
		}
	}
}
