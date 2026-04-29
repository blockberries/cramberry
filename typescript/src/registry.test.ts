import { describe, it, expect } from 'vitest';
import { Registry } from './registry';
import { Writer } from './writer';
import { Reader } from './reader';
import { WireType } from './types';

describe('Registry', () => {
  describe('encodePolymorphic / decodePolymorphic', () => {
    it('round-trips a registered value', () => {
      const reg = new Registry();
      const id = reg.register<string>(
        'Greeting',
        (w, v) => w.writeString(v),
        (r) => r.readString(),
        128,
      );
      expect(id).toBe(128);

      const writer = new Writer();
      reg.encodePolymorphic(writer, 1, 'Greeting', 'hello');
      const bytes = writer.bytes();

      // Verify the field tag uses Bytes (V2 canonical), not the deleted
      // TypeRef wire type (which would land as undefined → 0 = Varint).
      const reader = new Reader(bytes);
      const tag = reader.readTag();
      expect(tag.fieldNumber).toBe(1);
      expect(tag.wireType).toBe(WireType.Bytes);

      const { name, value } = reg.decodePolymorphic(reader);
      expect(name).toBe('Greeting');
      expect(value).toBe('hello');
    });

    it('rejects unregistered type at encode time', () => {
      const reg = new Registry();
      const writer = new Writer();
      expect(() => reg.encodePolymorphic(writer, 1, 'Missing', 'x')).toThrow();
    });

    it('rejects unregistered type ID at decode time', () => {
      const reg = new Registry();
      reg.register<string>(
        'A',
        (w, v) => w.writeString(v),
        (r) => r.readString(),
        128,
      );
      const writer = new Writer();
      reg.encodePolymorphic(writer, 1, 'A', 'x');

      const decoder = new Registry(); // empty
      const reader = new Reader(writer.bytes());
      reader.readTag(); // consume the field tag
      expect(() => decoder.decodePolymorphic(reader)).toThrow();
    });
  });
});
