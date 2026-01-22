package extract

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/blockberries/cramberry/pkg/schema"
)

// Extractor extracts schemas from Go packages.
type Extractor struct {
	loader *PackageLoader
}

// NewExtractor creates a new schema extractor.
func NewExtractor() *Extractor {
	return &Extractor{
		loader: NewPackageLoader(),
	}
}

// ExtractorConfig configures the extraction process.
type ExtractorConfig struct {
	Config     *Config  // Type collector configuration
	Patterns   []string // Go package patterns to load
	OutputPath string   // Output file path (empty for stdout)
	Package    string   // Package name for generated schema
}

// Extract extracts a schema from Go packages.
func (e *Extractor) Extract(cfg *ExtractorConfig) (*schema.Schema, error) {
	// Load packages
	pkgs, err := e.loader.Load(cfg.Patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages matched patterns: %v", cfg.Patterns)
	}

	// Collect types
	collectorCfg := cfg.Config
	if collectorCfg == nil {
		collectorCfg = DefaultConfig()
	}
	collector := NewTypeCollector(pkgs, collectorCfg)
	if err := collector.Collect(); err != nil {
		return nil, fmt.Errorf("failed to collect types: %w", err)
	}

	// Determine package name
	packageName := cfg.Package
	if packageName == "" && len(pkgs) > 0 {
		packageName = pkgs[0].Name
	}

	// Build schema
	builder := NewSchemaBuilder(collector.Types(), collector.Interfaces(), collector.Enums())
	s, err := builder.Build(packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to build schema: %w", err)
	}

	return s, nil
}

// ExtractAndWrite extracts a schema and writes it to the specified output.
func (e *Extractor) ExtractAndWrite(cfg *ExtractorConfig) error {
	s, err := e.Extract(cfg)
	if err != nil {
		return err
	}

	// Determine output destination
	var out io.Writer = os.Stdout
	if cfg.OutputPath != "" {
		// Ensure output directory exists
		dir := filepath.Dir(cfg.OutputPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		f, err := os.Create(cfg.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	// Write schema
	writer := schema.NewWriter()
	return writer.WriteSchema(out, s)
}

// ExtractToString is a convenience function that extracts a schema and returns it as a string.
func ExtractToString(patterns []string, config *Config) (string, error) {
	extractor := NewExtractor()
	s, err := extractor.Extract(&ExtractorConfig{
		Config:   config,
		Patterns: patterns,
	})
	if err != nil {
		return "", err
	}
	return schema.FormatSchema(s), nil
}
