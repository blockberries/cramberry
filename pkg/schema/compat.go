package schema

import (
	"fmt"
)

// BreakingChangeType indicates the kind of breaking change detected.
type BreakingChangeType int

const (
	// FieldNumberReused indicates a field number was reused with a different type.
	FieldNumberReused BreakingChangeType = iota
	// FieldTypeChanged indicates a field's type was changed.
	FieldTypeChanged
	// RequiredFieldAdded indicates a required field was added.
	RequiredFieldAdded
	// RequiredFieldRemoved indicates a required field was removed.
	RequiredFieldRemoved
	// EnumValueReused indicates an enum value number was reused with a different name.
	EnumValueReused
	// EnumValueRemoved indicates an enum value was removed.
	EnumValueRemoved
	// MessageRemoved indicates a message was removed.
	MessageRemoved
	// EnumRemoved indicates an enum was removed.
	EnumRemoved
	// InterfaceTypeRemoved indicates an interface type binding was removed.
	InterfaceTypeRemoved
	// InterfaceTypeIDReused indicates an interface type ID was reused.
	InterfaceTypeIDReused
)

// String returns a human-readable description of the breaking change type.
func (t BreakingChangeType) String() string {
	switch t {
	case FieldNumberReused:
		return "field number reused"
	case FieldTypeChanged:
		return "field type changed"
	case RequiredFieldAdded:
		return "required field added"
	case RequiredFieldRemoved:
		return "required field removed"
	case EnumValueReused:
		return "enum value number reused"
	case EnumValueRemoved:
		return "enum value removed"
	case MessageRemoved:
		return "message removed"
	case EnumRemoved:
		return "enum removed"
	case InterfaceTypeRemoved:
		return "interface type removed"
	case InterfaceTypeIDReused:
		return "interface type ID reused"
	default:
		return "unknown breaking change"
	}
}

// BreakingChange represents an incompatible schema change.
type BreakingChange struct {
	// Type is the kind of breaking change.
	Type BreakingChangeType
	// Message describes the specific change.
	Message string
	// Location identifies where in the schema the change occurred.
	Location string
}

// Error returns the breaking change as an error string.
func (b BreakingChange) Error() string {
	if b.Location != "" {
		return fmt.Sprintf("%s: %s at %s", b.Type, b.Message, b.Location)
	}
	return fmt.Sprintf("%s: %s", b.Type, b.Message)
}

// CompatibilityReport contains the results of a schema compatibility check.
type CompatibilityReport struct {
	// Breaking contains all breaking changes detected.
	Breaking []BreakingChange
	// Warnings contains non-breaking but notable changes.
	Warnings []string
}

// IsCompatible returns true if no breaking changes were detected.
func (r *CompatibilityReport) IsCompatible() bool {
	return len(r.Breaking) == 0
}

// CheckCompatibility compares two schemas and returns a compatibility report.
// The 'old' schema is the existing/deployed version, and 'new' is the proposed version.
func CheckCompatibility(oldSchema, newSchema *Schema) *CompatibilityReport {
	report := &CompatibilityReport{}

	// Build message lookup maps from slices
	oldMessages := make(map[string]*Message)
	for _, m := range oldSchema.Messages {
		oldMessages[m.Name] = m
	}
	newMessages := make(map[string]*Message)
	for _, m := range newSchema.Messages {
		newMessages[m.Name] = m
	}

	// Check messages
	for name, oldMsg := range oldMessages {
		if newMsg, exists := newMessages[name]; exists {
			checkMessageCompat(oldMsg, newMsg, report)
		} else {
			report.Breaking = append(report.Breaking, BreakingChange{
				Type:     MessageRemoved,
				Message:  fmt.Sprintf("message %q was removed", name),
				Location: name,
			})
		}
	}

	// Check for new required fields in existing messages
	for name, newMsg := range newMessages {
		if oldMsg, exists := oldMessages[name]; exists {
			checkNewRequiredFields(oldMsg, newMsg, report)
		}
	}

	// Build enum lookup maps from slices
	oldEnums := make(map[string]*Enum)
	for _, e := range oldSchema.Enums {
		oldEnums[e.Name] = e
	}
	newEnums := make(map[string]*Enum)
	for _, e := range newSchema.Enums {
		newEnums[e.Name] = e
	}

	// Check enums
	for name, oldEnum := range oldEnums {
		if newEnum, exists := newEnums[name]; exists {
			checkEnumCompat(oldEnum, newEnum, report)
		} else {
			report.Breaking = append(report.Breaking, BreakingChange{
				Type:     EnumRemoved,
				Message:  fmt.Sprintf("enum %q was removed", name),
				Location: name,
			})
		}
	}

	// Build interface lookup maps from slices
	oldInterfaces := make(map[string]*Interface)
	for _, i := range oldSchema.Interfaces {
		oldInterfaces[i.Name] = i
	}
	newInterfaces := make(map[string]*Interface)
	for _, i := range newSchema.Interfaces {
		newInterfaces[i.Name] = i
	}

	// Check interfaces
	for name, oldIface := range oldInterfaces {
		if newIface, exists := newInterfaces[name]; exists {
			checkInterfaceCompat(oldIface, newIface, report)
		} else {
			// Removing an interface is usually breaking
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("interface %q was removed", name))
		}
	}

	return report
}

// checkMessageCompat checks for breaking changes between two message versions.
func checkMessageCompat(oldMsg, newMsg *Message, report *CompatibilityReport) {
	// Build maps for faster lookup
	oldFields := make(map[int]*Field)
	for _, f := range oldMsg.Fields {
		oldFields[f.Number] = f
	}

	newFields := make(map[int]*Field)
	for _, f := range newMsg.Fields {
		newFields[f.Number] = f
	}

	// Check each field in old schema
	for num, oldF := range oldFields {
		if newF, exists := newFields[num]; exists {
			// Field number exists in both - check type compatibility
			if !areTypesCompatible(oldF.Type, newF.Type) {
				report.Breaking = append(report.Breaking, BreakingChange{
					Type: FieldTypeChanged,
					Message: fmt.Sprintf("field %q type changed from %s to %s",
						oldF.Name, oldF.Type.String(), newF.Type.String()),
					Location: fmt.Sprintf("%s.%s", oldMsg.Name, oldF.Name),
				})
			}
		} else {
			// Field was removed
			if oldF.Required {
				report.Breaking = append(report.Breaking, BreakingChange{
					Type:     RequiredFieldRemoved,
					Message:  fmt.Sprintf("required field %q was removed", oldF.Name),
					Location: fmt.Sprintf("%s.%s", oldMsg.Name, oldF.Name),
				})
			}
			// Non-required field removal is a warning, not breaking
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("field %s.%s was removed", oldMsg.Name, oldF.Name))
		}
	}
}

// checkNewRequiredFields checks for new required fields added to a message.
func checkNewRequiredFields(oldMsg, newMsg *Message, report *CompatibilityReport) {
	oldFieldNums := make(map[int]bool)
	for _, f := range oldMsg.Fields {
		oldFieldNums[f.Number] = true
	}

	for _, newF := range newMsg.Fields {
		if !oldFieldNums[newF.Number] && newF.Required {
			report.Breaking = append(report.Breaking, BreakingChange{
				Type:     RequiredFieldAdded,
				Message:  fmt.Sprintf("required field %q was added", newF.Name),
				Location: fmt.Sprintf("%s.%s", newMsg.Name, newF.Name),
			})
		}
	}
}

// checkEnumCompat checks for breaking changes between two enum versions.
func checkEnumCompat(oldEnum, newEnum *Enum, report *CompatibilityReport) {
	// Build maps
	oldValues := make(map[int]*EnumValue)
	for _, v := range oldEnum.Values {
		oldValues[v.Number] = v
	}

	newValues := make(map[int]*EnumValue)
	for _, v := range newEnum.Values {
		newValues[v.Number] = v
	}

	// Check each value in old enum
	for num, oldV := range oldValues {
		if newV, exists := newValues[num]; exists {
			// Value number exists - check if name changed (might indicate reuse)
			if oldV.Name != newV.Name {
				report.Breaking = append(report.Breaking, BreakingChange{
					Type: EnumValueReused,
					Message: fmt.Sprintf("enum value %d changed from %q to %q",
						num, oldV.Name, newV.Name),
					Location: fmt.Sprintf("%s.%s", oldEnum.Name, oldV.Name),
				})
			}
		} else {
			// Value was removed
			report.Breaking = append(report.Breaking, BreakingChange{
				Type:     EnumValueRemoved,
				Message:  fmt.Sprintf("enum value %q (%d) was removed", oldV.Name, num),
				Location: fmt.Sprintf("%s.%s", oldEnum.Name, oldV.Name),
			})
		}
	}
}

// checkInterfaceCompat checks for breaking changes between two interface versions.
func checkInterfaceCompat(oldIface, newIface *Interface, report *CompatibilityReport) {
	// Build maps of type name -> type ID
	oldTypes := make(map[string]int)
	oldTypeIDs := make(map[int]string)
	for _, impl := range oldIface.Implementations {
		name := impl.Type.Name
		oldTypes[name] = impl.TypeID
		oldTypeIDs[impl.TypeID] = name
	}

	newTypes := make(map[string]int)
	newTypeIDs := make(map[int]string)
	for _, impl := range newIface.Implementations {
		name := impl.Type.Name
		newTypes[name] = impl.TypeID
		newTypeIDs[impl.TypeID] = name
	}

	// Check each type in old interface
	for name, oldID := range oldTypes {
		if newID, exists := newTypes[name]; exists {
			// Type exists - check if ID changed
			if oldID != newID {
				report.Breaking = append(report.Breaking, BreakingChange{
					Type: InterfaceTypeIDReused,
					Message: fmt.Sprintf("type %q ID changed from %d to %d",
						name, oldID, newID),
					Location: fmt.Sprintf("%s.%s", oldIface.Name, name),
				})
			}
		} else {
			// Type was removed from interface
			report.Breaking = append(report.Breaking, BreakingChange{
				Type:     InterfaceTypeRemoved,
				Message:  fmt.Sprintf("type %q was removed from interface", name),
				Location: fmt.Sprintf("%s.%s", oldIface.Name, name),
			})
		}
	}

	// Check for ID reuse with different types
	for id, oldName := range oldTypeIDs {
		if newName, exists := newTypeIDs[id]; exists && oldName != newName {
			// Already reported as type ID changed, but also note the reuse
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("interface %s: type ID %d changed from %q to %q",
					oldIface.Name, id, oldName, newName))
		}
	}
}

// areTypesCompatible checks if two types are compatible for wire format.
// Some type changes are safe (e.g., int32 -> int64 for reading), others are not.
func areTypesCompatible(oldType, newType TypeRef) bool {
	// Exact match is always compatible
	if oldType.String() == newType.String() {
		return true
	}

	// Handle some safe upgrades
	oldBase := baseTypeName(oldType)
	newBase := baseTypeName(newType)

	// Same base type with different modifiers might be compatible
	if oldBase == newBase {
		// Check if it's just a pointer/optional change
		oldOpt := isOptionalType(oldType)
		newOpt := isOptionalType(newType)
		if oldOpt != newOpt {
			// Changing optionality is generally safe for reading
			return true
		}
	}

	// Integer widening is safe for reading (smaller -> larger)
	intWidening := map[string]int{
		"int8": 1, "int16": 2, "int32": 3, "int64": 4,
		"uint8": 1, "uint16": 2, "uint32": 3, "uint64": 4,
	}
	if oldW, ok := intWidening[oldBase]; ok {
		if newW, ok := intWidening[newBase]; ok {
			// Check sign compatibility
			oldSigned := oldBase[0] != 'u'
			newSigned := newBase[0] != 'u'
			if oldSigned == newSigned && newW >= oldW {
				return true
			}
		}
	}

	return false
}

// baseTypeName extracts the base type name, stripping modifiers.
func baseTypeName(t TypeRef) string {
	switch v := t.(type) {
	case *NamedType:
		return v.Name
	case *ScalarType:
		return v.Name
	case *PointerType:
		return baseTypeName(v.Element)
	case *ArrayType:
		return "[]" + baseTypeName(v.Element)
	case *MapType:
		return "map[" + baseTypeName(v.Key) + "]" + baseTypeName(v.Value)
	default:
		return t.String()
	}
}

// isOptionalType checks if a type is optional (pointer).
func isOptionalType(t TypeRef) bool {
	_, ok := t.(*PointerType)
	return ok
}
