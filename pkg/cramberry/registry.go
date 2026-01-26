package cramberry

import (
	"fmt"
	"reflect"
	"sync"
)

// TypeRegistration holds metadata for a registered type.
type TypeRegistration struct {
	// ID is the unique type identifier for wire format.
	ID TypeID

	// Name is the fully qualified type name.
	Name string

	// Type is the reflect.Type of the registered type.
	Type reflect.Type

	// Interfaces lists interfaces this type can satisfy.
	Interfaces []reflect.Type
}

// Registry manages type registrations for polymorphic serialization.
// It is safe for concurrent use.
type Registry struct {
	mu sync.RWMutex

	// byID maps TypeID to registration.
	byID map[TypeID]*TypeRegistration

	// byType maps reflect.Type to registration.
	byType map[reflect.Type]*TypeRegistration

	// byName maps type name to registration.
	byName map[string]*TypeRegistration

	// interfaceTypes maps interface types to their implementations.
	interfaceTypes map[reflect.Type][]TypeID

	// nextID is the next auto-assigned user type ID.
	nextID TypeID
}

// NewRegistry creates a new type registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:           make(map[TypeID]*TypeRegistration),
		byType:         make(map[reflect.Type]*TypeRegistration),
		byName:         make(map[string]*TypeRegistration),
		interfaceTypes: make(map[reflect.Type][]TypeID),
		nextID:         TypeIDUserStart,
	}
}

// DefaultRegistry is the global default registry.
var DefaultRegistry = NewRegistry()

// Register registers a type with an auto-assigned ID.
// The type must be a pointer to a struct or a struct.
func Register[T any]() (TypeID, error) {
	return DefaultRegistry.RegisterType(reflect.TypeOf((*T)(nil)).Elem())
}

// RegisterWithID registers a type with a specific ID.
func RegisterWithID[T any](id TypeID) error {
	return DefaultRegistry.RegisterTypeWithID(reflect.TypeOf((*T)(nil)).Elem(), id)
}

// RegisterInterface registers an interface type.
func RegisterInterface[T any]() error {
	return DefaultRegistry.RegisterInterfaceType(reflect.TypeOf((*T)(nil)).Elem())
}

// RegisterValue registers the type of the given value with an auto-assigned ID.
// Returns an error if the value is nil.
func (r *Registry) RegisterValue(v any) (TypeID, error) {
	if v == nil {
		return 0, NewRegistrationError("", 0, "cannot register nil value", nil)
	}
	return r.RegisterType(reflect.TypeOf(v))
}

// RegisterValueWithID registers the type of the given value with a specific ID.
// Returns an error if the value is nil.
func (r *Registry) RegisterValueWithID(v any, id TypeID) error {
	if v == nil {
		return NewRegistrationError("", id, "cannot register nil value", nil)
	}
	return r.RegisterTypeWithID(reflect.TypeOf(v), id)
}

// RegisterType registers a reflect.Type with an auto-assigned ID.
func (r *Registry) RegisterType(t reflect.Type) (TypeID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextID
	r.nextID++

	if err := r.registerLocked(t, id); err != nil {
		return 0, err
	}
	return id, nil
}

// RegisterTypeWithID registers a reflect.Type with a specific ID.
func (r *Registry) RegisterTypeWithID(t reflect.Type, id TypeID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.registerLocked(t, id)
}

// registerLocked performs the actual registration (must hold lock).
func (r *Registry) registerLocked(t reflect.Type, id TypeID) error {
	// Validate input type is not nil
	if t == nil {
		return NewRegistrationError("", id, "cannot register nil type", nil)
	}

	// Get the concrete type (dereference pointers)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Validate the dereferenced type is valid
	if t == nil || t.Kind() == reflect.Invalid {
		return NewRegistrationError("", id, "cannot register invalid type", nil)
	}

	name := typeName(t)

	// Check for duplicate registration
	if existing, ok := r.byType[t]; ok {
		if existing.ID != id {
			return NewRegistrationError(name, id, "type already registered with different ID", ErrDuplicateType)
		}
		return nil // Already registered with same ID
	}

	// Check for ID conflict
	if existing, ok := r.byID[id]; ok {
		return NewRegistrationError(name, id, fmt.Sprintf("ID already used by %s", existing.Name), ErrDuplicateTypeID)
	}

	// Check for name conflict
	if existing, ok := r.byName[name]; ok {
		return NewRegistrationError(name, id, fmt.Sprintf("name already used by type with ID %d", existing.ID), ErrDuplicateType)
	}

	// Validate ID range
	if !id.IsValid() {
		return NewRegistrationError(name, id, "invalid type ID", nil)
	}

	// Create registration
	reg := &TypeRegistration{
		ID:   id,
		Name: name,
		Type: t,
	}

	// Store registration
	r.byID[id] = reg
	r.byType[t] = reg
	r.byName[name] = reg

	return nil
}

// RegisterInterfaceType registers an interface type.
// Use reflect.TypeOf((*InterfaceName)(nil)).Elem() to get the interface type.
func (r *Registry) RegisterInterfaceType(t reflect.Type) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t.Kind() != reflect.Interface {
		return NewRegistrationError(t.String(), 0, "not an interface type", nil)
	}

	// Just mark this as a known interface
	if _, ok := r.interfaceTypes[t]; !ok {
		r.interfaceTypes[t] = nil
	}

	return nil
}

// RegisterImplementation registers that a type implements an interface.
func (r *Registry) RegisterImplementation(interfaceType, implType reflect.Type) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get concrete implementation type
	for implType.Kind() == reflect.Ptr {
		implType = implType.Elem()
	}

	// Check that implementation exists
	reg, ok := r.byType[implType]
	if !ok {
		return NewRegistrationError(typeName(implType), 0, "type not registered", ErrUnregisteredType)
	}

	// Check that interface is registered
	if _, ok := r.interfaceTypes[interfaceType]; !ok {
		return NewRegistrationError(interfaceType.String(), 0, "interface not registered", nil)
	}

	// Add to interface implementations
	r.interfaceTypes[interfaceType] = append(r.interfaceTypes[interfaceType], reg.ID)

	// Add interface to type's interface list
	reg.Interfaces = append(reg.Interfaces, interfaceType)

	return nil
}

// Lookup finds a registration by type ID.
func (r *Registry) Lookup(id TypeID) (*TypeRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.byID[id]
	return reg, ok
}

// LookupType finds a registration by reflect.Type.
func (r *Registry) LookupType(t reflect.Type) (*TypeRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Dereference pointers
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	reg, ok := r.byType[t]
	return reg, ok
}

// LookupName finds a registration by type name.
func (r *Registry) LookupName(name string) (*TypeRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.byName[name]
	return reg, ok
}

// TypeIDFor returns the type ID for a value.
// Returns TypeIDNil if the value is nil or not registered.
func (r *Registry) TypeIDFor(v any) TypeID {
	if v == nil {
		return TypeIDNil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return TypeIDNil
	}

	t := rv.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if reg, ok := r.byType[t]; ok {
		return reg.ID
	}
	return TypeIDNil
}

// NewValue creates a new value of the type with the given ID.
// Returns nil if the ID is not registered.
func (r *Registry) NewValue(id TypeID) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reg, ok := r.byID[id]
	if !ok {
		return nil, false
	}

	return reflect.New(reg.Type).Interface(), true
}

// Implementations returns all type IDs that implement an interface.
func (r *Registry) Implementations(interfaceType reflect.Type) []TypeID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.interfaceTypes[interfaceType]
	if ids == nil {
		return nil
	}

	// Return a copy to prevent modification
	result := make([]TypeID, len(ids))
	copy(result, ids)
	return result
}

// All returns all registered types.
func (r *Registry) All() []*TypeRegistration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*TypeRegistration, 0, len(r.byID))
	for _, reg := range r.byID {
		result = append(result, reg)
	}
	return result
}

// Size returns the number of registered types.
func (r *Registry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byID)
}

// Clear removes all registrations.
// This is primarily useful for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID = make(map[TypeID]*TypeRegistration)
	r.byType = make(map[reflect.Type]*TypeRegistration)
	r.byName = make(map[string]*TypeRegistration)
	r.interfaceTypes = make(map[reflect.Type][]TypeID)
	r.nextID = TypeIDUserStart
}

// typeName returns the fully qualified type name.
func typeName(t reflect.Type) string {
	if t.PkgPath() == "" {
		return t.Name()
	}
	return t.PkgPath() + "." + t.Name()
}

// MustRegister is like Register but panics on error.
// Returns the assigned TypeID.
//
// Deprecated: MustRegister can crash production services if called with
// a duplicate type. Consider using RegisterOrGet() for idempotent registration
// or Register() with proper error handling.
func MustRegister[T any]() TypeID {
	id, err := Register[T]()
	if err != nil {
		panic(err)
	}
	return id
}

// MustRegisterWithID is like RegisterWithID but panics on error.
//
// Deprecated: MustRegisterWithID can crash production services if called with
// a duplicate type or ID. Consider using RegisterOrGetWithID() for idempotent
// registration or RegisterWithID() with proper error handling.
func MustRegisterWithID[T any](id TypeID) {
	if err := RegisterWithID[T](id); err != nil {
		panic(err)
	}
}

// RegisterOrGet registers a type and returns its ID, or returns the existing
// ID if the type is already registered. This is safe for concurrent use
// and will never return an error for already-registered types.
func RegisterOrGet[T any]() TypeID {
	return DefaultRegistry.RegisterOrGetType(reflect.TypeOf((*T)(nil)).Elem())
}

// RegisterOrGetType registers a type and returns its ID, or returns the existing
// ID if the type is already registered.
func (r *Registry) RegisterOrGetType(t reflect.Type) TypeID {
	// Get the concrete type (dereference pointers)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Fast path: check if already registered (read lock)
	r.mu.RLock()
	if reg, ok := r.byType[t]; ok {
		r.mu.RUnlock()
		return reg.ID
	}
	r.mu.RUnlock()

	// Slow path: register with write lock
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have registered it)
	if reg, ok := r.byType[t]; ok {
		return reg.ID
	}

	// Register the type
	id := r.nextID
	r.nextID++

	name := typeName(t)
	reg := &TypeRegistration{
		ID:   id,
		Name: name,
		Type: t,
	}

	r.byID[id] = reg
	r.byType[t] = reg
	r.byName[name] = reg

	return id
}

// RegisterOrGetWithID registers a type with a specific ID, or returns the existing
// ID if the type is already registered. If the type is already registered with a
// different ID, the existing ID is returned (not the requested one).
func RegisterOrGetWithID[T any](id TypeID) TypeID {
	return DefaultRegistry.RegisterOrGetTypeWithID(reflect.TypeOf((*T)(nil)).Elem(), id)
}

// RegisterOrGetTypeWithID registers a type with a specific ID, or returns the existing
// ID if the type is already registered.
func (r *Registry) RegisterOrGetTypeWithID(t reflect.Type, id TypeID) TypeID {
	// Get the concrete type (dereference pointers)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Fast path: check if already registered (read lock)
	r.mu.RLock()
	if reg, ok := r.byType[t]; ok {
		r.mu.RUnlock()
		return reg.ID
	}
	r.mu.RUnlock()

	// Slow path: register with write lock
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if reg, ok := r.byType[t]; ok {
		return reg.ID
	}

	// Check if the requested ID is already in use
	if _, ok := r.byID[id]; ok {
		// ID is in use, fall back to auto-assigned ID
		id = r.nextID
		r.nextID++
	} else if id >= r.nextID {
		// Ensure nextID stays ahead of manually assigned IDs
		r.nextID = id + 1
	}

	name := typeName(t)
	reg := &TypeRegistration{
		ID:   id,
		Name: name,
		Type: t,
	}

	r.byID[id] = reg
	r.byType[t] = reg
	r.byName[name] = reg

	return id
}
