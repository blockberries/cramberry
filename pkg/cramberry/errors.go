// Package cramberry provides high-performance, deterministic binary serialization.
package cramberry

import (
	"errors"
	"fmt"
)

// Sentinel errors for common conditions.
// These can be checked using errors.Is().
var (
	// ErrInvalidVarint indicates the varint encoding is malformed.
	ErrInvalidVarint = errors.New("cramberry: invalid varint")

	// ErrUnexpectedEOF indicates the data was truncated unexpectedly.
	ErrUnexpectedEOF = errors.New("cramberry: unexpected end of data")

	// ErrInvalidWireType indicates an unknown or invalid wire type.
	ErrInvalidWireType = errors.New("cramberry: invalid wire type")

	// ErrUnknownType indicates a type ID was not found in the registry.
	ErrUnknownType = errors.New("cramberry: unknown type")

	// ErrUnregisteredType indicates a type was not registered for polymorphic encoding.
	ErrUnregisteredType = errors.New("cramberry: unregistered type")

	// ErrTypeMismatch indicates the decoded type does not match the expected type.
	ErrTypeMismatch = errors.New("cramberry: type mismatch")

	// ErrNotPointer indicates the target for unmarshaling is not a pointer.
	ErrNotPointer = errors.New("cramberry: target must be a pointer")

	// ErrNilPointer indicates the target pointer is nil.
	ErrNilPointer = errors.New("cramberry: nil pointer")

	// ErrMaxDepthExceeded indicates the maximum nesting depth was exceeded.
	ErrMaxDepthExceeded = errors.New("cramberry: maximum nesting depth exceeded")

	// ErrMaxSizeExceeded indicates the maximum message size was exceeded.
	ErrMaxSizeExceeded = errors.New("cramberry: maximum message size exceeded")

	// ErrMaxStringLength indicates the maximum string length was exceeded.
	ErrMaxStringLength = errors.New("cramberry: maximum string length exceeded")

	// ErrMaxBytesLength indicates the maximum bytes length was exceeded.
	ErrMaxBytesLength = errors.New("cramberry: maximum bytes length exceeded")

	// ErrMaxArrayLength indicates the maximum array/slice length was exceeded.
	ErrMaxArrayLength = errors.New("cramberry: maximum array length exceeded")

	// ErrMaxMapSize indicates the maximum map size was exceeded.
	ErrMaxMapSize = errors.New("cramberry: maximum map size exceeded")

	// ErrInvalidUTF8 indicates a string contains invalid UTF-8.
	ErrInvalidUTF8 = errors.New("cramberry: invalid UTF-8 string")

	// ErrDuplicateType indicates a type was registered more than once.
	ErrDuplicateType = errors.New("cramberry: duplicate type registration")

	// ErrDuplicateTypeID indicates a type ID was registered more than once.
	ErrDuplicateTypeID = errors.New("cramberry: duplicate type ID")

	// ErrInvalidFieldNumber indicates an invalid field number (must be > 0).
	ErrInvalidFieldNumber = errors.New("cramberry: invalid field number")

	// ErrUnknownField indicates an unknown field was encountered in strict mode.
	ErrUnknownField = errors.New("cramberry: unknown field")

	// ErrRequiredFieldMissing indicates a required field was not present.
	ErrRequiredFieldMissing = errors.New("cramberry: required field missing")

	// ErrNegativeLength indicates a negative length value was decoded.
	ErrNegativeLength = errors.New("cramberry: negative length")

	// ErrOverflow indicates an integer overflow during decoding.
	ErrOverflow = errors.New("cramberry: integer overflow")

	// ErrNotImplemented indicates a feature is not yet implemented.
	ErrNotImplemented = errors.New("cramberry: not implemented")
)

// DecodeError provides detailed context for decoding failures.
// It implements the error interface and supports error unwrapping.
type DecodeError struct {
	// Type is the name of the type being decoded (if known).
	Type string

	// Field is the name of the field being decoded (if applicable).
	Field string

	// FieldNumber is the wire field number (if applicable).
	FieldNumber int

	// Offset is the byte offset in the input where the error occurred.
	Offset int

	// Message describes what went wrong.
	Message string

	// Cause is the underlying error, if any.
	Cause error
}

// Error returns a formatted error message.
func (e *DecodeError) Error() string {
	var prefix string
	if e.Type != "" && e.Field != "" {
		prefix = fmt.Sprintf("%s.%s", e.Type, e.Field)
	} else if e.Type != "" {
		prefix = e.Type
	} else if e.Field != "" {
		prefix = e.Field
	}

	if prefix != "" {
		if e.Offset >= 0 {
			return fmt.Sprintf("cramberry: decode %s at offset %d: %s", prefix, e.Offset, e.Message)
		}
		return fmt.Sprintf("cramberry: decode %s: %s", prefix, e.Message)
	}

	if e.Offset >= 0 {
		return fmt.Sprintf("cramberry: decode at offset %d: %s", e.Offset, e.Message)
	}
	return fmt.Sprintf("cramberry: decode: %s", e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *DecodeError) Unwrap() error {
	return e.Cause
}

// Is reports whether the error matches the target.
// This supports errors.Is() for checking the cause.
func (e *DecodeError) Is(target error) bool {
	if e.Cause != nil && errors.Is(e.Cause, target) {
		return true
	}
	return false
}

// NewDecodeError creates a new DecodeError.
func NewDecodeError(message string, cause error) *DecodeError {
	return &DecodeError{
		Offset:  -1,
		Message: message,
		Cause:   cause,
	}
}

// NewDecodeErrorAt creates a new DecodeError with offset information.
func NewDecodeErrorAt(offset int, message string, cause error) *DecodeError {
	return &DecodeError{
		Offset:  offset,
		Message: message,
		Cause:   cause,
	}
}

// NewFieldDecodeError creates a DecodeError for a specific field.
func NewFieldDecodeError(typeName, fieldName string, fieldNum int, offset int, message string, cause error) *DecodeError {
	return &DecodeError{
		Type:        typeName,
		Field:       fieldName,
		FieldNumber: fieldNum,
		Offset:      offset,
		Message:     message,
		Cause:       cause,
	}
}

// EncodeError provides detailed context for encoding failures.
type EncodeError struct {
	// Type is the name of the type being encoded.
	Type string

	// Field is the name of the field being encoded (if applicable).
	Field string

	// Message describes what went wrong.
	Message string

	// Cause is the underlying error, if any.
	Cause error
}

// Error returns a formatted error message.
func (e *EncodeError) Error() string {
	var prefix string
	if e.Type != "" && e.Field != "" {
		prefix = fmt.Sprintf("%s.%s", e.Type, e.Field)
	} else if e.Type != "" {
		prefix = e.Type
	} else if e.Field != "" {
		prefix = e.Field
	}

	if prefix != "" {
		return fmt.Sprintf("cramberry: encode %s: %s", prefix, e.Message)
	}
	return fmt.Sprintf("cramberry: encode: %s", e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *EncodeError) Unwrap() error {
	return e.Cause
}

// Is reports whether the error matches the target.
func (e *EncodeError) Is(target error) bool {
	if e.Cause != nil && errors.Is(e.Cause, target) {
		return true
	}
	return false
}

// NewEncodeError creates a new EncodeError.
func NewEncodeError(message string, cause error) *EncodeError {
	return &EncodeError{
		Message: message,
		Cause:   cause,
	}
}

// NewFieldEncodeError creates an EncodeError for a specific field.
func NewFieldEncodeError(typeName, fieldName string, message string, cause error) *EncodeError {
	return &EncodeError{
		Type:    typeName,
		Field:   fieldName,
		Message: message,
		Cause:   cause,
	}
}

// RegistrationError represents an error during type registration.
type RegistrationError struct {
	// TypeName is the name of the type being registered.
	TypeName string

	// TypeID is the type ID being registered (if applicable).
	TypeID TypeID

	// Message describes what went wrong.
	Message string

	// Cause is the underlying error, if any.
	Cause error
}

// Error returns a formatted error message.
func (e *RegistrationError) Error() string {
	if e.TypeID != 0 {
		return fmt.Sprintf("cramberry: register %s (id=%d): %s", e.TypeName, e.TypeID, e.Message)
	}
	return fmt.Sprintf("cramberry: register %s: %s", e.TypeName, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *RegistrationError) Unwrap() error {
	return e.Cause
}

// NewRegistrationError creates a new RegistrationError.
func NewRegistrationError(typeName string, typeID TypeID, message string, cause error) *RegistrationError {
	return &RegistrationError{
		TypeName: typeName,
		TypeID:   typeID,
		Message:  message,
		Cause:    cause,
	}
}

// WrapError wraps an error with additional context.
// If the error is nil, nil is returned.
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// IsRetryable returns true if the error might succeed on retry.
// Currently, no cramberry errors are retryable.
func IsRetryable(_ error) bool {
	return false
}

// IsFatal returns true if the error indicates a programming error
// that should not occur in correct code.
func IsFatal(err error) bool {
	switch {
	case errors.Is(err, ErrNotPointer),
		errors.Is(err, ErrNilPointer),
		errors.Is(err, ErrUnregisteredType),
		errors.Is(err, ErrDuplicateType),
		errors.Is(err, ErrDuplicateTypeID):
		return true
	default:
		return false
	}
}

// IsLimitExceeded returns true if the error indicates a configured limit was exceeded.
func IsLimitExceeded(err error) bool {
	switch {
	case errors.Is(err, ErrMaxDepthExceeded),
		errors.Is(err, ErrMaxSizeExceeded),
		errors.Is(err, ErrMaxStringLength),
		errors.Is(err, ErrMaxBytesLength),
		errors.Is(err, ErrMaxArrayLength),
		errors.Is(err, ErrMaxMapSize):
		return true
	default:
		return false
	}
}
