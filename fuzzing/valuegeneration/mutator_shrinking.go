package valuegeneration

import (
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"math/rand"
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
// Returns the mutated value. If any element of the returned array is nil, the value generator will be called upon
// to generate it new.
// This type is not mutated by the ShrinkingValueMutator.
func (g *ShrinkingValueMutator) MutateArray(value []any, fixedLength bool) []any {
	return value
}

// MutateBool takes a boolean input and returns a mutated value based off the input.
// This type is not mutated by the ShrinkingValueMutator.
func (g *ShrinkingValueMutator) MutateBool(bl bool) bool {
	return bl
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

// integerShrinkingMethods define methods which take a big integer and a set of inputs and
// transform the integer with a random input and operation.
var integerShrinkingMethods = []func(*ShrinkingValueMutator, *big.Int, ...*big.Int) *big.Int{
	func(g *ShrinkingValueMutator, x *big.Int, inputs ...*big.Int) *big.Int {
		// If our base value is positive, we subtract from it. If it's positive, we add to it.
		// If it's zero, we leave it unchanged.
		r := big.NewInt(0)
		if x.Cmp(r) > 0 {
			r = r.Sub(x, inputs[g.randomProvider.Intn(len(inputs))])
		} else if x.Cmp(r) < 0 {
			r = r.Add(x, inputs[g.randomProvider.Intn(len(inputs))])
		}
		return r

	},
	func(g *ShrinkingValueMutator, x *big.Int, inputs ...*big.Int) *big.Int {
		// Divide by two
		return big.NewInt(0).Div(x, big.NewInt(2))
	},
}

// MutateInteger takes an integer input and applies optional mutations to the provided value.
// Returns an optionally mutated copy of the input.
func (g *ShrinkingValueMutator) MutateInteger(i *big.Int, signed bool, bitLength int) *big.Int {
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.ShrinkValueProbability {
		// Calculate our integer bounds
		min, max := utils.GetIntegerConstraints(signed, bitLength)

		// Obtain our inputs. We also add our min/max values for this range to the list of inputs.
		// Note: We exclude min being added if we're requesting an unsigned integer, as zero is already
		// in our set, and we don't want duplicates.
		var inputs []*big.Int
		inputs = append(inputs, g.valueSet.Integers()...)
		if signed {
			inputs = append(inputs, min, max)
		} else {
			inputs = append(inputs, max)
		}

		// Set the input and ensure it is constrained to the value boundaries
		input := new(big.Int).Set(i)
		input = utils.ConstrainIntegerToBounds(input, min, max)

		// Shrink input
		input = integerShrinkingMethods[g.randomProvider.Intn(len(integerShrinkingMethods))](g, input, inputs...)

		// Correct value boundaries (underflow/overflow)
		input = utils.ConstrainIntegerToBounds(input, min, max)
		return input
	}
	return i
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
