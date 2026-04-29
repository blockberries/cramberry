package cramberry

import (
	"sort"
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
// but user code should rely on cramberry.Marshal (with Deterministic=true)
// or generated EncodeTo methods rather than calling this directly. The
// signature and ordering rules may change without a major version bump.
func SortedMapKeys[K orderedMapKey, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// SortedMapKeysFloat32 returns the keys of a float32-keyed map sorted with
// the same total order as the reflection marshaller (NaN sorts last; -0.0
// equals +0.0; differing NaN payloads compare by raw bits).
func SortedMapKeysFloat32[V any](m map[float32]V) []float32 {
	keys := make([]float32, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return CompareFloat32Keys(keys[i], keys[j]) })
	return keys
}

// SortedMapKeysFloat64 is the float64 analogue of SortedMapKeysFloat32.
func SortedMapKeysFloat64[V any](m map[float64]V) []float64 {
	keys := make([]float64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return CompareFloatKeys(keys[i], keys[j]) })
	return keys
}

// SortedMapKeysBool returns the keys of a bool-keyed map with `false` sorted
// before `true`. A bool-keyed map has at most two entries, but the helper
// exists so generated code can dispatch on key type uniformly.
func SortedMapKeysBool[V any](m map[bool]V) []bool {
	keys := make([]bool, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return !keys[i] && keys[j] })
	return keys
}
