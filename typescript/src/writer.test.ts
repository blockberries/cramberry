import { describe, it, expect } from 'vitest';
import { Writer } from './writer';
import { Reader } from './reader';
import { WireType } from './types';

describe('Writer', () => {
  describe('varint', () => {
    it('encodes 0', () => {
      const writer = new Writer();
      writer.writeVarint(0);
      expect(writer.bytes()).toEqual(new Uint8Array([0]));
    });

    it('encodes 1', () => {
      const writer = new Writer();
      writer.writeVarint(1);
      expect(writer.bytes()).toEqual(new Uint8Array([1]));
    });

    it('encodes 127', () => {
      const writer = new Writer();
      writer.writeVarint(127);
      expect(writer.bytes()).toEqual(new Uint8Array([127]));
    });

    it('encodes 128', () => {
      const writer = new Writer();
      writer.writeVarint(128);
      expect(writer.bytes()).toEqual(new Uint8Array([0x80, 0x01]));
    });

    it('encodes 300', () => {
      const writer = new Writer();
      writer.writeVarint(300);
      expect(writer.bytes()).toEqual(new Uint8Array([0xac, 0x02]));
    });
  });

  describe('svarint', () => {
    it('encodes 0', () => {
      const writer = new Writer();
      writer.writeSVarint(0);
      expect(writer.bytes()).toEqual(new Uint8Array([0]));
    });

    it('encodes -1', () => {
      const writer = new Writer();
      writer.writeSVarint(-1);
      expect(writer.bytes()).toEqual(new Uint8Array([1]));
    });

    it('encodes 1', () => {
      const writer = new Writer();
      writer.writeSVarint(1);
      expect(writer.bytes()).toEqual(new Uint8Array([2]));
    });
  });

  describe('string', () => {
    it('encodes "hello"', () => {
      const writer = new Writer();
      writer.writeString('hello');
      expect(writer.bytes()).toEqual(new Uint8Array([5, 104, 101, 108, 108, 111]));
    });

    it('encodes empty string', () => {
      const writer = new Writer();
      writer.writeString('');
      expect(writer.bytes()).toEqual(new Uint8Array([0]));
    });
  });

  describe('field methods', () => {
    it('writes int32 field', () => {
      const writer = new Writer();
      writer.writeInt32Field(1, -42);
      const data = writer.bytes();

      const reader = new Reader(data);
      const tag = reader.readTag();
      expect(tag.fieldNumber).toBe(1);
      expect(tag.wireType).toBe(WireType.SVarint);
      expect(reader.readInt32()).toBe(-42);
    });

    it('writes string field', () => {
      const writer = new Writer();
      writer.writeStringField(2, 'hello');
      const data = writer.bytes();

      const reader = new Reader(data);
      const tag = reader.readTag();
      expect(tag.fieldNumber).toBe(2);
      expect(tag.wireType).toBe(WireType.Bytes);
      expect(reader.readString()).toBe('hello');
    });

    it('writes multiple fields', () => {
      const writer = new Writer();
      writer.writeInt32Field(1, 42);
      writer.writeStringField(2, 'test');
      writer.writeBoolField(3, true);
      const data = writer.bytes();

      const reader = new Reader(data);

      let tag = reader.readTag();
      expect(tag.fieldNumber).toBe(1);
      expect(reader.readInt32()).toBe(42);

      tag = reader.readTag();
      expect(tag.fieldNumber).toBe(2);
      expect(reader.readString()).toBe('test');

      tag = reader.readTag();
      expect(tag.fieldNumber).toBe(3);
      expect(reader.readBool()).toBe(true);

      expect(reader.hasMore).toBe(false);
    });
  });

  describe('floats', () => {
    it('writes float32', () => {
      const writer = new Writer();
      writer.writeFloat32Field(1, 3.14);
      const data = writer.bytes();

      const reader = new Reader(data);
      const tag = reader.readTag();
      expect(tag.wireType).toBe(WireType.Fixed32);
      expect(reader.readFloat32()).toBeCloseTo(3.14, 5);
    });

    it('writes float64', () => {
      const writer = new Writer();
      writer.writeFloat64Field(1, Math.PI);
      const data = writer.bytes();

      const reader = new Reader(data);
      const tag = reader.readTag();
      expect(tag.wireType).toBe(WireType.Fixed64);
      expect(reader.readFloat64()).toBeCloseTo(Math.PI, 10);
    });
  });

  describe('int64 precision', () => {
    it('readInt64AsNumber returns number for safe values', () => {
      const writer = new Writer();
      writer.writeSVarint64(BigInt(12345));
      const data = writer.bytes();

      const reader = new Reader(data);
      const value = reader.readInt64AsNumber(false);
      expect(value).toBe(12345);
    });

    it('readInt64AsNumber handles negative values', () => {
      const writer = new Writer();
      writer.writeSVarint64(BigInt(-12345));
      const data = writer.bytes();

      const reader = new Reader(data);
      const value = reader.readInt64AsNumber(false);
      expect(value).toBe(-12345);
    });

    it('readUint64AsNumber returns number for safe values', () => {
      const writer = new Writer();
      writer.writeVarint64(BigInt(12345));
      const data = writer.bytes();

      const reader = new Reader(data);
      const value = reader.readUint64AsNumber(false);
      expect(value).toBe(12345);
    });

    it('readInt64 returns bigint with full precision', () => {
      const largeValue = BigInt(Number.MAX_SAFE_INTEGER) + 100n;
      const writer = new Writer();
      writer.writeSVarint64(largeValue);
      const data = writer.bytes();

      const reader = new Reader(data);
      const value = reader.readInt64();
      expect(value).toBe(largeValue);
    });

    it('readUint64 returns bigint with full precision', () => {
      const largeValue = BigInt(Number.MAX_SAFE_INTEGER) + 100n;
      const writer = new Writer();
      writer.writeVarint64(largeValue);
      const data = writer.bytes();

      const reader = new Reader(data);
      const value = reader.readUint64();
      expect(value).toBe(largeValue);
    });
  });

  describe('typeRef', () => {
    it('writes and reads typeRef field', () => {
      // Create some value data
      const valueWriter = new Writer();
      valueWriter.writeString('polymorphic value');
      const valueData = valueWriter.bytes();

      // Write a typeRef field (encoded as Bytes in V2)
      const writer = new Writer();
      writer.writeTypeRefField(1, 128, valueData);
      const data = writer.bytes();

      // Read it back
      const reader = new Reader(data);
      const tag = reader.readTag();
      expect(tag.fieldNumber).toBe(1);
      expect(tag.wireType).toBe(WireType.Bytes); // V2 uses Bytes for type refs

      // Read the outer length-prefixed bytes, then extract type ref
      const outerBytes = reader.readLengthPrefixedBytes();
      const innerReader = new Reader(outerBytes);
      const { typeId, reader: subReader } = innerReader.readTypeRef();
      expect(typeId).toBe(128);
      expect(subReader.readString()).toBe('polymorphic value');
    });

    it('writes typeRef without field tag', () => {
      const valueWriter = new Writer();
      valueWriter.writeVarint(42);
      const valueData = valueWriter.bytes();

      const writer = new Writer();
      writer.writeTypeRef(256, valueData);
      const data = writer.bytes();

      const reader = new Reader(data);
      const { typeId, reader: subReader } = reader.readTypeRef();
      expect(typeId).toBe(256);
      expect(subReader.readVarint()).toBe(42);
    });
  });
});
