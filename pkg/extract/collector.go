package extract

import (
	"go/ast"
	"go/types"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Config configures the type collector.
type Config struct {
	IncludePrivate         bool     // Include unexported types
	IncludePatterns        []string // Type name patterns to include (glob)
	ExcludePatterns        []string // Type name patterns to exclude (glob)
	DetectInterfaces       bool     // Auto-detect interface implementations
	IncludeEmptyInterfaces bool     // Include empty interfaces (marker interfaces for polymorphic grouping)
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		IncludePrivate:   false,
		DetectInterfaces: true,
	}
}

// TypeCollector collects type information from Go packages.
type TypeCollector struct {
	packages   []*packages.Package
	config     *Config
	types      map[string]*TypeInfo
	interfaces map[string]*InterfaceInfo
	enums      map[string]*EnumInfo
}

// NewTypeCollector creates a new type collector.
func NewTypeCollector(pkgs []*packages.Package, cfg *Config) *TypeCollector {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &TypeCollector{
		packages:   pkgs,
		config:     cfg,
		types:      make(map[string]*TypeInfo),
		interfaces: make(map[string]*InterfaceInfo),
		enums:      make(map[string]*EnumInfo),
	}
}

// Collect analyzes all packages and collects type information.
func (c *TypeCollector) Collect() error {
	for _, pkg := range c.packages {
		if err := c.collectPackage(pkg); err != nil {
			return err
		}
	}

	// Detect interface implementations if enabled
	if c.config.DetectInterfaces {
		c.detectImplementations()
	}

	return nil
}

// Types returns collected struct types.
func (c *TypeCollector) Types() map[string]*TypeInfo {
	return c.types
}

// Interfaces returns collected interfaces.
func (c *TypeCollector) Interfaces() map[string]*InterfaceInfo {
	return c.interfaces
}

// Enums returns collected enum types.
func (c *TypeCollector) Enums() map[string]*EnumInfo {
	return c.enums
}

func (c *TypeCollector) collectPackage(pkg *packages.Package) error {
	// Collect from syntax (for comments)
	typeComments := make(map[string]string)
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						doc := extractDoc(genDecl.Doc)
						if doc == "" {
							doc = extractDoc(typeSpec.Doc)
						}
						typeComments[typeSpec.Name.Name] = strings.TrimSpace(doc)
					}
				}
			}
		}
	}

	// Collect from types
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil {
			continue
		}

		// Filter by export status
		if !c.config.IncludePrivate && !obj.Exported() {
			continue
		}

		// Filter by patterns
		if !c.matchesPatterns(name) {
			continue
		}

		if typeName, ok := obj.(*types.TypeName); ok {
			c.collectType(typeName, pkg.PkgPath, typeComments[name])
		}
	}

	// Collect enum values
	c.collectEnumValues(pkg)

	return nil
}

func (c *TypeCollector) collectType(typeName *types.TypeName, pkgPath string, doc string) {
	underlying := typeName.Type().Underlying()
	qualifiedName := pkgPath + "." + typeName.Name()

	switch t := underlying.(type) {
	case *types.Struct:
		info := &TypeInfo{
			Name:       typeName.Name(),
			Package:    typeName.Pkg().Name(),
			PkgPath:    pkgPath,
			Doc:        doc,
			GoType:     typeName.Type(),
			IsExported: typeName.Exported(),
		}

		// Parse @typeID annotation from doc comment
		typeID, hasTypeID := parseTypeIDFromDoc(doc)
		if hasTypeID {
			info.TypeID = typeID
		}

		// Collect fields
		for i := 0; i < t.NumFields(); i++ {
			field := t.Field(i)
			if !c.config.IncludePrivate && !field.Exported() {
				continue
			}

			tag := t.Tag(i)
			structTag := c.parseTag(tag, i+1)
			if structTag.Skip {
				continue
			}

			fieldInfo := &FieldInfo{
				Name:      field.Name(),
				FieldNum:  structTag.FieldNum,
				GoType:    field.Type(),
				TypeName:  c.typeToString(field.Type()),
				Tag:       structTag,
				Optional:  structTag.OmitEmpty || isPointer(field.Type()),
				Repeated:  isSliceOrArray(field.Type()),
				IsPointer: isPointer(field.Type()),
			}
			info.Fields = append(info.Fields, fieldInfo)
		}

		c.types[qualifiedName] = info

	case *types.Interface:
		// Include interfaces with methods, or empty interfaces if configured
		if t.NumMethods() > 0 || c.config.IncludeEmptyInterfaces {
			info := &InterfaceInfo{
				Name:    typeName.Name(),
				Package: typeName.Pkg().Name(),
				PkgPath: pkgPath,
				Doc:     doc,
			}

			for i := 0; i < t.NumMethods(); i++ {
				info.Methods = append(info.Methods, t.Method(i).Name())
			}

			c.interfaces[qualifiedName] = info
		}

	case *types.Basic:
		// Check if it's an enum (int type with constants)
		if t.Info()&types.IsInteger != 0 {
			info := &EnumInfo{
				Name:    typeName.Name(),
				Package: typeName.Pkg().Name(),
				PkgPath: pkgPath,
				Doc:     doc,
				GoType:  typeName.Type(),
			}
			c.enums[qualifiedName] = info
		}
	}
}

func (c *TypeCollector) collectEnumValues(pkg *packages.Package) {
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil {
			continue
		}

		if cnst, ok := obj.(*types.Const); ok {
			// Get the type of this constant
			if named, ok := cnst.Type().(*types.Named); ok {
				// Skip types without a package (builtins)
				if named.Obj().Pkg() == nil {
					continue
				}
				qualifiedName := named.Obj().Pkg().Path() + "." + named.Obj().Name()
				if enumInfo, exists := c.enums[qualifiedName]; exists {
					// Get the constant value
					if val, ok := constantToInt64(cnst); ok {
						enumInfo.Values = append(enumInfo.Values, &EnumValueInfo{
							Name:   cnst.Name(),
							Number: val,
						})
					}
				}
			}
		}
	}
}

func constantToInt64(cnst *types.Const) (int64, bool) {
	if cnst.Val() == nil {
		return 0, false
	}
	val := cnst.Val().String()
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (c *TypeCollector) detectImplementations() {
	for _, iface := range c.interfaces {
		// Get the interface type
		ifaceType := c.findInterfaceType(iface.PkgPath, iface.Name)
		if ifaceType == nil {
			continue
		}

		// Check each collected type for implementation
		for _, typ := range c.types {
			if c.implements(typ.GoType, ifaceType) {
				iface.Implementations = append(iface.Implementations, typ)
				typ.Implements = append(typ.Implements, iface.PkgPath+"."+iface.Name)
			}
		}
	}
}

func (c *TypeCollector) findInterfaceType(pkgPath, name string) *types.Interface {
	for _, pkg := range c.packages {
		if pkg.PkgPath == pkgPath {
			obj := pkg.Types.Scope().Lookup(name)
			if obj != nil {
				if named, ok := obj.Type().(*types.Named); ok {
					if iface, ok := named.Underlying().(*types.Interface); ok {
						return iface
					}
				}
			}
		}
	}
	return nil
}

func (c *TypeCollector) implements(typ types.Type, iface *types.Interface) bool {
	// Check if typ implements iface
	// Need to check both *T and T
	if types.Implements(typ, iface) {
		return true
	}
	if ptr, ok := typ.(*types.Pointer); ok {
		return types.Implements(ptr.Elem(), iface)
	}
	return types.Implements(types.NewPointer(typ), iface)
}

func (c *TypeCollector) parseTag(tag string, defaultNum int) *StructTag {
	st := &StructTag{FieldNum: defaultNum}

	// Parse reflect tag
	structTag := reflect.StructTag(tag)
	cramberryTag := structTag.Get("cramberry")

	if cramberryTag == "-" {
		st.Skip = true
		return st
	}

	if cramberryTag != "" {
		parts := strings.Split(cramberryTag, ",")
		for i, part := range parts {
			if i == 0 {
				// First part is field number
				if num, err := strconv.Atoi(part); err == nil && num > 0 {
					st.FieldNum = num
				}
			} else {
				switch {
				case part == "omitempty":
					st.OmitEmpty = true
				case part == "required":
					st.Required = true
				case strings.HasPrefix(part, "typeID:"):
					// Parse typeID:N format
					if num, err := strconv.ParseUint(strings.TrimPrefix(part, "typeID:"), 10, 32); err == nil && num > 0 {
						st.TypeID = uint32(num)
						st.HasTypeID = true
					}
				}
			}
		}
	}

	return st
}

func (c *TypeCollector) matchesPatterns(name string) bool {
	// If no include patterns, include all
	if len(c.config.IncludePatterns) == 0 {
		// Check excludes
		for _, pattern := range c.config.ExcludePatterns {
			if matchGlob(pattern, name) {
				return false
			}
		}
		return true
	}

	// Check includes
	matched := false
	for _, pattern := range c.config.IncludePatterns {
		if matchGlob(pattern, name) {
			matched = true
			break
		}
	}

	if !matched {
		return false
	}

	// Check excludes
	for _, pattern := range c.config.ExcludePatterns {
		if matchGlob(pattern, name) {
			return false
		}
	}

	return true
}

func matchGlob(pattern, name string) bool {
	// Simple glob matching: * matches any sequence
	regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, `.*`) + "$"
	matched, _ := regexp.MatchString(regexPattern, name)
	return matched
}

// parseTypeIDFromDoc extracts a @typeID:N annotation from doc comments.
// Returns the type ID and true if found, otherwise 0 and false.
func parseTypeIDFromDoc(doc string) (uint32, bool) {
	// Look for @typeID:N or @cramberry:typeID=N patterns
	patterns := []string{
		`@typeID:(\d+)`,
		`@cramberry:typeID=(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(doc); len(matches) > 1 {
			if num, err := strconv.ParseUint(matches[1], 10, 32); err == nil && num > 0 {
				return uint32(num), true
			}
		}
	}
	return 0, false
}

func (c *TypeCollector) typeToString(t types.Type) string {
	return types.TypeString(t, func(pkg *types.Package) string {
		return pkg.Name()
	})
}

func isPointer(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

func isSliceOrArray(t types.Type) bool {
	switch t.(type) {
	case *types.Slice, *types.Array:
		return true
	}
	return false
}
