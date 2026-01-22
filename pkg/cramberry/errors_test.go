package cramberry

import (
	"errors"
	"testing"
)

func TestDecodeErrorFormat(t *testing.T) {
	tests := []struct {
		name     string
		err      *DecodeError
		expected string
	}{
		{
			name: "basic",
			err: &DecodeError{
				Offset:  -1,
				Message: "something wrong",
			},
			expected: "cramberry: decode: something wrong",
		},
		{
			name: "with_offset",
			err: &DecodeError{
				Offset:  42,
				Message: "bad byte",
			},
			expected: "cramberry: decode at offset 42: bad byte",
		},
		{
			name: "with_type",
			err: &DecodeError{
				Type:    "Person",
				Offset:  -1,
				Message: "unknown field",
			},
			expected: "cramberry: decode Person: unknown field",
		},
		{
			name: "with_type_and_field",
			err: &DecodeError{
				Type:    "Person",
				Field:   "age",
				Offset:  -1,
				Message: "invalid value",
			},
			expected: "cramberry: decode Person.age: invalid value",
		},
		{
			name: "with_type_field_and_offset",
			err: &DecodeError{
				Type:    "Person",
				Field:   "name",
				Offset:  100,
				Message: "invalid UTF-8",
			},
			expected: "cramberry: decode Person.name at offset 100: invalid UTF-8",
		},
		{
			name: "field_only",
			err: &DecodeError{
				Field:   "items",
				Offset:  -1,
				Message: "too many items",
			},
			expected: "cramberry: decode items: too many items",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Error() != tc.expected {
				t.Errorf("Error() = %q, want %q", tc.err.Error(), tc.expected)
			}
		})
	}
}

func TestDecodeErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &DecodeError{
		Message: "decode failed",
		Cause:   cause,
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return cause")
	}

	if !errors.Is(err, cause) {
		t.Error("errors.Is should match cause")
	}
}

func TestDecodeErrorIs(t *testing.T) {
	err := &DecodeError{
		Message: "decode failed",
		Cause:   ErrUnexpectedEOF,
	}

	if !errors.Is(err, ErrUnexpectedEOF) {
		t.Error("Should match ErrUnexpectedEOF via cause")
	}

	if errors.Is(err, ErrInvalidWireType) {
		t.Error("Should not match ErrInvalidWireType")
	}

	// Error without cause
	errNoCause := &DecodeError{Message: "test"}
	if errors.Is(errNoCause, ErrUnexpectedEOF) {
		t.Error("Should not match without cause")
	}
}

func TestNewDecodeError(t *testing.T) {
	err := NewDecodeError("test message", ErrInvalidVarint)
	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}
	if err.Cause != ErrInvalidVarint {
		t.Error("Cause not set correctly")
	}
	if err.Offset != -1 {
		t.Errorf("Offset = %d, want -1", err.Offset)
	}
}

func TestNewDecodeErrorAt(t *testing.T) {
	err := NewDecodeErrorAt(42, "test message", ErrInvalidVarint)
	if err.Offset != 42 {
		t.Errorf("Offset = %d, want 42", err.Offset)
	}
}

func TestNewFieldDecodeError(t *testing.T) {
	err := NewFieldDecodeError("Person", "name", 1, 100, "invalid", ErrInvalidUTF8)
	if err.Type != "Person" {
		t.Errorf("Type = %q, want %q", err.Type, "Person")
	}
	if err.Field != "name" {
		t.Errorf("Field = %q, want %q", err.Field, "name")
	}
	if err.FieldNumber != 1 {
		t.Errorf("FieldNumber = %d, want 1", err.FieldNumber)
	}
	if err.Offset != 100 {
		t.Errorf("Offset = %d, want 100", err.Offset)
	}
}

func TestEncodeErrorFormat(t *testing.T) {
	tests := []struct {
		name     string
		err      *EncodeError
		expected string
	}{
		{
			name: "basic",
			err: &EncodeError{
				Message: "cannot encode",
			},
			expected: "cramberry: encode: cannot encode",
		},
		{
			name: "with_type",
			err: &EncodeError{
				Type:    "Person",
				Message: "nil value",
			},
			expected: "cramberry: encode Person: nil value",
		},
		{
			name: "with_type_and_field",
			err: &EncodeError{
				Type:    "Person",
				Field:   "age",
				Message: "negative age",
			},
			expected: "cramberry: encode Person.age: negative age",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Error() != tc.expected {
				t.Errorf("Error() = %q, want %q", tc.err.Error(), tc.expected)
			}
		})
	}
}

func TestEncodeErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying")
	err := &EncodeError{
		Message: "encode failed",
		Cause:   cause,
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return cause")
	}
}

func TestNewEncodeError(t *testing.T) {
	err := NewEncodeError("test", ErrUnregisteredType)
	if err.Message != "test" {
		t.Errorf("Message = %q, want %q", err.Message, "test")
	}
	if err.Cause != ErrUnregisteredType {
		t.Error("Cause not set correctly")
	}
}

func TestNewFieldEncodeError(t *testing.T) {
	err := NewFieldEncodeError("Order", "items", "too large", nil)
	if err.Type != "Order" {
		t.Errorf("Type = %q, want %q", err.Type, "Order")
	}
	if err.Field != "items" {
		t.Errorf("Field = %q, want %q", err.Field, "items")
	}
}

func TestRegistrationErrorFormat(t *testing.T) {
	tests := []struct {
		name     string
		err      *RegistrationError
		expected string
	}{
		{
			name: "without_id",
			err: &RegistrationError{
				TypeName: "User",
				Message:  "already registered",
			},
			expected: "cramberry: register User: already registered",
		},
		{
			name: "with_id",
			err: &RegistrationError{
				TypeName: "User",
				TypeID:   128,
				Message:  "ID conflict",
			},
			expected: "cramberry: register User (id=128): ID conflict",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Error() != tc.expected {
				t.Errorf("Error() = %q, want %q", tc.err.Error(), tc.expected)
			}
		})
	}
}

func TestNewRegistrationError(t *testing.T) {
	err := NewRegistrationError("User", 128, "conflict", ErrDuplicateTypeID)
	if err.TypeName != "User" {
		t.Errorf("TypeName = %q, want %q", err.TypeName, "User")
	}
	if err.TypeID != 128 {
		t.Errorf("TypeID = %d, want 128", err.TypeID)
	}
	if err.Unwrap() != ErrDuplicateTypeID {
		t.Error("Unwrap should return cause")
	}
}

func TestWrapError(t *testing.T) {
	// Nil error should return nil
	if WrapError(nil, "prefix") != nil {
		t.Error("WrapError(nil) should return nil")
	}

	// Non-nil error should be wrapped
	err := WrapError(ErrInvalidVarint, "context")
	if err == nil {
		t.Error("WrapError should return non-nil")
	}
	if !errors.Is(err, ErrInvalidVarint) {
		t.Error("Wrapped error should match original")
	}
}

func TestIsRetryable(t *testing.T) {
	// Currently all errors are non-retryable
	errs := []error{
		ErrInvalidVarint,
		ErrUnexpectedEOF,
		ErrNotPointer,
		NewDecodeError("test", nil),
	}

	for _, err := range errs {
		if IsRetryable(err) {
			t.Errorf("IsRetryable(%v) = true, want false", err)
		}
	}
}

func TestIsFatal(t *testing.T) {
	fatalErrors := []error{
		ErrNotPointer,
		ErrNilPointer,
		ErrUnregisteredType,
		ErrDuplicateType,
		ErrDuplicateTypeID,
	}

	for _, err := range fatalErrors {
		if !IsFatal(err) {
			t.Errorf("IsFatal(%v) = false, want true", err)
		}
	}

	nonFatalErrors := []error{
		ErrInvalidVarint,
		ErrUnexpectedEOF,
		ErrMaxSizeExceeded,
		NewDecodeError("test", nil),
	}

	for _, err := range nonFatalErrors {
		if IsFatal(err) {
			t.Errorf("IsFatal(%v) = true, want false", err)
		}
	}
}

func TestIsLimitExceeded(t *testing.T) {
	limitErrors := []error{
		ErrMaxDepthExceeded,
		ErrMaxSizeExceeded,
		ErrMaxStringLength,
		ErrMaxBytesLength,
		ErrMaxArrayLength,
		ErrMaxMapSize,
	}

	for _, err := range limitErrors {
		if !IsLimitExceeded(err) {
			t.Errorf("IsLimitExceeded(%v) = false, want true", err)
		}
	}

	nonLimitErrors := []error{
		ErrInvalidVarint,
		ErrUnexpectedEOF,
		ErrNotPointer,
	}

	for _, err := range nonLimitErrors {
		if IsLimitExceeded(err) {
			t.Errorf("IsLimitExceeded(%v) = true, want false", err)
		}
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are distinct
	errs := []error{
		ErrInvalidVarint,
		ErrUnexpectedEOF,
		ErrInvalidWireType,
		ErrUnknownType,
		ErrUnregisteredType,
		ErrTypeMismatch,
		ErrNotPointer,
		ErrNilPointer,
		ErrMaxDepthExceeded,
		ErrMaxSizeExceeded,
		ErrMaxStringLength,
		ErrMaxBytesLength,
		ErrMaxArrayLength,
		ErrMaxMapSize,
		ErrInvalidUTF8,
		ErrDuplicateType,
		ErrDuplicateTypeID,
		ErrInvalidFieldNumber,
		ErrUnknownField,
		ErrRequiredFieldMissing,
		ErrNegativeLength,
		ErrOverflow,
		ErrNotImplemented,
	}

	seen := make(map[string]bool)
	for _, err := range errs {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("Duplicate error message: %s", msg)
		}
		seen[msg] = true
	}
}
