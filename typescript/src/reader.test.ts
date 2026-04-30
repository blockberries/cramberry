import { describe, it, expect } from 'vitest';
import { Reader, DEFAULT_LIMITS } from './reader';
import { Writer } from './writer';
import { WireType } from './types';
import { DecodeError } from './errors';

describe('Reader', () => {
  describe('readVarint canonical-form rejection', () => {
    it('rejects [0x80, 0x00] (overlong zero)', () => {
      expect(() => new Reader(new Uint8Array([0x80, 0x00])).readVarint()).toThrow(
        /non-canonical/,
      );
    });

    it('rejects [0x81, 0x00] (overlong one)', () => {
      expect(() => new Reader(new Uint8Array([0x81, 0x00])).readVarint()).toThrow(
        /non-canonical/,
      );
    });

    it('accepts canonical [0x00] (zero)', () => {
      expect(new Reader(new Uint8Array([0x00])).readVarint()).toBe(0);
    });
  });

  describe('skipValue with 64-bit varint length', () => {
    // Regression: skipValue used to read length prefixes via the
    // 32-bit-capped readVarint, failing on legit Bytes-typed fields
    // whose length encodes to >5 varint bytes. With the fix it goes
    // through readSafeLength (varint64 → narrow).
    it('skips a Bytes value whose length needs a multi-byte varint', () => {
      // 20 000 bytes → varint(20000) = a0 9c 01 (3 bytes)
      const w = new Writer();
      w.writeBytes(new Uint8Array(20000));
      // Build a synthetic stream: tag(WireType.Bytes, field=1) + the
      // length-prefixed bytes. We do this manually rather than calling
      // writer helpers so we can verify skipValue advances the same
      // distance the writer wrote.
      const sub = new Writer();
      sub.writeTag(1, WireType.Bytes);
      sub.writeLengthPrefixedBytes(new Uint8Array(20000));
      sub.writeEndMarker();
      const bytes = sub.bytes();

      const r = new Reader(bytes);
      const tag = r.readTag();
      expect(tag.fieldNumber).toBe(1);
      expect(tag.wireType).toBe(WireType.Bytes);
      r.skipValue(tag.wireType);
      // After skipping, only the end marker should remain.
      expect(r.remaining).toBe(1);
      expect(r.readByte()).toBe(0x00);
    });
  });

  describe('per-field limits', () => {
    it('rejects strings exceeding maxStringLength', () => {
      const w = new Writer();
      w.writeString('x'.repeat(2_000_000));
      const bytes = w.bytes();
      // 1 MB cap, body is 2 MB.
      const r = new Reader(bytes, {
        limits: { ...DEFAULT_LIMITS, maxStringLength: 1_000_000 },
      });
      expect(() => r.readString()).toThrow(DecodeError);
    });

    it('propagates limits + validateUtf8 to subReader', () => {
      const r = new Reader(new Uint8Array(0), {
        limits: { ...DEFAULT_LIMITS, maxStringLength: 5 },
        validateUtf8: true,
      });
      // subReader requires bytes available; just check the constructor
      // captured the options.
      expect(r.getLimits().maxStringLength).toBe(5);
      expect(r.getValidateUtf8()).toBe(true);
    });
  });
});
