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

// resolveNamedEnum returns the schema.Enum for a NamedType if it refers to
// an enum (rather than a message), looking in both the local schema and any
// imported schemas. Without the cross-package lookup, an enum imported from
// another package falls back to WireBytes (the default for messages) and
// produces a malformed wire format.
func (c *tsContext) resolveNamedEnum(typ *schema.NamedType) (*schema.Enum, bool) {
	if typ.Package == "" {
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return e, true
			}
		}
		return nil, false
	}
	if c.Options.ImportedSchemas != nil {
		if imported, ok := c.Options.ImportedSchemas[typ.Package]; ok && imported != nil {
			for _, e := range imported.Enums {
				if e.Name == typ.Name {
					return e, true
				}
			}
		}
	}
	return nil, false
}

func (c *tsContext) isNamedEnum(typ *schema.NamedType) bool {
	_, ok := c.resolveNamedEnum(typ)
	return ok
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
		"jsonFieldName":    c.jsonFieldName,
		"jsonEncodeField":  c.jsonEncodeField,
		"jsonDecodeField":  c.jsonDecodeField,
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

func (c *tsContext) tsTypeInternal(t schema.TypeRef, _ bool) string {
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
		// Always emit `Map<K, V>`. The runtime decoder produces a `Map`
		// (readMap returns `new Map()`), so typing the field as `Record`
		// caused a tsc strict-mode error on every map field. The encoder
		// helper writeMap accepts both Map and Record at runtime, so users
		// who want the `Record` ergonomics can still pass one in.
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
			return "WireType.Varint" // Unsigned varint
		case "int8", "int16", "int32", "int", "int64":
			return "WireType.SVarint" // Signed zigzag varint
		case "float32":
			return "WireType.Fixed32"
		case "float64":
			return "WireType.Fixed64"
		case "string", "bytes":
			return "WireType.Bytes"
		default:
			return "WireType.Bytes"
		}
	case *schema.NamedType:
		if c.isNamedEnum(typ) {
			return "WireType.SVarint"
		}
		return "WireType.Bytes"
	case *schema.ArrayType, *schema.MapType:
		return "WireType.Bytes"
	case *schema.PointerType:
		return "WireType.Bytes" // Nullable fields use bytes with length prefix
	default:
		return "WireType.Bytes"
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
		if c.isNamedEnum(typ) {
			return fmt.Sprintf("%s.writeSVarint(%s)", writerName, value)
		}
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
		if c.isNamedEnum(typ) {
			return fmt.Sprintf("writer.writeSVarint(%s)", value)
		}
		// Nested message bodies are end-marker-terminated, so we wrap the
		// encoded body in a length-prefixed payload. Without this, a future
		// schema that doesn't recognize the field can't skip it cleanly:
		// the SkipValue(WireType.Bytes) path reads the first body byte as a
		// length and corrupts subsequent decoding.
		return fmt.Sprintf("{ const __sub = new Writer(); encode%s(__sub, %s); writer.writeLengthPrefixedBytes(__sub.bytes()); }", ToPascalCase(typ.Name), value)
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

// tsReadField generates the code to read a field value at the per-field
// switch in a generated decoder. Field-level NamedType (message) values
// are length-prefixed on the wire (see encodeStruct's field-wrapping
// layer), so we read the length first and decode the body via a sub-reader.
//
// Inside readArray / readMap callbacks the SAME schema NamedType is
// emitted INLINE (no length prefix per element) — the encoder mirrors
// that asymmetry, and a wrong choice here is what produced the
// "BufferUnderflowError in readArray callback" earlier. Use
// tsReadValueInline for the callback path.
func (c *tsContext) tsReadField(f *schema.Field) string {
	if f.Repeated {
		// Repeated fields use the slice-callback variant for elements.
		elemType := f.Type
		if arr, ok := f.Type.(*schema.ArrayType); ok {
			elemType = arr.Element
		}
		return fmt.Sprintf("readArray(reader, (r) => %s)", c.tsReadValueInline("r", elemType))
	}
	if mt, ok := f.Type.(*schema.MapType); ok {
		keyRead := c.tsReadValueInline("r", mt.Key)
		valRead := c.tsReadValueInline("r", mt.Value)
		return fmt.Sprintf("readMap(reader, (r) => %s, (r) => %s)", keyRead, valRead)
	}
	// Walk through PointerType: an optional/pointer field has the same
	// on-wire shape as the underlying type (the optionality is conveyed by
	// presence/absence of the field tag, not by the body).
	t := f.Type
	for {
		ptr, ok := t.(*schema.PointerType)
		if !ok {
			break
		}
		t = ptr.Element
	}
	if typ, ok := t.(*schema.NamedType); ok {
		if c.isNamedEnum(typ) {
			return "reader.readSVarint()"
		}
		// Field-level: length-prefixed.
		return fmt.Sprintf("(() => { const __data = reader.readLengthPrefixedBytes(); return decode%s(new Reader(__data)); })()", ToPascalCase(typ.Name))
	}
	return c.tsReadValueInline("reader", t)
}

// tsReadValueInline emits a read expression for a value using the named
// reader, treating composite values as INLINE (no length prefix). Used
// inside readArray / readMap callbacks: the surrounding helper has already
// taken care of the outer length prefix, and individual elements / map
// entries are written inline by the matching encoder helpers.
func (c *tsContext) tsReadValueInline(readerName string, t schema.TypeRef) string {
	switch typ := t.(type) {
	case *schema.ScalarType:
		switch typ.Name {
		case "bool":
			return fmt.Sprintf("%s.readBool()", readerName)
		case "int8", "int16", "int32", "int":
			return fmt.Sprintf("%s.readSVarint()", readerName)
		case "uint8", "uint16", "uint32", "uint":
			return fmt.Sprintf("%s.readVarint()", readerName)
		case "int64":
			return fmt.Sprintf("%s.readSVarint64()", readerName)
		case "uint64":
			return fmt.Sprintf("%s.readVarint64()", readerName)
		case "float32":
			return fmt.Sprintf("%s.readFloat32()", readerName)
		case "float64":
			return fmt.Sprintf("%s.readFloat64()", readerName)
		case "string":
			return fmt.Sprintf("%s.readString()", readerName)
		case "bytes":
			return fmt.Sprintf("%s.readLengthPrefixedBytes()", readerName)
		default:
			return fmt.Sprintf("%s.readString()", readerName)
		}
	case *schema.NamedType:
		if c.isNamedEnum(typ) {
			return fmt.Sprintf("%s.readSVarint()", readerName)
		}
		// Inline: the element has no length prefix of its own.
		return fmt.Sprintf("decode%s(%s)", ToPascalCase(typ.Name), readerName)
	case *schema.ArrayType:
		return fmt.Sprintf("readArray(%s, (r) => %s)", readerName, c.tsReadValueInline("r", typ.Element))
	case *schema.MapType:
		keyRead := c.tsReadValueInline("r", typ.Key)
		valRead := c.tsReadValueInline("r", typ.Value)
		return fmt.Sprintf("readMap(%s, (r) => %s, (r) => %s)", readerName, keyRead, valRead)
	case *schema.PointerType:
		return c.tsReadValueInline(readerName, typ.Element)
	default:
		return fmt.Sprintf("%s.readString()", readerName)
	}
}

// JSON helper functions

// jsonFieldName returns the snake_case JSON field name.
func (c *tsContext) jsonFieldName(f *schema.Field) string {
	return ToSnakeCase(f.Name)
}

// jsonEncodeField generates JSON encoding code for a field. idx is the
// field's position within the message (NOT its tag number) — only the
// position-zero field skips the leading comma. Using f.Number here would
// break for any schema whose first declared field doesn't happen to have
// tag 1.
func (c *tsContext) jsonEncodeField(idx int, f *schema.Field) string {
	fieldName := ToCamelCase(f.Name)
	jsonName := ToSnakeCase(f.Name)

	var code strings.Builder

	// Add comma if not first field
	if idx > 0 {
		code.WriteString("  result += ',';\n")
	}

	// Write field name
	code.WriteString(fmt.Sprintf("  result += '\"%s\":';\n", jsonName))

	// Generate value encoding
	valueCode := c.jsonEncodeValue(f.Type, "msg."+fieldName, f.Repeated)
	code.WriteString(valueCode)

	return code.String()
}

// jsonEncodeValue generates code to encode a value to JSON.
func (c *tsContext) jsonEncodeValue(t schema.TypeRef, varName string, repeated bool) string {
	if repeated {
		return c.jsonEncodeArray(t, varName)
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.jsonEncodeScalar(typ, varName)
	case *schema.NamedType:
		if c.isNamedEnum(typ) {
			return c.jsonEncodeEnum(varName)
		}
		// Dispatch to the generated toJSON_<TypeName> function so we honor
		// cramberry's deterministic JSON rules (sorted keys, integer-as-string,
		// etc.) instead of falling back to runtime JSON.stringify.
		return c.jsonEncodeMessage(typ, varName)
	case *schema.MapType:
		return c.jsonEncodeMap(typ, varName)
	case *schema.PointerType:
		// Handle optional pointer
		inner := c.jsonEncodeValue(typ.Element, varName, false)
		return fmt.Sprintf("  if (%s != null) {\n  %s  } else {\n    result += 'null';\n  }\n", varName, strings.ReplaceAll(inner, "  ", "    "))
	default:
		return "  result += 'null';\n"
	}
}

// jsonEncodeScalar generates code to encode a scalar value.
func (c *tsContext) jsonEncodeScalar(t *schema.ScalarType, varName string) string {
	switch t.Name {
	case "bool":
		return fmt.Sprintf("  result += %s ? 'true' : 'false';\n", varName)
	case "int8", "int16", "int32", "uint8", "uint16", "uint32":
		return fmt.Sprintf("  result += '\"' + formatNumberToString(%s) + '\"';\n", varName)
	case "int64", "uint64":
		return fmt.Sprintf("  result += '\"' + formatBigIntToString(%s) + '\"';\n", varName)
	case "float32":
		return fmt.Sprintf("  result += formatFloat32(%s);\n", varName)
	case "float64":
		return fmt.Sprintf("  result += formatFloat64(%s);\n", varName)
	case "string":
		return fmt.Sprintf("  result += escapeJSONString(%s);\n", varName)
	case "bytes":
		return fmt.Sprintf("  result += '\"' + encodeBase64(%s) + '\"';\n", varName)
	default:
		return "  result += 'null';\n"
	}
}

// jsonEncodeEnum generates code to encode an enum as string name.
func (c *tsContext) jsonEncodeEnum(varName string) string {
	return fmt.Sprintf("  result += '\"' + %s.toString() + '\"';\n", varName)
}

// jsonEncodeMessage generates code to encode a nested message by dispatching
// to the typed toJSON_<TypeName> helper. This preserves cramberry's
// deterministic JSON output (sorted keys, integer-as-string, fixed float
// format) instead of falling back to the runtime's default JSON.stringify.
func (c *tsContext) jsonEncodeMessage(typ *schema.NamedType, varName string) string {
	return fmt.Sprintf("  result += toJSON_%s(%s);\n", ToPascalCase(typ.Name), varName)
}

// jsonEncodeArray generates code to encode an array.
func (c *tsContext) jsonEncodeArray(elemType schema.TypeRef, varName string) string {
	var code strings.Builder
	code.WriteString("  result += '[';\n")
	code.WriteString(fmt.Sprintf("  for (let i = 0; i < %s.length; i++) {\n", varName))
	code.WriteString("    if (i > 0) result += ',';\n")
	code.WriteString(fmt.Sprintf("    const elem = %s[i];\n", varName))

	// Generate element encoding
	elemCode := c.jsonEncodeValue(elemType, "elem", false)
	code.WriteString(strings.ReplaceAll(elemCode, "  ", "    "))

	code.WriteString("  }\n")
	code.WriteString("  result += ']';\n")
	return code.String()
}

// jsonEncodeMap generates code to encode a map.
func (c *tsContext) jsonEncodeMap(t *schema.MapType, varName string) string {
	var code strings.Builder
	code.WriteString("  {\n")
	code.WriteString("    result += '{';\n")
	code.WriteString(fmt.Sprintf("    const keys = Array.from(%s.keys());\n", varName))

	// Convert keys to strings for sorting
	keyType := t.Key.(*schema.ScalarType)
	switch keyType.Name {
	case "string":
		code.WriteString("    const sortedKeys = sortMapKeysLexicographic(keys);\n")
	case "int8", "int16", "int32", "uint8", "uint16", "uint32":
		code.WriteString("    const sortedKeys = sortMapKeysLexicographic(keys.map(k => formatNumberToString(k)));\n")
	case "int64", "uint64":
		code.WriteString("    const sortedKeys = sortMapKeysLexicographic(keys.map(k => formatBigIntToString(k)));\n")
	default:
		code.WriteString("    const sortedKeys = sortMapKeysLexicographic(keys.map(k => String(k)));\n")
	}

	code.WriteString("    for (let i = 0; i < sortedKeys.length; i++) {\n")
	code.WriteString("      if (i > 0) result += ',';\n")
	code.WriteString("      result += escapeJSONString(sortedKeys[i]) + ':';\n")

	// Get the actual key and value
	switch keyType.Name {
	case "string":
		code.WriteString("      const k = sortedKeys[i];\n")
	case "int8", "int16", "int32", "uint8", "uint16", "uint32":
		code.WriteString("      const k = parseNumberFromJSON(sortedKeys[i]);\n")
	case "int64", "uint64":
		code.WriteString("      const k = parseBigIntFromJSON(sortedKeys[i]);\n")
	default:
		code.WriteString("      const k = sortedKeys[i];\n")
	}

	code.WriteString(fmt.Sprintf("      const v = %s.get(k)!;\n", varName))

	// Encode value
	valueCode := c.jsonEncodeValue(t.Value, "v", false)
	code.WriteString(strings.ReplaceAll(valueCode, "  ", "      "))

	code.WriteString("    }\n")
	code.WriteString("    result += '}';\n")
	code.WriteString("  }\n")
	return code.String()
}

// jsonDecodeField generates JSON decoding code for a field.
func (c *tsContext) jsonDecodeField(f *schema.Field) string {
	fieldName := ToCamelCase(f.Name)
	jsonName := ToSnakeCase(f.Name)

	var code strings.Builder
	code.WriteString(fmt.Sprintf("  if ('%s' in obj) {\n", jsonName))
	code.WriteString(fmt.Sprintf("    const value = obj['%s'];\n", jsonName))

	// Generate decoding based on type
	decodeCode := c.jsonDecodeValue(f.Type, "msg."+fieldName, "value", f.Repeated)
	code.WriteString(decodeCode)

	code.WriteString("  }\n")
	return code.String()
}

// jsonDecodeValue generates code to decode a JSON value.
func (c *tsContext) jsonDecodeValue(t schema.TypeRef, targetVar string, sourceVar string, repeated bool) string {
	if repeated {
		return c.jsonDecodeArray(t, targetVar, sourceVar)
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.jsonDecodeScalar(typ, targetVar, sourceVar)
	case *schema.NamedType:
		if e, ok := c.resolveNamedEnum(typ); ok {
			return c.jsonDecodeEnum(e, targetVar, sourceVar)
		}
		return c.jsonDecodeMessage(typ, targetVar, sourceVar)
	case *schema.MapType:
		return c.jsonDecodeMap(typ, targetVar, sourceVar)
	case *schema.PointerType:
		return c.jsonDecodePointer(typ, targetVar, sourceVar)
	default:
		return "    // Unsupported type\n"
	}
}

// jsonDecodeScalar generates code to decode a scalar JSON value.
func (c *tsContext) jsonDecodeScalar(t *schema.ScalarType, targetVar string, sourceVar string) string {
	switch t.Name {
	case "bool":
		return fmt.Sprintf("    %s = Boolean(%s);\n", targetVar, sourceVar)
	case "int8", "int16", "int32", "uint8", "uint16", "uint32":
		return fmt.Sprintf("    %s = parseNumberFromJSON(%s);\n", targetVar, sourceVar)
	case "int64", "uint64":
		return fmt.Sprintf("    %s = parseBigIntFromJSON(%s);\n", targetVar, sourceVar)
	case "float32", "float64":
		return fmt.Sprintf("    %s = Number(%s);\n", targetVar, sourceVar)
	case "string":
		return fmt.Sprintf("    %s = String(%s);\n", targetVar, sourceVar)
	case "bytes":
		return fmt.Sprintf("    %s = decodeBase64(String(%s));\n", targetVar, sourceVar)
	default:
		return "    // Unsupported scalar type\n"
	}
}

// jsonDecodeEnum generates code to decode an enum from string name.
func (c *tsContext) jsonDecodeEnum(e *schema.Enum, targetVar string, sourceVar string) string {
	var code strings.Builder
	enumType := ToPascalCase(e.Name)

	code.WriteString(fmt.Sprintf("    const strVal = String(%s);\n", sourceVar))
	code.WriteString("    switch (strVal) {\n")
	for _, v := range e.Values {
		code.WriteString(fmt.Sprintf("      case '%s':\n", v.Name))
		code.WriteString(fmt.Sprintf("        %s = %s.%s;\n", targetVar, enumType, ToPascalCase(v.Name)))
		code.WriteString("        break;\n")
	}
	code.WriteString("      default:\n")
	code.WriteString("        throw new Error(`unknown enum value: ${strVal}`);\n")
	code.WriteString("    }\n")

	return code.String()
}

// jsonDecodeMessage generates code to decode a nested message.
func (c *tsContext) jsonDecodeMessage(t *schema.NamedType, targetVar string, sourceVar string) string {
	msgType := ToPascalCase(t.Name)
	return fmt.Sprintf("    %s = fromJSON_%s(JSON.stringify(%s));\n", targetVar, msgType, sourceVar)
}

// jsonDecodePointer generates code to decode an optional pointer.
func (c *tsContext) jsonDecodePointer(t *schema.PointerType, targetVar string, sourceVar string) string {
	var code strings.Builder
	code.WriteString(fmt.Sprintf("    if (%s != null) {\n", sourceVar))

	innerCode := c.jsonDecodeValue(t.Element, targetVar, sourceVar, false)
	code.WriteString(strings.ReplaceAll(innerCode, "    ", "      "))

	code.WriteString("    } else {\n")
	code.WriteString(fmt.Sprintf("      %s = null;\n", targetVar))
	code.WriteString("    }\n")

	return code.String()
}

// jsonDecodeArray generates code to decode an array.
func (c *tsContext) jsonDecodeArray(elemType schema.TypeRef, targetVar string, sourceVar string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("    if (!Array.isArray(%s)) throw new Error('expected array');\n", sourceVar))
	code.WriteString(fmt.Sprintf("    %s = [];\n", targetVar))
	code.WriteString(fmt.Sprintf("    for (const elem of %s) {\n", sourceVar))
	code.WriteString(fmt.Sprintf("      let decoded: %s;\n", c.tsType(elemType)))

	// Decode element
	elemCode := c.jsonDecodeValue(elemType, "decoded", "elem", false)
	code.WriteString(strings.ReplaceAll(elemCode, "    ", "      "))

	code.WriteString(fmt.Sprintf("      %s.push(decoded);\n", targetVar))
	code.WriteString("    }\n")

	return code.String()
}

// jsonDecodeMap generates code to decode a map.
func (c *tsContext) jsonDecodeMap(t *schema.MapType, targetVar string, sourceVar string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("    if (typeof %s !== 'object' || %s === null) throw new Error('expected object');\n", sourceVar, sourceVar))
	code.WriteString(fmt.Sprintf("    %s = new Map();\n", targetVar))
	code.WriteString(fmt.Sprintf("    for (const [keyStr, val] of Object.entries(%s)) {\n", sourceVar))

	// Convert string key to actual key type
	keyType := t.Key.(*schema.ScalarType)
	switch keyType.Name {
	case "string":
		code.WriteString("      const k = keyStr;\n")
	case "int8", "int16", "int32", "uint8", "uint16", "uint32":
		code.WriteString("      const k = parseNumberFromJSON(keyStr);\n")
	case "int64", "uint64":
		code.WriteString("      const k = parseBigIntFromJSON(keyStr);\n")
	default:
		code.WriteString("      const k = keyStr;\n")
	}

	code.WriteString(fmt.Sprintf("      let v: %s;\n", c.tsType(t.Value)))

	// Decode value
	valueCode := c.jsonDecodeValue(t.Value, "v", "val", false)
	code.WriteString(strings.ReplaceAll(valueCode, "    ", "      "))

	code.WriteString(fmt.Sprintf("      %s.set(k, v);\n", targetVar))
	code.WriteString("    }\n")

	return code.String()
}

func init() {
	Register(NewTypeScriptGenerator())
}

const tsTemplate = `// Code generated by cramberry. DO NOT EDIT.
// Source: {{.Schema.Position.Filename}}
{{if generateMarshal}}
import {
  Writer,
  Reader,
  WireType,
  formatBigIntToString,
  formatNumberToString,
  formatFloat32,
  formatFloat64,
  encodeBase64,
  decodeBase64,
  parseBigIntFromJSON,
  parseNumberFromJSON,
  sortMapKeysLexicographic,
  escapeJSONString,
} from 'cramberry';

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

// compareMapKeys provides the canonical map-key ordering used on the wire.
// String keys compare by UTF-8 byte order (matching Go's sort.Strings).
// Numeric keys compare numerically with NaN sorted last and -0 == +0.
// BigInt keys compare numerically. Boolean keys order false before true.
const __mapKeyEncoder = new TextEncoder();
function compareMapKeys(a: unknown, b: unknown): number {
  if (typeof a === 'string' && typeof b === 'string') {
    const ab = __mapKeyEncoder.encode(a);
    const bb = __mapKeyEncoder.encode(b);
    const n = Math.min(ab.length, bb.length);
    for (let i = 0; i < n; i++) if (ab[i] !== bb[i]) return ab[i] - bb[i];
    return ab.length - bb.length;
  }
  if (typeof a === 'number' && typeof b === 'number') {
    const aNaN = Number.isNaN(a), bNaN = Number.isNaN(b);
    if (aNaN && bNaN) return 0; // payloads are not observable in JS
    if (aNaN) return 1;
    if (bNaN) return -1;
    return a < b ? -1 : a > b ? 1 : 0; // -0 and +0 compare equal here
  }
  if (typeof a === 'bigint' && typeof b === 'bigint') {
    return a < b ? -1 : a > b ? 1 : 0;
  }
  if (typeof a === 'boolean' && typeof b === 'boolean') {
    return a === b ? 0 : a ? 1 : -1;
  }
  // Mixed types: fall back to string comparison so the ordering is at least
  // total, even if the schema validator should have rejected this.
  return String(a) < String(b) ? -1 : String(a) > String(b) ? 1 : 0;
}

function writeMap<K, V>(writer: Writer, map: Map<K, V> | Record<string, V>, writeKey: (w: Writer, k: K) => void, writeVal: (w: Writer, v: V) => void): void {
  const subWriter = new Writer();
  const entries = map instanceof Map ? Array.from(map.entries()) : Object.entries(map);
  // Sort by key for deterministic output. Map iteration order is
  // implementation-defined and varies per insertion order; the wire format
  // requires a canonical order matching the Go reflection marshaller.
  entries.sort((a, b) => compareMapKeys(a[0], b[0]));
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
    writer.writeTag({{.Number}}, {{tsWireType .}});
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
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 0) break; // End marker

    switch (fieldNumber) {
{{- range $msg.Fields}}
      case {{.Number}}:
        result.{{tsFieldName .}} = {{tsReadField .}};
        break;
{{- end}}
      default:
        reader.skipValue(wireType);
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

/** Encodes a {{tsMessageType $msg}} to deterministic JSON format. */
export function toJSON_{{tsMessageType $msg}}(msg: {{tsMessageType $msg}}): string {
  let result = '{';
{{range $i, $f := $msg.Fields}}
{{jsonEncodeField $i $f}}
{{- end}}
  result += '}';
  return result;
}

/** Decodes a {{tsMessageType $msg}} from JSON format. */
export function fromJSON_{{tsMessageType $msg}}(json: string): {{tsMessageType $msg}} {
  const obj = JSON.parse(json);
  if (typeof obj !== 'object' || obj === null) {
    throw new Error('expected JSON object');
  }

  // Check for unknown fields (strict mode)
  const allowedFields = new Set([
{{- range $msg.Fields}}
    '{{jsonFieldName .}}',
{{- end}}
  ]);
  for (const key of Object.keys(obj)) {
    if (!allowedFields.has(key)) {
      throw new Error('unknown field: ' + key);
    }
  }

  const msg: Partial<{{tsMessageType $msg}}> = {};

  // Decode fields
{{range $msg.Fields}}
{{jsonDecodeField .}}
{{- end}}

  return msg as {{tsMessageType $msg}};
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
