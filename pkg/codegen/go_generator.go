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
		"goType":               c.goType,
		"goFieldType":          c.goFieldType,
		"goEnumType":           c.goEnumType,
		"goMessageType":        c.goMessageType,
		"goInterfaceType":      c.goInterfaceType,
		"goPackage":            c.goPackage,
		"goFieldName":          c.goFieldName,
		"goEnumValueName":      c.goEnumValueName,
		"fieldTag":             c.fieldTag,
		"hasRequired":          c.hasRequired,
		"needsPointer":         c.needsPointer,
		"isPointerField":       c.isPointerField,
		"needsCramberryImport": c.needsCramberryImport,
		"comment":              GoComment,
		"indent":               Indent,
		"toCamel":              ToCamelCase,
		"toPascal":             ToPascalCase,
		"toSnake":              ToSnakeCase,
		"toUpperSnake":         ToUpperSnakeCase,
		"generateMarshal":      func() bool { return c.Options.GenerateMarshal },
		"generateJSON":         func() bool { return c.Options.GenerateJSON },
		"generateComments":     func() bool { return c.Options.GenerateComments },
		"wireTypeV2":           c.wireTypeV2,
		"encodeFieldV2":        c.encodeFieldV2,
		"decodeFieldV2":        c.decodeFieldV2,
		"zeroCheck":            c.zeroCheck,
		"isPackableSlice":      c.isPackableSlice,
	}
}

// wireTypeV2 returns the V2 wire type constant name for a field.
func (c *goContext) wireTypeV2(f *schema.Field) string {
	return c.wireTypeV2ForType(f.Type, f.Repeated)
}

func (c *goContext) wireTypeV2ForType(t schema.TypeRef, repeated bool) string {
	// Slices of packable types use Bytes wire type
	if repeated {
		return "cramberry.WireTypeV2Bytes"
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return "cramberry.WireTypeV2Varint"
		case "int8", "int16", "int32", "int64", "int":
			return "cramberry.WireTypeV2SVarint"
		case "uint8", "uint16", "uint32", "uint64", "uint", "byte":
			return "cramberry.WireTypeV2Varint"
		case "float32":
			return "cramberry.WireTypeV2Fixed32"
		case "float64":
			return "cramberry.WireTypeV2Fixed64"
		case "string", "bytes":
			return "cramberry.WireTypeV2Bytes"
		default:
			return "cramberry.WireTypeV2Bytes"
		}
	case *schema.ArrayType, *schema.MapType, *schema.NamedType:
		return "cramberry.WireTypeV2Bytes"
	default:
		return "cramberry.WireTypeV2Bytes"
	}
}

// encodeFieldV2 generates the encoding code for a field using V2 format.
func (c *goContext) encodeFieldV2(f *schema.Field) string {
	fieldName := "m." + ToPascalCase(f.Name)
	fieldNum := f.Number

	// Handle pointers first
	if c.isPointerField(f) {
		return c.encodePointerFieldV2(f, fieldName, fieldNum)
	}

	// Handle repeated fields
	if f.Repeated {
		return c.encodeRepeatedFieldV2(f, fieldName, fieldNum)
	}

	// Handle regular fields
	return c.encodeScalarFieldV2(f, fieldName, fieldNum)
}

func (c *goContext) encodePointerFieldV2(f *schema.Field, fieldName string, fieldNum int) string {
	wireType := c.wireTypeV2(f)
	inner := c.encodeValueV2(f.Type, fieldName, true)

	return fmt.Sprintf(`if %s != nil {
		w.WriteCompactTag(%d, %s)
		%s
	}`, fieldName, fieldNum, wireType, inner)
}

func (c *goContext) encodeRepeatedFieldV2(f *schema.Field, fieldName string, fieldNum int) string {
	wireType := c.wireTypeV2(f)

	// Check if it's a packable type
	if c.isPackableType(f.Type) {
		return fmt.Sprintf(`if len(%s) > 0 {
		w.WriteCompactTag(%d, %s)
		w.WriteUvarint(uint64(len(%s)))
		for _, v := range %s {
			%s
		}
	}`, fieldName, fieldNum, wireType, fieldName, fieldName, c.encodePackedElementV2(f.Type))
	}

	// Non-packable types (messages, strings, etc.)
	// Note: range variable v is the value, not a pointer
	return fmt.Sprintf(`if len(%s) > 0 {
		w.WriteCompactTag(%d, %s)
		w.WriteUvarint(uint64(len(%s)))
		for _, v := range %s {
			%s
		}
	}`, fieldName, fieldNum, wireType, fieldName, fieldName, c.encodeValueV2(f.Type, "v", false))
}

func (c *goContext) encodeScalarFieldV2(f *schema.Field, fieldName string, fieldNum int) string {
	wireType := c.wireTypeV2(f)
	zeroCheck := c.zeroCheck(f)
	inner := c.encodeValueV2(f.Type, fieldName, false)

	// For optional fields, always emit if non-zero
	if zeroCheck != "" {
		return fmt.Sprintf(`if %s {
		w.WriteCompactTag(%d, %s)
		%s
	}`, zeroCheck, fieldNum, wireType, inner)
	}

	// Always emit for required fields
	return fmt.Sprintf(`w.WriteCompactTag(%d, %s)
	%s`, fieldNum, wireType, inner)
}

func (c *goContext) encodeValueV2(t schema.TypeRef, varName string, isPointer bool) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		if isPointer {
			return c.encodeScalarV2(typ.Name, "*"+varName)
		}
		return c.encodeScalarV2(typ.Name, varName)
	case *schema.NamedType:
		// Named types are messages or enums
		// For enums and messages, call encodeTo
		return fmt.Sprintf(`%s.encodeTo(w)`, varName)
	case *schema.ArrayType:
		// Fixed-size arrays
		if typ.Size > 0 {
			return fmt.Sprintf(`w.WriteUvarint(uint64(len(%s)))
		for _, v := range %s {
			%s
		}`, varName, varName, c.encodeValueV2(typ.Element, "v", false))
		}
		// Dynamic slices
		return fmt.Sprintf(`w.WriteUvarint(uint64(len(%s)))
		for _, v := range %s {
			%s
		}`, varName, varName, c.encodeValueV2(typ.Element, "v", false))
	case *schema.MapType:
		return fmt.Sprintf(`w.WriteUvarint(uint64(len(%s)))
		for k, v := range %s {
			%s
			%s
		}`, varName, varName, c.encodeValueV2(typ.Key, "k", false), c.encodeValueV2(typ.Value, "v", false))
	default:
		return fmt.Sprintf("// TODO: encode %s", varName)
	}
}

func (c *goContext) encodeScalarV2(typeName, varName string) string {
	switch typeName {
	case "bool":
		return fmt.Sprintf("w.WriteBool(%s)", varName)
	case "int8":
		return fmt.Sprintf("w.WriteInt8(%s)", varName)
	case "int16":
		return fmt.Sprintf("w.WriteInt16(%s)", varName)
	case "int32":
		return fmt.Sprintf("w.WriteInt32(%s)", varName)
	case "int64":
		return fmt.Sprintf("w.WriteInt64(%s)", varName)
	case "int":
		return fmt.Sprintf("w.WriteInt64(int64(%s))", varName)
	case "uint8", "byte":
		return fmt.Sprintf("w.WriteUint8(%s)", varName)
	case "uint16":
		return fmt.Sprintf("w.WriteUint16(%s)", varName)
	case "uint32":
		return fmt.Sprintf("w.WriteUint32(%s)", varName)
	case "uint64":
		return fmt.Sprintf("w.WriteUint64(%s)", varName)
	case "uint":
		return fmt.Sprintf("w.WriteUint64(uint64(%s))", varName)
	case "float32":
		return fmt.Sprintf("w.WriteFloat32(%s)", varName)
	case "float64":
		return fmt.Sprintf("w.WriteFloat64(%s)", varName)
	case "string":
		return fmt.Sprintf("w.WriteString(%s)", varName)
	case "bytes":
		return fmt.Sprintf("w.WriteBytes(%s)", varName)
	default:
		return fmt.Sprintf("// TODO: encode %s", typeName)
	}
}

func (c *goContext) encodePackedElementV2(t schema.TypeRef) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.encodeScalarV2(typ.Name, "v")
	default:
		return "// TODO: encode packed element"
	}
}

// decodeFieldV2 generates the decoding code for a field using V2 format.
func (c *goContext) decodeFieldV2(f *schema.Field) string {
	fieldName := "m." + ToPascalCase(f.Name)

	// Handle repeated fields first
	if f.Repeated {
		return c.decodeRepeatedFieldV2(f, fieldName)
	}

	// Handle maps - they're reference types, no pointer wrapping needed
	if _, isMap := f.Type.(*schema.MapType); isMap {
		return c.decodeMapFieldV2(f, fieldName)
	}

	// Handle pointers (optional scalars or message fields)
	if c.isPointerField(f) {
		return c.decodePointerFieldV2(f, fieldName)
	}

	// Handle regular fields
	return c.decodeScalarFieldV2(f, fieldName)
}

func (c *goContext) decodePointerFieldV2(f *schema.Field, fieldName string) string {
	goType := c.goTypeInternal(f.Type, false)
	inner := c.decodeValueV2(f.Type, "tmp")

	return fmt.Sprintf(`var tmp %s
		%s
		%s = &tmp`, goType, inner, fieldName)
}

func (c *goContext) decodeMapFieldV2(f *schema.Field, fieldName string) string {
	mapType := f.Type.(*schema.MapType)
	keyType := c.goTypeInternal(mapType.Key, false)
	valType := c.goTypeInternal(mapType.Value, false)

	return fmt.Sprintf(`n := int(r.ReadUvarint())
		%s = make(map[%s]%s, n)
		for i := 0; i < n; i++ {
			var k %s
			%s
			var v %s
			%s
			%s[k] = v
		}`, fieldName, keyType, valType, keyType, c.decodeValueV2(mapType.Key, "k"), valType, c.decodeValueV2(mapType.Value, "v"), fieldName)
}

func (c *goContext) decodeRepeatedFieldV2(f *schema.Field, fieldName string) string {
	goType := c.goTypeInternal(f.Type, false)

	// Check if it's a packable type
	if c.isPackableType(f.Type) {
		return fmt.Sprintf(`n := int(r.ReadUvarint())
		%s = make([]%s, n)
		for i := 0; i < n; i++ {
			%s
		}`, fieldName, goType, c.decodePackedElementV2(f.Type, fieldName+"[i]"))
	}

	// Non-packable types
	return fmt.Sprintf(`n := int(r.ReadUvarint())
		%s = make([]%s, n)
		for i := 0; i < n; i++ {
			%s
		}`, fieldName, goType, c.decodeValueV2(f.Type, fieldName+"[i]"))
}

func (c *goContext) decodeScalarFieldV2(f *schema.Field, fieldName string) string {
	return c.decodeValueV2(f.Type, fieldName)
}

func (c *goContext) decodeValueV2(t schema.TypeRef, varName string) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.decodeScalarV2(typ.Name, varName)
	case *schema.NamedType:
		// Named types are messages or enums
		return fmt.Sprintf(`%s.decodeFrom(r)`, varName)
	case *schema.ArrayType:
		goType := c.goTypeInternal(typ.Element, true)
		return fmt.Sprintf(`n := int(r.ReadUvarint())
		%s = make([]%s, n)
		for i := 0; i < n; i++ {
			%s
		}`, varName, goType, c.decodeValueV2(typ.Element, varName+"[i]"))
	case *schema.MapType:
		keyType := c.goTypeInternal(typ.Key, false)
		valType := c.goTypeInternal(typ.Value, false)
		return fmt.Sprintf(`n := int(r.ReadUvarint())
		%s = make(map[%s]%s, n)
		for i := 0; i < n; i++ {
			var k %s
			%s
			var v %s
			%s
			%s[k] = v
		}`, varName, keyType, valType, keyType, c.decodeValueV2(typ.Key, "k"), valType, c.decodeValueV2(typ.Value, "v"), varName)
	default:
		return fmt.Sprintf("// TODO: decode %s", varName)
	}
}

func (c *goContext) decodeScalarV2(typeName, varName string) string {
	switch typeName {
	case "bool":
		return fmt.Sprintf("%s = r.ReadBool()", varName)
	case "int8":
		return fmt.Sprintf("%s = r.ReadInt8()", varName)
	case "int16":
		return fmt.Sprintf("%s = r.ReadInt16()", varName)
	case "int32":
		return fmt.Sprintf("%s = r.ReadInt32()", varName)
	case "int64":
		return fmt.Sprintf("%s = r.ReadInt64()", varName)
	case "int":
		return fmt.Sprintf("%s = int(r.ReadInt64())", varName)
	case "uint8", "byte":
		return fmt.Sprintf("%s = r.ReadUint8()", varName)
	case "uint16":
		return fmt.Sprintf("%s = r.ReadUint16()", varName)
	case "uint32":
		return fmt.Sprintf("%s = r.ReadUint32()", varName)
	case "uint64":
		return fmt.Sprintf("%s = r.ReadUint64()", varName)
	case "uint":
		return fmt.Sprintf("%s = uint(r.ReadUint64())", varName)
	case "float32":
		return fmt.Sprintf("%s = r.ReadFloat32()", varName)
	case "float64":
		return fmt.Sprintf("%s = r.ReadFloat64()", varName)
	case "string":
		return fmt.Sprintf("%s = r.ReadString()", varName)
	case "bytes":
		return fmt.Sprintf("%s = r.ReadBytes()", varName)
	default:
		return fmt.Sprintf("// TODO: decode %s", typeName)
	}
}

func (c *goContext) decodePackedElementV2(t schema.TypeRef, varName string) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.decodeScalarV2(typ.Name, varName)
	default:
		return "// TODO: decode packed element"
	}
}

// zeroCheck returns the condition to check if a field is non-zero (for omitempty).
func (c *goContext) zeroCheck(f *schema.Field) string {
	fieldName := "m." + ToPascalCase(f.Name)

	if f.Repeated {
		return fmt.Sprintf("len(%s) > 0", fieldName)
	}

	switch typ := f.Type.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return fieldName
		case "string":
			return fmt.Sprintf("%s != \"\"", fieldName)
		case "bytes":
			return fmt.Sprintf("len(%s) > 0", fieldName)
		case "int8", "int16", "int32", "int64", "int",
			"uint8", "uint16", "uint32", "uint64", "uint",
			"float32", "float64", "byte":
			return fmt.Sprintf("%s != 0", fieldName)
		default:
			return ""
		}
	case *schema.NamedType, *schema.ArrayType, *schema.MapType:
		return "" // Always encode nested types
	default:
		return ""
	}
}

// isPackableType returns true if the type can be packed in a contiguous byte sequence.
func (c *goContext) isPackableType(t schema.TypeRef) bool {
	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool", "int8", "int16", "int32", "int64", "int",
			"uint8", "uint16", "uint32", "uint64", "uint", "byte",
			"float32", "float64":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// isPackableSlice returns true if the field is a repeated packable type.
func (c *goContext) isPackableSlice(f *schema.Field) bool {
	return f.Repeated && c.isPackableType(f.Type)
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

	// Wrap repeated fields in slice
	if f.Repeated {
		// Don't double-wrap if already a slice
		if _, isArray := f.Type.(*schema.ArrayType); !isArray {
			t = "[]" + t
		}
	}

	// Optional fields become pointers
	if f.Optional && !c.needsPointer(f.Type) && !f.Repeated {
		return "*" + t
	}

	// Required scalar fields become pointers so we can distinguish nil (not set) from zero value
	if f.Required && c.isScalarType(f.Type) && !f.Repeated {
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

// isScalarType returns true if the type is a scalar (non-reference) type.
func (c *goContext) isScalarType(t schema.TypeRef) bool {
	switch typ := t.(type) {
	case *schema.ScalarType:
		// bytes is []byte which is a reference type
		return typ.Name != "bytes"
	default:
		return false
	}
}

// isPointerField returns true if the field will be generated as a pointer.
func (c *goContext) isPointerField(f *schema.Field) bool {
	if f.Repeated {
		return false
	}
	if c.needsPointer(f.Type) {
		return true
	}
	if f.Optional {
		return true
	}
	if f.Required && c.isScalarType(f.Type) {
		return true
	}
	return false
}

// needsCramberryImport returns true if the generated code needs to import cramberry.
// This is true when:
// - GenerateMarshal is enabled (for Marshal/Unmarshal methods)
// - There are messages with required fields (for Validate method)
// - There are interfaces (for TypeID function)
func (c *goContext) needsCramberryImport() bool {
	if c.Options.GenerateMarshal {
		return true
	}
	// Check for required fields in any message
	for _, msg := range c.Schema.Messages {
		for _, f := range msg.Fields {
			if f.Required {
				return true
			}
		}
	}
	// Check for interfaces
	if len(c.Schema.Interfaces) > 0 {
		return true
	}
	return false
}

func init() {
	Register(NewGoGenerator())
}

const goTemplate = `// Code generated by cramberry. DO NOT EDIT.
// Source: {{.Schema.Position.Filename}}

package {{goPackage}}
{{if needsCramberryImport}}
import (
	"github.com/cramberry/cramberry-go/pkg/cramberry"
)
{{end}}
{{$ctx := .}}
{{range $enum := .Schema.Enums}}
{{if generateComments}}{{range $enum.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
type {{goEnumType $enum}} int32

const (
{{- range $i, $v := $enum.Values}}
	{{goEnumValueName $enum $v}} {{goEnumType $enum}} = {{$v.Number}}
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

// encodeTo encodes the enum value directly to the writer.
func (e {{goEnumType $enum}}) encodeTo(w *cramberry.Writer) {
	w.WriteInt32(int32(e))
}

// decodeFrom decodes the enum value from the reader.
func (e *{{goEnumType $enum}}) decodeFrom(r *cramberry.Reader) {
	*e = {{goEnumType $enum}}(r.ReadInt32())
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
// MarshalCramberry encodes the message to binary format using optimized V2 encoding.
// This method uses direct field access without reflection for maximum performance.
func (m *{{goMessageType $msg}}) MarshalCramberry() ([]byte, error) {
	w := cramberry.GetWriter()
	defer cramberry.PutWriter(w)

	m.encodeTo(w)

	if w.Err() != nil {
		return nil, w.Err()
	}
	return w.BytesCopy(), nil
}

// encodeTo encodes the message directly to the writer using V2 format.
func (m *{{goMessageType $msg}}) encodeTo(w *cramberry.Writer) {
{{- range $msg.Fields}}
	{{encodeFieldV2 .}}
{{- end}}
	w.WriteEndMarker()
}

// UnmarshalCramberry decodes the message from binary format using optimized V2 decoding.
// This method uses direct field access without reflection for maximum performance.
func (m *{{goMessageType $msg}}) UnmarshalCramberry(data []byte) error {
	r := cramberry.NewReaderWithOptions(data, cramberry.DefaultOptions)
	m.decodeFrom(r)
	return r.Err()
}

// decodeFrom decodes the message from the reader using V2 format.
func (m *{{goMessageType $msg}}) decodeFrom(r *cramberry.Reader) {
	for {
		fieldNum, _ := r.ReadCompactTag()
		if fieldNum == 0 {
			break
		}
		switch fieldNum {
{{- range $msg.Fields}}
		case {{.Number}}:
			{{decodeFieldV2 .}}
{{- end}}
		default:
			// Skip unknown field - read wire type would have been needed
			// For now, just break as we can't determine how to skip
			break
		}
		if r.Err() != nil {
			return
		}
	}
}
{{end}}
{{- if hasRequired $msg}}
// Validate validates that all required fields are set.
func (m *{{goMessageType $msg}}) Validate() error {
{{- range $msg.Fields}}{{if .Required}}
	// Field {{.Name}} is required
	if m.{{goFieldName .}} == nil {
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
