// Cross-language bench harness — TypeScript port.
//
// Mirrors `internal/bench/benchmark_test.go` (Go) and
// `internal/bench/rust/benches/encode_decode.rs` (Rust).
//
// Run with:
//     npm run bench                    # full suite
//     npm run bench -- SmallMessage    # filter

import { Bench } from 'tinybench';
import { Reader, Writer } from '@cramberry/runtime';
import protobuf from 'protobufjs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

import * as cg from './messages.js';
import * as fx from './fixtures.js';

// ---------------------------------------------------------------------------
// Protobuf types loaded from the .proto file
// ---------------------------------------------------------------------------

const here = dirname(fileURLToPath(import.meta.url));
const protoPath = resolve(here, '../../schemas/messages.proto');
const root = await protobuf.load(protoPath);

const PB = {
  SmallMessage: root.lookupType('benchmark.SmallMessage'),
  Metrics: root.lookupType('benchmark.Metrics'),
  Person: root.lookupType('benchmark.Person'),
  Document: root.lookupType('benchmark.Document'),
  Event: root.lookupType('benchmark.Event'),
  BatchRequest: root.lookupType('benchmark.BatchRequest'),
};

// Convert the Cramberry-typed fixtures into the plain-object shape
// protobufjs expects (no bigints — protobufjs uses Long.js for int64).
// Keeping the construction in line with the Go fixtures means the
// encoded payload sizes line up with the cross-language baselines.

function pbSmall(): Record<string, unknown> {
  return { id: 12345, name: 'test-item', active: true };
}

function pbMetrics(): Record<string, unknown> {
  return {
    count: 1_000_000,
    sum: 12_345_678.9,
    min: 0.001,
    max: 99_999.99,
    avg: 12_345.67,
    p50: 10_000.0,
    p95: 50_000.0,
    p99: 90_000.0,
    totalBytes: 1_073_741_824,
    errorCount: 42,
  };
}

function pbPerson(): Record<string, unknown> {
  return {
    id: 1001,
    firstName: 'John',
    lastName: 'Doe',
    middleName: 'Robert',
    dateOfBirth: { seconds: 1705900800, nanos: 123456789 },
    contact: {
      email: 'john.doe@example.com',
      phone: '+1-555-123-4567',
      mobile: '+1-555-987-6543',
      mailingAddress: {
        street1: '123 Main Street',
        street2: 'Suite 100',
        city: 'San Francisco',
        state: 'CA',
        postalCode: '94105',
        country: 'USA',
        coordinates: { x: 123.456, y: 789.012, z: 345.678 },
      },
    },
    status: 2, // STATUS_ACTIVE
    createdAt: { seconds: 1705900800, nanos: 123456789 },
    updatedAt: { seconds: 1705900800, nanos: 123456789 },
  };
}

function pbDocument(): Record<string, unknown> {
  return {
    id: 2001,
    title: 'Important Document Title',
    content:
      'This is the document content with some meaningful text that would typically be much longer in a real application.',
    authorId: 1001,
    status: 2,
    priority: 2,
    tags: [
      { key: 'category', value: 'technical' },
      { key: 'status', value: 'reviewed' },
      { key: 'version', value: '2.0' },
    ],
    attachments: [
      {
        id: 'att-001',
        filename: 'report.pdf',
        mimeType: 'application/pdf',
        sizeBytes: 1_048_576,
        checksum: new Uint8Array([0xde, 0xad, 0xbe, 0xef]),
        uploadedAt: { seconds: 1705900800, nanos: 123456789 },
      },
    ],
    comments: [
      {
        id: 3001,
        authorId: 1002,
        content: 'Great document!',
        createdAt: { seconds: 1705900800, nanos: 123456789 },
        reactions: [1001, 1003, 1004],
      },
    ],
    metadata: { source: 'import', encoding: 'utf-8', version: '1.0' },
    collaborators: [1001, 1002, 1003],
    createdAt: { seconds: 1705900800, nanos: 123456789 },
    updatedAt: { seconds: 1705900800, nanos: 123456789 },
    publishedAt: { seconds: 1705900800, nanos: 123456789 },
  };
}

function pbEvent(): Record<string, unknown> {
  return {
    id: 'evt-001',
    type: 0, // CREATED
    entityType: 'document',
    entityId: 'doc-2001',
    source: {
      service: 'document-service',
      instance: 'prod-01',
      version: '1.2.3',
      region: 'us-west-2',
    },
    timestamp: { seconds: 1705900800, nanos: 123456789 },
    attributes: { user_id: '1001', action: 'create' },
    payload: new TextEncoder().encode('{"action":"click","element":"button"}'),
    correlationId: 'corr-123',
    causationId: 'caus-456',
  };
}

function pbBatch(size: number): Record<string, unknown> {
  const items = new Array(size);
  for (let i = 0; i < size; i++) {
    items[i] = { id: i, name: 'batch-item', active: i % 2 === 0 };
  }
  return {
    requestId: 'batch-001',
    items,
    headers: { 'Content-Type': 'application/x-cramberry', 'X-Request-Id': 'req-123' },
    submittedAt: { seconds: 1705900800, nanos: 123456789 },
    priority: 1,
  };
}

// ---------------------------------------------------------------------------
// Bench scenario builder
// ---------------------------------------------------------------------------

interface Scenario {
  name: string;
  encodeCram: (w: Writer) => void;
  decodeCram: (r: Reader) => unknown;
  cramBytes: Uint8Array;
  pbType: protobuf.Type;
  pbMessage: protobuf.Message;
  pbBytes: Uint8Array;
  jsonMsg: unknown;
  jsonBytes: string;
}

function scenario<T>(opts: {
  name: string;
  cramMsg: T;
  encodeCram: (w: Writer, msg: T) => void;
  decodeCram: (r: Reader) => T;
  pbType: protobuf.Type;
  pbObj: Record<string, unknown>;
}): Scenario {
  const { name, cramMsg, encodeCram, decodeCram, pbType, pbObj } = opts;

  // Pre-compute reference bytes for the decode benches.
  const w = new Writer();
  encodeCram(w, cramMsg);
  const cramBytes = w.bytes().slice();

  const pbMessage = pbType.create(pbObj);
  const pbBytes = pbType.encode(pbMessage).finish();

  const jsonBytes = JSON.stringify(cramMsg, fx.jsonReplacer);

  return {
    name,
    encodeCram: (w) => encodeCram(w, cramMsg),
    decodeCram,
    cramBytes,
    pbType,
    pbMessage,
    pbBytes,
    jsonMsg: cramMsg,
    jsonBytes,
  };
}

const scenarios: Scenario[] = [
  scenario({
    name: 'SmallMessage',
    cramMsg: fx.smallMessage(),
    encodeCram: cg.encodeSmallMessage,
    decodeCram: cg.decodeSmallMessage,
    pbType: PB.SmallMessage,
    pbObj: pbSmall(),
  }),
  scenario({
    name: 'Metrics',
    cramMsg: fx.metrics(),
    encodeCram: cg.encodeMetrics,
    decodeCram: cg.decodeMetrics,
    pbType: PB.Metrics,
    pbObj: pbMetrics(),
  }),
  scenario({
    name: 'Person',
    cramMsg: fx.person(),
    encodeCram: cg.encodePerson,
    decodeCram: cg.decodePerson,
    pbType: PB.Person,
    pbObj: pbPerson(),
  }),
  scenario({
    name: 'Document',
    cramMsg: fx.document(),
    encodeCram: cg.encodeDocument,
    decodeCram: cg.decodeDocument,
    pbType: PB.Document,
    pbObj: pbDocument(),
  }),
  scenario({
    name: 'Event',
    cramMsg: fx.event(),
    encodeCram: cg.encodeEvent,
    decodeCram: cg.decodeEvent,
    pbType: PB.Event,
    pbObj: pbEvent(),
  }),
  scenario({
    name: 'Batch100',
    cramMsg: fx.batchRequest(100),
    encodeCram: cg.encodeBatchRequest,
    decodeCram: cg.decodeBatchRequest,
    pbType: PB.BatchRequest,
    pbObj: pbBatch(100),
  }),
  scenario({
    name: 'Batch1000',
    cramMsg: fx.batchRequest(1000),
    encodeCram: cg.encodeBatchRequest,
    decodeCram: cg.decodeBatchRequest,
    pbType: PB.BatchRequest,
    pbObj: pbBatch(1000),
  }),
];

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

const filter = process.argv[2];

const bench = new Bench({ time: 1500, warmupTime: 250 });

for (const sc of scenarios) {
  if (filter && !sc.name.toLowerCase().includes(filter.toLowerCase())) continue;

  const writer = new Writer();
  bench.add(`${sc.name}/cramberry/encode`, () => {
    writer.reset();
    sc.encodeCram(writer);
    writer.bytes();
  });

  bench.add(`${sc.name}/cramberry/decode`, () => {
    const r = new Reader(sc.cramBytes);
    sc.decodeCram(r);
  });

  bench.add(`${sc.name}/protobuf/encode`, () => {
    sc.pbType.encode(sc.pbMessage).finish();
  });

  bench.add(`${sc.name}/protobuf/decode`, () => {
    sc.pbType.decode(sc.pbBytes);
  });

  bench.add(`${sc.name}/json/encode`, () => {
    JSON.stringify(sc.jsonMsg, fx.jsonReplacer);
  });

  bench.add(`${sc.name}/json/decode`, () => {
    JSON.parse(sc.jsonBytes);
  });
}

await bench.run();

// Print results in a comparable shape to Go's `go test -bench` output.
// tinybench reports `mean` in milliseconds; convert to ns/op and ops/sec.
const rows = bench.tasks.map((t) => {
  const r = t.result;
  if (!r) return null;
  const meanMs = r.mean;
  return {
    name: t.name,
    'ns/op': (meanMs * 1_000_000).toFixed(0),
    'ops/sec': (1_000 / meanMs).toFixed(0),
    samples: r.samples.length,
    error: r.error?.message ?? '',
  };
});
console.table(rows.filter((row): row is NonNullable<typeof row> => row !== null));

// Print encoded sizes — useful at-a-glance comparison alongside latency.
console.log('\nEncoded sizes (bytes):');
for (const sc of scenarios) {
  if (filter && !sc.name.toLowerCase().includes(filter.toLowerCase())) continue;
  console.log(
    `  ${sc.name.padEnd(13)} cramberry=${String(sc.cramBytes.length).padStart(6)}  protobuf=${String(sc.pbBytes.length).padStart(6)}  json=${String(sc.jsonBytes.length).padStart(6)}`,
  );
}
