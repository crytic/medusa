package reflectionutils

import (
	"reflect"
	"unsafe"
)

// GetField obtains a given field even if it is unexported.
func GetField(field reflect.Value) any {
	// If we can't grab a value, but we can address it, try to create a pointer to the field's data to fetch it.
	if !field.CanInterface() && field.CanAddr() {
		dataPointer := unsafe.Pointer(field.UnsafeAddr())
		return reflect.NewAt(field.Type(), dataPointer).Elem().Interface()
	}

	// Otherwise we try to simply fetch the data.
	return field.Interface()
}

// SetField sets a given field's value, even if it is unexported.
func SetField(field reflect.Value, value any) {
	// If this is an unexported field, we can create a new value that shares the same data pointer, and set that to
	// write to the data.
	if !field.CanSet() && field.CanAddr() {
		// Create a pointer to the field's data.
		dataPointer := unsafe.Pointer(field.UnsafeAddr())

		// Create a new value of the same type which shares the data pointer
		newValue := reflect.NewAt(field.Type(), dataPointer).Elem()

		// Now set the data for the new value to the provided value. This sets the data in the same memory location.
		newValue.Set(reflect.ValueOf(value))
		return
	}

	// Otherwise we try to simply fetch the data.
	field.Set(reflect.ValueOf(value))
}

// ArrayToSlice converts a reflected array into a slice.
// Returns the slice.
func ArrayToSlice(reflectedArray reflect.Value) any {
	sliceType := reflect.SliceOf(reflectedArray.Type().Elem())
	resultingSlice := reflect.MakeSlice(sliceType, reflectedArray.Len(), reflectedArray.Len())
	for i := 0; i < reflectedArray.Len(); i++ {
		resultingSlice.Index(i).Set(reflect.ValueOf(reflectedArray.Index(i).Interface()))
	}
	return resultingSlice.Interface()
}

// SliceToArray converts a reflected slice into an array of the same size.
// Returns the array.
func SliceToArray(reflectedSlice reflect.Value) any {
	arrayType := reflect.ArrayOf(reflectedSlice.Len(), reflectedSlice.Type().Elem())
	resultingArray := reflect.New(arrayType).Elem()
	for i := 0; i < reflectedSlice.Len(); i++ {
		resultingArray.Index(i).Set(reflect.ValueOf(reflectedSlice.Index(i).Interface()))
	}
	return resultingArray.Interface()
}
