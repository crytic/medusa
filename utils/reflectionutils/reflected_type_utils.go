package reflectionutils

import (
	"fmt"
	"github.com/crytic/medusa/logging"
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
// This method panics if an array, slice, or struct type is not provided.
// Returns the reflected copied value.
func CopyReflectedType(reflectedValue reflect.Value) reflect.Value {
	switch reflectedValue.Kind() {
	case reflect.Slice:
		elementType := reflectedValue.Type().Elem()
		newSlice := reflect.MakeSlice(reflect.SliceOf(elementType), reflectedValue.Len(), reflectedValue.Cap())
		for i := 0; i < reflectedValue.Len(); i++ {
			newSlice.Index(i).Set(reflect.ValueOf(reflectedValue.Index(i).Interface()))
		}
		return newSlice
	case reflect.Array:
		arrayType := reflect.ArrayOf(reflectedValue.Len(), reflectedValue.Type().Elem())
		newArray := reflect.New(arrayType).Elem()
		for i := 0; i < reflectedValue.Len(); i++ {
			newArray.Index(i).Set(reflect.ValueOf(reflectedValue.Index(i).Interface()))
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

	logging.GlobalLogger.Panic("Failed to copy reflected value", fmt.Errorf("type not supported"))
	return reflectedValue
}

// GetReflectedArrayValues obtains the values of each element of a reflected array or slice variable.
// This method panics if an array or slice type is not provided.
// Returns a slice containing all values of each element in the provided array or slice.
func GetReflectedArrayValues(reflectedArray reflect.Value) []any {
	switch reflectedArray.Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		values := make([]any, reflectedArray.Len())
		for i := 0; i < len(values); i++ {
			values[i] = GetField(reflectedArray.Index(i))
		}
		return values
	}

	logging.GlobalLogger.Panic("Failed to get reflected array values", fmt.Errorf("type not supported"))
	return nil
}

// SetReflectedArrayValues takes an array or slice of the same length as the values provided, and sets each element
// to the corresponding element of the values provided.
// Returns an error if one occurred during value setting.
func SetReflectedArrayValues(reflectedArray reflect.Value, values []any) error {
	switch reflectedArray.Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		// Validate the length of our array is equal to the length of values provided.
		if reflectedArray.Len() != len(values) {
			return fmt.Errorf("failed to set reflected array values, a slice/array of length %v was provided, while %v values were provided", reflectedArray.Len(), len(values))
		}

		// Set each element in the array to the corresponding element of the values provided.
		for i := 0; i < len(values); i++ {
			SetField(reflectedArray.Index(i), values[i])
		}
		return nil
	}

	logging.GlobalLogger.Panic("Failed to set reflected array values", fmt.Errorf("type not supported"))
	return nil
}
