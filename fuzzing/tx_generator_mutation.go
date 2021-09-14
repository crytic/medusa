package fuzzing

import (
	"github.com/trailofbits/medusa/utils"
	"math/big"
)

// txGeneratorMutation represents an interface for a provider used to generate transaction fields and call arguments
// using mutation-based approaches against items within the corpus, such as AST literals.
type txGeneratorMutation struct {
	// baseIntegers represents the base integer values to be used in mutations.
	baseIntegers []*big.Int
	// baseStrings represents the base strings values to be used in mutations.
	baseStrings []string
	// baseBytes represents the base byte arrays to be used in mutations.
	baseBytes [][]byte

	maxMutationRounds int

	// txGeneratorRandom is included to inherit from the random generator
	*txGeneratorRandom
}

// newTxGeneratorMutation creates a new txGeneratorMutation using a provided corpus to seed base-values for mutation.
func newTxGeneratorMutation(corpus *Corpus) *txGeneratorMutation {
	// Obtain our list of integers as a big int pointer array.
	corpusIntegers := corpus.Integers()
	formattedBaseIntegers := make([]*big.Int, len(corpusIntegers))
	for i := 0; i < len(corpusIntegers); i++ {
		formattedBaseIntegers[i] = &corpusIntegers[i]
	}

	// Create and return our generator
	generator := &txGeneratorMutation{
		baseIntegers: formattedBaseIntegers,
		baseStrings: corpus.Strings(),
		baseBytes: corpus.Bytes(),
		maxMutationRounds: 3,
		txGeneratorRandom: newTxGeneratorRandom(),
	}
	return generator
}

// generateInteger generates/selects an integer to use when populating transaction fields.
func (g *txGeneratorMutation) generateInteger(worker *fuzzerWorker, signed bool, bitLength int) *big.Int {
	// Calculate our integer bounds
	min, max := utils.GetIntegerConstraints(signed, bitLength)

	// Determine additional inputs based off the value type
	var inputs []*big.Int
	inputs = append(inputs,  g.baseIntegers...)
	if signed {
		inputs = append(inputs, big.NewInt(0), big.NewInt(1), big.NewInt(-1), big.NewInt(2), min, max)
	} else {
		inputs = append(inputs, big.NewInt(1), big.NewInt(2), min, max) // zero is included in minimum
	}

	// Generate a mutated integer.
	return g.generateMutatedInteger(worker, min, max, inputs)
}

// integerMutationMethods define methods which take a big integer and a set of inputs and
// transform the integer with a random input and operation. This is used in a loop to create
// mutated integer values.
var integerMutationMethods = []func(*txGeneratorMutation, *big.Int, ...*big.Int) *big.Int {
	func(g *txGeneratorMutation, x *big.Int, inputs ...*big.Int) *big.Int {
		// Add a random input
		return big.NewInt(0).Add(x, inputs[g.randomProvider.Int() % len(inputs)])
	},
	func(g *txGeneratorMutation, x *big.Int, inputs ...*big.Int) *big.Int {
		// Subtract a random input
		return big.NewInt(0).Sub(x, inputs[g.randomProvider.Int() % len(inputs)])
	},
	func(g *txGeneratorMutation, x *big.Int, inputs ...*big.Int) *big.Int {
		// Multiply a random input
		return big.NewInt(0).Mul(x, inputs[g.randomProvider.Int() % len(inputs)])
	},
	func(g *txGeneratorMutation, x *big.Int, inputs ...*big.Int) *big.Int {
		// Divide a random input
		divisor := inputs[g.randomProvider.Int() % len(inputs)]
		if divisor.Cmp(big.NewInt(0)) == 0 {
			return big.NewInt(1) // leave unchanged if divisor was zero (would've caused panic)
		}
		return big.NewInt(0).Div(x, divisor)
	},
	func(g *txGeneratorMutation, x *big.Int, inputs ...*big.Int) *big.Int {
		// Modulo divide a random input
		divisor := inputs[g.randomProvider.Int() % len(inputs)]
		if divisor.Cmp(big.NewInt(0)) == 0 {
			return big.NewInt(0).Set(x) // leave unchanged if divisor was zero (would've caused panic)
		}
		return big.NewInt(0).Mod(x, divisor)
	},
}

// generateMutatedInteger generates an integer of the specified type by taking a starting value and mutating it
// such that it is derived from a sensible value and may reach some code paths more effectively, but it is still
// sufficiently randomized.
// Returns the generated mutated integer.
func (g *txGeneratorMutation) generateMutatedInteger(worker *fuzzerWorker, min *big.Int, max *big.Int, inputs []*big.Int) *big.Int {
	// Declare additional inputs for our integer to mutate and select one.
	// Note: We add some basic cases as well.

	g.randomProviderLock.Lock()
	input := inputs[g.randomProvider.Int() % len(inputs)]

	// Determine how many times to mutate our integer and begin each pass
	mutationCount := g.randomProvider.Int() % (g.maxMutationRounds + 1)
	g.randomProviderLock.Unlock()
	for i := 0; i < mutationCount; i++ {
		// Mutate input
		g.randomProviderLock.Lock()
		input = integerMutationMethods[g.randomProvider.Int() % len(integerMutationMethods)](g, input, inputs...)
		g.randomProviderLock.Unlock()

		// Correct value boundaries (underflow/overflow)
		input = utils.ConstrainIntegerToBounds(input, min, max)
	}
	return input
}
