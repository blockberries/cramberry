package cramberry

import (
	"reflect"
)

// Unmarshal decodes cramberry binary data into a Go value.
// The target must be a non-nil pointer to the value to decode into.
func Unmarshal(data []byte, v any) error {
	return UnmarshalWithOptions(data, v, DefaultOptions)
}

// UnmarshalWithOptions decodes data with the specified options.
func UnmarshalWithOptions(data []byte, v any, opts Options) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return ErrNotPointer
	}
	if rv.IsNil() {
		return ErrNilPointer
	}

	r := NewReaderWithOptions(data, opts)
	if err := decodeValue(r, rv.Elem()); err != nil {
		return err
	}
	return r.Err()
}

// decodeValue decodes a value from the reader into the reflect.Value.
func decodeValue(r *Reader, v reflect.Value) error {
	if !v.CanSet() {
		return NewDecodeError("cannot set value", nil)
	}

	// Handle pointers specially
	if v.Kind() == reflect.Ptr {
		return decodePointer(r, v)
	}

	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(r.ReadBool())
	case reflect.Int8:
		v.SetInt(int64(r.ReadInt8()))
	case reflect.Int16:
		v.SetInt(int64(r.ReadInt16()))
	case reflect.Int32:
		v.SetInt(int64(r.ReadInt32()))
	case reflect.Int64, reflect.Int:
		v.SetInt(r.ReadInt64())
	case reflect.Uint8:
		v.SetUint(uint64(r.ReadUint8()))
	case reflect.Uint16:
		v.SetUint(uint64(r.ReadUint16()))
	case reflect.Uint32:
		v.SetUint(uint64(r.ReadUint32()))
	case reflect.Uint64, reflect.Uint, reflect.Uintptr:
		v.SetUint(r.ReadUint64())
	case reflect.Float32:
		v.SetFloat(float64(r.ReadFloat32()))
	case reflect.Float64:
		v.SetFloat(r.ReadFloat64())
	case reflect.Complex64:
		v.SetComplex(complex128(r.ReadComplex64()))
	case reflect.Complex128:
		v.SetComplex(r.ReadComplex128())
	case reflect.String:
		v.SetString(r.ReadString())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// []byte special case
			v.SetBytes(r.ReadBytes())
		} else {
			return decodeSlice(r, v)
		}
	case reflect.Array:
		return decodeArray(r, v)
	case reflect.Map:
		return decodeMap(r, v)
	case reflect.Struct:
		return decodeStruct(r, v)
	case reflect.Interface:
		return decodeInterface(r, v)
	default:
		return NewDecodeError("unsupported type: "+v.Type().String(), ErrNotImplemented)
	}
	return r.Err()
}

// decodePointer decodes a pointer value.
func decodePointer(r *Reader, v reflect.Value) error {
	// Peek at the first byte to check for nil
	if r.Len() == 0 {
		return r.Err()
	}

	// Check if this is a nil value (TypeIDNil encoded as varint 0)
	if r.Remaining()[0] == 0 {
		r.Skip(1) // Consume the nil marker
		v.SetZero()
		return r.Err()
	}

	// Allocate if needed
	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}

	return decodeValue(r, v.Elem())
}

// decodeSlice decodes a slice value.
func decodeSlice(r *Reader, v reflect.Value) error {
	// Use packed decoding for primitive types (no depth tracking needed for primitives)
	if isPackableType(v.Type().Elem()) {
		return decodePackedSlice(r, v)
	}

	// Check depth limit for non-primitive element types
	if !r.enterNested() {
		return r.Err()
	}
	defer r.exitNested()

	n := r.ReadArrayHeader()
	if r.Err() != nil {
		return r.Err()
	}

	if n == 0 {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		return nil
	}

	// Create the slice
	slice := reflect.MakeSlice(v.Type(), n, n)

	for i := 0; i < n; i++ {
		if err := decodeValue(r, slice.Index(i)); err != nil {
			return err
		}
	}

	v.Set(slice)
	return r.Err()
}

// decodePackedSlice decodes a slice of primitive types in packed format.
func decodePackedSlice(r *Reader, v reflect.Value) error {
	// Use ReadArrayHeader for overflow protection and limit checking
	n := r.ReadArrayHeader()
	if r.Err() != nil {
		return r.Err()
	}

	if n == 0 {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		return nil
	}

	slice := reflect.MakeSlice(v.Type(), n, n)
	elemKind := v.Type().Elem().Kind()

	for i := 0; i < n; i++ {
		elem := slice.Index(i)
		switch elemKind {
		case reflect.Bool:
			elem.SetBool(r.ReadBool())
		case reflect.Int8:
			elem.SetInt(int64(r.ReadInt8()))
		case reflect.Int16:
			elem.SetInt(int64(r.ReadInt16()))
		case reflect.Int32:
			elem.SetInt(int64(r.ReadInt32()))
		case reflect.Int64, reflect.Int:
			elem.SetInt(r.ReadInt64())
		case reflect.Uint8:
			elem.SetUint(uint64(r.ReadUint8()))
		case reflect.Uint16:
			elem.SetUint(uint64(r.ReadUint16()))
		case reflect.Uint32:
			elem.SetUint(uint64(r.ReadUint32()))
		case reflect.Uint64, reflect.Uint:
			elem.SetUint(r.ReadUint64())
		case reflect.Float32:
			elem.SetFloat(float64(r.ReadFloat32()))
		case reflect.Float64:
			elem.SetFloat(r.ReadFloat64())
		}
		if r.Err() != nil {
			return r.Err()
		}
	}

	v.Set(slice)
	return r.Err()
}

// decodeArray decodes an array value.
func decodeArray(r *Reader, v reflect.Value) error {
	// Use packed decoding for primitive types (no depth tracking needed for primitives)
	if isPackableType(v.Type().Elem()) {
		return decodePackedArray(r, v)
	}

	// Check depth limit for non-primitive element types
	if !r.enterNested() {
		return r.Err()
	}
	defer r.exitNested()

	n := r.ReadArrayHeader()
	if r.Err() != nil {
		return r.Err()
	}

	if n > v.Len() {
		return NewDecodeError("array length mismatch", nil)
	}

	for i := 0; i < n; i++ {
		if err := decodeValue(r, v.Index(i)); err != nil {
			return err
		}
	}

	// Zero out remaining elements if the encoded array was shorter
	for i := n; i < v.Len(); i++ {
		v.Index(i).SetZero()
	}

	return r.Err()
}

// decodePackedArray decodes an array of primitive types in packed format.
func decodePackedArray(r *Reader, v reflect.Value) error {
	// Use ReadArrayHeader for overflow protection and limit checking
	n := r.ReadArrayHeader()
	if r.Err() != nil {
		return r.Err()
	}

	if n > v.Len() {
		return NewDecodeError("array length mismatch", nil)
	}

	elemKind := v.Type().Elem().Kind()

	for i := 0; i < n; i++ {
		elem := v.Index(i)
		switch elemKind {
		case reflect.Bool:
			elem.SetBool(r.ReadBool())
		case reflect.Int8:
			elem.SetInt(int64(r.ReadInt8()))
		case reflect.Int16:
			elem.SetInt(int64(r.ReadInt16()))
		case reflect.Int32:
			elem.SetInt(int64(r.ReadInt32()))
		case reflect.Int64, reflect.Int:
			elem.SetInt(r.ReadInt64())
		case reflect.Uint8:
			elem.SetUint(uint64(r.ReadUint8()))
		case reflect.Uint16:
			elem.SetUint(uint64(r.ReadUint16()))
		case reflect.Uint32:
			elem.SetUint(uint64(r.ReadUint32()))
		case reflect.Uint64, reflect.Uint:
			elem.SetUint(r.ReadUint64())
		case reflect.Float32:
			elem.SetFloat(float64(r.ReadFloat32()))
		case reflect.Float64:
			elem.SetFloat(r.ReadFloat64())
		}
		if r.Err() != nil {
			return r.Err()
		}
	}

	// Zero out remaining elements if the encoded array was shorter
	for i := n; i < v.Len(); i++ {
		v.Index(i).SetZero()
	}

	return r.Err()
}

// decodeMap decodes a map value.
func decodeMap(r *Reader, v reflect.Value) error {
	// Check depth limit
	if !r.enterNested() {
		return r.Err()
	}
	defer r.exitNested()

	n := r.ReadMapHeader()
	if r.Err() != nil {
		return r.Err()
	}

	// Create the map if it's nil
	if v.IsNil() {
		v.Set(reflect.MakeMapWithSize(v.Type(), n))
	}

	keyType := v.Type().Key()
	elemType := v.Type().Elem()

	for i := 0; i < n; i++ {
		key := reflect.New(keyType).Elem()
		if err := decodeValue(r, key); err != nil {
			return err
		}

		elem := reflect.New(elemType).Elem()
		if err := decodeValue(r, elem); err != nil {
			return err
		}

		v.SetMapIndex(key, elem)
	}

	return r.Err()
}

// decodeStruct decodes a struct value using field tags.
// Uses compact tags and reads until end marker.
func decodeStruct(r *Reader, v reflect.Value) error {
	// Check depth limit
	if !r.enterNested() {
		return r.Err()
	}
	defer r.exitNested()

	info := getStructInfo(v.Type())

	// Create a map of field number to field info for quick lookup
	fieldMap := make(map[int]*fieldInfo, len(info.fields))
	for i := range info.fields {
		fieldMap[info.fields[i].num] = &info.fields[i]
	}

	// Track which fields were set (for required field checking)
	fieldsSeen := make(map[int]bool)

	// Read fields until end marker
	for {
		fieldNum, wireType := r.ReadCompactTag()
		if r.Err() != nil {
			return r.Err()
		}

		// fieldNum=0 indicates end marker
		if fieldNum == 0 {
			break
		}

		fi, ok := fieldMap[fieldNum]
		if !ok {
			// Unknown field - skip it in non-strict mode
			if r.Options().StrictMode {
				return NewFieldDecodeError(v.Type().Name(), "", fieldNum, r.Pos(), "unknown field", ErrUnknownField)
			}
			r.SkipValueV2(wireType)
			continue
		}

		fieldsSeen[fieldNum] = true
		fv := v.Field(fi.index)

		if err := decodeValue(r, fv); err != nil {
			return err
		}
	}

	// Check for missing required fields
	for _, fi := range info.fields {
		if fi.required && !fieldsSeen[fi.num] {
			return NewFieldDecodeError(v.Type().Name(), fi.name, fi.num, -1, "required field missing", ErrRequiredFieldMissing)
		}
	}

	return r.Err()
}

// decodeInterface decodes an interface value using the type registry.
func decodeInterface(r *Reader, v reflect.Value) error {
	return decodeInterfaceWithRegistry(r, v, DefaultRegistry)
}

// decodeInterfaceWithRegistry decodes an interface value using the specified registry.
func decodeInterfaceWithRegistry(r *Reader, v reflect.Value, reg *Registry) error {
	// Read the type ID
	typeID := r.ReadTypeID()
	if r.Err() != nil {
		return r.Err()
	}

	if typeID == TypeIDNil {
		v.SetZero()
		return nil
	}

	// Look up the type in the registry
	registration, ok := reg.Lookup(typeID)
	if !ok {
		return NewDecodeError("unknown type ID: "+typeID.String(), ErrUnknownType)
	}

	// Create a new instance of the concrete type
	newVal := reflect.New(registration.Type)

	// Decode into the new value
	if err := decodeValue(r, newVal.Elem()); err != nil {
		return err
	}

	// Set the interface value
	v.Set(newVal)
	return r.Err()
}

// Size returns the encoded size of a value without actually encoding it.
// This can be used to pre-allocate buffers.
func Size(v any) int {
	return SizeWithOptions(v, DefaultOptions)
}

// SizeWithOptions returns the encoded size with the specified options.
func SizeWithOptions(v any, opts Options) int {
	return sizeValue(reflect.ValueOf(v), opts)
}

// sizeValue calculates the encoded size of a reflect.Value.
func sizeValue(v reflect.Value, opts Options) int {
	// Handle nil interface or invalid values
	if !v.IsValid() {
		return 1 // nil marker
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return 1 // nil marker
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Bool:
		return BoolSize
	case reflect.Int8, reflect.Uint8:
		return 1
	case reflect.Int16:
		return SizeOfInt16(int16(v.Int()))
	case reflect.Int32:
		return SizeOfInt32(int32(v.Int()))
	case reflect.Int64, reflect.Int:
		return SizeOfInt64(v.Int())
	case reflect.Uint16:
		return SizeOfUint16(uint16(v.Uint()))
	case reflect.Uint32:
		return SizeOfUint32(uint32(v.Uint()))
	case reflect.Uint64, reflect.Uint, reflect.Uintptr:
		return SizeOfUint64(v.Uint())
	case reflect.Float32:
		return Float32Size
	case reflect.Float64:
		return Float64Size
	case reflect.Complex64:
		return Complex64Size
	case reflect.Complex128:
		return Complex128Size
	case reflect.String:
		return SizeOfString(v.String())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return SizeOfBytes(v.Bytes())
		}
		return sizeSlice(v, opts)
	case reflect.Array:
		return sizeArray(v, opts)
	case reflect.Map:
		return sizeMap(v, opts)
	case reflect.Struct:
		return sizeStruct(v, opts)
	default:
		return 0
	}
}

func sizeSlice(v reflect.Value, opts Options) int {
	if v.IsNil() {
		return SizeOfUvarint(0)
	}
	n := v.Len()
	size := SizeOfUvarint(uint64(n))
	for i := 0; i < n; i++ {
		size += sizeValue(v.Index(i), opts)
	}
	return size
}

func sizeArray(v reflect.Value, opts Options) int {
	n := v.Len()
	size := SizeOfUvarint(uint64(n))
	for i := 0; i < n; i++ {
		size += sizeValue(v.Index(i), opts)
	}
	return size
}

func sizeMap(v reflect.Value, opts Options) int {
	if v.IsNil() {
		return SizeOfUvarint(0)
	}
	n := v.Len()
	size := SizeOfUvarint(uint64(n))
	iter := v.MapRange()
	for iter.Next() {
		size += sizeValue(iter.Key(), opts)
		size += sizeValue(iter.Value(), opts)
	}
	return size
}

// sizeStruct calculates the encoded size of a struct.
func sizeStruct(v reflect.Value, opts Options) int {
	info := getStructInfo(v.Type())

	size := 0
	for _, field := range info.fields {
		fv := v.Field(field.index)
		if opts.OmitEmpty && isZeroValue(fv) {
			continue
		}
		// Compact tag size + value size
		size += CompactTagSize(field.num)
		size += sizeValue(fv, opts)
	}

	// Add end marker size (1 byte)
	size++

	return size
}
