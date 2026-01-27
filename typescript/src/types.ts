/**
 * Wire types used in the Cramberry V2 encoding format.
 *
 * V2 uses a simplified set of wire types (0-4) compared to protobuf.
 * Type references are encoded as Bytes with a type ID prefix.
 */
export enum WireType {
  /** Variable-length unsigned integer (LEB128) */
  Varint = 0,
  /** Fixed 64-bit value (little-endian) */
  Fixed64 = 1,
  /** Length-prefixed bytes (string, bytes, messages, packed arrays) */
  Bytes = 2,
  /** Fixed 32-bit value (little-endian) */
  Fixed32 = 3,
  /** ZigZag-encoded signed integer */
  SVarint = 4,
}

/**
 * Type ID for polymorphic type registration.
 */
export type TypeID = number;

/**
 * Field tag combining field number and wire type.
 */
export interface FieldTag {
  fieldNumber: number;
  wireType: WireType;
}

/**
 * Maximum values for varint encoding.
 */
export const MaxVarint32 = 0xffffffff;
export const MaxVarint64 = BigInt("0xffffffffffffffff");

/**
 * Signed 64-bit integer bounds.
 */
export const MinInt64 = BigInt("-9223372036854775808"); // -2^63
export const MaxInt64 = BigInt("9223372036854775807"); // 2^63 - 1

/**
 * V2 Compact Tag Format Constants
 *
 * Tag encoding:
 *   Fields 1-15:  [fieldNum:4][wireType:3][0:1] = single byte
 *   Fields 16+:   [0:4][wireType:3][1:1] followed by varint fieldNum
 *   End marker:   0x00 (fieldNum=0, wireType=0, extended=0)
 */
export const END_MARKER = 0x00;
export const TAG_EXTENDED_BIT = 0x01;
export const TAG_WIRE_TYPE_MASK = 0x0e;
export const TAG_WIRE_TYPE_SHIFT = 1;
export const TAG_FIELD_NUM_SHIFT = 4;
export const MAX_COMPACT_FIELD_NUM = 15;

/**
 * Encode a V2 compact tag from field number and wire type.
 * Returns the encoded bytes.
 */
export function encodeCompactTag(fieldNumber: number, wireType: WireType): Uint8Array {
  if (fieldNumber <= 0) {
    return new Uint8Array(0); // Invalid field number
  }

  if (fieldNumber <= MAX_COMPACT_FIELD_NUM) {
    // Compact format: single byte
    const tag = (fieldNumber << TAG_FIELD_NUM_SHIFT) | (wireType << TAG_WIRE_TYPE_SHIFT);
    return new Uint8Array([tag]);
  }

  // Extended format: marker byte + varint field number
  const marker = (wireType << TAG_WIRE_TYPE_SHIFT) | TAG_EXTENDED_BIT;
  const result: number[] = [marker];

  // Append varint-encoded field number
  let num = fieldNumber;
  while (num >= 0x80) {
    result.push((num & 0x7f) | 0x80);
    num >>>= 7;
  }
  result.push(num);

  return new Uint8Array(result);
}

/**
 * Result of decoding a V2 compact tag.
 */
export interface CompactTagResult {
  fieldNumber: number;
  wireType: WireType;
  bytesRead: number;
}

/**
 * Decode a V2 compact tag from a buffer.
 * Returns fieldNumber (0 for end marker), wireType, and bytes consumed.
 */
export function decodeCompactTag(data: Uint8Array, offset: number = 0): CompactTagResult {
  if (offset >= data.length) {
    return { fieldNumber: 0, wireType: 0, bytesRead: 0 };
  }

  const tag = data[offset];

  // Check for end marker
  if (tag === END_MARKER) {
    return { fieldNumber: 0, wireType: 0, bytesRead: 1 };
  }

  const wireType = ((tag & TAG_WIRE_TYPE_MASK) >> TAG_WIRE_TYPE_SHIFT) as WireType;

  if ((tag & TAG_EXTENDED_BIT) === 0) {
    // Compact format: field number in upper 4 bits
    const fieldNumber = tag >> TAG_FIELD_NUM_SHIFT;
    return { fieldNumber, wireType, bytesRead: 1 };
  }

  // Extended format: read varint field number
  let fieldNumber = 0;
  let shift = 0;
  let pos = offset + 1;

  for (let i = 0; i < 10 && pos < data.length; i++) {
    const b = data[pos++];
    fieldNumber |= (b & 0x7f) << shift;
    if ((b & 0x80) === 0) {
      return { fieldNumber, wireType, bytesRead: pos - offset };
    }
    shift += 7;
  }

  // Invalid varint
  return { fieldNumber: 0, wireType: 0, bytesRead: 0 };
}

/**
 * @deprecated Use encodeCompactTag for V2 format
 * Legacy protobuf-style tag encoding (kept for reference).
 */
export function encodeTag(fieldNumber: number, wireType: WireType): number {
  return (fieldNumber << 3) | wireType;
}

/**
 * @deprecated Use decodeCompactTag for V2 format
 * Legacy protobuf-style tag decoding (kept for reference).
 */
export function decodeTag(tag: number): FieldTag {
  return {
    fieldNumber: tag >>> 3,
    wireType: tag & 0x07,
  };
}

/**
 * Encode a signed integer using ZigZag encoding.
 */
export function zigzagEncode(n: number): number {
  return (n << 1) ^ (n >> 31);
}

/**
 * Encode a signed bigint using ZigZag encoding.
 * @throws RangeError if n is outside the valid 64-bit signed integer range
 */
export function zigzagEncode64(n: bigint): bigint {
  if (n < MinInt64 || n > MaxInt64) {
    throw new RangeError(
      `BigInt value ${n} is outside valid 64-bit signed integer range [${MinInt64}, ${MaxInt64}]`
    );
  }
  return (n << 1n) ^ (n >> 63n);
}

/**
 * Decode a ZigZag encoded integer.
 */
export function zigzagDecode(n: number): number {
  return (n >>> 1) ^ -(n & 1);
}

/**
 * Decode a ZigZag encoded bigint.
 */
export function zigzagDecode64(n: bigint): bigint {
  return (n >> 1n) ^ -(n & 1n);
}
