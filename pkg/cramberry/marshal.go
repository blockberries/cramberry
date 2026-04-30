package cramberry

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/blockberries/cramberry/internal/wire"
)

// Marshal encodes a Go value into cramberry binary format.
// The value must be a supported type (see package documentation).
//
// For struct types, fields are encoded in field number order.
// Field numbers are assigned based on the "cramberry" struct tag,
// or sequentially if no tag is present.
func Marshal(v any) ([]byte, error) {
	return MarshalWithOptions(v, DefaultOptions)
}

// MarshalWithOptions encodes a Go value with the specified options.
func MarshalWithOptions(v any, opts Options) ([]byte, error) {
	w := GetWriter()
	defer PutWriter(w)
	w.SetOptions(opts)

	if err := encodeValue(w, reflect.ValueOf(v)); err != nil {
		return nil, err
	}
	if w.Err() != nil {
		return nil, w.Err()
	}
	return w.BytesCopy(), nil
}

// MarshalAppend appends the encoded value to the provided buffer.
// This can be used to reduce allocations.
func MarshalAppend(buf []byte, v any) ([]byte, error) {
	return MarshalAppendWithOptions(buf, v, DefaultOptions)
}

// MarshalAppendWithOptions appends the encoded value with the specified options.
func MarshalAppendWithOptions(buf []byte, v any, opts Options) ([]byte, error) {
	w := NewWriterWithBuffer(buf, opts)
	if err := encodeValue(w, reflect.ValueOf(v)); err != nil {
		return nil, err
	}
	if w.Err() != nil {
		return nil, w.Err()
	}
	return w.Bytes(), nil
}

// encodeValue encodes a reflect.Value to the writer.
func encodeValue(w *Writer, v reflect.Value) error {
	return encodeValueWithRegistry(w, v, DefaultRegistry)
}

// encodeValueWithRegistry encodes a reflect.Value using the specified registry.
func encodeValueWithRegistry(w *Writer, v reflect.Value, reg *Registry) error {
	// Handle nil interface or invalid values
	if !v.IsValid() {
		w.WriteNil()
		return w.Err()
	}

	// Handle interfaces specially for polymorphic encoding
	if v.Kind() == reflect.Interface {
		return encodeInterface(w, v, reg)
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			w.WriteNil()
			return w.Err()
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Bool:
		w.WriteBool(v.Bool())
	case reflect.Int8:
		w.WriteInt8(int8(v.Int()))
	case reflect.Int16:
		w.WriteInt16(int16(v.Int()))
	case reflect.Int32:
		w.WriteInt32(int32(v.Int()))
	case reflect.Int64, reflect.Int:
		w.WriteInt64(v.Int())
	case reflect.Uint8:
		w.WriteUint8(uint8(v.Uint()))
	case reflect.Uint16:
		w.WriteUint16(uint16(v.Uint()))
	case reflect.Uint32:
		w.WriteUint32(uint32(v.Uint()))
	case reflect.Uint64, reflect.Uint, reflect.Uintptr:
		w.WriteUint64(v.Uint())
	case reflect.Float32:
		w.WriteFloat32(float32(v.Float()))
	case reflect.Float64:
		w.WriteFloat64(v.Float())
	// NOTE: complex64/complex128 are Go-only types. TypeScript and Rust runtimes
	// do not support complex numbers. Use two separate float fields if cross-language
	// compatibility is needed.
	case reflect.Complex64:
		w.WriteComplex64(complex64(v.Complex()))
	case reflect.Complex128:
		w.WriteComplex128(v.Complex())
	case reflect.String:
		w.WriteString(v.String())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// []byte special case
			w.WriteBytes(v.Bytes())
		} else {
			return encodeSlice(w, v)
		}
	case reflect.Array:
		return encodeArray(w, v)
	case reflect.Map:
		return encodeMap(w, v)
	case reflect.Struct:
		return encodeStruct(w, v)
	default:
		return NewEncodeError("unsupported type: "+v.Type().String(), ErrNotImplemented)
	}
	return w.Err()
}

// encodeInterface encodes an interface value with its type ID.
func encodeInterface(w *Writer, v reflect.Value, reg *Registry) error {
	if v.IsNil() {
		w.WriteTypeID(TypeIDNil)
		return w.Err()
	}

	// Get the concrete value
	elem := v.Elem()

	// Look up the type ID
	typeID := reg.TypeIDFor(elem.Interface())
	if typeID == TypeIDNil {
		return NewEncodeError("unregistered interface type: "+elem.Type().String(), ErrUnregisteredType)
	}

	// Write the type ID
	w.WriteTypeID(typeID)
	if w.Err() != nil {
		return w.Err()
	}

	// Encode the concrete value
	return encodeValueWithRegistry(w, elem, reg)
}

// encodeSlice encodes a slice value. As with encodeStruct, depth tracking
// is owned by the BeginMessage at the field-wrapping layer; entering nested
// here too would double-count.
func encodeSlice(w *Writer, v reflect.Value) error {
	if v.IsNil() {
		w.WriteArrayHeader(0)
		return w.Err()
	}

	// Use packed encoding for primitive types
	if isPackableTypeCached(v.Type().Elem()) {
		return encodePackedSlice(w, v)
	}

	n := v.Len()
	w.WriteArrayHeader(n)
	if w.Err() != nil {
		return w.Err()
	}
	for i := range n {
		if err := encodeValue(w, v.Index(i)); err != nil {
			return err
		}
	}
	return w.Err()
}

// isPackableType returns true if the type can be packed in a contiguous byte sequence.
// Packable types are fixed-size primitives: integers, floats, and bools.
// isPackableTypeCached returns whether the type supports packed encoding, using cache.
func isPackableTypeCached(t reflect.Type) bool {
	if p, ok := packableCache.Load(t); ok {
		return p.(bool)
	}
	computed := computePackable(t)
	packableCache.Store(t, computed)
	return computed
}

// computePackable computes whether a type can be packed.
func computePackable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// encodePackedSlice encodes a slice of primitive types in packed format.
// Format: [count:varint][elem1][elem2]...[elemN]
// Elements are encoded without individual tags, contiguously.
func encodePackedSlice(w *Writer, v reflect.Value) error {
	n := v.Len()
	// Honor MaxArrayLength on the encode side too. Without this the encoder
	// could produce arrays that the decoder rejects, leaving cross-language
	// validators in disagreement on identical input.
	if lim := w.Options().Limits.MaxArrayLength; lim > 0 && n > lim {
		w.setError(ErrMaxArrayLength)
		return w.Err()
	}
	w.WriteUvarint(uint64(n))
	if w.Err() != nil {
		return w.Err()
	}

	elemKind := v.Type().Elem().Kind()

	for i := range n {
		elem := v.Index(i)
		switch elemKind {
		case reflect.Bool:
			w.WriteBool(elem.Bool())
		case reflect.Int8:
			w.WriteInt8(int8(elem.Int()))
		case reflect.Int16:
			w.WriteInt16(int16(elem.Int()))
		case reflect.Int32:
			w.WriteInt32(int32(elem.Int()))
		case reflect.Int64, reflect.Int:
			w.WriteInt64(elem.Int())
		case reflect.Uint8:
			w.WriteUint8(uint8(elem.Uint()))
		case reflect.Uint16:
			w.WriteUint16(uint16(elem.Uint()))
		case reflect.Uint32:
			w.WriteUint32(uint32(elem.Uint()))
		case reflect.Uint64, reflect.Uint:
			w.WriteUint64(elem.Uint())
		case reflect.Float32:
			w.WriteFloat32(float32(elem.Float()))
		case reflect.Float64:
			w.WriteFloat64(elem.Float())
		}
		if w.Err() != nil {
			return w.Err()
		}
	}

	return w.Err()
}

// encodeArray encodes an array value.
func encodeArray(w *Writer, v reflect.Value) error {
	if isPackableTypeCached(v.Type().Elem()) {
		return encodePackedArray(w, v)
	}

	// Depth tracking owned by BeginMessage at the field-wrapping layer.
	n := v.Len()
	w.WriteArrayHeader(n)
	if w.Err() != nil {
		return w.Err()
	}
	for i := range n {
		if err := encodeValue(w, v.Index(i)); err != nil {
			return err
		}
	}
	return w.Err()
}

// encodePackedArray encodes an array of primitive types in packed format.
func encodePackedArray(w *Writer, v reflect.Value) error {
	n := v.Len()
	if lim := w.Options().Limits.MaxArrayLength; lim > 0 && n > lim {
		w.setError(ErrMaxArrayLength)
		return w.Err()
	}
	w.WriteUvarint(uint64(n))
	if w.Err() != nil {
		return w.Err()
	}

	elemKind := v.Type().Elem().Kind()

	for i := range n {
		elem := v.Index(i)
		switch elemKind {
		case reflect.Bool:
			w.WriteBool(elem.Bool())
		case reflect.Int8:
			w.WriteInt8(int8(elem.Int()))
		case reflect.Int16:
			w.WriteInt16(int16(elem.Int()))
		case reflect.Int32:
			w.WriteInt32(int32(elem.Int()))
		case reflect.Int64, reflect.Int:
			w.WriteInt64(elem.Int())
		case reflect.Uint8:
			w.WriteUint8(uint8(elem.Uint()))
		case reflect.Uint16:
			w.WriteUint16(uint16(elem.Uint()))
		case reflect.Uint32:
			w.WriteUint32(uint32(elem.Uint()))
		case reflect.Uint64, reflect.Uint:
			w.WriteUint64(elem.Uint())
		case reflect.Float32:
			w.WriteFloat32(float32(elem.Float()))
		case reflect.Float64:
			w.WriteFloat64(elem.Float())
		}
		if w.Err() != nil {
			return w.Err()
		}
	}

	return w.Err()
}

// encodeMap encodes a map value. Keys are always sorted: the wire format
// is deterministic (CLAUDE.md determinism rule #1) and the codegen path
// always sorts via cramberry.SortedMapKeys, so sorting unconditionally
// here is the only way the reflection and codegen paths produce the same
// bytes for the same input.
func encodeMap(w *Writer, v reflect.Value) error {
	if v.IsNil() {
		w.WriteMapHeader(0)
		return w.Err()
	}

	// Depth tracking owned by BeginMessage at the field-wrapping layer.

	// Validate that the key type is supported for encoding
	keyType := v.Type().Key()
	if !isValidMapKeyType(keyType) {
		return NewEncodeError("unsupported map key type "+keyType.String()+" in "+v.Type().String()+"; map keys must be string, integer, float, or bool", nil)
	}

	keys := v.MapKeys()

	// Reject NaN keys: distinct NaN bit patterns canonicalize to the
	// same wire bytes (information loss), and `MapIndex(nan)` always
	// returns the zero Value because Go's map lookup uses ==-equality
	// and NaN != NaN — so every value in a NaN-keyed map silently
	// becomes the zero value on encode. The canon JSON path
	// (FormatFloat32/64) already refuses NaN; the binary path now
	// matches.
	if keyKind := v.Type().Key().Kind(); keyKind == reflect.Float32 || keyKind == reflect.Float64 {
		for _, k := range keys {
			if math.IsNaN(k.Float()) {
				return NewEncodeError("cannot encode NaN as map key in "+v.Type().String(), nil)
			}
		}
	}

	n := len(keys)
	w.WriteMapHeader(n)
	if w.Err() != nil {
		return w.Err()
	}

	keys = sortMapKeys(keys)

	for _, key := range keys {
		if err := encodeValue(w, key); err != nil {
			return err
		}
		if err := encodeValue(w, v.MapIndex(key)); err != nil {
			return err
		}
	}
	return w.Err()
}

// encodeStruct encodes a struct value using field tags.
// Uses compact tags and end marker format.
//
// Depth tracking is owned by BeginMessage at the field-wrapping layer (see
// the per-field loop below). encodeStruct itself doesn't enter a nested
// scope: at top level the depth is 0, and each nested struct field passes
// through BeginMessage which increments depth before this function is
// re-entered. Counting in both places double-counted depth and tripped
// the limit at half the documented depth.
func encodeStruct(w *Writer, v reflect.Value) error {
	info, err := getStructInfo(v.Type())
	if err != nil {
		return err
	}

	omitEmpty := w.Options().OmitEmpty
	for i := range info.fields {
		field := &info.fields[i]
		fv := v.Field(field.index)

		// A field is skipped if it is the zero value of an "omittable" kind
		// AND either the global OmitEmpty option is on, or the field
		// carries an explicit `,omitempty` tag. Composite kinds (struct,
		// map, named-type, interface) are always emitted to match the
		// codegen path — see isOmittableZero.
		if (omitEmpty || field.omitEmpty) && isOmittableZero(fv) {
			continue
		}

		// Emit the precomputed tag bytes directly. Avoids the per-field
		// sync.Map.Load (was: getWireTypeCached) and the WriteTag arithmetic.
		w.WriteRawBytes(field.tagBytes)
		if w.Err() != nil {
			return w.Err()
		}

		// Wrap composite bodies in BeginMessage/EndMessage so the on-wire
		// layout is `tag(WireBytes) length:varint body 0x00`, which
		// SkipValue(WireBytes) can skip even on schema mismatch.
		if field.needsLenPrefix {
			cp := w.BeginMessage()
			if cp < 0 {
				return w.Err()
			}
			if err := encodeValue(w, fv); err != nil {
				return err
			}
			w.EndMessage(cp)
		} else {
			if err := encodeValue(w, fv); err != nil {
				return err
			}
		}
	}

	// Write end marker
	w.WriteEndMarker()
	return w.Err()
}

// needsBodyLengthPrefix reports whether a field value requires its body to
// be length-prefixed by the surrounding encoder.
//
// Tag wire-type WireBytes is the canonical "length-prefixed payload" tag,
// so SkipValue(WireBytes) reads a length and skips that many bytes. For
// the impl to satisfy that contract, every value that gets a WireBytes tag
// has to either length-prefix itself (strings via WriteString and []byte
// via WriteBytes do this) or be wrapped here.
//
// Composite types — struct, slice (excluding []byte), array (excluding
// [N]byte), map, interface, and pointers to any of those — write inline
// bodies (count + elements, fields + end marker, or typeID + value), so we
// wrap them at the field boundary.
//
// Pointers to non-composite values (e.g. `*int32`, `*float64`) and
// `complex128` are also length-prefixed even though their *body* is a
// raw varint or 16 fixed bytes. The reason is the wire-type chosen for
// these on the encode side: `computeWireType` returns `WireBytes` for
// all pointers and for `complex128`. A `WireBytes` tag is the
// canonical "length-prefixed payload" — `SkipValue(WireBytes)` reads
// the length-varint and skips that many bytes. Without the wrapping
// here, an old decoder that doesn't know the field would read the
// first body byte as the length, mis-frame the rest of the message,
// and corrupt every following field. (Verified via a forward-compat
// test that previously failed for `*int32` set to 64.)
func needsBodyLengthPrefix(v reflect.Value) bool {
	return needsBodyLengthPrefixForType(v.Type())
}

// needsBodyLengthPrefixForType is the type-only version used by the
// per-field cache populated in parseStructInfo.
func needsBodyLengthPrefixForType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct, reflect.Map, reflect.Interface:
		return true
	case reflect.Slice, reflect.Array:
		return t.Elem().Kind() != reflect.Uint8
	case reflect.Complex128:
		return true
	}
	return false
}

// encodeFieldTag returns a freshly-allocated byte slice containing the
// wire-format encoding of (fieldNum, wireType): a single byte for
// fieldNum 1-15, an extended marker + varint for fieldNum >=16. Returns
// nil for fieldNum <= 0 (caller handles that as an error path).
func encodeFieldTag(fieldNum int, wireType byte) []byte {
	if fieldNum <= 0 {
		return nil
	}
	if fieldNum <= maxCompactFieldNum {
		return []byte{byte(fieldNum<<tagFieldNumShift) | (wireType << tagWireTypeShift)}
	}
	marker := (wireType << tagWireTypeShift) | tagExtendedBit
	buf := make([]byte, 0, 1+MaxVarintLen64)
	buf = append(buf, marker)
	buf = wire.AppendUvarint(buf, uint64(fieldNum))
	return buf
}

// computeWireType computes the wire type for a reflect.Type.
//
// Note for struct/pointer/interface fields: this returns WireBytes (the
// length-prefixed payload tag), but the body is end-marker-terminated.
// The encoder length-prefixes nested message bodies at the field-wrapping
// layer (see encodeStruct's per-field loop and needsBodyLengthPrefix); the
// resulting on-wire layout — `tag(WireBytes) length:varint body 0x00` —
// is consistent with what SkipValue(WireBytes) expects, so unknown fields
// can be skipped without knowing the schema.
func computeWireType(t reflect.Type) byte {
	switch t.Kind() {
	case reflect.Bool, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint, reflect.Uint64, reflect.Uintptr:
		return WireVarint
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int, reflect.Int64:
		return WireSVarint
	case reflect.Float32:
		return WireFixed32
	case reflect.Float64:
		return WireFixed64
	case reflect.Complex64:
		return WireFixed64 // 2x float32 = 8 bytes
	case reflect.Complex128, reflect.String, reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		return WireBytes
	case reflect.Ptr:
		// A pointer field uses the wire type of its pointee, NOT
		// WireBytes. Treating *int32 as WireBytes would make
		// reflection emit `tag(WireBytes) | length | svarint` while
		// codegen emits `tag(SVarint) | svarint` — the same logical
		// value as different bytes. Recurse so e.g. *int32 → SVarint,
		// *string → WireBytes, *Address → WireBytes.
		return computeWireType(t.Elem())
	case reflect.Interface:
		// Polymorphic interface — typeID + payload, length-prefixed
		// at the field boundary by needsBodyLengthPrefix.
		return WireBytes
	default:
		return WireBytes
	}
}

// fieldInfo holds metadata about a struct field.
//
// wireType, tagBytes, and needsLenPrefix are precomputed at parse time
// so the per-field encode loop in encodeStruct skips the per-field
// sync.Map.Load + kind switch + tag arithmetic that the original
// implementation re-did on every Marshal.
type fieldInfo struct {
	name           string
	num            int
	index          int
	omitEmpty      bool
	required       bool
	wireType       byte   // cached result of computeWireType(field.Type)
	tagBytes       []byte // pre-encoded compact (1B) or extended (2-6B) tag
	needsLenPrefix bool   // cached result of needsBodyLengthPrefixForType
}

// structInfo holds cached metadata about a struct type.
type structInfo struct {
	fields     []fieldInfo
	fieldByNum map[int]*fieldInfo // Pre-computed lookup for O(1) decode access
}

// structInfoCache caches struct metadata for performance.
var structInfoCache sync.Map

// packableCache caches whether element types support packed encoding.
var packableCache sync.Map

// cachedStructInfo holds either a parsed structInfo or the error that
// occurred while parsing — caching both so a bad struct tag fails the
// same way (with the same error) on every call without re-walking the
// reflect.Type each time.
type cachedStructInfo struct {
	info *structInfo
	err  error
}

// getStructInfo returns cached struct metadata.
//
// A struct with a malformed `cramberry:"..."` tag returns an error
// (rather than panicking, as it used to). Callers that import a
// third-party type with a bad tag get a normal Marshal/Unmarshal error
// instead of a process crash.
func getStructInfo(t reflect.Type) (*structInfo, error) {
	if cached, ok := structInfoCache.Load(t); ok {
		c := cached.(cachedStructInfo)
		return c.info, c.err
	}

	info, err := parseStructInfo(t)
	structInfoCache.Store(t, cachedStructInfo{info: info, err: err})
	return info, err
}

func parseStructInfo(t reflect.Type) (*structInfo, error) {
	info := &structInfo{
		fields: make([]fieldInfo, 0, t.NumField()),
	}

	// Track seen field numbers for uniqueness validation
	seenFieldNums := make(map[int]string)

	fieldNum := 1
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Skip unexported fields
		if !f.IsExported() {
			continue
		}

		fi := fieldInfo{
			name:  f.Name,
			index: i,
		}

		// Parse tag
		tag := f.Tag.Get("cramberry")
		if tag == "-" {
			continue // Skip this field
		}
		if tag != "" {
			parsed, err := parseFieldTag(tag, fi, fieldNum, f.Name)
			if err != nil {
				return nil, fmt.Errorf("cramberry: %s.%s: %w", t.Name(), f.Name, err)
			}
			fi = parsed
		} else {
			fi.num = fieldNum
		}

		// Validate field number uniqueness
		if existingField, ok := seenFieldNums[fi.num]; ok {
			return nil, fmt.Errorf("cramberry: duplicate field number %d in %s (fields %q and %q)",
				fi.num, t.Name(), existingField, f.Name)
		}
		seenFieldNums[fi.num] = f.Name

		// Precompute hot-path values for the per-field encode loop.
		fi.wireType = computeWireType(f.Type)
		fi.tagBytes = encodeFieldTag(fi.num, fi.wireType)
		fi.needsLenPrefix = needsBodyLengthPrefixForType(f.Type)

		info.fields = append(info.fields, fi)
		fieldNum++
	}

	// Sort fields by field number for consistent encoding
	sort.Slice(info.fields, func(i, j int) bool {
		return info.fields[i].num < info.fields[j].num
	})

	// Build fieldByNum lookup map for O(1) decode access
	info.fieldByNum = make(map[int]*fieldInfo, len(info.fields))
	for i := range info.fields {
		info.fieldByNum[info.fields[i].num] = &info.fields[i]
	}

	return info, nil
}

// parseFieldTag parses a cramberry struct tag.
// Format: "num,option,option,..."
// Options: omitempty, required
func parseFieldTag(tag string, fi fieldInfo, defaultNum int, fieldName string) (fieldInfo, error) {
	parts := strings.Split(tag, ",")
	if parts[0] != "" {
		n, err := strconv.Atoi(parts[0])
		if err != nil || n <= 0 {
			return fi, fmt.Errorf("invalid field number %q in tag for field %q", parts[0], fieldName)
		}
		fi.num = n
	} else {
		fi.num = defaultNum
	}

	for _, opt := range parts[1:] {
		switch opt {
		case "":
			// Trailing comma — tolerate.
		case "omitempty":
			fi.omitEmpty = true
		case "required":
			fi.required = true
		default:
			return fi, fmt.Errorf("unknown tag option %q on field %q", opt, fieldName)
		}
	}

	return fi, nil
}

// maxZeroValueDepth is the maximum recursion depth for isZeroValue.
// This prevents stack overflow on deeply nested structures.
const maxZeroValueDepth = 100

// isZeroValue returns true if the value is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
	return isZeroValueWithDepth(v, 0)
}

// isOmittableZero reports whether a struct field with `omitempty` semantics
// should be skipped given its current value. The rule mirrors what the
// codegen-emitted EncodeTo methods produce so reflection and codegen
// produce identical bytes for the same input:
//
//   - bool false, numeric 0, empty string, nil/empty []byte: omit
//   - nil pointer/interface, empty repeated slice (not []byte): omit
//   - struct, map, array, named-type: NEVER omit (always emit)
//
// The "always emit composites" rule matches Go's generator (see
// pkg/codegen/go_generator.go::zeroCheck), and corresponds to the
// "presence semantics" the wire format uses for messages and maps:
// emitting an empty map (`tag + length-prefix + 0-count`) is observably
// different from omitting the field entirely.
func isOmittableZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.String:
		return v.Len() == 0
	case reflect.Ptr:
		return v.IsNil()
	case reflect.Slice:
		// []byte and other repeated scalar/composite slices: empty/nil
		// counts as omittable. This matches Go codegen's
		// `if len(field) > 0 { ... }` guard.
		return v.IsNil() || v.Len() == 0
	}
	// Composite kinds (struct, map, array, interface) are not omitted.
	return false
}

// isZeroValueWithDepth returns true if the value is the zero value, with depth limiting.
// If depth exceeds maxZeroValueDepth, returns false (assume not zero) which is conservative:
// the field will be encoded rather than omitted.
func isZeroValueWithDepth(v reflect.Value, depth int) bool {
	if depth > maxZeroValueDepth {
		return false // Conservative: assume not zero, encode the field
	}

	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Struct:
		// Only consider exported fields. The encoder skips unexported fields,
		// so a struct that is "non-zero only in an unexported field" must
		// still be treated as zero — otherwise OmitEmpty emits an empty
		// body that round-trips to a zero value, breaking the invariant
		// "encoded → decoded yields the originally-emitted fields".
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			if !isZeroValueWithDepth(v.Field(i), depth+1) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// sortMapKeys sorts map keys for deterministic encoding. Uses
// slices.SortFunc on the typed []reflect.Value slice instead of
// sort.Slice, which avoids the reflect.Swapper-based generic swap that
// was the dominant cost in earlier profiles. The comparator still has
// to extract the typed value via reflect.Value.String/Int/Uint/Float —
// but those are constant-time accessors on a discriminated union.
func sortMapKeys(keys []reflect.Value) []reflect.Value {
	if len(keys) <= 1 {
		return keys
	}

	switch keys[0].Kind() {
	case reflect.String:
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return strings.Compare(a.String(), b.String())
		})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			ai, bi := a.Int(), b.Int()
			switch {
			case ai < bi:
				return -1
			case ai > bi:
				return 1
			}
			return 0
		})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			au, bu := a.Uint(), b.Uint()
			switch {
			case au < bu:
				return -1
			case au > bu:
				return 1
			}
			return 0
		})
	case reflect.Float32, reflect.Float64:
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			if CompareFloatKeys(a.Float(), b.Float()) {
				return -1
			}
			if CompareFloatKeys(b.Float(), a.Float()) {
				return 1
			}
			return 0
		})
	case reflect.Bool:
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			ab, bb := a.Bool(), b.Bool()
			if !ab && bb {
				return -1
			}
			if ab && !bb {
				return 1
			}
			return 0
		})
	default:
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return strings.Compare(a.String(), b.String())
		})
	}
	return keys
}

// CompareFloatKeys compares two float64 values with a total ordering that handles
// NaN and -0.0 correctly for deterministic sorting:
//   - All NaN values sort to the end (after +Inf).
//   - -0.0 and +0.0 are considered equal (both treated as 0.0).
//   - Different NaN bit patterns are compared by their raw bits for full
//     determinism.
//
// Exported for use by generated code, which must sort map keys with the same
// total order as the reflection marshaller.
func CompareFloatKeys(a, b float64) bool {
	aNaN := math.IsNaN(a)
	bNaN := math.IsNaN(b)

	// NaN values sort after everything else
	if aNaN && bNaN {
		// Both NaN: compare by raw bit pattern for full determinism.
		// This handles different NaN payloads deterministically.
		return math.Float64bits(a) < math.Float64bits(b)
	}
	if aNaN {
		return false // NaN is not less than any non-NaN value
	}
	if bNaN {
		return true // Any non-NaN value is less than NaN
	}

	// Handle negative zero: treat -0.0 as equal to +0.0
	// by comparing the actual values (both compare as 0.0).
	return a < b
}

// CompareFloat32Keys is the float32 analog of CompareFloatKeys.
func CompareFloat32Keys(a, b float32) bool {
	aNaN := math.IsNaN(float64(a))
	bNaN := math.IsNaN(float64(b))

	if aNaN && bNaN {
		return math.Float32bits(a) < math.Float32bits(b)
	}
	if aNaN {
		return false
	}
	if bNaN {
		return true
	}
	return a < b
}

// isValidMapKeyType checks if a type is valid as a map key for Cramberry encoding.
// Valid key types are: string, all integer types, float types, and bool.
// Complex types, slices, maps, arrays, structs, pointers, and interfaces are not
// supported as map keys because they cannot be reliably serialized and sorted
// for deterministic encoding.
func isValidMapKeyType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	case reflect.Bool:
		return true
	default:
		return false
	}
}
