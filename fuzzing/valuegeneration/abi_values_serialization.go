package valuegeneration

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/trailofbits/medusa/utils/reflectionutils"
	"math/big"
	"reflect"
)

const (
	// ArgumentValueTypeAddress describes an Ethereum address value.
	ArgumentValueTypeAddress    = "address"
	ArgumentValueTypeInteger    = "integer"
	ArgumentValueTypeBool       = "bool"
	ArgumentValueTypeString     = "string"
	ArgumentValueTypeBytes      = "bytes"
	ArgumentValueTypeFixedBytes = "bytesN"
	ArgumentValueTypeArray      = "array"
	ArgumentValueTypeSlice      = "slice"
	ArgumentValueTypeTuple      = "tuple"
)

// AbiValueToMap takes a given value and ABI value type definition and encodes the value into a dictionary.
func AbiValueToMap(valueType *abi.Type, value any) map[string]any {
	valueData := make(map[string]any, 0)

	// Determine the type of value to generate based on the ABI type.
	if valueType.T == abi.AddressTy {
		valueData["type"] = ArgumentValueTypeAddress
		valueData["value"] = value.(common.Address)
	} else if valueType.T == abi.UintTy || valueType.T == abi.IntTy {
		valueData["type"] = ArgumentValueTypeInteger
		valueData["unsigned"] = valueType.T == abi.UintTy
		valueData["size"] = valueType.Size
		if valueType.Size <= 64 {
			valueData["value"] = value // smaller integers use primitive types
		} else {
			valueData["value"] = (*hexutil.Big)(value.(*big.Int)) // larger integers use big.Int
		}
	} else if valueType.T == abi.BoolTy {
		valueData["type"] = ArgumentValueTypeBool
		valueData["value"] = value
	} else if valueType.T == abi.StringTy {
		valueData["type"] = ArgumentValueTypeString
		valueData["value"] = value
	} else if valueType.T == abi.BytesTy {
		valueData["type"] = ArgumentValueTypeBytes
		valueData["value"] = (hexutil.Bytes)(value.([]byte))
	} else if valueType.T == abi.FixedBytesTy {
		valueData["type"] = ArgumentValueTypeFixedBytes
		valueData["size"] = valueType.Size
		valueData["value"] = (hexutil.Bytes)(reflectionutils.ArrayToSlice(reflect.ValueOf(value)).([]byte))
	} else if valueType.T == abi.ArrayTy {
		valueData["type"] = ArgumentValueTypeArray
		valueData["size"] = valueType.Size

		// Convert all underlying elements in our array
		reflectedArray := reflect.ValueOf(value)
		arrayData := make([]any, 0)
		for i := 0; i < reflectedArray.Len(); i++ {
			elementData := AbiValueToMap(valueType.Elem, reflectedArray.Index(i).Interface())
			arrayData = append(arrayData, elementData)
		}
		valueData["values"] = arrayData
	} else if valueType.T == abi.SliceTy {
		valueData["type"] = ArgumentValueTypeSlice

		// Convert all underlying elements in our slice
		reflectedArray := reflect.ValueOf(value)
		sliceData := make([]any, 0)
		for i := 0; i < reflectedArray.Len(); i++ {
			elementData := AbiValueToMap(valueType.Elem, reflectedArray.Index(i).Interface())
			sliceData = append(sliceData, elementData)
		}
		valueData["value"] = sliceData
	} else if valueType.T == abi.TupleTy {
		valueData["type"] = ArgumentValueTypeTuple

		// Convert all underlying elements in our array
		reflectedTuple := reflect.ValueOf(value)
		tupleData := make([]any, 0)
		for i := 0; i < len(valueType.TupleElems); i++ {
			fieldData := AbiValueToMap(valueType.TupleElems[i], reflectionutils.GetField(reflectedTuple.Field(i)))
			tupleData = append(tupleData, fieldData)
		}
		valueData["value"] = tupleData
	} else {
		// Unexpected types will result in a panic as we should support these values as soon as possible:
		// - Mappings cannot be used in public/external methods and must reference storage, so we shouldn't ever
		//	 see cases of it unless Solidity is updated in the future.
		// - FixedPoint types are currently unsupported.
		panic(fmt.Sprintf("attempt to generate function argument of unsupported type: '%s'", valueType.String()))
	}
	return valueData
}

// AbiValueFromMap takes an ABI value type definition and decodes the value encoded in the provided map.
func AbiValueFromMap(valueType *abi.Type, valueData map[string]any) (any, error) {
	errValueTypeMismatchString := errors.New("failed to decode ABI value, the 'type' key did not match the compiled definition")
	errValueSizeMismatchString := errors.New("failed to decode ABI value, the 'size' key did not match the compiled definition")

	// Every value has a 'type' and 'value' key.
	rawValue, ok := valueData["type"]
	if !ok {
		return nil, errors.New("failed to decode ABI value, the 'type' key did not exist")
	}
	valueDataType, ok := rawValue.(string)
	if !ok {
		return nil, errors.New("failed to decode ABI value, the 'type' key was the wrong type")
	}
	valueDataValue, ok := valueData["value"]
	if !ok {
		return nil, errors.New("failed to decode ABI value, the 'value' key did not exist")
	}

	// Determine the type of value to generate based on the ABI type.
	if valueType.T == abi.AddressTy {
		if valueDataType != ArgumentValueTypeAddress {
			return nil, errValueTypeMismatchString
		}
		return valueDataValue.(common.Address), nil
	} else if valueType.T == abi.UintTy || valueType.T == abi.IntTy {
		if valueDataType != ArgumentValueTypeInteger {
			return nil, errValueTypeMismatchString
		}
		valueDataUnsigned := valueData["unsigned"].(bool)
		if valueDataUnsigned != (valueType.T == abi.UintTy) {
			return nil, errors.New("failed to decode ABI value, the 'unsigned' key for an integer type did not match the compiled definition")
		}
		valueDataSize := valueData["size"].(int)
		if valueDataSize != valueType.Size {
			return nil, errValueSizeMismatchString
		}
		if valueType.T == abi.UintTy {
			if valueType.Size == 64 {
				return valueDataValue.(uint64), nil
			} else if valueType.Size == 32 {
				return valueDataValue.(uint32), nil
			} else if valueType.Size == 16 {
				return valueDataValue.(uint16), nil
			} else if valueType.Size == 8 {
				return valueDataValue.(uint8), nil
			}
		} else if valueType.T == abi.IntTy {
			if valueType.Size == 64 {
				return valueDataValue.(int64), nil
			} else if valueType.Size == 32 {
				return valueDataValue.(int32), nil
			} else if valueType.Size == 16 {
				return valueDataValue.(int16), nil
			} else if valueType.Size == 8 {
				return valueDataValue.(int8), nil
			}
		}
		return (*big.Int)(valueDataValue.(*hexutil.Big)), nil
	} else if valueType.T == abi.BoolTy {
		if valueDataType != ArgumentValueTypeBool {
			return nil, errValueTypeMismatchString
		}
		return valueDataValue.(bool), nil
	} else if valueType.T == abi.StringTy {
		if valueDataType != ArgumentValueTypeString {
			return nil, errValueTypeMismatchString
		}
		return valueDataValue.(string), nil
	} else if valueType.T == abi.BytesTy {
		if valueDataType != ArgumentValueTypeBytes {
			return nil, errValueTypeMismatchString
		}
		return ([]byte)(valueDataValue.(hexutil.Bytes)), nil
	} else if valueType.T == abi.FixedBytesTy {
		if valueDataType != ArgumentValueTypeFixedBytes {
			return nil, errValueTypeMismatchString
		}
		valueDataSize := valueData["size"].(int)
		if valueDataSize != valueType.Size {
			return nil, errValueSizeMismatchString
		}
		valueDataValue = ([]byte)(reflectionutils.SliceToArray(reflect.ValueOf(valueDataValue.(hexutil.Bytes))).(hexutil.Bytes))
		return valueDataValue, nil
	} else if valueType.T == abi.ArrayTy {
		if valueDataType != ArgumentValueTypeArray {
			return nil, errValueTypeMismatchString
		}
		valueData["size"] = valueType.Size

		// Create an array of the defined type
		array := reflect.Indirect(reflect.New(valueType.GetType()))
		valueDataValueSlice := valueDataValue.([]any)
		for i := 0; i < array.Len(); i++ {
			elementValue, err := AbiValueFromMap(valueType.Elem, valueDataValueSlice[i].(map[string]any))
			if err != nil {
				return nil, err
			}
			array.Index(i).Set(reflect.ValueOf(elementValue))
		}
		return array.Interface(), nil
	} else if valueType.T == abi.SliceTy {
		if valueDataType != ArgumentValueTypeSlice {
			return nil, errValueTypeMismatchString
		}

		// Convert all underlying elements in our slice
		valueDataValueSlice := valueDataValue.([]any)
		slice := reflect.MakeSlice(valueType.GetType(), len(valueDataValueSlice), len(valueDataValueSlice))
		for i := 0; i < slice.Len(); i++ {
			elementValue, err := AbiValueFromMap(valueType.Elem, valueDataValueSlice[i].(map[string]any))
			if err != nil {
				return nil, err
			}
			slice.Index(i).Set(reflect.ValueOf(elementValue))
		}
		return slice.Interface(), nil
	} else if valueType.T == abi.TupleTy {
		if valueDataType != ArgumentValueTypeTuple {
			return nil, errValueTypeMismatchString
		}

		// Tuples are used to represent structs. For go-ethereum's ABI provider, we're intended to supply matching
		// struct implementations, so we create and populate them through reflection.
		valueDataValueSlice := valueDataValue.([]any)
		st := reflect.Indirect(reflect.New(valueType.GetType()))
		for i := 0; i < len(valueType.TupleElems); i++ {
			elementValue, err := AbiValueFromMap(valueType.TupleElems[i], valueDataValueSlice[i].(map[string]any))
			if err != nil {
				return nil, err
			}
			st.Field(i).Set(reflect.ValueOf(elementValue))
		}
		return st.Interface(), nil
	}

	// Unexpected types will result in a panic as we should support these values as soon as possible:
	// - Mappings cannot be used in public/external methods and must reference storage, so we shouldn't ever
	//	 see cases of it unless Solidity is updated in the future.
	// - FixedPoint types are currently unsupported.
	panic(fmt.Sprintf("attempt to generate function argument of unsupported type: '%s'", valueType.String()))
}
