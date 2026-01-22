package schema

import (
	"fmt"
	"sort"
)

// ValidationError represents a schema validation error.
type ValidationError struct {
	Position Position
	Message  string
	Severity Severity
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s: %s",
		e.Position.Filename, e.Position.Line, e.Position.Column,
		e.Severity, e.Message)
}

// Severity indicates the severity of a validation error.
type Severity int

const (
	// SeverityError is a fatal error that prevents code generation.
	SeverityError Severity = iota
	// SeverityWarning is a non-fatal issue.
	SeverityWarning
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return "unknown"
	}
}

// Validator validates schema definitions.
type Validator struct {
	schema  *Schema
	errors  []ValidationError
	types   map[string]TypeDef   // All defined types
	imports map[string]*Schema   // Imported schemas by alias/path
}

// TypeDef represents a type definition (message, enum, or interface).
type TypeDef struct {
	Name     string
	Kind     TypeDefKind
	Position Position
	TypeID   int // For messages with @TypeID annotation
}

// TypeDefKind indicates the kind of type definition.
type TypeDefKind int

const (
	TypeDefMessage TypeDefKind = iota
	TypeDefEnum
	TypeDefInterface
)

func (k TypeDefKind) String() string {
	switch k {
	case TypeDefMessage:
		return "message"
	case TypeDefEnum:
		return "enum"
	case TypeDefInterface:
		return "interface"
	default:
		return "unknown"
	}
}

// NewValidator creates a new validator for the given schema.
func NewValidator(schema *Schema) *Validator {
	return &Validator{
		schema:  schema,
		types:   make(map[string]TypeDef),
		imports: make(map[string]*Schema),
	}
}

// AddImport registers an imported schema.
func (v *Validator) AddImport(path string, alias string, schema *Schema) {
	key := alias
	if key == "" {
		key = path
	}
	v.imports[key] = schema
}

// Validate performs validation and returns any errors.
func (v *Validator) Validate() []ValidationError {
	v.errors = nil

	// First pass: collect all type definitions
	v.collectTypes()

	// Validate messages
	for _, msg := range v.schema.Messages {
		v.validateMessage(msg)
	}

	// Validate enums
	for _, enum := range v.schema.Enums {
		v.validateEnum(enum)
	}

	// Validate interfaces
	for _, iface := range v.schema.Interfaces {
		v.validateInterface(iface)
	}

	// Sort errors by position
	sort.Slice(v.errors, func(i, j int) bool {
		if v.errors[i].Position.Line != v.errors[j].Position.Line {
			return v.errors[i].Position.Line < v.errors[j].Position.Line
		}
		return v.errors[i].Position.Column < v.errors[j].Position.Column
	})

	return v.errors
}

// collectTypes collects all type definitions for reference checking.
func (v *Validator) collectTypes() {
	// Collect messages
	for _, msg := range v.schema.Messages {
		if existing, ok := v.types[msg.Name]; ok {
			v.addError(msg.Position, "duplicate type name %q (previously defined at %d:%d)",
				msg.Name, existing.Position.Line, existing.Position.Column)
		} else {
			v.types[msg.Name] = TypeDef{
				Name:     msg.Name,
				Kind:     TypeDefMessage,
				Position: msg.Position,
				TypeID:   msg.TypeID,
			}
		}
	}

	// Collect enums
	for _, enum := range v.schema.Enums {
		if existing, ok := v.types[enum.Name]; ok {
			v.addError(enum.Position, "duplicate type name %q (previously defined at %d:%d)",
				enum.Name, existing.Position.Line, existing.Position.Column)
		} else {
			v.types[enum.Name] = TypeDef{
				Name:     enum.Name,
				Kind:     TypeDefEnum,
				Position: enum.Position,
			}
		}
	}

	// Collect interfaces
	for _, iface := range v.schema.Interfaces {
		if existing, ok := v.types[iface.Name]; ok {
			v.addError(iface.Position, "duplicate type name %q (previously defined at %d:%d)",
				iface.Name, existing.Position.Line, existing.Position.Column)
		} else {
			v.types[iface.Name] = TypeDef{
				Name:     iface.Name,
				Kind:     TypeDefInterface,
				Position: iface.Position,
			}
		}
	}
}

// validateMessage validates a message definition.
func (v *Validator) validateMessage(msg *Message) {
	// Check for duplicate field numbers
	fieldNumbers := make(map[int]string) // number -> field name
	fieldNames := make(map[string]bool)

	for _, field := range msg.Fields {
		// Check field number is valid
		if field.Number <= 0 {
			v.addError(field.Position, "field number must be positive, got %d", field.Number)
		}

		// Check field number range
		if field.Number > 536870911 { // 2^29 - 1, max protobuf field number
			v.addError(field.Position, "field number %d exceeds maximum (536870911)", field.Number)
		}

		// Reserved field numbers (19000-19999 in protobuf, we'll use same range)
		if field.Number >= 19000 && field.Number <= 19999 {
			v.addWarning(field.Position, "field number %d is in reserved range (19000-19999)", field.Number)
		}

		// Check for duplicate field numbers
		if existing, ok := fieldNumbers[field.Number]; ok {
			v.addError(field.Position, "duplicate field number %d (also used by field %q)",
				field.Number, existing)
		} else {
			fieldNumbers[field.Number] = field.Name
		}

		// Check for duplicate field names
		if fieldNames[field.Name] {
			v.addError(field.Position, "duplicate field name %q", field.Name)
		} else {
			fieldNames[field.Name] = true
		}

		// Validate field type
		v.validateTypeRef(field.Type, msg.Name, field.Name)

		// Check modifier combinations
		modifierCount := 0
		if field.Required {
			modifierCount++
		}
		if field.Optional {
			modifierCount++
		}
		if modifierCount > 1 {
			v.addError(field.Position, "field cannot be both required and optional")
		}

		// Validate map key type
		if mt, ok := field.Type.(*MapType); ok {
			v.validateMapKeyType(mt.Key, msg.Name, field.Name)
		}
	}

	// Check TypeID if specified
	if msg.TypeID < 0 {
		v.addError(msg.Position, "type ID must be non-negative, got %d", msg.TypeID)
	}
}

// validateEnum validates an enum definition.
func (v *Validator) validateEnum(enum *Enum) {
	valueNumbers := make(map[int]string) // number -> value name
	valueNames := make(map[string]bool)

	// Check for zero value
	hasZero := false
	for _, val := range enum.Values {
		if val.Number == 0 {
			hasZero = true
			break
		}
	}
	if !hasZero && len(enum.Values) > 0 {
		v.addWarning(enum.Position, "enum %q should have a zero value (conventionally for unknown/default)", enum.Name)
	}

	for _, val := range enum.Values {
		// Check for negative values
		if val.Number < 0 {
			v.addError(val.Position, "enum value number must be non-negative, got %d", val.Number)
		}

		// Check for duplicate numbers
		if existing, ok := valueNumbers[val.Number]; ok {
			v.addError(val.Position, "duplicate enum value number %d (also used by %q)",
				val.Number, existing)
		} else {
			valueNumbers[val.Number] = val.Name
		}

		// Check for duplicate names
		if valueNames[val.Name] {
			v.addError(val.Position, "duplicate enum value name %q", val.Name)
		} else {
			valueNames[val.Name] = true
		}
	}
}

// validateInterface validates an interface definition.
func (v *Validator) validateInterface(iface *Interface) {
	typeIDs := make(map[int]string) // typeID -> type name

	for _, impl := range iface.Implementations {
		// Check type ID is valid
		if impl.TypeID <= 0 {
			v.addError(impl.Position, "type ID must be positive, got %d", impl.TypeID)
		}

		// Check for duplicate type IDs
		if existing, ok := typeIDs[impl.TypeID]; ok {
			v.addError(impl.Position, "duplicate type ID %d (also used by %q)",
				impl.TypeID, existing)
		} else {
			typeIDs[impl.TypeID] = impl.Type.Name
		}

		// Validate that the referenced type exists and is a message
		typeName := impl.Type.Name
		if impl.Type.Package != "" {
			// Qualified type - check imported schema
			importedSchema, ok := v.imports[impl.Type.Package]
			if !ok {
				v.addError(impl.Position, "unknown package %q", impl.Type.Package)
				continue
			}
			// Check type exists in imported schema
			found := false
			for _, msg := range importedSchema.Messages {
				if msg.Name == impl.Type.Name {
					found = true
					break
				}
			}
			if !found {
				v.addError(impl.Position, "type %q not found in package %q",
					impl.Type.Name, impl.Type.Package)
			}
		} else {
			// Local type
			typeDef, ok := v.types[typeName]
			if !ok {
				v.addError(impl.Position, "undefined type %q", typeName)
			} else if typeDef.Kind != TypeDefMessage {
				v.addError(impl.Position, "interface implementation must reference a message, not %s %q",
					typeDef.Kind, typeName)
			}
		}
	}
}

// validateTypeRef validates a type reference.
func (v *Validator) validateTypeRef(typeRef TypeRef, msgName, fieldName string) {
	switch t := typeRef.(type) {
	case *ScalarType:
		// Scalar types are always valid (checked during parsing)

	case *NamedType:
		if t.Package != "" {
			// Qualified type - check imported schema
			if _, ok := v.imports[t.Package]; !ok {
				v.addError(t.Position, "unknown package %q in field %s.%s",
					t.Package, msgName, fieldName)
			}
		} else {
			// Local type - check it exists
			if _, ok := v.types[t.Name]; !ok {
				v.addError(t.Position, "undefined type %q in field %s.%s",
					t.Name, msgName, fieldName)
			}
		}

	case *ArrayType:
		v.validateTypeRef(t.Element, msgName, fieldName)
		if t.Size < 0 {
			v.addError(t.Position, "array size must be non-negative")
		}

	case *MapType:
		v.validateTypeRef(t.Key, msgName, fieldName)
		v.validateTypeRef(t.Value, msgName, fieldName)

	case *PointerType:
		v.validateTypeRef(t.Element, msgName, fieldName)
	}
}

// validateMapKeyType ensures map key types are valid (must be comparable).
func (v *Validator) validateMapKeyType(keyType TypeRef, msgName, fieldName string) {
	switch t := keyType.(type) {
	case *ScalarType:
		// Most scalar types are valid keys
		switch t.Name {
		case "bytes", "float32", "float64", "complex64", "complex128":
			v.addError(t.Position, "map key type %q is not comparable in field %s.%s",
				t.Name, msgName, fieldName)
		}

	case *NamedType:
		// Named types can only be enums for keys
		if t.Package == "" {
			if typeDef, ok := v.types[t.Name]; ok && typeDef.Kind != TypeDefEnum {
				v.addError(t.Position, "map key type must be scalar or enum, not %s in field %s.%s",
					typeDef.Kind, msgName, fieldName)
			}
		}

	case *ArrayType, *MapType, *PointerType:
		v.addError(keyType.Pos(), "map key type must be scalar or enum in field %s.%s",
			msgName, fieldName)
	}
}

func (v *Validator) addError(pos Position, format string, args ...any) {
	v.errors = append(v.errors, ValidationError{
		Position: pos,
		Message:  fmt.Sprintf(format, args...),
		Severity: SeverityError,
	})
}

func (v *Validator) addWarning(pos Position, format string, args ...any) {
	v.errors = append(v.errors, ValidationError{
		Position: pos,
		Message:  fmt.Sprintf(format, args...),
		Severity: SeverityWarning,
	})
}

// HasErrors returns true if there are any errors (not warnings).
func (v *Validator) HasErrors() bool {
	for _, err := range v.errors {
		if err.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only the error-severity issues.
func (v *Validator) Errors() []ValidationError {
	var errors []ValidationError
	for _, err := range v.errors {
		if err.Severity == SeverityError {
			errors = append(errors, err)
		}
	}
	return errors
}

// Warnings returns only the warning-severity issues.
func (v *Validator) Warnings() []ValidationError {
	var warnings []ValidationError
	for _, err := range v.errors {
		if err.Severity == SeverityWarning {
			warnings = append(warnings, err)
		}
	}
	return warnings
}

// Validate is a convenience function that validates a schema.
func Validate(schema *Schema) []ValidationError {
	validator := NewValidator(schema)
	return validator.Validate()
}

// ValidateWithImports validates a schema with imported schemas.
func ValidateWithImports(schema *Schema, imports map[string]*Schema) []ValidationError {
	validator := NewValidator(schema)
	for path, s := range imports {
		validator.AddImport(path, "", s)
	}
	return validator.Validate()
}
