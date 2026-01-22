/**
 * Wire types used in the Cramberry encoding format.
 */
export enum WireType {
  /** Variable-length unsigned integer (LEB128) */
  Varint = 0,
  /** Fixed 64-bit value (little-endian) */
  Fixed64 = 1,
  /** Length-prefixed bytes */
  Bytes = 2,
  /** Fixed 32-bit value (little-endian) */
  Fixed32 = 5,
  /** ZigZag-encoded signed integer */
  SVarint = 6,
  /** Type reference for polymorphic values */
  TypeRef = 7,
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
 * Encode a field tag from field number and wire type.
 */
export function encodeTag(fieldNumber: number, wireType: WireType): number {
  return (fieldNumber << 3) | wireType;
}

/**
 * Decode a field tag into field number and wire type.
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
 */
export function zigzagEncode64(n: bigint): bigint {
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
