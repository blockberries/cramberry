// Package testdata contains test types for schema extraction.
package testdata

// Status represents the status of a user.
type Status int

const (
	StatusUnknown Status = iota
	StatusActive
	StatusInactive
)

// Priority represents a priority level using uint.
type Priority uint8

const (
	PriorityLow    Priority = 0
	PriorityMedium Priority = 1
	PriorityHigh   Priority = 2
)

// User represents a user in the system.
type User struct {
	ID        int64  `cramberry:"1,required"`
	Name      string `cramberry:"2"`
	Email     string `cramberry:"3"`
	Status    Status `cramberry:"4"`
	Age       int32  `cramberry:"5,omitempty"`
	Tags      []string `cramberry:"6"`
	Metadata  map[string]string `cramberry:"7"`
	Address   *Address `cramberry:"8"`
	Internal  string `cramberry:"-"` // Should be skipped
}

// Address represents a physical address.
type Address struct {
	Street  string `cramberry:"1"`
	City    string `cramberry:"2"`
	Country string `cramberry:"3"`
	ZipCode string `cramberry:"4"`
}

// Admin is a user with admin privileges.
type Admin struct {
	User
	Permissions []string `cramberry:"10"`
}

// Person is an interface for any person type.
type Person interface {
	GetName() string
}

// GetName returns the user's name.
func (u *User) GetName() string {
	return u.Name
}

// GetName returns the admin's name.
func (a *Admin) GetName() string {
	return a.Name
}

// privateType is an unexported type that should be excluded by default.
type privateType struct {
	Value int
}

// Serializable is a marker interface for types that can be serialized.
// This is an empty interface used for polymorphic type grouping.
type Serializable interface{}

// Ensure User and Admin implement Serializable (no methods required)
var _ Serializable = (*User)(nil)
var _ Serializable = (*Admin)(nil)
