package schema

import (
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
	validator.Validate()
	warnings := validator.Warnings()

	if len(warnings) == 0 {
		t.Error("expected warning about missing zero value")
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
		{"bool key", "map[bool]string", false},
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
