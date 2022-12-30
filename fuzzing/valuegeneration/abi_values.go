package valuegeneration

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"reflect"
)

// GenerateAbiValue generates a value of the provided abi.Type using the provided ValueGenerator.
// The generated value is returned.
func GenerateAbiValue(generator ValueGenerator, inputType *abi.Type) any {
	// Determine the type of value to generate based on the ABI type.
	if inputType.T == abi.AddressTy {
		return generator.GenerateAddress()
	} else if inputType.T == abi.UintTy {
		if inputType.Size == 64 {
			return generator.GenerateInteger(false, inputType.Size).Uint64()
		} else if inputType.Size == 32 {
			return uint32(generator.GenerateInteger(false, inputType.Size).Uint64())
		} else if inputType.Size == 16 {
			return uint16(generator.GenerateInteger(false, inputType.Size).Uint64())
		} else if inputType.Size == 8 {
			return uint8(generator.GenerateInteger(false, inputType.Size).Uint64())
		} else {
			return generator.GenerateInteger(false, inputType.Size)
		}
	} else if inputType.T == abi.IntTy {
		if inputType.Size == 64 {
			return generator.GenerateInteger(true, inputType.Size).Int64()
		} else if inputType.Size == 32 {
			return int32(generator.GenerateInteger(true, inputType.Size).Int64())
		} else if inputType.Size == 16 {
			return int16(generator.GenerateInteger(true, inputType.Size).Int64())
		} else if inputType.Size == 8 {
			return int8(generator.GenerateInteger(true, inputType.Size).Int64())
		} else {
			return generator.GenerateInteger(true, inputType.Size)
		}
	} else if inputType.T == abi.BoolTy {
		return generator.GenerateBool()
	} else if inputType.T == abi.StringTy {
		return generator.GenerateString()
	} else if inputType.T == abi.BytesTy {
		return generator.GenerateBytes()
	} else if inputType.T == abi.FixedBytesTy {
		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		// We opt to keep our API for generators simple, creating the array here and copying elements from a slice.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		bytes := reflect.ValueOf(generator.GenerateFixedBytes(inputType.Size))
		for i := 0; i < array.Len(); i++ {
			array.Index(i).Set(bytes.Index(i))
		}
		return array.Interface()
	} else if inputType.T == abi.ArrayTy {
		// Read notes for fixed bytes to understand the need to create this array through reflection.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < array.Len(); i++ {
			array.Index(i).Set(reflect.ValueOf(GenerateAbiValue(generator, inputType.Elem)))
		}
		return array.Interface()
	} else if inputType.T == abi.SliceTy {
		// Dynamic sized arrays are represented as slices.
		sliceSize := generator.GenerateArrayLength()
		slice := reflect.MakeSlice(inputType.GetType(), sliceSize, sliceSize)
		for i := 0; i < slice.Len(); i++ {
			slice.Index(i).Set(reflect.ValueOf(GenerateAbiValue(generator, inputType.Elem)))
		}
		return slice.Interface()
	} else if inputType.T == abi.TupleTy {
		// Tuples are used to represent structs. For go-ethereum's ABI provider, we're intended to supply matching
		// struct implementations, so we create and populate them through reflection.
		st := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < len(inputType.TupleElems); i++ {
			st.Field(i).Set(reflect.ValueOf(GenerateAbiValue(generator, inputType.TupleElems[i])))
		}
		return st.Interface()
	}

	// Unexpected types will result in a panic as we should support these values as soon as possible:
	// - Mappings cannot be used in public/external methods and must reference storage, so we shouldn't ever
	//	 see cases of it unless Solidity was updated in the future.
	// - FixedPoint types are currently unsupported.
	panic(fmt.Sprintf("attempt to generate function argument of unsupported type: '%s'", inputType.String()))
}
