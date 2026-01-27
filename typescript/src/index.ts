/**
 * @cramberry/runtime - Cramberry serialization runtime for TypeScript
 *
 * A high-performance, compact binary serialization library.
 *
 * @example
 * ```typescript
 * import { Writer, Reader } from '@cramberry/runtime';
 *
 * // Encoding
 * const writer = new Writer();
 * writer.writeInt32Field(1, 42);
 * writer.writeStringField(2, "hello");
 * const data = writer.bytes();
 *
 * // Decoding
 * const reader = new Reader(data);
 * while (reader.hasMore) {
 *   const { fieldNumber, wireType } = reader.readTag();
 *   // Handle fields...
 * }
 * ```
 */

// Core types
export {
  WireType,
  TypeID,
  FieldTag,
  CompactTagResult,
  MaxVarint32,
  MaxVarint64,
  MinInt64,
  MaxInt64,
  // V2 compact tag format
  END_MARKER,
  TAG_EXTENDED_BIT,
  TAG_WIRE_TYPE_MASK,
  TAG_WIRE_TYPE_SHIFT,
  TAG_FIELD_NUM_SHIFT,
  MAX_COMPACT_FIELD_NUM,
  encodeCompactTag,
  decodeCompactTag,
  // Legacy (deprecated)
  encodeTag,
  decodeTag,
  zigzagEncode,
  zigzagEncode64,
  zigzagDecode,
  zigzagDecode64,
} from "./types";

// Errors
export {
  CramberryError,
  EncodeError,
  DecodeError,
  BufferOverflowError,
  BufferUnderflowError,
  UnknownTypeError,
  TypeNotRegisteredError,
  InvalidWireTypeError,
  EndOfStreamError,
  MessageSizeExceededError,
  StreamClosedError,
} from "./errors";

// Streaming support
export {
  StreamWriter,
  StreamWriterOptions,
  StreamReader,
  StreamReaderOptions,
  MessageIterator,
} from "./stream";

// Writer
import { Writer } from "./writer";
export { Writer };

// Reader
import { Reader } from "./reader";
export { Reader };

// Registry
export {
  Registry,
  Encoder,
  Decoder,
  Sizer,
  defaultRegistry,
  register,
} from "./registry";

/**
 * Library version.
 */
export const VERSION = "1.4.3";

/**
 * Marshal encodes a value using a custom encoder function.
 */
export function marshal<T>(value: T, encoder: (writer: Writer, value: T) => void): Uint8Array {
  const writer = new Writer();
  encoder(writer, value);
  return writer.bytes();
}

/**
 * Unmarshal decodes a value using a custom decoder function.
 */
export function unmarshal<T>(data: Uint8Array, decoder: (reader: Reader) => T): T {
  const reader = new Reader(data);
  return decoder(reader);
}
