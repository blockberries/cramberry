//! Wire format types and utilities.

/// Wire types used in the Cramberry V2 encoding format.
///
/// V2 uses a compact 3-bit wire type encoding:
/// - 0: Varint (unsigned LEB128)
/// - 1: Fixed64 (8 bytes, little-endian)
/// - 2: Bytes (length-prefixed)
/// - 3: Fixed32 (4 bytes, little-endian)
/// - 4: SVarint (ZigZag-encoded signed integer)
/// - 5-7: Reserved for future use
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum WireType {
    /// Variable-length unsigned integer (LEB128).
    Varint = 0,
    /// Fixed 64-bit value (little-endian).
    Fixed64 = 1,
    /// Length-prefixed bytes.
    Bytes = 2,
    /// Fixed 32-bit value (little-endian).
    Fixed32 = 3,
    /// ZigZag-encoded signed integer.
    SVarint = 4,
}

impl WireType {
    /// Converts a u8 to a WireType.
    pub fn from_u8(value: u8) -> Option<Self> {
        match value {
            0 => Some(WireType::Varint),
            1 => Some(WireType::Fixed64),
            2 => Some(WireType::Bytes),
            3 => Some(WireType::Fixed32),
            4 => Some(WireType::SVarint),
            _ => None,
        }
    }
}

/// V2 compact tag format constants.
/// Tag format: [field_num:4][wire_type:3][ext:1]
/// For fields 1-15: single byte tag
/// For fields 16+: ext bit set, followed by varint field number
pub const END_MARKER: u8 = 0x00;
pub const TAG_EXTENDED_BIT: u8 = 0x01;
pub const TAG_WIRE_TYPE_MASK: u8 = 0x0e;
pub const TAG_WIRE_TYPE_SHIFT: u8 = 1;
pub const TAG_FIELD_NUM_SHIFT: u8 = 4;
pub const MAX_COMPACT_FIELD_NUM: u32 = 15;

/// Type ID for polymorphic type registration.
pub type TypeId = u32;

/// Field tag containing field number and wire type.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct FieldTag {
    pub field_number: u32,
    pub wire_type: WireType,
}

impl FieldTag {
    /// Creates a new field tag.
    pub fn new(field_number: u32, wire_type: WireType) -> Self {
        Self {
            field_number,
            wire_type,
        }
    }

    /// Encodes the field tag to V2 compact format bytes.
    /// Returns a Vec<u8> containing 1-6 bytes depending on field number.
    pub fn encode_compact(&self) -> Vec<u8> {
        if self.field_number == 0 {
            return vec![];
        }

        if self.field_number <= MAX_COMPACT_FIELD_NUM {
            // Single byte: [field_num:4][wire_type:3][ext:0]
            let tag = ((self.field_number as u8) << TAG_FIELD_NUM_SHIFT)
                | ((self.wire_type as u8) << TAG_WIRE_TYPE_SHIFT);
            vec![tag]
        } else {
            // Extended: [0:4][wire_type:3][ext:1] + varint(field_num)
            let marker = ((self.wire_type as u8) << TAG_WIRE_TYPE_SHIFT) | TAG_EXTENDED_BIT;
            let mut result = vec![marker];
            let mut num = self.field_number;
            while num >= 0x80 {
                result.push((num as u8 & 0x7f) | 0x80);
                num >>= 7;
            }
            result.push(num as u8);
            result
        }
    }

    /// Encodes the field tag to a u32 (legacy format, deprecated).
    #[deprecated(note = "Use encode_compact for V2 format")]
    pub fn encode(&self) -> u32 {
        (self.field_number << 3) | (self.wire_type as u32)
    }

    /// Decodes a u32 to a field tag (legacy format, deprecated).
    #[deprecated(note = "Use decode_compact for V2 format")]
    pub fn decode(value: u32) -> Option<Self> {
        let wire_type = WireType::from_u8((value & 0x07) as u8)?;
        Some(Self {
            field_number: value >> 3,
            wire_type,
        })
    }
}

/// Result of decoding a V2 compact tag.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct CompactTagResult {
    pub field_number: u32,
    pub wire_type: WireType,
    pub bytes_read: usize,
}

/// Decodes a V2 compact tag from a byte slice.
/// Returns None if the buffer is empty or the wire type is invalid.
pub fn decode_compact_tag(data: &[u8]) -> Option<CompactTagResult> {
    if data.is_empty() {
        return None;
    }

    let tag = data[0];

    // End marker check
    if tag == END_MARKER {
        return Some(CompactTagResult {
            field_number: 0,
            wire_type: WireType::Varint, // Placeholder
            bytes_read: 1,
        });
    }

    let wire_type_val = (tag & TAG_WIRE_TYPE_MASK) >> TAG_WIRE_TYPE_SHIFT;
    let wire_type = WireType::from_u8(wire_type_val)?;

    if (tag & TAG_EXTENDED_BIT) == 0 {
        // Compact format: single byte
        let field_number = (tag >> TAG_FIELD_NUM_SHIFT) as u32;
        Some(CompactTagResult {
            field_number,
            wire_type,
            bytes_read: 1,
        })
    } else {
        // Extended format: read varint field number
        let mut field_number: u32 = 0;
        let mut shift = 0;
        let mut i = 1;
        loop {
            if i >= data.len() {
                return None; // Buffer underflow
            }
            let b = data[i];
            field_number |= ((b & 0x7f) as u32) << shift;
            i += 1;
            if (b & 0x80) == 0 {
                break;
            }
            shift += 7;
            if shift >= 35 {
                return None; // Varint overflow
            }
        }
        Some(CompactTagResult {
            field_number,
            wire_type,
            bytes_read: i,
        })
    }
}

/// Encodes a signed integer using ZigZag encoding.
#[inline]
pub fn zigzag_encode_32(n: i32) -> u32 {
    ((n << 1) ^ (n >> 31)) as u32
}

/// Encodes a signed 64-bit integer using ZigZag encoding.
#[inline]
pub fn zigzag_encode_64(n: i64) -> u64 {
    ((n << 1) ^ (n >> 63)) as u64
}

/// Decodes a ZigZag encoded integer.
#[inline]
pub fn zigzag_decode_32(n: u32) -> i32 {
    ((n >> 1) as i32) ^ (-((n & 1) as i32))
}

/// Decodes a ZigZag encoded 64-bit integer.
#[inline]
pub fn zigzag_decode_64(n: u64) -> i64 {
    ((n >> 1) as i64) ^ (-((n & 1) as i64))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_zigzag_encode_32() {
        assert_eq!(zigzag_encode_32(0), 0);
        assert_eq!(zigzag_encode_32(-1), 1);
        assert_eq!(zigzag_encode_32(1), 2);
        assert_eq!(zigzag_encode_32(-2), 3);
        assert_eq!(zigzag_encode_32(2), 4);
    }

    #[test]
    fn test_zigzag_decode_32() {
        assert_eq!(zigzag_decode_32(0), 0);
        assert_eq!(zigzag_decode_32(1), -1);
        assert_eq!(zigzag_decode_32(2), 1);
        assert_eq!(zigzag_decode_32(3), -2);
        assert_eq!(zigzag_decode_32(4), 2);
    }

    #[test]
    fn test_compact_tag_field_1_varint() {
        let tag = FieldTag::new(1, WireType::Varint);
        let encoded = tag.encode_compact();
        assert_eq!(encoded.len(), 1);
        // Field 1, wire type 0: (1 << 4) | (0 << 1) | 0 = 0x10
        assert_eq!(encoded[0], 0x10);

        let decoded = decode_compact_tag(&encoded).unwrap();
        assert_eq!(decoded.field_number, 1);
        assert_eq!(decoded.wire_type, WireType::Varint);
        assert_eq!(decoded.bytes_read, 1);
    }

    #[test]
    fn test_compact_tag_field_1_bytes() {
        let tag = FieldTag::new(1, WireType::Bytes);
        let encoded = tag.encode_compact();
        assert_eq!(encoded.len(), 1);
        // Field 1, wire type 2: (1 << 4) | (2 << 1) | 0 = 0x14
        assert_eq!(encoded[0], 0x14);

        let decoded = decode_compact_tag(&encoded).unwrap();
        assert_eq!(decoded.field_number, 1);
        assert_eq!(decoded.wire_type, WireType::Bytes);
    }

    #[test]
    fn test_compact_tag_field_15() {
        let tag = FieldTag::new(15, WireType::SVarint);
        let encoded = tag.encode_compact();
        assert_eq!(encoded.len(), 1);
        // Field 15, wire type 4: (15 << 4) | (4 << 1) | 0 = 0xf8
        assert_eq!(encoded[0], 0xf8);

        let decoded = decode_compact_tag(&encoded).unwrap();
        assert_eq!(decoded.field_number, 15);
        assert_eq!(decoded.wire_type, WireType::SVarint);
    }

    #[test]
    fn test_compact_tag_field_16_extended() {
        let tag = FieldTag::new(16, WireType::Varint);
        let encoded = tag.encode_compact();
        assert_eq!(encoded.len(), 2);
        // Extended: (0 << 1) | 1 = 0x01, then varint 16
        assert_eq!(encoded[0], 0x01);
        assert_eq!(encoded[1], 16);

        let decoded = decode_compact_tag(&encoded).unwrap();
        assert_eq!(decoded.field_number, 16);
        assert_eq!(decoded.wire_type, WireType::Varint);
        assert_eq!(decoded.bytes_read, 2);
    }

    #[test]
    fn test_compact_tag_large_field_number() {
        let tag = FieldTag::new(1000, WireType::SVarint);
        let encoded = tag.encode_compact();
        // Field 1000 needs 2 varint bytes (1000 = 0x3e8)
        assert!(encoded.len() >= 3);

        let decoded = decode_compact_tag(&encoded).unwrap();
        assert_eq!(decoded.field_number, 1000);
        assert_eq!(decoded.wire_type, WireType::SVarint);
    }

    #[test]
    fn test_compact_tag_roundtrip() {
        // Test compact format (fields 1-15)
        for field in 1..=15 {
            for wire_type in [
                WireType::Varint,
                WireType::Fixed64,
                WireType::Bytes,
                WireType::Fixed32,
                WireType::SVarint,
            ] {
                let tag = FieldTag::new(field, wire_type);
                let encoded = tag.encode_compact();
                assert_eq!(encoded.len(), 1, "Field {} should be single byte", field);

                let decoded = decode_compact_tag(&encoded).unwrap();
                assert_eq!(decoded.field_number, field);
                assert_eq!(decoded.wire_type, wire_type);
            }
        }

        // Test extended format (fields 16+)
        for field in [16, 100, 1000, 10000] {
            for wire_type in [
                WireType::Varint,
                WireType::Fixed64,
                WireType::Bytes,
                WireType::Fixed32,
                WireType::SVarint,
            ] {
                let tag = FieldTag::new(field, wire_type);
                let encoded = tag.encode_compact();
                assert!(encoded.len() >= 2, "Field {} should be extended format", field);

                let decoded = decode_compact_tag(&encoded).unwrap();
                assert_eq!(decoded.field_number, field);
                assert_eq!(decoded.wire_type, wire_type);
            }
        }
    }

    #[test]
    fn test_end_marker() {
        let decoded = decode_compact_tag(&[END_MARKER]).unwrap();
        assert_eq!(decoded.field_number, 0);
        assert_eq!(decoded.bytes_read, 1);
    }
}
