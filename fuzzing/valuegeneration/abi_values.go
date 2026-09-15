package valuegeneration

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/crytic/medusa/logging"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa/utils/reflectionutils"
)

// GenerateAbiValue generates a value of the provided abi.Type using the provided ValueGenerator.
// The generated value is returned.
func GenerateAbiValue(generator ValueGenerator, inputType *abi.Type) any {
	// Determine the type of value to generate based on the ABI type.
	switch inputType.T {
	case abi.AddressTy:
		return generator.GenerateAddress()
	case abi.UintTy:
		switch inputType.Size {
		case 64:
			return generator.GenerateInteger(false, inputType.Size).Uint64()
		case 32:
			return uint32(generator.GenerateInteger(false, inputType.Size).Uint64())
		case 16:
			return uint16(generator.GenerateInteger(false, inputType.Size).Uint64())
		case 8:
			return uint8(generator.GenerateInteger(false, inputType.Size).Uint64())
		default:
			return generator.GenerateInteger(false, inputType.Size)
		}
	case abi.IntTy:
		switch inputType.Size {
		case 64:
			return generator.GenerateInteger(true, inputType.Size).Int64()
		case 32:
			return int32(generator.GenerateInteger(true, inputType.Size).Int64())
		case 16:
			return int16(generator.GenerateInteger(true, inputType.Size).Int64())
		case 8:
			return int8(generator.GenerateInteger(true, inputType.Size).Int64())
		default:
			return generator.GenerateInteger(true, inputType.Size)
		}
	case abi.BoolTy:
		return generator.GenerateBool()
	case abi.StringTy:
		return generator.GenerateString()
	case abi.BytesTy:
		return generator.GenerateBytes()
	case abi.FixedBytesTy:
		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		// We opt to keep our API for generators simple, creating the array here and copying elements from a slice.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		bytes := reflect.ValueOf(generator.GenerateFixedBytes(inputType.Size))
		for i := 0; i < array.Len(); i++ {
			array.Index(i).Set(bytes.Index(i))
		}
		return array.Interface()
	case abi.ArrayTy:
		// Read notes for fixed bytes to understand the need to create this array through reflection.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < array.Len(); i++ {
			array.Index(i).Set(reflect.ValueOf(GenerateAbiValue(generator, inputType.Elem)))
		}
		return array.Interface()
	case abi.SliceTy:
		// Dynamic sized arrays are represented as slices.
		sliceSize := generator.GenerateArrayOfLength()
		slice := reflect.MakeSlice(inputType.GetType(), sliceSize, sliceSize)
		for i := 0; i < slice.Len(); i++ {
			slice.Index(i).Set(reflect.ValueOf(GenerateAbiValue(generator, inputType.Elem)))
		}
		return slice.Interface()
	case abi.TupleTy:
		// Tuples are used to represent structs. For go-ethereum's ABI provider, we're intended to supply matching
		// struct implementations, so we create and populate them through reflection.
		st := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < len(inputType.TupleElems); i++ {
			field := st.Field(i)
			fieldValue := GenerateAbiValue(generator, inputType.TupleElems[i])
			reflectionutils.SetField(field, fieldValue)
		}
		return st.Interface()
	default:
		// Unexpected types will result in a panic as we should support these values as soon as possible:
		// - Mappings cannot be used in public/external methods and must reference storage, so we shouldn't ever
		//	 see cases of it unless Solidity was updated in the future.
		// - FixedPoint types are currently unsupported.

		err := fmt.Errorf("attempt to generate function argument of unsupported type: '%s'", inputType.String())
		logging.GlobalLogger.Panic("Failed to generate abi value", err)
		return nil
	}
}

// MutateAbiValue takes an ABI packable input value, alongside its type definition and a value generator, to mutate
// existing ABI input values.
func MutateAbiValue(generator ValueGenerator, mutator ValueMutator, inputType *abi.Type, value any) (any, error) {
	// Switch on the type of value and mutate it recursively.
	switch inputType.T {
	case abi.AddressTy:
		addr, ok := value.(common.Address)
		if !ok {
			return nil, fmt.Errorf("could not mutate address input as the value provided is not an address type")
		}
		return mutator.MutateAddress(addr), nil
	case abi.UintTy:
		switch inputType.Size {
		case 64:
			v, ok := value.(uint64)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return mutator.MutateInteger(new(big.Int).SetUint64(v), false, inputType.Size).Uint64(), nil
		case 32:
			v, ok := value.(uint32)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return uint32(mutator.MutateInteger(new(big.Int).SetUint64(uint64(v)), false, inputType.Size).Uint64()), nil
		case 16:
			v, ok := value.(uint16)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return uint16(mutator.MutateInteger(new(big.Int).SetUint64(uint64(v)), false, inputType.Size).Uint64()), nil
		case 8:
			v, ok := value.(uint8)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return uint8(mutator.MutateInteger(new(big.Int).SetUint64(uint64(v)), false, inputType.Size).Uint64()), nil
		default:
			v, ok := value.(*big.Int)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return mutator.MutateInteger(new(big.Int).Set(v), false, inputType.Size), nil
		}
	case abi.IntTy:
		switch inputType.Size {
		case 64:
			v, ok := value.(int64)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return mutator.MutateInteger(new(big.Int).SetInt64(v), true, inputType.Size).Int64(), nil
		case 32:
			v, ok := value.(int32)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return int32(mutator.MutateInteger(new(big.Int).SetInt64(int64(v)), true, inputType.Size).Int64()), nil
		case 16:
			v, ok := value.(int16)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return int16(mutator.MutateInteger(new(big.Int).SetInt64(int64(v)), true, inputType.Size).Int64()), nil
		case 8:
			v, ok := value.(int8)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return int8(mutator.MutateInteger(new(big.Int).SetInt64(int64(v)), true, inputType.Size).Int64()), nil
		default:
			v, ok := value.(*big.Int)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return mutator.MutateInteger(new(big.Int).Set(v), true, inputType.Size), nil
		}
	case abi.BoolTy:
		v, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("could not mutate boolean input as the value provided is not a boolean type")
		}
		return mutator.MutateBool(v), nil
	case abi.StringTy:
		v, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("could not mutate string input as the value provided is not a string type")
		}
		return mutator.MutateString(v), nil
	case abi.BytesTy:
		v, ok := value.([]byte)
		if !ok {
			return nil, fmt.Errorf("could not mutate dynamic-sized bytes input as the value provided is not a byte slice type")
		}
		return mutator.MutateBytes(v), nil
	case abi.FixedBytesTy:
		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		// We opt to keep our API for generators simple, creating the array here and copying elements from a slice.
		valueAsSlice := reflectionutils.ArrayToSlice(reflect.ValueOf(value)).([]byte)
		mutatedValue := mutator.MutateFixedBytes(valueAsSlice)
		mutatedValueAsArray := reflectionutils.SliceToArray(reflect.ValueOf(mutatedValue))
		mutatedValueAsArrayLen := reflect.ValueOf(mutatedValueAsArray).Len()
		if mutatedValueAsArrayLen != inputType.Size {
			return nil, fmt.Errorf("could not mutate fixed-sized bytes input as the mutated value returned was not of the correct length. expected %v, got %v", inputType.Size, mutatedValueAsArrayLen)
		}
		return mutatedValueAsArray, nil
	case abi.ArrayTy:
		// Look through our array, recursively mutate each element, and set the result in the array.
		// Note: We create a copy, as existing arrays may not be assignable.
		array := reflectionutils.CopyReflectedType(reflect.ValueOf(value))

		// Mutate our array structure first
		mutatedValues := mutator.MutateArray(reflectionutils.GetReflectedArrayValues(array), true)

		// Create a new array of the appropriate size
		array = reflect.New(reflect.ArrayOf(array.Len(), array.Type().Elem())).Elem()

		// Next mutate each element in the array.
		for i := 0; i < array.Len(); i++ {
			// Obtain the element's reflected value to access its getter/setters
			reflectedElement := array.Index(i)

			// If any item is nil, we generate a new element in its place instead. Otherwise, we mutate the existing value.
			if mutatedValues[i] == nil {
				generatedElement := GenerateAbiValue(generator, inputType.Elem)
				reflectedElement.Set(reflect.ValueOf(generatedElement))
			} else {
				mutatedElement, err := MutateAbiValue(generator, mutator, inputType.Elem, mutatedValues[i])
				if err != nil {
					return nil, fmt.Errorf("could not mutate array input as the value generator encountered an error: %v", err)
				}
				reflectedElement.Set(reflect.ValueOf(mutatedElement))
			}
		}

		return array.Interface(), nil
	case abi.SliceTy:
		// Dynamic sized arrays are represented as slices.
		// Note: We create a copy, as existing slices may not be assignable.
		slice := reflectionutils.CopyReflectedType(reflect.ValueOf(value))

		// Mutate our slice structure first
		mutatedValues := mutator.MutateArray(reflectionutils.GetReflectedArrayValues(slice), false)

		// Create a new slice of the appropriate size
		slice = reflect.MakeSlice(reflect.SliceOf(slice.Type().Elem()), len(mutatedValues), len(mutatedValues))

		// Next mutate each element in the slice.
		for i := 0; i < slice.Len(); i++ {
			// Obtain the element's reflected value to access its getter/setters
			reflectedElement := slice.Index(i)

			// If any item is nil, we generate a new element in its place instead. Otherwise, we mutate the existing value.
			if mutatedValues[i] == nil {
				generatedElement := GenerateAbiValue(generator, inputType.Elem)
				reflectedElement.Set(reflect.ValueOf(generatedElement))
			} else {
				mutatedElement, err := MutateAbiValue(generator, mutator, inputType.Elem, mutatedValues[i])
				if err != nil {
					return nil, fmt.Errorf("could not mutate slice input as the value generator encountered an error: %v", err)
				}
				reflectedElement.Set(reflect.ValueOf(mutatedElement))
			}
		}
		return slice.Interface(), nil
	case abi.TupleTy:
		// Structs are used to represent tuples.
		// Note: We create a copy, as existing tuples may not be assignable.
		tuple := reflectionutils.CopyReflectedType(reflect.ValueOf(value))
		for i := 0; i < len(inputType.TupleElems); i++ {
			field := tuple.Field(i)
			fieldValue := reflectionutils.GetField(field)
			mutatedValue, err := MutateAbiValue(generator, mutator, inputType.TupleElems[i], fieldValue)
			if err != nil {
				return nil, fmt.Errorf("could not mutate struct/tuple input as the value generator encountered an error: %v", err)
			}
			reflectionutils.SetField(field, mutatedValue)
		}
		return tuple.Interface(), nil
	default:
		return nil, fmt.Errorf("could not mutate argument, type is unsupported: %v", inputType)
	}
}
