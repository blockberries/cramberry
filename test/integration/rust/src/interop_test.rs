//! Cross-runtime interoperability tests for Rust.
//!
//! These tests verify that Rust runtime produces identical binary
//! encodings to Go and can decode Go-generated golden files.

use cramberry::{Reader, Result, WireType, Writer};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

use crate::interop::*;

// Golden bytes are produced by the Go reflection marshaller in
// testdata/golden/. cargo test runs from the integration crate dir
// (test/integration/rust/), so the relative path goes up three levels
// to the project root.
const GOLDEN_DIR: &str = "../../../testdata/golden";

// Test data matching Go's TestData
#[allow(clippy::approx_constant)] // literals match Go's TestData fixtures, not π/e
fn test_scalar_types() -> ScalarTypes {
    ScalarTypes {
        bool_val: true,
        int32_val: -42,
        int64_val: -9223372036854775807,
        uint32_val: 4294967295,
        uint64_val: 18446744073709551615,
        float32_val: 3.14159,
        float64_val: 2.718281828459045,
        string_val: "hello, cramberry!".to_string(),
        bytes_val: vec![0xde, 0xad, 0xbe, 0xef],
    }
}

fn test_nested_message() -> NestedMessage {
    NestedMessage {
        name: "nested".to_string(),
        value: 123,
    }
}

fn test_all_field_numbers() -> AllFieldNumbers {
    AllFieldNumbers {
        field_1: 100,
        field_15: 1500,
        field_16: 1600,
        field_127: 12700,
        field_128: 12800,
        field_1000: 100000,
    }
}

fn test_complex_types() -> ComplexTypes {
    let mut string_int_map = HashMap::new();
    string_int_map.insert("one".to_string(), 1);
    string_int_map.insert("two".to_string(), 2);
    string_int_map.insert("three".to_string(), 3);

    let mut int_string_map = HashMap::new();
    int_string_map.insert(1, "one".to_string());
    int_string_map.insert(2, "two".to_string());
    int_string_map.insert(3, "three".to_string());

    ComplexTypes {
        status: Status::Active,
        optional_nested: Some(Box::new(NestedMessage {
            name: "optional".to_string(),
            value: 456,
        })),
        required_nested: NestedMessage {
            name: "required".to_string(),
            value: 789,
        },
        nested_list: vec![
            NestedMessage {
                name: "first".to_string(),
                value: 1,
            },
            NestedMessage {
                name: "second".to_string(),
                value: 2,
            },
        ],
        string_int_map,
        int_string_map,
    }
}

fn test_edge_cases() -> EdgeCases {
    EdgeCases {
        zero_int: 0,
        negative_one: -1,
        max_int32: i32::MAX,
        min_int32: i32::MIN,
        max_int64: i64::MAX,
        min_int64: i64::MIN,
        max_uint32: u32::MAX,
        max_uint64: u64::MAX,
        empty_string: "".to_string(),
        unicode_string: "Hello, 世界! 🎉".to_string(),
        empty_bytes: vec![],
    }
}

// Encoder functions - format uses an end marker (not a field count)
fn encode_nested_message(writer: &mut Writer, msg: &NestedMessage) -> Result<()> {
    // Field 1: name
    writer.write_tag(1, WireType::Bytes)?;
    writer.write_string(&msg.name)?;

    // Field 2: value - uses SVarint wire type
    writer.write_tag(2, WireType::SVarint)?;
    writer.write_svarint(msg.value)?;

    // End marker
    writer.write_end_marker()?;

    Ok(())
}

fn decode_nested_message(reader: &mut Reader) -> Result<NestedMessage> {
    let mut name = String::new();
    let mut value = 0i32;

    while reader.has_more() {
        let tag = reader.read_tag()?;
        if Reader::is_end_marker(&tag) {
            break;
        }
        match tag.field_number {
            1 => name = reader.read_string()?.to_string(),
            2 => value = reader.read_svarint()?,
            _ => reader.skip_value(tag.wire_type)?,
        }
    }

    Ok(NestedMessage { name, value })
}

fn encode_scalar_types(writer: &mut Writer, msg: &ScalarTypes) -> Result<()> {
    // No field count prefix; messages terminate with an end marker.

    // Field 1: bool_val
    writer.write_tag(1, WireType::Varint)?;
    writer.write_bool(msg.bool_val)?;

    // Field 2: int32_val - uses SVarint wire type
    writer.write_tag(2, WireType::SVarint)?;
    writer.write_svarint(msg.int32_val)?;

    // Field 3: int64_val - uses SVarint wire type
    writer.write_tag(3, WireType::SVarint)?;
    writer.write_svarint64(msg.int64_val)?;

    // Field 4: uint32_val
    writer.write_tag(4, WireType::Varint)?;
    writer.write_varint(msg.uint32_val)?;

    // Field 5: uint64_val
    writer.write_tag(5, WireType::Varint)?;
    writer.write_varint64(msg.uint64_val)?;

    // Field 6: float32_val
    writer.write_tag(6, WireType::Fixed32)?;
    writer.write_float32(msg.float32_val)?;

    // Field 7: float64_val
    writer.write_tag(7, WireType::Fixed64)?;
    writer.write_float64(msg.float64_val)?;

    // Field 8: string_val
    writer.write_tag(8, WireType::Bytes)?;
    writer.write_string(&msg.string_val)?;

    // Field 9: bytes_val
    writer.write_tag(9, WireType::Bytes)?;
    writer.write_length_prefixed_bytes(&msg.bytes_val)?;

    // End marker
    writer.write_end_marker()?;

    Ok(())
}

fn decode_scalar_types(reader: &mut Reader) -> Result<ScalarTypes> {
    let mut result = ScalarTypes {
        bool_val: false,
        int32_val: 0,
        int64_val: 0,
        uint32_val: 0,
        uint64_val: 0,
        float32_val: 0.0,
        float64_val: 0.0,
        string_val: String::new(),
        bytes_val: vec![],
    };

    while reader.has_more() {
        let tag = reader.read_tag()?;
        if Reader::is_end_marker(&tag) {
            break;
        }
        match tag.field_number {
            1 => result.bool_val = reader.read_bool()?,
            2 => result.int32_val = reader.read_svarint()?,
            3 => result.int64_val = reader.read_svarint64()?,
            4 => result.uint32_val = reader.read_varint()?,
            5 => result.uint64_val = reader.read_varint64()?,
            6 => result.float32_val = reader.read_float32()?,
            7 => result.float64_val = reader.read_float64()?,
            8 => result.string_val = reader.read_string()?.to_string(),
            9 => result.bytes_val = reader.read_length_prefixed_bytes()?.to_vec(),
            _ => reader.skip_value(tag.wire_type)?,
        }
    }

    Ok(result)
}

fn encode_all_field_numbers(writer: &mut Writer, msg: &AllFieldNumbers) -> Result<()> {
    // No field count prefix; messages terminate with an end marker.
    // All int32 fields use SVarint wire type

    writer.write_tag(1, WireType::SVarint)?;
    writer.write_svarint(msg.field_1)?;

    writer.write_tag(15, WireType::SVarint)?;
    writer.write_svarint(msg.field_15)?;

    writer.write_tag(16, WireType::SVarint)?;
    writer.write_svarint(msg.field_16)?;

    writer.write_tag(127, WireType::SVarint)?;
    writer.write_svarint(msg.field_127)?;

    writer.write_tag(128, WireType::SVarint)?;
    writer.write_svarint(msg.field_128)?;

    writer.write_tag(1000, WireType::SVarint)?;
    writer.write_svarint(msg.field_1000)?;

    // End marker
    writer.write_end_marker()?;

    Ok(())
}

fn decode_all_field_numbers(reader: &mut Reader) -> Result<AllFieldNumbers> {
    let mut result = AllFieldNumbers {
        field_1: 0,
        field_15: 0,
        field_16: 0,
        field_127: 0,
        field_128: 0,
        field_1000: 0,
    };

    while reader.has_more() {
        let tag = reader.read_tag()?;
        if Reader::is_end_marker(&tag) {
            break;
        }
        match tag.field_number {
            1 => result.field_1 = reader.read_svarint()?,
            15 => result.field_15 = reader.read_svarint()?,
            16 => result.field_16 = reader.read_svarint()?,
            127 => result.field_127 = reader.read_svarint()?,
            128 => result.field_128 = reader.read_svarint()?,
            1000 => result.field_1000 = reader.read_svarint()?,
            _ => reader.skip_value(tag.wire_type)?,
        }
    }

    Ok(result)
}

// Encode a NestedMessage into a fresh sub-buffer so it can be wrapped
// as length-prefixed bytes by the parent encoder. This mirrors the Go
// codegen's BeginMessage / EndMessage pattern.
fn encode_nested_to_bytes(msg: &NestedMessage) -> Result<Vec<u8>> {
    let mut sub = Writer::new();
    encode_nested_message(&mut sub, msg)?;
    Ok(sub.into_bytes())
}

fn encode_complex_types(writer: &mut Writer, msg: &ComplexTypes) -> Result<()> {
    // Field 1: status (always emitted; codegen does not check zero-value
    // because enums have a default variant).
    writer.write_tag(1, WireType::SVarint)?;
    writer.write_svarint(msg.status as i32)?;

    // Field 2: optional_nested (length-prefixed message body, only if set).
    if let Some(nested) = &msg.optional_nested {
        writer.write_tag(2, WireType::Bytes)?;
        let body = encode_nested_to_bytes(nested)?;
        writer.write_length_prefixed_bytes(&body)?;
    }

    // Field 3: required_nested (always emitted, length-prefixed body).
    writer.write_tag(3, WireType::Bytes)?;
    let body = encode_nested_to_bytes(&msg.required_nested)?;
    writer.write_length_prefixed_bytes(&body)?;

    // Field 4: nested_list (length-prefixed body containing count + elements).
    if !msg.nested_list.is_empty() {
        writer.write_tag(4, WireType::Bytes)?;
        let mut sub = Writer::new();
        sub.write_varint64(msg.nested_list.len() as u64)?;
        for n in &msg.nested_list {
            encode_nested_message(&mut sub, n)?;
        }
        writer.write_length_prefixed_bytes(&sub.into_bytes())?;
    }

    // Field 5: string_int_map (sorted by UTF-8 byte order).
    if !msg.string_int_map.is_empty() {
        writer.write_tag(5, WireType::Bytes)?;
        let mut sub = Writer::new();
        sub.write_varint64(msg.string_int_map.len() as u64)?;
        let mut keys: Vec<&String> = msg.string_int_map.keys().collect();
        keys.sort();
        for k in keys {
            sub.write_string(k)?;
            sub.write_int32(msg.string_int_map[k])?;
        }
        writer.write_length_prefixed_bytes(&sub.into_bytes())?;
    }

    // Field 6: int_string_map (sorted by numeric key).
    if !msg.int_string_map.is_empty() {
        writer.write_tag(6, WireType::Bytes)?;
        let mut sub = Writer::new();
        sub.write_varint64(msg.int_string_map.len() as u64)?;
        let mut keys: Vec<&i32> = msg.int_string_map.keys().collect();
        keys.sort();
        for k in keys {
            sub.write_int32(*k)?;
            sub.write_string(&msg.int_string_map[k])?;
        }
        writer.write_length_prefixed_bytes(&sub.into_bytes())?;
    }

    writer.write_end_marker()?;
    Ok(())
}

fn decode_complex_types(reader: &mut Reader) -> Result<ComplexTypes> {
    let mut result = ComplexTypes::default();

    while reader.has_more() {
        let tag = reader.read_tag()?;
        if Reader::is_end_marker(&tag) {
            break;
        }
        match tag.field_number {
            1 => {
                let raw = reader.read_svarint()?;
                result.status = Status::from_i32(raw).unwrap_or(Status::Unknown);
            }
            2 => {
                let body = reader.read_length_prefixed_bytes()?;
                let mut sub = Reader::new(body);
                result.optional_nested = Some(Box::new(decode_nested_message(&mut sub)?));
            }
            3 => {
                let body = reader.read_length_prefixed_bytes()?;
                let mut sub = Reader::new(body);
                result.required_nested = decode_nested_message(&mut sub)?;
            }
            4 => {
                let body = reader.read_length_prefixed_bytes()?;
                let mut sub = Reader::new(body);
                let n = sub.read_varint64()? as usize;
                let mut list = Vec::with_capacity(n);
                for _ in 0..n {
                    list.push(decode_nested_message(&mut sub)?);
                }
                result.nested_list = list;
            }
            5 => {
                let body = reader.read_length_prefixed_bytes()?;
                let mut sub = Reader::new(body);
                let n = sub.read_varint64()? as usize;
                let mut m = HashMap::with_capacity(n);
                for _ in 0..n {
                    let k = sub.read_string()?.to_string();
                    let v = sub.read_int32()?;
                    m.insert(k, v);
                }
                result.string_int_map = m;
            }
            6 => {
                let body = reader.read_length_prefixed_bytes()?;
                let mut sub = Reader::new(body);
                let n = sub.read_varint64()? as usize;
                let mut m = HashMap::with_capacity(n);
                for _ in 0..n {
                    let k = sub.read_int32()?;
                    let v = sub.read_string()?.to_string();
                    m.insert(k, v);
                }
                result.int_string_map = m;
            }
            _ => reader.skip_value(tag.wire_type)?,
        }
    }

    Ok(result)
}

fn encode_edge_cases(writer: &mut Writer, msg: &EdgeCases) -> Result<()> {
    // Mirrors the codegen omitempty rules: scalar fields with the
    // zero value are skipped. The Go fixture relies on this to keep
    // the golden bytes minimal — zero_int, empty_string and
    // empty_bytes never appear on the wire.
    if msg.zero_int != 0 {
        writer.write_tag(1, WireType::SVarint)?;
        writer.write_int32(msg.zero_int)?;
    }
    if msg.negative_one != 0 {
        writer.write_tag(2, WireType::SVarint)?;
        writer.write_int32(msg.negative_one)?;
    }
    if msg.max_int32 != 0 {
        writer.write_tag(3, WireType::SVarint)?;
        writer.write_int32(msg.max_int32)?;
    }
    if msg.min_int32 != 0 {
        writer.write_tag(4, WireType::SVarint)?;
        writer.write_int32(msg.min_int32)?;
    }
    if msg.max_int64 != 0 {
        writer.write_tag(5, WireType::SVarint)?;
        writer.write_int64(msg.max_int64)?;
    }
    if msg.min_int64 != 0 {
        writer.write_tag(6, WireType::SVarint)?;
        writer.write_int64(msg.min_int64)?;
    }
    if msg.max_uint32 != 0 {
        writer.write_tag(7, WireType::Varint)?;
        writer.write_uint32(msg.max_uint32)?;
    }
    if msg.max_uint64 != 0 {
        writer.write_tag(8, WireType::Varint)?;
        writer.write_uint64(msg.max_uint64)?;
    }
    if !msg.empty_string.is_empty() {
        writer.write_tag(9, WireType::Bytes)?;
        writer.write_string(&msg.empty_string)?;
    }
    if !msg.unicode_string.is_empty() {
        writer.write_tag(10, WireType::Bytes)?;
        writer.write_string(&msg.unicode_string)?;
    }
    if !msg.empty_bytes.is_empty() {
        writer.write_tag(11, WireType::Bytes)?;
        writer.write_length_prefixed_bytes(&msg.empty_bytes)?;
    }
    writer.write_end_marker()?;
    Ok(())
}

fn decode_edge_cases(reader: &mut Reader) -> Result<EdgeCases> {
    let mut result = EdgeCases::default();

    while reader.has_more() {
        let tag = reader.read_tag()?;
        if Reader::is_end_marker(&tag) {
            break;
        }
        match tag.field_number {
            1 => result.zero_int = reader.read_int32()?,
            2 => result.negative_one = reader.read_int32()?,
            3 => result.max_int32 = reader.read_int32()?,
            4 => result.min_int32 = reader.read_int32()?,
            5 => result.max_int64 = reader.read_int64()?,
            6 => result.min_int64 = reader.read_int64()?,
            7 => result.max_uint32 = reader.read_uint32()?,
            8 => result.max_uint64 = reader.read_uint64()?,
            9 => result.empty_string = reader.read_string()?.to_string(),
            10 => result.unicode_string = reader.read_string()?.to_string(),
            11 => result.empty_bytes = reader.read_length_prefixed_bytes()?.to_vec(),
            _ => reader.skip_value(tag.wire_type)?,
        }
    }

    Ok(result)
}

fn load_golden(name: &str) -> Option<Vec<u8>> {
    let path = PathBuf::from(GOLDEN_DIR).join(format!("{}.bin", name));
    fs::read(&path).ok()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_nested_message_encode_decode() {
        let msg = test_nested_message();
        let mut writer = Writer::new();
        encode_nested_message(&mut writer, &msg).unwrap();
        let encoded = writer.into_bytes();

        println!("NestedMessage encoded: {}", hex::encode(&encoded));

        let mut reader = Reader::new(&encoded);
        let decoded = decode_nested_message(&mut reader).unwrap();

        assert_eq!(decoded.name, msg.name);
        assert_eq!(decoded.value, msg.value);
    }

    #[test]
    fn test_nested_message_golden() {
        let golden = match load_golden("nested_message") {
            Some(data) => data,
            None => {
                println!("Golden file not found, skipping");
                return;
            }
        };

        println!("Golden NestedMessage hex: {}", hex::encode(&golden));

        let mut reader = Reader::new(&golden);
        let decoded = decode_nested_message(&mut reader).unwrap();

        let expected = test_nested_message();
        assert_eq!(decoded.name, expected.name);
        assert_eq!(decoded.value, expected.value);
    }

    #[test]
    fn test_scalar_types_encode_decode() {
        let msg = test_scalar_types();
        let mut writer = Writer::new();
        encode_scalar_types(&mut writer, &msg).unwrap();
        let encoded = writer.into_bytes();

        println!("ScalarTypes encoded: {}", hex::encode(&encoded));
        println!("ScalarTypes size: {} bytes", encoded.len());

        let mut reader = Reader::new(&encoded);
        let decoded = decode_scalar_types(&mut reader).unwrap();

        assert_eq!(decoded.bool_val, msg.bool_val);
        assert_eq!(decoded.int32_val, msg.int32_val);
        assert_eq!(decoded.int64_val, msg.int64_val);
        assert_eq!(decoded.uint32_val, msg.uint32_val);
        assert_eq!(decoded.uint64_val, msg.uint64_val);
        assert!((decoded.float32_val - msg.float32_val).abs() < 0.0001);
        assert_eq!(decoded.float64_val, msg.float64_val);
        assert_eq!(decoded.string_val, msg.string_val);
        assert_eq!(decoded.bytes_val, msg.bytes_val);
    }

    #[test]
    fn test_scalar_types_golden() {
        let golden = match load_golden("scalar_types") {
            Some(data) => data,
            None => {
                println!("Golden file not found, skipping");
                return;
            }
        };

        println!("Golden ScalarTypes hex: {}", hex::encode(&golden));

        let mut reader = Reader::new(&golden);
        let decoded = decode_scalar_types(&mut reader).unwrap();

        let expected = test_scalar_types();
        assert_eq!(decoded.bool_val, expected.bool_val);
        assert_eq!(decoded.int32_val, expected.int32_val);
        assert_eq!(decoded.int64_val, expected.int64_val);
        assert_eq!(decoded.uint32_val, expected.uint32_val);
        assert_eq!(decoded.uint64_val, expected.uint64_val);
        assert!((decoded.float32_val - expected.float32_val).abs() < 0.0001);
        assert_eq!(decoded.float64_val, expected.float64_val);
        assert_eq!(decoded.string_val, expected.string_val);
        assert_eq!(decoded.bytes_val, expected.bytes_val);
    }

    #[test]
    fn test_all_field_numbers_encode_decode() {
        let msg = test_all_field_numbers();
        let mut writer = Writer::new();
        encode_all_field_numbers(&mut writer, &msg).unwrap();
        let encoded = writer.into_bytes();

        println!("AllFieldNumbers encoded: {}", hex::encode(&encoded));

        let mut reader = Reader::new(&encoded);
        let decoded = decode_all_field_numbers(&mut reader).unwrap();

        assert_eq!(decoded.field_1, msg.field_1);
        assert_eq!(decoded.field_15, msg.field_15);
        assert_eq!(decoded.field_16, msg.field_16);
        assert_eq!(decoded.field_127, msg.field_127);
        assert_eq!(decoded.field_128, msg.field_128);
        assert_eq!(decoded.field_1000, msg.field_1000);
    }

    #[test]
    fn test_all_field_numbers_golden() {
        let golden = match load_golden("all_field_numbers") {
            Some(data) => data,
            None => {
                println!("Golden file not found, skipping");
                return;
            }
        };

        println!("Golden AllFieldNumbers hex: {}", hex::encode(&golden));

        let mut reader = Reader::new(&golden);
        let decoded = decode_all_field_numbers(&mut reader).unwrap();

        let expected = test_all_field_numbers();
        assert_eq!(decoded.field_1, expected.field_1);
        assert_eq!(decoded.field_15, expected.field_15);
        assert_eq!(decoded.field_16, expected.field_16);
        assert_eq!(decoded.field_127, expected.field_127);
        assert_eq!(decoded.field_128, expected.field_128);
        assert_eq!(decoded.field_1000, expected.field_1000);
    }

    #[test]
    fn test_varint_encoding_matches_go() {
        let test_cases = vec![
            (0u32, "00"),
            (1, "01"),
            (127, "7f"),
            (128, "8001"),
            (300, "ac02"),
            (16384, "808001"),
        ];

        for (value, expected) in test_cases {
            let mut writer = Writer::new();
            writer.write_varint(value).unwrap();
            let hex = hex::encode(writer.as_bytes());
            assert_eq!(hex, expected, "varint({}) failed", value);
        }
    }

    #[test]
    fn test_zigzag_encoding_matches_go() {
        let test_cases = vec![
            (0i32, "00"),
            (-1, "01"),
            (1, "02"),
            (-2, "03"),
            (2, "04"),
            (-42, "53"),
            (42, "54"),
        ];

        for (value, expected) in test_cases {
            let mut writer = Writer::new();
            writer.write_svarint(value).unwrap();
            let hex = hex::encode(writer.as_bytes());
            assert_eq!(hex, expected, "svarint({}) failed", value);
        }
    }

    #[test]
    fn test_complex_types_encode_decode() {
        let msg = test_complex_types();
        let mut writer = Writer::new();
        encode_complex_types(&mut writer, &msg).unwrap();
        let encoded = writer.into_bytes();

        println!("ComplexTypes encoded: {}", hex::encode(&encoded));

        let mut reader = Reader::new(&encoded);
        let decoded = decode_complex_types(&mut reader).unwrap();

        assert_eq!(decoded.status, msg.status);
        assert_eq!(decoded.optional_nested, msg.optional_nested);
        assert_eq!(decoded.required_nested, msg.required_nested);
        assert_eq!(decoded.nested_list, msg.nested_list);
        assert_eq!(decoded.string_int_map, msg.string_int_map);
        assert_eq!(decoded.int_string_map, msg.int_string_map);
    }

    #[test]
    fn test_complex_types_golden_decode() {
        let golden = match load_golden("complex_types") {
            Some(data) => data,
            None => {
                println!("Golden file not found, skipping");
                return;
            }
        };

        println!("Golden ComplexTypes hex: {}", hex::encode(&golden));

        let mut reader = Reader::new(&golden);
        let decoded = decode_complex_types(&mut reader).unwrap();

        let expected = test_complex_types();
        assert_eq!(decoded.status, expected.status);
        assert_eq!(decoded.optional_nested, expected.optional_nested);
        assert_eq!(decoded.required_nested, expected.required_nested);
        assert_eq!(decoded.nested_list, expected.nested_list);
        assert_eq!(decoded.string_int_map, expected.string_int_map);
        assert_eq!(decoded.int_string_map, expected.int_string_map);
    }

    #[test]
    fn test_edge_cases_encode_decode() {
        let msg = test_edge_cases();
        let mut writer = Writer::new();
        encode_edge_cases(&mut writer, &msg).unwrap();
        let encoded = writer.into_bytes();

        let mut reader = Reader::new(&encoded);
        let decoded = decode_edge_cases(&mut reader).unwrap();

        assert_eq!(decoded.zero_int, msg.zero_int);
        assert_eq!(decoded.negative_one, msg.negative_one);
        assert_eq!(decoded.max_int32, msg.max_int32);
        assert_eq!(decoded.min_int32, msg.min_int32);
        assert_eq!(decoded.max_int64, msg.max_int64);
        assert_eq!(decoded.min_int64, msg.min_int64);
        assert_eq!(decoded.max_uint32, msg.max_uint32);
        assert_eq!(decoded.max_uint64, msg.max_uint64);
        assert_eq!(decoded.empty_string, msg.empty_string);
        assert_eq!(decoded.unicode_string, msg.unicode_string);
        assert_eq!(decoded.empty_bytes, msg.empty_bytes);
    }

    #[test]
    fn test_edge_cases_encoding_matches_golden() {
        let golden = match load_golden("edge_cases") {
            Some(data) => data,
            None => {
                println!("Golden file not found, skipping");
                return;
            }
        };

        let msg = test_edge_cases();
        let mut writer = Writer::new();
        encode_edge_cases(&mut writer, &msg).unwrap();
        let encoded = writer.into_bytes();

        assert_eq!(
            hex::encode(&encoded),
            hex::encode(&golden),
            "Rust EdgeCases encoding diverges from Go golden bytes"
        );
    }

    #[test]
    fn test_complex_types_encoding_matches_golden() {
        let golden = match load_golden("complex_types") {
            Some(data) => data,
            None => {
                println!("Golden file not found, skipping");
                return;
            }
        };

        let msg = test_complex_types();
        let mut writer = Writer::new();
        encode_complex_types(&mut writer, &msg).unwrap();
        let encoded = writer.into_bytes();

        assert_eq!(
            hex::encode(&encoded),
            hex::encode(&golden),
            "Rust encoding diverges from Go golden bytes"
        );
    }
}
