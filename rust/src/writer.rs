//! Cramberry encoder.

use crate::error::Result;
use crate::types::{zigzag_encode_32, zigzag_encode_64, FieldTag, WireType, END_MARKER};

const INITIAL_CAPACITY: usize = 256;

/// Writer encodes Cramberry data into a binary buffer.
pub struct Writer {
    buffer: Vec<u8>,
}

impl Writer {
    /// Creates a new writer with default capacity.
    pub fn new() -> Self {
        Self::with_capacity(INITIAL_CAPACITY)
    }

    /// Creates a new writer with the specified capacity.
    pub fn with_capacity(capacity: usize) -> Self {
        Self {
            buffer: Vec::with_capacity(capacity),
        }
    }

    /// Returns the current length of the buffer.
    pub fn len(&self) -> usize {
        self.buffer.len()
    }

    /// Returns true if the buffer is empty.
    pub fn is_empty(&self) -> bool {
        self.buffer.is_empty()
    }

    /// Returns the encoded bytes as a slice.
    pub fn as_bytes(&self) -> &[u8] {
        &self.buffer
    }

    /// Consumes the writer and returns the encoded bytes.
    pub fn into_bytes(self) -> Vec<u8> {
        self.buffer
    }

    /// Resets the writer for reuse.
    pub fn reset(&mut self) {
        self.buffer.clear();
    }

    /// Writes a V2 compact field tag.
    pub fn write_tag(&mut self, field_number: u32, wire_type: WireType) -> Result<()> {
        let tag = FieldTag::new(field_number, wire_type);
        let encoded = tag.encode_compact();
        self.buffer.extend_from_slice(&encoded);
        Ok(())
    }

    /// Writes the end marker (0x00) to signal end of struct fields.
    pub fn write_end_marker(&mut self) -> Result<()> {
        self.buffer.push(END_MARKER);
        Ok(())
    }

    /// Writes a raw byte.
    pub fn write_byte(&mut self, value: u8) -> Result<()> {
        self.buffer.push(value);
        Ok(())
    }

    /// Writes raw bytes.
    pub fn write_bytes(&mut self, data: &[u8]) -> Result<()> {
        self.buffer.extend_from_slice(data);
        Ok(())
    }

    /// Writes an unsigned varint (LEB128).
    pub fn write_varint(&mut self, mut value: u32) -> Result<()> {
        while value > 0x7f {
            self.buffer.push((value as u8 & 0x7f) | 0x80);
            value >>= 7;
        }
        self.buffer.push(value as u8);
        Ok(())
    }

    /// Writes an unsigned 64-bit varint (LEB128).
    pub fn write_varint64(&mut self, mut value: u64) -> Result<()> {
        while value > 0x7f {
            self.buffer.push((value as u8 & 0x7f) | 0x80);
            value >>= 7;
        }
        self.buffer.push(value as u8);
        Ok(())
    }

    /// Writes a signed varint using ZigZag encoding.
    pub fn write_svarint(&mut self, value: i32) -> Result<()> {
        self.write_varint(zigzag_encode_32(value))
    }

    /// Writes a signed 64-bit varint using ZigZag encoding.
    pub fn write_svarint64(&mut self, value: i64) -> Result<()> {
        self.write_varint64(zigzag_encode_64(value))
    }

    /// Writes a boolean.
    pub fn write_bool(&mut self, value: bool) -> Result<()> {
        self.write_byte(if value { 1 } else { 0 })
    }

    /// Writes a 32-bit signed integer.
    pub fn write_int32(&mut self, value: i32) -> Result<()> {
        self.write_svarint(value)
    }

    /// Writes a 64-bit signed integer.
    pub fn write_int64(&mut self, value: i64) -> Result<()> {
        self.write_svarint64(value)
    }

    /// Writes a 32-bit unsigned integer.
    pub fn write_uint32(&mut self, value: u32) -> Result<()> {
        self.write_varint(value)
    }

    /// Writes a 64-bit unsigned integer.
    pub fn write_uint64(&mut self, value: u64) -> Result<()> {
        self.write_varint64(value)
    }

    /// Writes a 32-bit float (IEEE 754, little-endian).
    pub fn write_float32(&mut self, value: f32) -> Result<()> {
        self.buffer.extend_from_slice(&value.to_le_bytes());
        Ok(())
    }

    /// Writes a 64-bit float (IEEE 754, little-endian).
    pub fn write_float64(&mut self, value: f64) -> Result<()> {
        self.buffer.extend_from_slice(&value.to_le_bytes());
        Ok(())
    }

    /// Writes a fixed 32-bit value (little-endian).
    pub fn write_fixed32(&mut self, value: u32) -> Result<()> {
        self.buffer.extend_from_slice(&value.to_le_bytes());
        Ok(())
    }

    /// Writes a fixed 64-bit value (little-endian).
    pub fn write_fixed64(&mut self, value: u64) -> Result<()> {
        self.buffer.extend_from_slice(&value.to_le_bytes());
        Ok(())
    }

    /// Writes a length-prefixed string.
    pub fn write_string(&mut self, value: &str) -> Result<()> {
        self.write_varint(value.len() as u32)?;
        self.buffer.extend_from_slice(value.as_bytes());
        Ok(())
    }

    /// Writes length-prefixed bytes.
    pub fn write_length_prefixed_bytes(&mut self, data: &[u8]) -> Result<()> {
        self.write_varint(data.len() as u32)?;
        self.buffer.extend_from_slice(data);
        Ok(())
    }

    /// Writes a tagged field with boolean value.
    pub fn write_bool_field(&mut self, field_number: u32, value: bool) -> Result<()> {
        self.write_tag(field_number, WireType::Varint)?;
        self.write_bool(value)
    }

    /// Writes a tagged field with int32 value.
    pub fn write_int32_field(&mut self, field_number: u32, value: i32) -> Result<()> {
        self.write_tag(field_number, WireType::SVarint)?;
        self.write_int32(value)
    }

    /// Writes a tagged field with int64 value.
    pub fn write_int64_field(&mut self, field_number: u32, value: i64) -> Result<()> {
        self.write_tag(field_number, WireType::SVarint)?;
        self.write_int64(value)
    }

    /// Writes a tagged field with uint32 value.
    pub fn write_uint32_field(&mut self, field_number: u32, value: u32) -> Result<()> {
        self.write_tag(field_number, WireType::Varint)?;
        self.write_uint32(value)
    }

    /// Writes a tagged field with uint64 value.
    pub fn write_uint64_field(&mut self, field_number: u32, value: u64) -> Result<()> {
        self.write_tag(field_number, WireType::Varint)?;
        self.write_uint64(value)
    }

    /// Writes a tagged field with float32 value.
    pub fn write_float32_field(&mut self, field_number: u32, value: f32) -> Result<()> {
        self.write_tag(field_number, WireType::Fixed32)?;
        self.write_float32(value)
    }

    /// Writes a tagged field with float64 value.
    pub fn write_float64_field(&mut self, field_number: u32, value: f64) -> Result<()> {
        self.write_tag(field_number, WireType::Fixed64)?;
        self.write_float64(value)
    }

    /// Writes a tagged field with string value.
    pub fn write_string_field(&mut self, field_number: u32, value: &str) -> Result<()> {
        self.write_tag(field_number, WireType::Bytes)?;
        self.write_string(value)
    }

    /// Writes a tagged field with bytes value.
    pub fn write_bytes_field(&mut self, field_number: u32, value: &[u8]) -> Result<()> {
        self.write_tag(field_number, WireType::Bytes)?;
        self.write_length_prefixed_bytes(value)
    }
}

impl Default for Writer {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_write_varint() {
        let mut writer = Writer::new();
        writer.write_varint(0).unwrap();
        assert_eq!(writer.as_bytes(), &[0]);

        let mut writer = Writer::new();
        writer.write_varint(1).unwrap();
        assert_eq!(writer.as_bytes(), &[1]);

        let mut writer = Writer::new();
        writer.write_varint(127).unwrap();
        assert_eq!(writer.as_bytes(), &[127]);

        let mut writer = Writer::new();
        writer.write_varint(128).unwrap();
        assert_eq!(writer.as_bytes(), &[0x80, 0x01]);

        let mut writer = Writer::new();
        writer.write_varint(300).unwrap();
        assert_eq!(writer.as_bytes(), &[0xac, 0x02]);
    }

    #[test]
    fn test_write_svarint() {
        let mut writer = Writer::new();
        writer.write_svarint(0).unwrap();
        assert_eq!(writer.as_bytes(), &[0]);

        let mut writer = Writer::new();
        writer.write_svarint(-1).unwrap();
        assert_eq!(writer.as_bytes(), &[1]);

        let mut writer = Writer::new();
        writer.write_svarint(1).unwrap();
        assert_eq!(writer.as_bytes(), &[2]);
    }

    #[test]
    fn test_write_string() {
        let mut writer = Writer::new();
        writer.write_string("hello").unwrap();
        assert_eq!(writer.as_bytes(), &[5, b'h', b'e', b'l', b'l', b'o']);
    }
}
