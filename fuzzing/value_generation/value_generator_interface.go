package value_generation

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// ValueGenerator represents an interface for a provider used to generate function inputs and call arguments for use
// in fuzzing campaigns.
type ValueGenerator interface {
	// GenerateAddress generates/selects an address to use when populating inputs.
	GenerateAddress() common.Address
	// GenerateArrayLength generates/selects an array length to use when populating inputs.
	GenerateArrayLength() int
	// GenerateBool generates/selects a bool to use when populating inputs.
	GenerateBool() bool
	// GenerateBytes generates/selects a dynamic-sized byte array to use when populating inputs.
	GenerateBytes() []byte
	// GenerateFixedBytes generates/selects a fixed-sized byte array to use when populating inputs.
	GenerateFixedBytes(length int) []byte
	// GenerateString generates/selects a dynamic-sized string to use when populating inputs.
	GenerateString() string
	// GenerateInteger generates/selects an integer to use when populating inputs.
	GenerateInteger(signed bool, bitLength int) *big.Int
}
