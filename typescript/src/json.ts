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
 * Matches the output of Go's strconv.FormatFloat(v, 'g', 9, 32).
 * Throws error if the value is NaN or Infinity.
 */
export function formatFloat32(value: number): string {
  if (isNaN(value)) {
    throw new Error('cannot encode NaN to JSON');
  }
  if (!isFinite(value)) {
    throw new Error('cannot encode Infinity to JSON');
  }
  if (value === 0) {
    return '0';
  }
  // Narrow to f32 precision so the formatted bits match what Go would
  // produce when given a float32. Math.fround returns the f64 closest to
  // the f32 quantization of value.
  return formatGoG(Math.fround(value), 9);
}

/**
 * Formats a float64 with 17 significant digits for deterministic JSON.
 * Matches the output of Go's strconv.FormatFloat(v, 'g', 17, 64).
 * Throws error if the value is NaN or Infinity.
 */
export function formatFloat64(value: number): string {
  if (isNaN(value)) {
    throw new Error('cannot encode NaN to JSON');
  }
  if (!isFinite(value)) {
    throw new Error('cannot encode Infinity to JSON');
  }
  if (value === 0) {
    return '0';
  }
  return formatGoG(value, 17);
}

/**
 * Formats a non-zero finite number using Go's strconv 'g' verb with the
 * given precision (number of significant digits).
 *
 *   - Decimal form when -4 <= exponent < precision.
 *   - Otherwise scientific: `mantissa "e" sign digits` with the exponent
 *     always signed and at least two digits.
 *   - Trailing zeros in the fractional part are stripped.
 */
function formatGoG(value: number, precision: number): string {
  // toExponential(p) gives p digits after the decimal — i.e. p+1 significant
  // digits total — using IEEE round-half-to-even, the same rule Go uses.
  const exp = value.toExponential(precision - 1);
  const match = /^(-?\d+(?:\.\d+)?)e([+-]?\d+)$/.exec(exp);
  if (!match) {
    throw new Error('toExponential produced unexpected format: ' + exp);
  }
  let mantissa = match[1];
  const exponent = parseInt(match[2], 10);

  // Strip trailing zeros from the fractional part of the mantissa.
  if (mantissa.includes('.')) {
    mantissa = mantissa.replace(/0+$/, '');
    if (mantissa.endsWith('.')) mantissa = mantissa.slice(0, -1);
  }

  if (exponent >= -4 && exponent < precision) {
    return decimalFromMantissaExp(mantissa, exponent);
  }
  const sign = exponent >= 0 ? '+' : '-';
  const absExp = Math.abs(exponent);
  const expStr = absExp < 10 ? '0' + absExp : String(absExp);
  return mantissa + 'e' + sign + expStr;
}

/**
 * Renders mantissa * 10^exp in plain decimal form, stripping trailing zeros
 * from any fractional part.
 */
function decimalFromMantissaExp(mantissa: string, exp: number): string {
  let sign = '';
  if (mantissa.startsWith('-')) {
    sign = '-';
    mantissa = mantissa.slice(1);
  }
  const dotIdx = mantissa.indexOf('.');
  const digits =
    dotIdx < 0 ? mantissa : mantissa.slice(0, dotIdx) + mantissa.slice(dotIdx + 1);
  const integerLen = dotIdx < 0 ? mantissa.length : dotIdx;
  const newIntegerLen = integerLen + exp;

  if (newIntegerLen <= 0) {
    const padding = '0'.repeat(-newIntegerLen);
    let result = '0.' + padding + digits;
    result = result.replace(/0+$/, '');
    if (result.endsWith('.')) result = result.slice(0, -1);
    return sign + result;
  }
  if (newIntegerLen >= digits.length) {
    const padding = '0'.repeat(newIntegerLen - digits.length);
    return sign + digits + padding;
  }
  const intPart = digits.slice(0, newIntegerLen);
  const fracPart = digits.slice(newIntegerLen).replace(/0+$/, '');
  if (fracPart === '') return sign + intPart;
  return sign + intPart + '.' + fracPart;
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
 * Parses a strict integer from a JSON string or number.
 *
 * Rejects malformed inputs:
 *   - "123.5" / "12abc" / "" / " 12" → throws Error.
 *   - non-integer numbers like 12.5 → throws Error.
 * This matches Go's strconv.ParseInt which rejects all of the above.
 * The previous implementation used `parseInt(value, 10)`, which silently
 * truncated "123.5" to 123 and "12abc" to 12.
 */
export function parseNumberFromJSON(value: string | number): number {
  if (typeof value === 'string') {
    if (!/^-?\d+$/.test(value)) {
      throw new Error(`invalid integer: ${JSON.stringify(value)}`);
    }
    const n = Number(value);
    if (!Number.isFinite(n) || !Number.isSafeInteger(n)) {
      throw new Error(`integer out of safe range: ${value}`);
    }
    return n;
  }
  if (!Number.isFinite(value) || !Number.isInteger(value)) {
    throw new Error(`expected integer, got ${value}`);
  }
  return value;
}

const __sortKeyEncoder = new TextEncoder();

/**
 * Compares two strings by raw UTF-8 byte order, matching Go's `sort.Strings`.
 *
 * The default JS string comparator (`<`/`>`) uses UTF-16 code unit order,
 * which agrees with UTF-8 for BMP characters but disagrees for code points
 * `>= U+10000` (which encode as a surrogate pair `0xD800..0xDFFF` in UTF-16
 * but as bytes starting with `0xF0` in UTF-8). Using this comparator ensures
 * non-BMP keys sort the same way they do on the Go side.
 */
export function compareUtf8(a: string, b: string): number {
  const ab = __sortKeyEncoder.encode(a);
  const bb = __sortKeyEncoder.encode(b);
  const n = Math.min(ab.length, bb.length);
  for (let i = 0; i < n; i++) {
    if (ab[i] !== bb[i]) return ab[i] - bb[i];
  }
  return ab.length - bb.length;
}

/**
 * Sorts map keys lexicographically by UTF-8 byte order.
 * This ensures deterministic JSON output for maps.
 */
export function sortMapKeysLexicographic(keys: string[]): string[] {
  return [...keys].sort(compareUtf8);
}

/**
 * Returns a JSON string literal — the escaped contents *plus* surrounding
 * double quotes. Equivalent to Go's `cramberry.EscapeJSONString`. Use this
 * when you need a complete JSON string token to splice into a JSON document.
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
