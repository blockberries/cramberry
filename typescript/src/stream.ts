/**
 * Streaming support for Cramberry serialization.
 *
 * Provides classes for reading and writing length-delimited messages,
 * enabling efficient batch processing and incremental decoding.
 *
 * Wire format: [length: varint][message_data: bytes]
 * Compatible with Go and Rust Cramberry streaming implementations.
 */

import { Writer } from "./writer";
import { Reader } from "./reader";
import {
  DecodeError,
  EndOfStreamError,
  MessageSizeExceededError,
  StreamClosedError,
} from "./errors";

/** Default initial buffer capacity for stream writer. */
const DEFAULT_STREAM_BUFFER_CAPACITY = 4096;

/** Growth factor for stream writer buffer. */
const STREAM_GROWTH_FACTOR = 2;

/** Default maximum message size (64 MB). */
const DEFAULT_MAX_MESSAGE_SIZE = 64 * 1024 * 1024;

/**
 * Options for StreamWriter configuration.
 */
export interface StreamWriterOptions {
  /** Initial buffer capacity. Default: 4096 */
  initialCapacity?: number;
}

/**
 * Options for StreamReader configuration.
 */
export interface StreamReaderOptions {
  /** Maximum allowed message size in bytes. Default: 64 MB */
  maxMessageSize?: number;
}

/**
 * StreamWriter writes length-delimited messages to a buffer.
 *
 * Messages are written as [length: varint][data: bytes], enabling
 * streaming multiple messages and reading them back incrementally.
 *
 * @example
 * ```typescript
 * const stream = new StreamWriter();
 *
 * // Write multiple messages
 * const msg1 = new Writer();
 * msg1.writeInt32Field(1, 42);
 * stream.writeMessage(msg1.bytes());
 *
 * const msg2 = new Writer();
 * msg2.writeStringField(1, "hello");
 * stream.writeMessage(msg2.bytes());
 *
 * // Get all encoded data
 * const data = stream.bytes();
 * ```
 */
export class StreamWriter {
  private buffer: Uint8Array;
  private view: DataView;
  private pos: number;
  private closed: boolean;

  constructor(options: StreamWriterOptions = {}) {
    const capacity = options.initialCapacity ?? DEFAULT_STREAM_BUFFER_CAPACITY;
    this.buffer = new Uint8Array(capacity);
    this.view = new DataView(this.buffer.buffer);
    this.pos = 0;
    this.closed = false;
  }

  /**
   * Returns the current position (bytes written).
   */
  get position(): number {
    return this.pos;
  }

  /**
   * Returns true if the writer is closed.
   */
  get isClosed(): boolean {
    return this.closed;
  }

  /**
   * Ensures the buffer has at least the specified additional capacity.
   */
  private ensureCapacity(needed: number): void {
    const required = this.pos + needed;
    if (required <= this.buffer.length) {
      return;
    }

    let newCapacity = this.buffer.length * STREAM_GROWTH_FACTOR;
    while (newCapacity < required) {
      newCapacity *= STREAM_GROWTH_FACTOR;
    }

    const newBuffer = new Uint8Array(newCapacity);
    newBuffer.set(this.buffer.subarray(0, this.pos));
    this.buffer = newBuffer;
    this.view = new DataView(this.buffer.buffer);
  }

  /**
   * Writes a varint to the buffer.
   */
  private writeVarint(value: number): void {
    this.ensureCapacity(5); // Max 5 bytes for 32-bit varint
    while (value > 0x7f) {
      this.buffer[this.pos++] = (value & 0x7f) | 0x80;
      value >>>= 7;
    }
    this.buffer[this.pos++] = value;
  }

  /**
   * Writes raw bytes to the buffer.
   */
  private writeBytes(data: Uint8Array): void {
    this.ensureCapacity(data.length);
    this.buffer.set(data, this.pos);
    this.pos += data.length;
  }

  /**
   * Writes a length-delimited message.
   *
   * @param data - The message bytes to write
   * @throws StreamClosedError if the writer is closed
   */
  writeMessage(data: Uint8Array): void {
    if (this.closed) {
      throw new StreamClosedError();
    }
    this.writeVarint(data.length);
    this.writeBytes(data);
  }

  /**
   * Writes an encoded value using the provided encoder function.
   *
   * @param value - The value to encode
   * @param encoder - Function that encodes a value to a Writer
   * @throws StreamClosedError if the writer is closed
   */
  writeEncoded<T>(value: T, encoder: (writer: Writer, value: T) => void): void {
    const writer = new Writer();
    encoder(writer, value);
    this.writeMessage(writer.bytes());
  }

  /**
   * Returns the encoded bytes.
   */
  bytes(): Uint8Array {
    return this.buffer.subarray(0, this.pos);
  }

  /**
   * Resets the writer for reuse, clearing all written data.
   */
  reset(): void {
    this.pos = 0;
    this.closed = false;
  }

  /**
   * Closes the writer. No more messages can be written after closing.
   */
  close(): void {
    this.closed = true;
  }
}

/**
 * StreamReader reads length-delimited messages from a buffer.
 *
 * Messages are expected as [length: varint][data: bytes].
 *
 * @example
 * ```typescript
 * const reader = new StreamReader(data);
 *
 * // Read all messages
 * for (const msgData of reader.messages()) {
 *   const msg = new Reader(msgData);
 *   // decode message...
 * }
 *
 * // Or using async iteration
 * for await (const msgData of reader) {
 *   const msg = new Reader(msgData);
 *   // decode message...
 * }
 * ```
 */
export class StreamReader implements AsyncIterable<Uint8Array> {
  private buffer: Uint8Array;
  private pos: number;
  private maxMessageSize: number;

  constructor(data: Uint8Array, options: StreamReaderOptions = {}) {
    this.buffer = data;
    this.pos = 0;
    this.maxMessageSize = options.maxMessageSize ?? DEFAULT_MAX_MESSAGE_SIZE;
  }

  /**
   * Returns the current position in the buffer.
   */
  get position(): number {
    return this.pos;
  }

  /**
   * Returns the number of bytes remaining.
   */
  get remaining(): number {
    return this.buffer.length - this.pos;
  }

  /**
   * Returns true if there is more data to read.
   */
  get hasMore(): boolean {
    return this.pos < this.buffer.length;
  }

  /**
   * Sets the maximum allowed message size.
   */
  setMaxMessageSize(size: number): void {
    this.maxMessageSize = size;
  }

  /**
   * Reads a varint from the buffer.
   * Returns null if at end of stream.
   * @throws EndOfStreamError if varint is incomplete
   */
  private readVarint(): number | null {
    if (this.pos >= this.buffer.length) {
      return null; // Clean EOF
    }

    let result = 0;
    let shift = 0;
    const startPos = this.pos;

    for (let i = 0; i < 5; i++) {
      if (this.pos >= this.buffer.length) {
        // Incomplete varint - restore position and throw
        this.pos = startPos;
        throw new EndOfStreamError("Incomplete varint at end of stream");
      }

      const b = this.buffer[this.pos++];
      result |= (b & 0x7f) << shift;
      if ((b & 0x80) === 0) {
        return result >>> 0; // Ensure unsigned
      }
      shift += 7;
    }

    throw new DecodeError("Varint overflow: exceeded 5 bytes for message length");
  }

  /**
   * Reads a length-delimited message.
   *
   * @returns The message data
   * @throws EndOfStreamError if stream ends unexpectedly
   * @throws MessageSizeExceededError if message exceeds max size
   */
  readMessage(): Uint8Array {
    const length = this.readVarint();
    if (length === null) {
      throw new EndOfStreamError("No more messages");
    }

    if (length > this.maxMessageSize) {
      throw new MessageSizeExceededError(length, this.maxMessageSize);
    }

    if (this.pos + length > this.buffer.length) {
      throw new EndOfStreamError(
        `Message claims ${length} bytes but only ${this.remaining} available`
      );
    }

    const data = this.buffer.subarray(this.pos, this.pos + length);
    this.pos += length;
    return data;
  }

  /**
   * Attempts to read a message, returning null if at end of stream.
   *
   * @returns The message data, or null if at end of stream
   * @throws EndOfStreamError if stream ends mid-message
   * @throws MessageSizeExceededError if message exceeds max size
   */
  tryReadMessage(): Uint8Array | null {
    if (!this.hasMore) {
      return null;
    }

    const length = this.readVarint();
    if (length === null) {
      return null;
    }

    if (length > this.maxMessageSize) {
      throw new MessageSizeExceededError(length, this.maxMessageSize);
    }

    if (this.pos + length > this.buffer.length) {
      throw new EndOfStreamError(
        `Message claims ${length} bytes but only ${this.remaining} available`
      );
    }

    const data = this.buffer.subarray(this.pos, this.pos + length);
    this.pos += length;
    return data;
  }

  /**
   * Reads and decodes a message using the provided decoder function.
   *
   * @param decoder - Function that decodes from a Reader
   * @returns The decoded value
   */
  readDecoded<T>(decoder: (reader: Reader) => T): T {
    const data = this.readMessage();
    const reader = new Reader(data);
    return decoder(reader);
  }

  /**
   * Skips the next message without reading its contents.
   *
   * @returns The number of bytes skipped (including length prefix)
   * @throws EndOfStreamError if no message to skip
   */
  skipMessage(): number {
    const startPos = this.pos;
    const length = this.readVarint();
    if (length === null) {
      throw new EndOfStreamError("No message to skip");
    }

    if (this.pos + length > this.buffer.length) {
      throw new EndOfStreamError(
        `Cannot skip: message claims ${length} bytes but only ${this.remaining} available`
      );
    }

    this.pos += length;
    return this.pos - startPos;
  }

  /**
   * Resets the reader to the beginning of the buffer.
   */
  reset(): void {
    this.pos = 0;
  }

  /**
   * Creates an async iterator over all messages.
   * Implements AsyncIterable for use with for-await-of.
   */
  async *[Symbol.asyncIterator](): AsyncIterableIterator<Uint8Array> {
    while (this.hasMore) {
      const data = this.tryReadMessage();
      if (data === null) {
        break;
      }
      yield data;
    }
  }

  /**
   * Returns a synchronous iterator over all messages.
   */
  *messages(): IterableIterator<Uint8Array> {
    while (this.hasMore) {
      const data = this.tryReadMessage();
      if (data === null) {
        break;
      }
      yield data;
    }
  }
}

/**
 * MessageIterator provides a convenient wrapper for iterating over
 * messages with automatic decoding.
 *
 * @example
 * ```typescript
 * interface MyMessage { id: number; name: string; }
 *
 * const decoder = (reader: Reader): MyMessage => ({
 *   id: reader.readInt32(),
 *   name: reader.readString(),
 * });
 *
 * const iterator = new MessageIterator(data, decoder);
 *
 * for (const msg of iterator) {
 *   console.log(msg.id, msg.name);
 * }
 *
 * // Or collect all at once
 * const messages = iterator.toArray();
 * ```
 */
export class MessageIterator<T> implements Iterable<T>, AsyncIterable<T> {
  private reader: StreamReader;
  private decoder: (reader: Reader) => T;
  private _error: Error | null = null;

  constructor(
    data: Uint8Array,
    decoder: (reader: Reader) => T,
    options: StreamReaderOptions = {}
  ) {
    this.reader = new StreamReader(data, options);
    this.decoder = decoder;
  }

  /**
   * Returns any error that occurred during iteration.
   */
  get error(): Error | null {
    return this._error;
  }

  /**
   * Returns true if there are more messages to read.
   */
  get hasMore(): boolean {
    return this.reader.hasMore;
  }

  /**
   * Reads and decodes the next message.
   *
   * @returns The decoded value, or null if at end of stream or on error
   */
  next(): T | null {
    try {
      const data = this.reader.tryReadMessage();
      if (data === null) {
        return null;
      }
      const reader = new Reader(data);
      return this.decoder(reader);
    } catch (e) {
      this._error = e instanceof Error ? e : new Error(String(e));
      return null;
    }
  }

  /**
   * Implements AsyncIterable for use with for-await-of.
   */
  async *[Symbol.asyncIterator](): AsyncIterableIterator<T> {
    while (this.reader.hasMore) {
      const value = this.next();
      if (value === null) {
        break;
      }
      yield value;
    }
  }

  /**
   * Returns a synchronous iterator.
   */
  *[Symbol.iterator](): IterableIterator<T> {
    while (this.reader.hasMore) {
      const value = this.next();
      if (value === null) {
        break;
      }
      yield value;
    }
  }

  /**
   * Collects all messages into an array.
   */
  toArray(): T[] {
    const results: T[] = [];
    for (const item of this) {
      results.push(item);
    }
    return results;
  }

  /**
   * Resets the iterator to the beginning.
   */
  reset(): void {
    this.reader.reset();
    this._error = null;
  }
}
