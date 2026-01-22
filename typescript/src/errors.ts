/**
 * Base error class for Cramberry errors.
 */
export class CramberryError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "CramberryError";
  }
}

/**
 * Error thrown when encoding fails.
 */
export class EncodeError extends CramberryError {
  constructor(message: string) {
    super(message);
    this.name = "EncodeError";
  }
}

/**
 * Error thrown when decoding fails.
 */
export class DecodeError extends CramberryError {
  constructor(message: string) {
    super(message);
    this.name = "DecodeError";
  }
}

/**
 * Error thrown when buffer overflows during encoding.
 */
export class BufferOverflowError extends EncodeError {
  constructor(needed: number, available: number) {
    super(`Buffer overflow: needed ${needed} bytes, only ${available} available`);
    this.name = "BufferOverflowError";
  }
}

/**
 * Error thrown when buffer is exhausted during decoding.
 */
export class BufferUnderflowError extends DecodeError {
  constructor(needed: number, available: number) {
    super(`Buffer underflow: needed ${needed} bytes, only ${available} available`);
    this.name = "BufferUnderflowError";
  }
}

/**
 * Error thrown when an unknown type is encountered.
 */
export class UnknownTypeError extends CramberryError {
  constructor(typeId: number) {
    super(`Unknown type ID: ${typeId}`);
    this.name = "UnknownTypeError";
  }
}

/**
 * Error thrown when a type is not registered.
 */
export class TypeNotRegisteredError extends CramberryError {
  constructor(typeName: string) {
    super(`Type not registered: ${typeName}`);
    this.name = "TypeNotRegisteredError";
  }
}

/**
 * Error thrown when an invalid wire type is encountered.
 */
export class InvalidWireTypeError extends DecodeError {
  constructor(expected: number, actual: number) {
    super(`Invalid wire type: expected ${expected}, got ${actual}`);
    this.name = "InvalidWireTypeError";
  }
}
