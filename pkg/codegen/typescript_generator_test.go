package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/blockberries/cramberry/pkg/schema"
)

func TestTypeScriptGeneratorSimpleMessage(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
					{Name: "name", Number: 2, Type: &schema.ScalarType{Name: "string"}},
					{Name: "active", Number: 3, Type: &schema.ScalarType{Name: "bool"}},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check interface
	if !strings.Contains(output, "export interface User {") {
		t.Error("expected User interface")
	}

	// Check fields with TypeScript types
	if !strings.Contains(output, "id: number;") {
		t.Errorf("expected id field with number type, got: %s", output)
	}
	if !strings.Contains(output, "name: string;") {
		t.Error("expected name field with string type")
	}
	if !strings.Contains(output, "active: boolean;") {
		t.Error("expected active field with boolean type")
	}
}

func TestTypeScriptGeneratorEnum(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Enums: []*schema.Enum{
			{
				Name: "Status",
				Values: []*schema.EnumValue{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ACTIVE", Number: 1},
					{Name: "INACTIVE", Number: 2},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check enum
	if !strings.Contains(output, "export enum Status {") {
		t.Error("expected Status enum")
	}

	// Check enum values
	if !strings.Contains(output, "Unknown = 0,") {
		t.Errorf("expected Unknown value, got: %s", output)
	}
	if !strings.Contains(output, "Active = 1,") {
		t.Error("expected Active value")
	}
}

func TestTypeScriptGeneratorInterface(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{Name: "Dog", Fields: []*schema.Field{{Name: "name", Number: 1, Type: &schema.ScalarType{Name: "string"}}}},
			{Name: "Cat", Fields: []*schema.Field{{Name: "name", Number: 1, Type: &schema.ScalarType{Name: "string"}}}},
		},
		Interfaces: []*schema.Interface{
			{
				Name: "Animal",
				Implementations: []*schema.Implementation{
					{TypeID: 128, Type: &schema.NamedType{Name: "Dog"}},
					{TypeID: 129, Type: &schema.NamedType{Name: "Cat"}},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check union type
	if !strings.Contains(output, "export type Animal = Dog | Cat;") {
		t.Errorf("expected Animal union type, got: %s", output)
	}

	// Check type ID mapping
	if !strings.Contains(output, "export const AnimalTypeIds = {") {
		t.Error("expected AnimalTypeIds constant")
	}
	if !strings.Contains(output, "Dog: 128,") {
		t.Error("expected Dog type ID")
	}
}

func TestTypeScriptGeneratorComplexTypes(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "Complex",
				Fields: []*schema.Field{
					{Name: "tags", Number: 1, Type: &schema.ArrayType{Element: &schema.ScalarType{Name: "string"}}},
					{Name: "data", Number: 2, Type: &schema.ScalarType{Name: "bytes"}},
					{Name: "scores", Number: 3, Type: &schema.MapType{Key: &schema.ScalarType{Name: "string"}, Value: &schema.ScalarType{Name: "int32"}}},
					{Name: "user", Number: 4, Type: &schema.PointerType{Element: &schema.NamedType{Name: "User"}}},
					{Name: "bigNum", Number: 5, Type: &schema.ScalarType{Name: "int64"}},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check array type
	if !strings.Contains(output, "tags: string[];") {
		t.Errorf("expected string array, got: %s", output)
	}

	// Check bytes type
	if !strings.Contains(output, "data: Uint8Array;") {
		t.Error("expected Uint8Array for bytes")
	}

	// Check map type
	if !strings.Contains(output, "scores: Record<string, number>;") {
		t.Errorf("expected Record type for map, got: %s", output)
	}

	// Check nullable type
	if !strings.Contains(output, "user: User | null;") {
		t.Errorf("expected nullable User, got: %s", output)
	}

	// Check bigint for int64
	if !strings.Contains(output, "bigNum: bigint;") {
		t.Error("expected bigint for int64")
	}
}

func TestTypeScriptGeneratorOptionalFields(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "Request",
				Fields: []*schema.Field{
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}, Required: true},
					{Name: "name", Number: 2, Type: &schema.ScalarType{Name: "string"}, Optional: true},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Required field should not have ?
	if !strings.Contains(output, "id: number;") {
		t.Error("expected required id without ?")
	}

	// Optional field should have ?
	if !strings.Contains(output, "name?: string;") {
		t.Errorf("expected optional name with ?, got: %s", output)
	}
}

func TestTypeScriptGeneratorRepeatedFields(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "Request",
				Fields: []*schema.Field{
					{Name: "tags", Number: 1, Type: &schema.ScalarType{Name: "string"}, Repeated: true},
					{Name: "users", Number: 2, Type: &schema.NamedType{Name: "User"}, Repeated: true},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Repeated fields should be arrays
	if !strings.Contains(output, "tags: string[];") {
		t.Errorf("expected string array for tags, got: %s", output)
	}
	if !strings.Contains(output, "users: User[];") {
		t.Error("expected User array for users")
	}
}

func TestTypeScriptGeneratorDocComments(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Comments: []*schema.Comment{
					{Text: "User represents a user.", IsDoc: true},
				},
				Fields: []*schema.Field{
					{
						Name:   "id",
						Number: 1,
						Type:   &schema.ScalarType{Name: "int32"},
						Comments: []*schema.Comment{
							{Text: "Unique identifier.", IsDoc: true},
						},
					},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.GenerateComments = true

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check JSDoc comments
	if !strings.Contains(output, "/** User represents a user. */") {
		t.Errorf("expected message JSDoc comment, got: %s", output)
	}
	if !strings.Contains(output, "/** Unique identifier. */") {
		t.Error("expected field JSDoc comment")
	}
}

func TestTypeScriptGeneratorRegistered(t *testing.T) {
	gen, ok := Get(LanguageTypeScript)
	if !ok {
		t.Fatal("TypeScript generator not registered")
	}

	if gen.Language() != LanguageTypeScript {
		t.Errorf("expected TypeScript language, got %s", gen.Language())
	}

	if gen.FileExtension() != ".ts" {
		t.Errorf("expected .ts extension, got %s", gen.FileExtension())
	}
}

func TestTypeScriptMapWithNonStringKey(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "Test",
				Fields: []*schema.Field{
					{Name: "intMap", Number: 1, Type: &schema.MapType{Key: &schema.ScalarType{Name: "int32"}, Value: &schema.ScalarType{Name: "string"}}},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Non-string keys should use Map instead of Record
	if !strings.Contains(output, "intMap: Map<number, string>;") {
		t.Errorf("expected Map for non-string key, got: %s", output)
	}
}
