package valuegeneration

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ValueMutator represents an interface for a provider used to mutate function inputs and call arguments for use
// in fuzzing campaigns.
type ValueMutator interface {
	// MutateAddress takes an address input and returns a mutated value based off the input.
	MutateAddress(addr common.Address) common.Address

	// MutateArray takes a dynamic or fixed sized array as input, and returns a mutated value based off of the input.
	// The ABI type of the array is also provided in case new values need to be generated. Returns the mutated value.
	MutateArray(value []any, fixedLength bool, abiType *abi.Type) []any

	// MutateBool takes a boolean input and returns a mutated value based off the input.
	MutateBool(bl bool) bool

	// MutateBytes takes a dynamic-sized byte array input and returns a mutated value based off the input.
	MutateBytes(b []byte) []byte

	// MutateFixedBytes takes a fixed-sized byte array input and returns a mutated value based off the input.
	MutateFixedBytes(b []byte) []byte

	// MutateString takes a string input and returns a mutated value based off the input.
	MutateString(s string) string

	// MutateInteger takes an integer input and returns a mutated value based off the input.
	MutateInteger(i *big.Int, signed bool, bitLength int) *big.Int
}
