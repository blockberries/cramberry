import { describe, it, expect } from 'vitest';
import {
  zigzagEncode,
  zigzagDecode,
  zigzagEncode64,
  zigzagDecode64,
  MinInt64,
  MaxInt64,
  encodeTag,
  decodeTag,
  WireType,
} from './types';

describe('zigzag encoding (32-bit)', () => {
  it('encodes 0 to 0', () => {
    expect(zigzagEncode(0)).toBe(0);
  });

  it('encodes -1 to 1', () => {
    expect(zigzagEncode(-1)).toBe(1);
  });

  it('encodes 1 to 2', () => {
    expect(zigzagEncode(1)).toBe(2);
  });

  it('encodes -2 to 3', () => {
    expect(zigzagEncode(-2)).toBe(3);
  });

  it('roundtrips positive values', () => {
    for (const n of [0, 1, 127, 128, 255, 256, 65535, 2147483647]) {
      expect(zigzagDecode(zigzagEncode(n))).toBe(n);
    }
  });

  it('roundtrips negative values', () => {
    for (const n of [-1, -127, -128, -255, -256, -65535, -2147483648]) {
      expect(zigzagDecode(zigzagEncode(n))).toBe(n);
    }
  });
});

describe('zigzag encoding (64-bit)', () => {
  it('encodes 0n to 0n', () => {
    expect(zigzagEncode64(0n)).toBe(0n);
  });

  it('encodes -1n to 1n', () => {
    expect(zigzagEncode64(-1n)).toBe(1n);
  });

  it('encodes 1n to 2n', () => {
    expect(zigzagEncode64(1n)).toBe(2n);
  });

  it('encodes -2n to 3n', () => {
    expect(zigzagEncode64(-2n)).toBe(3n);
  });

  it('roundtrips positive values', () => {
    for (const n of [0n, 1n, 127n, 128n, 255n, 256n, 65535n, 2147483647n, MaxInt64]) {
      expect(zigzagDecode64(zigzagEncode64(n))).toBe(n);
    }
  });

  it('roundtrips negative values', () => {
    for (const n of [-1n, -127n, -128n, -255n, -256n, -65535n, -2147483648n, MinInt64]) {
      expect(zigzagDecode64(zigzagEncode64(n))).toBe(n);
    }
  });

  it('handles boundary values', () => {
    // Maximum positive 64-bit signed integer
    expect(zigzagEncode64(MaxInt64)).toBeDefined();
    expect(zigzagDecode64(zigzagEncode64(MaxInt64))).toBe(MaxInt64);

    // Minimum negative 64-bit signed integer
    expect(zigzagEncode64(MinInt64)).toBeDefined();
    expect(zigzagDecode64(zigzagEncode64(MinInt64))).toBe(MinInt64);
  });

  describe('bounds validation', () => {
    it('throws RangeError for values larger than MaxInt64', () => {
      const tooBig = MaxInt64 + 1n;
      expect(() => zigzagEncode64(tooBig)).toThrow(RangeError);
      expect(() => zigzagEncode64(tooBig)).toThrow(/outside valid 64-bit signed integer range/);
    });

    it('throws RangeError for values smaller than MinInt64', () => {
      const tooSmall = MinInt64 - 1n;
      expect(() => zigzagEncode64(tooSmall)).toThrow(RangeError);
      expect(() => zigzagEncode64(tooSmall)).toThrow(/outside valid 64-bit signed integer range/);
    });

    it('throws RangeError for very large positive values', () => {
      const veryBig = BigInt("0x10000000000000000"); // 2^64
      expect(() => zigzagEncode64(veryBig)).toThrow(RangeError);
    });

    it('throws RangeError for very large negative values', () => {
      const verySmall = -(BigInt("0x10000000000000000")); // -2^64
      expect(() => zigzagEncode64(verySmall)).toThrow(RangeError);
    });
  });
});

describe('tag encoding', () => {
  it('encodes field 1 with Varint wire type', () => {
    expect(encodeTag(1, WireType.Varint)).toBe(8);
  });

  it('encodes field 2 with Bytes wire type', () => {
    expect(encodeTag(2, WireType.Bytes)).toBe(18);
  });

  it('roundtrips tag values', () => {
    for (const fieldNumber of [1, 2, 15, 16, 100, 536870911]) {
      for (const wireType of [
        WireType.Varint,
        WireType.Fixed64,
        WireType.Bytes,
        WireType.Fixed32,
        WireType.SVarint,
        WireType.TypeRef,
      ]) {
        const tag = encodeTag(fieldNumber, wireType);
        const decoded = decodeTag(tag);
        expect(decoded.fieldNumber).toBe(fieldNumber);
        expect(decoded.wireType).toBe(wireType);
      }
    }
  });
});
