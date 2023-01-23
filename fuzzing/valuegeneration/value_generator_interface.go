package valuegeneration

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"math/rand"
)

// ValueGenerator represents an interface for a provider used to generate function inputs and call arguments for use
// in fuzzing campaigns.
type ValueGenerator interface {
	// RandomProvider returns the internal random provider used for value generation.
	RandomProvider() *rand.Rand

	// GenerateAddress generates/selects an address to use when populating inputs.
	GenerateAddress() common.Address
	// MutateAddress takes an address input and returns a mutated value based off the input.
	MutateAddress(addr common.Address) common.Address

	// GenerateArrayOfLength generates/selects an array length to use when populating inputs.
	GenerateArrayOfLength() int
	// MutateArray takes a dynamic or fixed sized array as input, and returns a mutated value based off of the input.
	// Returns the mutated value. If any element of the returned array is nil, the value generator will be called upon
	// to generate it new.
	MutateArray(value []any, fixedLength bool) []any

	// GenerateBool generates/selects a bool to use when populating inputs.
	GenerateBool() bool
	// MutateBool takes a boolean input and returns a mutated value based off the input.
	MutateBool(bl bool) bool

	// GenerateBytes generates/selects a dynamic-sized byte array to use when populating inputs.
	GenerateBytes() []byte
	// MutateBytes takes a dynamic-sized byte array input and returns a mutated value based off the input.
	MutateBytes(b []byte) []byte

	// GenerateFixedBytes generates/selects a fixed-sized byte array to use when populating inputs.
	GenerateFixedBytes(length int) []byte
	// MutateFixedBytes takes a fixed-sized byte array input and returns a mutated value based off the input.
	MutateFixedBytes(b []byte) []byte

	// GenerateString generates/selects a dynamic-sized string to use when populating inputs.
	GenerateString() string
	// MutateString takes a string input and returns a mutated value based off the input.
	MutateString(s string) string

	// GenerateInteger generates/selects an integer to use when populating inputs.
	GenerateInteger(signed bool, bitLength int) *big.Int
	// MutateInteger takes an integer input and returns a mutated value based off the input.
	MutateInteger(i *big.Int, signed bool, bitLength int) *big.Int
}
