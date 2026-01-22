package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cramberry/cramberry-go/pkg/schema"
)

func TestRustGeneratorSimpleMessage(t *testing.T) {
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

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check struct
	if !strings.Contains(output, "pub struct User {") {
		t.Errorf("expected User struct, got: %s", output)
	}

	// Check fields with Rust types
	if !strings.Contains(output, "pub id: i32,") {
		t.Errorf("expected id field with i32 type, got: %s", output)
	}
	if !strings.Contains(output, "pub name: String,") {
		t.Error("expected name field with String type")
	}
	if !strings.Contains(output, "pub active: bool,") {
		t.Error("expected active field with bool type")
	}

	// Check derive attributes
	if !strings.Contains(output, "#[derive(Debug, Clone, PartialEq, Default)]") {
		t.Error("expected derive attributes")
	}
}

func TestRustGeneratorEnum(t *testing.T) {
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

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check enum
	if !strings.Contains(output, "pub enum Status {") {
		t.Errorf("expected Status enum, got: %s", output)
	}

	// Check repr attribute
	if !strings.Contains(output, "#[repr(i32)]") {
		t.Error("expected repr(i32) attribute")
	}

	// Check enum values
	if !strings.Contains(output, "Unknown = 0,") {
		t.Errorf("expected Unknown value, got: %s", output)
	}
	if !strings.Contains(output, "Active = 1,") {
		t.Error("expected Active value")
	}

	// Check from_i32 method
	if !strings.Contains(output, "pub fn from_i32(value: i32) -> Option<Self>") {
		t.Error("expected from_i32 method")
	}
}

func TestRustGeneratorInterface(t *testing.T) {
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

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check Rust enum for interface
	if !strings.Contains(output, "pub enum Animal {") {
		t.Errorf("expected Animal enum, got: %s", output)
	}

	// Check variants
	if !strings.Contains(output, "Dog(Dog),") {
		t.Error("expected Dog variant")
	}
	if !strings.Contains(output, "Cat(Cat),") {
		t.Error("expected Cat variant")
	}

	// Check type_id method
	if !strings.Contains(output, "pub fn type_id(&self) -> u32") {
		t.Error("expected type_id method")
	}
	if !strings.Contains(output, "Self::Dog(_) => 128,") {
		t.Error("expected Dog type ID")
	}
}

func TestRustGeneratorComplexTypes(t *testing.T) {
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
					{Name: "fixedArray", Number: 6, Type: &schema.ArrayType{Element: &schema.ScalarType{Name: "int32"}, Size: 4}},
				},
			},
		},
	}

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check Vec type
	if !strings.Contains(output, "pub tags: Vec<String>,") {
		t.Errorf("expected Vec<String>, got: %s", output)
	}

	// Check bytes type
	if !strings.Contains(output, "pub data: Vec<u8>,") {
		t.Error("expected Vec<u8> for bytes")
	}

	// Check HashMap type
	if !strings.Contains(output, "pub scores: std::collections::HashMap<String, i32>,") {
		t.Errorf("expected HashMap type, got: %s", output)
	}

	// Check Option<Box<>> type
	if !strings.Contains(output, "pub user: Option<Box<User>>,") {
		t.Errorf("expected Option<Box<User>>, got: %s", output)
	}

	// Check i64 for int64
	if !strings.Contains(output, "pub big_num: i64,") {
		t.Error("expected i64 for int64")
	}

	// Check fixed-size array
	if !strings.Contains(output, "pub fixed_array: [i32; 4],") {
		t.Errorf("expected fixed-size array, got: %s", output)
	}
}

func TestRustGeneratorOptionalFields(t *testing.T) {
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

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Required field should not have Option
	if !strings.Contains(output, "pub id: i32,") {
		t.Error("expected required id without Option")
	}

	// Optional field should have Option
	if !strings.Contains(output, "pub name: Option<String>,") {
		t.Errorf("expected optional name with Option, got: %s", output)
	}
}

func TestRustGeneratorRepeatedFields(t *testing.T) {
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

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Repeated fields should be Vec
	if !strings.Contains(output, "pub tags: Vec<String>,") {
		t.Errorf("expected Vec for tags, got: %s", output)
	}
	if !strings.Contains(output, "pub users: Vec<User>,") {
		t.Error("expected Vec for users")
	}
}

func TestRustGeneratorDocComments(t *testing.T) {
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

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.GenerateComments = true

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check Rust doc comments
	if !strings.Contains(output, "/// User represents a user.") {
		t.Errorf("expected struct doc comment, got: %s", output)
	}
	if !strings.Contains(output, "/// Unique identifier.") {
		t.Error("expected field doc comment")
	}
}

func TestRustGeneratorSerde(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "user_id", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
				},
			},
		},
	}

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.GenerateJSON = true

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check serde import
	if !strings.Contains(output, "use serde::{Deserialize, Serialize};") {
		t.Errorf("expected serde import, got: %s", output)
	}

	// Check serde derive
	if !strings.Contains(output, "#[derive(Serialize, Deserialize)]") {
		t.Error("expected serde derive")
	}

	// Check serde rename
	if !strings.Contains(output, `#[serde(rename = "user_id")]`) {
		t.Errorf("expected serde rename attribute, got: %s", output)
	}
}

func TestRustGeneratorKeywordEscape(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "Test",
				Fields: []*schema.Field{
					{Name: "type", Number: 1, Type: &schema.ScalarType{Name: "string"}},
					{Name: "match", Number: 2, Type: &schema.ScalarType{Name: "int32"}},
					{Name: "async", Number: 3, Type: &schema.ScalarType{Name: "bool"}},
				},
			},
		},
	}

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check escaped keywords
	if !strings.Contains(output, "pub r#type: String,") {
		t.Errorf("expected escaped type field, got: %s", output)
	}
	if !strings.Contains(output, "pub r#match: i32,") {
		t.Error("expected escaped match field")
	}
	if !strings.Contains(output, "pub r#async: bool,") {
		t.Error("expected escaped async field")
	}
}

func TestRustGeneratorRegistered(t *testing.T) {
	gen, ok := Get(LanguageRust)
	if !ok {
		t.Fatal("Rust generator not registered")
	}

	if gen.Language() != LanguageRust {
		t.Errorf("expected Rust language, got %s", gen.Language())
	}

	if gen.FileExtension() != ".rs" {
		t.Errorf("expected .rs extension, got %s", gen.FileExtension())
	}
}

func TestRustGeneratorAllScalarTypes(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "AllTypes",
				Fields: []*schema.Field{
					{Name: "b", Number: 1, Type: &schema.ScalarType{Name: "bool"}},
					{Name: "i8_val", Number: 2, Type: &schema.ScalarType{Name: "int8"}},
					{Name: "i16_val", Number: 3, Type: &schema.ScalarType{Name: "int16"}},
					{Name: "i32_val", Number: 4, Type: &schema.ScalarType{Name: "int32"}},
					{Name: "i64_val", Number: 5, Type: &schema.ScalarType{Name: "int64"}},
					{Name: "u8_val", Number: 6, Type: &schema.ScalarType{Name: "uint8"}},
					{Name: "u16_val", Number: 7, Type: &schema.ScalarType{Name: "uint16"}},
					{Name: "u32_val", Number: 8, Type: &schema.ScalarType{Name: "uint32"}},
					{Name: "u64_val", Number: 9, Type: &schema.ScalarType{Name: "uint64"}},
					{Name: "f32_val", Number: 10, Type: &schema.ScalarType{Name: "float32"}},
					{Name: "f64_val", Number: 11, Type: &schema.ScalarType{Name: "float64"}},
					{Name: "str", Number: 12, Type: &schema.ScalarType{Name: "string"}},
					{Name: "data", Number: 13, Type: &schema.ScalarType{Name: "bytes"}},
					{Name: "byte_val", Number: 14, Type: &schema.ScalarType{Name: "byte"}},
					{Name: "complex64_val", Number: 15, Type: &schema.ScalarType{Name: "complex64"}},
					{Name: "complex128_val", Number: 16, Type: &schema.ScalarType{Name: "complex128"}},
				},
			},
		},
	}

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	expectedTypes := map[string]string{
		"b":              "bool",
		"i8_val":         "i8",
		"i16_val":        "i16",
		"i32_val":        "i32",
		"i64_val":        "i64",
		"u8_val":         "u8",
		"u16_val":        "u16",
		"u32_val":        "u32",
		"u64_val":        "u64",
		"f32_val":        "f32",
		"f64_val":        "f64",
		"str":            "String",
		"data":           "Vec<u8>",
		"byte_val":       "u8",
		"complex64_val":  "(f32, f32)",
		"complex128_val": "(f64, f64)",
	}

	for field, rustType := range expectedTypes {
		expected := "pub " + field + ": " + rustType + ","
		if !strings.Contains(output, expected) {
			t.Errorf("expected %s, got: %s", expected, output)
		}
	}
}

func TestRustGeneratorPackagePrefix(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "address", Number: 1, Type: &schema.NamedType{Name: "Address", Package: "common"}},
				},
			},
		},
	}

	gen := NewRustGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check that package prefix is snake_case for Rust
	if !strings.Contains(output, "pub address: common::Address,") {
		t.Errorf("expected package prefix, got: %s", output)
	}
}
