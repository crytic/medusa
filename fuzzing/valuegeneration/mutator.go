package valuegeneration

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// ValueMutator represents an interface for a provider used to mutate function inputs and call arguments for use
// in fuzzing campaigns.
type ValueMutator interface {
	// MutateAddress takes an address input and returns a mutated value based off the input.
	MutateAddress(addr common.Address) common.Address

	// MutateArray takes a dynamic or fixed sized array as input, and returns a mutated value based off of the input.
	// Returns the mutated value. If any element of the returned array is nil, the value generator will be called upon
	// to generate a new value in its place.
	MutateArray(value []any, fixedLength bool) []any

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
