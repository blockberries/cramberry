package cramberry

import (
	"slices"
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

// Bool / float32 / float64 map-key types are rejected at the schema
// validator (pkg/schema/validator.go::validateMapKeyType), so the codegen
// path never emits calls to dedicated SortedMapKeys helpers for them. The
// reflection marshaller handles its own ordering inline via
// CompareFloatKeys / CompareFloat32Keys (pkg/cramberry/marshal.go).
