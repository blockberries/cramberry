//! Type registry for polymorphic encoding/decoding.

use std::collections::HashMap;
use std::sync::RwLock;

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

/// Internal data structure holding registry state.
struct RegistryInner {
    by_id: HashMap<TypeId, TypeRegistration>,
    by_name: HashMap<String, TypeId>,
    next_type_id: TypeId,
}

/// Registry manages type registrations for polymorphic encoding/decoding.
/// Thread-safe: uses RwLock for concurrent read access.
pub struct Registry {
    inner: RwLock<RegistryInner>,
}

impl Registry {
    /// Creates a new empty registry.
    pub fn new() -> Self {
        Self {
            inner: RwLock::new(RegistryInner {
                by_id: HashMap::new(),
                by_name: HashMap::new(),
                next_type_id: 128, // User types start at 128
            }),
        }
    }

    /// Registers a type with automatic ID assignment.
    /// Thread-safe: acquires write lock.
    pub fn register<T>(&self, name: &str, encoder: Encoder<T>, decoder: Decoder<T>) -> TypeId
    where
        T: 'static + Send + Sync,
    {
        let mut inner = self.inner.write().unwrap();
        let type_id = inner.next_type_id;
        self.register_with_id_inner(&mut inner, name, type_id, encoder, decoder)
    }

    /// Registers a type with a specific ID.
    /// Thread-safe: acquires write lock.
    pub fn register_with_id<T>(
        &self,
        name: &str,
        type_id: TypeId,
        encoder: Encoder<T>,
        decoder: Decoder<T>,
    ) -> TypeId
    where
        T: 'static + Send + Sync,
    {
        let mut inner = self.inner.write().unwrap();
        self.register_with_id_inner(&mut inner, name, type_id, encoder, decoder)
    }

    /// Internal registration helper (caller must hold write lock).
    fn register_with_id_inner<T>(
        &self,
        inner: &mut RegistryInner,
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

        inner.by_id.insert(type_id, registration);
        inner.by_name.insert(name.to_string(), type_id);

        if type_id >= inner.next_type_id {
            inner.next_type_id = type_id + 1;
        }

        type_id
    }

    /// Registers a type if not already registered; returns existing ID if registered.
    /// Thread-safe: uses read-write pattern for efficiency.
    pub fn register_or_get<T>(&self, name: &str, encoder: Encoder<T>, decoder: Decoder<T>) -> TypeId
    where
        T: 'static + Send + Sync,
    {
        // Fast path: check if already registered (read lock)
        {
            let inner = self.inner.read().unwrap();
            if let Some(&type_id) = inner.by_name.get(name) {
                return type_id;
            }
        }

        // Slow path: register with write lock
        let mut inner = self.inner.write().unwrap();

        // Double-check after acquiring write lock
        if let Some(&type_id) = inner.by_name.get(name) {
            return type_id;
        }

        let type_id = inner.next_type_id;
        self.register_with_id_inner(&mut inner, name, type_id, encoder, decoder)
    }

    /// Gets the type ID for a registered type name.
    /// Thread-safe: acquires read lock.
    pub fn get_type_id(&self, name: &str) -> Result<TypeId> {
        let inner = self.inner.read().unwrap();
        inner.by_name
            .get(name)
            .copied()
            .ok_or_else(|| Error::TypeNotRegistered(name.to_string()))
    }

    /// Gets the type name for a registered type ID.
    /// Thread-safe: acquires read lock.
    pub fn get_type_name(&self, type_id: TypeId) -> Result<String> {
        let inner = self.inner.read().unwrap();
        inner.by_id
            .get(&type_id)
            .map(|r| r.name.clone())
            .ok_or_else(|| Error::UnknownTypeId(type_id))
    }

    /// Checks if a type name is registered.
    /// Thread-safe: acquires read lock.
    pub fn is_registered(&self, name: &str) -> bool {
        let inner = self.inner.read().unwrap();
        inner.by_name.contains_key(name)
    }

    /// Checks if a type ID is registered.
    /// Thread-safe: acquires read lock.
    pub fn is_registered_id(&self, type_id: TypeId) -> bool {
        let inner = self.inner.read().unwrap();
        inner.by_id.contains_key(&type_id)
    }

    /// Encodes a polymorphic value with its type ID.
    /// In V2 format, type references are encoded as Bytes with [type_id + length-prefixed data].
    /// Thread-safe: acquires read lock.
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
        let inner = self.inner.read().unwrap();
        let type_id = inner.by_name
            .get(name)
            .copied()
            .ok_or_else(|| Error::TypeNotRegistered(name.to_string()))?;
        let reg = inner.by_id.get(&type_id).unwrap();

        // Write field tag with Bytes wire type (V2 format)
        writer.write_tag(field_number, WireType::Bytes)?;

        // Create a temporary writer for the type ref content
        let mut type_ref_writer = Writer::new();
        type_ref_writer.write_varint(type_id)?;

        // Create another temporary writer for the value
        let mut value_writer = Writer::new();
        (reg.encoder)(&mut value_writer, value)?;

        // Write length-prefixed value bytes to type_ref_writer
        type_ref_writer.write_length_prefixed_bytes(value_writer.as_bytes())?;

        // Write the entire type ref as length-prefixed bytes
        writer.write_length_prefixed_bytes(type_ref_writer.as_bytes())?;

        Ok(())
    }

    /// Decodes a polymorphic value and returns its type name.
    /// Thread-safe: acquires read lock.
    pub fn decode_polymorphic(
        &self,
        reader: &mut Reader,
    ) -> Result<(String, Box<dyn std::any::Any + Send + Sync>)> {
        // Read type ID
        let type_id = reader.read_varint()?;

        let inner = self.inner.read().unwrap();
        let reg = inner
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
    /// Thread-safe: acquires write lock.
    pub fn clear(&self) {
        let mut inner = self.inner.write().unwrap();
        inner.by_id.clear();
        inner.by_name.clear();
        inner.next_type_id = 128;
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
        writer.write_end_marker()?;
        Ok(())
    }

    fn decode_test_message(reader: &mut Reader) -> Result<TestMessage> {
        let mut value = 0;
        let mut name = String::new();

        while reader.has_more() {
            let tag = reader.read_tag()?;
            if Reader::is_end_marker(&tag) {
                break;
            }
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
        let registry = Registry::new();
        let type_id = registry.register("TestMessage", encode_test_message, decode_test_message);

        assert_eq!(type_id, 128);
        assert!(registry.is_registered("TestMessage"));
        assert!(registry.is_registered_id(128));
        assert_eq!(registry.get_type_id("TestMessage").unwrap(), 128);
        assert_eq!(registry.get_type_name(128).unwrap(), "TestMessage");
    }

    #[test]
    fn test_registry_thread_safe() {
        use std::sync::Arc;
        use std::thread;

        let registry = Arc::new(Registry::new());

        // Register type from main thread
        registry.register("TestMessage", encode_test_message, decode_test_message);

        // Access from multiple threads
        let handles: Vec<_> = (0..4).map(|_| {
            let reg = Arc::clone(&registry);
            thread::spawn(move || {
                assert!(reg.is_registered("TestMessage"));
                assert_eq!(reg.get_type_id("TestMessage").unwrap(), 128);
            })
        }).collect();

        for handle in handles {
            handle.join().unwrap();
        }
    }

    #[test]
    fn test_register_or_get() {
        let registry = Registry::new();

        // First registration
        let id1 = registry.register_or_get("TestMessage", encode_test_message, decode_test_message);
        assert_eq!(id1, 128);

        // Subsequent call returns the same ID
        let id2 = registry.register_or_get("TestMessage", encode_test_message, decode_test_message);
        assert_eq!(id2, 128);
        assert_eq!(id1, id2);
    }
}
