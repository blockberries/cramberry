import { DuplicateTypeRegistrationError, TypeNotRegisteredError, UnknownTypeError } from "./errors";
import { Reader } from "./reader";
import { Writer } from "./writer";
import { TypeID, WireType } from "./types";

/**
 * Encoder function for a registered type.
 */
export type Encoder<T> = (writer: Writer, value: T) => void;

/**
 * Decoder function for a registered type.
 */
export type Decoder<T> = (reader: Reader) => T;

/**
 * Sizer function to compute encoded size.
 */
export type Sizer<T> = (value: T) => number;

/**
 * Registration information for a type.
 */
interface TypeRegistration {
  typeId: TypeID;
  name: string;
  encoder: Encoder<unknown>;
  decoder: Decoder<unknown>;
  sizer?: Sizer<unknown>;
}

/**
 * Registry manages type registrations for polymorphic encoding/decoding.
 */
export class Registry {
  private byId: Map<TypeID, TypeRegistration> = new Map();
  private byName: Map<string, TypeRegistration> = new Map();
  private nextTypeId: TypeID = 128; // User types start at 128

  /**
   * Registers a type with the registry.
   *
   * Throws DuplicateTypeRegistrationError if `name` or `typeId` is
   * already bound to a different registration. The previous "last
   * write wins" behaviour silently masked schema bugs (two messages
   * claiming the same wire ID); Go's `Registry.RegisterTypeWithID`
   * returns the same error.
   *
   * Re-registering the SAME (name, typeId) pair is allowed and is a
   * no-op; this lets idempotent module-loading patterns work.
   */
  register<T>(
    name: string,
    encoder: Encoder<T>,
    decoder: Decoder<T>,
    typeId?: TypeID,
    sizer?: Sizer<T>
  ): TypeID {
    const id = typeId ?? this.nextTypeId++;

    const existingById = this.byId.get(id);
    if (existingById !== undefined && existingById.name !== name) {
      throw new DuplicateTypeRegistrationError(
        `type ID ${id} is already registered to ${existingById.name}, cannot reassign to ${name}`,
      );
    }
    const existingByName = this.byName.get(name);
    if (existingByName !== undefined && existingByName.typeId !== id) {
      throw new DuplicateTypeRegistrationError(
        `type name ${name} is already registered with ID ${existingByName.typeId}, cannot reassign to ${id}`,
      );
    }

    const registration: TypeRegistration = {
      typeId: id,
      name,
      encoder: encoder as Encoder<unknown>,
      decoder: decoder as Decoder<unknown>,
      sizer: sizer as Sizer<unknown> | undefined,
    };

    this.byId.set(id, registration);
    this.byName.set(name, registration);

    return id;
  }

  /**
   * Gets the type ID for a registered type name.
   */
  getTypeId(name: string): TypeID {
    const reg = this.byName.get(name);
    if (!reg) {
      throw new TypeNotRegisteredError(name);
    }
    return reg.typeId;
  }

  /**
   * Gets the type name for a registered type ID.
   */
  getTypeName(typeId: TypeID): string {
    const reg = this.byId.get(typeId);
    if (!reg) {
      throw new UnknownTypeError(typeId);
    }
    return reg.name;
  }

  /**
   * Encodes a polymorphic value with its type ID.
   *
   * Wire shape: [field tag (Bytes)] [typeID varint] [value bytes (length-prefixed)].
   * The Bytes wire type is the canonical carrier for variable-length payloads;
   * the typeID + value-bytes pair forms the body that the matching
   * `decodePolymorphic` reads.
   */
  encodePolymorphic<T>(writer: Writer, fieldNumber: number, name: string, value: T): void {
    const reg = this.byName.get(name);
    if (!reg) {
      throw new TypeNotRegisteredError(name);
    }

    // Field tag: Bytes is the wire type for length-prefixed payloads.
    writer.writeTag(fieldNumber, WireType.Bytes);

    writer.writeVarint(reg.typeId);

    // Length-prefix the encoded value using beginMessage/endMessage so
    // the encoder writes straight into `writer` instead of a fresh
    // sub-Writer (one Uint8Array + DataView allocation eliminated per
    // polymorphic field).
    const cp = writer.beginMessage();
    reg.encoder(writer, value);
    writer.endMessage(cp);
  }

  /**
   * Decodes a polymorphic value.
   */
  decodePolymorphic(reader: Reader): { name: string; value: unknown } {
    // Read type ID
    const typeId = reader.readVarint();

    const reg = this.byId.get(typeId);
    if (!reg) {
      throw new UnknownTypeError(typeId);
    }

    // Read length-prefixed value
    const length = reader.readVarint();
    const subReader = reader.subReader(length);

    // Decode value
    const value = reg.decoder(subReader);

    return { name: reg.name, value };
  }

  /**
   * Checks if a type name is registered.
   */
  isRegistered(name: string): boolean {
    return this.byName.has(name);
  }

  /**
   * Checks if a type ID is registered.
   */
  isRegisteredId(typeId: TypeID): boolean {
    return this.byId.has(typeId);
  }

  /**
   * Clears all registrations.
   */
  clear(): void {
    this.byId.clear();
    this.byName.clear();
    this.nextTypeId = 128;
  }
}

/**
 * Global default registry instance.
 */
export const defaultRegistry = new Registry();

/**
 * Registers a type with the default registry.
 */
export function register<T>(
  name: string,
  encoder: Encoder<T>,
  decoder: Decoder<T>,
  typeId?: TypeID,
  sizer?: Sizer<T>
): TypeID {
  return defaultRegistry.register(name, encoder, decoder, typeId, sizer);
}
