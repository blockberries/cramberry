// Cross-language interop test fixture. Holds the message-shape struct
// definitions referenced by the integration tests.
//
// The encode/decode functions for these types live in interop_test.rs as
// hand-written reference implementations of the wire format. We don't
// re-export the codegen-emitted ones here because the Rust generator's
// output differs in error type and JSON helper signatures from what the
// integration crate links against; keeping the fixture to type-defs only
// avoids that drift.

use serde::{Deserialize, Serialize};

/// Status enum for testing enum serialization.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default, Serialize, Deserialize)]
#[repr(i32)]
pub enum Status {
    #[default]
    Unknown = 0,
    Active = 1,
    Inactive = 2,
    Pending = 3,
}

impl Status {
    pub fn from_i32(value: i32) -> Option<Self> {
        match value {
            0 => Some(Self::Unknown),
            1 => Some(Self::Active),
            2 => Some(Self::Inactive),
            3 => Some(Self::Pending),
            _ => None,
        }
    }
}

/// ScalarTypes tests all scalar type serialization.
#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct ScalarTypes {
    #[serde(rename = "bool_val")]
    pub bool_val: bool,
    #[serde(rename = "int32_val")]
    pub int32_val: i32,
    #[serde(rename = "int64_val")]
    pub int64_val: i64,
    #[serde(rename = "uint32_val")]
    pub uint32_val: u32,
    #[serde(rename = "uint64_val")]
    pub uint64_val: u64,
    #[serde(rename = "float32_val")]
    pub float32_val: f32,
    #[serde(rename = "float64_val")]
    pub float64_val: f64,
    #[serde(rename = "string_val")]
    pub string_val: String,
    #[serde(rename = "bytes_val")]
    pub bytes_val: Vec<u8>,
}

/// RepeatedTypes tests repeated field serialization.
#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct RepeatedTypes {
    #[serde(rename = "int32_list")]
    pub int32_list: Vec<i32>,
    #[serde(rename = "string_list")]
    pub string_list: Vec<String>,
    #[serde(rename = "bytes_list")]
    pub bytes_list: Vec<Vec<u8>>,
}

/// NestedMessage tests nested message serialization.
#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct NestedMessage {
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "value")]
    pub value: i32,
}

/// ComplexTypes tests complex type serialization.
#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct ComplexTypes {
    #[serde(rename = "status")]
    pub status: Status,
    #[serde(rename = "optional_nested")]
    pub optional_nested: Option<Box<NestedMessage>>,
    #[serde(rename = "required_nested")]
    pub required_nested: NestedMessage,
    #[serde(rename = "nested_list")]
    pub nested_list: Vec<NestedMessage>,
    #[serde(rename = "string_int_map")]
    pub string_int_map: std::collections::HashMap<String, i32>,
    #[serde(rename = "int_string_map")]
    pub int_string_map: std::collections::HashMap<i32, String>,
}

/// EdgeCases tests edge case values.
#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct EdgeCases {
    #[serde(rename = "zero_int")]
    pub zero_int: i32,
    #[serde(rename = "negative_one")]
    pub negative_one: i32,
    #[serde(rename = "max_int32")]
    pub max_int32: i32,
    #[serde(rename = "min_int32")]
    pub min_int32: i32,
    #[serde(rename = "max_int64")]
    pub max_int64: i64,
    #[serde(rename = "min_int64")]
    pub min_int64: i64,
    #[serde(rename = "max_uint32")]
    pub max_uint32: u32,
    #[serde(rename = "max_uint64")]
    pub max_uint64: u64,
    #[serde(rename = "empty_string")]
    pub empty_string: String,
    #[serde(rename = "unicode_string")]
    pub unicode_string: String,
    #[serde(rename = "empty_bytes")]
    pub empty_bytes: Vec<u8>,
}

/// AllFieldNumbers tests field numbers across the compact-tag boundary.
#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct AllFieldNumbers {
    #[serde(rename = "field_1")]
    pub field_1: i32,
    #[serde(rename = "field_15")]
    pub field_15: i32,
    #[serde(rename = "field_16")]
    pub field_16: i32,
    #[serde(rename = "field_127")]
    pub field_127: i32,
    #[serde(rename = "field_128")]
    pub field_128: i32,
    #[serde(rename = "field_1000")]
    pub field_1000: i32,
}
