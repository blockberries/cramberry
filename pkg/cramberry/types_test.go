package cramberry

import "testing"

func TestTypeIDRanges(t *testing.T) {
	tests := []struct {
		id        TypeID
		isBuiltin bool
		isStdlib  bool
		isUser    bool
		isNil     bool
		isValid   bool
	}{
		{TypeIDNil, false, false, false, true, false},
		{1, true, false, false, false, true},
		{63, true, false, false, false, true},
		{64, false, true, false, false, true},
		{127, false, true, false, false, true},
		{128, false, false, true, false, true},
		{1000, false, false, true, false, true},
	}

	for _, tc := range tests {
		if tc.id.IsBuiltin() != tc.isBuiltin {
			t.Errorf("TypeID(%d).IsBuiltin() = %v, want %v", tc.id, tc.id.IsBuiltin(), tc.isBuiltin)
		}
		if tc.id.IsStdlib() != tc.isStdlib {
			t.Errorf("TypeID(%d).IsStdlib() = %v, want %v", tc.id, tc.id.IsStdlib(), tc.isStdlib)
		}
		if tc.id.IsUser() != tc.isUser {
			t.Errorf("TypeID(%d).IsUser() = %v, want %v", tc.id, tc.id.IsUser(), tc.isUser)
		}
		if tc.id.IsNil() != tc.isNil {
			t.Errorf("TypeID(%d).IsNil() = %v, want %v", tc.id, tc.id.IsNil(), tc.isNil)
		}
		if tc.id.IsValid() != tc.isValid {
			t.Errorf("TypeID(%d).IsValid() = %v, want %v", tc.id, tc.id.IsValid(), tc.isValid)
		}
	}
}

func TestTypeIDConstants(t *testing.T) {
	// Verify range boundaries
	if TypeIDBuiltinStart != 1 {
		t.Errorf("TypeIDBuiltinStart = %d, want 1", TypeIDBuiltinStart)
	}
	if TypeIDBuiltinEnd != 63 {
		t.Errorf("TypeIDBuiltinEnd = %d, want 63", TypeIDBuiltinEnd)
	}
	if TypeIDStdlibStart != 64 {
		t.Errorf("TypeIDStdlibStart = %d, want 64", TypeIDStdlibStart)
	}
	if TypeIDStdlibEnd != 127 {
		t.Errorf("TypeIDStdlibEnd = %d, want 127", TypeIDStdlibEnd)
	}
	if TypeIDUserStart != 128 {
		t.Errorf("TypeIDUserStart = %d, want 128", TypeIDUserStart)
	}

	// Verify ranges don't overlap
	if TypeIDBuiltinEnd >= TypeIDStdlibStart {
		t.Error("Builtin and stdlib ranges overlap")
	}
	if TypeIDStdlibEnd >= TypeIDUserStart {
		t.Error("Stdlib and user ranges overlap")
	}
}

func TestWireTypeString(t *testing.T) {
	tests := []struct {
		wt       WireType
		expected string
	}{
		{WireVarint, "Varint"},
		{WireFixed64, "Fixed64"},
		{WireBytes, "Bytes"},
		{WireFixed32, "Fixed32"},
		{WireSVarint, "SVarint"},
		{WireTypeRef, "TypeRef"},
		{WireType(100), "Unknown"},
	}

	for _, tc := range tests {
		if tc.wt.String() != tc.expected {
			t.Errorf("WireType(%d).String() = %q, want %q", tc.wt, tc.wt.String(), tc.expected)
		}
	}
}

func TestWireTypeIsValid(t *testing.T) {
	validTypes := []WireType{WireVarint, WireFixed64, WireBytes, WireFixed32, WireSVarint, WireTypeRef}
	for _, wt := range validTypes {
		if !wt.IsValid() {
			t.Errorf("WireType(%d).IsValid() = false, want true", wt)
		}
	}

	invalidTypes := []WireType{3, 4, 8, 100}
	for _, wt := range invalidTypes {
		if wt.IsValid() {
			t.Errorf("WireType(%d).IsValid() = true, want false", wt)
		}
	}
}

func TestWireTypeConstants(t *testing.T) {
	// Verify wire type values (must match protobuf for compatibility)
	if WireVarint != 0 {
		t.Errorf("WireVarint = %d, want 0", WireVarint)
	}
	if WireFixed64 != 1 {
		t.Errorf("WireFixed64 = %d, want 1", WireFixed64)
	}
	if WireBytes != 2 {
		t.Errorf("WireBytes = %d, want 2", WireBytes)
	}
	if WireFixed32 != 5 {
		t.Errorf("WireFixed32 = %d, want 5", WireFixed32)
	}
	if WireSVarint != 6 {
		t.Errorf("WireSVarint = %d, want 6", WireSVarint)
	}
	if WireTypeRef != 7 {
		t.Errorf("WireTypeRef = %d, want 7", WireTypeRef)
	}
}

func TestDefaultLimits(t *testing.T) {
	l := DefaultLimits
	if l.MaxMessageSize != 64*1024*1024 {
		t.Errorf("MaxMessageSize = %d, want %d", l.MaxMessageSize, 64*1024*1024)
	}
	if l.MaxDepth != 100 {
		t.Errorf("MaxDepth = %d, want 100", l.MaxDepth)
	}
	if l.MaxStringLength != 10*1024*1024 {
		t.Errorf("MaxStringLength = %d, want %d", l.MaxStringLength, 10*1024*1024)
	}
	if l.MaxBytesLength != 100*1024*1024 {
		t.Errorf("MaxBytesLength = %d, want %d", l.MaxBytesLength, 100*1024*1024)
	}
	if l.MaxArrayLength != 1_000_000 {
		t.Errorf("MaxArrayLength = %d, want %d", l.MaxArrayLength, 1_000_000)
	}
	if l.MaxMapSize != 1_000_000 {
		t.Errorf("MaxMapSize = %d, want %d", l.MaxMapSize, 1_000_000)
	}
}

func TestSecureLimits(t *testing.T) {
	l := SecureLimits
	if l.MaxMessageSize != 1*1024*1024 {
		t.Errorf("MaxMessageSize = %d, want %d", l.MaxMessageSize, 1*1024*1024)
	}
	if l.MaxDepth != 32 {
		t.Errorf("MaxDepth = %d, want 32", l.MaxDepth)
	}
	if l.MaxArrayLength != 10_000 {
		t.Errorf("MaxArrayLength = %d, want %d", l.MaxArrayLength, 10_000)
	}

	// Secure limits should be more restrictive than default
	if l.MaxMessageSize >= DefaultLimits.MaxMessageSize {
		t.Error("SecureLimits.MaxMessageSize should be less than DefaultLimits")
	}
	if l.MaxDepth >= DefaultLimits.MaxDepth {
		t.Error("SecureLimits.MaxDepth should be less than DefaultLimits")
	}
}

func TestNoLimits(t *testing.T) {
	l := NoLimits
	if l.MaxMessageSize != 0 {
		t.Errorf("MaxMessageSize = %d, want 0", l.MaxMessageSize)
	}
	if l.MaxDepth != 0 {
		t.Errorf("MaxDepth = %d, want 0", l.MaxDepth)
	}
	if l.MaxStringLength != 0 {
		t.Errorf("MaxStringLength = %d, want 0", l.MaxStringLength)
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions
	if opts.StrictMode {
		t.Error("DefaultOptions.StrictMode should be false")
	}
	if !opts.ValidateUTF8 {
		t.Error("DefaultOptions.ValidateUTF8 should be true")
	}
	if !opts.OmitEmpty {
		t.Error("DefaultOptions.OmitEmpty should be true")
	}
}

func TestSecureOptions(t *testing.T) {
	opts := SecureOptions
	if !opts.ValidateUTF8 {
		t.Error("SecureOptions.ValidateUTF8 should be true")
	}
	if opts.Limits.MaxMessageSize != SecureLimits.MaxMessageSize {
		t.Error("SecureOptions should use SecureLimits")
	}
}

func TestStrictOptions(t *testing.T) {
	opts := StrictOptions
	if !opts.StrictMode {
		t.Error("StrictOptions.StrictMode should be true")
	}
	if !opts.ValidateUTF8 {
		t.Error("StrictOptions.ValidateUTF8 should be true")
	}
}

func TestVersionInfo(t *testing.T) {
	info := VersionInfo()
	// Should contain version, commit, and date
	if len(info) == 0 {
		t.Error("VersionInfo() should not be empty")
	}
	// Check it contains the default values
	if Version == "dev" {
		if info != "dev (unknown, unknown)" {
			t.Errorf("VersionInfo() = %q, want %q", info, "dev (unknown, unknown)")
		}
	}
}

func TestSizeConstants(t *testing.T) {
	if BoolSize != 1 {
		t.Errorf("BoolSize = %d, want 1", BoolSize)
	}
	if Fixed32Size != 4 {
		t.Errorf("Fixed32Size = %d, want 4", Fixed32Size)
	}
	if Fixed64Size != 8 {
		t.Errorf("Fixed64Size = %d, want 8", Fixed64Size)
	}
	if Float32Size != 4 {
		t.Errorf("Float32Size = %d, want 4", Float32Size)
	}
	if Float64Size != 8 {
		t.Errorf("Float64Size = %d, want 8", Float64Size)
	}
	if Complex64Size != 8 {
		t.Errorf("Complex64Size = %d, want 8", Complex64Size)
	}
	if Complex128Size != 16 {
		t.Errorf("Complex128Size = %d, want 16", Complex128Size)
	}
	if MaxVarintLen64 != 10 {
		t.Errorf("MaxVarintLen64 = %d, want 10", MaxVarintLen64)
	}
	if MaxTagSize != 10 {
		t.Errorf("MaxTagSize = %d, want 10", MaxTagSize)
	}
}
