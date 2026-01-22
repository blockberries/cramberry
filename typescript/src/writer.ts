import { BufferOverflowError } from "./errors";
import { WireType, TypeID, encodeTag, zigzagEncode, zigzagEncode64 } from "./types";

const INITIAL_CAPACITY = 256;
const GROWTH_FACTOR = 2;

// Module-level singleton to avoid repeated instantiation
const textEncoder = new TextEncoder();

/**
 * Writer encodes Cramberry data into a binary buffer.
 */
export class Writer {
  private buffer: Uint8Array;
  private view: DataView;
  private pos: number;

  constructor(initialCapacity: number = INITIAL_CAPACITY) {
    this.buffer = new Uint8Array(initialCapacity);
    this.view = new DataView(this.buffer.buffer);
    this.pos = 0;
  }

  /**
   * Returns the current position in the buffer.
   */
  get position(): number {
    return this.pos;
  }

  /**
   * Returns the encoded bytes.
   */
  bytes(): Uint8Array {
    return this.buffer.subarray(0, this.pos);
  }

  /**
   * Resets the writer for reuse.
   */
  reset(): void {
    this.pos = 0;
  }

  /**
   * Ensures the buffer has at least the specified capacity.
   */
  private ensureCapacity(needed: number): void {
    const required = this.pos + needed;
    if (required <= this.buffer.length) {
      return;
    }

    let newCapacity = this.buffer.length * GROWTH_FACTOR;
    while (newCapacity < required) {
      newCapacity *= GROWTH_FACTOR;
    }

    const newBuffer = new Uint8Array(newCapacity);
    newBuffer.set(this.buffer);
    this.buffer = newBuffer;
    this.view = new DataView(this.buffer.buffer);
  }

  /**
   * Writes a field tag.
   */
  writeTag(fieldNumber: number, wireType: WireType): void {
    this.writeVarint(encodeTag(fieldNumber, wireType));
  }

  /**
   * Writes a raw byte.
   */
  writeByte(value: number): void {
    this.ensureCapacity(1);
    this.buffer[this.pos++] = value & 0xff;
  }

  /**
   * Writes raw bytes.
   */
  writeBytes(data: Uint8Array): void {
    this.ensureCapacity(data.length);
    this.buffer.set(data, this.pos);
    this.pos += data.length;
  }

  /**
   * Writes an unsigned varint (LEB128).
   */
  writeVarint(value: number): void {
    this.ensureCapacity(5); // Max 5 bytes for 32-bit
    while (value > 0x7f) {
      this.buffer[this.pos++] = (value & 0x7f) | 0x80;
      value >>>= 7;
    }
    this.buffer[this.pos++] = value;
  }

  /**
   * Writes an unsigned 64-bit varint (LEB128).
   */
  writeVarint64(value: bigint): void {
    this.ensureCapacity(10); // Max 10 bytes for 64-bit
    while (value > 0x7fn) {
      this.buffer[this.pos++] = Number(value & 0x7fn) | 0x80;
      value >>= 7n;
    }
    this.buffer[this.pos++] = Number(value);
  }

  /**
   * Writes a signed varint using ZigZag encoding.
   */
  writeSVarint(value: number): void {
    this.writeVarint(zigzagEncode(value));
  }

  /**
   * Writes a signed 64-bit varint using ZigZag encoding.
   */
  writeSVarint64(value: bigint): void {
    this.writeVarint64(zigzagEncode64(value));
  }

  /**
   * Writes a boolean.
   */
  writeBool(value: boolean): void {
    this.writeByte(value ? 1 : 0);
  }

  /**
   * Writes a 32-bit signed integer.
   */
  writeInt32(value: number): void {
    this.writeSVarint(value);
  }

  /**
   * Writes a 64-bit signed integer.
   */
  writeInt64(value: bigint): void {
    this.writeSVarint64(value);
  }

  /**
   * Writes a 32-bit unsigned integer.
   */
  writeUint32(value: number): void {
    this.writeVarint(value);
  }

  /**
   * Writes a 64-bit unsigned integer.
   */
  writeUint64(value: bigint): void {
    this.writeVarint64(value);
  }

  /**
   * Writes a 32-bit float (IEEE 754).
   */
  writeFloat32(value: number): void {
    this.ensureCapacity(4);
    this.view.setFloat32(this.pos, value, true); // Little-endian
    this.pos += 4;
  }

  /**
   * Writes a 64-bit float (IEEE 754).
   */
  writeFloat64(value: number): void {
    this.ensureCapacity(8);
    this.view.setFloat64(this.pos, value, true); // Little-endian
    this.pos += 8;
  }

  /**
   * Writes a fixed 32-bit value.
   */
  writeFixed32(value: number): void {
    this.ensureCapacity(4);
    this.view.setUint32(this.pos, value, true); // Little-endian
    this.pos += 4;
  }

  /**
   * Writes a fixed 64-bit value.
   */
  writeFixed64(value: bigint): void {
    this.ensureCapacity(8);
    this.view.setBigUint64(this.pos, value, true); // Little-endian
    this.pos += 8;
  }

  /**
   * Writes a length-prefixed string.
   */
  writeString(value: string): void {
    const bytes = textEncoder.encode(value);
    this.writeVarint(bytes.length);
    this.writeBytes(bytes);
  }

  /**
   * Writes length-prefixed bytes.
   */
  writeLengthPrefixedBytes(data: Uint8Array): void {
    this.writeVarint(data.length);
    this.writeBytes(data);
  }

  /**
   * Writes a tagged field with boolean value.
   */
  writeBoolField(fieldNumber: number, value: boolean): void {
    this.writeTag(fieldNumber, WireType.Varint);
    this.writeBool(value);
  }

  /**
   * Writes a tagged field with int32 value.
   */
  writeInt32Field(fieldNumber: number, value: number): void {
    this.writeTag(fieldNumber, WireType.SVarint);
    this.writeInt32(value);
  }

  /**
   * Writes a tagged field with int64 value.
   */
  writeInt64Field(fieldNumber: number, value: bigint): void {
    this.writeTag(fieldNumber, WireType.SVarint);
    this.writeInt64(value);
  }

  /**
   * Writes a tagged field with uint32 value.
   */
  writeUint32Field(fieldNumber: number, value: number): void {
    this.writeTag(fieldNumber, WireType.Varint);
    this.writeUint32(value);
  }

  /**
   * Writes a tagged field with uint64 value.
   */
  writeUint64Field(fieldNumber: number, value: bigint): void {
    this.writeTag(fieldNumber, WireType.Varint);
    this.writeUint64(value);
  }

  /**
   * Writes a tagged field with float32 value.
   */
  writeFloat32Field(fieldNumber: number, value: number): void {
    this.writeTag(fieldNumber, WireType.Fixed32);
    this.writeFloat32(value);
  }

  /**
   * Writes a tagged field with float64 value.
   */
  writeFloat64Field(fieldNumber: number, value: number): void {
    this.writeTag(fieldNumber, WireType.Fixed64);
    this.writeFloat64(value);
  }

  /**
   * Writes a tagged field with string value.
   */
  writeStringField(fieldNumber: number, value: string): void {
    this.writeTag(fieldNumber, WireType.Bytes);
    this.writeString(value);
  }

  /**
   * Writes a tagged field with bytes value.
   */
  writeBytesField(fieldNumber: number, value: Uint8Array): void {
    this.writeTag(fieldNumber, WireType.Bytes);
    this.writeLengthPrefixedBytes(value);
  }

  /**
   * Writes a type reference (for polymorphic types).
   * Format: [type_id: varint] [data_length: varint] [data: bytes]
   */
  writeTypeRef(typeId: TypeID, data: Uint8Array): void {
    this.writeVarint(typeId);
    this.writeLengthPrefixedBytes(data);
  }

  /**
   * Writes a tagged field with a type reference value.
   */
  writeTypeRefField(fieldNumber: number, typeId: TypeID, data: Uint8Array): void {
    this.writeTag(fieldNumber, WireType.TypeRef);
    this.writeTypeRef(typeId, data);
  }
}
