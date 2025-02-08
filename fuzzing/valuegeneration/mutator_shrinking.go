package valuegeneration

import (
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ShrinkingValueMutator represents a ValueMutator used to shrink function inputs and call arguments.
type ShrinkingValueMutator struct {
	// config describes the configuration defining value mutation parameters.
	config *ShrinkingValueMutatorConfig

	// valueSet contains a set of values which the ValueGenerator may use to aid in value generation and mutation
	// operations.
	valueSet *ValueSet

	// randomProvider offers a source of random data.
	randomProvider *rand.Rand
}

// ShrinkingValueMutatorConfig defines the operating parameters for a ShrinkingValueMutator.
type ShrinkingValueMutatorConfig struct {
	// ShrinkValueProbability is the probability that any shrinkable value will be shrunk/mutated when a mutation
	// method is invoked.
	ShrinkValueProbability float32
}

// NewShrinkingValueMutator creates a new ShrinkingValueMutator using a ValueSet to seed base-values for mutation.
func NewShrinkingValueMutator(config *ShrinkingValueMutatorConfig, valueSet *ValueSet, randomProvider *rand.Rand) *ShrinkingValueMutator {
	// Create and return our generator
	generator := &ShrinkingValueMutator{
		config:         config,
		valueSet:       valueSet,
		randomProvider: randomProvider,
	}

	// Ensure some initial values this mutator will depend on for basic mutations to the set.
	generator.valueSet.AddInteger(big.NewInt(0))
	generator.valueSet.AddInteger(big.NewInt(1))
	generator.valueSet.AddInteger(big.NewInt(2))
	return generator
}

// MutateAddress takes an address input and sometimes returns a mutated value based off the input.
// This type is not mutated by the ShrinkingValueMutator.
func (g *ShrinkingValueMutator) MutateAddress(addr common.Address) common.Address {
	return addr
}

// MutateArray takes a dynamic or fixed sized array as input, and returns a mutated value based off of the input.
// The ABI type of the array is also provided in case new values need to be generated. Returns the mutated value.
// This type is not mutated by the ShrinkingValueMutator.
func (g *ShrinkingValueMutator) MutateArray(value []any, fixedLength bool, abiType *abi.Type) []any {
	return value
}

// MutateBool takes a boolean input and returns a mutated value based off the input.
// This type is not mutated by the ShrinkingValueMutator.
func (g *ShrinkingValueMutator) MutateBool(bl bool) bool {
	// Always return false
	return false
}

// MutateFixedBytes takes a fixed-sized byte array input and returns a mutated value based off the input.
// This type is not mutated by the ShrinkingValueMutator.
func (g *ShrinkingValueMutator) MutateFixedBytes(b []byte) []byte {
	return b
}

// bytesShrinkingMethods define methods which take an initial bytes and a set of inputs to transform the input. The
// transformed input is returned.
var bytesShrinkingMethods = []func(*ShrinkingValueMutator, []byte) []byte{
	// Replace a random index with a zero byte
	func(g *ShrinkingValueMutator, b []byte) []byte {
		if len(b) > 0 {
			b[g.randomProvider.Intn(len(b))] = 0
		}
		return b
	},
	// Remove a random byte
	func(g *ShrinkingValueMutator, b []byte) []byte {
		// If we have no bytes to remove, do nothing.
		if len(b) == 0 {
			return b
		}

		i := g.randomProvider.Intn(len(b))
		return append(b[:i], b[i+1:]...)
	},
}

// MutateBytes takes a dynamic-sized byte array input and returns a mutated value based off the input.
func (g *ShrinkingValueMutator) MutateBytes(b []byte) []byte {
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.ShrinkValueProbability {
		// Mutate the data for our desired number of rounds
		input := bytesShrinkingMethods[g.randomProvider.Intn(len(bytesShrinkingMethods))](g, b)
		return input
	}
	return b
}

// MutateInteger takes an integer input and applies optional mutations to the provided value.
// Returns an optionally mutated copy of the input.
func (g *ShrinkingValueMutator) MutateInteger(i *big.Int, signed bool, bitLength int) *big.Int {
	// If the integer is zero, we can simply return it as-is.
	if i.Sign() == 0 {
		return i
	}

	// For unsigned integers or positive signed integers, generate a new integer between [0, i)
	if !signed || i.Sign() > 0 {
		return big.NewInt(0).Rand(g.randomProvider, i)
	}

	// For negative numbers, generate between (i, 0]
	// First get absolute value and generate random number between [0, abs(i))
	offset := big.NewInt(0).Rand(g.randomProvider, big.NewInt(0).Abs(i))
	offset.Add(offset, big.NewInt(1))

	// Add it to the original value to reach the range (i, 0]
	return i.Add(i, offset)
}

// stringShrinkingMethods define methods which take an initial string and a set of inputs to transform the input. The
// transformed input is returned.
var stringShrinkingMethods = []func(*ShrinkingValueMutator, string) string{
	// Replace a random index with a NULL char
	func(g *ShrinkingValueMutator, s string) string {
		// If the string is empty, we can simply return a new string with just the rune in it.
		r := []rune(s)
		if len(r) == 0 {
			return string(r)
		}

		// Otherwise, we replace a rune in it and return it.
		r[g.randomProvider.Intn(len(r))] = 0
		return string(r)
	},
	// Remove a random character
	func(g *ShrinkingValueMutator, s string) string {
		// If we have no characters to remove, do nothing
		if len(s) == 0 {
			return s
		}

		// Otherwise, remove a random character.
		i := g.randomProvider.Intn(len(s))
		return s[:i] + s[i+1:]
	},
}

// MutateString takes a string input and returns a mutated value based off the input.
func (g *ShrinkingValueMutator) MutateString(s string) string {
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.ShrinkValueProbability {
		input := stringShrinkingMethods[g.randomProvider.Intn(len(stringShrinkingMethods))](g, s)
		return input
	}
	return s
}
