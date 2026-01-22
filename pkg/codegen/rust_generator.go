package codegen

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/blockberries/cramberry/pkg/schema"
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
		"rustWireType":      c.rustWireType,
		"rustWriteField":    c.rustWriteField,
		"rustReadField":     c.rustReadField,
		"comment":           c.rustComment,
		"toCamel":           ToCamelCase,
		"toPascal":          ToPascalCase,
		"toSnake":           ToSnakeCase,
		"generateComments":  func() bool { return c.Options.GenerateComments },
		"generateMarshal":   func() bool { return c.Options.GenerateMarshal },
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

// rustWireType returns the V2 wire type constant for a field type.
// This matches Go's V2 wire format for cross-runtime compatibility.
func (c *rustContext) rustWireType(f *schema.Field) string {
	return c.rustWireTypeForType(f.Type)
}

func (c *rustContext) rustWireTypeForType(t schema.TypeRef) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool", "uint8", "uint16", "uint32", "uint", "uint64":
			return "WireTypeV2::Varint" // Unsigned varint
		case "int8", "int16", "int32", "int", "int64":
			return "WireTypeV2::SVarint" // Signed zigzag varint
		case "float32":
			return "WireTypeV2::Fixed32"
		case "float64":
			return "WireTypeV2::Fixed64"
		case "string", "bytes":
			return "WireTypeV2::Bytes"
		default:
			return "WireTypeV2::Bytes"
		}
	case *schema.NamedType:
		// Named types (enums, messages) - enums are svarint, messages are bytes
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return "WireTypeV2::SVarint"
			}
		}
		return "WireTypeV2::Bytes"
	case *schema.ArrayType, *schema.MapType:
		return "WireTypeV2::Bytes"
	case *schema.PointerType:
		return "WireTypeV2::Bytes" // Nullable fields use bytes with length prefix
	default:
		return "WireTypeV2::Bytes"
	}
}

// rustWriteField generates the code to write a field value.
func (c *rustContext) rustWriteField(f *schema.Field) string {
	fieldName := "msg." + ToSnakeCase(f.Name)
	return c.rustWriteValue(f.Type, fieldName, f.Repeated)
}

// rustWriteValueForSubWriter generates write code using sub_writer instead of writer
func (c *rustContext) rustWriteValueForSubWriter(t schema.TypeRef, value string) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return fmt.Sprintf("sub_writer.write_bool(*%s)", value)
		case "int8", "int16", "int32", "int":
			return fmt.Sprintf("sub_writer.write_svarint(*%s)", value)
		case "uint8", "uint16", "uint32", "uint":
			return fmt.Sprintf("sub_writer.write_varint(*%s)", value)
		case "int64":
			return fmt.Sprintf("sub_writer.write_svarint64(*%s)", value)
		case "uint64":
			return fmt.Sprintf("sub_writer.write_varint64(*%s)", value)
		case "float32":
			return fmt.Sprintf("sub_writer.write_float32(*%s)", value)
		case "float64":
			return fmt.Sprintf("sub_writer.write_float64(*%s)", value)
		case "string":
			return fmt.Sprintf("sub_writer.write_string(%s)", value)
		case "bytes":
			return fmt.Sprintf("sub_writer.write_length_prefixed_bytes(%s)", value)
		default:
			return fmt.Sprintf("sub_writer.write_string(%s)", value)
		}
	case *schema.NamedType:
		// Check if it's an enum
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return fmt.Sprintf("sub_writer.write_svarint(*%s as i32)", value)
			}
		}
		// It's a message
		return fmt.Sprintf("encode_%s(&mut sub_writer, %s)", ToSnakeCase(typ.Name), value)
	default:
		return fmt.Sprintf("sub_writer.write_string(&format!(\"{:?}\", %s))", value)
	}
}

func (c *rustContext) rustWriteValue(t schema.TypeRef, value string, repeated bool) string {
	if repeated {
		elemType := t
		if arr, ok := t.(*schema.ArrayType); ok {
			elemType = arr.Element
		}
		elemWrite := c.rustWriteValueForSubWriter(elemType, "elem")
		return fmt.Sprintf(`{
        let mut sub_writer = Writer::new();
        sub_writer.write_varint(%s.len() as u32)?;
        for elem in &%s {
            %s?;
        }
        writer.write_length_prefixed_bytes(sub_writer.as_bytes())
    }`, value, value, elemWrite)
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return fmt.Sprintf("writer.write_bool(%s)", value)
		case "int8", "int16", "int32", "int":
			return fmt.Sprintf("writer.write_svarint(%s)", value)
		case "uint8", "uint16", "uint32", "uint":
			return fmt.Sprintf("writer.write_varint(%s)", value)
		case "int64":
			return fmt.Sprintf("writer.write_svarint64(%s)", value)
		case "uint64":
			return fmt.Sprintf("writer.write_varint64(%s)", value)
		case "float32":
			return fmt.Sprintf("writer.write_float32(%s)", value)
		case "float64":
			return fmt.Sprintf("writer.write_float64(%s)", value)
		case "string":
			return fmt.Sprintf("writer.write_string(&%s)", value)
		case "bytes":
			return fmt.Sprintf("writer.write_length_prefixed_bytes(&%s)", value)
		default:
			return fmt.Sprintf("writer.write_string(&%s)", value)
		}
	case *schema.NamedType:
		// Check if it's an enum
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return fmt.Sprintf("writer.write_svarint(%s as i32)", value)
			}
		}
		// It's a message
		return fmt.Sprintf("encode_%s(writer, &%s)", ToSnakeCase(typ.Name), value)
	case *schema.ArrayType:
		return c.rustWriteValue(typ.Element, value, true)
	case *schema.MapType:
		keyWrite := c.rustWriteValueForSubWriter(typ.Key, "k")
		valWrite := c.rustWriteValueForSubWriter(typ.Value, "v")
		return fmt.Sprintf(`{
        let mut sub_writer = Writer::new();
        sub_writer.write_varint(%s.len() as u32)?;
        for (k, v) in &%s {
            %s?;
            %s?;
        }
        writer.write_length_prefixed_bytes(sub_writer.as_bytes())
    }`, value, value, keyWrite, valWrite)
	case *schema.PointerType:
		innerWrite := c.rustWriteValue(typ.Element, "inner", false)
		return fmt.Sprintf(`if let Some(inner) = &%s {
        %s
    } else {
        Ok(())
    }`, value, innerWrite)
	default:
		return fmt.Sprintf("writer.write_string(&format!(\"{:?}\", %s))", value)
	}
}

// rustReadField generates the code to read a field value.
func (c *rustContext) rustReadField(f *schema.Field) string {
	return c.rustReadValue(f.Type, f.Repeated)
}

func (c *rustContext) rustReadValue(t schema.TypeRef, repeated bool) string {
	if repeated {
		elemType := t
		if arr, ok := t.(*schema.ArrayType); ok {
			elemType = arr.Element
		}
		elemRead := c.rustReadValue(elemType, false)
		return fmt.Sprintf(`{
            let data = reader.read_length_prefixed_bytes()?;
            let mut sub_reader = Reader::new(data);
            let len = sub_reader.read_varint()? as usize;
            let mut result = Vec::with_capacity(len);
            for _ in 0..len {
                result.push(%s);
            }
            result
        }`, elemRead)
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return "reader.read_bool()?"
		case "int8", "int16", "int32", "int":
			return "reader.read_svarint()?"
		case "uint8", "uint16", "uint32", "uint":
			return "reader.read_varint()?"
		case "int64":
			return "reader.read_svarint64()?"
		case "uint64":
			return "reader.read_varint64()?"
		case "float32":
			return "reader.read_float32()?"
		case "float64":
			return "reader.read_float64()?"
		case "string":
			return "reader.read_string()?.to_string()"
		case "bytes":
			return "reader.read_length_prefixed_bytes()?.to_vec()"
		default:
			return "reader.read_string()?.to_string()"
		}
	case *schema.NamedType:
		// Check if it's an enum
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				enumType := c.rustEnumType(e)
				return fmt.Sprintf("%s::from_i32(reader.read_svarint()?).unwrap_or(%s::%s)", enumType, enumType, ToPascalCase(e.Values[0].Name))
			}
		}
		// It's a message
		return fmt.Sprintf("decode_%s(reader)?", ToSnakeCase(typ.Name))
	case *schema.ArrayType:
		return c.rustReadValue(typ.Element, true)
	case *schema.MapType:
		keyRead := c.rustReadValue(typ.Key, false)
		valRead := c.rustReadValue(typ.Value, false)
		return fmt.Sprintf(`{
            let data = reader.read_length_prefixed_bytes()?;
            let mut sub_reader = Reader::new(data);
            let len = sub_reader.read_varint()? as usize;
            let mut result = std::collections::HashMap::with_capacity(len);
            for _ in 0..len {
                let k = %s;
                let v = %s;
                result.insert(k, v);
            }
            result
        }`, keyRead, valRead)
	case *schema.PointerType:
		innerRead := c.rustReadValue(typ.Element, false)
		return fmt.Sprintf("Some(Box::new(%s))", innerRead)
	default:
		return "reader.read_string()?.to_string()"
	}
}

func init() {
	Register(NewRustGenerator())
}

const rustTemplate = `// Code generated by cramberry. DO NOT EDIT.
// Source: {{.Schema.Position.Filename}}

{{if hasSerde}}use serde::{Deserialize, Serialize};
{{end}}{{if generateMarshal}}use cramberry::{Reader, Result, WireTypeV2, Writer};
{{end}}
{{$ctx := .}}
{{range $enum := .Schema.Enums}}
{{if generateComments}}{{range $enum.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default)]
{{if hasSerde}}#[derive(Serialize, Deserialize)]
{{end}}#[repr(i32)]
pub enum {{rustEnumType $enum}} {
#[default]
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
#[derive(Debug, Clone, PartialEq, Default)]
{{if hasSerde}}#[derive(Serialize, Deserialize)]
{{end}}pub struct {{rustMessageType $msg}} {
{{- range $msg.Fields}}
{{if generateComments}}{{range .Comments}}{{if .IsDoc}}    {{comment .Text}}
{{end}}{{end}}{{end -}}
{{if hasSerde}}    #[serde(rename = "{{toSnake .Name}}")]
{{end}}    pub {{rustFieldName .}}: {{rustFieldType .}},
{{- end}}
}
{{if generateMarshal}}
/// Encodes a {{rustMessageType $msg}} to the writer using V2 wire format.
pub fn encode_{{toSnake $msg.Name}}(writer: &mut Writer, msg: &{{rustMessageType $msg}}) -> Result<()> {
{{range $msg.Fields}}
    // Field {{.Number}}: {{.Name}}
    writer.write_compact_tag({{.Number}}, {{rustWireType .}})?;
    {{rustWriteField .}}?;
{{end}}
    // End marker
    writer.write_end_marker()?;
    Ok(())
}

/// Decodes a {{rustMessageType $msg}} from the reader using V2 wire format.
pub fn decode_{{toSnake $msg.Name}}(reader: &mut Reader) -> Result<{{rustMessageType $msg}}> {
{{- range $msg.Fields}}
    let mut {{rustFieldName .}}: {{rustFieldType .}} = Default::default();
{{- end}}

    loop {
        let (field_num, wire_type) = reader.read_compact_tag()?;
        if field_num == 0 { break; } // End marker

        match field_num {
{{- range $msg.Fields}}
            {{.Number}} => {{rustFieldName .}} = {{rustReadField .}},
{{- end}}
            _ => reader.skip_value_v2(wire_type)?,
        }
    }

    Ok({{rustMessageType $msg}} {
{{- range $msg.Fields}}
        {{rustFieldName .}},
{{- end}}
    })
}

/// Marshals a {{rustMessageType $msg}} to bytes.
pub fn marshal_{{toSnake $msg.Name}}(msg: &{{rustMessageType $msg}}) -> Result<Vec<u8>> {
    let mut writer = Writer::new();
    encode_{{toSnake $msg.Name}}(&mut writer, msg)?;
    Ok(writer.into_bytes())
}

/// Unmarshals a {{rustMessageType $msg}} from bytes.
pub fn unmarshal_{{toSnake $msg.Name}}(data: &[u8]) -> Result<{{rustMessageType $msg}}> {
    let mut reader = Reader::new(data);
    decode_{{toSnake $msg.Name}}(&mut reader)
}
{{end}}
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
