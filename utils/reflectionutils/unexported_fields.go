package reflectionutils

import (
	"reflect"
	"unsafe"
)

// GetField obtains a given field even if it is unexported.
func GetField(field reflect.Value) any {
	// Create a pointer to the field's data.
	dataPointer := unsafe.Pointer(field.UnsafeAddr())
	return reflect.NewAt(field.Type(), dataPointer).Elem().Interface()
}

// SetField sets a given field's value, even if it is unexported.
func SetField(field reflect.Value, value any) {
	// If this is an exported field, we can set it directly.
	if field.CanSet() {
		field.Set(reflect.ValueOf(value))
	} else {
		// If it's not exported, we can create a new value that shares the same data pointer, and set that to write
		// to the data.

		// Create a pointer to the field's data.
		dataPointer := unsafe.Pointer(field.UnsafeAddr())

		// Create a new value of the same type which shares the data pointer
		newValue := reflect.NewAt(field.Type(), dataPointer).Elem()

		// Now set the data for the new value to the provided value. This sets the data in the same memory location.
		newValue.Set(reflect.ValueOf(value))
	}
}

func ArrayToSlice(reflectedArray reflect.Value) any {
	sliceType := reflect.SliceOf(reflectedArray.Type().Elem())
	resultingSlice := reflect.MakeSlice(sliceType, reflectedArray.Len(), reflectedArray.Len())
	for i := 0; i < reflectedArray.Len(); i++ {
		resultingSlice.Index(i).Set(reflect.ValueOf(reflectedArray.Index(i).Interface()))
	}
	return resultingSlice.Interface()
}

func SliceToArray(reflectedSlice reflect.Value) any {
	arrayType := reflect.ArrayOf(reflectedSlice.Len(), reflectedSlice.Type().Elem())
	resultingArray := reflect.New(arrayType).Elem()
	for i := 0; i < reflectedSlice.Len(); i++ {
		resultingArray.Index(i).Set(reflect.ValueOf(reflectedSlice.Index(i).Interface()))
	}
	return resultingArray.Interface()
}
