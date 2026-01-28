package schema

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Loader loads and resolves schema files.
type Loader struct {
	// SearchPaths are directories to search for imported schemas.
	SearchPaths []string

	// Loaded caches loaded schemas by their resolved path.
	loaded map[string]*Schema

	// LoadedErrors caches parse/validation errors by path.
	loadedErrors map[string][]error
}

// NewLoader creates a new schema loader with the given search paths.
func NewLoader(searchPaths ...string) *Loader {
	return &Loader{
		SearchPaths:  searchPaths,
		loaded:       make(map[string]*Schema),
		loadedErrors: make(map[string][]error),
	}
}

// LoadFile loads a schema file and all its imports.
func (l *Loader) LoadFile(path string) (*Schema, []error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to resolve path: %w", err)}
	}

	return l.loadFileInternal(absPath, nil)
}

// loadFileInternal loads a schema file, tracking the import chain to detect cycles.
func (l *Loader) loadFileInternal(absPath string, importChain []string) (*Schema, []error) {
	// Check for circular imports
	for _, p := range importChain {
		if p == absPath {
			return nil, []error{fmt.Errorf("circular import detected: %s", strings.Join(append(importChain, absPath), " -> "))}
		}
	}

	// Return cached schema if available
	if schema, ok := l.loaded[absPath]; ok {
		return schema, l.loadedErrors[absPath]
	}

	// Read file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read file %s: %w", absPath, err)}
	}

	// Parse
	schema, parseErrors := ParseFile(absPath, string(content))
	var allErrors []error
	for _, e := range parseErrors {
		allErrors = append(allErrors, e)
	}

	if len(parseErrors) > 0 {
		l.loaded[absPath] = schema
		l.loadedErrors[absPath] = allErrors
		return schema, allErrors
	}

	// Cache early to handle recursive imports
	l.loaded[absPath] = schema

	// Resolve imports
	baseDir := filepath.Dir(absPath)
	importedSchemas := make(map[string]*Schema)
	newChain := append(importChain, absPath)

	for _, imp := range schema.Imports {
		importPath := l.resolveImportPath(imp.Path, baseDir)
		if importPath == "" {
			allErrors = append(allErrors, fmt.Errorf("%s:%d: import not found: %s",
				absPath, imp.Position.Line, imp.Path))
			continue
		}

		importedSchema, importErrors := l.loadFileInternal(importPath, newChain)
		if len(importErrors) > 0 {
			allErrors = append(allErrors, importErrors...)
		}
		if importedSchema != nil {
			key := imp.Alias
			if key == "" {
				key = imp.Path
			}
			importedSchemas[key] = importedSchema
		}
	}

	// Validate with imports
	valErrors := ValidateWithImports(schema, importedSchemas)
	for _, e := range valErrors {
		if e.Severity == SeverityError {
			allErrors = append(allErrors, e)
		}
	}

	l.loadedErrors[absPath] = allErrors
	return schema, allErrors
}

// resolveImportPath resolves an import path to an absolute file path.
func (l *Loader) resolveImportPath(importPath, baseDir string) string {
	// Try relative to current file first
	candidate := filepath.Join(baseDir, importPath)
	if _, err := os.Stat(candidate); err == nil {
		absPath, _ := filepath.Abs(candidate)
		return absPath
	}

	// Try search paths
	for _, searchPath := range l.SearchPaths {
		candidate := filepath.Join(searchPath, importPath)
		if _, err := os.Stat(candidate); err == nil {
			absPath, _ := filepath.Abs(candidate)
			return absPath
		}
	}

	return ""
}

// GetSchema returns a loaded schema by its path.
func (l *Loader) GetSchema(path string) *Schema {
	absPath, _ := filepath.Abs(path)
	return l.loaded[absPath]
}

// AllSchemas returns all loaded schemas.
func (l *Loader) AllSchemas() map[string]*Schema {
	result := make(map[string]*Schema, len(l.loaded))
	for k, v := range l.loaded {
		result[k] = v
	}
	return result
}

// GetImportedSchemas returns the imported schemas for a given schema file,
// mapped by their import aliases. This is useful for code generators that
// need to know whether imported types are from the same package.
func (l *Loader) GetImportedSchemas(path string) map[string]*Schema {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil
	}

	s := l.loaded[absPath]
	if s == nil {
		return nil
	}

	result := make(map[string]*Schema)
	baseDir := filepath.Dir(absPath)

	for _, imp := range s.Imports {
		importPath := l.resolveImportPath(imp.Path, baseDir)
		if importPath == "" {
			continue
		}

		importedSchema := l.loaded[importPath]
		if importedSchema != nil {
			key := imp.Alias
			if key == "" {
				key = imp.Path
			}
			result[key] = importedSchema
		}
	}

	return result
}

// Writer writes schemas to various formats.
type Writer struct {
	indent string
}

// NewWriter creates a new schema writer.
func NewWriter() *Writer {
	return &Writer{
		indent: "  ",
	}
}

// SetIndent sets the indentation string (default is two spaces).
func (w *Writer) SetIndent(indent string) {
	w.indent = indent
}

// WriteSchema writes a schema to the writer.
func (w *Writer) WriteSchema(out io.Writer, schema *Schema) error {
	// Write package
	if schema.Package != nil {
		fmt.Fprintf(out, "package %s;\n\n", schema.Package.Name)
	}

	// Write imports
	for _, imp := range schema.Imports {
		if imp.Alias != "" {
			fmt.Fprintf(out, "import %q as %s;\n", imp.Path, imp.Alias)
		} else {
			fmt.Fprintf(out, "import %q;\n", imp.Path)
		}
	}
	if len(schema.Imports) > 0 {
		fmt.Fprintln(out)
	}

	// Write options
	for _, opt := range schema.Options {
		fmt.Fprintf(out, "option %s = %s;\n", opt.Name, w.formatValue(opt.Value))
	}
	if len(schema.Options) > 0 {
		fmt.Fprintln(out)
	}

	// Write messages
	for i, msg := range schema.Messages {
		w.writeMessage(out, msg)
		if i < len(schema.Messages)-1 || len(schema.Enums) > 0 || len(schema.Interfaces) > 0 {
			fmt.Fprintln(out)
		}
	}

	// Write enums
	for i, enum := range schema.Enums {
		w.writeEnum(out, enum)
		if i < len(schema.Enums)-1 || len(schema.Interfaces) > 0 {
			fmt.Fprintln(out)
		}
	}

	// Write interfaces
	for i, iface := range schema.Interfaces {
		w.writeInterface(out, iface)
		if i < len(schema.Interfaces)-1 {
			fmt.Fprintln(out)
		}
	}

	return nil
}

// writeMessage writes a message definition.
func (w *Writer) writeMessage(out io.Writer, msg *Message) {
	// Write doc comments
	for _, comment := range msg.Comments {
		if comment.IsDoc {
			fmt.Fprintf(out, "/// %s\n", comment.Text)
		}
	}

	// Write message header
	if msg.TypeID > 0 {
		fmt.Fprintf(out, "message %s @%d {\n", msg.Name, msg.TypeID)
	} else {
		fmt.Fprintf(out, "message %s {\n", msg.Name)
	}

	// Write options
	for _, opt := range msg.Options {
		fmt.Fprintf(out, "%soption %s = %s;\n", w.indent, opt.Name, w.formatValue(opt.Value))
	}

	// Write fields
	for _, field := range msg.Fields {
		w.writeField(out, field)
	}

	fmt.Fprintln(out, "}")
}

// writeField writes a field definition.
func (w *Writer) writeField(out io.Writer, field *Field) {
	// Write doc comments
	for _, comment := range field.Comments {
		if comment.IsDoc {
			fmt.Fprintf(out, "%s/// %s\n", w.indent, comment.Text)
		}
	}

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
	if field.Deprecated {
		modifiers = append(modifiers, "deprecated")
	}

	modStr := ""
	if len(modifiers) > 0 {
		modStr = strings.Join(modifiers, " ") + " "
	}

	typeStr := field.Type.String()

	// Format field options
	optStr := ""
	if len(field.Options) > 0 {
		var optParts []string
		for _, opt := range field.Options {
			optParts = append(optParts, fmt.Sprintf("%s = %s", opt.Name, w.formatValue(opt.Value)))
		}
		optStr = " [" + strings.Join(optParts, ", ") + "]"
	}

	fmt.Fprintf(out, "%s%s%s %s = %d%s;\n", w.indent, modStr, typeStr, field.Name, field.Number, optStr)
}

// writeEnum writes an enum definition.
func (w *Writer) writeEnum(out io.Writer, enum *Enum) {
	// Write doc comments
	for _, comment := range enum.Comments {
		if comment.IsDoc {
			fmt.Fprintf(out, "/// %s\n", comment.Text)
		}
	}

	fmt.Fprintf(out, "enum %s {\n", enum.Name)

	// Write options
	for _, opt := range enum.Options {
		fmt.Fprintf(out, "%soption %s = %s;\n", w.indent, opt.Name, w.formatValue(opt.Value))
	}

	// Write values
	for _, val := range enum.Values {
		for _, comment := range val.Comments {
			if comment.IsDoc {
				fmt.Fprintf(out, "%s/// %s\n", w.indent, comment.Text)
			}
		}
		fmt.Fprintf(out, "%s%s = %d;\n", w.indent, val.Name, val.Number)
	}

	fmt.Fprintln(out, "}")
}

// writeInterface writes an interface definition.
func (w *Writer) writeInterface(out io.Writer, iface *Interface) {
	// Write doc comments
	for _, comment := range iface.Comments {
		if comment.IsDoc {
			fmt.Fprintf(out, "/// %s\n", comment.Text)
		}
	}

	fmt.Fprintf(out, "interface %s {\n", iface.Name)

	// Write options
	for _, opt := range iface.Options {
		fmt.Fprintf(out, "%soption %s = %s;\n", w.indent, opt.Name, w.formatValue(opt.Value))
	}

	// Write implementations
	for _, impl := range iface.Implementations {
		for _, comment := range impl.Comments {
			if comment.IsDoc {
				fmt.Fprintf(out, "%s/// %s\n", w.indent, comment.Text)
			}
		}
		fmt.Fprintf(out, "%s%d = %s;\n", w.indent, impl.TypeID, impl.Type.String())
	}

	fmt.Fprintln(out, "}")
}

// formatValue formats a value for output.
func (w *Writer) formatValue(v Value) string {
	switch val := v.(type) {
	case *StringValue:
		return fmt.Sprintf("%q", val.Value)
	case *NumberValue:
		return val.Value
	case *BoolValue:
		if val.Value {
			return "true"
		}
		return "false"
	case *ListValue:
		var parts []string
		for _, elem := range val.Values {
			parts = append(parts, w.formatValue(elem))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// WriteToFile writes a schema to a file.
func WriteToFile(path string, schema *Schema) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := NewWriter()
	return writer.WriteSchema(f, schema)
}

// FormatSchema returns a formatted string representation of a schema.
func FormatSchema(schema *Schema) string {
	var sb strings.Builder
	writer := NewWriter()
	_ = writer.WriteSchema(&sb, schema) // Error can't happen with strings.Builder
	return sb.String()
}

// LoadAndValidate is a convenience function that loads a schema file
// and returns all errors (parse + validation).
func LoadAndValidate(path string, searchPaths ...string) (*Schema, []error) {
	loader := NewLoader(searchPaths...)
	return loader.LoadFile(path)
}
