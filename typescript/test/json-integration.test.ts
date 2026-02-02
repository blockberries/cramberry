import { describe, it, expect } from 'vitest';
import {
  Status,
  ScalarTypes,
  EnumTest,
  toJSON_ScalarTypes,
  fromJSON_ScalarTypes,
  toJSON_EnumTest,
  fromJSON_EnumTest,
} from './json_test';

describe('TypeScript JSON Integration', () => {
  it('serializes scalar types to JSON', () => {
    const msg: ScalarTypes = {
      boolVal: true,
      int8Val: -42,
      int16Val: -1000,
      int32Val: -100000,
      int64Val: -9223372036854775807n,
      uint8Val: 255,
      uint16Val: 65535,
      uint32Val: 4294967295,
      uint64Val: 18446744073709551615n,
      float32Val: 3.14159,
      float64Val: 2.718281828459045,
      stringVal: 'hello, world!',
      bytesVal: new Uint8Array([0xde, 0xad, 0xbe, 0xef]),
    };

    const json = toJSON_ScalarTypes(msg);
    console.log('JSON output:', json);

    // Verify it's compact (no whitespace)
    expect(json).not.toMatch(/\s/);

    // Verify integers are strings
    expect(json).toContain('"int64_val":"-9223372036854775807"');
    expect(json).toContain('"uint64_val":"18446744073709551615"');

    // Verify round-trip
    const decoded = fromJSON_ScalarTypes(json);
    const json2 = toJSON_ScalarTypes(decoded);
    expect(json2).toBe(json);
  });

  it('serializes enums as string names', () => {
    const msg: EnumTest = {
      status: Status.Active,
      statuses: [Status.Active, Status.Pending, Status.Inactive],
    };

    const json = toJSON_EnumTest(msg);
    console.log('Enum JSON:', json);

    // Verify enums are string names
    expect(json).toContain('"status":"ACTIVE"');
    expect(json).toContain('[Active","PENDING","INACTIVE"]');

    // Round-trip
    const decoded = fromJSON_EnumTest(json);
    expect(decoded.status).toBe(Status.Active);
    expect(decoded.statuses).toEqual([Status.Active, Status.Pending, Status.Inactive]);
  });
});
