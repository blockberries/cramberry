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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blockberries/cramberry/internal/atomicfile"
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
	case "validate", "val":
		cmdValidate(os.Args[2:])
	case "format", "fmt", "f":
		cmdFormat(os.Args[2:])
	case "schema", "extract", "s":
		cmdSchema(os.Args[2:])
	case "version", "--version", "-V":
		cmdVersion()
	case "help", "-h", "--help":
		// `cramberry help <subcommand>` should print the subcommand's
		// usage; bare `help` prints the top-level usage. Without this
		// the user sees the same top-level dump for both forms.
		if len(os.Args) >= 3 {
			printSubcommandHelp(os.Args[2])
		} else {
			printUsage()
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// printSubcommandHelp dispatches `help <subcmd>` by re-invoking the
// subcommand with `-h`, which makes the standard `flag` package print
// the FlagSet's Usage and exit.
func printSubcommandHelp(name string) {
	switch name {
	case "generate", "gen", "g":
		cmdGenerate([]string{"-h"})
	case "validate", "val":
		cmdValidate([]string{"-h"})
	case "format", "fmt", "f":
		cmdFormat([]string{"-h"})
	case "schema", "extract", "s":
		cmdSchema([]string{"-h"})
	case "version":
		cmdVersion()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", name)
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

// importPathFlag allows multiple -M flags for import path mappings
type importPathFlag map[string]string

func (m *importPathFlag) String() string {
	if m == nil {
		return ""
	}
	var parts []string
	for k, v := range *m {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

func (m *importPathFlag) Set(value string) error {
	if *m == nil {
		*m = make(map[string]string)
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid import mapping: %s (expected alias=path)", value)
	}
	(*m)[parts[0]] = parts[1]
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
	// Initialize as an empty (non-nil) map so downstream codegen can rely
	// on a usable map even when -M is never passed.
	importPaths := importPathFlag{}
	fs.Var(&importPaths, "M", "Map schema import alias to Go import path (alias=path, can be repeated)")

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
		langs := codegen.Languages()
		names := make([]string, len(langs))
		for i, l := range langs {
			names[i] = string(l)
		}
		fmt.Fprintf(os.Stderr, "Supported languages: %s\n", strings.Join(names, ", "))
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
	opts.ImportPaths = importPaths

	// Create output directory
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "cramberry generate: %v\n", err)
		os.Exit(1)
	}

	// Process each input file
	loader := schema.NewLoader(searchPaths...)
	hasErrors := false

	for _, inputFile := range fs.Args() {
		s, diagnostics := loader.LoadFile(inputFile)
		// Treat only Severity=Error as fatal. Warnings are printed but
		// don't block generation — `cramberry validate` already serves
		// as the strict gate. This brings `generate` in line with
		// `validate`'s own classification of warnings vs errors.
		fatal := false
		for _, d := range diagnostics {
			fmt.Fprintln(os.Stderr, d)
			if isFatalDiagnostic(d) {
				fatal = true
			}
		}
		if fatal {
			hasErrors = true
			continue
		}

		// Get imported schemas for same-package detection
		opts.ImportedSchemas = loader.GetImportedSchemas(inputFile)

		// Generate output filename
		baseName := filepath.Base(inputFile)
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
		outputFile := filepath.Join(*outDir, baseName+gen.FileExtension())

		// Generate code via atomic write so a crash mid-generation can't
		// leave a partial .go/.ts/.rs file at the destination.
		if err := atomicfile.Write(outputFile, 0o644, func(w io.Writer) error {
			return gen.Generate(w, s, opts)
		}); err != nil {
			fmt.Fprintf(os.Stderr, "cramberry generate: %s: %v\n", inputFile, err)
			hasErrors = true
			continue
		}
		fmt.Printf("Generated: %s\n", outputFile)
	}

	if hasErrors {
		os.Exit(1)
	}
}

// isFatalDiagnostic returns true for parser/validator diagnostics that
// represent errors (not warnings/info). The schema package's
// ValidationError carries an explicit severity; anything else (parse
// errors, wrapped IO errors) is treated as fatal.
func isFatalDiagnostic(err error) bool {
	if v, ok := err.(*schema.ValidationError); ok {
		return v.Severity == schema.SeverityError
	}
	return true
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

	if *write && *diff {
		fmt.Fprintln(os.Stderr, "Error: -w and -d are mutually exclusive")
		os.Exit(1)
	}

	hasErrors := false
	formatChanged := false
	for _, inputFile := range fs.Args() {
		content, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cramberry format: %v\n", err)
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

		switch {
		case *diff:
			// Unified-style diff between the on-disk file and the
			// formatter's output. Prints nothing for already-formatted
			// files; sets exit code 1 if any file would change so this
			// command can be used as a CI check.
			if string(content) != formatted {
				formatChanged = true
				printUnifiedDiff(inputFile, string(content), formatted)
			}
		case *write:
			// Atomic write via temp + rename: a crash mid-write must
			// not leave the user's source file truncated.
			err := atomicfile.Write(inputFile, 0o644, func(w io.Writer) error {
				_, werr := w.Write([]byte(formatted))
				return werr
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "cramberry format: write %s: %v\n", inputFile, err)
				hasErrors = true
				continue
			}
			fmt.Printf("Formatted: %s\n", inputFile)
		default:
			fmt.Print(formatted)
		}
	}

	if formatChanged {
		// Same convention as gofmt -d: any formatting drift exits 1 so
		// the command can gate a CI step.
		os.Exit(1)
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

// printUnifiedDiff emits a minimal unified-diff-shaped output between the
// on-disk content and the formatted output. It's intentionally simple
// (whole-file context, no run-length compression of unchanged regions);
// the goal is a human-readable "this is what would change" signal, not
// a fully-conforming diff(1) replacement.
func printUnifiedDiff(path, original, formatted string) {
	fmt.Printf("--- %s (current)\n", path)
	fmt.Printf("+++ %s (formatted)\n", path)
	origLines := strings.Split(original, "\n")
	newLines := strings.Split(formatted, "\n")
	// Pad to equal length so iteration is symmetric.
	for len(origLines) < len(newLines) {
		origLines = append(origLines, "")
	}
	for len(newLines) < len(origLines) {
		newLines = append(newLines, "")
	}
	for i := range origLines {
		if origLines[i] == newLines[i] {
			fmt.Printf(" %s\n", origLines[i])
			continue
		}
		fmt.Printf("-%s\n", origLines[i])
		fmt.Printf("+%s\n", newLines[i])
	}
}
