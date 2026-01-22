package codegen

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/cramberry/cramberry-go/pkg/schema"
)

// GoGenerator generates Go code from schemas.
type GoGenerator struct{}

// NewGoGenerator creates a new Go code generator.
func NewGoGenerator() *GoGenerator {
	return &GoGenerator{}
}

// Language returns the target language.
func (g *GoGenerator) Language() Language {
	return LanguageGo
}

// FileExtension returns the file extension for generated files.
func (g *GoGenerator) FileExtension() string {
	return ".go"
}

// Generate produces Go code from a schema.
func (g *GoGenerator) Generate(w io.Writer, s *schema.Schema, opts Options) error {
	ctx := &goContext{
		Schema:  s,
		Options: opts,
	}

	tmpl, err := template.New("go").Funcs(ctx.funcMap()).Parse(goTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl.Execute(w, ctx)
}

// goContext holds context for Go code generation.
type goContext struct {
	Schema  *schema.Schema
	Options Options
}

func (c *goContext) funcMap() template.FuncMap {
	return template.FuncMap{
		"goType":           c.goType,
		"goFieldType":      c.goFieldType,
		"goEnumType":       c.goEnumType,
		"goMessageType":    c.goMessageType,
		"goInterfaceType":  c.goInterfaceType,
		"goPackage":        c.goPackage,
		"goFieldName":      c.goFieldName,
		"goEnumValueName":  c.goEnumValueName,
		"fieldTag":         c.fieldTag,
		"hasRequired":      c.hasRequired,
		"needsPointer":     c.needsPointer,
		"comment":          GoComment,
		"indent":           Indent,
		"toCamel":          ToCamelCase,
		"toPascal":         ToPascalCase,
		"toSnake":          ToSnakeCase,
		"toUpperSnake":     ToUpperSnakeCase,
		"generateMarshal":  func() bool { return c.Options.GenerateMarshal },
		"generateJSON":     func() bool { return c.Options.GenerateJSON },
		"generateComments": func() bool { return c.Options.GenerateComments },
	}
}

func (c *goContext) goPackage() string {
	if c.Options.Package != "" {
		return c.Options.Package
	}
	if c.Schema.Package != nil {
		return c.Schema.Package.Name
	}
	return "generated"
}

func (c *goContext) goType(t schema.TypeRef) string {
	return c.goTypeInternal(t, false)
}

func (c *goContext) goFieldType(f *schema.Field) string {
	t := c.goTypeInternal(f.Type, false)
	if f.Optional && !c.needsPointer(f.Type) {
		return "*" + t
	}
	return t
}

func (c *goContext) goTypeInternal(t schema.TypeRef, inArray bool) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.goScalarType(typ.Name)
	case *schema.NamedType:
		name := c.Options.TypePrefix + ToPascalCase(typ.Name) + c.Options.TypeSuffix
		if typ.Package != "" {
			return typ.Package + "." + name
		}
		return name
	case *schema.ArrayType:
		elem := c.goTypeInternal(typ.Element, true)
		if typ.Size > 0 {
			return fmt.Sprintf("[%d]%s", typ.Size, elem)
		}
		return "[]" + elem
	case *schema.MapType:
		key := c.goTypeInternal(typ.Key, false)
		val := c.goTypeInternal(typ.Value, false)
		return fmt.Sprintf("map[%s]%s", key, val)
	case *schema.PointerType:
		elem := c.goTypeInternal(typ.Element, false)
		return "*" + elem
	default:
		return "interface{}"
	}
}

func (c *goContext) goScalarType(name string) string {
	switch name {
	case "bool":
		return "bool"
	case "int8":
		return "int8"
	case "int16":
		return "int16"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "int":
		return "int"
	case "uint8", "byte":
		return "uint8"
	case "uint16":
		return "uint16"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "uint":
		return "uint"
	case "float32":
		return "float32"
	case "float64":
		return "float64"
	case "complex64":
		return "complex64"
	case "complex128":
		return "complex128"
	case "string":
		return "string"
	case "bytes":
		return "[]byte"
	default:
		return name
	}
}

func (c *goContext) goEnumType(e *schema.Enum) string {
	return c.Options.TypePrefix + ToPascalCase(e.Name) + c.Options.TypeSuffix
}

func (c *goContext) goMessageType(m *schema.Message) string {
	return c.Options.TypePrefix + ToPascalCase(m.Name) + c.Options.TypeSuffix
}

func (c *goContext) goInterfaceType(i *schema.Interface) string {
	return c.Options.TypePrefix + ToPascalCase(i.Name) + c.Options.TypeSuffix
}

func (c *goContext) goFieldName(f *schema.Field) string {
	return ToPascalCase(f.Name)
}

func (c *goContext) goEnumValueName(e *schema.Enum, v *schema.EnumValue) string {
	enumName := c.goEnumType(e)
	valueName := ToPascalCase(v.Name)
	return enumName + valueName
}

func (c *goContext) fieldTag(f *schema.Field) string {
	var parts []string

	// Cramberry tag
	cramTag := fmt.Sprintf("%d", f.Number)
	if f.Required {
		cramTag += ",required"
	}
	if f.Optional {
		cramTag += ",omitempty"
	}
	parts = append(parts, fmt.Sprintf(`cramberry:"%s"`, cramTag))

	// JSON tag if enabled
	if c.Options.GenerateJSON {
		jsonName := ToSnakeCase(f.Name)
		jsonTag := jsonName
		if f.Optional {
			jsonTag += ",omitempty"
		}
		parts = append(parts, fmt.Sprintf(`json:"%s"`, jsonTag))
	}

	return strings.Join(parts, " ")
}

func (c *goContext) hasRequired(m *schema.Message) bool {
	for _, f := range m.Fields {
		if f.Required {
			return true
		}
	}
	return false
}

func (c *goContext) needsPointer(t schema.TypeRef) bool {
	switch t.(type) {
	case *schema.PointerType:
		return true
	case *schema.ArrayType, *schema.MapType:
		return true // slices and maps are already pointer-like
	default:
		return false
	}
}

func init() {
	Register(NewGoGenerator())
}

const goTemplate = `// Code generated by cramberry. DO NOT EDIT.
// Source: {{.Schema.Position.Filename}}

package {{goPackage}}

import (
{{- if generateMarshal}}
	"github.com/cramberry/cramberry-go/pkg/cramberry"
{{- end}}
)
{{$ctx := .}}
{{range $enum := .Schema.Enums}}
{{if generateComments}}{{range $enum.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
type {{goEnumType $enum}} int32

const (
{{- range $i, $v := $enum.Values}}
	{{goEnumValueName $enum $v}} {{if eq $i 0}}{{goEnumType $enum}} = {{end}}{{$v.Number}}
{{- end}}
)

// String returns the string representation of the enum value.
func (e {{goEnumType $enum}}) String() string {
	switch e {
{{- range $enum.Values}}
	case {{goEnumValueName $enum .}}:
		return "{{.Name}}"
{{- end}}
	default:
		return "UNKNOWN"
	}
}

// IsValid returns true if the value is a valid enum value.
func (e {{goEnumType $enum}}) IsValid() bool {
	switch e {
{{- range $enum.Values}}
	case {{goEnumValueName $enum .}}:
		return true
{{- end}}
	default:
		return false
	}
}
{{end}}
{{range $msg := .Schema.Messages}}
{{if generateComments}}{{range $msg.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
type {{goMessageType $msg}} struct {
{{- range $msg.Fields}}
{{if generateComments}}{{range .Comments}}{{if .IsDoc}}	{{comment .Text}}
{{end}}{{end}}{{end -}}
	{{goFieldName .}} {{goFieldType .}} ` + "`{{fieldTag .}}`" + `
{{- end}}
}
{{if generateMarshal}}
// MarshalCramberry encodes the message to binary format.
func (m *{{goMessageType $msg}}) MarshalCramberry() ([]byte, error) {
	return cramberry.Marshal(m)
}

// UnmarshalCramberry decodes the message from binary format.
func (m *{{goMessageType $msg}}) UnmarshalCramberry(data []byte) error {
	return cramberry.Unmarshal(data, m)
}
{{end}}
{{- if hasRequired $msg}}
// Validate validates that all required fields are set.
func (m *{{goMessageType $msg}}) Validate() error {
{{- range $msg.Fields}}{{if .Required}}
	// Field {{.Name}} is required
	if m.{{goFieldName .}} == {{if eq (goFieldType .) "string"}}""{{else if eq (goFieldType .) "bool"}}false{{else}}0{{end}} {
		return cramberry.NewValidationError("{{goMessageType $msg}}", "{{.Name}}", "required field is missing")
	}
{{- end}}{{end}}
	return nil
}
{{end}}
{{end}}
{{range $iface := .Schema.Interfaces}}
{{if generateComments}}{{range $iface.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
// {{goInterfaceType $iface}} is a polymorphic interface.
type {{goInterfaceType $iface}} interface {
	is{{goInterfaceType $iface}}()
}

{{range $iface.Implementations}}
func (*{{.Type.Name}}) is{{goInterfaceType $iface}}() {}
{{end}}

// {{goInterfaceType $iface}}TypeID returns the type ID for interface implementations.
func {{goInterfaceType $iface}}TypeID(v {{goInterfaceType $iface}}) cramberry.TypeID {
	switch v.(type) {
{{- range $iface.Implementations}}
	case *{{.Type.Name}}:
		return {{.TypeID}}
{{- end}}
	default:
		return 0
	}
}
{{end}}
`
