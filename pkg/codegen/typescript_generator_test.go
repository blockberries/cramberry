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

	// Check tagged-union type. The earlier assertion expected
	// `Animal = Dog | Cat` — but that bare-union has no runtime
	// discriminator, so the encoder couldn't dispatch correctly.
	// The generator now emits a tagged union with a `kind` field.
	if !strings.Contains(output, "kind: 'Dog'") || !strings.Contains(output, "kind: 'Cat'") {
		t.Errorf("expected tagged-union variants for Animal, got: %s", output)
	}
	if !strings.Contains(output, "value: Dog") || !strings.Contains(output, "value: Cat") {
		t.Errorf("expected tagged-union value field for Animal, got: %s", output)
	}

	// Check type ID mapping
	if !strings.Contains(output, "export const AnimalTypeIds = {") {
		t.Error("expected AnimalTypeIds constant")
	}
	if !strings.Contains(output, "Dog: 128,") {
		t.Error("expected Dog type ID")
	}

	// Check polymorphic encode/decode helpers exist.
	if !strings.Contains(output, "export function encodeAnimal(") {
		t.Errorf("expected encodeAnimal function, got: %s", output)
	}
	if !strings.Contains(output, "export function decodeAnimal(") {
		t.Errorf("expected decodeAnimal function, got: %s", output)
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

	// Check map type. We emit `Map<K, V>` for all map fields, even
	// string-keyed ones, because the runtime decoder produces a Map and
	// typing the field as `Record` caused tsc strict-mode errors.
	if !strings.Contains(output, "scores: Map<string, number>;") {
		t.Errorf("expected Map type for map, got: %s", output)
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

// Regression test: non-optional NamedType / MapType / ArrayType fields
// must emit unconditionally in TypeScript, matching the Go and Rust
// generators byte-for-byte. Earlier the TS template wrapped every field
// in `if (presence)`, silently skipping non-optional composite fields
// when a type-violating caller passed `undefined` — diverging from
// Go/Rust which always emit.
func TestTypeScriptGenerator_NonOptionalComposite_EmitsUnconditionally(t *testing.T) {
	s := &schema.Schema{
		Messages: []*schema.Message{
			{
				Name: "Inner",
				Fields: []*schema.Field{
					{Name: "x", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
				},
			},
			{
				Name: "Outer",
				Fields: []*schema.Field{
					// Non-optional nested message — must always emit.
					{Name: "inner", Number: 1, Type: &schema.NamedType{Name: "Inner"}},
					// Non-optional map — must always emit.
					{Name: "tags", Number: 2, Type: &schema.MapType{
						Key:   &schema.ScalarType{Name: "string"},
						Value: &schema.ScalarType{Name: "string"},
					}},
				},
			},
		},
	}

	gen := NewTypeScriptGenerator()
	var buf bytes.Buffer
	if err := gen.Generate(&buf, s, DefaultOptions()); err != nil {
		t.Fatalf("generate error: %v", err)
	}
	output := buf.String()

	// The non-optional `inner` field must NOT be wrapped in a presence
	// check. The exact pattern of an unconditional emit is the writeTag
	// call appearing on a line with no preceding `if` on the same field.
	if strings.Contains(output, "if (msg.inner !== undefined && msg.inner !== null)") {
		t.Errorf("non-optional NamedType field 'inner' must emit unconditionally; output wraps it in presence check:\n%s", output)
	}
	if strings.Contains(output, "if (msg.tags !== undefined && msg.tags !== null)") {
		t.Errorf("non-optional MapType field 'tags' must emit unconditionally; output wraps it in presence check:\n%s", output)
	}
	// Sanity: the unconditional writeTag MUST still appear for both fields.
	if !strings.Contains(output, "writer.writeTag(1, WireType.Bytes)") {
		t.Errorf("expected writeTag for inner field; got:\n%s", output)
	}
	if !strings.Contains(output, "writer.writeTag(2, WireType.Bytes)") {
		t.Errorf("expected writeTag for tags field; got:\n%s", output)
	}
}
