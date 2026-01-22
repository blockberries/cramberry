package cramberry

import (
	"reflect"
	"sync"
	"testing"
)

// Test types for registry
type Person struct {
	Name string
	Age  int
}

type Animal struct {
	Species string
}

type Flyer interface {
	Fly()
}

type Bird struct {
	Wingspan float64
}

func (b *Bird) Fly() {}

type Plane struct {
	Model string
}

func (p *Plane) Fly() {}

func TestRegistryBasic(t *testing.T) {
	r := NewRegistry()

	// Register a type
	id, err := r.RegisterType(reflect.TypeOf(Person{}))
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if id < TypeIDUserStart {
		t.Errorf("ID = %d, should be >= %d", id, TypeIDUserStart)
	}

	// Should be able to look up
	reg, ok := r.LookupType(reflect.TypeOf(Person{}))
	if !ok {
		t.Fatal("LookupType failed")
	}
	if reg.ID != id {
		t.Errorf("ID = %d, should be %d", reg.ID, id)
	}

	// Look up by ID
	reg2, ok := r.Lookup(reg.ID)
	if !ok || reg2 != reg {
		t.Error("Lookup by ID failed")
	}

	// Look up by name
	reg3, ok := r.LookupName(reg.Name)
	if !ok || reg3 != reg {
		t.Error("LookupName failed")
	}
}

func TestRegistryWithID(t *testing.T) {
	r := NewRegistry()

	// Register with specific ID
	id := TypeID(200)
	err := r.RegisterTypeWithID(reflect.TypeOf(Animal{}), id)
	if err != nil {
		t.Fatalf("RegisterWithID error: %v", err)
	}

	reg, ok := r.Lookup(id)
	if !ok {
		t.Fatal("Lookup failed")
	}
	if reg.ID != id {
		t.Errorf("ID = %d, want %d", reg.ID, id)
	}
}

func TestRegistryDuplicateType(t *testing.T) {
	r := NewRegistry()

	_, err := r.RegisterType(reflect.TypeOf(Person{}))
	if err != nil {
		t.Fatalf("First Register error: %v", err)
	}

	// Registering same type again with same ID should succeed
	reg, _ := r.LookupType(reflect.TypeOf(Person{}))
	err = r.RegisterTypeWithID(reflect.TypeOf(Person{}), reg.ID)
	if err != nil {
		t.Errorf("Re-registering with same ID should succeed: %v", err)
	}

	// Registering same type with different ID should fail
	err = r.RegisterTypeWithID(reflect.TypeOf(Person{}), TypeID(999))
	if err == nil {
		t.Error("Registering same type with different ID should fail")
	}
}

func TestRegistryDuplicateID(t *testing.T) {
	r := NewRegistry()

	id := TypeID(200)
	err := r.RegisterTypeWithID(reflect.TypeOf(Person{}), id)
	if err != nil {
		t.Fatalf("First RegisterWithID error: %v", err)
	}

	// Using same ID for different type should fail
	err = r.RegisterTypeWithID(reflect.TypeOf(Animal{}), id)
	if err == nil {
		t.Error("Using same ID for different type should fail")
	}
}

func TestRegistryTypeIDFor(t *testing.T) {
	r := NewRegistry()

	_, err := r.RegisterType(reflect.TypeOf(Person{}))
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	// Get ID for registered type
	p := Person{Name: "Alice", Age: 30}
	id := r.TypeIDFor(p)
	if !id.IsUser() {
		t.Errorf("TypeIDFor = %d, expected user range", id)
	}

	// Get ID for pointer
	id2 := r.TypeIDFor(&p)
	if id2 != id {
		t.Errorf("TypeIDFor pointer = %d, want %d", id2, id)
	}

	// Nil returns TypeIDNil
	if r.TypeIDFor(nil) != TypeIDNil {
		t.Error("TypeIDFor(nil) should return TypeIDNil")
	}

	// Nil pointer returns TypeIDNil
	var pNil *Person
	if r.TypeIDFor(pNil) != TypeIDNil {
		t.Error("TypeIDFor(nil pointer) should return TypeIDNil")
	}

	// Unregistered type returns TypeIDNil
	if r.TypeIDFor(Animal{}) != TypeIDNil {
		t.Error("TypeIDFor(unregistered) should return TypeIDNil")
	}
}

func TestRegistryNewValue(t *testing.T) {
	r := NewRegistry()

	id, err := r.RegisterType(reflect.TypeOf(Person{}))
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	// Create new value
	v, ok := r.NewValue(id)
	if !ok {
		t.Fatal("NewValue failed")
	}

	p, ok := v.(*Person)
	if !ok {
		t.Fatalf("NewValue returned %T, want *Person", v)
	}
	if p == nil {
		t.Error("NewValue returned nil")
	}

	// Unknown ID returns nil
	_, ok = r.NewValue(TypeID(9999))
	if ok {
		t.Error("NewValue for unknown ID should return false")
	}
}

func TestRegistryInterface(t *testing.T) {
	r := NewRegistry()

	// Register interface
	flyerType := reflect.TypeOf((*Flyer)(nil)).Elem()
	err := r.RegisterInterfaceType(flyerType)
	if err != nil {
		t.Fatalf("RegisterInterface error: %v", err)
	}

	// Register implementing types
	_, err = r.RegisterType(reflect.TypeOf(Bird{}))
	if err != nil {
		t.Fatalf("Register Bird error: %v", err)
	}
	_, err = r.RegisterType(reflect.TypeOf(Plane{}))
	if err != nil {
		t.Fatalf("Register Plane error: %v", err)
	}

	// Register implementations
	err = r.RegisterImplementation(flyerType, reflect.TypeOf(Bird{}))
	if err != nil {
		t.Fatalf("RegisterImplementation Bird error: %v", err)
	}
	err = r.RegisterImplementation(flyerType, reflect.TypeOf(Plane{}))
	if err != nil {
		t.Fatalf("RegisterImplementation Plane error: %v", err)
	}

	// Get implementations
	impls := r.Implementations(flyerType)
	if len(impls) != 2 {
		t.Errorf("Implementations = %d, want 2", len(impls))
	}
}

func TestRegistryImplementationErrors(t *testing.T) {
	r := NewRegistry()

	flyerType := reflect.TypeOf((*Flyer)(nil)).Elem()

	// Register implementation for unregistered interface
	r.RegisterType(reflect.TypeOf(Bird{}))
	err := r.RegisterImplementation(flyerType, reflect.TypeOf(Bird{}))
	if err == nil {
		t.Error("Should fail for unregistered interface")
	}

	// Register interface, then try unregistered type
	r.RegisterInterfaceType(flyerType)
	err = r.RegisterImplementation(flyerType, reflect.TypeOf(Plane{}))
	if err == nil {
		t.Error("Should fail for unregistered implementation type")
	}
}

func TestRegistryNonInterface(t *testing.T) {
	r := NewRegistry()

	// Try to register non-interface as interface
	err := r.RegisterInterfaceType(reflect.TypeOf(Person{}))
	if err == nil {
		t.Error("RegisterInterface with non-interface should fail")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()

	r.RegisterType(reflect.TypeOf(Person{}))
	r.RegisterType(reflect.TypeOf(Animal{}))
	r.RegisterType(reflect.TypeOf(Bird{}))

	all := r.All()
	if len(all) != 3 {
		t.Errorf("All() = %d types, want 3", len(all))
	}
}

func TestRegistrySize(t *testing.T) {
	r := NewRegistry()

	if r.Size() != 0 {
		t.Errorf("Size() = %d, want 0", r.Size())
	}

	r.RegisterType(reflect.TypeOf(Person{}))
	if r.Size() != 1 {
		t.Errorf("Size() = %d, want 1", r.Size())
	}

	r.RegisterType(reflect.TypeOf(Animal{}))
	if r.Size() != 2 {
		t.Errorf("Size() = %d, want 2", r.Size())
	}
}

func TestRegistryClear(t *testing.T) {
	r := NewRegistry()

	r.RegisterType(reflect.TypeOf(Person{}))
	r.RegisterType(reflect.TypeOf(Animal{}))

	r.Clear()

	if r.Size() != 0 {
		t.Errorf("Size() after Clear = %d, want 0", r.Size())
	}

	// Should be able to register again
	_, err := r.RegisterType(reflect.TypeOf(Person{}))
	if err != nil {
		t.Fatalf("Register after Clear error: %v", err)
	}
}

func TestRegistryConcurrency(t *testing.T) {
	r := NewRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Register
			r.RegisterTypeWithID(reflect.TypeOf(Person{}), TypeID(128+i))

			// Lookup
			r.Lookup(TypeID(128 + i))
			r.LookupType(reflect.TypeOf(Person{}))
			r.TypeIDFor(Person{})
			r.All()
			r.Size()
		}(i)
	}
	wg.Wait()
}

func TestDefaultRegistry(t *testing.T) {
	// Clear default registry first
	DefaultRegistry.Clear()

	// Register using package-level functions
	_, err := Register[Person]()
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	if DefaultRegistry.Size() != 1 {
		t.Errorf("DefaultRegistry.Size() = %d, want 1", DefaultRegistry.Size())
	}

	// Clean up
	DefaultRegistry.Clear()
}

func TestMustRegister(t *testing.T) {
	r := NewRegistry()
	savedRegistry := DefaultRegistry
	DefaultRegistry = r // Temporarily replace
	defer func() { DefaultRegistry = savedRegistry }()

	// Should not panic
	MustRegister[Person]()

	// Duplicate with different ID should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegister should panic on duplicate")
		}
	}()
	MustRegisterWithID[Person](TypeID(999))
}

func TestAutoIncrementID(t *testing.T) {
	r := NewRegistry()

	r.RegisterType(reflect.TypeOf(Person{}))
	r.RegisterType(reflect.TypeOf(Animal{}))
	r.RegisterType(reflect.TypeOf(Bird{}))

	// IDs should be sequential starting from TypeIDUserStart
	reg1, _ := r.LookupType(reflect.TypeOf(Person{}))
	reg2, _ := r.LookupType(reflect.TypeOf(Animal{}))
	reg3, _ := r.LookupType(reflect.TypeOf(Bird{}))

	if reg1.ID != TypeIDUserStart {
		t.Errorf("First ID = %d, want %d", reg1.ID, TypeIDUserStart)
	}
	if reg2.ID != TypeIDUserStart+1 {
		t.Errorf("Second ID = %d, want %d", reg2.ID, TypeIDUserStart+1)
	}
	if reg3.ID != TypeIDUserStart+2 {
		t.Errorf("Third ID = %d, want %d", reg3.ID, TypeIDUserStart+2)
	}
}

func TestRegisterPointerType(t *testing.T) {
	r := NewRegistry()

	// Register struct type
	_, err := r.RegisterType(reflect.TypeOf(Person{}))
	if err != nil {
		t.Fatalf("RegisterType error: %v", err)
	}

	// Looking up pointer should find the same registration
	reg, ok := r.LookupType(reflect.TypeOf(&Person{}))
	if !ok {
		t.Error("LookupType for pointer should work")
	}
	if reg == nil {
		t.Error("Registration should not be nil")
	}
}

func TestRegistryNilValidation(t *testing.T) {
	r := NewRegistry()

	t.Run("RegisterType nil", func(t *testing.T) {
		_, err := r.RegisterType(nil)
		if err == nil {
			t.Error("RegisterType(nil) should return error")
		}
	})

	t.Run("RegisterTypeWithID nil", func(t *testing.T) {
		err := r.RegisterTypeWithID(nil, TypeID(200))
		if err == nil {
			t.Error("RegisterTypeWithID(nil, id) should return error")
		}
	})

	t.Run("RegisterValue nil", func(t *testing.T) {
		_, err := r.RegisterValue(nil)
		if err == nil {
			t.Error("RegisterValue(nil) should return error")
		}
	})

	t.Run("RegisterValueWithID nil", func(t *testing.T) {
		err := r.RegisterValueWithID(nil, TypeID(200))
		if err == nil {
			t.Error("RegisterValueWithID(nil, id) should return error")
		}
	})
}

func BenchmarkRegistryLookup(b *testing.B) {
	r := NewRegistry()
	id, _ := r.RegisterType(reflect.TypeOf(Person{}))

	b.Run("ByID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r.Lookup(id)
		}
	})

	b.Run("ByType", func(b *testing.B) {
		t := reflect.TypeOf(Person{})
		for i := 0; i < b.N; i++ {
			r.LookupType(t)
		}
	})

	b.Run("TypeIDFor", func(b *testing.B) {
		p := Person{Name: "Alice", Age: 30}
		for i := 0; i < b.N; i++ {
			r.TypeIDFor(p)
		}
	})
}
