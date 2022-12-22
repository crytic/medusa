package utils

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"reflect"
	"strings"
)

const DeployedContractAddressPrefix = "DeployedContract:"

// DecodeJSONArguments Decode provided map of JSON values for provided list of abi arguments
// This function assumes that the values map is generated by JSON unmarshal function
func DecodeJSONArguments(inputs abi.Arguments, values map[string]any, deployedContractAddr map[string]common.Address) ([]any, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	var args = make([]any, len(inputs))
	for i, input := range inputs {
		value, ok := values[input.Name]
		if !ok {
			err := fmt.Errorf("constructor argument not provided for: name: %v", input.Name)
			return nil, err
		}
		arg, err := decodeJSONArgument(&input.Type, value, deployedContractAddr)
		if err != nil {
			err = fmt.Errorf("invalid constructor argument: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, value, err)
			return nil, err
		}
		args[i] = arg
	}
	return args, nil
}

// decodeArgument Decode JSON value for provided ABI type
// This function assumes that the value is generated by JSON unmarshal function
func decodeJSONArgument(inputType *abi.Type, value any, deployedContractAddr map[string]common.Address) (any, error) {
	var v any
	switch inputType.T {
	case abi.IntTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("interger value should be added as string in JSON")
		}
		val := big.NewInt(0)
		_, success := val.SetString(str, 10)
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
	case abi.UintTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("interger value should be specified as string in JSON")
		}
		val := big.NewInt(0)
		_, success := val.SetString(str, 10)
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
	case abi.BoolTy:
		str, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid bool value")
		}
		v = str
	case abi.StringTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string value")
		}
		v = str
	case abi.AddressTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("address value should be added as string in JSON")
		}
		// Check if this is a Magic value to get deployed contract address
		if _, contractName, found := strings.Cut(str, DeployedContractAddressPrefix); found {
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
			fieldName := inputType.TupleRawNames[i]
			fieldValue, ok := object[fieldName]
			if !ok {
				return nil, fmt.Errorf("value for struct field %s not provided", fieldName)
			}
			eleValue, err := decodeJSONArgument(eleType, fieldValue, deployedContractAddr)
			if !ok {
				return nil, fmt.Errorf("can not parse struct field %s, error: %s", fieldName, err)
			}
			st.Field(i).Set(reflect.ValueOf(eleValue))
		}
		v = st.Interface()
	default:
		err := fmt.Errorf("argument type is not supported: %v", inputType)
		return nil, err
	}

	return v, nil
}
