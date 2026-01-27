import { describe, it, expect } from "vitest";
import {
  StreamWriter,
  StreamReader,
  MessageIterator,
} from "./stream";
import { Writer } from "./writer";
import { Reader } from "./reader";
import {
  EndOfStreamError,
  MessageSizeExceededError,
  StreamClosedError,
} from "./errors";

describe("StreamWriter", () => {
  describe("basic operations", () => {
    it("writes a single message", () => {
      const stream = new StreamWriter();
      stream.writeMessage(new Uint8Array([1, 2, 3]));
      const data = stream.bytes();

      // Length prefix (3) + data (1, 2, 3)
      expect(data).toEqual(new Uint8Array([3, 1, 2, 3]));
    });

    it("writes multiple messages", () => {
      const stream = new StreamWriter();
      stream.writeMessage(new Uint8Array([1, 2]));
      stream.writeMessage(new Uint8Array([3, 4, 5]));
      const data = stream.bytes();

      expect(data).toEqual(new Uint8Array([2, 1, 2, 3, 3, 4, 5]));
    });

    it("writes empty message", () => {
      const stream = new StreamWriter();
      stream.writeMessage(new Uint8Array([]));
      const data = stream.bytes();

      expect(data).toEqual(new Uint8Array([0]));
    });

    it("handles large messages requiring multi-byte varint", () => {
      const stream = new StreamWriter();
      const largeData = new Uint8Array(300);
      largeData.fill(0xab);
      stream.writeMessage(largeData);
      const data = stream.bytes();

      // 300 = 0xAC 0x02 in varint encoding
      expect(data[0]).toBe(0xac);
      expect(data[1]).toBe(0x02);
      expect(data.length).toBe(302);
    });

    it("tracks position correctly", () => {
      const stream = new StreamWriter();
      expect(stream.position).toBe(0);

      stream.writeMessage(new Uint8Array([1, 2, 3]));
      expect(stream.position).toBe(4); // 1 byte length + 3 bytes data

      stream.writeMessage(new Uint8Array([4, 5]));
      expect(stream.position).toBe(7); // Previous + 1 byte length + 2 bytes data
    });
  });

  describe("writeEncoded", () => {
    it("encodes value using provided encoder", () => {
      const stream = new StreamWriter();
      stream.writeEncoded(42, (writer, value) => {
        writer.writeInt32Field(1, value);
      });

      const data = stream.bytes();
      expect(data.length).toBeGreaterThan(1);

      // Verify we can read it back
      const reader = new StreamReader(data);
      const msgData = reader.readMessage();
      const msgReader = new Reader(msgData);
      const tag = msgReader.readTag();
      expect(tag.fieldNumber).toBe(1);
      expect(msgReader.readInt32()).toBe(42);
    });

    it("writes multiple encoded values", () => {
      const stream = new StreamWriter();

      stream.writeEncoded("hello", (writer, value) => {
        writer.writeStringField(1, value);
      });
      stream.writeEncoded("world", (writer, value) => {
        writer.writeStringField(1, value);
      });

      const reader = new StreamReader(stream.bytes());
      const messages = [...reader.messages()];
      expect(messages.length).toBe(2);
    });
  });

  describe("lifecycle", () => {
    it("throws when writing to closed writer", () => {
      const stream = new StreamWriter();
      stream.close();

      expect(() => stream.writeMessage(new Uint8Array([1]))).toThrow(
        StreamClosedError
      );
    });

    it("reports closed state", () => {
      const stream = new StreamWriter();
      expect(stream.isClosed).toBe(false);

      stream.close();
      expect(stream.isClosed).toBe(true);
    });

    it("reset allows reuse", () => {
      const stream = new StreamWriter();
      stream.writeMessage(new Uint8Array([1, 2, 3]));
      stream.close();
      stream.reset();

      expect(stream.isClosed).toBe(false);
      stream.writeMessage(new Uint8Array([4, 5]));
      expect(stream.bytes()).toEqual(new Uint8Array([2, 4, 5]));
    });
  });

  describe("buffer growth", () => {
    it("grows buffer for large writes", () => {
      const stream = new StreamWriter({ initialCapacity: 16 });
      const largeData = new Uint8Array(100);
      largeData.fill(0x42);

      stream.writeMessage(largeData);
      expect(stream.bytes().length).toBe(101); // 1 byte length + 100 bytes data
    });

    it("handles many small messages", () => {
      const stream = new StreamWriter({ initialCapacity: 16 });

      for (let i = 0; i < 100; i++) {
        stream.writeMessage(new Uint8Array([i & 0xff]));
      }

      const reader = new StreamReader(stream.bytes());
      let count = 0;
      for (const msg of reader.messages()) {
        expect(msg[0]).toBe(count & 0xff);
        count++;
      }
      expect(count).toBe(100);
    });
  });
});

describe("StreamReader", () => {
  describe("basic operations", () => {
    it("reads a single message", () => {
      const data = new Uint8Array([3, 1, 2, 3]);
      const reader = new StreamReader(data);

      const msg = reader.readMessage();
      expect(msg).toEqual(new Uint8Array([1, 2, 3]));
      expect(reader.hasMore).toBe(false);
    });

    it("reads multiple messages", () => {
      const data = new Uint8Array([2, 1, 2, 3, 3, 4, 5]);
      const reader = new StreamReader(data);

      expect(reader.readMessage()).toEqual(new Uint8Array([1, 2]));
      expect(reader.readMessage()).toEqual(new Uint8Array([3, 4, 5]));
      expect(reader.hasMore).toBe(false);
    });

    it("reads empty message", () => {
      const data = new Uint8Array([0]);
      const reader = new StreamReader(data);

      expect(reader.readMessage()).toEqual(new Uint8Array([]));
    });

    it("handles multi-byte varint length", () => {
      const stream = new StreamWriter();
      const largeData = new Uint8Array(300);
      largeData.fill(0xab);
      stream.writeMessage(largeData);

      const reader = new StreamReader(stream.bytes());
      const msg = reader.readMessage();
      expect(msg.length).toBe(300);
      expect(msg.every((b) => b === 0xab)).toBe(true);
    });

    it("tracks position and remaining", () => {
      const data = new Uint8Array([3, 1, 2, 3, 2, 4, 5]);
      const reader = new StreamReader(data);

      expect(reader.position).toBe(0);
      expect(reader.remaining).toBe(7);

      reader.readMessage();
      expect(reader.position).toBe(4);
      expect(reader.remaining).toBe(3);
    });
  });

  describe("tryReadMessage", () => {
    it("returns null at end of stream", () => {
      const data = new Uint8Array([3, 1, 2, 3]);
      const reader = new StreamReader(data);

      expect(reader.tryReadMessage()).toEqual(new Uint8Array([1, 2, 3]));
      expect(reader.tryReadMessage()).toBeNull();
    });

    it("returns null for empty buffer", () => {
      const reader = new StreamReader(new Uint8Array([]));
      expect(reader.tryReadMessage()).toBeNull();
    });
  });

  describe("readDecoded", () => {
    it("decodes message using provided decoder", () => {
      const stream = new StreamWriter();
      stream.writeEncoded({ id: 42, name: "test" }, (writer, value) => {
        writer.writeInt32Field(1, value.id);
        writer.writeStringField(2, value.name);
        writer.writeEndMarker();
      });

      const reader = new StreamReader(stream.bytes());
      const decoded = reader.readDecoded((r) => {
        const result: { id?: number; name?: string } = {};
        while (!r.isEndMarker()) {
          const { fieldNumber } = r.readTag();
          switch (fieldNumber) {
            case 1:
              result.id = r.readInt32();
              break;
            case 2:
              result.name = r.readString();
              break;
          }
        }
        return result;
      });

      expect(decoded.id).toBe(42);
      expect(decoded.name).toBe("test");
    });
  });

  describe("error handling", () => {
    it("throws on incomplete varint", () => {
      const data = new Uint8Array([0x80]); // Incomplete varint
      const reader = new StreamReader(data);

      expect(() => reader.readMessage()).toThrow(EndOfStreamError);
    });

    it("throws when message data is incomplete", () => {
      const data = new Uint8Array([10, 1, 2, 3]); // Claims 10 bytes, has 3
      const reader = new StreamReader(data);

      expect(() => reader.readMessage()).toThrow(EndOfStreamError);
    });

    it("throws when message exceeds max size", () => {
      const stream = new StreamWriter();
      stream.writeMessage(new Uint8Array(1000));

      const reader = new StreamReader(stream.bytes(), { maxMessageSize: 100 });
      expect(() => reader.readMessage()).toThrow(MessageSizeExceededError);
    });

    it("throws on readMessage when no more data", () => {
      const reader = new StreamReader(new Uint8Array([]));
      expect(() => reader.readMessage()).toThrow(EndOfStreamError);
    });

    it("allows setting max message size", () => {
      const stream = new StreamWriter();
      stream.writeMessage(new Uint8Array(500));

      const reader = new StreamReader(stream.bytes());
      reader.setMaxMessageSize(100);
      expect(() => reader.readMessage()).toThrow(MessageSizeExceededError);
    });
  });

  describe("skipMessage", () => {
    it("skips a message", () => {
      const data = new Uint8Array([3, 1, 2, 3, 2, 4, 5]);
      const reader = new StreamReader(data);

      const skipped = reader.skipMessage();
      expect(skipped).toBe(4); // 1 byte length + 3 bytes data
      expect(reader.readMessage()).toEqual(new Uint8Array([4, 5]));
    });

    it("throws when no message to skip", () => {
      const reader = new StreamReader(new Uint8Array([]));
      expect(() => reader.skipMessage()).toThrow(EndOfStreamError);
    });

    it("throws when message to skip is incomplete", () => {
      const data = new Uint8Array([10, 1, 2]); // Claims 10 bytes
      const reader = new StreamReader(data);
      expect(() => reader.skipMessage()).toThrow(EndOfStreamError);
    });
  });

  describe("reset", () => {
    it("resets to beginning", () => {
      const data = new Uint8Array([3, 1, 2, 3]);
      const reader = new StreamReader(data);

      reader.readMessage();
      expect(reader.hasMore).toBe(false);

      reader.reset();
      expect(reader.position).toBe(0);
      expect(reader.hasMore).toBe(true);
      expect(reader.readMessage()).toEqual(new Uint8Array([1, 2, 3]));
    });
  });

  describe("iteration", () => {
    it("iterates with for-of using messages()", () => {
      const stream = new StreamWriter();
      stream.writeMessage(new Uint8Array([1]));
      stream.writeMessage(new Uint8Array([2]));
      stream.writeMessage(new Uint8Array([3]));

      const reader = new StreamReader(stream.bytes());
      const messages: Uint8Array[] = [];

      for (const msg of reader.messages()) {
        messages.push(msg);
      }

      expect(messages.length).toBe(3);
      expect(messages[0]).toEqual(new Uint8Array([1]));
      expect(messages[1]).toEqual(new Uint8Array([2]));
      expect(messages[2]).toEqual(new Uint8Array([3]));
    });

    it("supports async iteration", async () => {
      const stream = new StreamWriter();
      stream.writeMessage(new Uint8Array([1]));
      stream.writeMessage(new Uint8Array([2]));

      const reader = new StreamReader(stream.bytes());
      const messages: Uint8Array[] = [];

      for await (const msg of reader) {
        messages.push(msg);
      }

      expect(messages.length).toBe(2);
    });

    it("handles empty stream in iteration", () => {
      const reader = new StreamReader(new Uint8Array([]));
      const messages = [...reader.messages()];
      expect(messages.length).toBe(0);
    });
  });
});

describe("MessageIterator", () => {
  interface TestMessage {
    value: number;
  }

  const decoder = (reader: Reader): TestMessage => {
    const { fieldNumber } = reader.readTag();
    expect(fieldNumber).toBe(1);
    return { value: reader.readInt32() };
  };

  it("iterates over decoded messages", () => {
    const stream = new StreamWriter();

    for (const v of [10, 20, 30]) {
      stream.writeEncoded(v, (writer, value) => {
        writer.writeInt32Field(1, value);
      });
    }

    const iterator = new MessageIterator(stream.bytes(), decoder);
    const values = iterator.toArray();

    expect(values.length).toBe(3);
    expect(values[0].value).toBe(10);
    expect(values[1].value).toBe(20);
    expect(values[2].value).toBe(30);
  });

  it("supports for-of iteration", () => {
    const stream = new StreamWriter();
    stream.writeEncoded(42, (writer, value) => {
      writer.writeInt32Field(1, value);
    });

    const iterator = new MessageIterator(stream.bytes(), decoder);
    let count = 0;

    for (const msg of iterator) {
      expect(msg.value).toBe(42);
      count++;
    }

    expect(count).toBe(1);
  });

  it("supports async iteration", async () => {
    const stream = new StreamWriter();
    stream.writeEncoded(42, (writer, value) => {
      writer.writeInt32Field(1, value);
    });

    const iterator = new MessageIterator(stream.bytes(), decoder);
    const values: TestMessage[] = [];

    for await (const msg of iterator) {
      values.push(msg);
    }

    expect(values.length).toBe(1);
    expect(values[0].value).toBe(42);
  });

  it("captures errors during decoding", () => {
    const badDecoder = (): TestMessage => {
      throw new Error("Decode failed");
    };

    const stream = new StreamWriter();
    stream.writeMessage(new Uint8Array([1, 2, 3]));

    const iterator = new MessageIterator(stream.bytes(), badDecoder);
    const result = iterator.next();

    expect(result).toBeNull();
    expect(iterator.error).not.toBeNull();
    expect(iterator.error?.message).toBe("Decode failed");
  });

  it("reports hasMore correctly", () => {
    const stream = new StreamWriter();
    stream.writeEncoded(1, (writer, value) => {
      writer.writeInt32Field(1, value);
    });
    stream.writeEncoded(2, (writer, value) => {
      writer.writeInt32Field(1, value);
    });

    const iterator = new MessageIterator(stream.bytes(), decoder);
    expect(iterator.hasMore).toBe(true);

    iterator.next();
    expect(iterator.hasMore).toBe(true);

    iterator.next();
    expect(iterator.hasMore).toBe(false);
  });

  it("reset allows re-iteration", () => {
    const stream = new StreamWriter();
    stream.writeEncoded(42, (writer, value) => {
      writer.writeInt32Field(1, value);
    });

    const iterator = new MessageIterator(stream.bytes(), decoder);

    // First iteration
    const first = iterator.toArray();
    expect(first.length).toBe(1);
    expect(iterator.hasMore).toBe(false);

    // Reset and iterate again
    iterator.reset();
    expect(iterator.hasMore).toBe(true);
    const second = iterator.toArray();
    expect(second.length).toBe(1);
  });

  it("handles empty stream", () => {
    const iterator = new MessageIterator(new Uint8Array([]), decoder);
    expect(iterator.hasMore).toBe(false);
    expect(iterator.next()).toBeNull();
    expect(iterator.toArray()).toEqual([]);
  });
});

describe("roundtrip", () => {
  it("roundtrips multiple complex messages", () => {
    interface Person {
      id: number;
      name: string;
      active: boolean;
    }

    const encoder = (writer: Writer, person: Person) => {
      writer.writeInt32Field(1, person.id);
      writer.writeStringField(2, person.name);
      writer.writeBoolField(3, person.active);
      writer.writeEndMarker();
    };

    const decoder = (reader: Reader): Person => {
      const person: Partial<Person> = {};
      while (!reader.isEndMarker()) {
        const { fieldNumber } = reader.readTag();
        switch (fieldNumber) {
          case 1:
            person.id = reader.readInt32();
            break;
          case 2:
            person.name = reader.readString();
            break;
          case 3:
            person.active = reader.readBool();
            break;
        }
      }
      return person as Person;
    };

    const people: Person[] = [
      { id: 1, name: "Alice", active: true },
      { id: 2, name: "Bob", active: false },
      { id: 3, name: "Charlie", active: true },
    ];

    // Encode
    const stream = new StreamWriter();
    for (const person of people) {
      stream.writeEncoded(person, encoder);
    }

    // Decode
    const iterator = new MessageIterator(stream.bytes(), decoder);
    const decoded = iterator.toArray();

    expect(decoded).toEqual(people);
  });

  it("roundtrips with Go-compatible wire format", () => {
    // Verify wire format is compatible with Go/Rust
    const stream = new StreamWriter();
    stream.writeMessage(new Uint8Array([1, 2, 3, 4, 5]));
    stream.writeMessage(new Uint8Array([6, 7, 8]));

    // Expected wire format: [5, 1, 2, 3, 4, 5, 3, 6, 7, 8]
    const expected = new Uint8Array([5, 1, 2, 3, 4, 5, 3, 6, 7, 8]);
    expect(stream.bytes()).toEqual(expected);

    // Verify reading produces same data
    const reader = new StreamReader(expected);
    expect(reader.readMessage()).toEqual(new Uint8Array([1, 2, 3, 4, 5]));
    expect(reader.readMessage()).toEqual(new Uint8Array([6, 7, 8]));
  });
});
