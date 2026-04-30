import { WireType, TypeID, encodeTag, zigzagEncode, zigzagEncode64, END_MARKER } from "./types";

const INITIAL_CAPACITY = 1024;
const GROWTH_FACTOR = 2;

// Number of bytes reserved up front by beginMessage() for the length
// prefix. 2 covers messages up to 16383 bytes (the modal nested-
// message size); larger bodies retroactively grow the placeholder.
const LENGTH_PLACEHOLDER = 2;

// Module-level singleton to avoid repeated instantiation
const textEncoder = new TextEncoder();

/** uvarintSize returns the number of bytes a u32 takes in LEB128. */
function uvarintSize(v: number): number {
  if (v < 1 << 7) return 1;
  if (v < 1 << 14) return 2;
  if (v < 1 << 21) return 3;
  if (v < 1 << 28) return 4;
  return 5;
}

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
   * Writes a compact field tag.
   */
  writeTag(fieldNumber: number, wireType: WireType): void {
    const tagBytes = encodeTag(fieldNumber, wireType);
    this.writeBytes(tagBytes);
  }

  /**
   * Writes the end marker (0x00) to signal end of struct fields.
   */
  writeEndMarker(): void {
    this.writeByte(END_MARKER);
  }

  /**
   * Begins a length-prefixed message. Reserves a small placeholder for
   * the length (2 bytes — covers messages up to 16383 bytes, which is
   * the modal nested-message size); endMessage() back-patches the
   * actual varint length and shifts the body if the reservation was
   * too tight.
   *
   * Returns a checkpoint that must be passed to endMessage(). The
   * returned offset points to the start of the placeholder, NOT the
   * start of the body (body starts at checkpoint + 2).
   *
   * Replaces the `new Writer(); ...; writer.writeLengthPrefixedBytes(sub.bytes())`
   * pattern that allocated a fresh sub-Writer per nested message /
   * array / map field. With back-patch we encode straight into the
   * parent's buffer; allocations drop to ~zero per nested level.
   */
  beginMessage(): number {
    this.ensureCapacity(LENGTH_PLACEHOLDER);
    const checkpoint = this.pos;
    this.buffer[this.pos++] = 0;
    this.buffer[this.pos++] = 0;
    return checkpoint;
  }

  /**
   * Finishes a length-prefixed message. `checkpoint` must be the value
   * previously returned by beginMessage().
   */
  endMessage(checkpoint: number): void {
    const msgStart = checkpoint + LENGTH_PLACEHOLDER;
    const msgLen = this.pos - msgStart;
    const lenSize = uvarintSize(msgLen);

    if (lenSize === LENGTH_PLACEHOLDER) {
      // Length fits exactly in the placeholder; encode in place.
      this.encodeVarintAt(checkpoint, msgLen);
      return;
    }
    if (lenSize < LENGTH_PLACEHOLDER) {
      // Common: 1-byte varint (msgLen < 128). Shift body left by 1 and truncate.
      const shift = LENGTH_PLACEHOLDER - lenSize;
      this.buffer.copyWithin(checkpoint + lenSize, msgStart, this.pos);
      this.pos -= shift;
      this.encodeVarintAt(checkpoint, msgLen);
      return;
    }
    // Rare: msgLen >= 16384, length needs 3+ varint bytes. Grow the
    // placeholder retroactively and memmove the body right.
    const extra = lenSize - LENGTH_PLACEHOLDER;
    this.ensureCapacity(extra);
    this.buffer.copyWithin(msgStart + extra, msgStart, this.pos);
    this.pos += extra;
    this.encodeVarintAt(checkpoint, msgLen);
  }

  /**
   * Writes a varint of `value` directly at `offset`. Used by
   * endMessage() to back-patch the length prefix without going through
   * pos/ensureCapacity (caller already reserved the bytes).
   */
  private encodeVarintAt(offset: number, value: number): void {
    let v = value;
    while (v > 0x7f) {
      this.buffer[offset++] = (v & 0x7f) | 0x80;
      v >>>= 7;
    }
    this.buffer[offset] = v;
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
   * Writes an unsigned varint (LEB128) up to 32 bits.
   *
   * Throws if `value` exceeds `0xFFFFFFFF` — JavaScript's `>>>= 7`
   * coerces its operand to a 32-bit integer, so a silent overflow
   * here would produce wire bytes the Go and Rust runtimes would
   * decode to a different value (and consensus splits). Use
   * `writeVarint64` for length prefixes and counts that may exceed
   * 4 GiB / 2^32.
   */
  writeVarint(value: number): void {
    if (!Number.isInteger(value) || value < 0 || value > 0xffffffff) {
      throw new RangeError(
        `writeVarint: value ${value} out of u32 range — use writeVarint64 for >2^32`
      );
    }
    this.ensureCapacity(5); // Max 5 bytes for 32-bit
    while (value > 0x7f) {
      this.buffer[this.pos++] = (value & 0x7f) | 0x80;
      value >>>= 7;
    }
    this.buffer[this.pos++] = value;
  }

  /**
   * Writes an unsigned 64-bit varint (LEB128).
   *
   * Fast path: when the value fits in a 32-bit unsigned integer, run the
   * varint loop on a plain Number. BigInt arithmetic in V8 is one to
   * two orders of magnitude slower than Number arithmetic; the modal
   * blockchain-consensus value (heights, timestamps, balances, sizes)
   * is well under 2^32, so this path covers nearly every call.
   *
   * Slow path: values >= 2^32 fall back to true BigInt arithmetic. The
   * encoding is byte-identical to the slow path.
   */
  writeVarint64(value: bigint): void {
    this.ensureCapacity(10); // Max 10 bytes for 64-bit
    if (value <= 0xffffffffn && value >= 0n) {
      let v = Number(value);
      while (v > 0x7f) {
        this.buffer[this.pos++] = (v & 0x7f) | 0x80;
        v >>>= 7;
      }
      this.buffer[this.pos++] = v;
      return;
    }
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
   * Writes a 32-bit float (IEEE 754, little-endian).
   *
   * NaN bit patterns are canonicalized to 0x7FC00000 (quiet NaN, no payload)
   * and -0.0 is normalized to +0.0 so that two values that compare equal
   * always produce identical wire bytes.
   */
  writeFloat32(value: number): void {
    this.ensureCapacity(4);
    this.view.setFloat32(this.pos, value, true);
    let bits = this.view.getUint32(this.pos, true);
    // Exponent all 1s + non-zero significand → NaN. Replace with canonical.
    if ((bits & 0x7f800000) === 0x7f800000 && (bits & 0x007fffff) !== 0) {
      bits = 0x7fc00000;
    } else if (bits === 0x80000000) {
      bits = 0; // -0 → +0
    }
    this.view.setUint32(this.pos, bits, true);
    this.pos += 4;
  }

  /**
   * Writes a 64-bit float (IEEE 754, little-endian).
   *
   * NaN bit patterns are canonicalized to 0x7FF8000000000000 (quiet NaN, no
   * payload) and -0.0 is normalized to +0.0.
   */
  writeFloat64(value: number): void {
    this.ensureCapacity(8);
    this.view.setFloat64(this.pos, value, true);
    // Read low and high halves separately (no native u64 in DataView).
    const lo = this.view.getUint32(this.pos, true);
    const hi = this.view.getUint32(this.pos + 4, true);
    // Exponent all 1s lives in hi (bits 52-62 of the 64-bit value);
    // significand is non-zero if any bit of (hi & 0xFFFFF) | lo is set.
    if ((hi & 0x7ff00000) === 0x7ff00000 && ((hi & 0x000fffff) !== 0 || lo !== 0)) {
      // Canonical quiet NaN: 0x7FF8000000000000.
      this.view.setUint32(this.pos, 0, true);
      this.view.setUint32(this.pos + 4, 0x7ff80000, true);
    } else if (hi === 0x80000000 && lo === 0) {
      // -0 → +0
      this.view.setUint32(this.pos, 0, true);
      this.view.setUint32(this.pos + 4, 0, true);
    }
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
   *
   * The length prefix uses `writeVarint64` (BigInt) so byte lengths
   * above 2^32 — possible for very large strings on V8 — encode
   * losslessly. For lengths < 2^7 the BigInt path produces the same
   * single byte as the u32 path, so this is byte-identical to Go for
   * all reasonable inputs.
   */
  writeString(value: string): void {
    // Single-pass encode: reserve worst-case (3 bytes per UTF-16 unit
    // + 5-byte length placeholder), encode straight into the buffer
    // via TextEncoder.encodeInto, then back-patch the actual length.
    // Avoids the (encoded Uint8Array) + (set into buffer) double-copy
    // and the BigInt(length) overhead of the previous implementation.
    const maxBytes = value.length * 3;
    this.ensureCapacity(maxBytes + 5);
    const written = textEncoder.encodeInto(value, this.buffer.subarray(this.pos + 5)).written ?? 0;
    const lenSize = uvarintSize(written);
    if (lenSize === 5) {
      // Length already at offset, body already at offset + 5.
      this.encodeVarintAt(this.pos, written);
      this.pos += 5 + written;
      return;
    }
    // Shift the body left to close the gap between the length prefix
    // and the encoded bytes.
    this.buffer.copyWithin(this.pos + lenSize, this.pos + 5, this.pos + 5 + written);
    this.encodeVarintAt(this.pos, written);
    this.pos += lenSize + written;
  }

  /**
   * Writes length-prefixed bytes. See `writeString` for the
   * fast-path rationale; same Number-arithmetic optimization for the
   * length prefix.
   */
  writeLengthPrefixedBytes(data: Uint8Array): void {
    const len = data.length;
    // Length is a JS Number (Uint8Array.length is a safe integer); use
    // the 32-bit varint path even when the value technically fits in
    // 64 bits — Go and Rust accept any canonical varint up to 10
    // bytes, but writeVarint's RangeError throws above 2^32 so we
    // fall back to writeVarint64 for that pathological case.
    if (len <= 0xffffffff) {
      this.writeVarint(len);
    } else {
      this.writeVarint64(BigInt(len));
    }
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
   * Type references are encoded as Bytes with type ID prefix.
   * Uses beginMessage/endMessage to avoid the sub-Writer allocation.
   */
  writeTypeRefField(fieldNumber: number, typeId: TypeID, data: Uint8Array): void {
    this.writeTag(fieldNumber, WireType.Bytes);
    const cp = this.beginMessage();
    this.writeVarint(typeId);
    this.writeLengthPrefixedBytes(data);
    this.endMessage(cp);
  }
}
