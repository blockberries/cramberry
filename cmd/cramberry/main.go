// Command cramberry is the Cramberry schema compiler and code generator.
//
// Usage:
//
//	cramberry generate [options] <schema-file>...
//	cramberry validate <schema-file>...
//	cramberry format <schema-file>...
//	cramberry schema [options] <go-package>...
//	cramberry version
//
// Generate Command:
//
//	Generate code from schema files.
//
//	Options:
//	  -lang string      Target language: go, typescript, rust (default "go")
//	  -out string       Output directory (default ".")
//	  -package string   Override package name
//	  -prefix string    Add prefix to all type names
//	  -suffix string    Add suffix to all type names
//	  -marshal          Generate marshal/unmarshal methods (default true)
//	  -json             Generate JSON tags/methods (default true)
//	  -I string         Add import search path (can be repeated)
//
// Validate Command:
//
//	Validate schema files without generating code.
//
// Format Command:
//
//	Format schema files in place.
//
// Schema Command:
//
//	Extract schema from Go source code.
//
//	Options:
//	  -out string       Output file (default: stdout)
//	  -package string   Override package name
//	  -private          Include unexported types
//	  -include string   Type name pattern to include (glob, can be repeated)
//	  -exclude string   Type name pattern to exclude (glob, can be repeated)
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blockberries/cramberry/pkg/codegen"
	"github.com/blockberries/cramberry/pkg/cramberry"
	"github.com/blockberries/cramberry/pkg/extract"
	"github.com/blockberries/cramberry/pkg/schema"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate", "gen", "g":
		cmdGenerate(os.Args[2:])
	case "validate", "val", "v":
		cmdValidate(os.Args[2:])
	case "format", "fmt", "f":
		cmdFormat(os.Args[2:])
	case "schema", "extract", "s":
		cmdSchema(os.Args[2:])
	case "version":
		cmdVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Cramberry Schema Compiler

Usage:
  cramberry <command> [options] <files>...

Commands:
  generate    Generate code from schema files
  validate    Validate schema files
  format      Format schema files
  schema      Extract schema from Go source code
  version     Print version information
  help        Print this help message

Run 'cramberry <command> -h' for command-specific help.`)
}

// stringSliceFlag allows multiple -I flags
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	lang := fs.String("lang", "go", "Target language: go, typescript, rust")
	outDir := fs.String("out", ".", "Output directory")
	pkg := fs.String("package", "", "Override package name")
	prefix := fs.String("prefix", "", "Add prefix to all type names")
	suffix := fs.String("suffix", "", "Add suffix to all type names")
	marshal := fs.Bool("marshal", true, "Generate marshal/unmarshal methods")
	jsonTags := fs.Bool("json", true, "Generate JSON tags/methods")
	var searchPaths stringSliceFlag
	fs.Var(&searchPaths, "I", "Add import search path (can be repeated)")

	fs.Usage = func() {
		fmt.Println(`Usage: cramberry generate [options] <schema-file>...

Generate code from Cramberry schema files.

Options:`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: no input files")
		fs.Usage()
		os.Exit(1)
	}

	// Get generator
	gen, ok := codegen.Get(codegen.Language(*lang))
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unsupported language: %s\n", *lang)
		fmt.Fprintln(os.Stderr, "Supported languages: go")
		os.Exit(1)
	}

	// Configure options
	opts := codegen.DefaultOptions()
	opts.Package = *pkg
	opts.OutputPath = *outDir
	opts.TypePrefix = *prefix
	opts.TypeSuffix = *suffix
	opts.GenerateMarshal = *marshal
	opts.GenerateJSON = *jsonTags

	// Create output directory
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Process each input file
	loader := schema.NewLoader(searchPaths...)
	hasErrors := false

	for _, inputFile := range fs.Args() {
		s, errors := loader.LoadFile(inputFile)
		if len(errors) > 0 {
			hasErrors = true
			for _, err := range errors {
				fmt.Fprintln(os.Stderr, err)
			}
			continue
		}

		// Generate output filename
		baseName := filepath.Base(inputFile)
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
		outputFile := filepath.Join(*outDir, baseName+gen.FileExtension())

		// Generate code
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			hasErrors = true
			continue
		}

		if err := gen.Generate(f, s, opts); err != nil {
			f.Close()
			os.Remove(outputFile)
			fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
			hasErrors = true
			continue
		}

		f.Close()
		fmt.Printf("Generated: %s\n", outputFile)
	}

	if hasErrors {
		os.Exit(1)
	}
}

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	var searchPaths stringSliceFlag
	fs.Var(&searchPaths, "I", "Add import search path (can be repeated)")

	fs.Usage = func() {
		fmt.Println(`Usage: cramberry validate [options] <schema-file>...

Validate Cramberry schema files without generating code.

Options:`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: no input files")
		fs.Usage()
		os.Exit(1)
	}

	loader := schema.NewLoader(searchPaths...)
	hasErrors := false
	hasWarnings := false

	for _, inputFile := range fs.Args() {
		_, errors := loader.LoadFile(inputFile)
		if len(errors) > 0 {
			for _, err := range errors {
				fmt.Fprintln(os.Stderr, err)
				// Check if it's a warning
				if valErr, ok := err.(schema.ValidationError); ok && valErr.Severity == schema.SeverityWarning {
					hasWarnings = true
				} else {
					hasErrors = true
				}
			}
		} else {
			fmt.Printf("Valid: %s\n", inputFile)
		}
	}

	if hasErrors {
		os.Exit(1)
	}
	if hasWarnings {
		os.Exit(2)
	}
}

func cmdFormat(args []string) {
	fs := flag.NewFlagSet("format", flag.ExitOnError)
	write := fs.Bool("w", false, "Write result to (source) file instead of stdout")
	diff := fs.Bool("d", false, "Display diffs instead of rewriting files")

	fs.Usage = func() {
		fmt.Println(`Usage: cramberry format [options] <schema-file>...

Format Cramberry schema files.

Options:`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: no input files")
		fs.Usage()
		os.Exit(1)
	}

	_ = diff // TODO: implement diff output

	hasErrors := false
	for _, inputFile := range fs.Args() {
		content, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputFile, err)
			hasErrors = true
			continue
		}

		s, parseErrors := schema.ParseFile(inputFile, string(content))
		if len(parseErrors) > 0 {
			for _, e := range parseErrors {
				fmt.Fprintln(os.Stderr, e)
			}
			hasErrors = true
			continue
		}

		formatted := schema.FormatSchema(s)

		if *write {
			if err := os.WriteFile(inputFile, []byte(formatted), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", inputFile, err)
				hasErrors = true
				continue
			}
			fmt.Printf("Formatted: %s\n", inputFile)
		} else {
			fmt.Print(formatted)
		}
	}

	if hasErrors {
		os.Exit(1)
	}
}

func cmdSchema(args []string) {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	outFile := fs.String("out", "", "Output file (default: stdout)")
	pkg := fs.String("package", "", "Override package name")
	private := fs.Bool("private", false, "Include unexported types")
	var includePatterns stringSliceFlag
	fs.Var(&includePatterns, "include", "Type name pattern to include (glob, can be repeated)")
	var excludePatterns stringSliceFlag
	fs.Var(&excludePatterns, "exclude", "Type name pattern to exclude (glob, can be repeated)")

	fs.Usage = func() {
		fmt.Println(`Usage: cramberry schema [options] <go-package>...

Extract Cramberry schema from Go source code.

Examples:
  cramberry schema ./...
  cramberry schema -out schema.cram ./pkg/models
  cramberry schema -include "User*" -exclude "*Internal" ./...

Options:`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: no Go packages specified")
		fs.Usage()
		os.Exit(1)
	}

	// Configure extraction
	cfg := &extract.ExtractorConfig{
		Config: &extract.Config{
			IncludePrivate:   *private,
			IncludePatterns:  includePatterns,
			ExcludePatterns:  excludePatterns,
			DetectInterfaces: true,
		},
		Patterns:   fs.Args(),
		OutputPath: *outFile,
		Package:    *pkg,
	}

	// Extract schema
	extractor := extract.NewExtractor()
	if err := extractor.ExtractAndWrite(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *outFile != "" {
		fmt.Printf("Extracted: %s\n", *outFile)
	}
}

func cmdVersion() {
	fmt.Printf("cramberry version %s\n", cramberry.VersionInfo())
}
