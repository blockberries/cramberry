package extract

import (
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/blockberries/cramberry/pkg/schema"
)

// SchemaBuilder converts collected type information into a Cramberry schema.
type SchemaBuilder struct {
	types      map[string]*TypeInfo
	interfaces map[string]*InterfaceInfo
	enums      map[string]*EnumInfo
	schema     *schema.Schema
	warnings   []string
}

// NewSchemaBuilder creates a new schema builder.
func NewSchemaBuilder(types map[string]*TypeInfo, interfaces map[string]*InterfaceInfo, enums map[string]*EnumInfo) *SchemaBuilder {
	return &SchemaBuilder{
		types:      types,
		interfaces: interfaces,
		enums:      enums,
		warnings:   nil,
	}
}

// Warnings returns any warnings generated during schema building.
func (b *SchemaBuilder) Warnings() []string {
	return b.warnings
}

// addWarning records a warning message.
func (b *SchemaBuilder) addWarning(msg string) {
	b.warnings = append(b.warnings, msg)
}

// Build constructs a schema from the collected types.
func (b *SchemaBuilder) Build(packageName string) (*schema.Schema, error) {
	b.schema = &schema.Schema{
		Package: &schema.Package{
			Name: packageName,
		},
	}

	// Build enums first (they may be referenced by messages)
	b.buildEnums()

	// Build messages
	b.buildMessages()

	// Build interfaces
	b.buildInterfaces()

	return b.schema, nil
}

func (b *SchemaBuilder) buildEnums() {
	// Sort enums by name for deterministic output
	var names []string
	for name := range b.enums {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		enum := b.enums[name]
		schemaEnum := &schema.Enum{
			Name: enum.Name,
		}

		// Add doc comment if present
		if enum.Doc != "" {
			schemaEnum.Comments = []*schema.Comment{
				{Text: enum.Doc, IsDoc: true},
			}
		}

		// Sort values by number
		values := make([]*EnumValueInfo, len(enum.Values))
		copy(values, enum.Values)
		sort.Slice(values, func(i, j int) bool {
			return values[i].Number < values[j].Number
		})

		for _, val := range values {
			enumVal := &schema.EnumValue{
				Name:   val.Name,
				Number: int(val.Number),
			}
			if val.Doc != "" {
				enumVal.Comments = []*schema.Comment{
					{Text: val.Doc, IsDoc: true},
				}
			}
			schemaEnum.Values = append(schemaEnum.Values, enumVal)
		}

		b.schema.Enums = append(b.schema.Enums, schemaEnum)
	}
}

func (b *SchemaBuilder) buildMessages() {
	// Sort types by name for deterministic output
	var names []string
	for name := range b.types {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		typ := b.types[name]
		msg := &schema.Message{
			Name: typ.Name,
		}

		// Add doc comment if present
		if typ.Doc != "" {
			msg.Comments = []*schema.Comment{
				{Text: typ.Doc, IsDoc: true},
			}
		}

		// Sort fields by field number
		fields := make([]*FieldInfo, len(typ.Fields))
		copy(fields, typ.Fields)
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].FieldNum < fields[j].FieldNum
		})

		// Check for field number collisions
		usedFieldNums := make(map[int]string)
		for _, field := range fields {
			if existingField, exists := usedFieldNums[field.FieldNum]; exists {
				b.addWarning("field number collision in type '" + typ.Name +
					"': fields '" + existingField + "' and '" + field.Name +
					"' both have field number " + strconv.Itoa(field.FieldNum))
			}
			usedFieldNums[field.FieldNum] = field.Name
		}

		for _, field := range fields {
			fieldType := b.goTypeToSchemaType(field.GoType)

			// For repeated fields, unwrap the array type and mark as repeated
			repeated := field.Repeated
			if repeated {
				if arr, ok := fieldType.(*schema.ArrayType); ok {
					fieldType = arr.Element
				}
			}

			schemaField := &schema.Field{
				Name:     toSnakeCase(field.Name),
				Number:   field.FieldNum,
				Type:     fieldType,
				Optional: field.Optional,
				Repeated: repeated,
			}

			// Add doc comment if present
			if field.Doc != "" {
				schemaField.Comments = []*schema.Comment{
					{Text: field.Doc, IsDoc: true},
				}
			}

			// Add options from tag
			if field.Tag != nil {
				if field.Tag.Required {
					schemaField.Required = true
				}
				if field.Tag.Deprecated != "" {
					schemaField.Deprecated = true
				}
			}

			msg.Fields = append(msg.Fields, schemaField)
		}

		b.schema.Messages = append(b.schema.Messages, msg)
	}
}

func (b *SchemaBuilder) buildInterfaces() {
	// Sort interfaces by name for deterministic output
	var names []string
	for name := range b.interfaces {
		names = append(names, name)
	}
	sort.Strings(names)

	// Track used type IDs globally across all interfaces to detect collisions
	usedTypeIDs := make(map[int]string) // typeID -> type name that uses it

	for _, name := range names {
		iface := b.interfaces[name]
		schemaIface := &schema.Interface{
			Name: iface.Name,
		}

		// Add doc comment if present
		if iface.Doc != "" {
			schemaIface.Comments = []*schema.Comment{
				{Text: iface.Doc, IsDoc: true},
			}
		}

		// Sort implementations by name for deterministic output
		impls := make([]*TypeInfo, len(iface.Implementations))
		copy(impls, iface.Implementations)
		sort.Slice(impls, func(i, j int) bool {
			return impls[i].Name < impls[j].Name
		})

		// First pass: collect explicitly assigned type IDs
		for _, impl := range impls {
			if impl.TypeID > 0 {
				if existingType, exists := usedTypeIDs[int(impl.TypeID)]; exists {
					b.addWarning(fmt.Sprintf(
						"type ID collision: %s and %s both use type ID %d",
						existingType, impl.Name, impl.TypeID,
					))
				}
				usedTypeIDs[int(impl.TypeID)] = impl.Name
			}
		}

		// Second pass: assign type IDs to implementations
		nextAutoID := 128 // Start auto-assigned IDs at 128
		for _, impl := range impls {
			var typeID int

			if impl.TypeID > 0 {
				// Use explicitly assigned type ID
				typeID = int(impl.TypeID)
			} else {
				// Auto-assign: find next available ID starting at 128
				for usedTypeIDs[nextAutoID] != "" {
					nextAutoID++
				}
				typeID = nextAutoID
				usedTypeIDs[typeID] = impl.Name
				nextAutoID++
			}

			schemaIface.Implementations = append(schemaIface.Implementations, &schema.Implementation{
				Type:   &schema.NamedType{Name: impl.Name},
				TypeID: typeID,
			})
		}

		b.schema.Interfaces = append(b.schema.Interfaces, schemaIface)
	}
}

func (b *SchemaBuilder) goTypeToSchemaType(t types.Type) schema.TypeRef {
	// Handle pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		elemType := b.goTypeToSchemaType(ptr.Elem())
		return &schema.PointerType{Element: elemType}
	}

	// Handle named types
	if named, ok := t.(*types.Named); ok {
		typeName := named.Obj().Name()
		pkgPath := ""
		if named.Obj().Pkg() != nil {
			pkgPath = named.Obj().Pkg().Path()
		}
		qualifiedName := pkgPath + "." + typeName

		// Check if it's an enum we know about
		if _, isEnum := b.enums[qualifiedName]; isEnum {
			return &schema.NamedType{Name: typeName}
		}

		// Check if it's a type we know about
		if _, isType := b.types[qualifiedName]; isType {
			return &schema.NamedType{Name: typeName}
		}

		// Recurse to underlying type for basic type aliases
		return b.goTypeToSchemaType(named.Underlying())
	}

	// Handle basic types
	if basic, ok := t.(*types.Basic); ok {
		return b.basicTypeToSchemaType(basic)
	}

	// Handle slices
	if slice, ok := t.(*types.Slice); ok {
		elemType := b.goTypeToSchemaType(slice.Elem())
		// Check for []byte specially
		if basic, ok := slice.Elem().(*types.Basic); ok && basic.Kind() == types.Byte {
			return &schema.ScalarType{Name: "bytes"}
		}
		return &schema.ArrayType{Element: elemType}
	}

	// Handle arrays
	if array, ok := t.(*types.Array); ok {
		elemType := b.goTypeToSchemaType(array.Elem())
		return &schema.ArrayType{Element: elemType, Size: int(array.Len())}
	}

	// Handle maps
	if mapType, ok := t.(*types.Map); ok {
		keyType := b.goTypeToSchemaType(mapType.Key())
		valueType := b.goTypeToSchemaType(mapType.Elem())
		return &schema.MapType{Key: keyType, Value: valueType}
	}

	// Handle interfaces
	if _, ok := t.(*types.Interface); ok {
		return &schema.NamedType{Name: "any"}
	}

	// Fallback to bytes
	return &schema.ScalarType{Name: "bytes"}
}

func (b *SchemaBuilder) basicTypeToSchemaType(t *types.Basic) schema.TypeRef {
	switch t.Kind() {
	case types.Bool:
		return &schema.ScalarType{Name: "bool"}
	case types.Int8:
		return &schema.ScalarType{Name: "int8"}
	case types.Int16:
		return &schema.ScalarType{Name: "int16"}
	case types.Int:
		// Warn about platform-dependent int type
		b.addWarning("type 'int' is platform-dependent (32 or 64 bits); " +
			"mapped to int32, consider using explicit int32 or int64 for cross-platform compatibility")
		return &schema.ScalarType{Name: "int32"}
	case types.Int32:
		return &schema.ScalarType{Name: "int32"}
	case types.Int64:
		return &schema.ScalarType{Name: "int64"}
	case types.Uint8:
		return &schema.ScalarType{Name: "uint8"}
	case types.Uint16:
		return &schema.ScalarType{Name: "uint16"}
	case types.Uint:
		// Warn about platform-dependent uint type
		b.addWarning("type 'uint' is platform-dependent (32 or 64 bits); " +
			"mapped to uint32, consider using explicit uint32 or uint64 for cross-platform compatibility")
		return &schema.ScalarType{Name: "uint32"}
	case types.Uint32:
		return &schema.ScalarType{Name: "uint32"}
	case types.Uint64:
		return &schema.ScalarType{Name: "uint64"}
	case types.Float32:
		return &schema.ScalarType{Name: "float32"}
	case types.Float64:
		return &schema.ScalarType{Name: "float64"}
	case types.String:
		return &schema.ScalarType{Name: "string"}
	default:
		return &schema.ScalarType{Name: "bytes"}
	}
}

// toSnakeCase converts CamelCase to snake_case.
// It properly handles runs of uppercase letters (e.g., "HTTPServer" -> "http_server").
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Check if it's uppercase
		if r >= 'A' && r <= 'Z' {
			// Add underscore before uppercase if:
			// - Not at the beginning
			// - Previous char was lowercase, OR
			// - Next char exists and is lowercase (end of acronym)
			if i > 0 {
				prev := runes[i-1]
				isLowerPrev := prev >= 'a' && prev <= 'z'
				isUpperNext := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
				if isLowerPrev || isUpperNext {
					result.WriteByte('_')
				}
			}
			// Convert to lowercase
			result.WriteRune(r + 32)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
