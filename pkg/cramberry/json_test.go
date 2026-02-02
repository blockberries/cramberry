package cramberry

import (
	"math"
	"testing"
)

func TestFormatInt64ToString(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{-456, "-456"},
		{9223372036854775807, "9223372036854775807"},   // MaxInt64
		{-9223372036854775808, "-9223372036854775808"}, // MinInt64
	}

	for _, tt := range tests {
		result := FormatInt64ToString(tt.input)
		if result != tt.expected {
			t.Errorf("FormatInt64ToString(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatUint64ToString(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{18446744073709551615, "18446744073709551615"}, // MaxUint64
	}

	for _, tt := range tests {
		result := FormatUint64ToString(tt.input)
		if result != tt.expected {
			t.Errorf("FormatUint64ToString(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatFloat32(t *testing.T) {
	tests := []struct {
		name      string
		input     float32
		wantErr   bool
		wantValue string
	}{
		{"zero", 0.0, false, "0"},
		{"positive", 3.14159, false, "3.14159012"},   // float32 precision
		{"negative", -2.71828, false, "-2.71828008"}, // float32 precision
		{"scientific", 1.23e10, false, "1.23000003e+10"},
		{"small", 0.0000001, false, "1.00000001e-07"},
		{"NaN", float32(math.NaN()), true, ""},
		{"Inf", float32(math.Inf(1)), true, ""},
		{"NegInf", float32(math.Inf(-1)), true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatFloat32(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("FormatFloat32(%v) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("FormatFloat32(%v) unexpected error: %v", tt.input, err)
				}
				if result != tt.wantValue {
					t.Errorf("FormatFloat32(%v) = %s, want %s", tt.input, result, tt.wantValue)
				}
			}
		})
	}
}

func TestFormatFloat32NegativeZero(t *testing.T) {
	// Test negative zero separately since Go treats -0.0 literal as 0.0
	negZero := float32(math.Copysign(0, -1))
	result, err := FormatFloat32(negZero)
	if err != nil {
		t.Errorf("FormatFloat32(-0.0) unexpected error: %v", err)
	}
	if result != "0" {
		t.Errorf("FormatFloat32(-0.0) = %s, want 0", result)
	}
}

func TestFormatFloat64(t *testing.T) {
	tests := []struct {
		name      string
		input     float64
		wantErr   bool
		wantValue string
	}{
		{"zero", 0.0, false, "0"},
		{"positive", 2.718281828459045, false, "2.7182818284590451"},   // float64 precision
		{"negative", -3.141592653589793, false, "-3.1415926535897931"}, // float64 precision
		{"scientific", 1.23e100, false, "1.2300000000000001e+100"},
		{"small", 1e-100, false, "1e-100"},
		{"NaN", math.NaN(), true, ""},
		{"Inf", math.Inf(1), true, ""},
		{"NegInf", math.Inf(-1), true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatFloat64(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("FormatFloat64(%v) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("FormatFloat64(%v) unexpected error: %v", tt.input, err)
				}
				if result != tt.wantValue {
					t.Errorf("FormatFloat64(%v) = %s, want %s", tt.input, result, tt.wantValue)
				}
			}
		})
	}
}

func TestFormatFloat64NegativeZero(t *testing.T) {
	// Test negative zero separately since Go treats -0.0 literal as 0.0
	negZero := math.Copysign(0, -1)
	result, err := FormatFloat64(negZero)
	if err != nil {
		t.Errorf("FormatFloat64(-0.0) unexpected error: %v", err)
	}
	if result != "0" {
		t.Errorf("FormatFloat64(-0.0) = %s, want 0", result)
	}
}

func TestValidateFloat32(t *testing.T) {
	tests := []struct {
		name    string
		input   float32
		wantErr bool
	}{
		{"valid positive", 3.14, false},
		{"valid negative", -2.71, false},
		{"valid zero", 0.0, false},
		{"NaN", float32(math.NaN()), true},
		{"Inf", float32(math.Inf(1)), true},
		{"NegInf", float32(math.Inf(-1)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFloat32(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateFloat32(%v) expected error, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateFloat32(%v) unexpected error: %v", tt.input, err)
			}
		})
	}
}

func TestValidateFloat64(t *testing.T) {
	tests := []struct {
		name    string
		input   float64
		wantErr bool
	}{
		{"valid positive", 3.14159, false},
		{"valid negative", -2.71828, false},
		{"valid zero", 0.0, false},
		{"NaN", math.NaN(), true},
		{"Inf", math.Inf(1), true},
		{"NegInf", math.Inf(-1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFloat64(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateFloat64(%v) expected error, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateFloat64(%v) unexpected error: %v", tt.input, err)
			}
		})
	}
}

func TestEncodeDecodeBase64(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"empty", []byte{}, ""},
		{"simple", []byte{0xde, 0xad, 0xbe, 0xef}, "3q2+7w=="},
		{"hello", []byte("hello"), "aGVsbG8="},
		{"unicode", []byte("hello, 世界"), "aGVsbG8sIOS4lueVjA=="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeBase64(tt.input)
			if encoded != tt.want {
				t.Errorf("EncodeBase64(%v) = %s, want %s", tt.input, encoded, tt.want)
			}

			// Test round-trip
			decoded, err := DecodeBase64(encoded)
			if err != nil {
				t.Errorf("DecodeBase64(%s) unexpected error: %v", encoded, err)
			}
			if string(decoded) != string(tt.input) {
				t.Errorf("DecodeBase64(EncodeBase64(%v)) = %v, want %v", tt.input, decoded, tt.input)
			}
		})
	}
}

func TestDecodeBase64Invalid(t *testing.T) {
	tests := []string{
		"!!!invalid!!!",
		"not base64",
		"aGVsbG8", // Missing padding
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			_, err := DecodeBase64(tt)
			if err == nil {
				t.Errorf("DecodeBase64(%s) expected error, got nil", tt)
			}
		})
	}
}

func TestParseIntFromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		parseInt  func(string) (int64, error)
		expected  int64
		expectErr bool
	}{
		{"valid positive", "123", ParseInt64FromString, 123, false},
		{"valid negative", "-456", ParseInt64FromString, -456, false},
		{"zero", "0", ParseInt64FromString, 0, false},
		{"max int64", "9223372036854775807", ParseInt64FromString, 9223372036854775807, false},
		{"min int64", "-9223372036854775808", ParseInt64FromString, -9223372036854775808, false},
		{"invalid", "not a number", ParseInt64FromString, 0, true},
		{"overflow", "9223372036854775808", ParseInt64FromString, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.parseInt(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error parsing %s, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error parsing %s: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("parse(%s) = %d, want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestParseUintFromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  uint64
		expectErr bool
	}{
		{"valid", "123", 123, false},
		{"zero", "0", 0, false},
		{"max uint64", "18446744073709551615", 18446744073709551615, false},
		{"invalid", "not a number", 0, true},
		{"negative", "-123", 0, true},
		{"overflow", "18446744073709551616", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseUint64FromString(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error parsing %s, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error parsing %s: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParseUint64FromString(%s) = %d, want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestSortMapKeysLexicographic(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			"alphabetical",
			[]string{"zebra", "apple", "banana"},
			[]string{"apple", "banana", "zebra"},
		},
		{
			"numeric strings",
			[]string{"10", "2", "1"},
			[]string{"1", "10", "2"}, // Lexicographic, not numeric
		},
		{
			"mixed",
			[]string{"env", "version", "app", "debug"},
			[]string{"app", "debug", "env", "version"},
		},
		{
			"unicode",
			[]string{"世界", "hello", "мир"},
			[]string{"hello", "мир", "世界"}, // UTF-8 byte order
		},
		{
			"empty",
			[]string{},
			[]string{},
		},
		{
			"single",
			[]string{"only"},
			[]string{"only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortMapKeysLexicographic(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SortMapKeysLexicographic(%v) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if i < len(tt.expected) && result[i] != tt.expected[i] {
					t.Errorf("SortMapKeysLexicographic(%v)[%d] = %s, want %s", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestEscapeJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", `"hello"`},
		{"with quotes", `say "hello"`, `"say \"hello\""`},
		{"with backslash", `path\to\file`, `"path\\to\\file"`},
		{"with newline", "line1\nline2", `"line1\nline2"`},
		{"with tab", "col1\tcol2", `"col1\tcol2"`},
		{"with carriage return", "line1\rline2", `"line1\rline2"`},
		{"with backspace", "back\bspace", `"back\bspace"`},
		{"with form feed", "form\ffeed", `"form\ffeed"`},
		{"control char", "test\x01", `"test\u0001"`},
		{"empty", "", `""`},
		{"unicode", "hello, 世界", `"hello, 世界"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeJSONString(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeJSONString(%q) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestJSONStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", `hello`},
		{"with quotes", `say "hello"`, `say \"hello\"`},
		{"with newline", "line1\nline2", `line1\nline2`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONStringValue(tt.input)
			if result != tt.expected {
				t.Errorf("JSONStringValue(%q) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
