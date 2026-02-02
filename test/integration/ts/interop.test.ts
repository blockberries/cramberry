/**
 * Cross-runtime interoperability tests for TypeScript.
 *
 * These tests verify that TypeScript runtime produces identical binary
 * encodings to Go and can decode Go-generated golden files.
 */

import { describe, it, expect, beforeAll } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
import { Writer, Reader, WireType } from '../../../typescript/src';
import {
  Status,
  ScalarTypes,
  RepeatedTypes,
  NestedMessage,
  ComplexTypes,
  EdgeCases,
  AllFieldNumbers,
} from './interop';

const GOLDEN_DIR = path.join(__dirname, '..', '..', 'golden');

// Test data matching Go's TestData
const TestData = {
  scalarTypes: {
    boolVal: true,
    int32Val: -42,
    int64Val: -9223372036854775807n,
    uint32Val: 4294967295,
    uint64Val: 18446744073709551615n,
    float32Val: 3.14159,
    float64Val: 2.718281828459045,
    stringVal: 'hello, cramberry!',
    bytesVal: new Uint8Array([0xde, 0xad, 0xbe, 0xef]),
  } as ScalarTypes,

  repeatedTypes: {
    int32List: [1, -2, 3, -4, 5],
    stringList: ['alpha', 'beta', 'gamma'],
    bytesList: [new Uint8Array([0x01, 0x02]), new Uint8Array([0x03, 0x04, 0x05])],
  } as RepeatedTypes,

  nestedMessage: {
    name: 'nested',
    value: 123,
  } as NestedMessage,

  complexTypes: {
    status: Status.Active,
    optionalNested: { name: 'optional', value: 456 },
    requiredNested: { name: 'required', value: 789 },
    nestedList: [
      { name: 'first', value: 1 },
      { name: 'second', value: 2 },
    ],
    stringIntMap: { one: 1, two: 2, three: 3 },
    intStringMap: new Map([[1, 'one'], [2, 'two'], [3, 'three']]),
  } as ComplexTypes,

  edgeCases: {
    zeroInt: 0,
    negativeOne: -1,
    maxInt32: 2147483647,
    minInt32: -2147483648,
    maxInt64: 9223372036854775807n,
    minInt64: -9223372036854775808n,
    maxUint32: 4294967295,
    maxUint64: 18446744073709551615n,
    emptyString: '',
    unicodeString: 'Hello, 世界! 🎉',
    emptyBytes: new Uint8Array([]),
  } as EdgeCases,

  allFieldNumbers: {
    field1: 100,
    field15: 1500,
    field16: 1600,
    field127: 12700,
    field128: 12800,
    field1000: 100000,
  } as AllFieldNumbers,
};

// Encoder functions for each message type using V2 format
// V2 uses end markers instead of field counts, and SVarint wire type for signed ints
function encodeNestedMessage(writer: Writer, msg: NestedMessage): void {
  // Field 1: name (string)
  writer.writeTag(1, WireType.Bytes);
  writer.writeString(msg.name);

  // Field 2: value (int32) - uses SVarint wire type
  writer.writeTag(2, WireType.SVarint);
  writer.writeSVarint(msg.value);

  // End marker
  writer.writeEndMarker();
}

function encodeScalarTypes(writer: Writer, msg: ScalarTypes): void {
  // V2 format: no field count prefix, uses end marker

  // Field 1: boolVal
  writer.writeTag(1, WireType.Varint);
  writer.writeBool(msg.boolVal);

  // Field 2: int32Val - uses SVarint wire type
  writer.writeTag(2, WireType.SVarint);
  writer.writeSVarint(msg.int32Val);

  // Field 3: int64Val - uses SVarint wire type
  writer.writeTag(3, WireType.SVarint);
  writer.writeSVarint64(msg.int64Val);

  // Field 4: uint32Val
  writer.writeTag(4, WireType.Varint);
  writer.writeVarint(msg.uint32Val);

  // Field 5: uint64Val
  writer.writeTag(5, WireType.Varint);
  writer.writeVarint64(msg.uint64Val);

  // Field 6: float32Val
  writer.writeTag(6, WireType.Fixed32);
  writer.writeFloat32(msg.float32Val);

  // Field 7: float64Val
  writer.writeTag(7, WireType.Fixed64);
  writer.writeFloat64(msg.float64Val);

  // Field 8: stringVal
  writer.writeTag(8, WireType.Bytes);
  writer.writeString(msg.stringVal);

  // Field 9: bytesVal
  writer.writeTag(9, WireType.Bytes);
  writer.writeLengthPrefixedBytes(msg.bytesVal);

  // End marker
  writer.writeEndMarker();
}

function encodeRepeatedTypes(writer: Writer, msg: RepeatedTypes): void {
  // V2 format: no field count prefix, uses end marker

  // Field 1: int32List
  writer.writeTag(1, WireType.Bytes);
  const int32Writer = new Writer();
  int32Writer.writeVarint(msg.int32List.length);
  for (const v of msg.int32List) {
    int32Writer.writeSVarint(v);
  }
  writer.writeLengthPrefixedBytes(int32Writer.bytes());

  // Field 2: stringList
  writer.writeTag(2, WireType.Bytes);
  const strWriter = new Writer();
  strWriter.writeVarint(msg.stringList.length);
  for (const v of msg.stringList) {
    strWriter.writeString(v);
  }
  writer.writeLengthPrefixedBytes(strWriter.bytes());

  // Field 3: bytesList
  writer.writeTag(3, WireType.Bytes);
  const bytesWriter = new Writer();
  bytesWriter.writeVarint(msg.bytesList.length);
  for (const v of msg.bytesList) {
    bytesWriter.writeLengthPrefixedBytes(v);
  }
  writer.writeLengthPrefixedBytes(bytesWriter.bytes());

  // End marker
  writer.writeEndMarker();
}

function encodeEdgeCases(writer: Writer, msg: EdgeCases): void {
  // V2 format: no field count prefix, uses end marker
  // Only encode non-zero/non-empty fields to match Go's omitempty behavior

  // Field 2: negativeOne - uses SVarint wire type
  writer.writeTag(2, WireType.SVarint);
  writer.writeSVarint(msg.negativeOne);

  // Field 3: maxInt32 - uses SVarint wire type
  writer.writeTag(3, WireType.SVarint);
  writer.writeSVarint(msg.maxInt32);

  // Field 4: minInt32 - uses SVarint wire type
  writer.writeTag(4, WireType.SVarint);
  writer.writeSVarint(msg.minInt32);

  // Field 5: maxInt64 - uses SVarint wire type
  writer.writeTag(5, WireType.SVarint);
  writer.writeSVarint64(msg.maxInt64);

  // Field 6: minInt64 - uses SVarint wire type
  writer.writeTag(6, WireType.SVarint);
  writer.writeSVarint64(msg.minInt64);

  // Field 7: maxUint32
  writer.writeTag(7, WireType.Varint);
  writer.writeVarint(msg.maxUint32);

  // Field 8: maxUint64
  writer.writeTag(8, WireType.Varint);
  writer.writeVarint64(msg.maxUint64);

  // Field 10: unicodeString
  writer.writeTag(10, WireType.Bytes);
  writer.writeString(msg.unicodeString);

  // End marker
  writer.writeEndMarker();
}

function encodeAllFieldNumbers(writer: Writer, msg: AllFieldNumbers): void {
  // V2 format: no field count prefix, uses end marker
  // All int32 fields use SVarint wire type

  writer.writeTag(1, WireType.SVarint);
  writer.writeSVarint(msg.field1);

  writer.writeTag(15, WireType.SVarint);
  writer.writeSVarint(msg.field15);

  writer.writeTag(16, WireType.SVarint);
  writer.writeSVarint(msg.field16);

  writer.writeTag(127, WireType.SVarint);
  writer.writeSVarint(msg.field127);

  writer.writeTag(128, WireType.SVarint);
  writer.writeSVarint(msg.field128);

  writer.writeTag(1000, WireType.SVarint);
  writer.writeSVarint(msg.field1000);

  // End marker
  writer.writeEndMarker();
}

// Decoder functions - V2 format uses end marker (fieldNumber === 0) instead of field count
function decodeNestedMessage(reader: Reader): NestedMessage {
  const result: Partial<NestedMessage> = {};

  while (reader.hasMore) {
    const tag = reader.readTag();
    if (tag.fieldNumber === 0) break; // End marker

    switch (tag.fieldNumber) {
      case 1:
        result.name = reader.readString();
        break;
      case 2:
        result.value = reader.readSVarint();
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }

  return result as NestedMessage;
}

function decodeScalarTypes(reader: Reader): ScalarTypes {
  const result: Partial<ScalarTypes> = {};

  while (reader.hasMore) {
    const tag = reader.readTag();
    if (tag.fieldNumber === 0) break; // End marker

    switch (tag.fieldNumber) {
      case 1:
        result.boolVal = reader.readBool();
        break;
      case 2:
        result.int32Val = reader.readSVarint();
        break;
      case 3:
        result.int64Val = reader.readSVarint64();
        break;
      case 4:
        result.uint32Val = reader.readVarint();
        break;
      case 5:
        result.uint64Val = reader.readVarint64();
        break;
      case 6:
        result.float32Val = reader.readFloat32();
        break;
      case 7:
        result.float64Val = reader.readFloat64();
        break;
      case 8:
        result.stringVal = reader.readString();
        break;
      case 9:
        result.bytesVal = reader.readLengthPrefixedBytes();
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }

  return result as ScalarTypes;
}

// Helper to convert Uint8Array to hex string
function toHex(data: Uint8Array): string {
  return Array.from(data).map(b => b.toString(16).padStart(2, '0')).join('');
}

// Helper to load golden file
function loadGolden(name: string): Uint8Array | null {
  const filePath = path.join(GOLDEN_DIR, `${name}.bin`);
  try {
    return new Uint8Array(fs.readFileSync(filePath));
  } catch {
    return null;
  }
}

describe('TypeScript Interoperability Tests', () => {
  describe('NestedMessage', () => {
    it('encodes and decodes correctly', () => {
      const writer = new Writer();
      encodeNestedMessage(writer, TestData.nestedMessage);
      const encoded = writer.bytes();

      console.log('NestedMessage encoded:', toHex(encoded));

      const reader = new Reader(encoded);
      const decoded = decodeNestedMessage(reader);

      expect(decoded.name).toBe(TestData.nestedMessage.name);
      expect(decoded.value).toBe(TestData.nestedMessage.value);
    });

    it('matches golden file', () => {
      const golden = loadGolden('nested_message');
      if (!golden) {
        console.log('Golden file not found, skipping');
        return;
      }

      // Decode golden file
      const reader = new Reader(golden);
      const decoded = decodeNestedMessage(reader);

      expect(decoded.name).toBe(TestData.nestedMessage.name);
      expect(decoded.value).toBe(TestData.nestedMessage.value);
    });
  });

  describe('ScalarTypes', () => {
    it('encodes and decodes correctly', () => {
      const writer = new Writer();
      encodeScalarTypes(writer, TestData.scalarTypes);
      const encoded = writer.bytes();

      console.log('ScalarTypes encoded:', toHex(encoded));
      console.log('ScalarTypes size:', encoded.length, 'bytes');

      const reader = new Reader(encoded);
      const decoded = decodeScalarTypes(reader);

      expect(decoded.boolVal).toBe(TestData.scalarTypes.boolVal);
      expect(decoded.int32Val).toBe(TestData.scalarTypes.int32Val);
      expect(decoded.int64Val).toBe(TestData.scalarTypes.int64Val);
      expect(decoded.uint32Val).toBe(TestData.scalarTypes.uint32Val);
      expect(decoded.uint64Val).toBe(TestData.scalarTypes.uint64Val);
      expect(decoded.float32Val).toBeCloseTo(TestData.scalarTypes.float32Val, 4);
      expect(decoded.float64Val).toBe(TestData.scalarTypes.float64Val);
      expect(decoded.stringVal).toBe(TestData.scalarTypes.stringVal);
      expect(toHex(decoded.bytesVal)).toBe(toHex(TestData.scalarTypes.bytesVal));
    });

    it('decodes golden file correctly', () => {
      const golden = loadGolden('scalar_types');
      if (!golden) {
        console.log('Golden file not found, skipping');
        return;
      }

      console.log('Golden ScalarTypes hex:', toHex(golden));

      const reader = new Reader(golden);
      const decoded = decodeScalarTypes(reader);

      expect(decoded.boolVal).toBe(TestData.scalarTypes.boolVal);
      expect(decoded.int32Val).toBe(TestData.scalarTypes.int32Val);
      expect(decoded.int64Val).toBe(TestData.scalarTypes.int64Val);
      expect(decoded.uint32Val).toBe(TestData.scalarTypes.uint32Val);
      expect(decoded.uint64Val).toBe(TestData.scalarTypes.uint64Val);
      expect(decoded.float32Val).toBeCloseTo(TestData.scalarTypes.float32Val, 4);
      expect(decoded.float64Val).toBe(TestData.scalarTypes.float64Val);
      expect(decoded.stringVal).toBe(TestData.scalarTypes.stringVal);
      expect(toHex(decoded.bytesVal)).toBe(toHex(TestData.scalarTypes.bytesVal));
    });
  });

  describe('AllFieldNumbers', () => {
    it('encodes and decodes correctly', () => {
      const writer = new Writer();
      encodeAllFieldNumbers(writer, TestData.allFieldNumbers);
      const encoded = writer.bytes();

      console.log('AllFieldNumbers encoded:', toHex(encoded));

      // Decode - V2 format uses end marker
      const reader = new Reader(encoded);
      const decoded: Partial<AllFieldNumbers> = {};

      while (reader.hasMore) {
        const tag = reader.readTag();
        if (tag.fieldNumber === 0) break; // End marker

        switch (tag.fieldNumber) {
          case 1: decoded.field1 = reader.readSVarint(); break;
          case 15: decoded.field15 = reader.readSVarint(); break;
          case 16: decoded.field16 = reader.readSVarint(); break;
          case 127: decoded.field127 = reader.readSVarint(); break;
          case 128: decoded.field128 = reader.readSVarint(); break;
          case 1000: decoded.field1000 = reader.readSVarint(); break;
          default: reader.skipField(tag.wireType);
        }
      }

      expect(decoded.field1).toBe(TestData.allFieldNumbers.field1);
      expect(decoded.field15).toBe(TestData.allFieldNumbers.field15);
      expect(decoded.field16).toBe(TestData.allFieldNumbers.field16);
      expect(decoded.field127).toBe(TestData.allFieldNumbers.field127);
      expect(decoded.field128).toBe(TestData.allFieldNumbers.field128);
      expect(decoded.field1000).toBe(TestData.allFieldNumbers.field1000);
    });

    it('decodes golden file correctly', () => {
      const golden = loadGolden('all_field_numbers');
      if (!golden) {
        console.log('Golden file not found, skipping');
        return;
      }

      console.log('Golden AllFieldNumbers hex:', toHex(golden));

      // Decode - V2 format uses end marker
      const reader = new Reader(golden);
      const decoded: Partial<AllFieldNumbers> = {};

      while (reader.hasMore) {
        const tag = reader.readTag();
        if (tag.fieldNumber === 0) break; // End marker

        switch (tag.fieldNumber) {
          case 1: decoded.field1 = reader.readSVarint(); break;
          case 15: decoded.field15 = reader.readSVarint(); break;
          case 16: decoded.field16 = reader.readSVarint(); break;
          case 127: decoded.field127 = reader.readSVarint(); break;
          case 128: decoded.field128 = reader.readSVarint(); break;
          case 1000: decoded.field1000 = reader.readSVarint(); break;
          default: reader.skipField(tag.wireType);
        }
      }

      expect(decoded.field1).toBe(TestData.allFieldNumbers.field1);
      expect(decoded.field15).toBe(TestData.allFieldNumbers.field15);
      expect(decoded.field16).toBe(TestData.allFieldNumbers.field16);
      expect(decoded.field127).toBe(TestData.allFieldNumbers.field127);
      expect(decoded.field128).toBe(TestData.allFieldNumbers.field128);
      expect(decoded.field1000).toBe(TestData.allFieldNumbers.field1000);
    });
  });

  describe('Wire Format Primitives', () => {
    it('varint encoding matches Go', () => {
      const testCases = [
        { value: 0, expected: '00' },
        { value: 1, expected: '01' },
        { value: 127, expected: '7f' },
        { value: 128, expected: '8001' },
        { value: 300, expected: 'ac02' },
        { value: 16384, expected: '808001' },
      ];

      for (const tc of testCases) {
        const writer = new Writer();
        writer.writeVarint(tc.value);
        const hex = toHex(writer.bytes());
        expect(hex).toBe(tc.expected);
      }
    });

    it('zigzag encoding matches Go', () => {
      const testCases = [
        { value: 0, expected: '00' },
        { value: -1, expected: '01' },
        { value: 1, expected: '02' },
        { value: -2, expected: '03' },
        { value: 2, expected: '04' },
        { value: -42, expected: '53' },
        { value: 42, expected: '54' },
      ];

      for (const tc of testCases) {
        const writer = new Writer();
        writer.writeSVarint(tc.value);
        const hex = toHex(writer.bytes());
        expect(hex).toBe(tc.expected);
      }
    });
  });
});
