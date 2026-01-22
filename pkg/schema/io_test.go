package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriterSimpleMessage(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{
						Name:   "id",
						Number: 1,
						Type:   &ScalarType{Name: "int32"},
					},
					{
						Name:   "name",
						Number: 2,
						Type:   &ScalarType{Name: "string"},
					},
				},
			},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, "package test;") {
		t.Error("expected package declaration")
	}
	if !strings.Contains(output, "message User {") {
		t.Error("expected message declaration")
	}
	if !strings.Contains(output, "int32 id = 1;") {
		t.Error("expected id field")
	}
	if !strings.Contains(output, "string name = 2;") {
		t.Error("expected name field")
	}
}

func TestWriterWithModifiers(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Messages: []*Message{
			{
				Name: "Request",
				Fields: []*Field{
					{
						Name:     "id",
						Number:   1,
						Type:     &ScalarType{Name: "int32"},
						Required: true,
					},
					{
						Name:     "name",
						Number:   2,
						Type:     &ScalarType{Name: "string"},
						Optional: true,
					},
					{
						Name:     "tags",
						Number:   3,
						Type:     &ArrayType{Element: &ScalarType{Name: "string"}},
						Repeated: true,
					},
					{
						Name:       "old_field",
						Number:     4,
						Type:       &ScalarType{Name: "bool"},
						Deprecated: true,
					},
				},
			},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, "required int32 id = 1;") {
		t.Errorf("expected required modifier, got: %s", output)
	}
	if !strings.Contains(output, "optional string name = 2;") {
		t.Errorf("expected optional modifier, got: %s", output)
	}
	if !strings.Contains(output, "repeated []string tags = 3;") {
		t.Errorf("expected repeated modifier, got: %s", output)
	}
	if !strings.Contains(output, "deprecated bool old_field = 4;") {
		t.Errorf("expected deprecated modifier, got: %s", output)
	}
}

func TestWriterWithTypeID(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Messages: []*Message{
			{
				Name:   "User",
				TypeID: 128,
				Fields: []*Field{
					{
						Name:   "id",
						Number: 1,
						Type:   &ScalarType{Name: "int32"},
					},
				},
			},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, "message User @128 {") {
		t.Errorf("expected message with type ID, got: %s", output)
	}
}

func TestWriterEnum(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
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

	output := FormatSchema(schema)

	if !strings.Contains(output, "enum Status {") {
		t.Error("expected enum declaration")
	}
	if !strings.Contains(output, "UNKNOWN = 0;") {
		t.Error("expected UNKNOWN value")
	}
	if !strings.Contains(output, "ACTIVE = 1;") {
		t.Error("expected ACTIVE value")
	}
}

func TestWriterInterface(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
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

	output := FormatSchema(schema)

	if !strings.Contains(output, "interface Animal {") {
		t.Error("expected interface declaration")
	}
	if !strings.Contains(output, "128 = Dog;") {
		t.Error("expected Dog implementation")
	}
	if !strings.Contains(output, "129 = Cat;") {
		t.Error("expected Cat implementation")
	}
}

func TestWriterImports(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Imports: []*Import{
			{Path: "other.cramberry"},
			{Path: "types.cramberry", Alias: "types"},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, `import "other.cramberry";`) {
		t.Error("expected import without alias")
	}
	if !strings.Contains(output, `import "types.cramberry" as types;`) {
		t.Error("expected import with alias")
	}
}

func TestWriterOptions(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Options: []*Option{
			{Name: "go_package", Value: &StringValue{Value: "github.com/test"}},
			{Name: "optimize", Value: &BoolValue{Value: true}},
			{Name: "max_size", Value: &NumberValue{Value: "1024"}},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, `option go_package = "github.com/test";`) {
		t.Error("expected string option")
	}
	if !strings.Contains(output, "option optimize = true;") {
		t.Error("expected bool option")
	}
	if !strings.Contains(output, "option max_size = 1024;") {
		t.Error("expected number option")
	}
}

func TestWriterFieldOptions(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{
						Name:   "name",
						Number: 1,
						Type:   &ScalarType{Name: "string"},
						Options: []*Option{
							{Name: "max_length", Value: &NumberValue{Value: "100"}},
							{Name: "min_length", Value: &NumberValue{Value: "1"}},
						},
					},
				},
			},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, "[max_length = 100, min_length = 1]") {
		t.Errorf("expected field options, got: %s", output)
	}
}

func TestWriterComplexTypes(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Messages: []*Message{
			{
				Name: "Complex",
				Fields: []*Field{
					{
						Name:   "list",
						Number: 1,
						Type:   &ArrayType{Element: &ScalarType{Name: "string"}},
					},
					{
						Name:   "fixed",
						Number: 2,
						Type:   &ArrayType{Element: &ScalarType{Name: "byte"}, Size: 32},
					},
					{
						Name: "map_field",
						Number: 3,
						Type: &MapType{
							Key:   &ScalarType{Name: "string"},
							Value: &ScalarType{Name: "int32"},
						},
					},
					{
						Name:   "ptr",
						Number: 4,
						Type:   &PointerType{Element: &NamedType{Name: "User"}},
					},
					{
						Name:   "external",
						Number: 5,
						Type:   &NamedType{Package: "other", Name: "Type"},
					},
				},
			},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, "[]string list = 1;") {
		t.Errorf("expected slice type, got: %s", output)
	}
	if !strings.Contains(output, "[32]byte fixed = 2;") {
		t.Errorf("expected fixed array type, got: %s", output)
	}
	if !strings.Contains(output, "map[string]int32 map_field = 3;") {
		t.Errorf("expected map type, got: %s", output)
	}
	if !strings.Contains(output, "*User ptr = 4;") {
		t.Errorf("expected pointer type, got: %s", output)
	}
	if !strings.Contains(output, "other.Type external = 5;") {
		t.Errorf("expected qualified type, got: %s", output)
	}
}

func TestWriterDocComments(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Messages: []*Message{
			{
				Name: "User",
				Comments: []*Comment{
					{Text: "User represents a user.", IsDoc: true},
				},
				Fields: []*Field{
					{
						Name:   "id",
						Number: 1,
						Type:   &ScalarType{Name: "int32"},
						Comments: []*Comment{
							{Text: "Unique identifier.", IsDoc: true},
						},
					},
				},
			},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, "/// User represents a user.") {
		t.Errorf("expected message doc comment, got: %s", output)
	}
	if !strings.Contains(output, "/// Unique identifier.") {
		t.Errorf("expected field doc comment, got: %s", output)
	}
}

func TestWriterListValue(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Options: []*Option{
			{
				Name: "tags",
				Value: &ListValue{
					Values: []Value{
						&StringValue{Value: "a"},
						&StringValue{Value: "b"},
					},
				},
			},
		},
	}

	output := FormatSchema(schema)

	if !strings.Contains(output, `option tags = ["a", "b"];`) {
		t.Errorf("expected list value, got: %s", output)
	}
}

func TestRoundTrip(t *testing.T) {
	input := `package example;

import "other.cramberry";
import "types.cramberry" as types;

option go_package = "github.com/example";

/// User message.
message User @128 {
  required int32 id = 1;
  optional string name = 2;
  repeated string tags = 3;
}

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
}

interface Shape {
  130 = Circle;
  131 = Rectangle;
}
`

	// Parse
	schema, parseErrors := ParseFile("test.cramberry", input)
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	// Format
	output := FormatSchema(schema)

	// Parse again
	schema2, parseErrors2 := ParseFile("test.cramberry", output)
	if len(parseErrors2) > 0 {
		t.Fatalf("second parse errors: %v", parseErrors2)
	}

	// Compare
	if schema.Package.Name != schema2.Package.Name {
		t.Errorf("package mismatch: %s vs %s", schema.Package.Name, schema2.Package.Name)
	}
	if len(schema.Imports) != len(schema2.Imports) {
		t.Errorf("import count mismatch: %d vs %d", len(schema.Imports), len(schema2.Imports))
	}
	if len(schema.Messages) != len(schema2.Messages) {
		t.Errorf("message count mismatch: %d vs %d", len(schema.Messages), len(schema2.Messages))
	}
	if len(schema.Enums) != len(schema2.Enums) {
		t.Errorf("enum count mismatch: %d vs %d", len(schema.Enums), len(schema2.Enums))
	}
	if len(schema.Interfaces) != len(schema2.Interfaces) {
		t.Errorf("interface count mismatch: %d vs %d", len(schema.Interfaces), len(schema2.Interfaces))
	}
}

func TestLoaderSimpleFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Write test schema
	schemaContent := `
package test;

message User {
  int32 id = 1;
}
`
	schemaPath := filepath.Join(tmpDir, "test.cramberry")
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load
	loader := NewLoader()
	schema, errors := loader.LoadFile(schemaPath)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if schema.Package == nil || schema.Package.Name != "test" {
		t.Error("expected package 'test'")
	}
	if len(schema.Messages) != 1 || schema.Messages[0].Name != "User" {
		t.Error("expected message 'User'")
	}
}

func TestLoaderWithImports(t *testing.T) {
	tmpDir := t.TempDir()

	// Write types schema
	typesContent := `
package types;

message Address {
  string street = 1;
}
`
	typesPath := filepath.Join(tmpDir, "types.cramberry")
	if err := os.WriteFile(typesPath, []byte(typesContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Write main schema
	mainContent := `
package main;

import "types.cramberry" as types;

message User {
  int32 id = 1;
  types.Address address = 2;
}
`
	mainPath := filepath.Join(tmpDir, "main.cramberry")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load
	loader := NewLoader()
	schema, errors := loader.LoadFile(mainPath)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if schema.Package.Name != "main" {
		t.Errorf("expected package 'main', got %q", schema.Package.Name)
	}

	// Check that types.cramberry was also loaded
	allSchemas := loader.AllSchemas()
	if len(allSchemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(allSchemas))
	}
}

func TestLoaderMissingImport(t *testing.T) {
	tmpDir := t.TempDir()

	mainContent := `
package main;

import "missing.cramberry";

message User {
  int32 id = 1;
}
`
	mainPath := filepath.Join(tmpDir, "main.cramberry")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	_, errors := loader.LoadFile(mainPath)
	if len(errors) == 0 {
		t.Error("expected error for missing import")
	}
}

func TestLoaderCircularImport(t *testing.T) {
	tmpDir := t.TempDir()

	// a.cramberry imports b.cramberry
	aContent := `
package a;
import "b.cramberry";
message A { int32 x = 1; }
`
	aPath := filepath.Join(tmpDir, "a.cramberry")
	if err := os.WriteFile(aPath, []byte(aContent), 0644); err != nil {
		t.Fatal(err)
	}

	// b.cramberry imports a.cramberry
	bContent := `
package b;
import "a.cramberry";
message B { int32 y = 1; }
`
	bPath := filepath.Join(tmpDir, "b.cramberry")
	if err := os.WriteFile(bPath, []byte(bContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	_, errors := loader.LoadFile(aPath)
	if len(errors) == 0 {
		t.Error("expected circular import error")
	}

	foundCircular := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "circular import") {
			foundCircular = true
			break
		}
	}
	if !foundCircular {
		t.Error("expected circular import error message")
	}
}

func TestLoaderSearchPaths(t *testing.T) {
	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write library schema in lib/
	libContent := `
package lib;
message Common { int32 x = 1; }
`
	libPath := filepath.Join(libDir, "common.cramberry")
	if err := os.WriteFile(libPath, []byte(libContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Write main schema in tmpDir
	mainContent := `
package main;
import "common.cramberry" as lib;
message User { lib.Common common = 1; }
`
	mainPath := filepath.Join(tmpDir, "main.cramberry")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load with search path
	loader := NewLoader(libDir)
	schema, errors := loader.LoadFile(mainPath)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if schema.Package.Name != "main" {
		t.Errorf("expected package 'main', got %q", schema.Package.Name)
	}
}

func TestWriteToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.cramberry")

	schema := &Schema{
		Package: &Package{Name: "output"},
		Messages: []*Message{
			{
				Name: "Test",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &ScalarType{Name: "int32"}},
				},
			},
		},
	}

	if err := WriteToFile(outPath, schema); err != nil {
		t.Fatal(err)
	}

	// Read back
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "package output;") {
		t.Error("expected package in output file")
	}
	if !strings.Contains(string(content), "message Test {") {
		t.Error("expected message in output file")
	}
}

func TestWriterSetIndent(t *testing.T) {
	schema := &Schema{
		Package: &Package{Name: "test"},
		Messages: []*Message{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "id", Number: 1, Type: &ScalarType{Name: "int32"}},
				},
			},
		},
	}

	writer := NewWriter()
	writer.SetIndent("\t")

	var sb strings.Builder
	writer.WriteSchema(&sb, schema)
	output := sb.String()

	if !strings.Contains(output, "\tint32 id = 1;") {
		t.Errorf("expected tab indent, got: %s", output)
	}
}

func TestLoadAndValidate(t *testing.T) {
	tmpDir := t.TempDir()

	content := `
package test;

message User {
  int32 id = 1;
  string name = 2;
}
`
	path := filepath.Join(tmpDir, "test.cramberry")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	schema, errors := LoadAndValidate(path)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if schema == nil {
		t.Fatal("expected schema")
	}
	if schema.Package.Name != "test" {
		t.Errorf("expected package 'test', got %q", schema.Package.Name)
	}
}
