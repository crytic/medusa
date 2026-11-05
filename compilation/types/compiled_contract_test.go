package types

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseABIFromInterface_NormalizesUserDefinedTypes(t *testing.T) {
	rawABI := []map[string]any{
		{
			"type": "function",
			"name": "computeL",
			"inputs": []map[string]any{
				{
					"name":         "positions",
					"type":         "tuple[]",
					"internalType": "struct NftRef[]",
					"components": []map[string]any{
						{
							"name":         "kind",
							"type":         "NftKind",
							"internalType": "enum NftKind",
						},
						{
							"name":         "tokenId",
							"type":         "uint248",
							"internalType": "uint248",
						},
					},
				},
				{
					"name":         "positionManager",
					"type":         "IPositionManager",
					"internalType": "contract IPositionManager",
				},
				{
					"name":         "loManager",
					"type":         "ILimitOrderManager",
					"internalType": "interface ILimitOrderManager",
				},
				{
					"name":         "keepers",
					"type":         "IKeeper[]",
					"internalType": "contract IKeeper[]",
				},
			},
			"outputs": []map[string]any{
				{
					"name":         "",
					"type":         "uint256",
					"internalType": "uint256",
				},
			},
			"stateMutability": "view",
		},
	}

	encodedABI, err := json.Marshal(rawABI)
	require.NoError(t, err)

	_, err = abi.JSON(strings.NewReader(string(encodedABI)))
	require.Error(t, err, "control check: raw ABI should fail with user-defined type aliases")

	parsedABI, err := ParseABIFromInterface(string(encodedABI))
	require.NoError(t, err)

	method, ok := parsedABI.Methods["computeL"]
	require.True(t, ok)
	require.Len(t, method.Inputs, 4)

	positions := method.Inputs[0]
	require.Equal(t, abi.SliceTy, positions.Type.T)
	require.NotNil(t, positions.Type.Elem)
	require.Equal(t, abi.TupleTy, positions.Type.Elem.T)
	require.Len(t, positions.Type.Elem.TupleElems, 2)

	kindField := positions.Type.Elem.TupleElems[0]
	assert.Equal(t, abi.UintTy, kindField.T)
	assert.Equal(t, 8, kindField.Size)

	assert.Equal(t, abi.AddressTy, method.Inputs[1].Type.T)
	assert.Equal(t, abi.AddressTy, method.Inputs[2].Type.T)

	keepers := method.Inputs[3]
	require.Equal(t, abi.SliceTy, keepers.Type.T)
	require.NotNil(t, keepers.Type.Elem)
	assert.Equal(t, abi.AddressTy, keepers.Type.Elem.T)
}
