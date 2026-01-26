import { BufferUnderflowError, InvalidWireTypeError, DecodeError } from "./errors";
import { WireType, TypeID, FieldTag, decodeTag, zigzagDecode, zigzagDecode64 } from "./types";

// Module-level singleton to avoid repeated instantiation
const textDecoder = new TextDecoder();

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
   * Maximum number of bytes for a varint (64-bit value encoded as varint).
   * A uint64 has 64 bits, and each varint byte encodes 7 bits,
   * so we need ceil(64/7) = 10 bytes maximum.
   */
  private static readonly MAX_VARINT_BYTES = 10;

  /**
   * Reads an unsigned varint (LEB128).
   * For 32-bit values, this uses the same 10-byte limit as 64-bit for consistency,
   * but the result is capped to 32 bits.
   */
  readVarint(): number {
    let result = 0;
    let shift = 0;

    for (let i = 0; i < Reader.MAX_VARINT_BYTES; i++) {
      this.checkAvailable(1);
      const b = this.buffer[this.pos++];

      // At the 5th byte (index 4), we've consumed 28 bits.
      // The 5th byte can only contribute 4 more bits for a 32-bit value.
      if (i === 4 && (b & 0xf0) !== 0) {
        throw new DecodeError("Varint overflow: value exceeds 32 bits");
      }

      result |= (b & 0x7f) << shift;
      if ((b & 0x80) === 0) {
        return result >>> 0; // Ensure unsigned
      }
      shift += 7;
    }

    throw new DecodeError("Varint overflow: exceeded 10 bytes");
  }

  /**
   * Reads an unsigned 64-bit varint (LEB128).
   * Uses a maximum of 10 bytes, consistent with protobuf and Go implementation.
   */
  readVarint64(): bigint {
    let result = 0n;
    let shift = 0n;

    for (let i = 0; i < Reader.MAX_VARINT_BYTES; i++) {
      this.checkAvailable(1);
      const b = this.buffer[this.pos++];
      const bBigInt = BigInt(b);

      // At the 10th byte (index 9), we've consumed 63 bits.
      // The 10th byte can only contribute 1 more bit (bit 63 of uint64).
      if (i === 9) {
        // If continuation bit is set, we'd need 11+ bytes
        if (b >= 0x80) {
          throw new DecodeError("Varint64 overflow: exceeded 10 bytes");
        }
        // If data portion is > 1, value would overflow uint64
        if (b > 1) {
          throw new DecodeError("Varint64 overflow: 10th byte must be 0 or 1");
        }
      }

      result |= (bBigInt & 0x7fn) << shift;
      if ((b & 0x80) === 0) {
        return result;
      }
      shift += 7n;
    }

    throw new DecodeError("Varint64 overflow: exceeded 10 bytes");
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
   * Reads a 64-bit signed integer as bigint.
   * Use this for full 64-bit precision.
   */
  readInt64(): bigint {
    return this.readSVarint64();
  }

  /**
   * Reads a 64-bit signed integer as a JavaScript number.
   *
   * WARNING: JavaScript numbers can only safely represent integers
   * up to Number.MAX_SAFE_INTEGER (2^53-1). Values larger than this
   * will lose precision.
   *
   * For values that may exceed 2^53-1, use readInt64() instead which
   * returns a bigint with full 64-bit precision.
   *
   * @param warnOnPrecisionLoss - If true (default), logs a warning when
   *                              precision loss occurs
   */
  readInt64AsNumber(warnOnPrecisionLoss: boolean = true): number {
    const value = this.readSVarint64();
    if (warnOnPrecisionLoss) {
      if (value > BigInt(Number.MAX_SAFE_INTEGER) ||
          value < BigInt(Number.MIN_SAFE_INTEGER)) {
        console.warn(
          `cramberry: int64 value ${value} exceeds safe integer range ` +
          `(${Number.MIN_SAFE_INTEGER} to ${Number.MAX_SAFE_INTEGER}), ` +
          `precision may be lost. Use readInt64() for full precision.`
        );
      }
    }
    return Number(value);
  }

  /**
   * Reads a 32-bit unsigned integer.
   */
  readUint32(): number {
    return this.readVarint();
  }

  /**
   * Reads a 64-bit unsigned integer as bigint.
   * Use this for full 64-bit precision.
   */
  readUint64(): bigint {
    return this.readVarint64();
  }

  /**
   * Reads a 64-bit unsigned integer as a JavaScript number.
   *
   * WARNING: JavaScript numbers can only safely represent integers
   * up to Number.MAX_SAFE_INTEGER (2^53-1). Values larger than this
   * will lose precision.
   *
   * For values that may exceed 2^53-1, use readUint64() instead which
   * returns a bigint with full 64-bit precision.
   *
   * @param warnOnPrecisionLoss - If true (default), logs a warning when
   *                              precision loss occurs
   */
  readUint64AsNumber(warnOnPrecisionLoss: boolean = true): number {
    const value = this.readVarint64();
    if (warnOnPrecisionLoss) {
      if (value > BigInt(Number.MAX_SAFE_INTEGER)) {
        console.warn(
          `cramberry: uint64 value ${value} exceeds safe integer range ` +
          `(max ${Number.MAX_SAFE_INTEGER}), precision may be lost. ` +
          `Use readUint64() for full precision.`
        );
      }
    }
    return Number(value);
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
    return textDecoder.decode(bytes);
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

  /**
   * Reads a type reference (for polymorphic types).
   * Returns the type ID and a sub-reader for the value data.
   * Format: [type_id: varint] [data_length: varint] [data: bytes]
   */
  readTypeRef(): { typeId: TypeID; reader: Reader } {
    const typeId = this.readVarint();
    const length = this.readVarint();
    const reader = this.subReader(length);
    return { typeId, reader };
  }
}
