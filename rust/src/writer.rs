//! Cramberry encoder.

use crate::error::{Error, Result};
use crate::types::{zigzag_encode_32, zigzag_encode_64, FieldTag, WireType, END_MARKER};

/// Canonical 32-bit quiet NaN: sign 0, exponent all 1s, only the quiet bit set.
const CANONICAL_NAN_32: u32 = 0x7FC0_0000;

/// Canonical 64-bit quiet NaN: sign 0, exponent all 1s, only the quiet bit set.
const CANONICAL_NAN_64: u64 = 0x7FF8_0000_0000_0000;

/// Returns the canonical bit representation of an f32:
///
///   - any NaN value collapses to `CANONICAL_NAN_32`,
///   - `-0.0` becomes `+0.0`,
///   - all other values pass through unchanged.
#[inline]
pub(crate) fn canonicalize_f32_bits(v: f32) -> u32 {
    let bits = v.to_bits();
    if bits & 0x7F80_0000 == 0x7F80_0000 && bits & 0x007F_FFFF != 0 {
        return CANONICAL_NAN_32;
    }
    if bits == 0x8000_0000 {
        return 0;
    }
    bits
}

/// Returns the canonical bit representation of an f64. See `canonicalize_f32_bits`.
#[inline]
pub(crate) fn canonicalize_f64_bits(v: f64) -> u64 {
    let bits = v.to_bits();
    if bits & 0x7FF0_0000_0000_0000 == 0x7FF0_0000_0000_0000
        && bits & 0x000F_FFFF_FFFF_FFFF != 0
    {
        return CANONICAL_NAN_64;
    }
    if bits == 0x8000_0000_0000_0000 {
        return 0;
    }
    bits
}

const INITIAL_CAPACITY: usize = 1024;

/// Number of bytes reserved up front by `begin_message` for the length
/// prefix. 2 covers messages up to 16383 bytes (the modal nested-
/// message size); larger bodies retroactively grow the placeholder in
/// `end_message`.
const LENGTH_PLACEHOLDER: usize = 2;

/// Returns the number of bytes a u64 takes in LEB128 (1..=10).
#[inline]
fn uvarint_size(v: u64) -> usize {
    if v < 1 << 7 { return 1; }
    if v < 1 << 14 { return 2; }
    if v < 1 << 21 { return 3; }
    if v < 1 << 28 { return 4; }
    if v < 1 << 35 { return 5; }
    if v < 1 << 42 { return 6; }
    if v < 1 << 49 { return 7; }
    if v < 1 << 56 { return 8; }
    if v < 1 << 63 { return 9; }
    10
}

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

    /// Writes a compact field tag.
    ///
    /// Field number 0 is reserved for the end marker; calling
    /// `write_tag(0, ...)` is a programming error and used to silently
    /// emit no bytes (so the next value byte was decoded as a
    /// mis-shaped tag). Now it errors.
    #[inline]
    pub fn write_tag(&mut self, field_number: u32, wire_type: WireType) -> Result<()> {
        if field_number == 0 {
            return Err(Error::InvalidFieldNumber(field_number));
        }
        let tag = FieldTag::new(field_number, wire_type);
        let encoded = tag.encode_compact();
        self.buffer.extend_from_slice(&encoded);
        Ok(())
    }

    /// Writes the end marker (0x00) to signal end of struct fields.
    #[inline]
    pub fn write_end_marker(&mut self) -> Result<()> {
        self.buffer.push(END_MARKER);
        Ok(())
    }

    /// Begins a length-prefixed message. Reserves a small placeholder
    /// for the length prefix; `end_message` back-patches the actual
    /// varint length and shifts the body if the placeholder was too
    /// tight.
    ///
    /// Returns a checkpoint that must be passed to `end_message`. The
    /// returned offset points to the start of the placeholder, NOT
    /// the start of the body (body starts at checkpoint + 2).
    ///
    /// Replaces the `let mut sub = Writer::new(); ...; writer.write_length_prefixed_bytes(sub.as_bytes())`
    /// pattern that allocated a fresh sub-Writer (`Vec<u8>` with 1 KB
    /// capacity) per nested message / array / map field. With back-
    /// patch we encode straight into the parent's buffer; allocations
    /// drop to ~zero per nested level.
    pub fn begin_message(&mut self) -> usize {
        let checkpoint = self.buffer.len();
        self.buffer.push(0);
        self.buffer.push(0);
        checkpoint
    }

    /// Finishes a length-prefixed message. `checkpoint` must be the
    /// value previously returned by `begin_message`.
    pub fn end_message(&mut self, checkpoint: usize) {
        let msg_start = checkpoint + LENGTH_PLACEHOLDER;
        let msg_len = self.buffer.len() - msg_start;
        let len_size = uvarint_size(msg_len as u64);

        if len_size == LENGTH_PLACEHOLDER {
            self.encode_varint_at(checkpoint, msg_len as u64);
            return;
        }
        if len_size < LENGTH_PLACEHOLDER {
            // Common: 1-byte varint (msg_len < 128). Shift body left
            // by (placeholder - len_size) and truncate.
            let shift = LENGTH_PLACEHOLDER - len_size;
            self.buffer.copy_within(msg_start..msg_start + msg_len, checkpoint + len_size);
            self.buffer.truncate(self.buffer.len() - shift);
            self.encode_varint_at(checkpoint, msg_len as u64);
            return;
        }
        // Rare: msg_len >= 16384, length needs 3+ varint bytes. Grow
        // the placeholder retroactively and memmove the body right.
        let extra = len_size - LENGTH_PLACEHOLDER;
        self.buffer.resize(self.buffer.len() + extra, 0);
        self.buffer.copy_within(msg_start..msg_start + msg_len, msg_start + extra);
        self.encode_varint_at(checkpoint, msg_len as u64);
    }

    fn encode_varint_at(&mut self, mut offset: usize, mut value: u64) {
        while value > 0x7f {
            self.buffer[offset] = (value as u8 & 0x7f) | 0x80;
            value >>= 7;
            offset += 1;
        }
        self.buffer[offset] = value as u8;
    }

    /// Writes a raw byte.
    #[inline]
    pub fn write_byte(&mut self, value: u8) -> Result<()> {
        self.buffer.push(value);
        Ok(())
    }

    /// Writes raw bytes.
    #[inline]
    pub fn write_bytes(&mut self, data: &[u8]) -> Result<()> {
        self.buffer.extend_from_slice(data);
        Ok(())
    }

    /// Writes an unsigned varint (LEB128).
    #[inline]
    pub fn write_varint(&mut self, mut value: u32) -> Result<()> {
        while value > 0x7f {
            self.buffer.push((value as u8 & 0x7f) | 0x80);
            value >>= 7;
        }
        self.buffer.push(value as u8);
        Ok(())
    }

    /// Writes an unsigned 64-bit varint (LEB128).
    #[inline]
    pub fn write_varint64(&mut self, mut value: u64) -> Result<()> {
        while value > 0x7f {
            self.buffer.push((value as u8 & 0x7f) | 0x80);
            value >>= 7;
        }
        self.buffer.push(value as u8);
        Ok(())
    }

    /// Writes a signed varint using ZigZag encoding.
    #[inline]
    pub fn write_svarint(&mut self, value: i32) -> Result<()> {
        self.write_varint(zigzag_encode_32(value))
    }

    /// Writes a signed 64-bit varint using ZigZag encoding.
    #[inline]
    pub fn write_svarint64(&mut self, value: i64) -> Result<()> {
        self.write_varint64(zigzag_encode_64(value))
    }

    /// Writes a boolean.
    #[inline]
    pub fn write_bool(&mut self, value: bool) -> Result<()> {
        self.write_byte(if value { 1 } else { 0 })
    }

    /// Writes a 32-bit signed integer.
    #[inline]
    pub fn write_int32(&mut self, value: i32) -> Result<()> {
        self.write_svarint(value)
    }

    /// Writes a 64-bit signed integer.
    #[inline]
    pub fn write_int64(&mut self, value: i64) -> Result<()> {
        self.write_svarint64(value)
    }

    /// Writes a 32-bit unsigned integer.
    #[inline]
    pub fn write_uint32(&mut self, value: u32) -> Result<()> {
        self.write_varint(value)
    }

    /// Writes a 64-bit unsigned integer.
    #[inline]
    pub fn write_uint64(&mut self, value: u64) -> Result<()> {
        self.write_varint64(value)
    }

    /// Writes a 32-bit float (IEEE 754, little-endian).
    ///
    /// NaN bit patterns are canonicalized to `0x7FC00000` (quiet NaN, no
    /// payload) and `-0.0` is normalized to `+0.0` so that two values that
    /// compare equal always produce identical wire bytes.
    #[inline]
    pub fn write_float32(&mut self, value: f32) -> Result<()> {
        let bits = canonicalize_f32_bits(value);
        self.buffer.extend_from_slice(&bits.to_le_bytes());
        Ok(())
    }

    /// Writes a 64-bit float (IEEE 754, little-endian).
    ///
    /// NaN bit patterns are canonicalized to `0x7FF8000000000000` (quiet NaN,
    /// no payload) and `-0.0` is normalized to `+0.0`.
    #[inline]
    pub fn write_float64(&mut self, value: f64) -> Result<()> {
        let bits = canonicalize_f64_bits(value);
        self.buffer.extend_from_slice(&bits.to_le_bytes());
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
    ///
    /// Uses the 64-bit varint writer so a string longer than `u32::MAX`
    /// bytes (4 GiB) doesn't silently truncate its length prefix — Go's
    /// canon encodes lengths as `uint64`, and `usize as u32` would wrap.
    pub fn write_string(&mut self, value: &str) -> Result<()> {
        self.write_varint64(value.len() as u64)?;
        self.buffer.extend_from_slice(value.as_bytes());
        Ok(())
    }

    /// Writes length-prefixed bytes. See `write_string` re: the 64-bit
    /// varint length prefix.
    pub fn write_length_prefixed_bytes(&mut self, data: &[u8]) -> Result<()> {
        self.write_varint64(data.len() as u64)?;
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

    #[test]
    fn test_write_float32_canonicalizes_nan() {
        // A signalling NaN with a non-canonical payload.
        let nan = f32::from_bits(0x7FA0_0000);
        let mut writer = Writer::new();
        writer.write_float32(nan).unwrap();
        // Expect the canonical quiet-NaN bytes 0x7FC00000 (little-endian).
        assert_eq!(writer.as_bytes(), &[0x00, 0x00, 0xC0, 0x7F]);
    }

    #[test]
    fn test_write_float32_canonicalizes_negative_zero() {
        let mut writer = Writer::new();
        writer.write_float32(-0.0_f32).unwrap();
        assert_eq!(writer.as_bytes(), &[0x00, 0x00, 0x00, 0x00]);
    }

    #[test]
    fn test_write_float64_canonicalizes_nan() {
        let nan = f64::from_bits(0x7FF4_0000_0000_0000);
        let mut writer = Writer::new();
        writer.write_float64(nan).unwrap();
        // 0x7FF8000000000000 little-endian
        assert_eq!(
            writer.as_bytes(),
            &[0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF8, 0x7F]
        );
    }

    #[test]
    fn test_write_float64_canonicalizes_negative_zero() {
        let mut writer = Writer::new();
        writer.write_float64(-0.0_f64).unwrap();
        assert_eq!(
            writer.as_bytes(),
            &[0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00]
        );
    }
}
