package abiutils

import (
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/stretchr/testify/require"
)

func solidityErrorData(tb testing.TB, name, typeName string, value any) []byte {
	tb.Helper()
	argumentType, err := abi.NewType(typeName, "", nil)
	require.NoError(tb, err)
	method := abi.NewMethod(name, name, abi.Function, "", false, false,
		abi.Arguments{{Type: argumentType}}, nil)
	encoded, err := method.Inputs.Pack(value)
	require.NoError(tb, err)
	return append(slices.Clone(method.ID), encoded...)
}

func TestSolidityPanicDecoding(t *testing.T) {
	t.Parallel()
	maxValue := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	for _, code := range []*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(0x51), maxValue} {
		data := solidityErrorData(t, "Panic", "uint256", code)
		decoded := GetSolidityPanicCode(vm.ErrExecutionReverted, data, false)
		require.NotNil(t, decoded)
		require.Zero(t, code.Cmp(decoded))
		decoded.SetInt64(123)
		require.Zero(t, code.Cmp(GetSolidityPanicCode(vm.ErrExecutionReverted, data, false)))
		for _, invalid := range [][]byte{nil, data[:4], data[:35], append(slices.Clone(data), 0)} {
			require.Nil(t, GetSolidityPanicCode(vm.ErrExecutionReverted, invalid, true))
		}
		require.Nil(t, GetSolidityPanicCode(nil, data, true))
		wrapped := fmt.Errorf("call failed: %w", vm.ErrExecutionReverted)
		require.Zero(t, code.Cmp(GetSolidityPanicCode(wrapped, data, false)))
		data[0] ^= 1
		require.Nil(t, GetSolidityPanicCode(vm.ErrExecutionReverted, data, false))
	}
	invalidOpcode := new(vm.ErrInvalidOpCode)
	require.Equal(t, big.NewInt(1), GetSolidityPanicCode(invalidOpcode, nil, true))
	require.Nil(t, GetSolidityPanicCode(invalidOpcode, nil, false))
	wrapped := fmt.Errorf("call failed: %w", invalidOpcode)
	require.Nil(t, GetSolidityPanicCode(wrapped, nil, true))
}

func TestSolidityStringDecoding(t *testing.T) {
	t.Parallel()
	for _, message := range []string{"", "reverted", "こんにちは\x00世界"} {
		data := solidityErrorData(t, "Error", "string", message)
		for _, err := range []error{vm.ErrExecutionReverted,
			fmt.Errorf("wrapped: %w", vm.ErrExecutionReverted)} {
			require.Equal(t, &message, GetSolidityRevertErrorString(err, data))
			extra := append(slices.Clone(data), 1, 2, 3)
			require.Equal(t, &message, GetSolidityRevertErrorString(err, extra))
		}
		require.Nil(t, GetSolidityRevertErrorString(nil, data))
		for _, size := range []int{0, 4, 5, 35, 36, 67} {
			require.Nil(t, GetSolidityRevertErrorString(vm.ErrExecutionReverted, data[:size]))
		}
		badOffset := slices.Clone(data)
		badOffset[4] = 0xff
		require.Nil(t, GetSolidityRevertErrorString(vm.ErrExecutionReverted, badOffset))
		data[0] ^= 1
		require.Nil(t, GetSolidityRevertErrorString(vm.ErrExecutionReverted, data))
	}
}

func FuzzSolidityErrorDecoding(f *testing.F) {
	panicType, err := abi.NewType("uint256", "", nil)
	require.NoError(f, err)
	stringType, err := abi.NewType("string", "", nil)
	require.NoError(f, err)
	panicArgs, stringArgs := abi.Arguments{{Type: panicType}}, abi.Arguments{{Type: stringType}}
	panicData := solidityErrorData(f, "Panic", "uint256", big.NewInt(1))
	stringData := solidityErrorData(f, "Error", "string", "hello-世界")
	f.Add(panicData[4:])
	f.Add(stringData[4:])
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, payload []byte) {
		panicCode := GetSolidityPanicCode(vm.ErrExecutionReverted,
			append(slices.Clone(panicData[:4]), payload...), false)
		if len(payload) == 32 {
			decoded, err := panicArgs.Unpack(payload)
			require.NoError(t, err)
			require.NotNil(t, panicCode)
			require.Zero(t, decoded[0].(*big.Int).Cmp(panicCode))
		} else {
			require.Nil(t, panicCode)
		}
		message := GetSolidityRevertErrorString(vm.ErrExecutionReverted,
			append(slices.Clone(stringData[:4]), payload...))
		decoded, err := stringArgs.Unpack(payload)
		if err != nil {
			require.Nil(t, message)
		} else {
			require.NotNil(t, message)
			require.Equal(t, decoded[0].(string), *message)
		}
	})
}

// BenchmarkSolidityErrors measures valid and unrecognized standard Solidity error payloads.
func BenchmarkSolidityErrors(b *testing.B) {
	panicData := solidityErrorData(b, "Panic", "uint256", big.NewInt(1))
	stringData := solidityErrorData(b, "Error", "string", "nested revert")
	for _, recognized := range []bool{true, false} {
		b.Run(fmt.Sprintf("recognized=%t", recognized), func(b *testing.B) {
			panicData, stringData := slices.Clone(panicData), slices.Clone(stringData)
			if !recognized {
				panicData[0] ^= 1
				stringData[0] ^= 1
			}
			b.Run("panic", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					GetSolidityPanicCode(vm.ErrExecutionReverted, panicData, false)
				}
			})
			b.Run("string", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					GetSolidityRevertErrorString(vm.ErrExecutionReverted, stringData)
				}
			})
		})
	}
}
