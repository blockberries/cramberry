import { describe, it, expect } from 'vitest';
import {
  formatBigIntToString,
  formatNumberToString,
  formatFloat32,
  formatFloat64,
  validateFloat,
  encodeBase64,
  decodeBase64,
  parseBigIntFromJSON,
  parseNumberFromJSON,
  sortMapKeysLexicographic,
  escapeJSONString,
  JSONWriter,
  JSONReader,
} from './json';

describe('formatBigIntToString', () => {
  it('formats bigint to string', () => {
    expect(formatBigIntToString(0n)).toBe('0');
    expect(formatBigIntToString(123n)).toBe('123');
    expect(formatBigIntToString(-456n)).toBe('-456');
    expect(formatBigIntToString(9223372036854775807n)).toBe('9223372036854775807');
    expect(formatBigIntToString(18446744073709551615n)).toBe('18446744073709551615');
  });
});

describe('formatNumberToString', () => {
  it('formats number to string', () => {
    expect(formatNumberToString(0)).toBe('0');
    expect(formatNumberToString(123)).toBe('123');
    expect(formatNumberToString(-456)).toBe('-456');
    expect(formatNumberToString(123.7)).toBe('123'); // Floors decimals
  });
});

describe('formatFloat32', () => {
  // Outputs must match Go's strconv.FormatFloat(v, 'g', 9, 32).
  // Reference values come from pkg/cramberry/json_test.go::TestFormatFloat32.
  it('formats zero', () => {
    expect(formatFloat32(0)).toBe('0');
  });

  it('formats finite values with f32 quantization', () => {
    expect(formatFloat32(3.14159)).toBe('3.14159012');
    expect(formatFloat32(-2.71828)).toBe('-2.71828008');
  });

  it('uses scientific notation for large values', () => {
    expect(formatFloat32(1.23e10)).toBe('1.23000003e+10');
  });

  it('uses scientific notation for small values', () => {
    expect(formatFloat32(0.0000001)).toBe('1.00000001e-07');
  });

  it('rejects NaN', () => {
    expect(() => formatFloat32(NaN)).toThrow('cannot encode NaN');
  });

  it('rejects Infinity', () => {
    expect(() => formatFloat32(Infinity)).toThrow('cannot encode Infinity');
    expect(() => formatFloat32(-Infinity)).toThrow('cannot encode Infinity');
  });

  it('normalizes -0 to 0', () => {
    expect(formatFloat32(-0)).toBe('0');
  });
});

describe('formatFloat64', () => {
  // Outputs must match Go's strconv.FormatFloat(v, 'g', 17, 64).
  // Reference values come from pkg/cramberry/json_test.go::TestFormatFloat64.
  it('formats zero', () => {
    expect(formatFloat64(0)).toBe('0');
  });

  it('formats finite values with f64 precision', () => {
    expect(formatFloat64(2.718281828459045)).toBe('2.7182818284590451');
    expect(formatFloat64(-3.141592653589793)).toBe('-3.1415926535897931');
  });

  it('uses scientific notation for very large values', () => {
    expect(formatFloat64(1.23e100)).toBe('1.2300000000000001e+100');
  });

  it('uses scientific notation for very small values', () => {
    expect(formatFloat64(1e-100)).toBe('1e-100');
  });

  it('uses scientific for values with magnitude >= 10^17', () => {
    expect(formatFloat64(1e20)).toBe('1e+20');
  });

  it('uses scientific for values with magnitude < 1e-4', () => {
    // 1e-7 is not exactly representable in f64; Go's 17-digit form exposes the
    // true bit pattern (slightly less than 1e-7).
    expect(formatFloat64(1e-7)).toBe('9.9999999999999995e-08');
    expect(formatFloat64(1e-5)).toBe('1.0000000000000001e-05');
  });

  it('uses decimal for values in [1e-4, 10^17)', () => {
    expect(formatFloat64(1.5)).toBe('1.5');
    expect(formatFloat64(0.0001)).toBe('0.0001');
  });

  it('rejects NaN', () => {
    expect(() => formatFloat64(NaN)).toThrow('cannot encode NaN');
  });

  it('rejects Infinity', () => {
    expect(() => formatFloat64(Infinity)).toThrow('cannot encode Infinity');
    expect(() => formatFloat64(-Infinity)).toThrow('cannot encode Infinity');
  });

  it('normalizes -0 to 0', () => {
    expect(formatFloat64(-0)).toBe('0');
  });
});

describe('validateFloat', () => {
  it('accepts valid floats', () => {
    expect(() => validateFloat(0)).not.toThrow();
    expect(() => validateFloat(3.14)).not.toThrow();
    expect(() => validateFloat(-2.71)).not.toThrow();
  });

  it('rejects NaN', () => {
    expect(() => validateFloat(NaN)).toThrow();
  });

  it('rejects Infinity', () => {
    expect(() => validateFloat(Infinity)).toThrow();
    expect(() => validateFloat(-Infinity)).toThrow();
  });
});

describe('encodeBase64 and decodeBase64', () => {
  it('encodes and decodes bytes', () => {
    const data = new Uint8Array([0xde, 0xad, 0xbe, 0xef]);
    const encoded = encodeBase64(data);
    expect(encoded).toBe('3q2+7w==');

    const decoded = decodeBase64(encoded);
    expect(decoded).toEqual(data);
  });

  it('handles empty data', () => {
    const data = new Uint8Array([]);
    const encoded = encodeBase64(data);
    expect(encoded).toBe('');

    const decoded = decodeBase64(encoded);
    expect(decoded).toEqual(data);
  });

  it('handles string data', () => {
    const text = 'hello, world!';
    const data = new TextEncoder().encode(text);
    const encoded = encodeBase64(data);
    const decoded = decodeBase64(encoded);
    const result = new TextDecoder().decode(decoded);
    expect(result).toBe(text);
  });
});

describe('parseBigIntFromJSON', () => {
  it('parses string to bigint', () => {
    expect(parseBigIntFromJSON('123')).toBe(123n);
    expect(parseBigIntFromJSON('-456')).toBe(-456n);
    expect(parseBigIntFromJSON('9223372036854775807')).toBe(9223372036854775807n);
  });

  it('accepts numeric values', () => {
    expect(parseBigIntFromJSON(123)).toBe(123n);
    expect(parseBigIntFromJSON(-456)).toBe(-456n);
  });

  it('floors numeric values', () => {
    expect(parseBigIntFromJSON(123.7)).toBe(123n);
  });
});

describe('parseNumberFromJSON', () => {
  it('parses string to number', () => {
    expect(parseNumberFromJSON('123')).toBe(123);
    expect(parseNumberFromJSON('-456')).toBe(-456);
  });

  it('accepts numeric values', () => {
    expect(parseNumberFromJSON(123)).toBe(123);
    expect(parseNumberFromJSON(-456)).toBe(-456);
  });

  it('rejects non-integer numeric values', () => {
    // Previously truncated 123.7 → 123 silently. Now rejects to match
    // Go's strconv.ParseInt, which would fail on equivalent input.
    expect(() => parseNumberFromJSON(123.7)).toThrow();
  });

  it('rejects malformed strings', () => {
    expect(() => parseNumberFromJSON('123.5')).toThrow();
    expect(() => parseNumberFromJSON('12abc')).toThrow();
    expect(() => parseNumberFromJSON('')).toThrow();
    expect(() => parseNumberFromJSON(' 12')).toThrow();
  });
});

describe('sortMapKeysLexicographic', () => {
  it('sorts keys alphabetically', () => {
    const keys = ['zebra', 'apple', 'banana'];
    const sorted = sortMapKeysLexicographic(keys);
    expect(sorted).toEqual(['apple', 'banana', 'zebra']);
  });

  it('sorts numeric strings lexicographically', () => {
    const keys = ['10', '2', '1'];
    const sorted = sortMapKeysLexicographic(keys);
    expect(sorted).toEqual(['1', '10', '2']); // Lexicographic, not numeric
  });

  it('handles empty array', () => {
    const keys: string[] = [];
    const sorted = sortMapKeysLexicographic(keys);
    expect(sorted).toEqual([]);
  });

  it('handles unicode (BMP)', () => {
    const keys = ['世界', 'hello', 'мир'];
    const sorted = sortMapKeysLexicographic(keys);
    expect(sorted).toEqual(['hello', 'мир', '世界']);
  });

  it('sorts non-BMP keys by UTF-8 bytes (matches Go sort.Strings)', () => {
    //  (BMP private-use) encodes to UTF-8 bytes 0xEE 0x80 0x80.
    // 😀 (U+1F600) encodes to UTF-8 bytes 0xF0 0x9F 0x98 0x80.
    // In UTF-8 byte order:  < 😀.
    // In default JS UTF-16 code-unit order: 😀ʼs surrogate pair starts at
    // 0xD83D, which sorts BEFORE 0xE000 — the WRONG order. This test
    // catches a regression to that bug.
    const keys = ['', '😀'];
    const sorted = sortMapKeysLexicographic(keys);
    expect(sorted).toEqual(['', '😀']);
  });
});

describe('escapeJSONString', () => {
  it('escapes simple strings', () => {
    expect(escapeJSONString('hello')).toBe('"hello"');
  });

  it('escapes quotes', () => {
    expect(escapeJSONString('say "hello"')).toBe('"say \\"hello\\""');
  });

  it('escapes special characters', () => {
    expect(escapeJSONString('line1\nline2')).toBe('"line1\\nline2"');
    expect(escapeJSONString('col1\tcol2')).toBe('"col1\\tcol2"');
  });

  it('handles empty string', () => {
    expect(escapeJSONString('')).toBe('""');
  });

  it('handles unicode', () => {
    expect(escapeJSONString('hello, 世界')).toBe('"hello, 世界"');
  });
});

describe('JSONWriter', () => {
  it('builds JSON strings', () => {
    const writer = new JSONWriter();
    writer.writeString('{');
    writer.writeString('"key"');
    writer.writeString(':');
    writer.writeString('"value"');
    writer.writeString('}');
    expect(writer.toString()).toBe('{"key":"value"}');
  });

  it('can be reset', () => {
    const writer = new JSONWriter();
    writer.writeString('test');
    expect(writer.toString()).toBe('test');
    writer.reset();
    expect(writer.toString()).toBe('');
  });
});

describe('JSONReader', () => {
  it('parses JSON object', () => {
    const reader = new JSONReader('{"key":"value"}');
    const obj = reader.readObject();
    expect(obj).toEqual({ key: 'value' });
  });

  it('rejects non-object JSON', () => {
    expect(() => new JSONReader('[]').readObject()).toThrow('expected JSON object');
    expect(() => new JSONReader('null').readObject()).toThrow('expected JSON object');
    expect(() => new JSONReader('"string"').readObject()).toThrow('expected JSON object');
  });

  it('throws on invalid JSON', () => {
    expect(() => new JSONReader('{invalid')).toThrow();
  });
});
