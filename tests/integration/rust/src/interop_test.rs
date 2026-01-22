//! Cross-runtime interoperability tests for Rust.
//!
//! These tests verify that Rust runtime produces identical binary
//! encodings to Go and can decode Go-generated golden files.

use cramberry::{Reader, Result, WireType, Writer};
use std::fs;
use std::path::PathBuf;

use crate::interop::*;

const GOLDEN_DIR: &str = "../../golden";

// Test data matching Go's TestData
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

// Encoder functions
fn encode_nested_message(writer: &mut Writer, msg: &NestedMessage) -> Result<()> {
    // Field count
    writer.write_varint(2)?;

    // Field 1: name
    writer.write_tag(1, WireType::Bytes)?;
    writer.write_string(&msg.name)?;

    // Field 2: value - Go uses Varint wire type but zigzag encoding
    writer.write_tag(2, WireType::Varint)?;
    writer.write_svarint(msg.value)?;

    Ok(())
}

fn decode_nested_message(reader: &mut Reader) -> Result<NestedMessage> {
    let field_count = reader.read_varint()?;
    let mut name = String::new();
    let mut value = 0i32;

    for _ in 0..field_count {
        let tag = reader.read_tag()?;
        match tag.field_number {
            1 => name = reader.read_string()?.to_string(),
            2 => value = reader.read_svarint()?,
            _ => reader.skip_field(tag.wire_type)?,
        }
    }

    Ok(NestedMessage { name, value })
}

fn encode_scalar_types(writer: &mut Writer, msg: &ScalarTypes) -> Result<()> {
    // Field count
    writer.write_varint(9)?;

    // Field 1: bool_val
    writer.write_tag(1, WireType::Varint)?;
    writer.write_bool(msg.bool_val)?;

    // Field 2: int32_val - Go uses Varint wire type but zigzag encoding
    writer.write_tag(2, WireType::Varint)?;
    writer.write_svarint(msg.int32_val)?;

    // Field 3: int64_val - Go uses Varint wire type but zigzag encoding
    writer.write_tag(3, WireType::Varint)?;
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

    Ok(())
}

fn decode_scalar_types(reader: &mut Reader) -> Result<ScalarTypes> {
    let field_count = reader.read_varint()?;
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

    for _ in 0..field_count {
        let tag = reader.read_tag()?;
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
            _ => reader.skip_field(tag.wire_type)?,
        }
    }

    Ok(result)
}

fn encode_all_field_numbers(writer: &mut Writer, msg: &AllFieldNumbers) -> Result<()> {
    writer.write_varint(6)?;

    // All int32 fields use Varint wire type but zigzag encoding
    writer.write_tag(1, WireType::Varint)?;
    writer.write_svarint(msg.field_1)?;

    writer.write_tag(15, WireType::Varint)?;
    writer.write_svarint(msg.field_15)?;

    writer.write_tag(16, WireType::Varint)?;
    writer.write_svarint(msg.field_16)?;

    writer.write_tag(127, WireType::Varint)?;
    writer.write_svarint(msg.field_127)?;

    writer.write_tag(128, WireType::Varint)?;
    writer.write_svarint(msg.field_128)?;

    writer.write_tag(1000, WireType::Varint)?;
    writer.write_svarint(msg.field_1000)?;

    Ok(())
}

fn decode_all_field_numbers(reader: &mut Reader) -> Result<AllFieldNumbers> {
    let field_count = reader.read_varint()?;
    let mut result = AllFieldNumbers {
        field_1: 0,
        field_15: 0,
        field_16: 0,
        field_127: 0,
        field_128: 0,
        field_1000: 0,
    };

    for _ in 0..field_count {
        let tag = reader.read_tag()?;
        match tag.field_number {
            1 => result.field_1 = reader.read_svarint()?,
            15 => result.field_15 = reader.read_svarint()?,
            16 => result.field_16 = reader.read_svarint()?,
            127 => result.field_127 = reader.read_svarint()?,
            128 => result.field_128 = reader.read_svarint()?,
            1000 => result.field_1000 = reader.read_svarint()?,
            _ => reader.skip_field(tag.wire_type)?,
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
}
