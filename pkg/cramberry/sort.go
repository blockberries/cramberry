package cramberry

import (
	"slices"
	"sync"
)

// orderedMapKey constrains map-key types whose natural `<` operator yields
// the cramberry-canonical sort order. Floats and bools need custom orderings
// and are handled by SortedMapKeysFloat32/Float64/Bool.
type orderedMapKey interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// SortedMapKeys returns the keys of m sorted with the cramberry-canonical
// ordering for K. Generated EncodeTo methods call this so that on-wire map
// output is byte-identical regardless of the runtime's map iteration order.
//
// For string keys the comparison is by raw byte (UTF-8) order; for integer
// keys it is numeric order.
//
// Codegen-only API. The function is exported so emitted code can reach it,
// but user code should rely on cramberry.Marshal or generated EncodeTo
// methods rather than calling this directly. The signature and ordering
// rules may change without a major version bump.
func SortedMapKeys[K orderedMapKey, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// ----------------------------------------------------------------------------
// Pooled sorted-keys helpers used by codegen
//
// Per-encode allocation of the keys slice was the dominant residual
// alloc for codegen Marshal output. Generic sync.Pool is awkward to
// type-parameterize, so we provide specialized helpers for the common
// key types (string, int64, uint64). Callers must call PutStringKeys /
// PutInt64Keys / PutUint64Keys with the slice once they're done; not
// returning is correct (it'll just GC) but loses the pooling benefit.
//
// Slices larger than maxPooledKeySliceSize are not returned to the
// pool to avoid pinning huge buffers.
// ----------------------------------------------------------------------------

const maxPooledKeySliceSize = 1 << 16

var stringKeyPool = sync.Pool{New: func() any { s := make([]string, 0, 16); return &s }}
var int64KeyPool = sync.Pool{New: func() any { s := make([]int64, 0, 16); return &s }}
var uint64KeyPool = sync.Pool{New: func() any { s := make([]uint64, 0, 16); return &s }}

// GetSortedStringKeys returns a sorted []string of m's keys. The
// returned slice must be released with PutStringKeys when done.
func GetSortedStringKeys[V any](m map[string]V) []string {
	p := stringKeyPool.Get().(*[]string)
	keys := (*p)[:0]
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	*p = keys
	return keys
}

// PutStringKeys returns a slice acquired from GetSortedStringKeys to the pool.
func PutStringKeys(keys []string) {
	if cap(keys) > maxPooledKeySliceSize {
		return
	}
	keys = keys[:0]
	stringKeyPool.Put(&keys)
}

// GetSortedInt64Keys / PutInt64Keys mirror the string variant for int64-keyed maps.
func GetSortedInt64Keys[V any](m map[int64]V) []int64 {
	p := int64KeyPool.Get().(*[]int64)
	keys := (*p)[:0]
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	*p = keys
	return keys
}

// PutInt64Keys returns a slice to the int64 pool.
func PutInt64Keys(keys []int64) {
	if cap(keys) > maxPooledKeySliceSize {
		return
	}
	keys = keys[:0]
	int64KeyPool.Put(&keys)
}

// GetSortedUint64Keys / PutUint64Keys mirror the string variant for uint64-keyed maps.
func GetSortedUint64Keys[V any](m map[uint64]V) []uint64 {
	p := uint64KeyPool.Get().(*[]uint64)
	keys := (*p)[:0]
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	*p = keys
	return keys
}

// PutUint64Keys returns a slice to the uint64 pool.
func PutUint64Keys(keys []uint64) {
	if cap(keys) > maxPooledKeySliceSize {
		return
	}
	keys = keys[:0]
	uint64KeyPool.Put(&keys)
}

// Bool / float32 / float64 map-key types are rejected at the schema
// validator (pkg/schema/validator.go::validateMapKeyType), so the codegen
// path never emits calls to dedicated SortedMapKeys helpers for them. The
// reflection marshaller handles its own ordering inline via
// CompareFloatKeys / CompareFloat32Keys (pkg/cramberry/marshal.go).
