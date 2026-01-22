package cramberry

import (
	"bytes"
	"encoding/json"
	"testing"
)

// Benchmark types
type BenchSmall struct {
	ID   int32  `cramberry:"1" json:"id"`
	Name string `cramberry:"2" json:"name"`
}

type BenchMedium struct {
	ID     int64    `cramberry:"1" json:"id"`
	Name   string   `cramberry:"2" json:"name"`
	Email  string   `cramberry:"3" json:"email"`
	Active bool     `cramberry:"4" json:"active"`
	Score  float64  `cramberry:"5" json:"score"`
	Tags   []string `cramberry:"6" json:"tags"`
}

type BenchLarge struct {
	ID          int64             `cramberry:"1" json:"id"`
	Name        string            `cramberry:"2" json:"name"`
	Email       string            `cramberry:"3" json:"email"`
	Active      bool              `cramberry:"4" json:"active"`
	Score       float64           `cramberry:"5" json:"score"`
	Tags        []string          `cramberry:"6" json:"tags"`
	Metadata    map[string]string `cramberry:"7" json:"metadata"`
	Nested      *BenchMedium      `cramberry:"8" json:"nested"`
	Numbers     []int32           `cramberry:"9" json:"numbers"`
	Description string            `cramberry:"10" json:"description"`
}

type BenchNested struct {
	Level1 *BenchNestedLevel1 `cramberry:"1" json:"level1"`
}

type BenchNestedLevel1 struct {
	Level2 *BenchNestedLevel2 `cramberry:"1" json:"level2"`
	Value  string             `cramberry:"2" json:"value"`
}

type BenchNestedLevel2 struct {
	Level3 *BenchNestedLevel3 `cramberry:"1" json:"level3"`
	Value  string             `cramberry:"2" json:"value"`
}

type BenchNestedLevel3 struct {
	Value string `cramberry:"1" json:"value"`
}

var (
	benchSmall = BenchSmall{
		ID:   42,
		Name: "benchmark",
	}

	benchMedium = BenchMedium{
		ID:     12345678,
		Name:   "Test User",
		Email:  "test@example.com",
		Active: true,
		Score:  95.5,
		Tags:   []string{"tag1", "tag2", "tag3"},
	}

	benchLarge = BenchLarge{
		ID:          9876543210,
		Name:        "Complex User",
		Email:       "complex@example.com",
		Active:      true,
		Score:       87.3,
		Tags:        []string{"golang", "rust", "typescript", "performance"},
		Metadata:    map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
		Nested:      &benchMedium,
		Numbers:     []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		Description: "This is a longer description field to test string encoding performance with medium-length text content.",
	}

	benchNested = BenchNested{
		Level1: &BenchNestedLevel1{
			Level2: &BenchNestedLevel2{
				Level3: &BenchNestedLevel3{
					Value: "deep",
				},
				Value: "level3",
			},
			Value: "level2",
		},
	}
)

// Marshal benchmarks
func BenchmarkMarshalSmall(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(benchSmall)
	}
}

func BenchmarkMarshalMedium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(benchMedium)
	}
}

func BenchmarkMarshalLarge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(benchLarge)
	}
}

func BenchmarkMarshalNested(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(benchNested)
	}
}

// Unmarshal benchmarks
func BenchmarkUnmarshalSmall(b *testing.B) {
	data, _ := Marshal(benchSmall)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchSmall
		_ = Unmarshal(data, &result)
	}
}

func BenchmarkUnmarshalMedium(b *testing.B) {
	data, _ := Marshal(benchMedium)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchMedium
		_ = Unmarshal(data, &result)
	}
}

func BenchmarkUnmarshalLarge(b *testing.B) {
	data, _ := Marshal(benchLarge)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchLarge
		_ = Unmarshal(data, &result)
	}
}

func BenchmarkUnmarshalNested(b *testing.B) {
	data, _ := Marshal(benchNested)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchNested
		_ = Unmarshal(data, &result)
	}
}

// Writer pool benchmarks
func BenchmarkMarshalWithPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := GetWriter()
		// Manually encode using writer
		w.WriteUvarint(2) // field count
		w.WriteTag(1, WireSVarint)
		w.WriteSvarint(int64(benchSmall.ID))
		w.WriteTag(2, WireBytes)
		w.WriteString(benchSmall.Name)
		_ = w.BytesCopy()
		PutWriter(w)
	}
}

// Slice benchmarks
func BenchmarkMarshalInt32Slice(b *testing.B) {
	slice := make([]int32, 100)
	for i := range slice {
		slice[i] = int32(i * 2)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(slice)
	}
}

func BenchmarkUnmarshalInt32Slice(b *testing.B) {
	slice := make([]int32, 100)
	for i := range slice {
		slice[i] = int32(i * 2)
	}
	data, _ := Marshal(slice)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result []int32
		_ = Unmarshal(data, &result)
	}
}

func BenchmarkMarshalStringSlice(b *testing.B) {
	slice := make([]string, 50)
	for i := range slice {
		slice[i] = "string number " + string(rune('0'+i%10))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(slice)
	}
}

// Map benchmarks
func BenchmarkMarshalMap(b *testing.B) {
	m := make(map[string]int32, 20)
	for i := 0; i < 20; i++ {
		m["key"+string(rune('a'+i))] = int32(i * 10)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(m)
	}
}

func BenchmarkUnmarshalMap(b *testing.B) {
	m := make(map[string]int32, 20)
	for i := 0; i < 20; i++ {
		m["key"+string(rune('a'+i))] = int32(i * 10)
	}
	data, _ := Marshal(m)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]int32
		_ = Unmarshal(data, &result)
	}
}

// Comparison with JSON
func BenchmarkJSONMarshalSmall(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(benchSmall)
	}
}

func BenchmarkJSONUnmarshalSmall(b *testing.B) {
	data, _ := json.Marshal(benchSmall)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchSmall
		_ = json.Unmarshal(data, &result)
	}
}

func BenchmarkJSONMarshalMedium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(benchMedium)
	}
}

func BenchmarkJSONUnmarshalMedium(b *testing.B) {
	data, _ := json.Marshal(benchMedium)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchMedium
		_ = json.Unmarshal(data, &result)
	}
}

func BenchmarkJSONMarshalLarge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(benchLarge)
	}
}

func BenchmarkJSONUnmarshalLarge(b *testing.B) {
	data, _ := json.Marshal(benchLarge)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchLarge
		_ = json.Unmarshal(data, &result)
	}
}

// Streaming benchmarks
func BenchmarkStreamWriteMedium(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		sw := NewStreamWriter(&buf)
		sw.WriteDelimited(&benchMedium)
		sw.Flush()
	}
}

func BenchmarkStreamReadMedium(b *testing.B) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	sw.WriteDelimited(&benchMedium)
	sw.Flush()
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sr := NewStreamReader(bytes.NewReader(data))
		var result BenchMedium
		sr.ReadDelimited(&result)
	}
}

func BenchmarkStreamWriteMultiple(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		sw := NewStreamWriter(&buf)
		for j := 0; j < 10; j++ {
			sw.WriteDelimited(&benchSmall)
		}
		sw.Flush()
	}
}

func BenchmarkStreamReadMultiple(b *testing.B) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	for j := 0; j < 10; j++ {
		sw.WriteDelimited(&benchSmall)
	}
	sw.Flush()
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sr := NewStreamReader(bytes.NewReader(data))
		for j := 0; j < 10; j++ {
			var result BenchSmall
			sr.ReadDelimited(&result)
		}
	}
}

// Size calculation benchmarks
func BenchmarkSize(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Size(benchSmall)
		}
	})

	b.Run("Medium", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Size(benchMedium)
		}
	})

	b.Run("Large", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Size(benchLarge)
		}
	})
}

// Comparison summary: Print size comparison
func TestEncodingSizeComparison(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"Small", benchSmall},
		{"Medium", benchMedium},
		{"Large", benchLarge},
		{"Nested", benchNested},
	}

	for _, tc := range tests {
		cramberryData, _ := Marshal(tc.value)
		jsonData, _ := json.Marshal(tc.value)

		t.Logf("%s: Cramberry=%d bytes, JSON=%d bytes (%.1f%% smaller)",
			tc.name, len(cramberryData), len(jsonData),
			100*(1-float64(len(cramberryData))/float64(len(jsonData))))
	}
}

// Primitive encoding benchmarks
func BenchmarkWriteInt32(b *testing.B) {
	w := NewWriter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Reset()
		w.WriteInt32(int32(i))
	}
}

func BenchmarkWriteString(b *testing.B) {
	s := "this is a test string for benchmarking"
	w := NewWriter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Reset()
		w.WriteString(s)
	}
}

func BenchmarkReadInt32(b *testing.B) {
	w := NewWriter()
	w.WriteSvarint(12345)
	data := w.BytesCopy()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(data)
		r.ReadInt32()
	}
}

func BenchmarkReadString(b *testing.B) {
	w := NewWriter()
	w.WriteString("this is a test string for benchmarking")
	data := w.BytesCopy()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(data)
		r.ReadString()
	}
}
