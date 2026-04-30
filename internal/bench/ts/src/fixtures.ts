// Fixture builders shared between the bench harness. The values mirror
// `benchmark_test.go::makeCramberry*` in Go and `fixtures::cram_*` in
// Rust so wire output, allocation counts, and decode work are directly
// comparable across all three languages.

import {
  type Address,
  type Attachment,
  type BatchRequest,
  type Comment,
  type ContactInfo,
  type Document,
  type Event,
  EventType,
  type Metrics,
  type Person,
  type Point,
  Priority,
  type SmallMessage,
  Status,
  type Tag,
  type Timestamp,
} from './messages.js';

export function smallMessage(): SmallMessage {
  return { id: 12345n, name: 'test-item', active: true };
}

export function point(): Point {
  return { x: 123.456, y: 789.012, z: 345.678 };
}

export function timestamp(): Timestamp {
  return { seconds: 1705900800n, nanos: 123456789 };
}

export function metrics(): Metrics {
  return {
    count: 1_000_000n,
    sum: 12_345_678.9,
    min: 0.001,
    max: 99_999.99,
    avg: 12_345.67,
    p50: 10_000.0,
    p95: 50_000.0,
    p99: 90_000.0,
    totalBytes: 1_073_741_824n,
    errorCount: 42n,
  };
}

export function address(): Address {
  return {
    street1: '123 Main Street',
    street2: 'Suite 100',
    city: 'San Francisco',
    state: 'CA',
    postalCode: '94105',
    country: 'USA',
    coordinates: point(),
  };
}

export function contactInfo(): ContactInfo {
  return {
    email: 'john.doe@example.com',
    phone: '+1-555-123-4567',
    mobile: '+1-555-987-6543',
    mailingAddress: address(),
  };
}

export function person(): Person {
  return {
    id: 1001n,
    firstName: 'John',
    lastName: 'Doe',
    middleName: 'Robert',
    dateOfBirth: timestamp(),
    contact: contactInfo(),
    status: Status.Active,
    createdAt: timestamp(),
    updatedAt: timestamp(),
  };
}

export function document(): Document {
  const tags: Tag[] = [
    { key: 'category', value: 'technical' },
    { key: 'status', value: 'reviewed' },
    { key: 'version', value: '2.0' },
  ];
  const attachments: Attachment[] = [
    {
      id: 'att-001',
      filename: 'report.pdf',
      mimeType: 'application/pdf',
      sizeBytes: 1_048_576n,
      checksum: new Uint8Array([0xde, 0xad, 0xbe, 0xef]),
      uploadedAt: timestamp(),
    },
  ];
  const comments: Comment[] = [
    {
      id: 3001n,
      authorId: 1002n,
      content: 'Great document!',
      createdAt: timestamp(),
      reactions: [1001n, 1003n, 1004n],
    },
  ];
  return {
    id: 2001n,
    title: 'Important Document Title',
    content:
      'This is the document content with some meaningful text that would typically be much longer in a real application.',
    authorId: 1001n,
    status: Status.Active,
    priority: Priority.High,
    tags,
    attachments,
    comments,
    metadata: new Map([
      ['source', 'import'],
      ['encoding', 'utf-8'],
      ['version', '1.0'],
    ]),
    collaborators: [1001n, 1002n, 1003n],
    createdAt: timestamp(),
    updatedAt: timestamp(),
    publishedAt: timestamp(),
  };
}

export function event(): Event {
  return {
    id: 'evt-001',
    type: EventType.Created,
    entityType: 'document',
    entityId: 'doc-2001',
    source: {
      service: 'document-service',
      instance: 'prod-01',
      version: '1.2.3',
      region: 'us-west-2',
    },
    timestamp: timestamp(),
    attributes: new Map([
      ['user_id', '1001'],
      ['action', 'create'],
    ]),
    payload: new TextEncoder().encode('{"action":"click","element":"button"}'),
    correlationId: 'corr-123',
    causationId: 'caus-456',
  };
}

export function batchRequest(size: number): BatchRequest {
  const items: SmallMessage[] = new Array(size);
  for (let i = 0; i < size; i++) {
    items[i] = { id: BigInt(i), name: 'batch-item', active: i % 2 === 0 };
  }
  return {
    requestId: 'batch-001',
    items,
    headers: new Map([
      ['Content-Type', 'application/x-cramberry'],
      ['X-Request-Id', 'req-123'],
    ]),
    submittedAt: timestamp(),
    priority: Priority.Medium,
  };
}

// JSON serialisation cannot natively handle bigint or Map. The fixtures
// above use those, so we provide a small replacer/reviver pair to
// produce a valid JSON encoding shape comparable to Go's `encoding/json`
// output (numbers as numbers, maps as objects).
//
// This is informational only — the tinybench json/encode benchmark uses
// these helpers, not protobuf or cramberry.

export function jsonReplacer(_key: string, value: unknown): unknown {
  if (typeof value === 'bigint') return Number(value);
  if (value instanceof Map) return Object.fromEntries(value);
  if (value instanceof Uint8Array) return Array.from(value);
  return value;
}
