// Package cramberry — internal pool exports.
//
// This file used to expose `GetBuffer` / `PutBuffer` / `BufferPoolStats`
// for size-tiered byte-slice pooling, but nothing in the codebase ever
// called them and the pool sizing rules made them error-prone (a
// returned buffer's capacity could be smaller than the requested size
// hint after one round-trip). They were removed; the file is kept so a
// future intentional pool helper has an obvious home.
package cramberry
