package calls

import (
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/stretchr/testify/require"
)

func packTestABI(tb testing.TB) abi.ABI {
	tb.Helper()
	definition, err := abi.JSON(strings.NewReader(`[
		{"type":"function","name":"empty","inputs":[]},
		{"type":"function","name":"integer","inputs":[{"type":"uint256"}]},
		{"type":"function","name":"dynamic","inputs":[
			{"type":"bytes"},{"type":"string"},{"type":"uint256[]"}]}
	]`))
	require.NoError(tb, err)
	return definition
}

func TestABIPack(t *testing.T) {
	t.Parallel()
	definition := packTestABI(t)
	cases := map[string][]any{
		"empty":   nil,
		"integer": {big.NewInt(123)},
		"dynamic": {[]byte{1, 2, 3}, "hello-世界", []*big.Int{big.NewInt(1), big.NewInt(2)}},
	}
	for name, values := range cases {
		method := definition.Methods[name]
		data := CallMessageDataAbiValues{Method: &method, InputValues: values}
		expected, err := definition.Pack(name, values...)
		require.NoError(t, err)
		actual, err := data.Pack()
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		clear(actual)
		actual, err = data.Pack()
		require.NoError(t, err)
		require.Equal(t, expected, actual, "each packed result must own its bytes")
	}
}

func TestABIPackErrorsAndSelectors(t *testing.T) {
	t.Parallel()
	definition := packTestABI(t)
	method := definition.Methods["integer"]
	cases := []CallMessageDataAbiValues{
		{}, {Method: &method}, {Method: &method, InputValues: []any{"wrong type"}},
	}
	for _, data := range cases {
		packed, err := data.Pack()
		require.Error(t, err)
		require.Nil(t, packed)
	}
	for _, selector := range [][]byte{nil, {}, {1}, {1, 2, 3, 4, 5}} {
		method := definition.Methods["empty"]
		method.ID = slices.Clone(selector)
		data := CallMessageDataAbiValues{Method: &method}
		packed, err := data.Pack()
		require.NoError(t, err)
		require.NotNil(t, packed)
		require.Equal(t, append([]byte{}, selector...), packed)
	}
}

func TestABIPackDynamicBounds(t *testing.T) {
	t.Parallel()
	definition := packTestABI(t)
	for _, size := range []int{0, 1, 31, 32, 33, 63, 64, 65, 255, 256, 257, 1024} {
		payload := make([]byte, size)
		for i := range payload {
			payload[i] = byte(i)
		}
		values := []any{payload, "hello-世界", []*big.Int{big.NewInt(42)}}
		expected, err := definition.Pack("dynamic", values...)
		require.NoError(t, err)
		for _, selectorSize := range []int{0, 1, 4, 65} {
			method := definition.Methods["dynamic"]
			method.ID = make([]byte, selectorSize)
			data := CallMessageDataAbiValues{Method: &method, InputValues: values}
			want := append(slices.Clone(method.ID), expected[4:]...)
			actual, err := data.Pack()
			require.NoError(t, err)
			require.Equal(t, want, actual)
			clear(actual)
			actual, err = data.Pack()
			require.NoError(t, err)
			require.Equal(t, want, actual)
		}
	}
}

// BenchmarkABIPack measures complete call-data packing for empty, static, and dynamic arguments.
func BenchmarkABIPack(b *testing.B) {
	definition := packTestABI(b)
	for name, values := range map[string][]any{
		"empty":   nil,
		"integer": {big.NewInt(123)},
		"dynamic": {make([]byte, 1024), "hello-世界", []*big.Int{big.NewInt(1), big.NewInt(2)}},
	} {
		b.Run(name, func(b *testing.B) {
			method := definition.Methods[name]
			data := CallMessageDataAbiValues{Method: &method, InputValues: values}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := data.Pack(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
