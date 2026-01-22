import { BufferUnderflowError, InvalidWireTypeError, DecodeError } from "./errors";
import { WireType, FieldTag, decodeTag, zigzagDecode, zigzagDecode64 } from "./types";

/**
 * Reader decodes Cramberry data from a binary buffer.
 */
export class Reader {
  private buffer: Uint8Array;
  private view: DataView;
  private pos: number;
  private end: number;

  constructor(data: Uint8Array) {
    this.buffer = data;
    this.view = new DataView(data.buffer, data.byteOffset, data.byteLength);
    this.pos = 0;
    this.end = data.length;
  }

  /**
   * Returns the current position in the buffer.
   */
  get position(): number {
    return this.pos;
  }

  /**
   * Returns the number of bytes remaining.
   */
  get remaining(): number {
    return this.end - this.pos;
  }

  /**
   * Returns true if there is more data to read.
   */
  get hasMore(): boolean {
    return this.pos < this.end;
  }

  /**
   * Checks if there are enough bytes available.
   */
  private checkAvailable(needed: number): void {
    if (this.pos + needed > this.end) {
      throw new BufferUnderflowError(needed, this.remaining);
    }
  }

  /**
   * Reads a raw byte.
   */
  readByte(): number {
    this.checkAvailable(1);
    return this.buffer[this.pos++];
  }

  /**
   * Reads raw bytes.
   */
  readBytes(length: number): Uint8Array {
    this.checkAvailable(length);
    const bytes = this.buffer.subarray(this.pos, this.pos + length);
    this.pos += length;
    return bytes;
  }

  /**
   * Reads an unsigned varint (LEB128).
   */
  readVarint(): number {
    let result = 0;
    let shift = 0;

    while (shift < 35) {
      this.checkAvailable(1);
      const b = this.buffer[this.pos++];
      result |= (b & 0x7f) << shift;
      if ((b & 0x80) === 0) {
        return result >>> 0; // Ensure unsigned
      }
      shift += 7;
    }

    throw new DecodeError("Varint overflow");
  }

  /**
   * Reads an unsigned 64-bit varint (LEB128).
   */
  readVarint64(): bigint {
    let result = 0n;
    let shift = 0n;

    while (shift < 70n) {
      this.checkAvailable(1);
      const b = BigInt(this.buffer[this.pos++]);
      result |= (b & 0x7fn) << shift;
      if ((b & 0x80n) === 0n) {
        return result;
      }
      shift += 7n;
    }

    throw new DecodeError("Varint64 overflow");
  }

  /**
   * Reads a signed varint using ZigZag decoding.
   */
  readSVarint(): number {
    return zigzagDecode(this.readVarint());
  }

  /**
   * Reads a signed 64-bit varint using ZigZag decoding.
   */
  readSVarint64(): bigint {
    return zigzagDecode64(this.readVarint64());
  }

  /**
   * Reads a field tag.
   */
  readTag(): FieldTag {
    const tag = this.readVarint();
    return decodeTag(tag);
  }

  /**
   * Reads a boolean.
   */
  readBool(): boolean {
    return this.readByte() !== 0;
  }

  /**
   * Reads a 32-bit signed integer.
   */
  readInt32(): number {
    return this.readSVarint();
  }

  /**
   * Reads a 64-bit signed integer.
   */
  readInt64(): bigint {
    return this.readSVarint64();
  }

  /**
   * Reads a 32-bit unsigned integer.
   */
  readUint32(): number {
    return this.readVarint();
  }

  /**
   * Reads a 64-bit unsigned integer.
   */
  readUint64(): bigint {
    return this.readVarint64();
  }

  /**
   * Reads a 32-bit float (IEEE 754).
   */
  readFloat32(): number {
    this.checkAvailable(4);
    const value = this.view.getFloat32(this.pos, true); // Little-endian
    this.pos += 4;
    return value;
  }

  /**
   * Reads a 64-bit float (IEEE 754).
   */
  readFloat64(): number {
    this.checkAvailable(8);
    const value = this.view.getFloat64(this.pos, true); // Little-endian
    this.pos += 8;
    return value;
  }

  /**
   * Reads a fixed 32-bit value.
   */
  readFixed32(): number {
    this.checkAvailable(4);
    const value = this.view.getUint32(this.pos, true); // Little-endian
    this.pos += 4;
    return value;
  }

  /**
   * Reads a fixed 64-bit value.
   */
  readFixed64(): bigint {
    this.checkAvailable(8);
    const value = this.view.getBigUint64(this.pos, true); // Little-endian
    this.pos += 8;
    return value;
  }

  /**
   * Reads a length-prefixed string.
   */
  readString(): string {
    const length = this.readVarint();
    const bytes = this.readBytes(length);
    const decoder = new TextDecoder();
    return decoder.decode(bytes);
  }

  /**
   * Reads length-prefixed bytes.
   */
  readLengthPrefixedBytes(): Uint8Array {
    const length = this.readVarint();
    return this.readBytes(length);
  }

  /**
   * Skips a field based on its wire type.
   */
  skipField(wireType: WireType): void {
    switch (wireType) {
      case WireType.Varint:
      case WireType.SVarint:
        this.readVarint();
        break;
      case WireType.Fixed64:
        this.checkAvailable(8);
        this.pos += 8;
        break;
      case WireType.Bytes:
        const length = this.readVarint();
        this.checkAvailable(length);
        this.pos += length;
        break;
      case WireType.Fixed32:
        this.checkAvailable(4);
        this.pos += 4;
        break;
      case WireType.TypeRef:
        this.readVarint(); // Type ID
        // Then skip the actual value (as Bytes)
        const valueLength = this.readVarint();
        this.checkAvailable(valueLength);
        this.pos += valueLength;
        break;
      default:
        throw new InvalidWireTypeError(-1, wireType);
    }
  }

  /**
   * Creates a sub-reader for reading nested messages.
   */
  subReader(length: number): Reader {
    this.checkAvailable(length);
    const sub = new Reader(this.buffer.subarray(this.pos, this.pos + length));
    this.pos += length;
    return sub;
  }
}
