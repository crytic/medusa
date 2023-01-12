package reflectionutils

import (
	"reflect"
)

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

// CopyReflectedType creates a shallow copy of a reflected value. It supports slices, arrays, or structs.
// Returns the reflected copied value.
func CopyReflectedType(reflectedValue reflect.Value) reflect.Value {
	switch reflectedValue.Kind() {
	case reflect.Slice:
		elementType := reflectedValue.Type().Elem()
		newSlice := reflect.MakeSlice(reflect.SliceOf(elementType), reflectedValue.Len(), reflectedValue.Cap())
		if reflect.Copy(newSlice, reflectedValue) != reflectedValue.Len() {
			panic("failed to copy reflected value, unexpected resulting slice length")
		}
		return newSlice
	case reflect.Array:
		arrayType := reflect.ArrayOf(reflectedValue.Len(), reflectedValue.Type().Elem())
		newArray := reflect.New(arrayType).Elem()
		if reflect.Copy(newArray, reflectedValue) != reflectedValue.Len() {
			panic("failed to copy reflected value, unexpected resulting array length")
		}
		return newArray
	case reflect.Struct:
		newStruct := reflect.Indirect(reflect.New(reflectedValue.Type()))
		for i := 0; i < reflectedValue.NumField(); i++ {
			fieldValue := GetField(reflectedValue.Field(i))
			SetField(newStruct.Field(i), fieldValue)
		}
		return newStruct
	}
	panic("failed to copy reflected value, type not supported")
}
