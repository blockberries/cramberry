//! Streaming support for Cramberry encoding/decoding.
//!
//! This module provides streaming readers and writers that work with
//! any type implementing std::io::Read or std::io::Write traits.
//!
//! # Example
//!
//! ```rust,no_run
//! use std::io::Cursor;
//! use cramberry::stream::{StreamWriter, StreamReader};
//! use cramberry::{Writer, Reader, Result};
//!
//! fn main() -> Result<()> {
//!     // Create a buffer to write to
//!     let mut buffer = Vec::new();
//!
//!     // Write multiple messages
//!     {
//!         let mut stream = StreamWriter::new(&mut buffer);
//!
//!         let mut msg1 = Writer::new();
//!         msg1.write_int32_field(1, 42)?;
//!         stream.write_message(msg1.as_bytes())?;
//!
//!         let mut msg2 = Writer::new();
//!         msg2.write_string_field(1, "hello")?;
//!         stream.write_message(msg2.as_bytes())?;
//!
//!         stream.flush()?;
//!     }
//!
//!     // Read messages back
//!     {
//!         let cursor = Cursor::new(&buffer);
//!         let mut stream = StreamReader::new(cursor);
//!
//!         let msg1_data = stream.read_message()?;
//!         let mut reader1 = Reader::new(&msg1_data);
//!         let tag = reader1.read_tag()?;
//!         assert_eq!(reader1.read_int32()?, 42);
//!
//!         let msg2_data = stream.read_message()?;
//!         let mut reader2 = Reader::new(&msg2_data);
//!         let tag = reader2.read_tag()?;
//!         assert_eq!(reader2.read_string()?, "hello");
//!     }
//!
//!     Ok(())
//! }
//! ```

use std::io::{BufReader, BufWriter, Read, Write};

use crate::error::{Error, Result};

/// Default buffer capacity for stream readers/writers.
const DEFAULT_BUFFER_CAPACITY: usize = 8192;

/// Maximum message size allowed in streaming mode (64 MB by default).
const DEFAULT_MAX_MESSAGE_SIZE: usize = 64 * 1024 * 1024;

/// StreamWriter writes length-delimited messages to a byte stream.
///
/// Messages are written as [length: varint][data: bytes], where length
/// is the number of bytes in the message data.
pub struct StreamWriter<W: Write> {
    inner: BufWriter<W>,
}

impl<W: Write> StreamWriter<W> {
    /// Creates a new StreamWriter wrapping the given writer.
    pub fn new(writer: W) -> Self {
        Self::with_capacity(DEFAULT_BUFFER_CAPACITY, writer)
    }

    /// Creates a new StreamWriter with the specified buffer capacity.
    pub fn with_capacity(capacity: usize, writer: W) -> Self {
        Self {
            inner: BufWriter::with_capacity(capacity, writer),
        }
    }

    /// Writes a length-delimited message.
    ///
    /// The message is prefixed with its length as a varint.
    pub fn write_message(&mut self, data: &[u8]) -> Result<()> {
        self.write_varint(data.len() as u64)?;
        self.inner.write_all(data).map_err(Error::from)?;
        Ok(())
    }

    /// Flushes the underlying buffer.
    pub fn flush(&mut self) -> Result<()> {
        self.inner.flush().map_err(Error::from)
    }

    /// Returns a reference to the underlying writer.
    pub fn get_ref(&self) -> &W {
        self.inner.get_ref()
    }

    /// Returns a mutable reference to the underlying writer.
    pub fn get_mut(&mut self) -> &mut W {
        self.inner.get_mut()
    }

    /// Consumes this StreamWriter, returning the underlying writer.
    ///
    /// This will flush any buffered data before returning the inner writer.
    /// Returns an error if flushing fails.
    pub fn into_inner(self) -> Result<W> {
        self.inner.into_inner().map_err(|e| Error::from(e.into_error()))
    }

    /// Writes a varint to the stream.
    fn write_varint(&mut self, mut value: u64) -> Result<()> {
        let mut buf = [0u8; 10];
        let mut i = 0;

        while value > 0x7f {
            buf[i] = (value as u8 & 0x7f) | 0x80;
            value >>= 7;
            i += 1;
        }
        buf[i] = value as u8;
        i += 1;

        self.inner.write_all(&buf[..i]).map_err(Error::from)
    }
}

/// StreamReader reads length-delimited messages from a byte stream.
///
/// Messages are expected as [length: varint][data: bytes], where length
/// is the number of bytes in the message data.
pub struct StreamReader<R: Read> {
    inner: BufReader<R>,
    max_message_size: usize,
}

impl<R: Read> StreamReader<R> {
    /// Creates a new StreamReader wrapping the given reader.
    pub fn new(reader: R) -> Self {
        Self::with_capacity(DEFAULT_BUFFER_CAPACITY, reader)
    }

    /// Creates a new StreamReader with the specified buffer capacity.
    pub fn with_capacity(capacity: usize, reader: R) -> Self {
        Self {
            inner: BufReader::with_capacity(capacity, reader),
            max_message_size: DEFAULT_MAX_MESSAGE_SIZE,
        }
    }

    /// Sets the maximum allowed message size.
    pub fn set_max_message_size(&mut self, size: usize) {
        self.max_message_size = size;
    }

    /// Reads a length-delimited message.
    ///
    /// Returns the message data as a Vec<u8>.
    /// Returns an error if the stream ends before a complete message is read.
    pub fn read_message(&mut self) -> Result<Vec<u8>> {
        let length = self.read_varint()? as usize;

        // Check against max message size
        if length > self.max_message_size {
            return Err(Error::custom(format!(
                "message size {} exceeds maximum {}",
                length, self.max_message_size
            )));
        }

        let mut data = vec![0u8; length];
        self.inner.read_exact(&mut data).map_err(Error::from)?;
        Ok(data)
    }

    /// Attempts to read a message, returning None if the stream is at EOF.
    ///
    /// This is useful for iterating over all messages in a stream.
    pub fn try_read_message(&mut self) -> Result<Option<Vec<u8>>> {
        match self.try_read_varint()? {
            Some(length) => {
                let length = length as usize;
                if length > self.max_message_size {
                    return Err(Error::custom(format!(
                        "message size {} exceeds maximum {}",
                        length, self.max_message_size
                    )));
                }

                let mut data = vec![0u8; length];
                self.inner.read_exact(&mut data).map_err(Error::from)?;
                Ok(Some(data))
            }
            None => Ok(None),
        }
    }

    /// Returns a reference to the underlying reader.
    pub fn get_ref(&self) -> &R {
        self.inner.get_ref()
    }

    /// Returns a mutable reference to the underlying reader.
    pub fn get_mut(&mut self) -> &mut R {
        self.inner.get_mut()
    }

    /// Reads a varint from the stream.
    fn read_varint(&mut self) -> Result<u64> {
        let mut result: u64 = 0;
        let mut shift = 0;
        let mut buf = [0u8; 1];

        for _ in 0..10 {
            self.inner.read_exact(&mut buf).map_err(Error::from)?;
            let b = buf[0];
            result |= ((b & 0x7f) as u64) << shift;
            if b & 0x80 == 0 {
                return Ok(result);
            }
            shift += 7;
        }

        Err(Error::VarintOverflow)
    }

    /// Attempts to read a varint, returning None if at EOF.
    fn try_read_varint(&mut self) -> Result<Option<u64>> {
        let mut result: u64 = 0;
        let mut shift = 0;
        let mut buf = [0u8; 1];

        for i in 0..10 {
            match self.inner.read(&mut buf) {
                Ok(0) if i == 0 => return Ok(None), // EOF at start
                Ok(0) => return Err(Error::UnexpectedEof), // EOF mid-varint
                Ok(_) => {
                    let b = buf[0];
                    result |= ((b & 0x7f) as u64) << shift;
                    if b & 0x80 == 0 {
                        return Ok(Some(result));
                    }
                    shift += 7;
                }
                Err(e) => return Err(Error::from(e)),
            }
        }

        Err(Error::VarintOverflow)
    }
}

/// Iterator over messages in a stream.
pub struct MessageIter<'a, R: Read> {
    reader: &'a mut StreamReader<R>,
}

impl<R: Read> StreamReader<R> {
    /// Returns an iterator over messages in the stream.
    pub fn messages(&mut self) -> MessageIter<'_, R> {
        MessageIter { reader: self }
    }
}

impl<R: Read> Iterator for MessageIter<'_, R> {
    type Item = Result<Vec<u8>>;

    fn next(&mut self) -> Option<Self::Item> {
        match self.reader.try_read_message() {
            Ok(Some(data)) => Some(Ok(data)),
            Ok(None) => None,
            Err(e) => Some(Err(e)),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;

    #[test]
    fn test_stream_roundtrip() {
        let mut buffer = Vec::new();

        // Write messages
        {
            let mut stream = StreamWriter::new(&mut buffer);
            stream.write_message(b"hello").unwrap();
            stream.write_message(b"world").unwrap();
            stream.write_message(b"!").unwrap();
            stream.flush().unwrap();
        }

        // Read messages back
        {
            let cursor = Cursor::new(&buffer);
            let mut stream = StreamReader::new(cursor);

            assert_eq!(stream.read_message().unwrap(), b"hello");
            assert_eq!(stream.read_message().unwrap(), b"world");
            assert_eq!(stream.read_message().unwrap(), b"!");
        }
    }

    #[test]
    fn test_stream_empty_message() {
        let mut buffer = Vec::new();

        {
            let mut stream = StreamWriter::new(&mut buffer);
            stream.write_message(b"").unwrap();
            stream.flush().unwrap();
        }

        {
            let cursor = Cursor::new(&buffer);
            let mut stream = StreamReader::new(cursor);
            assert_eq!(stream.read_message().unwrap(), b"");
        }
    }

    #[test]
    fn test_stream_large_message() {
        let data = vec![0xABu8; 1000];
        let mut buffer = Vec::new();

        {
            let mut stream = StreamWriter::new(&mut buffer);
            stream.write_message(&data).unwrap();
            stream.flush().unwrap();
        }

        {
            let cursor = Cursor::new(&buffer);
            let mut stream = StreamReader::new(cursor);
            assert_eq!(stream.read_message().unwrap(), data);
        }
    }

    #[test]
    fn test_stream_iterator() {
        let mut buffer = Vec::new();

        {
            let mut stream = StreamWriter::new(&mut buffer);
            stream.write_message(b"one").unwrap();
            stream.write_message(b"two").unwrap();
            stream.write_message(b"three").unwrap();
            stream.flush().unwrap();
        }

        {
            let cursor = Cursor::new(&buffer);
            let mut stream = StreamReader::new(cursor);
            let messages: Vec<_> = stream.messages().collect::<Result<Vec<_>>>().unwrap();
            assert_eq!(messages.len(), 3);
            assert_eq!(messages[0], b"one");
            assert_eq!(messages[1], b"two");
            assert_eq!(messages[2], b"three");
        }
    }

    #[test]
    fn test_stream_try_read_eof() {
        let buffer: Vec<u8> = Vec::new();
        let cursor = Cursor::new(buffer);
        let mut stream = StreamReader::new(cursor);

        assert!(stream.try_read_message().unwrap().is_none());
    }

    #[test]
    fn test_stream_max_message_size() {
        // Create a message that claims to be very large
        let mut buffer = Vec::new();
        {
            let mut stream = StreamWriter::new(&mut buffer);
            // Write varint for 100MB size
            stream.write_varint(100 * 1024 * 1024).unwrap();
            stream.flush().unwrap();
        }

        let cursor = Cursor::new(buffer);
        let mut stream = StreamReader::new(cursor);
        stream.set_max_message_size(1024); // Only allow 1KB

        assert!(stream.read_message().is_err());
    }
}
