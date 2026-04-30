package cramberry

import (
	"bytes"
	"sort"
	"testing"
)

func TestSortedMapKeys_Strings_AreUTF8Bytes(t *testing.T) {
	// 'a' (0x61) < 'z' (0x7A) < non-BMP CJK 你 (0xE4 0xBD 0xA0 in UTF-8).
	// In UTF-16 code-unit order, 你 (0x4F60) would sort before 'a' — so this
	// test catches a regression to that bug.
	m := map[string]int{
		"你": 1, // 你
		"a": 2,
		"z": 3,
	}
	got := SortedMapKeys(m)
	want := []string{"a", "z", "你"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortedMapKeys[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSortedMapKeys_Integers_AreNumeric(t *testing.T) {
	m := map[int32]string{10: "a", 2: "b", 1: "c"}
	got := SortedMapKeys(m)
	want := []int32{1, 2, 10}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortedMapKeys[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

// TestCodegenMapDeterminism asserts that:
//  1. SortedMapKeys returns keys in lexicographic UTF-8 byte order
//     (not in any incidental Go map iteration order).
//  2. Encoding through the codegen path is byte-stable across runs.
//
// The earlier version of this test only ran point (2) — but it called
// SortedMapKeys *itself* inside the encode helper, so removing the
// sort entirely would still produce some output (just unsorted) that
// happened to be stable for any single run. The test would have
// passed for the wrong reason. Now we explicitly check the keys are
// sorted, and we additionally compare codegen output to a baseline
// that uses the raw (unsorted) iteration order to confirm the sort
// is actually doing work.
func TestCodegenMapDeterminism(t *testing.T) {
	m := map[string]int32{
		"alpha":   1,
		"beta":    2,
		"gamma":   3,
		"delta":   4,
		"epsilon": 5,
		"zeta":    6,
		"eta":     7,
	}

	// (1) SortedMapKeys must return keys in lexicographic order.
	keys := SortedMapKeys(m)
	if !sort.StringsAreSorted(keys) {
		t.Fatalf("SortedMapKeys returned non-sorted keys: %v", keys)
	}

	// (2) Encoding via the codegen path is byte-stable.
	encode := func() []byte {
		w := GetWriter()
		defer PutWriter(w)
		w.WriteTag(1, WireBytes)
		ks := SortedMapKeys(m)
		w.WriteUvarint(uint64(len(ks)))
		for _, k := range ks {
			w.WriteString(k)
			w.WriteInt32(m[k])
		}
		w.WriteEndMarker()
		return w.BytesCopy()
	}

	first := encode()
	for i := range 50 {
		got := encode()
		if !bytes.Equal(got, first) {
			t.Fatalf("iteration %d: bytes differ from first encode", i)
		}
	}

	// (3) Spot-check: an UNSORTED encode of the same map must produce
	//     a different byte stream than the sorted encode for at least
	//     one Go-runtime random seed. If they're equal here, sort is a
	//     no-op (e.g., map small enough to iterate in order on this
	//     run); try a few seeds. With 7 keys, P(all 100 iterations
	//     equal sorted) is astronomically small if the sort is real.
	encodeUnsorted := func() []byte {
		w := GetWriter()
		defer PutWriter(w)
		w.WriteTag(1, WireBytes)
		w.WriteUvarint(uint64(len(m)))
		for k, v := range m { // raw iteration; deliberately unsorted
			w.WriteString(k)
			w.WriteInt32(v)
		}
		w.WriteEndMarker()
		return w.BytesCopy()
	}
	differed := false
	for range 100 {
		if !bytes.Equal(first, encodeUnsorted()) {
			differed = true
			break
		}
	}
	if !differed {
		t.Fatal("unsorted iteration matched sorted iteration in 100 runs — " +
			"either SortedMapKeys is a no-op or Go's map iteration is no " +
			"longer randomized; the determinism guarantee is at risk")
	}
}
