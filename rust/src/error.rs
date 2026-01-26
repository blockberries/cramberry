//! Error types for Cramberry operations.

use thiserror::Error;

/// Result type for Cramberry operations.
pub type Result<T> = std::result::Result<T, Error>;

/// Error type for Cramberry operations.
#[derive(Error, Debug)]
pub enum Error {
    /// Buffer overflow during encoding.
    #[error("buffer overflow: needed {needed} bytes, only {available} available")]
    BufferOverflow { needed: usize, available: usize },

    /// Buffer underflow during decoding.
    #[error("buffer underflow: needed {needed} bytes, only {available} available")]
    BufferUnderflow { needed: usize, available: usize },

    /// Varint overflow (more than 10 bytes).
    #[error("varint overflow")]
    VarintOverflow,

    /// Invalid wire type.
    #[error("invalid wire type: {0}")]
    InvalidWireType(u8),

    /// Unknown type ID.
    #[error("unknown type ID: {0}")]
    UnknownTypeId(u32),

    /// Type not registered.
    #[error("type not registered: {0}")]
    TypeNotRegistered(String),

    /// Invalid UTF-8 string.
    #[error("invalid UTF-8 string")]
    InvalidUtf8,

    /// Unexpected end of file.
    #[error("unexpected end of file")]
    UnexpectedEof,

    /// IO error.
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    /// Custom error message.
    #[error("{0}")]
    Custom(String),
}

impl Error {
    /// Creates a buffer overflow error.
    pub fn buffer_overflow(needed: usize, available: usize) -> Self {
        Self::BufferOverflow { needed, available }
    }

    /// Creates a buffer underflow error.
    pub fn buffer_underflow(needed: usize, available: usize) -> Self {
        Self::BufferUnderflow { needed, available }
    }

    /// Creates a custom error.
    pub fn custom(msg: impl Into<String>) -> Self {
        Self::Custom(msg.into())
    }
}
