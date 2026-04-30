import { describe, it, expect } from 'vitest';
import { Registry } from './registry';
import { Writer } from './writer';
import { Reader } from './reader';
import { WireType } from './types';
import { DuplicateTypeRegistrationError } from './errors';

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

      // Verify the field tag uses Bytes (canonical for length-prefixed).
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

    it('rejects duplicate type ID with different name', () => {
      const reg = new Registry();
      reg.register<string>('A', (w, v) => w.writeString(v), (r) => r.readString(), 128);
      expect(() =>
        reg.register<string>('B', (w, v) => w.writeString(v), (r) => r.readString(), 128),
      ).toThrow(DuplicateTypeRegistrationError);
    });

    it('rejects duplicate name with different type ID', () => {
      const reg = new Registry();
      reg.register<string>('A', (w, v) => w.writeString(v), (r) => r.readString(), 128);
      expect(() =>
        reg.register<string>('A', (w, v) => w.writeString(v), (r) => r.readString(), 129),
      ).toThrow(DuplicateTypeRegistrationError);
    });

    it('idempotently re-registers the same (name, id) pair', () => {
      const reg = new Registry();
      const id1 = reg.register<string>('A', (w, v) => w.writeString(v), (r) => r.readString(), 128);
      const id2 = reg.register<string>('A', (w, v) => w.writeString(v), (r) => r.readString(), 128);
      expect(id1).toBe(id2);
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
