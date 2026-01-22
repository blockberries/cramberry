// Package schema provides types and parsing for Cramberry schema files.
//
// Schema files (.cramberry) define the structure of messages, enums,
// and interfaces for code generation across multiple languages.
package schema

// Position represents a position in source code.
type Position struct {
	Filename string
	Line     int
	Column   int
	Offset   int
}

// Node is the interface implemented by all AST nodes.
type Node interface {
	Pos() Position
	End() Position
}

// Schema represents a complete schema file.
type Schema struct {
	Position   Position
	Package    *Package
	Imports    []*Import
	Options    []*Option
	Messages   []*Message
	Enums      []*Enum
	Interfaces []*Interface
	Comments   []*Comment
}

func (s *Schema) Pos() Position { return s.Position }
func (s *Schema) End() Position {
	if len(s.Messages) > 0 {
		return s.Messages[len(s.Messages)-1].End()
	}
	if len(s.Enums) > 0 {
		return s.Enums[len(s.Enums)-1].End()
	}
	if s.Package != nil {
		return s.Package.End()
	}
	return s.Position
}

// Package declares the package name for generated code.
type Package struct {
	Position Position
	EndPos   Position
	Name     string
}

func (p *Package) Pos() Position { return p.Position }
func (p *Package) End() Position { return p.EndPos }

// Import imports definitions from another schema file.
type Import struct {
	Position Position
	EndPos   Position
	Path     string
	Alias    string // Optional alias for the import
}

func (i *Import) Pos() Position { return i.Position }
func (i *Import) End() Position { return i.EndPos }

// Option represents a schema-level or field-level option.
type Option struct {
	Position Position
	EndPos   Position
	Name     string
	Value    Value
}

func (o *Option) Pos() Position { return o.Position }
func (o *Option) End() Position { return o.EndPos }

// Value represents an option value (string, number, bool, or list).
type Value interface {
	Node
	valueNode()
}

// StringValue is a string literal value.
type StringValue struct {
	Position Position
	EndPos   Position
	Value    string
}

func (v *StringValue) Pos() Position { return v.Position }
func (v *StringValue) End() Position { return v.EndPos }
func (v *StringValue) valueNode()    {}

// NumberValue is a numeric literal value.
type NumberValue struct {
	Position Position
	EndPos   Position
	Value    string // Stored as string to preserve precision
	IsFloat  bool
}

func (v *NumberValue) Pos() Position { return v.Position }
func (v *NumberValue) End() Position { return v.EndPos }
func (v *NumberValue) valueNode()    {}

// BoolValue is a boolean literal value.
type BoolValue struct {
	Position Position
	EndPos   Position
	Value    bool
}

func (v *BoolValue) Pos() Position { return v.Position }
func (v *BoolValue) End() Position { return v.EndPos }
func (v *BoolValue) valueNode()    {}

// ListValue is a list of values.
type ListValue struct {
	Position Position
	EndPos   Position
	Values   []Value
}

func (v *ListValue) Pos() Position { return v.Position }
func (v *ListValue) End() Position { return v.EndPos }
func (v *ListValue) valueNode()    {}

// Message represents a message (struct) definition.
type Message struct {
	Position Position
	EndPos   Position
	Name     string
	Fields   []*Field
	Options  []*Option
	Comments []*Comment
	TypeID   int // Assigned type ID (0 = auto-assign)
}

func (m *Message) Pos() Position { return m.Position }
func (m *Message) End() Position { return m.EndPos }

// Field represents a field within a message.
type Field struct {
	Position   Position
	EndPos     Position
	Name       string
	Number     int
	Type       TypeRef
	Options    []*Option
	Comments   []*Comment
	Required   bool
	Repeated   bool
	Optional   bool
	MapKey     TypeRef // For map types
	MapValue   TypeRef // For map types
	Deprecated bool
}

func (f *Field) Pos() Position { return f.Position }
func (f *Field) End() Position { return f.EndPos }

// TypeRef represents a type reference.
type TypeRef interface {
	Node
	typeRefNode()
	String() string
}

// ScalarType represents a built-in scalar type.
type ScalarType struct {
	Position Position
	EndPos   Position
	Name     string // bool, int32, uint64, float32, float64, string, bytes, etc.
}

func (t *ScalarType) Pos() Position  { return t.Position }
func (t *ScalarType) End() Position  { return t.EndPos }
func (t *ScalarType) typeRefNode()   {}
func (t *ScalarType) String() string { return t.Name }

// NamedType represents a reference to a message, enum, or interface.
type NamedType struct {
	Position  Position
	EndPos    Position
	Package   string // Optional package prefix
	Name      string
	TypeArgs  []TypeRef // For generic types (future)
}

func (t *NamedType) Pos() Position { return t.Position }
func (t *NamedType) End() Position { return t.EndPos }
func (t *NamedType) typeRefNode()  {}
func (t *NamedType) String() string {
	if t.Package != "" {
		return t.Package + "." + t.Name
	}
	return t.Name
}

// ArrayType represents an array/slice type.
type ArrayType struct {
	Position Position
	EndPos   Position
	Element  TypeRef
	Size     int // 0 for slice, >0 for fixed-size array
}

func (t *ArrayType) Pos() Position { return t.Position }
func (t *ArrayType) End() Position { return t.EndPos }
func (t *ArrayType) typeRefNode()  {}
func (t *ArrayType) String() string {
	if t.Size > 0 {
		return "[" + string(rune('0'+t.Size)) + "]" + t.Element.String()
	}
	return "[]" + t.Element.String()
}

// MapType represents a map type.
type MapType struct {
	Position Position
	EndPos   Position
	Key      TypeRef
	Value    TypeRef
}

func (t *MapType) Pos() Position  { return t.Position }
func (t *MapType) End() Position  { return t.EndPos }
func (t *MapType) typeRefNode()   {}
func (t *MapType) String() string { return "map[" + t.Key.String() + "]" + t.Value.String() }

// PointerType represents a pointer/optional type.
type PointerType struct {
	Position Position
	EndPos   Position
	Element  TypeRef
}

func (t *PointerType) Pos() Position  { return t.Position }
func (t *PointerType) End() Position  { return t.EndPos }
func (t *PointerType) typeRefNode()   {}
func (t *PointerType) String() string { return "*" + t.Element.String() }

// Enum represents an enum definition.
type Enum struct {
	Position Position
	EndPos   Position
	Name     string
	Values   []*EnumValue
	Options  []*Option
	Comments []*Comment
}

func (e *Enum) Pos() Position { return e.Position }
func (e *Enum) End() Position { return e.EndPos }

// EnumValue represents a single enum value.
type EnumValue struct {
	Position Position
	EndPos   Position
	Name     string
	Number   int
	Options  []*Option
	Comments []*Comment
}

func (v *EnumValue) Pos() Position { return v.Position }
func (v *EnumValue) End() Position { return v.EndPos }

// Interface represents an interface definition for polymorphic types.
type Interface struct {
	Position       Position
	EndPos         Position
	Name           string
	Implementations []*Implementation
	Options        []*Option
	Comments       []*Comment
}

func (i *Interface) Pos() Position { return i.Position }
func (i *Interface) End() Position { return i.EndPos }

// Implementation maps a type ID to a message type.
type Implementation struct {
	Position Position
	EndPos   Position
	TypeID   int
	Type     *NamedType
	Comments []*Comment
}

func (i *Implementation) Pos() Position { return i.Position }
func (i *Implementation) End() Position { return i.EndPos }

// Comment represents a comment in the schema.
type Comment struct {
	Position Position
	EndPos   Position
	Text     string
	IsDoc    bool // True if this is a doc comment (///)
}

func (c *Comment) Pos() Position { return c.Position }
func (c *Comment) End() Position { return c.EndPos }

// ScalarTypes defines the built-in scalar types.
var ScalarTypes = map[string]bool{
	"bool":       true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"int":        true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uint":       true,
	"float32":    true,
	"float64":    true,
	"complex64":  true,
	"complex128": true,
	"string":     true,
	"bytes":      true,
}

// IsScalar returns true if the type name is a scalar type.
func IsScalar(name string) bool {
	return ScalarTypes[name]
}
