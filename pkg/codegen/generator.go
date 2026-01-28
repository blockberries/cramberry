// Package codegen provides code generation from Cramberry schema files.
package codegen

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/blockberries/cramberry/pkg/schema"
)

// Language represents a target code generation language.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
	LanguageRust       Language = "rust"
)

// Generator is the interface for code generators.
type Generator interface {
	// Generate produces code from a schema.
	Generate(w io.Writer, schema *schema.Schema, options Options) error

	// Language returns the target language.
	Language() Language

	// FileExtension returns the file extension for generated files.
	FileExtension() string
}

// Options configures code generation.
type Options struct {
	// Package overrides the package name from the schema.
	Package string

	// OutputPath is the base output directory.
	OutputPath string

	// GenerateMarshal generates Marshal/Unmarshal methods.
	GenerateMarshal bool

	// GenerateJSON generates JSON marshaling support.
	GenerateJSON bool

	// GenerateValidation generates validation methods.
	GenerateValidation bool

	// GenerateBuilder generates builder pattern methods.
	GenerateBuilder bool

	// GenerateComments includes comments from the schema.
	GenerateComments bool

	// TypePrefix adds a prefix to all type names.
	TypePrefix string

	// TypeSuffix adds a suffix to all type names.
	TypeSuffix string

	// ImportPaths maps schema import aliases to Go import paths.
	// For example: {"types": "example.com/myapp/types"}
	// This is used to generate proper import statements for imported types.
	ImportPaths map[string]string

	// ImportedSchemas maps import aliases to their parsed schemas.
	// This is used to determine if imported types are from the same package
	// (and thus don't need qualification in generated code).
	ImportedSchemas map[string]*schema.Schema
}

// DefaultOptions returns the default code generation options.
func DefaultOptions() Options {
	return Options{
		GenerateMarshal:    true,
		GenerateJSON:       true,
		GenerateComments:   true,
		GenerateValidation: false,
		GenerateBuilder:    false,
	}
}

// registry holds registered generators by language.
var registry = make(map[Language]Generator)

// Register registers a generator for a language.
func Register(gen Generator) {
	registry[gen.Language()] = gen
}

// Get returns the generator for a language.
func Get(lang Language) (Generator, bool) {
	gen, ok := registry[lang]
	return gen, ok
}

// Languages returns all registered languages.
func Languages() []Language {
	langs := make([]Language, 0, len(registry))
	for lang := range registry {
		langs = append(langs, lang)
	}
	return langs
}

// Helper functions for code generation

// titleCaser is used for converting strings to title case.
var titleCaser = cases.Title(language.English)

// ToPascalCase converts a string to PascalCase.
func ToPascalCase(s string) string {
	parts := splitName(s)
	for i, p := range parts {
		parts[i] = titleCaser.String(strings.ToLower(p))
	}
	return strings.Join(parts, "")
}

// ToCamelCase converts a string to camelCase.
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if len(pascal) == 0 {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

// ToSnakeCase converts a string to snake_case.
func ToSnakeCase(s string) string {
	parts := splitName(s)
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, "_")
}

// ToUpperSnakeCase converts a string to UPPER_SNAKE_CASE.
func ToUpperSnakeCase(s string) string {
	parts := splitName(s)
	for i, p := range parts {
		parts[i] = strings.ToUpper(p)
	}
	return strings.Join(parts, "_")
}

// ToKebabCase converts a string to kebab-case.
func ToKebabCase(s string) string {
	parts := splitName(s)
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, "-")
}

// splitName splits a name into parts based on underscores and case transitions.
func splitName(s string) []string {
	if s == "" {
		return nil
	}

	var parts []string
	var current strings.Builder

	for i, r := range s {
		if r == '_' || r == '-' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}

		// Check for case transition
		if i > 0 && isUpper(r) && !isUpper(rune(s[i-1])) {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// Indent indents each line of s by the given number of tabs.
func Indent(s string, tabs int) string {
	indent := strings.Repeat("\t", tabs)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// Comment wraps text as a comment with the given prefix.
func Comment(text, prefix string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + " " + line
	}
	return strings.Join(lines, "\n")
}

// GoComment wraps text as a Go doc comment.
func GoComment(text string) string {
	return Comment(text, "//")
}

// GeneratorError represents a code generation error.
type GeneratorError struct {
	Message  string
	Position schema.Position
}

func (e *GeneratorError) Error() string {
	if e.Position.Filename != "" {
		return fmt.Sprintf("%s:%d:%d: %s",
			e.Position.Filename, e.Position.Line, e.Position.Column, e.Message)
	}
	return e.Message
}
