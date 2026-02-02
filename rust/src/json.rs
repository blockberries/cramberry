//! JSON serialization utilities for deterministic JSON encoding.
//! Provides helpers for encoding Cramberry messages to/from JSON with
//! blockchain-grade determinism.

use base64::{engine::general_purpose, Engine as _};

/// Formats an i64 as a string for JSON encoding.
/// All integers are encoded as strings to prevent JavaScript precision loss.
pub fn format_i64_to_string(value: i64) -> String {
    value.to_string()
}

/// Formats a u64 as a string for JSON encoding.
pub fn format_u64_to_string(value: u64) -> String {
    value.to_string()
}

/// Formats an i32 as a string for JSON encoding.
pub fn format_i32_to_string(value: i32) -> String {
    value.to_string()
}

/// Formats a u32 as a string for JSON encoding.
pub fn format_u32_to_string(value: u32) -> String {
    value.to_string()
}

/// Formats an i16 as a string for JSON encoding.
pub fn format_i16_to_string(value: i16) -> String {
    value.to_string()
}

/// Formats a u16 as a string for JSON encoding.
pub fn format_u16_to_string(value: u16) -> String {
    value.to_string()
}

/// Formats an i8 as a string for JSON encoding.
pub fn format_i8_to_string(value: i8) -> String {
    value.to_string()
}

/// Formats a u8 as a string for JSON encoding.
pub fn format_u8_to_string(value: u8) -> String {
    value.to_string()
}

/// Formats a f32 with appropriate precision for deterministic JSON.
/// Returns error if the value is NaN or Infinity.
pub fn format_f32(value: f32) -> Result<String, String> {
    if value.is_nan() {
        return Err("cannot encode NaN to JSON".to_string());
    }
    if value.is_infinite() {
        return Err("cannot encode Infinity to JSON".to_string());
    }
    // Normalize -0.0 to 0.0
    let normalized = if value == 0.0 { 0.0 } else { value };
    // Format with appropriate precision
    Ok(format!("{:.9}", normalized).trim_end_matches('0').trim_end_matches('.').to_string())
}

/// Formats a f64 with appropriate precision for deterministic JSON.
/// Returns error if the value is NaN or Infinity.
pub fn format_f64(value: f64) -> Result<String, String> {
    if value.is_nan() {
        return Err("cannot encode NaN to JSON".to_string());
    }
    if value.is_infinite() {
        return Err("cannot encode Infinity to JSON".to_string());
    }
    // Normalize -0.0 to 0.0
    let normalized = if value == 0.0 { 0.0 } else { value };
    // Format with appropriate precision
    Ok(format!("{:.17}", normalized).trim_end_matches('0').trim_end_matches('.').to_string())
}

/// Validates that a float is encodable to JSON.
pub fn validate_f32(value: f32) -> Result<(), String> {
    if value.is_nan() {
        return Err("cannot encode NaN to JSON".to_string());
    }
    if value.is_infinite() {
        return Err("cannot encode Infinity to JSON".to_string());
    }
    Ok(())
}

/// Validates that a double is encodable to JSON.
pub fn validate_f64(value: f64) -> Result<(), String> {
    if value.is_nan() {
        return Err("cannot encode NaN to JSON".to_string());
    }
    if value.is_infinite() {
        return Err("cannot encode Infinity to JSON".to_string());
    }
    Ok(())
}

/// Encodes bytes to base64 string (RFC 4648 standard encoding).
pub fn encode_base64(data: &[u8]) -> String {
    general_purpose::STANDARD.encode(data)
}

/// Decodes a base64 string to bytes.
pub fn decode_base64(s: &str) -> Result<Vec<u8>, String> {
    general_purpose::STANDARD
        .decode(s)
        .map_err(|e| format!("invalid base64: {}", e))
}

/// Parses an i64 from a JSON value (string or number).
pub fn parse_i64_from_json(value: &serde_json::Value) -> Result<i64, String> {
    match value {
        serde_json::Value::String(s) => s.parse::<i64>()
            .map_err(|e| format!("failed to parse i64: {}", e)),
        serde_json::Value::Number(n) => n.as_i64()
            .ok_or_else(|| "number out of i64 range".to_string()),
        _ => Err("expected string or number for i64".to_string()),
    }
}

/// Parses a u64 from a JSON value (string or number).
pub fn parse_u64_from_json(value: &serde_json::Value) -> Result<u64, String> {
    match value {
        serde_json::Value::String(s) => s.parse::<u64>()
            .map_err(|e| format!("failed to parse u64: {}", e)),
        serde_json::Value::Number(n) => n.as_u64()
            .ok_or_else(|| "number out of u64 range".to_string()),
        _ => Err("expected string or number for u64".to_string()),
    }
}

/// Parses an i32 from a JSON value (string or number).
pub fn parse_i32_from_json(value: &serde_json::Value) -> Result<i32, String> {
    match value {
        serde_json::Value::String(s) => s.parse::<i32>()
            .map_err(|e| format!("failed to parse i32: {}", e)),
        serde_json::Value::Number(n) => n.as_i64()
            .ok_or_else(|| "not a number".to_string())
            .and_then(|v| {
                i32::try_from(v).map_err(|e| format!("i32 overflow: {}", e))
            }),
        _ => Err("expected string or number for i32".to_string()),
    }
}

/// Parses a u32 from a JSON value (string or number).
pub fn parse_u32_from_json(value: &serde_json::Value) -> Result<u32, String> {
    match value {
        serde_json::Value::String(s) => s.parse::<u32>()
            .map_err(|e| format!("failed to parse u32: {}", e)),
        serde_json::Value::Number(n) => n.as_u64()
            .ok_or_else(|| "not a number".to_string())
            .and_then(|v| {
                u32::try_from(v).map_err(|e| format!("u32 overflow: {}", e))
            }),
        _ => Err("expected string or number for u32".to_string()),
    }
}

/// Sorts map keys lexicographically by UTF-8 byte order.
/// This ensures deterministic JSON output for maps.
pub fn sort_map_keys_lexicographic(keys: &mut Vec<String>) {
    keys.sort();
}

/// Escapes a string for safe inclusion in JSON.
/// Uses serde_json for correct escaping.
pub fn escape_json_string(s: &str) -> String {
    serde_json::to_string(s).unwrap_or_else(|_| "\"\"".to_string())
}

/// Helper to build JSON strings efficiently.
pub struct JsonBuilder {
    parts: Vec<String>,
}

impl JsonBuilder {
    pub fn new() -> Self {
        JsonBuilder { parts: Vec::new() }
    }

    pub fn push(&mut self, s: &str) {
        self.parts.push(s.to_string());
    }

    pub fn push_string(&mut self, s: String) {
        self.parts.push(s);
    }

    pub fn build(self) -> String {
        self.parts.join("")
    }
}

impl Default for JsonBuilder {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_format_i64_to_string() {
        assert_eq!(format_i64_to_string(0), "0");
        assert_eq!(format_i64_to_string(123), "123");
        assert_eq!(format_i64_to_string(-456), "-456");
        assert_eq!(format_i64_to_string(i64::MAX), "9223372036854775807");
        assert_eq!(format_i64_to_string(i64::MIN), "-9223372036854775808");
    }

    #[test]
    fn test_format_u64_to_string() {
        assert_eq!(format_u64_to_string(0), "0");
        assert_eq!(format_u64_to_string(123), "123");
        assert_eq!(format_u64_to_string(u64::MAX), "18446744073709551615");
    }

    #[test]
    fn test_format_f32() {
        assert_eq!(format_f32(0.0).unwrap(), "0");
        // f32 precision
        assert_eq!(format_f32(3.14159).unwrap(), "3.141590118");
        assert!(format_f32(f32::NAN).is_err());
        assert!(format_f32(f32::INFINITY).is_err());
        assert!(format_f32(f32::NEG_INFINITY).is_err());
    }

    #[test]
    fn test_format_f64() {
        assert_eq!(format_f64(0.0).unwrap(), "0");
        // f64 precision
        assert_eq!(format_f64(2.718281828459045).unwrap(), "2.71828182845904509");
        assert!(format_f64(f64::NAN).is_err());
        assert!(format_f64(f64::INFINITY).is_err());
        assert!(format_f64(f64::NEG_INFINITY).is_err());
    }

    #[test]
    fn test_encode_decode_base64() {
        let data = vec![0xde, 0xad, 0xbe, 0xef];
        let encoded = encode_base64(&data);
        assert_eq!(encoded, "3q2+7w==");

        let decoded = decode_base64(&encoded).unwrap();
        assert_eq!(decoded, data);
    }

    #[test]
    fn test_parse_i64_from_json() {
        let value = serde_json::json!("123");
        assert_eq!(parse_i64_from_json(&value).unwrap(), 123);

        let value = serde_json::json!(456);
        assert_eq!(parse_i64_from_json(&value).unwrap(), 456);

        let value = serde_json::json!("-789");
        assert_eq!(parse_i64_from_json(&value).unwrap(), -789);
    }

    #[test]
    fn test_parse_u64_from_json() {
        let value = serde_json::json!("123");
        assert_eq!(parse_u64_from_json(&value).unwrap(), 123);

        let value = serde_json::json!(456);
        assert_eq!(parse_u64_from_json(&value).unwrap(), 456);
    }

    #[test]
    fn test_sort_map_keys_lexicographic() {
        let mut keys = vec!["zebra".to_string(), "apple".to_string(), "banana".to_string()];
        sort_map_keys_lexicographic(&mut keys);
        assert_eq!(keys, vec!["apple", "banana", "zebra"]);
    }

    #[test]
    fn test_escape_json_string() {
        assert_eq!(escape_json_string("hello"), "\"hello\"");
        assert_eq!(escape_json_string("say \"hello\""), "\"say \\\"hello\\\"\"");
        assert_eq!(escape_json_string("line1\nline2"), "\"line1\\nline2\"");
    }
}
