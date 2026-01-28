package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/blockberries/cramberry/pkg/schema"
)

func TestGoGeneratorSimpleMessage(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
					{Name: "name", Number: 2, Type: &schema.ScalarType{Name: "string"}},
				},
			},
		},
	}

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check package
	if !strings.Contains(output, "package test") {
		t.Error("expected package declaration")
	}

	// Check struct
	if !strings.Contains(output, "type User struct") {
		t.Error("expected User struct")
	}

	// Check fields
	if !strings.Contains(output, "Id int32") {
		t.Error("expected Id field")
	}
	if !strings.Contains(output, "Name string") {
		t.Error("expected Name field")
	}

	// Check tags
	if !strings.Contains(output, `cramberry:"1"`) {
		t.Error("expected cramberry tag for id")
	}
	if !strings.Contains(output, `json:"id"`) {
		t.Error("expected json tag for id")
	}
}

func TestGoGeneratorEnum(t *testing.T) {
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

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check enum type
	if !strings.Contains(output, "type Status int32") {
		t.Error("expected Status type")
	}

	// Check enum values
	if !strings.Contains(output, "StatusUnknown Status = 0") {
		t.Errorf("expected StatusUnknown, got: %s", output)
	}
	if !strings.Contains(output, "StatusActive") {
		t.Error("expected StatusActive")
	}

	// Check String method
	if !strings.Contains(output, "func (e Status) String() string") {
		t.Error("expected String method")
	}
}

func TestGoGeneratorInterface(t *testing.T) {
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

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check interface type
	if !strings.Contains(output, "type Animal interface") {
		t.Error("expected Animal interface")
	}

	// Check marker methods
	if !strings.Contains(output, "func (*Dog) isAnimal()") {
		t.Error("expected isAnimal for Dog")
	}
	if !strings.Contains(output, "func (*Cat) isAnimal()") {
		t.Error("expected isAnimal for Cat")
	}

	// Check TypeID function
	if !strings.Contains(output, "func AnimalTypeID(v Animal) cramberry.TypeID") {
		t.Error("expected AnimalTypeID function")
	}
}

func TestGoGeneratorModifiers(t *testing.T) {
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

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check required tag
	if !strings.Contains(output, `cramberry:"1,required"`) {
		t.Error("expected required tag")
	}

	// Check required scalar generates pointer (so we can distinguish nil from zero value)
	if !strings.Contains(output, "Id *int32") {
		t.Errorf("expected pointer for required scalar field, got: %s", output)
	}

	// Check optional generates pointer
	if !strings.Contains(output, "Name *string") {
		t.Errorf("expected pointer for optional field, got: %s", output)
	}

	// Check Validate method uses nil check
	if !strings.Contains(output, "func (m *Request) Validate() error") {
		t.Error("expected Validate method")
	}
	if !strings.Contains(output, "if m.Id == nil") {
		t.Errorf("expected nil check for required field validation, got: %s", output)
	}
}

func TestGoGeneratorComplexTypes(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "Complex",
				Fields: []*schema.Field{
					{Name: "tags", Number: 1, Type: &schema.ArrayType{Element: &schema.ScalarType{Name: "string"}}},
					{Name: "data", Number: 2, Type: &schema.ArrayType{Element: &schema.ScalarType{Name: "byte"}, Size: 32}},
					{Name: "scores", Number: 3, Type: &schema.MapType{Key: &schema.ScalarType{Name: "string"}, Value: &schema.ScalarType{Name: "int32"}}},
					{Name: "user", Number: 4, Type: &schema.PointerType{Element: &schema.NamedType{Name: "User"}}},
				},
			},
		},
	}

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check slice type
	if !strings.Contains(output, "Tags []string") {
		t.Error("expected slice type")
	}

	// Check fixed array
	if !strings.Contains(output, "Data [32]uint8") {
		t.Errorf("expected fixed array, got: %s", output)
	}

	// Check map
	if !strings.Contains(output, "Scores map[string]int32") {
		t.Error("expected map type")
	}

	// Check pointer
	if !strings.Contains(output, "User *User") {
		t.Error("expected pointer type")
	}
}

func TestGoGeneratorOptions(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
				},
			},
		},
	}

	t.Run("custom package", func(t *testing.T) {
		gen := NewGoGenerator()
		var buf bytes.Buffer
		opts := DefaultOptions()
		opts.Package = "mypackage"

		err := gen.Generate(&buf, s, opts)
		if err != nil {
			t.Fatalf("generate error: %v", err)
		}

		if !strings.Contains(buf.String(), "package mypackage") {
			t.Error("expected custom package name")
		}
	})

	t.Run("type prefix", func(t *testing.T) {
		gen := NewGoGenerator()
		var buf bytes.Buffer
		opts := DefaultOptions()
		opts.TypePrefix = "CB"

		err := gen.Generate(&buf, s, opts)
		if err != nil {
			t.Fatalf("generate error: %v", err)
		}

		if !strings.Contains(buf.String(), "type CBUser struct") {
			t.Errorf("expected prefixed type name, got: %s", buf.String())
		}
	})

	t.Run("disable marshal", func(t *testing.T) {
		gen := NewGoGenerator()
		var buf bytes.Buffer
		opts := DefaultOptions()
		opts.GenerateMarshal = false

		err := gen.Generate(&buf, s, opts)
		if err != nil {
			t.Fatalf("generate error: %v", err)
		}

		output := buf.String()
		if strings.Contains(output, "MarshalCramberry") {
			t.Error("expected no marshal methods")
		}
		// When no marshal and no required fields and no interfaces, should not have empty import block
		if strings.Contains(output, "import (") && !strings.Contains(output, `"github.com/cramberry`) {
			t.Error("expected no empty import block")
		}
	})

	t.Run("disable json", func(t *testing.T) {
		gen := NewGoGenerator()
		var buf bytes.Buffer
		opts := DefaultOptions()
		opts.GenerateJSON = false

		err := gen.Generate(&buf, s, opts)
		if err != nil {
			t.Fatalf("generate error: %v", err)
		}

		if strings.Contains(buf.String(), `json:"id"`) {
			t.Error("expected no json tags")
		}
	})
}

func TestCaseConversions(t *testing.T) {
	tests := []struct {
		input  string
		pascal string
		camel  string
		snake  string
		upper  string
		kebab  string
	}{
		{"foo", "Foo", "foo", "foo", "FOO", "foo"},
		{"fooBar", "FooBar", "fooBar", "foo_bar", "FOO_BAR", "foo-bar"},
		{"FooBar", "FooBar", "fooBar", "foo_bar", "FOO_BAR", "foo-bar"},
		{"foo_bar", "FooBar", "fooBar", "foo_bar", "FOO_BAR", "foo-bar"},
		{"FOO_BAR", "FooBar", "fooBar", "foo_bar", "FOO_BAR", "foo-bar"},
		{"foo-bar", "FooBar", "fooBar", "foo_bar", "FOO_BAR", "foo-bar"},
		{"ID", "Id", "id", "id", "ID", "id"},
		{"userID", "UserId", "userId", "user_id", "USER_ID", "user-id"},
		// Empty and single character
		{"", "", "", "", "", ""},
		{"a", "A", "a", "a", "A", "a"},
		// Unicode handling (using ASCII-only for deterministic behavior)
		{"café", "Café", "café", "café", "CAFÉ", "café"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ToPascalCase(tt.input); got != tt.pascal {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, got, tt.pascal)
			}
			if got := ToCamelCase(tt.input); got != tt.camel {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, got, tt.camel)
			}
			if got := ToSnakeCase(tt.input); got != tt.snake {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, got, tt.snake)
			}
			if got := ToUpperSnakeCase(tt.input); got != tt.upper {
				t.Errorf("ToUpperSnakeCase(%q) = %q, want %q", tt.input, got, tt.upper)
			}
			if got := ToKebabCase(tt.input); got != tt.kebab {
				t.Errorf("ToKebabCase(%q) = %q, want %q", tt.input, got, tt.kebab)
			}
		})
	}
}

func TestGeneratorRegistry(t *testing.T) {
	// Go generator should be registered
	gen, ok := Get(LanguageGo)
	if !ok {
		t.Fatal("Go generator not registered")
	}

	if gen.Language() != LanguageGo {
		t.Errorf("expected Go language, got %s", gen.Language())
	}

	if gen.FileExtension() != ".go" {
		t.Errorf("expected .go extension, got %s", gen.FileExtension())
	}

	// Check languages list
	langs := Languages()
	found := false
	for _, l := range langs {
		if l == LanguageGo {
			found = true
			break
		}
	}
	if !found {
		t.Error("Go not in languages list")
	}
}

func TestIndent(t *testing.T) {
	input := "line1\nline2\nline3"
	expected := "\t\tline1\n\t\tline2\n\t\tline3"
	got := Indent(input, 2)
	if got != expected {
		t.Errorf("Indent() = %q, want %q", got, expected)
	}
}

func TestGoComment(t *testing.T) {
	input := "This is a comment\nWith multiple lines"
	expected := "// This is a comment\n// With multiple lines"
	got := GoComment(input)
	if got != expected {
		t.Errorf("GoComment() = %q, want %q", got, expected)
	}
}

func TestGeneratorError(t *testing.T) {
	err := &GeneratorError{
		Message: "test error",
		Position: schema.Position{
			Filename: "test.go",
			Line:     10,
			Column:   5,
		},
	}

	expected := "test.go:10:5: test error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Test without position
	err2 := &GeneratorError{Message: "no position"}
	if err2.Error() != "no position" {
		t.Errorf("Error() = %q, want %q", err2.Error(), "no position")
	}
}

func TestGoGeneratorMarshalMethods(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
				},
			},
		},
	}

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.GenerateMarshal = true

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check MarshalCramberry
	if !strings.Contains(output, "func (m *User) MarshalCramberry() ([]byte, error)") {
		t.Error("expected MarshalCramberry method")
	}

	// Check UnmarshalCramberry
	if !strings.Contains(output, "func (m *User) UnmarshalCramberry(data []byte) error") {
		t.Error("expected UnmarshalCramberry method")
	}
}

func TestGoGeneratorDocComments(t *testing.T) {
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

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.GenerateComments = true

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check comments are present
	if !strings.Contains(output, "// User represents a user.") {
		t.Error("expected message doc comment")
	}
	if !strings.Contains(output, "// Unique identifier.") {
		t.Error("expected field doc comment")
	}
}

func TestGoGeneratorImportPaths(t *testing.T) {
	s := &schema.Schema{
		Package: &schema.Package{Name: "models"},
		Imports: []*schema.Import{
			{Path: "types.cram", Alias: "types"},
		},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
					{Name: "address", Number: 2, Type: &schema.NamedType{Package: "types", Name: "Address"}},
				},
			},
		},
	}

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.ImportPaths = map[string]string{
		"types": "example.com/myapp/types",
	}

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Check import statement is generated
	if !strings.Contains(output, `types "example.com/myapp/types"`) {
		t.Errorf("expected import statement, got: %s", output)
	}

	// Check field uses qualified type name
	if !strings.Contains(output, "Address types.Address") {
		t.Errorf("expected qualified type, got: %s", output)
	}

	// Check EncodeTo uses qualified method call
	if !strings.Contains(output, "m.Address.EncodeTo(w)") {
		t.Errorf("expected EncodeTo call, got: %s", output)
	}

	// Check DecodeFrom uses qualified method call
	if !strings.Contains(output, "m.Address.DecodeFrom(r)") {
		t.Errorf("expected DecodeFrom call, got: %s", output)
	}
}

func TestGoGeneratorNoExternalImports(t *testing.T) {
	// Test that no external imports are generated when no ImportPaths are specified
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
					{Name: "address", Number: 2, Type: &schema.NamedType{Package: "types", Name: "Address"}},
				},
			},
		},
	}

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()
	// No ImportPaths specified

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Should still have types.Address reference but no import
	if !strings.Contains(output, "Address types.Address") {
		t.Errorf("expected qualified type, got: %s", output)
	}

	// Should only have cramberry import
	if strings.Contains(output, `types "`) {
		t.Errorf("expected no external import, got: %s", output)
	}
}

func TestGoGeneratorSamePackageImport(t *testing.T) {
	// When imported schema has the same package name, types should not be qualified
	mainSchema := &schema.Schema{
		Package: &schema.Package{Name: "myproject"},
		Messages: []*schema.Message{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}},
					// This references a type from an import alias, but same package
					{Name: "address", Number: 2, Type: &schema.NamedType{Package: "types", Name: "Address"}},
				},
			},
		},
	}

	// Imported schema with same package name
	importedSchema := &schema.Schema{
		Package: &schema.Package{Name: "myproject"},
		Messages: []*schema.Message{
			{
				Name: "Address",
				Fields: []*schema.Field{
					{Name: "street", Number: 1, Type: &schema.ScalarType{Name: "string"}},
				},
			},
		},
	}

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.ImportedSchemas = map[string]*schema.Schema{
		"types": importedSchema,
	}

	err := gen.Generate(&buf, mainSchema, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Should NOT have types.Address - should be just Address since same package
	if strings.Contains(output, "types.Address") {
		t.Errorf("expected unqualified type for same-package import, got: %s", output)
	}

	// Should have just Address
	if !strings.Contains(output, "Address Address") {
		t.Errorf("expected unqualified Address type, got: %s", output)
	}

	// Should NOT have an import statement for types
	if strings.Contains(output, `types "`) {
		t.Errorf("expected no import for same-package, got: %s", output)
	}
}

func TestGoGeneratorRequiredStructFieldNoNilCheck(t *testing.T) {
	// Required struct (NamedType) fields should NOT have nil checks in Validate()
	// because they are value types, not pointers
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "Address",
				Fields: []*schema.Field{
					{Name: "street", Number: 1, Type: &schema.ScalarType{Name: "string"}},
				},
			},
			{
				Name: "User",
				Fields: []*schema.Field{
					// Required scalar - should have nil check (becomes *int32)
					{Name: "id", Number: 1, Type: &schema.ScalarType{Name: "int32"}, Required: true},
					// Required struct - should NOT have nil check (stays Address, not *Address)
					{Name: "address", Number: 2, Type: &schema.NamedType{Name: "Address"}, Required: true},
				},
			},
		},
	}

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Required scalar should be pointer and have nil check
	if !strings.Contains(output, "Id *int32") {
		t.Errorf("expected required scalar to be pointer, got: %s", output)
	}
	if !strings.Contains(output, "if m.Id == nil") {
		t.Errorf("expected nil check for required scalar, got: %s", output)
	}

	// Required struct should NOT be pointer (it's a value type)
	if strings.Contains(output, "Address *Address") {
		t.Errorf("required struct should not be pointer, got: %s", output)
	}
	// Should NOT have nil check for the struct field (can't compare struct to nil)
	if strings.Contains(output, "if m.Address == nil") {
		t.Errorf("should NOT have nil check for struct field, got: %s", output)
	}
}

func TestGoGeneratorSchemaPointerField(t *testing.T) {
	// Schema pointer type fields (*Hash) should decode correctly without double indirection
	s := &schema.Schema{
		Package: &schema.Package{Name: "test"},
		Messages: []*schema.Message{
			{
				Name: "Hash",
				Fields: []*schema.Field{
					{Name: "data", Number: 1, Type: &schema.ScalarType{Name: "bytes"}},
				},
			},
			{
				Name: "Block",
				Fields: []*schema.Field{
					{Name: "height", Number: 1, Type: &schema.ScalarType{Name: "int64"}},
					// Schema pointer type - should stay as *Hash, not become **Hash
					{Name: "previousHash", Number: 2, Type: &schema.PointerType{
						Element: &schema.NamedType{Name: "Hash"},
					}},
				},
			},
		},
	}

	gen := NewGoGenerator()
	var buf bytes.Buffer
	opts := DefaultOptions()

	err := gen.Generate(&buf, s, opts)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	output := buf.String()

	// Field should be *Hash (single pointer)
	if !strings.Contains(output, "PreviousHash *Hash") {
		t.Errorf("expected *Hash field type, got: %s", output)
	}

	// Decode should create var v Hash, then assign &v (not var tmp *Hash, then &tmp)
	if strings.Contains(output, "var tmp *Hash") {
		t.Errorf("should not have var tmp *Hash (would cause **Hash), got: %s", output)
	}

	// Should have the correct decode pattern: var v Hash; ... m.PreviousHash = &v
	if !strings.Contains(output, "var v Hash") {
		t.Errorf("expected var v Hash for decoding, got: %s", output)
	}
	if !strings.Contains(output, "m.PreviousHash = &v") {
		t.Errorf("expected m.PreviousHash = &v assignment, got: %s", output)
	}

	// Encode should have nil check
	if !strings.Contains(output, "if m.PreviousHash != nil") {
		t.Errorf("expected nil check for pointer field encoding, got: %s", output)
	}
}
