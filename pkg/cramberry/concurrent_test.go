package cramberry

import (
	"reflect"
	"sync"
	"testing"
)

// Concurrent access tests for REFACTORING item 33

// ConcurrentStruct is used for concurrent marshal/unmarshal tests.
type ConcurrentStruct struct {
	ID      int64             `cramberry:"1"`
	Name    string            `cramberry:"2"`
	Values  []int32           `cramberry:"3"`
	Mapping map[string]string `cramberry:"4"`
}

// TestConcurrentMarshal tests concurrent Marshal calls with the same type.
func TestConcurrentMarshal(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				data := ConcurrentStruct{
					ID:      int64(id*iterations + i),
					Name:    "test",
					Values:  []int32{1, 2, 3},
					Mapping: map[string]string{"key": "value"},
				}
				_, err := Marshal(data)
				if err != nil {
					errors <- err
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Marshal error: %v", err)
	}
}

// TestConcurrentUnmarshal tests concurrent Unmarshal calls.
func TestConcurrentUnmarshal(t *testing.T) {
	// First, create some encoded data to unmarshal
	original := ConcurrentStruct{
		ID:      12345,
		Name:    "concurrent test",
		Values:  []int32{10, 20, 30, 40, 50},
		Mapping: map[string]string{"a": "1", "b": "2", "c": "3"},
	}
	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)
	mismatches := make(chan string, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				var result ConcurrentStruct
				if err := Unmarshal(data, &result); err != nil {
					errors <- err
					continue
				}
				if result.ID != original.ID {
					mismatches <- "ID mismatch"
				}
				if result.Name != original.Name {
					mismatches <- "Name mismatch"
				}
			}
		}()
	}

	wg.Wait()
	close(errors)
	close(mismatches)

	for err := range errors {
		t.Errorf("Unmarshal error: %v", err)
	}
	for msg := range mismatches {
		t.Errorf("Data mismatch: %s", msg)
	}
}

// TestConcurrentMarshalUnmarshal tests concurrent Marshal and Unmarshal calls.
func TestConcurrentMarshalUnmarshal(t *testing.T) {
	const goroutines = 50

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*2)

	// Half the goroutines marshal
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				data := ConcurrentStruct{
					ID:     int64(id*100 + i),
					Name:   "marshal",
					Values: []int32{int32(i)},
				}
				encoded, err := Marshal(data)
				if err != nil {
					errors <- err
					continue
				}

				// Immediately unmarshal
				var result ConcurrentStruct
				if err := Unmarshal(encoded, &result); err != nil {
					errors <- err
					continue
				}
				if result.ID != data.ID {
					errors <- NewDecodeError("ID mismatch", nil)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Error: %v", err)
	}
}

// TestConcurrentRegistryAccess tests concurrent access to the type registry.
func TestConcurrentRegistryAccess(t *testing.T) {
	const goroutines = 100

	r := NewRegistry()

	// Define some test types
	type TestType1 struct{ Value int }
	type TestType2 struct{ Name string }
	type TestType3 struct{ Data []byte }

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*3)

	// Concurrent registrations
	for g := 0; g < goroutines; g++ {
		wg.Add(3)

		// Register different types concurrently
		go func() {
			defer wg.Done()
			_, err := r.RegisterType(reflect.TypeOf(TestType1{}))
			if err != nil && err.Error() != "type already registered" {
				// Ignore "already registered" errors from concurrent registration
				if _, ok := err.(*RegistrationError); !ok {
					errors <- err
				}
			}
		}()

		go func() {
			defer wg.Done()
			_, err := r.RegisterType(reflect.TypeOf(TestType2{}))
			if err != nil && err.Error() != "type already registered" {
				if _, ok := err.(*RegistrationError); !ok {
					errors <- err
				}
			}
		}()

		go func() {
			defer wg.Done()
			_, err := r.RegisterType(reflect.TypeOf(TestType3{}))
			if err != nil && err.Error() != "type already registered" {
				if _, ok := err.(*RegistrationError); !ok {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Registry error: %v", err)
	}

	// Verify all types are registered
	if r.Size() != 3 {
		t.Errorf("Expected 3 registered types, got %d", r.Size())
	}
}

// TestConcurrentRegistryLookup tests concurrent lookups in the registry.
func TestConcurrentRegistryLookup(t *testing.T) {
	r := NewRegistry()

	type LookupType struct{ Value int }
	id, err := r.RegisterType(reflect.TypeOf(LookupType{}))
	if err != nil {
		t.Fatalf("RegisterType error: %v", err)
	}

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				// Lookup by ID
				reg, ok := r.Lookup(id)
				if !ok || reg == nil {
					errors <- NewDecodeError("Lookup by ID failed", nil)
				}

				// Lookup by type
				reg, ok = r.LookupType(reflect.TypeOf(LookupType{}))
				if !ok || reg == nil {
					errors <- NewDecodeError("Lookup by type failed", nil)
				}

				// TypeIDFor
				typeID := r.TypeIDFor(LookupType{Value: i})
				if typeID != id {
					errors <- NewDecodeError("TypeIDFor mismatch", nil)
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Lookup error: %v", err)
	}
}

// TestConcurrentWriterPool tests concurrent access to the Writer pool.
func TestConcurrentWriterPool(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				w := GetWriter()
				w.WriteString("test data")
				w.WriteInt32(int32(id*iterations + i))
				data := w.BytesCopy()
				PutWriter(w)

				// Verify the data is valid
				if len(data) == 0 {
					t.Error("Empty data from pooled writer")
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestConcurrentReaderUsage tests concurrent Reader creation and usage.
func TestConcurrentReaderUsage(t *testing.T) {
	// Create test data
	w := GetWriter()
	w.WriteString("test string")
	w.WriteInt32(12345)
	testData := w.BytesCopy()
	PutWriter(w)

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r := NewReader(testData)
				s := r.ReadString()
				if r.Err() != nil {
					errors <- r.Err()
					continue
				}
				if s != "test string" {
					errors <- NewDecodeError("string mismatch", nil)
				}
				n := r.ReadInt32()
				if r.Err() != nil {
					errors <- r.Err()
					continue
				}
				if n != 12345 {
					errors <- NewDecodeError("int32 mismatch", nil)
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Reader error: %v", err)
	}
}

// TestConcurrentStructInfoCache tests concurrent access to struct info cache.
func TestConcurrentStructInfoCache(t *testing.T) {
	// Define multiple struct types to stress the cache
	type CacheTest1 struct {
		A int32  `cramberry:"1"`
		B string `cramberry:"2"`
	}
	type CacheTest2 struct {
		X float64 `cramberry:"1"`
		Y []byte  `cramberry:"2"`
	}
	type CacheTest3 struct {
		M map[string]int32 `cramberry:"1"`
	}

	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations*3)

	for g := 0; g < goroutines; g++ {
		wg.Add(3)

		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				data := CacheTest1{A: int32(id*iterations + i), B: "test"}
				encoded, err := Marshal(data)
				if err != nil {
					errors <- err
					continue
				}
				var result CacheTest1
				if err := Unmarshal(encoded, &result); err != nil {
					errors <- err
				}
			}
		}(g)

		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				data := CacheTest2{X: float64(id*iterations + i), Y: []byte{1, 2, 3}}
				encoded, err := Marshal(data)
				if err != nil {
					errors <- err
					continue
				}
				var result CacheTest2
				if err := Unmarshal(encoded, &result); err != nil {
					errors <- err
				}
			}
		}(g)

		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				data := CacheTest3{M: map[string]int32{"k": int32(id*iterations + i)}}
				encoded, err := Marshal(data)
				if err != nil {
					errors <- err
					continue
				}
				var result CacheTest3
				if err := Unmarshal(encoded, &result); err != nil {
					errors <- err
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Cache error: %v", err)
	}
}
