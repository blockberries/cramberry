//! Cramberry decoder.

use crate::error::{Error, Result};
use crate::types::{
    decode_compact_tag, zigzag_decode_32, zigzag_decode_64, FieldTag, WireType, END_MARKER,
};

/// Reader decodes Cramberry data from a binary buffer.
pub struct Reader<'a> {
    buffer: &'a [u8],
    pos: usize,
}

/// Maximum number of bytes for a varint-encoded uint64.
/// A uint64 has 64 bits, and each varint byte encodes 7 bits,
/// so we need ceil(64/7) = 10 bytes maximum.
const MAX_VARINT_BYTES: usize = 10;

impl<'a> Reader<'a> {
    /// Creates a new reader from a byte slice.
    pub fn new(data: &'a [u8]) -> Self {
        Self {
            buffer: data,
            pos: 0,
        }
    }

    /// Returns the current position in the buffer.
    pub fn position(&self) -> usize {
        self.pos
    }

    /// Returns the number of bytes remaining.
    pub fn remaining(&self) -> usize {
        self.buffer.len() - self.pos
    }

    /// Returns true if there is more data to read.
    pub fn has_more(&self) -> bool {
        self.pos < self.buffer.len()
    }

    /// Checks if there are enough bytes available.
    fn check_available(&self, needed: usize) -> Result<()> {
        if self.pos + needed > self.buffer.len() {
            return Err(Error::buffer_underflow(needed, self.remaining()));
        }
        Ok(())
    }

    /// Reads a raw byte.
    pub fn read_byte(&mut self) -> Result<u8> {
        self.check_available(1)?;
        let value = self.buffer[self.pos];
        self.pos += 1;
        Ok(value)
    }

    /// Reads raw bytes.
    pub fn read_bytes(&mut self, length: usize) -> Result<&'a [u8]> {
        self.check_available(length)?;
        let bytes = &self.buffer[self.pos..self.pos + length];
        self.pos += length;
        Ok(bytes)
    }

    /// Reads an unsigned varint (LEB128).
    /// For 32-bit values, this uses the same 10-byte limit as 64-bit for consistency,
    /// but the result is capped to 32 bits.
    pub fn read_varint(&mut self) -> Result<u32> {
        let mut result: u32 = 0;
        let mut shift = 0;

        for i in 0..MAX_VARINT_BYTES {
            self.check_available(1)?;
            let b = self.buffer[self.pos];
            self.pos += 1;

            // At the 5th byte (index 4), we've consumed 28 bits.
            // The 5th byte can only contribute 4 more bits for a 32-bit value.
            if i == 4 && (b & 0xf0) != 0 {
                return Err(Error::VarintOverflow);
            }

            result |= ((b & 0x7f) as u32) << shift;
            if b & 0x80 == 0 {
                return Ok(result);
            }
            shift += 7;
        }

        Err(Error::VarintOverflow)
    }

    /// Reads an unsigned 64-bit varint (LEB128).
    /// Uses a maximum of 10 bytes, consistent with protobuf and Go implementation.
    pub fn read_varint64(&mut self) -> Result<u64> {
        let mut result: u64 = 0;
        let mut shift = 0;

        for i in 0..MAX_VARINT_BYTES {
            self.check_available(1)?;
            let b = self.buffer[self.pos];
            self.pos += 1;

            // At the 10th byte (index 9), we've consumed 63 bits.
            // The 10th byte can only contribute 1 more bit (bit 63 of uint64).
            if i == 9 {
                // If continuation bit is set, we'd need 11+ bytes
                if b >= 0x80 {
                    return Err(Error::VarintOverflow);
                }
                // If data portion is > 1, value would overflow uint64
                if b > 1 {
                    return Err(Error::VarintOverflow);
                }
            }

            result |= ((b & 0x7f) as u64) << shift;
            if b & 0x80 == 0 {
                return Ok(result);
            }
            shift += 7;
        }

        Err(Error::VarintOverflow)
    }

    /// Reads a signed varint using ZigZag decoding.
    pub fn read_svarint(&mut self) -> Result<i32> {
        Ok(zigzag_decode_32(self.read_varint()?))
    }

    /// Reads a signed 64-bit varint using ZigZag decoding.
    pub fn read_svarint64(&mut self) -> Result<i64> {
        Ok(zigzag_decode_64(self.read_varint64()?))
    }

    /// Reads a V2 compact field tag.
    /// Returns a FieldTag with field_number=0 for end marker.
    pub fn read_tag(&mut self) -> Result<FieldTag> {
        let remaining = &self.buffer[self.pos..];
        if remaining.is_empty() {
            return Err(Error::buffer_underflow(1, 0));
        }

        let result = decode_compact_tag(remaining)
            .ok_or_else(|| Error::InvalidWireType(remaining[0]))?;

        self.pos += result.bytes_read;
        Ok(FieldTag::new(result.field_number, result.wire_type))
    }

    /// Checks if the next byte is the end marker without consuming it.
    pub fn peek_end_marker(&self) -> bool {
        self.pos < self.buffer.len() && self.buffer[self.pos] == END_MARKER
    }

    /// Checks if the given tag is an end marker.
    pub fn is_end_marker(tag: &FieldTag) -> bool {
        tag.field_number == 0
    }

    /// Reads a boolean.
    pub fn read_bool(&mut self) -> Result<bool> {
        Ok(self.read_byte()? != 0)
    }

    /// Reads a 32-bit signed integer.
    pub fn read_int32(&mut self) -> Result<i32> {
        self.read_svarint()
    }

    /// Reads a 64-bit signed integer.
    pub fn read_int64(&mut self) -> Result<i64> {
        self.read_svarint64()
    }

    /// Reads a 32-bit unsigned integer.
    pub fn read_uint32(&mut self) -> Result<u32> {
        self.read_varint()
    }

    /// Reads a 64-bit unsigned integer.
    pub fn read_uint64(&mut self) -> Result<u64> {
        self.read_varint64()
    }

    /// Reads a 32-bit float (IEEE 754, little-endian).
    pub fn read_float32(&mut self) -> Result<f32> {
        let bytes = self.read_bytes(4)?;
        Ok(f32::from_le_bytes([bytes[0], bytes[1], bytes[2], bytes[3]]))
    }

    /// Reads a 64-bit float (IEEE 754, little-endian).
    pub fn read_float64(&mut self) -> Result<f64> {
        let bytes = self.read_bytes(8)?;
        Ok(f64::from_le_bytes([
            bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5], bytes[6], bytes[7],
        ]))
    }

    /// Reads a fixed 32-bit value (little-endian).
    pub fn read_fixed32(&mut self) -> Result<u32> {
        let bytes = self.read_bytes(4)?;
        Ok(u32::from_le_bytes([bytes[0], bytes[1], bytes[2], bytes[3]]))
    }

    /// Reads a fixed 64-bit value (little-endian).
    pub fn read_fixed64(&mut self) -> Result<u64> {
        let bytes = self.read_bytes(8)?;
        Ok(u64::from_le_bytes([
            bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5], bytes[6], bytes[7],
        ]))
    }

    /// Reads a length-prefixed string.
    pub fn read_string(&mut self) -> Result<&'a str> {
        let length = self.read_varint()? as usize;
        let bytes = self.read_bytes(length)?;
        std::str::from_utf8(bytes).map_err(|_| Error::InvalidUtf8)
    }

    /// Reads length-prefixed bytes.
    pub fn read_length_prefixed_bytes(&mut self) -> Result<&'a [u8]> {
        let length = self.read_varint()? as usize;
        self.read_bytes(length)
    }

    /// Skips a field based on its wire type.
    pub fn skip_field(&mut self, wire_type: WireType) -> Result<()> {
        match wire_type {
            WireType::Varint | WireType::SVarint => {
                self.read_varint64()?; // Use 64-bit to handle large varints
            }
            WireType::Fixed64 => {
                self.check_available(8)?;
                self.pos += 8;
            }
            WireType::Bytes => {
                let length = self.read_varint()? as usize;
                self.check_available(length)?;
                self.pos += length;
            }
            WireType::Fixed32 => {
                self.check_available(4)?;
                self.pos += 4;
            }
        }
        Ok(())
    }

    /// Creates a sub-reader for reading nested messages.
    pub fn sub_reader(&mut self, length: usize) -> Result<Reader<'a>> {
        self.check_available(length)?;
        let sub = Reader::new(&self.buffer[self.pos..self.pos + length]);
        self.pos += length;
        Ok(sub)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_read_varint() {
        let mut reader = Reader::new(&[0]);
        assert_eq!(reader.read_varint().unwrap(), 0);

        let mut reader = Reader::new(&[1]);
        assert_eq!(reader.read_varint().unwrap(), 1);

        let mut reader = Reader::new(&[127]);
        assert_eq!(reader.read_varint().unwrap(), 127);

        let mut reader = Reader::new(&[0x80, 0x01]);
        assert_eq!(reader.read_varint().unwrap(), 128);

        let mut reader = Reader::new(&[0xac, 0x02]);
        assert_eq!(reader.read_varint().unwrap(), 300);
    }

    #[test]
    fn test_read_svarint() {
        let mut reader = Reader::new(&[0]);
        assert_eq!(reader.read_svarint().unwrap(), 0);

        let mut reader = Reader::new(&[1]);
        assert_eq!(reader.read_svarint().unwrap(), -1);

        let mut reader = Reader::new(&[2]);
        assert_eq!(reader.read_svarint().unwrap(), 1);
    }

    #[test]
    fn test_read_string() {
        let mut reader = Reader::new(&[5, b'h', b'e', b'l', b'l', b'o']);
        assert_eq!(reader.read_string().unwrap(), "hello");
    }

    #[test]
    fn test_read_v2_compact_tag() {
        // Field 1, wire type Varint: (1 << 4) | (0 << 1) = 0x10
        let mut reader = Reader::new(&[0x10]);
        let tag = reader.read_tag().unwrap();
        assert_eq!(tag.field_number, 1);
        assert_eq!(tag.wire_type, WireType::Varint);

        // Field 2, wire type SVarint: (2 << 4) | (4 << 1) = 0x28
        let mut reader = Reader::new(&[0x28]);
        let tag = reader.read_tag().unwrap();
        assert_eq!(tag.field_number, 2);
        assert_eq!(tag.wire_type, WireType::SVarint);

        // Field 15, wire type Bytes: (15 << 4) | (2 << 1) = 0xf4
        let mut reader = Reader::new(&[0xf4]);
        let tag = reader.read_tag().unwrap();
        assert_eq!(tag.field_number, 15);
        assert_eq!(tag.wire_type, WireType::Bytes);
    }

    #[test]
    fn test_read_v2_extended_tag() {
        // Field 16, wire type Varint: extended marker + varint 16
        // Extended: (0 << 1) | 1 = 0x01, then varint 16
        let mut reader = Reader::new(&[0x01, 16]);
        let tag = reader.read_tag().unwrap();
        assert_eq!(tag.field_number, 16);
        assert_eq!(tag.wire_type, WireType::Varint);

        // Field 1000, wire type SVarint
        // Extended: (4 << 1) | 1 = 0x09, then varint 1000 (0xe8, 0x07)
        let mut reader = Reader::new(&[0x09, 0xe8, 0x07]);
        let tag = reader.read_tag().unwrap();
        assert_eq!(tag.field_number, 1000);
        assert_eq!(tag.wire_type, WireType::SVarint);
    }

    #[test]
    fn test_read_end_marker() {
        let mut reader = Reader::new(&[END_MARKER]);
        let tag = reader.read_tag().unwrap();
        assert_eq!(tag.field_number, 0);
        assert!(Reader::is_end_marker(&tag));
    }

    #[test]
    fn test_read_write_roundtrip() {
        use crate::writer::Writer;

        let mut writer = Writer::new();
        writer.write_int32_field(1, -42).unwrap();
        writer.write_string_field(2, "hello").unwrap();
        writer.write_bool_field(3, true).unwrap();
        writer.write_end_marker().unwrap();

        let data = writer.into_bytes();
        let mut reader = Reader::new(&data);

        let tag1 = reader.read_tag().unwrap();
        assert_eq!(tag1.field_number, 1);
        assert_eq!(tag1.wire_type, WireType::SVarint);
        assert_eq!(reader.read_int32().unwrap(), -42);

        let tag2 = reader.read_tag().unwrap();
        assert_eq!(tag2.field_number, 2);
        assert_eq!(tag2.wire_type, WireType::Bytes);
        assert_eq!(reader.read_string().unwrap(), "hello");

        let tag3 = reader.read_tag().unwrap();
        assert_eq!(tag3.field_number, 3);
        assert_eq!(tag3.wire_type, WireType::Varint);
        assert!(reader.read_bool().unwrap());

        // Read end marker
        let end_tag = reader.read_tag().unwrap();
        assert!(Reader::is_end_marker(&end_tag));

        assert!(!reader.has_more());
    }

    #[test]
    fn test_peek_end_marker() {
        let mut reader = Reader::new(&[0x10, END_MARKER]);

        // First, peek - should not be end marker
        assert!(!reader.peek_end_marker());

        // Read the first tag
        let tag = reader.read_tag().unwrap();
        assert_eq!(tag.field_number, 1);

        // Now peek - should be end marker
        assert!(reader.peek_end_marker());

        // Read it
        let end_tag = reader.read_tag().unwrap();
        assert!(Reader::is_end_marker(&end_tag));
    }
}
