/**
 * Wire types used in the Cramberry encoding format.
 *
 * The wire type lives in 3 bits of the field tag and identifies how the
 * following value is encoded.
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
 * Tag format constants.
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
 * Maximum allowed field number (2^29 - 1, ~536 million).
 * Matches the Go runtime's wire.MaxFieldNumber. Anything larger is rejected
 * by both encoders and decoders.
 */
export const MAX_FIELD_NUMBER = (1n << 29n) - 1n;

/**
 * Encode a field tag from field number and wire type.
 * Returns the encoded bytes. Returns an empty array for invalid field
 * numbers (0, negative, or above MAX_FIELD_NUMBER) so callers can detect
 * the failure without throwing.
 */
export function encodeTag(fieldNumber: number, wireType: WireType): Uint8Array {
  if (fieldNumber <= 0 || BigInt(fieldNumber) > MAX_FIELD_NUMBER) {
    return new Uint8Array(0);
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
 * Result of decoding a field tag.
 */
export interface TagResult {
  fieldNumber: number;
  wireType: WireType;
  bytesRead: number;
}

/**
 * Decode a field tag from a buffer.
 * Returns fieldNumber (0 for end marker), wireType, and bytes consumed.
 */
export function decodeTag(data: Uint8Array, offset: number = 0): TagResult {
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

  // Extended format: read varint field number.
  // BigInt accumulator prevents 32-bit truncation that would silently corrupt
  // field numbers above 2^28 (JS bitwise `<<` truncates to int32).
  let fieldNumber = 0n;
  let shift = 0n;
  let pos = offset + 1;

  for (let i = 0; i < 5 && pos < data.length; i++) {
    const b = data[pos++];
    fieldNumber |= BigInt(b & 0x7f) << shift;
    if ((b & 0x80) === 0) {
      if (fieldNumber > MAX_FIELD_NUMBER) {
        return { fieldNumber: 0, wireType: 0, bytesRead: 0 };
      }
      return { fieldNumber: Number(fieldNumber), wireType, bytesRead: pos - offset };
    }
    shift += 7n;
  }

  // Invalid varint
  return { fieldNumber: 0, wireType: 0, bytesRead: 0 };
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
