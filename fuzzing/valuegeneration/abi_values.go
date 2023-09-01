package valuegeneration

import (
	"encoding/hex"
	"fmt"
	"github.com/crytic/medusa/logging"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/crytic/medusa/utils/reflectionutils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// addressJSONContractNameOverridePrefix defines a string prefix which is to be followed by a contract name. The
// contract address will be resolved by searching the deployed contracts for a contract with this name.
const addressJSONContractNameOverridePrefix = "DeployedContract:"

// GenerateAbiValue generates a value of the provided abi.Type using the provided ValueGenerator.
// The generated value is returned.
func GenerateAbiValue(generator ValueGenerator, inputType *abi.Type) any {
	// Determine the type of value to generate based on the ABI type.
	switch inputType.T {
	case abi.AddressTy:
		return generator.GenerateAddress()
	case abi.UintTy:
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
	case abi.IntTy:
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
		if inputType.Size == 64 {
			v, ok := value.(uint64)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return mutator.MutateInteger(new(big.Int).SetUint64(v), false, inputType.Size).Uint64(), nil
		} else if inputType.Size == 32 {
			v, ok := value.(uint32)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return uint32(mutator.MutateInteger(new(big.Int).SetUint64(uint64(v)), false, inputType.Size).Uint64()), nil
		} else if inputType.Size == 16 {
			v, ok := value.(uint16)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return uint16(mutator.MutateInteger(new(big.Int).SetUint64(uint64(v)), false, inputType.Size).Uint64()), nil
		} else if inputType.Size == 8 {
			v, ok := value.(uint8)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return uint8(mutator.MutateInteger(new(big.Int).SetUint64(uint64(v)), false, inputType.Size).Uint64()), nil
		} else {
			v, ok := value.(*big.Int)
			if !ok {
				return nil, fmt.Errorf("could not mutate uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return mutator.MutateInteger(new(big.Int).Set(v), false, inputType.Size), nil
		}
	case abi.IntTy:
		if inputType.Size == 64 {
			v, ok := value.(int64)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return mutator.MutateInteger(new(big.Int).SetInt64(v), true, inputType.Size).Int64(), nil
		} else if inputType.Size == 32 {
			v, ok := value.(int32)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return int32(mutator.MutateInteger(new(big.Int).SetInt64(int64(v)), true, inputType.Size).Int64()), nil
		} else if inputType.Size == 16 {
			v, ok := value.(int16)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return int16(mutator.MutateInteger(new(big.Int).SetInt64(int64(v)), true, inputType.Size).Int64()), nil
		} else if inputType.Size == 8 {
			v, ok := value.(int8)
			if !ok {
				return nil, fmt.Errorf("could not mutate int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return int8(mutator.MutateInteger(new(big.Int).SetInt64(int64(v)), true, inputType.Size).Int64()), nil
		} else {
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

// EncodeJSONArgumentsToMap encodes provided go-ethereum ABI packable input values into a generic JSON type values
// (e.g. []any, map[string]any, etc).
// Returns the encoded values, or an error if one occurs.
func EncodeJSONArgumentsToMap(inputs abi.Arguments, values []any) (map[string]any, error) {
	// Create a variable to store encoded arguments, fill it with the respective encoded arguments.
	var encodedArgs = make(map[string]any)
	for i, input := range inputs {
		arg, err := encodeJSONArgument(&input.Type, values[i])
		if err != nil {
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, values[i], err)
			return nil, err
		}
		encodedArgs[input.Name] = arg
	}
	return encodedArgs, nil
}

// EncodeJSONArgumentsToSlice encodes provided go-ethereum ABI packable input values into generic JSON compatible values
// (e.g. []any, map[string]any, etc).
// Returns the encoded values, or an error if one occurs.
func EncodeJSONArgumentsToSlice(inputs abi.Arguments, values []any) ([]any, error) {
	// Create a variable to store encoded arguments, fill it with the respective encoded arguments.
	var encodedArgs = make([]any, len(inputs))
	for i, input := range inputs {
		arg, err := encodeJSONArgument(&input.Type, values[i])
		if err != nil {
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, values[i], err)
			return nil, err
		}
		encodedArgs[i] = arg
	}
	return encodedArgs, nil
}

// EncodeABIArgumentsToString encodes provided go-ethereum ABI package input values into string that is
// human-readable for console output purpose.
// Returns the string, or an error if one occurs.
func EncodeABIArgumentsToString(inputs abi.Arguments, values []any) (string, error) {
	// Create a variable to store string arguments, fill it with the respective arguments
	var encodedArgs = make([]string, len(inputs))

	// Iterate over inputs
	for i, input := range inputs {
		// Encode the input value of a given type
		arg, err := encodeABIArgumentToString(&input.Type, values[i])
		if err != nil {
			// If error occurs while encoding the input value, return error message
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, values[i], err)
			return "", err
		}
		// Store the encoded argument at the current index in the encodedArgs slice
		encodedArgs[i] = arg
	}
	// Join the encoded arguments with a ", " separator
	return strings.Join(encodedArgs, ", "), nil
}

// encodeABIArgumentToString encodes a provided go-ethereum ABI packable input value of a given type, into
// a human-readable string format, depending on the input's type.
// Returns the string, or an error if one occurs.
func encodeABIArgumentToString(inputType *abi.Type, value any) (string, error) {
	// Switch on the type of the input argument to determine how to encode it
	switch inputType.T {
	case abi.AddressTy:
		// Prepare an address type. Return as a lowercase string without "".
		addr, ok := value.(common.Address)
		if !ok {
			return "", fmt.Errorf("could not encode address input as the value provided is not an address type")
		}
		return strings.ToLower(addr.String()), nil
	case abi.UintTy:
		// Prepare uint type. Return as a string without "".
		switch inputType.Size {
		case 64:
			v, ok := value.(uint64)
			if !ok {
				return "", fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(v, 10), nil
		case 32:
			v, ok := value.(uint32)
			if !ok {
				return "", fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		case 16:
			v, ok := value.(uint16)
			if !ok {
				return "", fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		case 8:
			v, ok := value.(uint8)
			if !ok {
				return "", fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		default:
			v, ok := value.(*big.Int)
			if !ok {
				return "", fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return v.String(), nil
		}
	case abi.IntTy:
		// Prepare int type. Return as a string without "".
		switch inputType.Size {
		case 64:
			v, ok := value.(int64)
			if !ok {
				return "", fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(v, 10), nil
		case 32:
			v, ok := value.(int32)
			if !ok {
				return "", fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		case 16:
			v, ok := value.(int16)
			if !ok {
				return "", fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		case 8:
			v, ok := value.(int8)
			if !ok {
				return "", fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		default:
			v, ok := value.(*big.Int)
			if !ok {
				return "", fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return v.String(), nil
		}
	case abi.BoolTy:
		// Return a bool type. Return as a string without "".
		b, ok := value.(bool)
		if !ok {
			return "", fmt.Errorf("could not encode bool as the value provided is not of the correct type")
		}
		return strconv.FormatBool(b), nil
	case abi.StringTy:
		// Prepare a string type. Return string is enclosed with "". The returned string uses Go escape
		// sequences (\t, \n, \xFF, \u0100) for non-ASCII characters and non-printable characters.
		str, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("could not encode string as the value provided is not of the correct type")
		}
		return strconv.QuoteToASCII(str), nil
	case abi.BytesTy:
		b, ok := value.([]byte)
		if !ok {
			return "", fmt.Errorf("could not encode dynamic-sized bytes as the value provided is not of the correct type")
		}
		// Convert the fixed byte array to a hex string
		return hex.EncodeToString(b), nil
	case abi.FixedBytesTy:
		// TODO: Error checking to ensure `value` is of the correct type.
		b := reflectionutils.ArrayToSlice(reflect.ValueOf(value)).([]byte)
		// Convert the byte array to a hex string
		return hex.EncodeToString(b), nil
	case abi.ArrayTy:
		// Prepare an array. Return as a string enclosed with [], where specific elements are comma-separated.
		reflectedArray := reflect.ValueOf(value)
		// Initialize an empty array to store the encoded elements
		arrayData := make([]string, 0)
		// Iterate through the elements of the input array
		for i := 0; i < reflectedArray.Len(); i++ {
			// Encode the element of a given type at the current index
			elementData, err := encodeABIArgumentToString(inputType.Elem, reflectedArray.Index(i).Interface())
			if err != nil {
				return "", err
			}
			// Append the encoded element to the arrayData
			arrayData = append(arrayData, elementData)
		}
		// Join the elements of the arrayData with ", " and enclose it with "[]"
		str := "[" + strings.Join(arrayData, ", ") + "]"
		return str, nil
	case abi.SliceTy:
		// Prepare a dynamic array. Return as a string enclosed with [], where specific elements are comma-separated.
		reflectedArray := reflect.ValueOf(value)
		// Initialize an empty slice to store the encoded elements
		sliceData := make([]string, 0)
		// Iterate through the elements of the input slice
		for i := 0; i < reflectedArray.Len(); i++ {
			// Encode the element of a given type at the current index
			elementData, err := encodeABIArgumentToString(inputType.Elem, reflectedArray.Index(i).Interface())
			if err != nil {
				return "", err
			}
			// Append the encoded element to the sliceData
			sliceData = append(sliceData, elementData)
		}
		// Join the elements of the sliceData with ", " and enclose it with "[]"
		str := "[" + strings.Join(sliceData, ", ") + "]"
		return str, nil
	case abi.TupleTy:
		// Prepare a tuple/struct. Return as a string enclosed with {}, where specific elements are presented
		// as a `key: value` and comma-separated.

		// Initialize an array to store our string representations of each tuple/struct field.
		tupleData := make([]string, 0)

		// Get the reflected value of the input tuple/struct
		reflectedTuple := reflect.ValueOf(value)
		// Iterate through the elements of the input tuple/struct
		for i := 0; i < len(inputType.TupleElems); i++ {
			// Get the field of the tuple/struct at the current index
			field := reflectedTuple.Field(i)
			// Get the value of the field
			fieldValue := reflectionutils.GetField(field)
			// Encode the field value of a given type
			fieldData, err := encodeABIArgumentToString(inputType.TupleElems[i], fieldValue)
			// Check if there is an error while encoding the field value
			if err != nil {
				return "", err
			}

			// Append the key-value pair in the format "key: value" to our tuple dat
			tupleData = append(tupleData, fmt.Sprintf("%v: %v", inputType.TupleRawNames[i], fieldData))
		}

		// Join the tuple string elements and close them in braces.
		str := "{" + strings.Join(tupleData, ", ") + "}"
		return str, nil
	default:
		return "", fmt.Errorf("could not encode argument as string, type is unsupported: %v", inputType)
	}
}

// encodeJSONArgument encodes a provided go-ethereum ABI packable input value of a given type, into generic JSON
// compatible values (e.g. []any, map[string]any, etc).
// Returns the encoded value, or an error if one occurs.
func encodeJSONArgument(inputType *abi.Type, value any) (any, error) {
	switch inputType.T {
	case abi.AddressTy:
		addr, ok := value.(common.Address)
		if !ok {
			return nil, fmt.Errorf("could not encode address input as the value provided is not an address type")
		}
		return addr.String(), nil
	case abi.UintTy:
		switch inputType.Size {
		case 64:
			v, ok := value.(uint64)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(v, 10), nil
		case 32:
			v, ok := value.(uint32)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		case 16:
			v, ok := value.(uint16)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		case 8:
			v, ok := value.(uint8)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		default:
			v, ok := value.(*big.Int)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return v.String(), nil
		}
	case abi.IntTy:
		switch inputType.Size {
		case 64:
			v, ok := value.(int64)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(v, 10), nil
		case 32:
			v, ok := value.(int32)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		case 16:
			v, ok := value.(int16)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		case 8:
			v, ok := value.(int8)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		default:
			v, ok := value.(*big.Int)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return v.String(), nil
		}
	case abi.BoolTy:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("could not encode bool as the value provided is not of the correct type")
		}
		return b, nil
	case abi.StringTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("could not encode string as the value provided is not of the correct type")
		}
		return str, nil
	case abi.BytesTy:
		b, ok := value.([]byte)
		if !ok {
			return nil, fmt.Errorf("could not encode dynamic-sized bytes as the value provided is not of the correct type")
		}
		return hex.EncodeToString(b), nil
	case abi.FixedBytesTy:
		// TODO: Error checking to ensure `value` is of the correct type.
		b := reflectionutils.ArrayToSlice(reflect.ValueOf(value)).([]byte)
		return hex.EncodeToString(b), nil
	case abi.ArrayTy:
		// Encode all underlying elements in our array
		reflectedArray := reflect.ValueOf(value)
		arrayData := make([]any, 0)
		for i := 0; i < reflectedArray.Len(); i++ {
			elementData, err := encodeJSONArgument(inputType.Elem, reflectedArray.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			arrayData = append(arrayData, elementData)
		}
		return arrayData, nil
	case abi.SliceTy:
		// Encode all underlying elements in our slice
		reflectedArray := reflect.ValueOf(value)
		sliceData := make([]any, 0)
		for i := 0; i < reflectedArray.Len(); i++ {
			elementData, err := encodeJSONArgument(inputType.Elem, reflectedArray.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			sliceData = append(sliceData, elementData)
		}
		return sliceData, nil
	case abi.TupleTy:
		// Encode all underlying fields in our tuple/struct.
		reflectedTuple := reflect.ValueOf(value)
		tupleData := make(map[string]any)
		for i := 0; i < len(inputType.TupleElems); i++ {
			field := reflectedTuple.Field(i)
			fieldValue := reflectionutils.GetField(field)
			fieldData, err := encodeJSONArgument(inputType.TupleElems[i], fieldValue)
			if err != nil {
				return nil, err
			}
			tupleData[inputType.TupleRawNames[i]] = fieldData
		}
		return tupleData, nil
	default:
		return nil, fmt.Errorf("could not encode argument, type is unsupported: %v", inputType)
	}
}

// DecodeJSONArgumentsFromMap decodes JSON values into a provided values of the given types, or returns an error of one occurs.
// The values provided must be generic JSON types (e.g. []any, map[string]any, etc) which will be transformed into
// a go-ethereum ABI packable values.
func DecodeJSONArgumentsFromMap(inputs abi.Arguments, values map[string]any, deployedContractAddr map[string]common.Address) ([]any, error) {
	// Create a variable to store decoded arguments, fill it with the respective decoded arguments.
	var decodedArgs = make([]any, len(inputs))
	for i, input := range inputs {
		value, ok := values[input.Name]
		if !ok {
			err := fmt.Errorf("constructor argument not provided for: name: %v", input.Name)
			return nil, err
		}
		arg, err := decodeJSONArgument(&input.Type, value, deployedContractAddr)
		if err != nil {
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, value, err)
			return nil, err
		}
		decodedArgs[i] = arg
	}
	return decodedArgs, nil
}

// DecodeJSONArgumentsFromSlice decodes JSON values into a provided values of the given types, or returns an error of one occurs.
// The values provided must be generic JSON types (e.g. []any, map[string]any, etc) which will be transformed into
// a go-ethereum ABI packable values.
func DecodeJSONArgumentsFromSlice(inputs abi.Arguments, values []any, deployedContractAddr map[string]common.Address) ([]any, error) {
	// Check our argument value count against our ABI method arguments count.
	if len(values) != len(inputs) {
		err := fmt.Errorf("constructor argument count mismatch, expected %v but got %v", len(inputs), len(values))
		return nil, err
	}

	// Create a variable to store decoded arguments, fill it with the respective decoded arguments.
	var decodedArgs = make([]any, len(inputs))
	for i, input := range inputs {
		arg, err := decodeJSONArgument(&input.Type, values[i], deployedContractAddr)
		if err != nil {
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, values[i], err)
			return nil, err
		}
		decodedArgs[i] = arg
	}
	return decodedArgs, nil
}

// decodeJSONArgument decodes JSON value into a provided value of a given type, or returns an error of one occurs.
// The value provided must be a generic JSON type (e.g. []any, map[string]any, etc) which will be transformed into
// a go-ethereum ABI packable value.
func decodeJSONArgument(inputType *abi.Type, value any, deployedContractAddr map[string]common.Address) (any, error) {
	var v any
	switch inputType.T {
	case abi.AddressTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("address value should be added as string in JSON")
		}
		// Check if this is a Magic value to get deployed contract address
		if _, contractName, found := strings.Cut(str, addressJSONContractNameOverridePrefix); found {
			v, ok = deployedContractAddr[contractName]
			if !ok {
				return nil, fmt.Errorf("contract %s not found in deployed contracts", contractName)
			}
		} else {
			if !((len(str) == (common.AddressLength*2 + 2)) || (len(str) == common.AddressLength*2)) {
				err := fmt.Errorf("invalid address length (%v)", len(str))
				return nil, err
			}
			v = common.HexToAddress(str)
		}
	case abi.UintTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("integer value should be specified as a string in JSON")
		}
		val := big.NewInt(0)
		_, success := val.SetString(str, 0)
		if !success {
			return nil, fmt.Errorf("invalid integer value")
		}
		switch inputType.Size {
		case 64:
			v = val.Uint64()
		case 32:
			v = uint32(val.Uint64())
		case 16:
			v = uint16(val.Uint64())
		case 8:
			v = uint8(val.Uint64())
		default:
			v = val
		}
	case abi.IntTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("integer value should be added as a string in JSON")
		}
		val := big.NewInt(0)
		_, success := val.SetString(str, 0)
		if !success {
			return nil, fmt.Errorf("invalid integer value")
		}
		switch inputType.Size {
		case 64:
			v = val.Int64()
		case 32:
			v = int32(val.Int64())
		case 16:
			v = int16(val.Int64())
		case 8:
			v = int8(val.Int64())
		default:
			v = val
		}
	case abi.BoolTy:
		bl, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid bool value")
		}
		v = bl
	case abi.StringTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string value")
		}
		v = str
	case abi.BytesTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("bytes value should be added as string in JSON")
		}
		if len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X') {
			str = str[2:]
		}
		decodedBytes, err := hex.DecodeString(str)
		if err != nil {
			return nil, err
		}
		v = decodedBytes
	case abi.FixedBytesTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s value should be added as string in JSON", inputType)
		}
		if len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X') {
			str = str[2:]
		}
		decodedBytes, err := hex.DecodeString(str)
		if err != nil {
			return nil, err
		}
		if len(decodedBytes) != inputType.Size {
			return nil, fmt.Errorf("invalid number of bytes %v", len(decodedBytes))
		}

		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		bytesValue := reflect.ValueOf(decodedBytes)
		fixedBytes := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < fixedBytes.Len(); i++ {
			fixedBytes.Index(i).Set(bytesValue.Index(i))
		}
		v = fixedBytes.Interface()
	case abi.ArrayTy:
		arr, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid JSON value, array expected")
		}
		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		for i, e := range arr {
			ele, err := decodeJSONArgument(inputType.Elem, e, deployedContractAddr)
			if err != nil {
				return nil, err
			}
			array.Index(i).Set(reflect.ValueOf(ele))
		}
		v = array.Interface()
	case abi.SliceTy:
		arr, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid JSON value, array expected")
		}
		// Element type of slice is dynamic therefore it needs to be created with reflection.
		slice := reflect.MakeSlice(inputType.GetType(), len(arr), len(arr))
		for i, e := range arr {
			ele, err := decodeJSONArgument(inputType.Elem, e, deployedContractAddr)
			if err != nil {
				return nil, err
			}
			slice.Index(i).Set(reflect.ValueOf(ele))
		}
		v = slice.Interface()
	case abi.TupleTy:
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid JSON value, object expected")
		}
		// Tuples are used to represent structs. struct fields are dynamic therefore we create them through reflection.
		st := reflect.Indirect(reflect.New(inputType.GetType()))
		for i, eleType := range inputType.TupleElems {
			field := st.Field(i)
			fieldName := inputType.TupleRawNames[i]
			fieldValue, ok := object[fieldName]
			if !ok {
				return nil, fmt.Errorf("value for struct field %s not provided", fieldName)
			}
			eleValue, err := decodeJSONArgument(eleType, fieldValue, deployedContractAddr)
			if !ok {
				return nil, fmt.Errorf("can not parse struct field %s, error: %s", fieldName, err)
			}
			reflectionutils.SetField(field, eleValue)
		}
		v = st.Interface()
	default:
		err := fmt.Errorf("argument type is not supported: %v", inputType)
		return nil, err
	}

	return v, nil
}
