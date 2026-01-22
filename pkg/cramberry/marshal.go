package cramberry

import (
	"reflect"
	"sort"
	"strings"
	"sync"
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

// encodeSlice encodes a slice value.
func encodeSlice(w *Writer, v reflect.Value) error {
	if v.IsNil() {
		w.WriteArrayHeader(0)
		return w.Err()
	}

	// Use packed encoding for primitive types in V2
	if w.Options().WireVersion == WireVersionV2 && isPackableType(v.Type().Elem()) {
		return encodePackedSlice(w, v)
	}

	n := v.Len()
	w.WriteArrayHeader(n)
	if w.Err() != nil {
		return w.Err()
	}
	for i := 0; i < n; i++ {
		if err := encodeValue(w, v.Index(i)); err != nil {
			return err
		}
	}
	return w.Err()
}

// isPackableType returns true if the type can be packed in a contiguous byte sequence.
// Packable types are fixed-size primitives: integers, floats, and bools.
func isPackableType(t reflect.Type) bool {
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
	w.WriteUvarint(uint64(n))
	if w.Err() != nil {
		return w.Err()
	}

	elemKind := v.Type().Elem().Kind()

	for i := 0; i < n; i++ {
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
	// Use packed encoding for primitive types in V2
	if w.Options().WireVersion == WireVersionV2 && isPackableType(v.Type().Elem()) {
		return encodePackedArray(w, v)
	}

	n := v.Len()
	w.WriteArrayHeader(n)
	if w.Err() != nil {
		return w.Err()
	}
	for i := 0; i < n; i++ {
		if err := encodeValue(w, v.Index(i)); err != nil {
			return err
		}
	}
	return w.Err()
}

// encodePackedArray encodes an array of primitive types in packed format.
func encodePackedArray(w *Writer, v reflect.Value) error {
	n := v.Len()
	w.WriteUvarint(uint64(n))
	if w.Err() != nil {
		return w.Err()
	}

	elemKind := v.Type().Elem().Kind()

	for i := 0; i < n; i++ {
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

// encodeMap encodes a map value.
// If Deterministic option is enabled, keys are sorted for reproducible output.
func encodeMap(w *Writer, v reflect.Value) error {
	if v.IsNil() {
		w.WriteMapHeader(0)
		return w.Err()
	}

	// Validate that the key type is supported for encoding
	keyType := v.Type().Key()
	if !isValidMapKeyType(keyType) {
		return NewEncodeError("unsupported map key type "+keyType.String()+" in "+v.Type().String()+"; map keys must be string, integer, float, or bool", nil)
	}

	keys := v.MapKeys()
	n := len(keys)
	w.WriteMapHeader(n)
	if w.Err() != nil {
		return w.Err()
	}

	// Sort keys only if deterministic mode is enabled
	if w.Options().Deterministic {
		keys = sortMapKeys(keys)
	}

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
// Dispatches to V1 or V2 encoding based on options.
func encodeStruct(w *Writer, v reflect.Value) error {
	if w.Options().WireVersion == WireVersionV2 {
		return encodeStructV2(w, v)
	}
	return encodeStructV1(w, v)
}

// encodeStructV1 encodes a struct using V1 wire format (field count prefix).
// Deprecated: Use V2 for new code.
func encodeStructV1(w *Writer, v reflect.Value) error {
	info := getStructInfo(v.Type())

	// Write number of fields being encoded (for efficient decoding)
	fieldCount := 0
	for _, field := range info.fields {
		fv := v.Field(field.index)
		if w.Options().OmitEmpty && isZeroValue(fv) {
			continue
		}
		fieldCount++
	}
	w.WriteUvarint(uint64(fieldCount))
	if w.Err() != nil {
		return w.Err()
	}

	for _, field := range info.fields {
		fv := v.Field(field.index)

		// Handle OmitEmpty
		if w.Options().OmitEmpty && isZeroValue(fv) {
			continue
		}

		// Write field tag
		w.WriteTag(field.num, getWireType(fv.Type()))
		if w.Err() != nil {
			return w.Err()
		}

		// Encode value
		if err := encodeValue(w, fv); err != nil {
			return err
		}
	}

	return w.Err()
}

// encodeStructV2 encodes a struct using V2 wire format (compact tags + end marker).
// Key differences from V1:
//   - Single byte tags for fields 1-15
//   - End marker (0x00) instead of field count prefix
//   - No two-pass encoding needed
func encodeStructV2(w *Writer, v reflect.Value) error {
	info := getStructInfo(v.Type())

	for _, field := range info.fields {
		fv := v.Field(field.index)

		// Handle OmitEmpty
		if w.Options().OmitEmpty && isZeroValue(fv) {
			continue
		}

		// Write compact field tag
		w.WriteCompactTag(field.num, getWireTypeV2(fv.Type()))
		if w.Err() != nil {
			return w.Err()
		}

		// Encode value
		if err := encodeValue(w, fv); err != nil {
			return err
		}
	}

	// Write end marker
	w.WriteEndMarker()
	return w.Err()
}

// getWireTypeV2 returns the V2 wire type for a reflect.Type.
func getWireTypeV2(t reflect.Type) byte {
	switch t.Kind() {
	case reflect.Bool, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint, reflect.Uint64, reflect.Uintptr:
		return WireTypeV2Varint
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int, reflect.Int64:
		return WireTypeV2SVarint
	case reflect.Float32:
		return WireTypeV2Fixed32
	case reflect.Float64:
		return WireTypeV2Fixed64
	case reflect.Complex64:
		return WireTypeV2Fixed64 // 2x float32 = 8 bytes
	case reflect.Complex128, reflect.String, reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		return WireTypeV2Bytes
	case reflect.Ptr, reflect.Interface:
		return WireTypeV2Bytes
	default:
		return WireTypeV2Bytes
	}
}

// fieldInfo holds metadata about a struct field.
type fieldInfo struct {
	name      string
	num       int
	index     int
	omitEmpty bool
	required  bool
}

// structInfo holds cached metadata about a struct type.
type structInfo struct {
	fields []fieldInfo
}

// structInfoCache caches struct metadata for performance.
var structInfoCache sync.Map

// getStructInfo returns cached struct metadata.
func getStructInfo(t reflect.Type) *structInfo {
	if cached, ok := structInfoCache.Load(t); ok {
		return cached.(*structInfo)
	}

	info := &structInfo{
		fields: make([]fieldInfo, 0, t.NumField()),
	}

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
			fi = parseFieldTag(tag, fi, fieldNum)
		} else {
			fi.num = fieldNum
		}

		info.fields = append(info.fields, fi)
		fieldNum++
	}

	// Sort fields by field number for consistent encoding
	sort.Slice(info.fields, func(i, j int) bool {
		return info.fields[i].num < info.fields[j].num
	})

	structInfoCache.Store(t, info)
	return info
}

// parseFieldTag parses a cramberry struct tag.
// Format: "num,option,option,..."
// Options: omitempty, required
func parseFieldTag(tag string, fi fieldInfo, defaultNum int) fieldInfo {
	parts := strings.Split(tag, ",")
	if parts[0] != "" {
		// Parse field number
		var num int
		for _, c := range parts[0] {
			if c < '0' || c > '9' {
				num = defaultNum
				break
			}
			num = num*10 + int(c-'0')
		}
		if num > 0 {
			fi.num = num
		} else {
			fi.num = defaultNum
		}
	} else {
		fi.num = defaultNum
	}

	// Parse options
	for _, opt := range parts[1:] {
		switch opt {
		case "omitempty":
			fi.omitEmpty = true
		case "required":
			fi.required = true
		}
	}

	return fi
}

// getWireType returns the wire type for a reflect.Type.
func getWireType(t reflect.Type) WireType {
	switch t.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return WireVarint
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64, reflect.Uintptr:
		return WireVarint
	case reflect.Float32:
		return WireFixed32
	case reflect.Float64:
		return WireFixed64
	case reflect.Complex64:
		return WireFixed64 // 2x float32
	case reflect.Complex128:
		return WireBytes // 2x float64 as bytes
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		return WireBytes
	case reflect.Ptr, reflect.Interface:
		return WireTypeRef
	default:
		return WireBytes
	}
}

// isZeroValue returns true if the value is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
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
		// Check all fields
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// sortMapKeys sorts map keys for deterministic encoding.
func sortMapKeys(keys []reflect.Value) []reflect.Value {
	if len(keys) <= 1 {
		return keys
	}

	// Determine the key type and sort accordingly
	switch keys[0].Kind() {
	case reflect.String:
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Int() < keys[j].Int()
		})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Uint() < keys[j].Uint()
		})
	case reflect.Float32, reflect.Float64:
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Float() < keys[j].Float()
		})
	case reflect.Bool:
		sort.Slice(keys, func(i, j int) bool {
			return !keys[i].Bool() && keys[j].Bool()
		})
	default:
		// For other types, use string representation
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})
	}
	return keys
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
