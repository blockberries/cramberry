package extract

import (
	"strings"
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ID", "id"},
		{"UserName", "user_name"},
		{"FirstName", "first_name"},
		{"HTTPRequest", "http_request"},
		{"HTTPServer", "http_server"},
		{"XMLParser", "xml_parser"},
		{"simple", "simple"},
		{"userID", "user_id"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		name     string
		expected bool
	}{
		{"User*", "User", true},
		{"User*", "UserInfo", true},
		{"User*", "Admin", false},
		{"*Info", "UserInfo", true},
		{"*Info", "User", false},
		{"*", "Anything", true},
		{"User", "User", true},
		{"User", "Admin", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.name, func(t *testing.T) {
			result := matchGlob(tt.pattern, tt.name)
			if result != tt.expected {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.name, result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if cfg.IncludePrivate {
		t.Error("IncludePrivate should be false by default")
	}
	if !cfg.DetectInterfaces {
		t.Error("DetectInterfaces should be true by default")
	}
	if len(cfg.IncludePatterns) != 0 {
		t.Error("IncludePatterns should be empty by default")
	}
	if len(cfg.ExcludePatterns) != 0 {
		t.Error("ExcludePatterns should be empty by default")
	}
}

func TestSchemaBuilderBuild(t *testing.T) {
	types := make(map[string]*TypeInfo)
	interfaces := make(map[string]*InterfaceInfo)
	enums := make(map[string]*EnumInfo)

	builder := NewSchemaBuilder(types, interfaces, enums)
	schema, err := builder.Build("testpackage")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if schema == nil {
		t.Fatal("Build() returned nil schema")
	}
	if schema.Package == nil {
		t.Fatal("Build() returned schema with nil Package")
	}
	if schema.Package.Name != "testpackage" {
		t.Errorf("Build() package name = %q, want %q", schema.Package.Name, "testpackage")
	}
}

func TestExtractorConfig(t *testing.T) {
	cfg := &ExtractorConfig{
		Config:     DefaultConfig(),
		Patterns:   []string{"./..."},
		OutputPath: "test.cramberry",
		Package:    "testpkg",
	}

	if cfg.Config == nil {
		t.Error("Config should not be nil")
	}
	if len(cfg.Patterns) != 1 {
		t.Error("Patterns should have one element")
	}
	if cfg.OutputPath != "test.cramberry" {
		t.Error("OutputPath mismatch")
	}
	if cfg.Package != "testpkg" {
		t.Error("Package mismatch")
	}
}

// TestExtractToString tests extraction from a simple test package.
func TestExtractToString(t *testing.T) {
	result, err := ExtractToString([]string{"github.com/blockberries/cramberry/pkg/extract/testdata"}, DefaultConfig())
	if err != nil {
		t.Fatalf("ExtractToString() error = %v", err)
	}
	if result == "" {
		t.Error("ExtractToString() returned empty string")
	}
	if !strings.Contains(result, "package") {
		t.Error("ExtractToString() result should contain 'package'")
	}

	// Check for expected types
	if !strings.Contains(result, "User") {
		t.Error("result should contain 'User' message")
	}
	if !strings.Contains(result, "Address") {
		t.Error("result should contain 'Address' message")
	}
	if !strings.Contains(result, "Status") {
		t.Error("result should contain 'Status' enum")
	}
	if !strings.Contains(result, "Person") {
		t.Error("result should contain 'Person' interface")
	}

	// Check that private types are excluded
	if strings.Contains(result, "privateType") {
		t.Error("result should NOT contain 'privateType' (unexported)")
	}
}

// TestExtractWithPrivate tests extraction including unexported types.
func TestExtractWithPrivate(t *testing.T) {
	cfg := &Config{
		IncludePrivate:   true,
		DetectInterfaces: true,
	}
	result, err := ExtractToString([]string{"github.com/blockberries/cramberry/pkg/extract/testdata"}, cfg)
	if err != nil {
		t.Fatalf("ExtractToString() error = %v", err)
	}

	// Check that private types are now included
	if !strings.Contains(result, "privateType") {
		t.Error("result should contain 'privateType' when IncludePrivate is true")
	}
}

// TestExtractWithPatterns tests extraction with include/exclude patterns.
func TestExtractWithPatterns(t *testing.T) {
	cfg := &Config{
		IncludePatterns:  []string{"User*"},
		DetectInterfaces: true,
	}
	result, err := ExtractToString([]string{"github.com/blockberries/cramberry/pkg/extract/testdata"}, cfg)
	if err != nil {
		t.Fatalf("ExtractToString() error = %v", err)
	}

	// Check that only User types are included
	if !strings.Contains(result, "User") {
		t.Error("result should contain 'User'")
	}
	if strings.Contains(result, "Address") {
		t.Error("result should NOT contain 'Address' (not matching User* pattern)")
	}
}

// TestExtractWithExclude tests extraction with exclude patterns.
func TestExtractWithExclude(t *testing.T) {
	cfg := &Config{
		ExcludePatterns:  []string{"Admin"},
		DetectInterfaces: true,
	}
	result, err := ExtractToString([]string{"github.com/blockberries/cramberry/pkg/extract/testdata"}, cfg)
	if err != nil {
		t.Fatalf("ExtractToString() error = %v", err)
	}

	// Check that Admin is excluded
	if strings.Contains(result, "Admin") {
		t.Error("result should NOT contain 'Admin' (excluded by pattern)")
	}
	if !strings.Contains(result, "User") {
		t.Error("result should contain 'User'")
	}
}

// TestExtractor tests the extractor directly.
func TestExtractor(t *testing.T) {
	extractor := NewExtractor(DefaultConfig())
	cfg := &ExtractorConfig{
		Config:   DefaultConfig(),
		Patterns: []string{"github.com/blockberries/cramberry/pkg/extract/testdata"},
		Package:  "custompackage",
	}

	s, err := extractor.Extract(cfg)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if s == nil {
		t.Fatal("Extract() returned nil schema")
	}
	if s.Package.Name != "custompackage" {
		t.Errorf("Package name = %q, want %q", s.Package.Name, "custompackage")
	}
}

func TestParseTypeIDFromDoc(t *testing.T) {
	tests := []struct {
		doc         string
		expectID    uint32
		expectFound bool
	}{
		{"@typeID:128", 128, true},
		{"@typeID:256", 256, true},
		{"@typeID:1", 1, true},
		{"Some comment with @typeID:200 in the middle", 200, true},
		{"@cramberry:typeID=150", 150, true},
		{"Multi-line\n@typeID:300\ncomment", 300, true},
		{"No type ID annotation", 0, false},
		{"@typeID:0", 0, false}, // 0 is not valid
		{"@typeID:", 0, false},
		{"@typeID:invalid", 0, false},
		{"@typeID:-1", 0, false}, // Negative not valid
		{"", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.doc, func(t *testing.T) {
			id, found := parseTypeIDFromDoc(tt.doc)
			if found != tt.expectFound {
				t.Errorf("parseTypeIDFromDoc(%q) found = %v, want %v", tt.doc, found, tt.expectFound)
			}
			if id != tt.expectID {
				t.Errorf("parseTypeIDFromDoc(%q) id = %d, want %d", tt.doc, id, tt.expectID)
			}
		})
	}
}

func TestUintBasedEnumDetection(t *testing.T) {
	result, err := ExtractToString([]string{"github.com/blockberries/cramberry/pkg/extract/testdata"}, DefaultConfig())
	if err != nil {
		t.Fatalf("ExtractToString() error = %v", err)
	}

	// Check that both int-based and uint-based enums are detected
	if !strings.Contains(result, "Status") {
		t.Error("result should contain 'Status' enum (int-based)")
	}
	if !strings.Contains(result, "Priority") {
		t.Error("result should contain 'Priority' enum (uint8-based)")
	}

	// Verify enum values are present (using the actual Go constant names)
	if !strings.Contains(result, "StatusUnknown") || !strings.Contains(result, "StatusActive") {
		t.Error("result should contain Status enum values")
	}
	if !strings.Contains(result, "PriorityLow") || !strings.Contains(result, "PriorityHigh") {
		t.Error("result should contain Priority enum values")
	}
}

func TestFieldNumberCollisionWarning(t *testing.T) {
	// Create types with field number collision
	types := map[string]*TypeInfo{
		"pkg.Collision": {
			Name: "Collision",
			Fields: []*FieldInfo{
				{Name: "First", FieldNum: 1},
				{Name: "Second", FieldNum: 2},
				{Name: "Third", FieldNum: 1}, // Collision with First
			},
		},
	}

	builder := NewSchemaBuilder(types, nil, nil)
	_, err := builder.Build("pkg")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	warnings := builder.Warnings()
	if len(warnings) == 0 {
		t.Error("Expected at least one warning for field number collision")
	}

	// Check that the warning mentions the collision
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "field number collision") &&
			strings.Contains(w, "First") &&
			strings.Contains(w, "Third") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected warning about field number collision between First and Third, got: %v", warnings)
	}
}

func TestPlatformDependentTypeWarnings(t *testing.T) {
	types := make(map[string]*TypeInfo)
	interfaces := make(map[string]*InterfaceInfo)
	enums := make(map[string]*EnumInfo)

	builder := NewSchemaBuilder(types, interfaces, enums)

	// Verify Warnings method exists and returns empty/nil initially
	warnings := builder.Warnings()
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings initially, got %d", len(warnings))
	}

	// Build empty schema
	_, err := builder.Build("testpkg")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Verify warnings are still accessible after build (may be nil or empty)
	warnings = builder.Warnings()
	// No warnings expected for empty schema - just verify method works
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings for empty schema, got %d", len(warnings))
	}
}

func TestEmptyInterfaceDetection(t *testing.T) {
	// Test that empty interfaces are NOT included by default
	t.Run("ExcludedByDefault", func(t *testing.T) {
		cfg := DefaultConfig()
		result, err := ExtractToString([]string{"github.com/blockberries/cramberry/pkg/extract/testdata"}, cfg)
		if err != nil {
			t.Fatalf("ExtractToString() error = %v", err)
		}

		// Serializable is an empty interface - should NOT be in result by default
		if strings.Contains(result, "Serializable") {
			t.Error("result should NOT contain 'Serializable' empty interface by default")
		}

		// Person interface has methods - should be in result
		if !strings.Contains(result, "Person") {
			t.Error("result should contain 'Person' interface (has methods)")
		}
	})

	// Test that empty interfaces ARE included when configured
	t.Run("IncludedWhenConfigured", func(t *testing.T) {
		cfg := &Config{
			IncludeEmptyInterfaces: true,
			DetectInterfaces:       true,
		}
		result, err := ExtractToString([]string{"github.com/blockberries/cramberry/pkg/extract/testdata"}, cfg)
		if err != nil {
			t.Fatalf("ExtractToString() error = %v", err)
		}

		// Serializable should now be in result
		if !strings.Contains(result, "Serializable") {
			t.Error("result should contain 'Serializable' empty interface when IncludeEmptyInterfaces is true")
		}

		// Person interface should also be in result
		if !strings.Contains(result, "Person") {
			t.Error("result should contain 'Person' interface (has methods)")
		}
	})
}

func TestTypeIDAutoAssignment(t *testing.T) {
	// Create test types with and without explicit type IDs
	types := map[string]*TypeInfo{
		"pkg.Dog":  {Name: "Dog", TypeID: 128}, // Explicit type ID
		"pkg.Cat":  {Name: "Cat", TypeID: 0},   // No type ID, should be auto-assigned
		"pkg.Bird": {Name: "Bird", TypeID: 0},  // No type ID, should be auto-assigned
	}

	interfaces := map[string]*InterfaceInfo{
		"pkg.Animal": {
			Name:            "Animal",
			Implementations: []*TypeInfo{types["pkg.Dog"], types["pkg.Cat"], types["pkg.Bird"]},
		},
	}

	builder := NewSchemaBuilder(types, interfaces, nil)
	schema, err := builder.Build("pkg")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(schema.Interfaces) != 1 {
		t.Fatalf("Expected 1 interface, got %d", len(schema.Interfaces))
	}

	animal := schema.Interfaces[0]
	if len(animal.Implementations) != 3 {
		t.Fatalf("Expected 3 implementations, got %d", len(animal.Implementations))
	}

	// Check type IDs
	typeIDs := make(map[int]string)
	for _, impl := range animal.Implementations {
		typeName := impl.Type.Name
		if existingType, exists := typeIDs[impl.TypeID]; exists {
			t.Errorf("Type ID collision: %s and %s both have typeID %d", typeName, existingType, impl.TypeID)
		}
		typeIDs[impl.TypeID] = typeName

		// Dog should have explicit ID 128
		if typeName == "Dog" && impl.TypeID != 128 {
			t.Errorf("Dog should have typeID 128, got %d", impl.TypeID)
		}
	}

	// All type IDs should be >= 128 (auto-assigned start at 128)
	for id, name := range typeIDs {
		if id < 128 {
			t.Errorf("Type %s has typeID %d, but auto-assigned IDs should be >= 128", name, id)
		}
	}
}
