//! Wire format types and utilities.

/// Wire types used in the Cramberry encoding format.
///
/// Note: Wire types 3 and 4 are intentionally skipped and reserved.
/// In Protocol Buffers, these values were used for the deprecated
/// "start group" (3) and "end group" (4) wire types. Cramberry skips
/// these values to maintain partial compatibility with protobuf tooling
/// and to reserve them for potential future use.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum WireType {
    /// Variable-length unsigned integer (LEB128).
    Varint = 0,
    /// Fixed 64-bit value (little-endian).
    Fixed64 = 1,
    /// Length-prefixed bytes.
    Bytes = 2,
    // Wire types 3 and 4 are reserved (see enum documentation)
    /// Fixed 32-bit value (little-endian).
    Fixed32 = 5,
    /// ZigZag-encoded signed integer.
    SVarint = 6,
    /// Type reference for polymorphic values.
    TypeRef = 7,
}

impl WireType {
    /// Converts a u8 to a WireType.
    pub fn from_u8(value: u8) -> Option<Self> {
        match value {
            0 => Some(WireType::Varint),
            1 => Some(WireType::Fixed64),
            2 => Some(WireType::Bytes),
            5 => Some(WireType::Fixed32),
            6 => Some(WireType::SVarint),
            7 => Some(WireType::TypeRef),
            _ => None,
        }
    }
}

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

    /// Encodes the field tag to a u32.
    pub fn encode(&self) -> u32 {
        (self.field_number << 3) | (self.wire_type as u32)
    }

    /// Decodes a u32 to a field tag.
    pub fn decode(value: u32) -> Option<Self> {
        let wire_type = WireType::from_u8((value & 0x07) as u8)?;
        Some(Self {
            field_number: value >> 3,
            wire_type,
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
    fn test_field_tag_encode_decode() {
        let tag = FieldTag::new(10, WireType::SVarint);
        let encoded = tag.encode();
        let decoded = FieldTag::decode(encoded).unwrap();
        assert_eq!(decoded.field_number, 10);
        assert_eq!(decoded.wire_type, WireType::SVarint);
    }
}
