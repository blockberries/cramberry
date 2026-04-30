// Package codegen provides code generation from Cramberry schema files.
package codegen

import (
	"fmt"
	"io"
	"slices"
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

// languageAliases maps the short / colloquial names a user might type
// to the canonical Language constants. This is keyed off the README's
// usage example, which shows `-lang ts` even though the canonical name
// is `typescript`.
var languageAliases = map[string]Language{
	"go":         LanguageGo,
	"golang":     LanguageGo,
	"ts":         LanguageTypeScript,
	"typescript": LanguageTypeScript,
	"js":         LanguageTypeScript, // close enough for a typo
	"rs":         LanguageRust,
	"rust":       LanguageRust,
}

// Get returns the generator for a language. Accepts both the canonical
// names (`go`, `typescript`, `rust`) and the short aliases (`ts`, `rs`,
// `golang`, `js`).
func Get(lang Language) (Generator, bool) {
	if gen, ok := registry[lang]; ok {
		return gen, true
	}
	if canonical, ok := languageAliases[string(lang)]; ok {
		gen, ok := registry[canonical]
		return gen, ok
	}
	return nil, false
}

// Languages returns all registered languages, sorted by name for stable
// output (used in error messages).
func Languages() []Language {
	langs := make([]Language, 0, len(registry))
	for lang := range registry {
		langs = append(langs, lang)
	}
	slices.Sort(langs)
	return langs
}

// ResolveNamedEnum returns the schema.Enum for a NamedType if it refers
// to an enum, looking in both the local schema and any imported
// schemas. Without the cross-package lookup, an imported enum field
// falls back to WireBytes (the default for messages) and produces a
// malformed wire format.
//
// Shared across all three generators — Go, TypeScript, and Rust — so
// the lookup behavior cannot drift between them.
func ResolveNamedEnum(local *schema.Schema, imports map[string]*schema.Schema, typ *schema.NamedType) (*schema.Enum, bool) {
	if typ.Package == "" {
		for _, e := range local.Enums {
			if e.Name == typ.Name {
				return e, true
			}
		}
		return nil, false
	}
	if imports != nil {
		if imported, ok := imports[typ.Package]; ok && imported != nil {
			for _, e := range imported.Enums {
				if e.Name == typ.Name {
					return e, true
				}
			}
		}
	}
	return nil, false
}

// IsNamedEnum is a convenience wrapper around ResolveNamedEnum.
func IsNamedEnum(local *schema.Schema, imports map[string]*schema.Schema, typ *schema.NamedType) bool {
	_, ok := ResolveNamedEnum(local, imports, typ)
	return ok
}

// ResolveNamedInterface returns the schema.Interface for a NamedType if
// it refers to an interface (rather than a message or enum), looking in
// both the local schema and any imported schemas.
//
// Shared across Go/Rust/TS generators so dispatch decisions stay
// consistent: a field whose type is a NamedType-pointing-at-Interface
// must invoke the polymorphic encoder/decoder, not the per-message one.
func ResolveNamedInterface(local *schema.Schema, imports map[string]*schema.Schema, typ *schema.NamedType) (*schema.Interface, bool) {
	if typ.Package == "" {
		for _, iface := range local.Interfaces {
			if iface.Name == typ.Name {
				return iface, true
			}
		}
		return nil, false
	}
	if imports != nil {
		if imported, ok := imports[typ.Package]; ok && imported != nil {
			for _, iface := range imported.Interfaces {
				if iface.Name == typ.Name {
					return iface, true
				}
			}
		}
	}
	return nil, false
}

// IsNamedInterface is a convenience wrapper around ResolveNamedInterface.
func IsNamedInterface(local *schema.Schema, imports map[string]*schema.Schema, typ *schema.NamedType) bool {
	_, ok := ResolveNamedInterface(local, imports, typ)
	return ok
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
