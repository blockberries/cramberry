package codegen

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/cramberry/cramberry-go/pkg/schema"
)

// RustGenerator generates Rust code from schemas.
type RustGenerator struct{}

// NewRustGenerator creates a new Rust code generator.
func NewRustGenerator() *RustGenerator {
	return &RustGenerator{}
}

// Language returns the target language.
func (g *RustGenerator) Language() Language {
	return LanguageRust
}

// FileExtension returns the file extension for generated files.
func (g *RustGenerator) FileExtension() string {
	return ".rs"
}

// Generate produces Rust code from a schema.
func (g *RustGenerator) Generate(w io.Writer, s *schema.Schema, opts Options) error {
	ctx := &rustContext{
		Schema:  s,
		Options: opts,
	}

	tmpl, err := template.New("rust").Funcs(ctx.funcMap()).Parse(rustTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl.Execute(w, ctx)
}

// rustContext holds context for Rust code generation.
type rustContext struct {
	Schema  *schema.Schema
	Options Options
}

func (c *rustContext) funcMap() template.FuncMap {
	return template.FuncMap{
		"rustType":          c.rustType,
		"rustFieldType":     c.rustFieldType,
		"rustEnumType":      c.rustEnumType,
		"rustMessageType":   c.rustMessageType,
		"rustInterfaceType": c.rustInterfaceType,
		"rustFieldName":     c.rustFieldName,
		"rustEnumValueName": c.rustEnumValueName,
		"comment":           c.rustComment,
		"toCamel":           ToCamelCase,
		"toPascal":          ToPascalCase,
		"toSnake":           ToSnakeCase,
		"generateComments":  func() bool { return c.Options.GenerateComments },
		"hasSerde":          func() bool { return c.Options.GenerateJSON },
	}
}

func (c *rustContext) rustType(t schema.TypeRef) string {
	return c.rustTypeInternal(t, false)
}

func (c *rustContext) rustFieldType(f *schema.Field) string {
	t := c.rustTypeInternal(f.Type, false)

	// Wrap repeated fields in Vec
	if f.Repeated {
		if _, isArray := f.Type.(*schema.ArrayType); !isArray {
			t = "Vec<" + t + ">"
		}
	}

	// Wrap optional fields in Option
	if f.Optional {
		if _, isPtr := f.Type.(*schema.PointerType); !isPtr {
			t = "Option<" + t + ">"
		}
	}

	return t
}

func (c *rustContext) rustTypeInternal(t schema.TypeRef, inArray bool) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.rustScalarType(typ.Name)
	case *schema.NamedType:
		name := c.Options.TypePrefix + ToPascalCase(typ.Name) + c.Options.TypeSuffix
		if typ.Package != "" {
			return ToSnakeCase(typ.Package) + "::" + name
		}
		return name
	case *schema.ArrayType:
		elem := c.rustTypeInternal(typ.Element, true)
		if typ.Size > 0 {
			return fmt.Sprintf("[%s; %d]", elem, typ.Size)
		}
		return "Vec<" + elem + ">"
	case *schema.MapType:
		key := c.rustTypeInternal(typ.Key, false)
		val := c.rustTypeInternal(typ.Value, false)
		return fmt.Sprintf("std::collections::HashMap<%s, %s>", key, val)
	case *schema.PointerType:
		elem := c.rustTypeInternal(typ.Element, false)
		return "Option<Box<" + elem + ">>"
	default:
		return "()"
	}
}

func (c *rustContext) rustScalarType(name string) string {
	switch name {
	case "bool":
		return "bool"
	case "int8":
		return "i8"
	case "int16":
		return "i16"
	case "int32", "int":
		return "i32"
	case "int64":
		return "i64"
	case "uint8", "byte":
		return "u8"
	case "uint16":
		return "u16"
	case "uint32", "uint":
		return "u32"
	case "uint64":
		return "u64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	case "complex64":
		return "(f32, f32)"
	case "complex128":
		return "(f64, f64)"
	case "string":
		return "String"
	case "bytes":
		return "Vec<u8>"
	default:
		return name
	}
}

func (c *rustContext) rustEnumType(e *schema.Enum) string {
	return c.Options.TypePrefix + ToPascalCase(e.Name) + c.Options.TypeSuffix
}

func (c *rustContext) rustMessageType(m *schema.Message) string {
	return c.Options.TypePrefix + ToPascalCase(m.Name) + c.Options.TypeSuffix
}

func (c *rustContext) rustInterfaceType(i *schema.Interface) string {
	return c.Options.TypePrefix + ToPascalCase(i.Name) + c.Options.TypeSuffix
}

func (c *rustContext) rustFieldName(f *schema.Field) string {
	name := ToSnakeCase(f.Name)
	// Handle Rust keywords
	switch name {
	case "type", "self", "super", "crate", "mod", "fn", "let", "mut", "ref",
		"const", "static", "move", "return", "if", "else", "match", "loop",
		"while", "for", "in", "break", "continue", "impl", "trait", "struct",
		"enum", "union", "pub", "use", "as", "where", "unsafe", "async", "await":
		return "r#" + name
	}
	return name
}

func (c *rustContext) rustEnumValueName(v *schema.EnumValue) string {
	return ToPascalCase(v.Name)
}

func (c *rustContext) rustComment(text string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, "/// "+line)
	}
	return strings.Join(result, "\n")
}

func init() {
	Register(NewRustGenerator())
}

const rustTemplate = `// Code generated by cramberry. DO NOT EDIT.
// Source: {{.Schema.Position.Filename}}

{{if hasSerde}}use serde::{Deserialize, Serialize};
{{end}}
{{$ctx := .}}
{{range $enum := .Schema.Enums}}
{{if generateComments}}{{range $enum.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
{{if hasSerde}}#[derive(Serialize, Deserialize)]
{{end}}#[repr(i32)]
pub enum {{rustEnumType $enum}} {
{{- range $enum.Values}}
{{if generateComments}}{{range .Comments}}{{if .IsDoc}}    {{comment .Text}}
{{end}}{{end}}{{end -}}
    {{rustEnumValueName .}} = {{.Number}},
{{- end}}
}

impl {{rustEnumType $enum}} {
    pub fn from_i32(value: i32) -> Option<Self> {
        match value {
{{- range $enum.Values}}
            {{.Number}} => Some(Self::{{rustEnumValueName .}}),
{{- end}}
            _ => None,
        }
    }
}

{{end}}
{{range $msg := .Schema.Messages}}
{{if generateComments}}{{range $msg.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
#[derive(Debug, Clone, PartialEq)]
{{if hasSerde}}#[derive(Serialize, Deserialize)]
{{end}}pub struct {{rustMessageType $msg}} {
{{- range $msg.Fields}}
{{if generateComments}}{{range .Comments}}{{if .IsDoc}}    {{comment .Text}}
{{end}}{{end}}{{end -}}
{{if hasSerde}}    #[serde(rename = "{{toSnake .Name}}")]
{{end}}    pub {{rustFieldName .}}: {{rustFieldType .}},
{{- end}}
}

{{end}}
{{range $iface := .Schema.Interfaces}}
{{if generateComments}}{{range $iface.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
#[derive(Debug, Clone, PartialEq)]
{{if hasSerde}}#[derive(Serialize, Deserialize)]
#[serde(tag = "_type")]
{{end}}pub enum {{rustInterfaceType $iface}} {
{{- range $iface.Implementations}}
    {{.Type.Name}}({{.Type.Name}}),
{{- end}}
}

impl {{rustInterfaceType $iface}} {
    pub fn type_id(&self) -> u32 {
        match self {
{{- range $iface.Implementations}}
            Self::{{.Type.Name}}(_) => {{.TypeID}},
{{- end}}
        }
    }
}

{{end}}
`
