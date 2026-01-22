//! Type registry for polymorphic encoding/decoding.

use std::collections::HashMap;

use crate::error::{Error, Result};
use crate::reader::Reader;
use crate::types::{TypeId, WireType};
use crate::writer::Writer;

/// Encoder function type.
pub type Encoder<T> = fn(&mut Writer, &T) -> Result<()>;

/// Decoder function type.
pub type Decoder<T> = fn(&mut Reader) -> Result<T>;

/// Type-erased encoder.
type AnyEncoder = Box<dyn Fn(&mut Writer, &dyn std::any::Any) -> Result<()> + Send + Sync>;

/// Type-erased decoder.
type AnyDecoder = Box<dyn Fn(&mut Reader) -> Result<Box<dyn std::any::Any + Send + Sync>> + Send + Sync>;

/// Registration information for a type.
struct TypeRegistration {
    type_id: TypeId,
    name: String,
    encoder: AnyEncoder,
    decoder: AnyDecoder,
}

/// Registry manages type registrations for polymorphic encoding/decoding.
pub struct Registry {
    by_id: HashMap<TypeId, TypeRegistration>,
    by_name: HashMap<String, TypeId>,
    next_type_id: TypeId,
}

impl Registry {
    /// Creates a new empty registry.
    pub fn new() -> Self {
        Self {
            by_id: HashMap::new(),
            by_name: HashMap::new(),
            next_type_id: 128, // User types start at 128
        }
    }

    /// Registers a type with automatic ID assignment.
    pub fn register<T>(&mut self, name: &str, encoder: Encoder<T>, decoder: Decoder<T>) -> TypeId
    where
        T: 'static + Send + Sync,
    {
        self.register_with_id(name, self.next_type_id, encoder, decoder)
    }

    /// Registers a type with a specific ID.
    pub fn register_with_id<T>(
        &mut self,
        name: &str,
        type_id: TypeId,
        encoder: Encoder<T>,
        decoder: Decoder<T>,
    ) -> TypeId
    where
        T: 'static + Send + Sync,
    {
        let name_owned = name.to_string();
        let any_encoder: AnyEncoder = Box::new(move |writer, value| {
            let typed = value.downcast_ref::<T>().ok_or_else(|| {
                Error::custom(format!("Type mismatch for {}", name_owned))
            })?;
            encoder(writer, typed)
        });

        let any_decoder: AnyDecoder = Box::new(move |reader| {
            let value = decoder(reader)?;
            Ok(Box::new(value) as Box<dyn std::any::Any + Send + Sync>)
        });

        let registration = TypeRegistration {
            type_id,
            name: name.to_string(),
            encoder: any_encoder,
            decoder: any_decoder,
        };

        self.by_id.insert(type_id, registration);
        self.by_name.insert(name.to_string(), type_id);

        if type_id >= self.next_type_id {
            self.next_type_id = type_id + 1;
        }

        type_id
    }

    /// Gets the type ID for a registered type name.
    pub fn get_type_id(&self, name: &str) -> Result<TypeId> {
        self.by_name
            .get(name)
            .copied()
            .ok_or_else(|| Error::TypeNotRegistered(name.to_string()))
    }

    /// Gets the type name for a registered type ID.
    pub fn get_type_name(&self, type_id: TypeId) -> Result<&str> {
        self.by_id
            .get(&type_id)
            .map(|r| r.name.as_str())
            .ok_or_else(|| Error::UnknownTypeId(type_id))
    }

    /// Checks if a type name is registered.
    pub fn is_registered(&self, name: &str) -> bool {
        self.by_name.contains_key(name)
    }

    /// Checks if a type ID is registered.
    pub fn is_registered_id(&self, type_id: TypeId) -> bool {
        self.by_id.contains_key(&type_id)
    }

    /// Encodes a polymorphic value with its type ID.
    pub fn encode_polymorphic<T>(
        &self,
        writer: &mut Writer,
        field_number: u32,
        name: &str,
        value: &T,
    ) -> Result<()>
    where
        T: 'static,
    {
        let type_id = self.get_type_id(name)?;
        let reg = self.by_id.get(&type_id).unwrap();

        // Write field tag with TypeRef wire type
        writer.write_tag(field_number, WireType::TypeRef)?;

        // Write type ID
        writer.write_varint(type_id)?;

        // Create a temporary writer for the value
        let mut temp_writer = Writer::new();
        (reg.encoder)(&mut temp_writer, value)?;

        // Write length-prefixed value bytes
        writer.write_length_prefixed_bytes(temp_writer.as_bytes())?;

        Ok(())
    }

    /// Decodes a polymorphic value and returns its type name.
    pub fn decode_polymorphic(
        &self,
        reader: &mut Reader,
    ) -> Result<(String, Box<dyn std::any::Any + Send + Sync>)> {
        // Read type ID
        let type_id = reader.read_varint()?;

        let reg = self
            .by_id
            .get(&type_id)
            .ok_or_else(|| Error::UnknownTypeId(type_id))?;

        // Read length-prefixed value
        let length = reader.read_varint()? as usize;
        let mut sub_reader = reader.sub_reader(length)?;

        // Decode value
        let value = (reg.decoder)(&mut sub_reader)?;

        Ok((reg.name.clone(), value))
    }

    /// Clears all registrations.
    pub fn clear(&mut self) {
        self.by_id.clear();
        self.by_name.clear();
        self.next_type_id = 128;
    }
}

impl Default for Registry {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[derive(Debug, PartialEq)]
    struct TestMessage {
        value: i32,
        name: String,
    }

    fn encode_test_message(writer: &mut Writer, msg: &TestMessage) -> Result<()> {
        writer.write_int32_field(1, msg.value)?;
        writer.write_string_field(2, &msg.name)?;
        Ok(())
    }

    fn decode_test_message(reader: &mut Reader) -> Result<TestMessage> {
        let mut value = 0;
        let mut name = String::new();

        while reader.has_more() {
            let tag = reader.read_tag()?;
            match tag.field_number {
                1 => value = reader.read_int32()?,
                2 => name = reader.read_string()?.to_string(),
                _ => reader.skip_field(tag.wire_type)?,
            }
        }

        Ok(TestMessage { value, name })
    }

    #[test]
    fn test_registry_register() {
        let mut registry = Registry::new();
        let type_id = registry.register("TestMessage", encode_test_message, decode_test_message);

        assert_eq!(type_id, 128);
        assert!(registry.is_registered("TestMessage"));
        assert!(registry.is_registered_id(128));
        assert_eq!(registry.get_type_id("TestMessage").unwrap(), 128);
        assert_eq!(registry.get_type_name(128).unwrap(), "TestMessage");
    }
}
