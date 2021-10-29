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

var bytesMutationMethods = []func(*txGeneratorMutation, []byte, ...[]byte) []byte {
	// Replace a random index with a random byte
	func(g *txGeneratorMutation, b []byte, inputs ...[]byte ) []byte {
		b[g.randomProvider.Int() % len(b)] = byte(g.randomProvider.Int() % 256)
		return b
	},
	// Append a random byte to the front
	func(g *txGeneratorMutation, b []byte, inputs ...[]byte ) []byte {
		return append([]byte{byte(g.randomProvider.Int() % 256)}, b[:]...)
	},
	// Append a random byte to the end
	func(g *txGeneratorMutation, b []byte, inputs ...[]byte ) []byte {
		return append(b, byte(g.randomProvider.Int() % 256))
	},
	// Remove a random byte
	func(g *txGeneratorMutation, b []byte, inputs ...[]byte ) []byte {
		i := g.randomProvider.Int() % len(b)
		return append(b[:i], b[i+1:]...)
	},
}

var stringMutationMethods = []func(*txGeneratorMutation, string, ...string) string {
	// Replace a random index with a random character
	func(g *txGeneratorMutation, s string, inputs ...string) string {
		r := []rune(s)
		r[g.randomProvider.Int() % len(s)] = rune(32 + g.randomProvider.Int() % 95)
		return string(r)
	},
	// Append a random character to the front
	func(g *txGeneratorMutation, s string, inputs ...string) string {
		return s + string(32 + g.randomProvider.Int() % 95)
	},
	// Append a random character to the end
	func(g *txGeneratorMutation, s string, inputs ...string) string {
		return string(32 + g.randomProvider.Int() % 95) + s
	},
	// Remove a random character
	func(g *txGeneratorMutation, s string, inputs ...string) string {
		i := g.randomProvider.Int() % len(s)
		return s[:i] + s[i+1:]
	},
}

func (g *txGeneratorMutation) getMutationParams(inputsLen int) (int, int) {
	g.randomProviderLock.Lock()
	inputIdx := g.randomProvider.Int() % inputsLen
	mutationCount := g.randomProvider.Int() % (g.maxMutationRounds + 1)
	g.randomProviderLock.Unlock()

	return inputIdx, mutationCount
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

	inputIdx, mutationCount := g.getMutationParams(len(inputs))
	input := inputs[inputIdx]

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

func (g *txGeneratorMutation) generateBytes(worker *fuzzerWorker) []byte {
	inputs := g.baseBytes
	inputIdx, mutationCount := g.getMutationParams(len(inputs))
	input := inputs[inputIdx]

	for i := 0; i < mutationCount; i++ {
		g.randomProviderLock.Lock()
		input = bytesMutationMethods[g.randomProvider.Int() % len(bytesMutationMethods)](g, input, inputs...)
		g.randomProviderLock.Unlock()
	}

	return input
}

func (g *txGeneratorMutation) generateString(worker *fuzzerWorker) string {
	inputs := g.baseStrings
	inputIdx, mutationCount := g.getMutationParams(len(inputs))
	input := inputs[inputIdx]

	for i := 0; i < mutationCount; i++ {
		g.randomProviderLock.Lock()
		input = stringMutationMethods[g.randomProvider.Int() % len(stringMutationMethods)](g, input, inputs...)
		g.randomProviderLock.Unlock()
	}

	return input
}
