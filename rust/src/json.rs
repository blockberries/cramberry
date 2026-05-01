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

/// Formats a f32 with 9 significant digits for deterministic JSON.
/// Matches the output of Go's `strconv.FormatFloat(v, 'g', 9, 32)`.
/// Returns error if the value is NaN or Infinity.
pub fn format_f32(value: f32) -> Result<String, String> {
    if value.is_nan() {
        return Err("cannot encode NaN to JSON".to_string());
    }
    if value.is_infinite() {
        return Err("cannot encode Infinity to JSON".to_string());
    }
    if value == 0.0 {
        return Ok("0".to_string());
    }
    // Widening to f64 preserves the f32 bit pattern exactly, so the 9-digit
    // round in format_go_g produces the same output as Go's '%.9g' on a f32.
    Ok(format_go_g(value as f64, 9))
}

/// Formats a f64 with 17 significant digits for deterministic JSON.
/// Matches the output of Go's `strconv.FormatFloat(v, 'g', 17, 64)`.
/// Returns error if the value is NaN or Infinity.
pub fn format_f64(value: f64) -> Result<String, String> {
    if value.is_nan() {
        return Err("cannot encode NaN to JSON".to_string());
    }
    if value.is_infinite() {
        return Err("cannot encode Infinity to JSON".to_string());
    }
    if value == 0.0 {
        return Ok("0".to_string());
    }
    Ok(format_go_g(value, 17))
}

/// Formats a non-zero finite `f64` using Go's `strconv` 'g' verb with the
/// given precision (number of significant digits).
///
///   - Decimal form when `-4 <= exponent < precision`.
///   - Otherwise scientific: `mantissa "e" sign digits` with the exponent
///     always signed and at least two digits.
///   - Trailing zeros in the fractional part are stripped.
fn format_go_g(value: f64, precision: usize) -> String {
    // {:.p$e} formats with `p` digits after the decimal point — i.e. p+1
    // significant digits total — using IEEE round-half-to-even, matching Go.
    let raw = format!("{:.*e}", precision - 1, value);
    let (mantissa_raw, exp_str) = match raw.find('e') {
        Some(idx) => (&raw[..idx], &raw[idx + 1..]),
        None => return raw,
    };
    let exponent: i32 = exp_str.parse().unwrap_or(0);

    let mantissa = strip_fractional_trailing_zeros(mantissa_raw);

    if exponent >= -4 && exponent < precision as i32 {
        return decimal_from_mantissa_exp(&mantissa, exponent);
    }
    let sign = if exponent >= 0 { '+' } else { '-' };
    let abs_exp = exponent.unsigned_abs();
    if abs_exp < 10 {
        format!("{}e{}0{}", mantissa, sign, abs_exp)
    } else {
        format!("{}e{}{}", mantissa, sign, abs_exp)
    }
}

fn strip_fractional_trailing_zeros(s: &str) -> String {
    if !s.contains('.') {
        return s.to_string();
    }
    let trimmed = s.trim_end_matches('0').trim_end_matches('.');
    trimmed.to_string()
}

/// Renders mantissa * 10^exp in plain decimal form, stripping trailing zeros
/// from any fractional part.
fn decimal_from_mantissa_exp(mantissa: &str, exp: i32) -> String {
    let (sign, mantissa) = if let Some(rest) = mantissa.strip_prefix('-') {
        ("-", rest)
    } else {
        ("", mantissa)
    };

    let (digits, integer_len) = match mantissa.find('.') {
        Some(idx) => {
            let mut s = String::with_capacity(mantissa.len() - 1);
            s.push_str(&mantissa[..idx]);
            s.push_str(&mantissa[idx + 1..]);
            (s, idx as i32)
        }
        None => (mantissa.to_string(), mantissa.len() as i32),
    };

    let new_integer_len = integer_len + exp;

    if new_integer_len <= 0 {
        let padding = "0".repeat((-new_integer_len) as usize);
        let raw = format!("0.{}{}", padding, digits);
        let trimmed = raw.trim_end_matches('0').trim_end_matches('.');
        return format!("{}{}", sign, trimmed);
    }
    if (new_integer_len as usize) >= digits.len() {
        let padding = "0".repeat((new_integer_len as usize) - digits.len());
        return format!("{}{}{}", sign, digits, padding);
    }
    let int_part = &digits[..new_integer_len as usize];
    let frac_part = digits[new_integer_len as usize..].trim_end_matches('0');
    if frac_part.is_empty() {
        format!("{}{}", sign, int_part)
    } else {
        format!("{}{}.{}", sign, int_part, frac_part)
    }
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
        serde_json::Value::String(s) => s
            .parse::<i64>()
            .map_err(|e| format!("failed to parse i64: {}", e)),
        serde_json::Value::Number(n) => n
            .as_i64()
            .ok_or_else(|| "number out of i64 range".to_string()),
        _ => Err("expected string or number for i64".to_string()),
    }
}

/// Parses a u64 from a JSON value (string or number).
pub fn parse_u64_from_json(value: &serde_json::Value) -> Result<u64, String> {
    match value {
        serde_json::Value::String(s) => s
            .parse::<u64>()
            .map_err(|e| format!("failed to parse u64: {}", e)),
        serde_json::Value::Number(n) => n
            .as_u64()
            .ok_or_else(|| "number out of u64 range".to_string()),
        _ => Err("expected string or number for u64".to_string()),
    }
}

/// Parses an i32 from a JSON value (string or number).
pub fn parse_i32_from_json(value: &serde_json::Value) -> Result<i32, String> {
    match value {
        serde_json::Value::String(s) => s
            .parse::<i32>()
            .map_err(|e| format!("failed to parse i32: {}", e)),
        serde_json::Value::Number(n) => n
            .as_i64()
            .ok_or_else(|| "not a number".to_string())
            .and_then(|v| i32::try_from(v).map_err(|e| format!("i32 overflow: {}", e))),
        _ => Err("expected string or number for i32".to_string()),
    }
}

/// Parses a u32 from a JSON value (string or number).
pub fn parse_u32_from_json(value: &serde_json::Value) -> Result<u32, String> {
    match value {
        serde_json::Value::String(s) => s
            .parse::<u32>()
            .map_err(|e| format!("failed to parse u32: {}", e)),
        serde_json::Value::Number(n) => n
            .as_u64()
            .ok_or_else(|| "not a number".to_string())
            .and_then(|v| u32::try_from(v).map_err(|e| format!("u32 overflow: {}", e))),
        _ => Err("expected string or number for u32".to_string()),
    }
}

/// Sorts map keys lexicographically by UTF-8 byte order.
/// This ensures deterministic JSON output for maps.
///
/// Accepts `&mut [String]` so the codegen-emitted callers passing
/// `&mut keys` (where `keys: Vec<String>`) auto-deref without requiring
/// the helper to own the vector type.
pub fn sort_map_keys_lexicographic(keys: &mut [String]) {
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
    #[allow(clippy::approx_constant)] // literals are reference values matched against Go output, not π/e
    fn test_format_f32() {
        // Reference values come from pkg/cramberry/json_test.go::TestFormatFloat32.
        // They must match Go's strconv.FormatFloat(v, 'g', 9, 32) byte-for-byte.
        assert_eq!(format_f32(0.0).unwrap(), "0");
        assert_eq!(format_f32(-0.0).unwrap(), "0");
        assert_eq!(format_f32(3.14159).unwrap(), "3.14159012");
        assert_eq!(format_f32(-2.71828).unwrap(), "-2.71828008");
        assert_eq!(format_f32(1.23e10).unwrap(), "1.23000003e+10");
        assert_eq!(format_f32(0.0000001).unwrap(), "1.00000001e-07");
        assert!(format_f32(f32::NAN).is_err());
        assert!(format_f32(f32::INFINITY).is_err());
        assert!(format_f32(f32::NEG_INFINITY).is_err());
    }

    #[test]
    #[allow(clippy::approx_constant)] // literals are reference values matched against Go output, not π/e
    fn test_format_f64() {
        // Reference values come from pkg/cramberry/json_test.go::TestFormatFloat64.
        assert_eq!(format_f64(0.0).unwrap(), "0");
        assert_eq!(format_f64(-0.0).unwrap(), "0");
        assert_eq!(format_f64(2.718281828459045).unwrap(), "2.7182818284590451");
        assert_eq!(
            format_f64(-3.141592653589793).unwrap(),
            "-3.1415926535897931"
        );
        assert_eq!(format_f64(1.23e100).unwrap(), "1.2300000000000001e+100");
        assert_eq!(format_f64(1e-100).unwrap(), "1e-100");
        assert_eq!(format_f64(1e20).unwrap(), "1e+20");
        assert_eq!(format_f64(1.5).unwrap(), "1.5");
        assert_eq!(format_f64(0.0001).unwrap(), "0.0001");
        assert_eq!(format_f64(1e-7).unwrap(), "9.9999999999999995e-08");
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
        let mut keys = vec![
            "zebra".to_string(),
            "apple".to_string(),
            "banana".to_string(),
        ];
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
