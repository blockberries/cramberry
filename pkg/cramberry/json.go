package cramberry

import (
	"encoding/base64"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// JSON encoding/decoding helper functions for deterministic JSON serialization.
// These are used by generated code to ensure consistent, blockchain-grade JSON output.

// FormatInt64ToString converts an int64 to a string for JSON encoding.
// All integers are encoded as strings to prevent JavaScript precision loss (>2^53).
func FormatInt64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

// FormatUint64ToString converts a uint64 to a string for JSON encoding.
func FormatUint64ToString(v uint64) string {
	return strconv.FormatUint(v, 10)
}

// FormatFloat32 formats a float32 with 9 significant digits for deterministic JSON.
// Returns error if the value is NaN or Infinity.
func FormatFloat32(v float32) (string, error) {
	if math.IsNaN(float64(v)) {
		return "", fmt.Errorf("cannot encode NaN to JSON")
	}
	if math.IsInf(float64(v), 0) {
		return "", fmt.Errorf("cannot encode Infinity to JSON")
	}
	// Normalize -0.0 to 0.0
	if v == 0 {
		v = 0
	}
	// Format with 9 significant digits, remove trailing zeros
	return strconv.FormatFloat(float64(v), 'g', 9, 32), nil
}

// FormatFloat64 formats a float64 with 17 significant digits for deterministic JSON.
// Returns error if the value is NaN or Infinity.
func FormatFloat64(v float64) (string, error) {
	if math.IsNaN(v) {
		return "", fmt.Errorf("cannot encode NaN to JSON")
	}
	if math.IsInf(v, 0) {
		return "", fmt.Errorf("cannot encode Infinity to JSON")
	}
	// Normalize -0.0 to 0.0
	if v == 0 {
		v = 0
	}
	// Format with 17 significant digits, remove trailing zeros
	return strconv.FormatFloat(v, 'g', 17, 64), nil
}

// ValidateFloat32 checks if a float32 is valid for JSON encoding.
func ValidateFloat32(v float32) error {
	if math.IsNaN(float64(v)) {
		return fmt.Errorf("cannot encode NaN to JSON")
	}
	if math.IsInf(float64(v), 0) {
		return fmt.Errorf("cannot encode Infinity to JSON")
	}
	return nil
}

// ValidateFloat64 checks if a float64 is valid for JSON encoding.
func ValidateFloat64(v float64) error {
	if math.IsNaN(v) {
		return fmt.Errorf("cannot encode NaN to JSON")
	}
	if math.IsInf(v, 0) {
		return fmt.Errorf("cannot encode Infinity to JSON")
	}
	return nil
}

// EncodeBase64 encodes a byte slice to base64 string (RFC 4648 standard encoding).
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes a base64 string to bytes.
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// ParseInt64FromString parses an int64 from a JSON string.
// Accepts both string and numeric JSON representations.
func ParseInt64FromString(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ParseUint64FromString parses a uint64 from a JSON string.
func ParseUint64FromString(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// SortMapKeysLexicographic sorts map keys lexicographically by UTF-8 byte order.
// This ensures deterministic JSON output for maps.
func SortMapKeysLexicographic(keys []string) []string {
	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)
	return sorted
}

// EscapeJSONString escapes a string for safe inclusion in JSON.
func EscapeJSONString(s string) string {
	var buf strings.Builder
	buf.Grow(len(s) + 2)
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return buf.String()
}
