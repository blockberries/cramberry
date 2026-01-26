//! Cramberry - High-performance binary serialization library for Rust
//!
//! # Example
//!
//! ```rust
//! use cramberry::{Writer, Reader, Result, WireType};
//!
//! fn main() -> Result<()> {
//!     // Encoding
//!     let mut writer = Writer::new();
//!     writer.write_int32_field(1, 42)?;
//!     writer.write_string_field(2, "hello")?;
//!     let data = writer.into_bytes();
//!
//!     // Decoding
//!     let mut reader = Reader::new(&data);
//!     while reader.has_more() {
//!         let tag = reader.read_tag()?;
//!         match tag.field_number {
//!             1 => {
//!                 let value = reader.read_int32()?;
//!                 assert_eq!(value, 42);
//!             }
//!             2 => {
//!                 let value = reader.read_string()?;
//!                 assert_eq!(value, "hello");
//!             }
//!             _ => reader.skip_field(tag.wire_type)?,
//!         }
//!     }
//!     Ok(())
//! }
//! ```

mod error;
mod reader;
mod registry;
pub mod stream;
mod types;
mod writer;

pub use error::{Error, Result};
pub use reader::Reader;
pub use registry::{Decoder, Encoder, Registry};
pub use stream::{StreamReader, StreamWriter};
pub use types::{FieldTag, TypeId, WireType};
pub use writer::Writer;

/// Library version.
pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Marshal encodes a value using a custom encoder function.
pub fn marshal<T, F>(value: &T, encoder: F) -> Result<Vec<u8>>
where
    F: FnOnce(&mut Writer, &T) -> Result<()>,
{
    let mut writer = Writer::new();
    encoder(&mut writer, value)?;
    Ok(writer.into_bytes())
}

/// Unmarshal decodes a value using a custom decoder function.
pub fn unmarshal<T, F>(data: &[u8], decoder: F) -> Result<T>
where
    F: FnOnce(&mut Reader) -> Result<T>,
{
    let mut reader = Reader::new(data);
    decoder(&mut reader)
}
