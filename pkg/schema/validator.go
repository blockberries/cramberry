package schema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/blockberries/cramberry/internal/wire"
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
	types   map[string]TypeDef // All defined types
	imports map[string]*Schema // Imported schemas by alias/path
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

	// Detect message-by-value recursion. The encoder is fine with cycles
	// through pointer fields (those terminate naturally), but a struct that
	// contains itself by value is an infinite type. Go rejects it at compile
	// time with "invalid recursive type"; without this check the cramberry
	// validator silently passes a schema that produces non-compiling output.
	v.checkMessageRecursion()

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
	// Collect messages. Two messages can't share a name AND can't share an
	// explicit @TypeID — a duplicate ID across messages would route
	// polymorphic decoding to whichever was registered last and silently
	// corrupt data on the other side of the wire.
	type typeIDOwner struct {
		name string
		pos  Position
	}
	typeIDOwners := make(map[int]typeIDOwner)
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
		if msg.TypeID > 0 {
			if owner, dup := typeIDOwners[msg.TypeID]; dup {
				v.addError(msg.Position,
					"duplicate type ID %d (also used by message %q at %d:%d)",
					msg.TypeID, owner.name, owner.pos.Line, owner.pos.Column)
			} else {
				typeIDOwners[msg.TypeID] = typeIDOwner{name: msg.Name, pos: msg.Position}
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

		// Check field number range. The single source of truth is
		// wire.MaxFieldNumber; the runtime decoder enforces the same
		// bound, and the TS port mirrors it as MAX_FIELD_NUMBER.
		if field.Number > wire.MaxFieldNumber {
			v.addError(field.Position, "field number %d exceeds maximum (%d)", field.Number, wire.MaxFieldNumber)
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

		// Check modifier combinations. The parser accepts stacked modifiers
		// like `required repeated optional T x = 1;` because each modifier
		// is independent on the AST. Mutual exclusion is enforced here.
		var modifiers []string
		if field.Required {
			modifiers = append(modifiers, "required")
		}
		if field.Optional {
			modifiers = append(modifiers, "optional")
		}
		if field.Repeated {
			modifiers = append(modifiers, "repeated")
		}
		if len(modifiers) > 1 {
			v.addError(field.Position, "field has mutually exclusive modifiers: %v", modifiers)
		}

		// Validate map key type
		if mt, ok := field.Type.(*MapType); ok {
			v.validateMapKeyType(mt.Key, msg.Name, field.Name)
		}
	}

	// Check TypeID if specified.
	//
	// Type IDs 1-127 are reserved by the runtime (1-63 builtin, 64-127
	// stdlib; see pkg/cramberry/types.go's TypeID* constants). User
	// schemas must pick IDs ≥ 128 — picking a reserved ID otherwise
	// causes silent registry collisions across the runtimes.
	if msg.TypeID < 0 {
		v.addError(msg.Position, "type ID must be non-negative, got %d", msg.TypeID)
	} else if msg.TypeID > 0 && msg.TypeID < 128 {
		v.addError(msg.Position, "type ID %d is in the reserved range (1-127); user-defined messages must use type IDs ≥ 128", msg.TypeID)
	} else if msg.TypeID > wire.MaxFieldNumber {
		// Type IDs share the wire-format field-number budget — values
		// past wire.MaxFieldNumber (2^29-1) push the encoded varint
		// past the documented spec and risk overflow on 32-bit
		// runtime types.
		v.addError(msg.Position, "type ID %d exceeds maximum (%d)", msg.TypeID, wire.MaxFieldNumber)
	}
}

// validateEnum validates an enum definition.
func (v *Validator) validateEnum(enum *Enum) {
	valueNumbers := make(map[int]string) // number -> value name
	valueNames := make(map[string]bool)

	// Reject empty enums. Without at least one variant, the codegen-
	// emitted Go produces an empty `const ()` block (compiles but is
	// unusable), Rust gets an enum with no variants (also useless),
	// and the wire format has no valid value. The cross-language
	// "must have a zero value" check below silently skipped empty
	// enums because its `len > 0` guard ran first.
	if len(enum.Values) == 0 {
		v.addError(enum.Position, "enum %q must have at least one value", enum.Name)
		return
	}

	// Check for zero value
	hasZero := false
	for _, val := range enum.Values {
		if val.Number == 0 {
			hasZero = true
			break
		}
	}
	if !hasZero && len(enum.Values) > 0 {
		// Without a 0-valued variant, the three runtimes disagree on
		// the default: Rust's Default derive picks the first declared
		// variant, Go's `var x EnumType` is 0 (not a valid variant),
		// TS leaves the field at 0. A common 0 variant — typically
		// UNKNOWN — keeps decode-default behavior consistent across
		// languages.
		v.addError(enum.Position, "enum %q must have a zero value (e.g. UNKNOWN = 0) for cross-language default consistency", enum.Name)
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
		// Check type ID is valid. IDs 1-127 are reserved by the runtime;
		// see validateMessage above for the same rationale.
		if impl.TypeID <= 0 {
			v.addError(impl.Position, "type ID must be positive, got %d", impl.TypeID)
		} else if impl.TypeID < 128 {
			v.addError(impl.Position, "type ID %d is in the reserved range (1-127); user-defined implementations must use type IDs ≥ 128", impl.TypeID)
		} else if impl.TypeID > wire.MaxFieldNumber {
			v.addError(impl.Position, "type ID %d exceeds maximum (%d)", impl.TypeID, wire.MaxFieldNumber)
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
			// Local type - check it exists locally or in same-package imports
			typeDef, ok := v.types[typeName]
			if ok {
				if typeDef.Kind != TypeDefMessage {
					v.addError(impl.Position, "interface implementation must reference a message, not %s %q",
						typeDef.Kind, typeName)
				}
			} else {
				// Check if the type exists in any imported schema from the same package
				found := v.findMessageInSamePackageImports(typeName)
				if !found {
					v.addError(impl.Position, "undefined type %q", typeName)
				}
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
			// Local type - check it exists locally or in same-package imports
			if _, ok := v.types[t.Name]; !ok {
				// Check if the type exists in any imported schema from the same package
				found := v.findTypeInSamePackageImports(t.Name)
				if !found {
					v.addError(t.Position, "undefined type %q in field %s.%s",
						t.Name, msgName, fieldName)
				}
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

// validateMapKeyType ensures map key types are valid (must be comparable
// AND consistently encodable across all three runtimes).
func (v *Validator) validateMapKeyType(keyType TypeRef, msgName, fieldName string) {
	switch t := keyType.(type) {
	case *ScalarType:
		switch t.Name {
		case "bytes", "float32", "float64", "complex64", "complex128":
			v.addError(t.Position, "map key type %q is not comparable in field %s.%s",
				t.Name, msgName, fieldName)
		case "bool":
			// Bool keys are technically comparable, but the codegen
			// JSON path assumes string-or-numeric keys; bool keys
			// produce uncompilable Go. They also offer no value
			// (only two possible keys) and confuse cross-language
			// users. Reject up front.
			v.addError(t.Position, "map key type \"bool\" is not supported in field %s.%s; use a sentinel struct or two named fields instead",
				msgName, fieldName)
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

// findTypeInSamePackageImports checks if a type exists in any imported schema
// that has the same package name as the current schema. This allows unqualified
// references to types from same-package imports.
func (v *Validator) findTypeInSamePackageImports(typeName string) bool {
	if v.schema.Package == nil {
		return false
	}
	currentPkg := v.schema.Package.Name

	for _, importedSchema := range v.imports {
		if importedSchema == nil || importedSchema.Package == nil {
			continue
		}
		// Only check imports from the same package
		if importedSchema.Package.Name != currentPkg {
			continue
		}
		// Check if the type exists in this imported schema
		for _, msg := range importedSchema.Messages {
			if msg.Name == typeName {
				return true
			}
		}
		for _, enum := range importedSchema.Enums {
			if enum.Name == typeName {
				return true
			}
		}
		for _, iface := range importedSchema.Interfaces {
			if iface.Name == typeName {
				return true
			}
		}
	}
	return false
}

// findMessageInSamePackageImports checks if a message exists in any imported schema
// that has the same package name as the current schema. This is used for interface
// implementations which must reference messages.
func (v *Validator) findMessageInSamePackageImports(typeName string) bool {
	if v.schema.Package == nil {
		return false
	}
	currentPkg := v.schema.Package.Name

	for _, importedSchema := range v.imports {
		if importedSchema == nil || importedSchema.Package == nil {
			continue
		}
		// Only check imports from the same package
		if importedSchema.Package.Name != currentPkg {
			continue
		}
		// Check if the message exists in this imported schema
		for _, msg := range importedSchema.Messages {
			if msg.Name == typeName {
				return true
			}
		}
	}
	return false
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

// checkMessageRecursion reports any message that contains itself by value
// (transitively) through a non-pointer, non-array, non-map field. Pointer,
// repeated, and map fields all introduce indirection, so the resulting Go
// type is finite-sized and acceptable.
func (v *Validator) checkMessageRecursion() {
	// Build adjacency: message → set of message types it contains by value.
	adj := make(map[string][]string, len(v.schema.Messages))
	for _, msg := range v.schema.Messages {
		for _, f := range msg.Fields {
			// Repeated fields are encoded as length-prefixed arrays — finite size.
			if f.Repeated {
				continue
			}
			t := f.Type
			// Pointer fields terminate the cycle: *T has fixed size.
			if _, isPtr := t.(*PointerType); isPtr {
				continue
			}
			// Map values are length-prefixed; map fields don't propagate recursion.
			if _, isMap := t.(*MapType); isMap {
				continue
			}
			// Array of fixed size with NamedType element would propagate, but
			// the schema language doesn't currently support fixed-size arrays
			// of messages, so we don't model them here.
			named, ok := t.(*NamedType)
			if !ok || named.Package != "" {
				// Cross-package types live in a different schema; assume safe.
				continue
			}
			adj[msg.Name] = append(adj[msg.Name], named.Name)
		}
	}

	// DFS for cycles. Track three states: unvisited, on-stack (gray), done (black).
	const (
		gray  = 1
		black = 2
	)
	state := make(map[string]int, len(v.schema.Messages))
	msgByName := make(map[string]*Message, len(v.schema.Messages))
	for _, m := range v.schema.Messages {
		msgByName[m.Name] = m
	}

	var dfs func(name string, path []string) bool
	dfs = func(name string, path []string) bool {
		switch state[name] {
		case gray:
			// Cycle: report at the first message in the path that touched this node.
			cycle := append(path, name)
			if msg, ok := msgByName[name]; ok {
				v.addError(msg.Position,
					"message %q is recursive by value (cycle: %s); use a pointer field to break the cycle",
					name, joinCycle(cycle))
			}
			return true
		case black:
			return false
		}
		state[name] = gray
		for _, child := range adj[name] {
			if dfs(child, append(path, name)) {
				state[name] = black
				return true
			}
		}
		state[name] = black
		return false
	}

	// Iterate messages in declaration order so error reports are stable.
	for _, msg := range v.schema.Messages {
		if state[msg.Name] == 0 {
			dfs(msg.Name, nil)
		}
	}
}

func joinCycle(path []string) string {
	if len(path) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(path[0])
	for _, p := range path[1:] {
		out.WriteString(" → " + p)
	}
	return out.String()
}
