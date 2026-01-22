// Package extract provides tools for extracting Cramberry schemas from Go source code.
package extract

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// PackageLoader loads Go packages for analysis.
type PackageLoader struct {
	config *packages.Config
}

// NewPackageLoader creates a new package loader.
func NewPackageLoader() *PackageLoader {
	return &PackageLoader{
		config: &packages.Config{
			Mode: packages.NeedName |
				packages.NeedTypes |
				packages.NeedTypesInfo |
				packages.NeedSyntax |
				packages.NeedImports |
				packages.NeedDeps,
		},
	}
}

// Load loads packages matching the given patterns.
func (l *PackageLoader) Load(patterns []string) ([]*packages.Package, error) {
	pkgs, err := packages.Load(l.config, patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	// Check for errors in loaded packages
	var errs []error
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			errs = append(errs, err)
		}
	})

	if len(errs) > 0 {
		return nil, fmt.Errorf("package errors: %v", errs[0])
	}

	return pkgs, nil
}

// TypeInfo contains information about an extracted type.
type TypeInfo struct {
	Name       string
	Package    string
	PkgPath    string
	Doc        string
	Fields     []*FieldInfo
	TypeID     uint32
	GoType     types.Type
	Implements []string
	IsExported bool
}

// FieldInfo contains information about a struct field.
type FieldInfo struct {
	Name      string
	FieldNum  int
	GoType    types.Type
	TypeName  string
	Tag       *StructTag
	Doc       string
	Optional  bool
	Repeated  bool
	IsPointer bool
}

// InterfaceInfo contains information about an interface.
type InterfaceInfo struct {
	Name            string
	Package         string
	PkgPath         string
	Doc             string
	Methods         []string
	Implementations []*TypeInfo
}

// EnumInfo contains information about an enum type.
type EnumInfo struct {
	Name    string
	Package string
	PkgPath string
	Doc     string
	Values  []*EnumValueInfo
	GoType  types.Type
}

// EnumValueInfo contains information about an enum value.
type EnumValueInfo struct {
	Name   string
	Number int64
	Doc    string
}

// StructTag represents a parsed cramberry struct tag.
type StructTag struct {
	FieldNum   int
	Name       string
	OmitEmpty  bool
	Required   bool
	Skip       bool
	Deprecated string
}

// extractDoc extracts documentation from an AST node.
func extractDoc(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return cg.Text()
}
