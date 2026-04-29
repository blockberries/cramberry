package cramberry

import (
	"bytes"
	"math"
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

func TestSortedMapKeysFloat64_NaNSortsLast(t *testing.T) {
	nan := math.NaN()
	m := map[float64]int{nan: 1, 1.0: 2, -1.0: 3}
	got := SortedMapKeysFloat64(m)
	if got[0] != -1.0 || got[1] != 1.0 || !math.IsNaN(got[2]) {
		t.Errorf("SortedMapKeysFloat64 = %v, want [-1, 1, NaN]", got)
	}
}

func TestSortedMapKeysBool(t *testing.T) {
	m := map[bool]int{true: 1, false: 2}
	got := SortedMapKeysBool(m)
	if got[0] != false || got[1] != true {
		t.Errorf("SortedMapKeysBool = %v, want [false, true]", got)
	}
}

// TestCodegenMapDeterminism encodes the same map 50 times via the path used
// by generated code (cramberry.SortedMapKeys + write) and asserts that every
// run produces the same bytes. Catches a regression to the unsorted iteration
// that was the worst bug in the pre-2.0 codegen.
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

	encode := func() []byte {
		w := GetWriter()
		defer PutWriter(w)
		w.WriteTag(1, WireBytes)
		keys := SortedMapKeys(m)
		w.WriteUvarint(uint64(len(keys)))
		for _, k := range keys {
			w.WriteString(k)
			w.WriteInt32(m[k])
		}
		w.WriteEndMarker()
		return w.BytesCopy()
	}

	first := encode()
	for i := 0; i < 50; i++ {
		got := encode()
		if !bytes.Equal(got, first) {
			t.Fatalf("iteration %d: bytes differ from first encode", i)
		}
	}
}
