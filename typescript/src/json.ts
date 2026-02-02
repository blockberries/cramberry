/**
 * JSON serialization utilities for deterministic JSON encoding.
 * Provides helpers for encoding Cramberry messages to/from JSON with
 * blockchain-grade determinism.
 */

/**
 * Formats a bigint as a string for JSON encoding.
 * All integers are encoded as strings to prevent JavaScript precision loss.
 */
export function formatBigIntToString(value: bigint): string {
  return value.toString();
}

/**
 * Formats a number (int32, uint32) as a string for JSON encoding.
 */
export function formatNumberToString(value: number): string {
  return Math.floor(value).toString();
}

/**
 * Formats a float32 with 9 significant digits for deterministic JSON.
 * Throws error if the value is NaN or Infinity.
 */
export function formatFloat32(value: number): string {
  if (isNaN(value)) {
    throw new Error('cannot encode NaN to JSON');
  }
  if (!isFinite(value)) {
    throw new Error('cannot encode Infinity to JSON');
  }
  // Normalize -0 to 0
  if (value === 0) {
    return '0';
  }
  // Use toPrecision for fixed significant digits
  return parseFloat(value.toPrecision(9)).toString();
}

/**
 * Formats a float64 with 17 significant digits for deterministic JSON.
 * Throws error if the value is NaN or Infinity.
 */
export function formatFloat64(value: number): string {
  if (isNaN(value)) {
    throw new Error('cannot encode NaN to JSON');
  }
  if (!isFinite(value)) {
    throw new Error('cannot encode Infinity to JSON');
  }
  // Normalize -0 to 0
  if (value === 0) {
    return '0';
  }
  // Use toPrecision for fixed significant digits
  return parseFloat(value.toPrecision(17)).toString();
}

/**
 * Validates that a float is encodable to JSON.
 */
export function validateFloat(value: number): void {
  if (isNaN(value)) {
    throw new Error('cannot encode NaN to JSON');
  }
  if (!isFinite(value)) {
    throw new Error('cannot encode Infinity to JSON');
  }
}

/**
 * Encodes a Uint8Array to base64 string (RFC 4648 standard encoding).
 */
export function encodeBase64(data: Uint8Array): string {
  // Node.js
  if (typeof Buffer !== 'undefined') {
    return Buffer.from(data).toString('base64');
  }
  // Browser
  const binary = Array.from(data, byte => String.fromCharCode(byte)).join('');
  return btoa(binary);
}

/**
 * Decodes a base64 string to Uint8Array.
 */
export function decodeBase64(str: string): Uint8Array {
  // Node.js
  if (typeof Buffer !== 'undefined') {
    return new Uint8Array(Buffer.from(str, 'base64'));
  }
  // Browser
  const binary = atob(str);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

/**
 * Parses a bigint from a JSON string or number.
 */
export function parseBigIntFromJSON(value: string | number): bigint {
  if (typeof value === 'string') {
    return BigInt(value);
  }
  // Accept numeric values but lose precision for large numbers
  return BigInt(Math.floor(value));
}

/**
 * Parses a number from a JSON string or number.
 */
export function parseNumberFromJSON(value: string | number): number {
  if (typeof value === 'string') {
    return parseInt(value, 10);
  }
  return Math.floor(value);
}

/**
 * Sorts map keys lexicographically by UTF-8 byte order.
 * This ensures deterministic JSON output for maps.
 */
export function sortMapKeysLexicographic(keys: string[]): string[] {
  return [...keys].sort();
}

/**
 * Escapes a string for safe inclusion in JSON.
 */
export function escapeJSONString(str: string): string {
  return JSON.stringify(str);
}

/**
 * JSONWriter helps build JSON strings efficiently.
 */
export class JSONWriter {
  private parts: string[] = [];

  writeString(s: string): void {
    this.parts.push(s);
  }

  toString(): string {
    return this.parts.join('');
  }

  reset(): void {
    this.parts = [];
  }
}

/**
 * JSONReader helps parse JSON with validation.
 */
export class JSONReader {
  private data: any;

  constructor(json: string) {
    this.data = JSON.parse(json);
  }

  readObject(): Record<string, any> {
    if (typeof this.data !== 'object' || this.data === null || Array.isArray(this.data)) {
      throw new Error('expected JSON object');
    }
    return this.data;
  }

  getData(): any {
    return this.data;
  }
}
