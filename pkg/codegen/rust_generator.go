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
		"jsonFieldName":     c.jsonFieldName,
		"jsonEncodeField":   c.jsonEncodeField,
		"jsonDecodeField":   c.jsonDecodeField,
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

func (c *rustContext) rustTypeInternal(t schema.TypeRef, _ bool) string {
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
			return "WireType::Varint" // Unsigned varint
		case "int8", "int16", "int32", "int", "int64":
			return "WireType::SVarint" // Signed zigzag varint
		case "float32":
			return "WireType::Fixed32"
		case "float64":
			return "WireType::Fixed64"
		case "string", "bytes":
			return "WireType::Bytes"
		default:
			return "WireType::Bytes"
		}
	case *schema.NamedType:
		// Named types (enums, messages) - enums are svarint, messages are bytes.
		// Only check local enums when the type has no package qualifier.
		// Cross-package types are assumed to be messages; cross-package enum
		// detection requires access to imported schemas which is not yet supported.
		if typ.Package == "" {
			for _, e := range c.Schema.Enums {
				if e.Name == typ.Name {
					return "WireType::SVarint"
				}
			}
		}
		return "WireType::Bytes"
	case *schema.ArrayType, *schema.MapType:
		return "WireType::Bytes"
	case *schema.PointerType:
		return "WireType::Bytes" // Nullable fields use bytes with length prefix
	default:
		return "WireType::Bytes"
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
		// Check if it's a local enum (no package qualifier)
		if typ.Package == "" {
			for _, e := range c.Schema.Enums {
				if e.Name == typ.Name {
					return fmt.Sprintf("sub_writer.write_svarint(*%s as i32)", value)
				}
			}
		}
		// It's a message (or cross-package enum, treated as message for now)
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
		// Check if it's a local enum (no package qualifier)
		if typ.Package == "" {
			for _, e := range c.Schema.Enums {
				if e.Name == typ.Name {
					return fmt.Sprintf("writer.write_svarint(%s as i32)", value)
				}
			}
		}
		// It's a message (or cross-package enum, treated as message for now)
		return fmt.Sprintf("encode_%s(writer, &%s)", ToSnakeCase(typ.Name), value)
	case *schema.ArrayType:
		return c.rustWriteValue(typ.Element, value, true)
	case *schema.MapType:
		keyWrite := c.rustWriteValueForSubWriter(typ.Key, "k")
		valWrite := c.rustWriteValueForSubWriter(typ.Value, "v")
		// Sort keys for deterministic output. HashMap iteration in Rust uses
		// a randomized hasher; the wire format requires the same canonical
		// order as the Go reflection marshaller.
		return fmt.Sprintf(`{
        use cramberry::CompareKeys;
        let mut __entries: Vec<_> = %s.iter().collect();
        __entries.sort_by(|a, b| a.0.cramberry_cmp(b.0));
        let mut sub_writer = Writer::new();
        sub_writer.write_varint(__entries.len() as u32)?;
        for (k, v) in __entries {
            %s?;
            %s?;
        }
        writer.write_length_prefixed_bytes(sub_writer.as_bytes())
    }`, value, keyWrite, valWrite)
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
		// Check if it's a local enum (no package qualifier)
		if typ.Package == "" {
			for _, e := range c.Schema.Enums {
				if e.Name == typ.Name {
					enumType := c.rustEnumType(e)
					return fmt.Sprintf("%s::from_i32(reader.read_svarint()?).unwrap_or(%s::%s)", enumType, enumType, ToPascalCase(e.Values[0].Name))
				}
			}
		}
		// It's a message (or cross-package enum, treated as message for now)
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

// JSON helper functions

// jsonFieldName returns the snake_case JSON field name.
func (c *rustContext) jsonFieldName(f *schema.Field) string {
	return ToSnakeCase(f.Name)
}

// jsonEncodeField generates JSON encoding code for a field.
func (c *rustContext) jsonEncodeField(f *schema.Field) string {
	fieldName := ToSnakeCase(f.Name)
	jsonName := ToSnakeCase(f.Name)
	isFirst := f.Number == 1

	var code strings.Builder

	// Add comma if not first field
	if !isFirst {
		code.WriteString("    result.push_str(\",\");\n")
	}

	// Write field name
	code.WriteString(fmt.Sprintf("    result.push_str(\"\\\"%s\\\":\");\n", jsonName))

	// Generate value encoding
	valueCode := c.jsonEncodeValue(f.Type, "msg."+fieldName, f.Repeated, f.Optional)
	code.WriteString(valueCode)

	return code.String()
}

// jsonEncodeValue generates code to encode a value to JSON.
func (c *rustContext) jsonEncodeValue(t schema.TypeRef, varName string, repeated bool, optional bool) string {
	if repeated {
		return c.jsonEncodeArray(t, varName)
	}

	if optional {
		// Handle Option<T>
		if _, isPtr := t.(*schema.PointerType); !isPtr {
			inner := c.jsonEncodeValue(t, "v", false, false)
			return fmt.Sprintf("    if let Some(v) = &%s {\n    %s    } else {\n        result.push_str(\"null\");\n    }\n", varName, strings.ReplaceAll(inner, "    ", "        "))
		}
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.jsonEncodeScalar(typ, varName)
	case *schema.NamedType:
		// Check if it's an enum
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return c.jsonEncodeEnum(e, varName)
			}
		}
		// It's a message
		return c.jsonEncodeMessage(varName)
	case *schema.MapType:
		return c.jsonEncodeMap(typ, varName)
	case *schema.PointerType:
		// Handle Box<T>
		inner := c.jsonEncodeValue(typ.Element, "(**v)", false, false)
		return fmt.Sprintf("    if let Some(v) = &%s {\n    %s    } else {\n        result.push_str(\"null\");\n    }\n", varName, strings.ReplaceAll(inner, "    ", "        "))
	default:
		return "    result.push_str(\"null\");\n"
	}
}

// jsonEncodeScalar generates code to encode a scalar value.
func (c *rustContext) jsonEncodeScalar(t *schema.ScalarType, varName string) string {
	switch t.Name {
	case "bool":
		return fmt.Sprintf("    result.push_str(if %s { \"true\" } else { \"false\" });\n", varName)
	case "int8", "int16", "int32":
		return fmt.Sprintf("    result.push_str(\"\\\"\");\n    result.push_str(&%s.to_string());\n    result.push_str(\"\\\"\");\n", varName)
	case "int64", "int":
		return fmt.Sprintf("    result.push_str(\"\\\"\");\n    result.push_str(&cramberry::json::format_i64_to_string(%s));\n    result.push_str(\"\\\"\");\n", varName)
	case "uint8", "uint16", "uint32", "byte":
		return fmt.Sprintf("    result.push_str(\"\\\"\");\n    result.push_str(&%s.to_string());\n    result.push_str(\"\\\"\");\n", varName)
	case "uint64", "uint":
		return fmt.Sprintf("    result.push_str(\"\\\"\");\n    result.push_str(&cramberry::json::format_u64_to_string(%s));\n    result.push_str(\"\\\"\");\n", varName)
	case "float32":
		return fmt.Sprintf("    result.push_str(&cramberry::json::format_f32(%s).map_err(|e| e.to_string())?);\n", varName)
	case "float64":
		return fmt.Sprintf("    result.push_str(&cramberry::json::format_f64(%s).map_err(|e| e.to_string())?);\n", varName)
	case "string":
		return fmt.Sprintf("    result.push_str(&cramberry::json::escape_json_string(&%s));\n", varName)
	case "bytes":
		return fmt.Sprintf("    result.push_str(\"\\\"\");\n    result.push_str(&cramberry::json::encode_base64(&%s));\n    result.push_str(\"\\\"\");\n", varName)
	default:
		return "    result.push_str(\"null\");\n"
	}
}

// jsonEncodeEnum generates code to encode an enum as string name.
func (c *rustContext) jsonEncodeEnum(e *schema.Enum, varName string) string {
	var code strings.Builder
	code.WriteString("    result.push_str(\"\\\"\");\n")
	code.WriteString("    result.push_str(match " + varName + " {\n")
	for _, v := range e.Values {
		enumType := ToPascalCase(e.Name)
		enumValue := ToPascalCase(v.Name)
		code.WriteString(fmt.Sprintf("        %s::%s => \"%s\",\n", enumType, enumValue, v.Name))
	}
	code.WriteString("        _ => \"UNKNOWN\",\n")
	code.WriteString("    });\n")
	code.WriteString("    result.push_str(\"\\\"\");\n")
	return code.String()
}

// jsonEncodeMessage generates code to encode a nested message.
func (c *rustContext) jsonEncodeMessage(varName string) string {
	return fmt.Sprintf("    result.push_str(&%s.to_json().map_err(|e| e.to_string())?);\n", varName)
}

// jsonEncodeArray generates code to encode an array.
func (c *rustContext) jsonEncodeArray(elemType schema.TypeRef, varName string) string {
	var code strings.Builder
	code.WriteString("    result.push_str(\"[\");\n")
	code.WriteString(fmt.Sprintf("    for (i, elem) in %s.iter().enumerate() {\n", varName))
	code.WriteString("        if i > 0 { result.push_str(\",\"); }\n")

	// Generate element encoding
	elemCode := c.jsonEncodeValue(elemType, "(*elem)", false, false)
	code.WriteString(strings.ReplaceAll(elemCode, "    ", "        "))

	code.WriteString("    }\n")
	code.WriteString("    result.push_str(\"]\");\n")
	return code.String()
}

// jsonEncodeMap generates code to encode a map.
func (c *rustContext) jsonEncodeMap(t *schema.MapType, varName string) string {
	var code strings.Builder
	code.WriteString("    {\n")
	code.WriteString("        result.push_str(\"{\");\n")
	code.WriteString(fmt.Sprintf("        let mut keys: Vec<String> = %s.keys().map(|k| k.to_string()).collect();\n", varName))
	code.WriteString("        cramberry::json::sort_map_keys_lexicographic(&mut keys);\n")
	code.WriteString("        for (i, key_str) in keys.iter().enumerate() {\n")
	code.WriteString("            if i > 0 { result.push_str(\",\"); }\n")
	code.WriteString("            result.push_str(&cramberry::json::escape_json_string(key_str));\n")
	code.WriteString("            result.push_str(\":\");\n")

	// Get the actual key and value
	keyType := t.Key.(*schema.ScalarType)
	switch keyType.Name {
	case "string":
		code.WriteString("            let k = key_str.as_str();\n")
	case "int8", "int16", "int32", "int64", "int":
		rustType := c.rustScalarType(keyType.Name)
		code.WriteString(fmt.Sprintf("            let k: %s = key_str.parse().unwrap();\n", rustType))
	case "uint8", "uint16", "uint32", "uint64", "uint", "byte":
		rustType := c.rustScalarType(keyType.Name)
		code.WriteString(fmt.Sprintf("            let k: %s = key_str.parse().unwrap();\n", rustType))
	default:
		code.WriteString("            let k = key_str.as_str();\n")
	}

	code.WriteString(fmt.Sprintf("            let v = %s.get(&k).unwrap();\n", varName))

	// Encode value
	valueCode := c.jsonEncodeValue(t.Value, "(*v)", false, false)
	code.WriteString(strings.ReplaceAll(valueCode, "    ", "            "))

	code.WriteString("        }\n")
	code.WriteString("        result.push_str(\"}\");\n")
	code.WriteString("    }\n")
	return code.String()
}

// jsonDecodeField generates JSON decoding code for a field.
func (c *rustContext) jsonDecodeField(f *schema.Field) string {
	fieldName := ToSnakeCase(f.Name)
	jsonName := ToSnakeCase(f.Name)

	var code strings.Builder
	code.WriteString(fmt.Sprintf("    if let Some(value) = obj.get(\"%s\") {\n", jsonName))

	// Generate decoding based on type
	decodeCode := c.jsonDecodeValue(f.Type, "msg."+fieldName, "value", f.Repeated, f.Optional)
	code.WriteString(decodeCode)

	code.WriteString("    }\n")
	return code.String()
}

// jsonDecodeValue generates code to decode a JSON value.
func (c *rustContext) jsonDecodeValue(t schema.TypeRef, targetVar string, sourceVar string, repeated bool, optional bool) string {
	if repeated {
		return c.jsonDecodeArray(t, targetVar, sourceVar)
	}

	if optional {
		if _, isPtr := t.(*schema.PointerType); !isPtr {
			// Handle Option<T>
			inner := c.jsonDecodeValue(t, "v", sourceVar, false, false)
			return fmt.Sprintf("        if %s.is_null() {\n            %s = None;\n        } else {\n            let mut v: %s = Default::default();\n        %s            %s = Some(v);\n        }\n",
				sourceVar, targetVar, c.rustType(t), strings.ReplaceAll(inner, "        ", "            "), targetVar)
		}
	}

	switch typ := t.(type) {
	case *schema.ScalarType:
		return c.jsonDecodeScalar(typ, targetVar, sourceVar)
	case *schema.NamedType:
		// Check if it's an enum
		for _, e := range c.Schema.Enums {
			if e.Name == typ.Name {
				return c.jsonDecodeEnum(e, targetVar, sourceVar)
			}
		}
		// It's a message
		return c.jsonDecodeMessage(typ, targetVar, sourceVar)
	case *schema.MapType:
		return c.jsonDecodeMap(typ, targetVar, sourceVar)
	case *schema.PointerType:
		// Handle Box<T>
		inner := c.jsonDecodeValue(typ.Element, "(*v)", sourceVar, false, false)
		return fmt.Sprintf("        if %s.is_null() {\n            %s = None;\n        } else {\n            let v = Box::new(Default::default());\n        %s            %s = Some(v);\n        }\n",
			sourceVar, targetVar, strings.ReplaceAll(inner, "        ", "            "), targetVar)
	default:
		return "        // Unsupported type\n"
	}
}

// jsonDecodeScalar generates code to decode a scalar JSON value.
func (c *rustContext) jsonDecodeScalar(t *schema.ScalarType, targetVar string, sourceVar string) string {
	switch t.Name {
	case "bool":
		return fmt.Sprintf("        %s = %s.as_bool().ok_or_else(|| \"expected boolean\".to_string())?;\n", targetVar, sourceVar)
	case "int8", "int16":
		return fmt.Sprintf("        %s = cramberry::json::parse_i32_from_json(%s)? as %s;\n", targetVar, sourceVar, t.Name)
	case "int32":
		return fmt.Sprintf("        %s = cramberry::json::parse_i32_from_json(%s)?;\n", targetVar, sourceVar)
	case "int64", "int":
		return fmt.Sprintf("        %s = cramberry::json::parse_i64_from_json(%s)?;\n", targetVar, sourceVar)
	case "uint8", "uint16", "byte":
		return fmt.Sprintf("        %s = cramberry::json::parse_u32_from_json(%s)? as %s;\n", targetVar, sourceVar, c.rustScalarType(t.Name))
	case "uint32":
		return fmt.Sprintf("        %s = cramberry::json::parse_u32_from_json(%s)?;\n", targetVar, sourceVar)
	case "uint64", "uint":
		return fmt.Sprintf("        %s = cramberry::json::parse_u64_from_json(%s)?;\n", targetVar, sourceVar)
	case "float32", "float64":
		return fmt.Sprintf("        %s = %s.as_f64().ok_or_else(|| \"expected number\".to_string())? as %s;\n", targetVar, sourceVar, c.rustScalarType(t.Name))
	case "string":
		return fmt.Sprintf("        %s = %s.as_str().ok_or_else(|| \"expected string\".to_string())?.to_string();\n", targetVar, sourceVar)
	case "bytes":
		return fmt.Sprintf("        let s = %s.as_str().ok_or_else(|| \"expected string\".to_string())?;\n        %s = cramberry::json::decode_base64(s).map_err(|e| e.to_string())?;\n", sourceVar, targetVar)
	default:
		return "        // Unsupported scalar type\n"
	}
}

// jsonDecodeEnum generates code to decode an enum from string name.
func (c *rustContext) jsonDecodeEnum(e *schema.Enum, targetVar string, sourceVar string) string {
	var code strings.Builder
	enumType := ToPascalCase(e.Name)

	code.WriteString(fmt.Sprintf("        let str_val = %s.as_str().ok_or_else(|| \"expected string for enum\".to_string())?;\n", sourceVar))
	code.WriteString("        " + targetVar + " = match str_val {\n")
	for _, v := range e.Values {
		code.WriteString(fmt.Sprintf("            \"%s\" => %s::%s,\n", v.Name, enumType, ToPascalCase(v.Name)))
	}
	code.WriteString("            _ => return Err(format!(\"unknown enum value: {}\", str_val)),\n")
	code.WriteString("        };\n")

	return code.String()
}

// jsonDecodeMessage generates code to decode a nested message.
func (c *rustContext) jsonDecodeMessage(t *schema.NamedType, targetVar string, sourceVar string) string {
	msgType := ToSnakeCase(t.Name)
	return fmt.Sprintf("        %s = from_json_%s(&%s.to_string()).map_err(|e| e.to_string())?;\n", targetVar, msgType, sourceVar)
}

// jsonDecodeArray generates code to decode an array.
func (c *rustContext) jsonDecodeArray(elemType schema.TypeRef, targetVar string, sourceVar string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("    let arr = %s.as_array().ok_or_else(|| \"expected array\".to_string())?;\n", sourceVar))
	code.WriteString(fmt.Sprintf("    %s = Vec::new();\n", targetVar))
	code.WriteString("    for elem in arr {\n")
	code.WriteString(fmt.Sprintf("        let mut decoded: %s = Default::default();\n", c.rustType(elemType)))

	// Decode element
	elemCode := c.jsonDecodeValue(elemType, "decoded", "elem", false, false)
	code.WriteString(elemCode)

	code.WriteString(fmt.Sprintf("        %s.push(decoded);\n", targetVar))
	code.WriteString("    }\n")

	return code.String()
}

// jsonDecodeMap generates code to decode a map.
func (c *rustContext) jsonDecodeMap(t *schema.MapType, targetVar string, sourceVar string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("    let map_obj = %s.as_object().ok_or_else(|| \"expected object\".to_string())?;\n", sourceVar))
	code.WriteString(fmt.Sprintf("    %s = std::collections::HashMap::new();\n", targetVar))
	code.WriteString("    for (key_str, val) in map_obj {\n")

	// Convert string key to actual key type
	keyType := t.Key.(*schema.ScalarType)
	switch keyType.Name {
	case "string":
		code.WriteString("        let k = key_str.to_string();\n")
	case "int8", "int16", "int32", "int64", "int":
		rustType := c.rustScalarType(keyType.Name)
		code.WriteString(fmt.Sprintf("        let k: %s = key_str.parse().map_err(|e| format!(\"invalid key: {{}}\", e))?;\n", rustType))
	case "uint8", "uint16", "uint32", "uint64", "uint", "byte":
		rustType := c.rustScalarType(keyType.Name)
		code.WriteString(fmt.Sprintf("        let k: %s = key_str.parse().map_err(|e| format!(\"invalid key: {{}}\", e))?;\n", rustType))
	default:
		code.WriteString("        let k = key_str.to_string();\n")
	}

	code.WriteString(fmt.Sprintf("        let mut v: %s = Default::default();\n", c.rustType(t.Value)))

	// Decode value
	valueCode := c.jsonDecodeValue(t.Value, "v", "val", false, false)
	code.WriteString(valueCode)

	code.WriteString("        " + targetVar + ".insert(k, v);\n")
	code.WriteString("    }\n")

	return code.String()
}

func init() {
	Register(NewRustGenerator())
}

const rustTemplate = `// Code generated by cramberry. DO NOT EDIT.
// Source: {{.Schema.Position.Filename}}

{{if hasSerde}}use serde::{Deserialize, Serialize};
{{end}}{{if generateMarshal}}use cramberry::{Reader, Result, WireType, Writer};
use serde_json;
use std::collections::HashMap;
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

    pub fn to_string(&self) -> &'static str {
        match self {
{{- range $enum.Values}}
            Self::{{rustEnumValueName .}} => "{{.Name}}",
{{- end}}
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
    writer.write_tag({{.Number}}, {{rustWireType .}})?;
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
        let (field_num, wire_type) = reader.read_tag()?;
        if field_num == 0 { break; } // End marker

        match field_num {
{{- range $msg.Fields}}
            {{.Number}} => {{rustFieldName .}} = {{rustReadField .}},
{{- end}}
            _ => reader.skip_value(wire_type)?,
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

/// Encodes a {{rustMessageType $msg}} to deterministic JSON format.
pub fn to_json_{{toSnake $msg.Name}}(msg: &{{rustMessageType $msg}}) -> Result<String, String> {
    let mut result = String::new();
    result.push_str("{");
{{range $msg.Fields}}
{{jsonEncodeField .}}
{{- end}}
    result.push_str("}");
    Ok(result)
}

/// Decodes a {{rustMessageType $msg}} from JSON format.
pub fn from_json_{{toSnake $msg.Name}}(json: &str) -> Result<{{rustMessageType $msg}}, String> {
    let parsed: serde_json::Value = serde_json::from_str(json)
        .map_err(|e| format!("JSON parse error: {}", e))?;

    let obj = parsed.as_object()
        .ok_or_else(|| "expected JSON object".to_string())?;

    // Check for unknown fields (strict mode)
    let allowed_fields: std::collections::HashSet<&str> = [
{{- range $msg.Fields}}
        "{{jsonFieldName .}}",
{{- end}}
    ].iter().copied().collect();

    for key in obj.keys() {
        if !allowed_fields.contains(key.as_str()) {
            return Err(format!("unknown field: {}", key));
        }
    }

    let mut msg: {{rustMessageType $msg}} = Default::default();

    // Decode fields
{{range $msg.Fields}}
{{jsonDecodeField .}}
{{- end}}

    Ok(msg)
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
