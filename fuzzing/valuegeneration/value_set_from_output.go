package valuegeneration

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// AddOutputAbiValueToValueSet adds the output values of a contract call (currently only pure calls) to the value set.
func (vs *ValueSet) AddOutputAbiValueToValueSet(outputTypes abi.Arguments, outputValues []interface{}) {
	// Return early to be robust against mismatched lengths
	if len(outputTypes) != len(outputValues) {
		return
	}
	for i, outputType := range outputTypes {
		switch outputType.Type.T {
		case abi.AddressTy:
			address, ok := outputValues[i].(common.Address)
			if ok {
				vs.AddAddress(address)
				vs.AddBytes(address.Bytes())
			}
		case abi.UintTy, abi.IntTy:
			i, ok := outputValues[i].(*big.Int)
			if ok {
				vs.AddInteger(i)
				vs.AddAddress(common.BigToAddress(i))
			}
		case abi.BoolTy:
			continue
		case abi.StringTy:
			str, ok := outputValues[i].(string)
			if ok {
				vs.AddString(str)
			}
		case abi.BytesTy, abi.FixedBytesTy:
			b, ok := outputValues[i].([]byte)
			if ok {
				vs.AddBytes(b)
				vs.AddAddress(common.BytesToAddress(b))
			}

		case abi.ArrayTy:

		case abi.SliceTy:

		case abi.TupleTy:

		default:

		}
	}
}
