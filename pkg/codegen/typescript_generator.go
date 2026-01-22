package codegen

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/blockberries/cramberry/pkg/schema"
)

// TypeScriptGenerator generates TypeScript code from schemas.
type TypeScriptGenerator struct{}

// NewTypeScriptGenerator creates a new TypeScript code generator.
func NewTypeScriptGenerator() *TypeScriptGenerator {
	return &TypeScriptGenerator{}
}

// Language returns the target language.
func (g *TypeScriptGenerator) Language() Language {
	return LanguageTypeScript
}

// FileExtension returns the file extension for generated files.
func (g *TypeScriptGenerator) FileExtension() string {
	return ".ts"
}

// Generate produces TypeScript code from a schema.
func (g *TypeScriptGenerator) Generate(w io.Writer, s *schema.Schema, opts Options) error {
	ctx := &tsContext{
		Schema:  s,
		Options: opts,
	}

	tmpl, err := template.New("typescript").Funcs(ctx.funcMap()).Parse(tsTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl.Execute(w, ctx)
}

// tsContext holds context for TypeScript code generation.
type tsContext struct {
	Schema  *schema.Schema
	Options Options
}

func (c *tsContext) funcMap() template.FuncMap {
	return template.FuncMap{
		"tsType":           c.tsType,
		"tsFieldType":      c.tsFieldType,
		"tsEnumType":       c.tsEnumType,
		"tsMessageType":    c.tsMessageType,
		"tsInterfaceType":  c.tsInterfaceType,
		"tsFieldName":      c.tsFieldName,
		"tsEnumValueName":  c.tsEnumValueName,
		"tsWireType":       c.tsWireType,
		"tsWriteField":     c.tsWriteField,
		"tsReadField":      c.tsReadField,
		"comment":          c.tsComment,
		"toCamel":          ToCamelCase,
		"toPascal":         ToPascalCase,
		"toSnake":          ToSnakeCase,
		"generateComments": func() bool { return c.Options.GenerateComments },
		"generateMarshal":  func() bool { return c.Options.GenerateMarshal },
	}
}

func (c *tsContext) tsType(t schema.TypeRef) string {
	return c.tsTypeInternal(t, false)
}

func (c *tsContext) tsFieldType(f *schema.Field) string {
	t := c.tsTypeInternal(f.Type, false)

	// Wrap repeated fields in array
	if f.Repeated {
		if _, isArray := f.Type.(*schema.ArrayType); !isArray {
			t = t + "[]"
		}
	}

	return t
}

func (c *tsContext) tsTypeInternal(t schema.TypeRef, inArray bool) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.tsScalarType(typ.Name)
	case *schema.NamedType:
		name := c.Options.TypePrefix + ToPascalCase(typ.Name) + c.Options.TypeSuffix
		if typ.Package != "" {
			return typ.Package + "." + name
		}
		return name
	case *schema.ArrayType:
		elem := c.tsTypeInternal(typ.Element, true)
		if typ.Size > 0 {
			// Fixed-size arrays become tuples in TypeScript
			return elem + "[]"
		}
		return elem + "[]"
	case *schema.MapType:
		key := c.tsTypeInternal(typ.Key, false)
		val := c.tsTypeInternal(typ.Value, false)
		// TypeScript maps use Record or Map
		if key == "string" {
			return fmt.Sprintf("Record<%s, %s>", key, val)
		}
		return fmt.Sprintf("Map<%s, %s>", key, val)
	case *schema.PointerType:
		elem := c.tsTypeInternal(typ.Element, false)
		return elem + " | null"
	default:
		return "unknown"
	}
}

func (c *tsContext) tsScalarType(name string) string {
	switch name {
	case "bool":
		return "boolean"
	case "int8", "int16", "int32", "int", "uint8", "uint16", "uint32", "uint":
		return "number"
	case "int64", "uint64":
		return "bigint"
	case "float32", "float64":
		return "number"
	case "complex64", "complex128":
		return "{ real: number; imag: number }"
	case "string":
		return "string"
	case "bytes", "byte":
		return "Uint8Array"
	default:
		return name
	}
}

func (c *tsContext) tsEnumType(e *schema.Enum) string {
	return c.Options.TypePrefix + ToPascalCase(e.Name) + c.Options.TypeSuffix
}

func (c *tsContext) tsMessageType(m *schema.Message) string {
	return c.Options.TypePrefix + ToPascalCase(m.Name) + c.Options.TypeSuffix
}

func (c *tsContext) tsInterfaceType(i *schema.Interface) string {
	return c.Options.TypePrefix + ToPascalCase(i.Name) + c.Options.TypeSuffix
}

func (c *tsContext) tsFieldName(f *schema.Field) string {
	return ToCamelCase(f.Name)
}

func (c *tsContext) tsEnumValueName(v *schema.EnumValue) string {
	return ToPascalCase(v.Name)
}

func (c *tsContext) tsComment(text string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		return "/** " + text + " */"
	}
	result := "/**\n"
	for _, line := range lines {
		result += " * " + line + "\n"
	}
	result += " */"
	return result
}

// tsWireType returns the V2 wire type constant for a field type.
// This matches Go's V2 wire format for cross-runtime compatibility.
func (c *tsContext) tsWireType(f *schema.Field) string {
	return c.tsWireTypeForType(f.Type)
}

func (c *tsContext) tsWireTypeForType(t schema.TypeRef) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool", "uint8", "uint16", "uint32", "uint", "uint64":
			return "WireTypeV2.Varint" // Unsigned varint
		case "int8", "int16", "int32", "int", "int64":
			return "WireTypeV2.SVarint" // Signed zigzag varint
		case "float32":
			return "WireTypeV2.Fixed32"
		case "float64":
			return "WireTypeV2.Fixed64"
		case "string", "bytes":
			return "WireTypeV2.Bytes"
		default:
			return "WireTypeV2.Bytes"
		}
	case *schema.NamedType:
		// Named types (enums, messages) - enums are svarint, messages are bytes
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return "WireTypeV2.SVarint"
			}
		}
		return "WireTypeV2.Bytes"
	case *schema.ArrayType, *schema.MapType:
		return "WireTypeV2.Bytes"
	case *schema.PointerType:
		return "WireTypeV2.Bytes" // Nullable fields use bytes with length prefix
	default:
		return "WireTypeV2.Bytes"
	}
}

// tsWriteField generates the code to write a field value.
func (c *tsContext) tsWriteField(f *schema.Field) string {
	fieldName := "msg." + ToCamelCase(f.Name)
	return c.tsWriteValue(f.Type, fieldName, f.Repeated)
}

// tsWriteValueWithWriter generates write code using a custom writer name
func (c *tsContext) tsWriteValueWithWriter(t schema.TypeRef, value string, writerName string) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return fmt.Sprintf("%s.writeBool(%s)", writerName, value)
		case "int8", "int16", "int32", "int":
			return fmt.Sprintf("%s.writeSVarint(%s)", writerName, value)
		case "uint8", "uint16", "uint32", "uint":
			return fmt.Sprintf("%s.writeVarint(%s)", writerName, value)
		case "int64":
			return fmt.Sprintf("%s.writeSVarint64(%s)", writerName, value)
		case "uint64":
			return fmt.Sprintf("%s.writeVarint64(%s)", writerName, value)
		case "float32":
			return fmt.Sprintf("%s.writeFloat32(%s)", writerName, value)
		case "float64":
			return fmt.Sprintf("%s.writeFloat64(%s)", writerName, value)
		case "string":
			return fmt.Sprintf("%s.writeString(%s)", writerName, value)
		case "bytes":
			return fmt.Sprintf("%s.writeLengthPrefixedBytes(%s)", writerName, value)
		default:
			return fmt.Sprintf("%s.writeString(%s)", writerName, value)
		}
	case *schema.NamedType:
		// Check if it's an enum
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return fmt.Sprintf("%s.writeSVarint(%s)", writerName, value)
			}
		}
		// It's a message - encode it
		return fmt.Sprintf("encode%s(%s, %s)", ToPascalCase(typ.Name), writerName, value)
	default:
		return fmt.Sprintf("%s.writeString(JSON.stringify(%s))", writerName, value)
	}
}

func (c *tsContext) tsWriteValue(t schema.TypeRef, value string, repeated bool) string {
	if repeated {
		// For repeated fields, we need to write the array
		elemType := t
		if arr, ok := t.(*schema.ArrayType); ok {
			elemType = arr.Element
		}
		return fmt.Sprintf("writeArray(writer, %s, (w, v) => { %s })", value, c.tsWriteValueWithWriter(elemType, "v", "w"))
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return fmt.Sprintf("writer.writeBool(%s)", value)
		case "int8", "int16", "int32", "int":
			return fmt.Sprintf("writer.writeSVarint(%s)", value)
		case "uint8", "uint16", "uint32", "uint":
			return fmt.Sprintf("writer.writeVarint(%s)", value)
		case "int64":
			return fmt.Sprintf("writer.writeSVarint64(%s)", value)
		case "uint64":
			return fmt.Sprintf("writer.writeVarint64(%s)", value)
		case "float32":
			return fmt.Sprintf("writer.writeFloat32(%s)", value)
		case "float64":
			return fmt.Sprintf("writer.writeFloat64(%s)", value)
		case "string":
			return fmt.Sprintf("writer.writeString(%s)", value)
		case "bytes":
			return fmt.Sprintf("writer.writeLengthPrefixedBytes(%s)", value)
		default:
			return fmt.Sprintf("writer.writeString(%s)", value)
		}
	case *schema.NamedType:
		// Check if it's an enum
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return fmt.Sprintf("writer.writeSVarint(%s)", value)
			}
		}
		// It's a message - encode it
		return fmt.Sprintf("encode%s(writer, %s)", ToPascalCase(typ.Name), value)
	case *schema.ArrayType:
		return c.tsWriteValue(typ.Element, value, true)
	case *schema.MapType:
		keyWrite := c.tsWriteValueWithWriter(typ.Key, "k", "w")
		valWrite := c.tsWriteValueWithWriter(typ.Value, "v", "w")
		return fmt.Sprintf("writeMap(writer, %s, (w, k) => { %s }, (w, v) => { %s })", value, keyWrite, valWrite)
	case *schema.PointerType:
		return fmt.Sprintf("if (%s !== null) { %s }", value, c.tsWriteValue(typ.Element, value, false))
	default:
		return fmt.Sprintf("writer.writeString(JSON.stringify(%s))", value)
	}
}

// tsReadField generates the code to read a field value.
func (c *tsContext) tsReadField(f *schema.Field) string {
	return c.tsReadValue(f.Type, f.Repeated)
}

func (c *tsContext) tsReadValue(t schema.TypeRef, repeated bool) string {
	if repeated {
		elemType := t
		if arr, ok := t.(*schema.ArrayType); ok {
			elemType = arr.Element
		}
		return fmt.Sprintf("readArray(reader, (r) => %s)", c.tsReadValue(elemType, false))
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return "reader.readBool()"
		case "int8", "int16", "int32", "int":
			return "reader.readSVarint()"
		case "uint8", "uint16", "uint32", "uint":
			return "reader.readVarint()"
		case "int64":
			return "reader.readSVarint64()"
		case "uint64":
			return "reader.readVarint64()"
		case "float32":
			return "reader.readFloat32()"
		case "float64":
			return "reader.readFloat64()"
		case "string":
			return "reader.readString()"
		case "bytes":
			return "reader.readLengthPrefixedBytes()"
		default:
			return "reader.readString()"
		}
	case *schema.NamedType:
		// Check if it's an enum
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return "reader.readSVarint()"
			}
		}
		// It's a message
		return fmt.Sprintf("decode%s(reader)", ToPascalCase(typ.Name))
	case *schema.ArrayType:
		return c.tsReadValue(typ.Element, true)
	case *schema.MapType:
		keyRead := c.tsReadValue(typ.Key, false)
		valRead := c.tsReadValue(typ.Value, false)
		return fmt.Sprintf("readMap(reader, (r) => %s, (r) => %s)", keyRead, valRead)
	case *schema.PointerType:
		return c.tsReadValue(typ.Element, false)
	default:
		return "reader.readString()"
	}
}

func init() {
	Register(NewTypeScriptGenerator())
}

const tsTemplate = `// Code generated by cramberry. DO NOT EDIT.
// Source: {{.Schema.Position.Filename}}
{{if generateMarshal}}
import { Writer, Reader, WireTypeV2 } from 'cramberry';

// Helper functions for encoding/decoding
function writeArray<T>(writer: Writer, arr: T[], writeElem: (w: Writer, v: T) => void): void {
  const subWriter = new Writer();
  subWriter.writeVarint(arr.length);
  for (const elem of arr) {
    writeElem(subWriter, elem);
  }
  writer.writeLengthPrefixedBytes(subWriter.bytes());
}

function readArray<T>(reader: Reader, readElem: (r: Reader) => T): T[] {
  const data = reader.readLengthPrefixedBytes();
  const subReader = new Reader(data);
  const len = subReader.readVarint();
  const result: T[] = [];
  for (let i = 0; i < len; i++) {
    result.push(readElem(subReader));
  }
  return result;
}

function writeMap<K, V>(writer: Writer, map: Map<K, V> | Record<string, V>, writeKey: (w: Writer, k: K) => void, writeVal: (w: Writer, v: V) => void): void {
  const subWriter = new Writer();
  const entries = map instanceof Map ? Array.from(map.entries()) : Object.entries(map);
  subWriter.writeVarint(entries.length);
  for (const [k, v] of entries) {
    writeKey(subWriter, k as K);
    writeVal(subWriter, v as V);
  }
  writer.writeLengthPrefixedBytes(subWriter.bytes());
}

function readMap<K, V>(reader: Reader, readKey: (r: Reader) => K, readVal: (r: Reader) => V): Map<K, V> {
  const data = reader.readLengthPrefixedBytes();
  const subReader = new Reader(data);
  const len = subReader.readVarint();
  const result = new Map<K, V>();
  for (let i = 0; i < len; i++) {
    const k = readKey(subReader);
    const v = readVal(subReader);
    result.set(k, v);
  }
  return result;
}
{{end}}
{{$ctx := .}}
{{range $enum := .Schema.Enums}}
{{if generateComments}}{{range $enum.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
export enum {{tsEnumType $enum}} {
{{- range $enum.Values}}
{{if generateComments}}{{range .Comments}}{{if .IsDoc}}  {{comment .Text}}
{{end}}{{end}}{{end -}}
  {{tsEnumValueName .}} = {{.Number}},
{{- end}}
}

{{end}}
{{range $msg := .Schema.Messages}}
{{if generateComments}}{{range $msg.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
export interface {{tsMessageType $msg}} {
{{- range $msg.Fields}}
{{if generateComments}}{{range .Comments}}{{if .IsDoc}}  {{comment .Text}}
{{end}}{{end}}{{end -}}
  {{tsFieldName .}}{{if .Optional}}?{{end}}: {{tsFieldType .}};
{{- end}}
}
{{if generateMarshal}}
/** Encodes a {{tsMessageType $msg}} to the writer using V2 wire format. */
export function encode{{tsMessageType $msg}}(writer: Writer, msg: {{tsMessageType $msg}}): void {
{{range $msg.Fields}}
  // Field {{.Number}}: {{.Name}}
  if (msg.{{tsFieldName .}} !== undefined{{if not .Optional}} && msg.{{tsFieldName .}} !== null{{end}}) {
    writer.writeCompactTag({{.Number}}, {{tsWireType .}});
    {{tsWriteField .}};
  }
{{end -}}
  // End marker
  writer.writeEndMarker();
}

/** Decodes a {{tsMessageType $msg}} from the reader using V2 wire format. */
export function decode{{tsMessageType $msg}}(reader: Reader): {{tsMessageType $msg}} {
  const result: Partial<{{tsMessageType $msg}}> = {};

  while (true) {
    const { fieldNum, wireType } = reader.readCompactTag();
    if (fieldNum === 0) break; // End marker

    switch (fieldNum) {
{{- range $msg.Fields}}
      case {{.Number}}:
        result.{{tsFieldName .}} = {{tsReadField .}};
        break;
{{- end}}
      default:
        reader.skipValueV2(wireType);
    }
  }

  return result as {{tsMessageType $msg}};
}

/** Marshals a {{tsMessageType $msg}} to bytes. */
export function marshal{{tsMessageType $msg}}(msg: {{tsMessageType $msg}}): Uint8Array {
  const writer = new Writer();
  encode{{tsMessageType $msg}}(writer, msg);
  return writer.bytes();
}

/** Unmarshals a {{tsMessageType $msg}} from bytes. */
export function unmarshal{{tsMessageType $msg}}(data: Uint8Array): {{tsMessageType $msg}} {
  const reader = new Reader(data);
  return decode{{tsMessageType $msg}}(reader);
}
{{end}}
{{end}}
{{range $iface := .Schema.Interfaces}}
{{if generateComments}}{{range $iface.Comments}}{{if .IsDoc}}{{comment .Text}}
{{end}}{{end}}{{end -}}
export type {{tsInterfaceType $iface}} = {{range $i, $impl := $iface.Implementations}}{{if $i}} | {{end}}{{$impl.Type.Name}}{{end}};

/** Type ID mapping for {{tsInterfaceType $iface}} */
export const {{tsInterfaceType $iface}}TypeIds = {
{{- range $iface.Implementations}}
  {{.Type.Name}}: {{.TypeID}},
{{- end}}
} as const;

{{end}}
`
