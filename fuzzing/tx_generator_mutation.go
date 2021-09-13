package fuzzing

import (
	"math/big"
	"math/rand"
)

type txGeneratorMutation struct {
	// baseIntegers represents the base integer values to be used in mutations.
	baseIntegers []*big.Int
	// baseStrings represents the base strings values to be used in mutations.
	baseStrings []string
	// baseBytes represents the base byte arrays to be used in mutations.
	baseBytes [][]byte

	maxMutationRounds int

	// Inherit the random mutation
	*txGeneratorRandom
}

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

func (g *txGeneratorMutation) generateBytes(worker *fuzzerWorker) []byte {
	g.randomProviderLock.Lock()
	b := make([]byte, rand.Uint64() % 100) // TODO: Right now we only generate 0-100 bytes, make this configurable.
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()
	return b
}

func (g *txGeneratorMutation) generateString(worker *fuzzerWorker) string {
	return string(g.generateBytes(worker))
}

func (g *txGeneratorMutation) generateFixedBytes(worker *fuzzerWorker, length int) []byte {
	g.randomProviderLock.Lock()
	b := make([]byte, length)
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()
	return b
}

func (g *txGeneratorMutation) generateArbitraryUint(worker *fuzzerWorker, bitLength int) *big.Int {
	return g.mutateUint(worker, bitLength)
}

func (g *txGeneratorMutation) generateUint64(worker *fuzzerWorker) uint64 {
	return g.generateArbitraryUint(worker, 64).Uint64()
}

func (g *txGeneratorMutation) generateUint32(worker *fuzzerWorker) uint32 {
	return uint32(g.generateArbitraryUint(worker, 32).Uint64())
}

func (g *txGeneratorMutation) generateUint16(worker *fuzzerWorker) uint16 {
	return uint16(g.generateArbitraryUint(worker, 16).Uint64())
}

func (g *txGeneratorMutation) generateUint8(worker *fuzzerWorker) uint8 {
	return uint8(g.generateArbitraryUint(worker, 8).Uint64())
}

func (g *txGeneratorMutation) generateArbitraryInt(worker *fuzzerWorker, bitLength int) *big.Int {
	return g.mutateInt(worker, bitLength)
}

func (g *txGeneratorMutation) generateInt64(worker *fuzzerWorker) int64 {
	return g.generateArbitraryUint(worker, 32).Int64()
}

func (g *txGeneratorMutation) generateInt32(worker *fuzzerWorker) int32 {
	return int32(g.generateArbitraryUint(worker, 32).Int64())
}

func (g *txGeneratorMutation) generateInt16(worker *fuzzerWorker) int16 {
	return int16(g.generateArbitraryUint(worker, 16).Int64())
}

func (g *txGeneratorMutation) generateInt8(worker *fuzzerWorker) int8 {
	return int8(g.generateArbitraryUint(worker, 8).Int64())
}

func (g *txGeneratorMutation) constrainIntegerToBounds(b *big.Int, min *big.Int, max *big.Int) *big.Int {
	// Get the bounding range
	boundingRange := big.NewInt(0).Add(big.NewInt(0).Sub(max, min), big.NewInt(1))

	// Check underflow
	if b.Cmp(min) < 0 {
		distance := big.NewInt(0).Sub(min, b)
		correction := big.NewInt(0).Div(big.NewInt(0).Add(distance, big.NewInt(0).Sub(boundingRange, big.NewInt(1))), boundingRange)
		correction.Mul(correction, boundingRange)
		return big.NewInt(0).Add(b, correction)
	}

	// Check overflow
	if b.Cmp(max) > 0 {
		distance := big.NewInt(0).Sub(b, max)
		correction := big.NewInt(0).Div(big.NewInt(0).Add(distance, big.NewInt(0).Sub(boundingRange, big.NewInt(1))), boundingRange)
		correction.Mul(correction, boundingRange)
		return big.NewInt(0).Sub(b, correction)
	}

	// b is in range, return a copy of it
	return big.NewInt(0).Set(b)
}


var mutationMethods = []func(*txGeneratorMutation, *big.Int, ...*big.Int) *big.Int {
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


func (g *txGeneratorMutation) mutateInt(worker *fuzzerWorker, bitLength int) *big.Int {
	// Determine our integer bounds
	max := big.NewInt(2)
	max.Exp(max, big.NewInt(int64(bitLength - 1)), nil)
	max.Sub(max, big.NewInt(1))
	min := big.NewInt(0).Mul(max, big.NewInt(-1))
	min.Sub(min, big.NewInt(1))

	// Declare additional inputs for our integer to mutate and select one.
	// Note: We add some basic cases as well.
	inputs := append(g.baseIntegers, big.NewInt(0), big.NewInt(1), big.NewInt(-1), min, max)
	g.randomProviderLock.Lock()
	input := inputs[g.randomProvider.Int() % len(inputs)]

	// Determine how many times to mutate our integer and begin each pass
	mutationCount := g.randomProvider.Int() % (g.maxMutationRounds + 1)
	g.randomProviderLock.Unlock()
	for i := 0; i < mutationCount; i++ {
		// Mutate input
		g.randomProviderLock.Lock()
		input = mutationMethods[g.randomProvider.Int() % len(mutationMethods)](g, input, inputs...)
		g.randomProviderLock.Unlock()

		// Correct value boundaries (underflow/overflow)
		input = g.constrainIntegerToBounds(input, min, max)
	}
	return input
}

func (g *txGeneratorMutation) mutateUint(worker *fuzzerWorker, bitLength int) *big.Int {
	// Determine our bounds
	max := big.NewInt(2)
	max.Exp(max, big.NewInt(int64(bitLength)), nil)
	max.Sub(max, big.NewInt(1))
	min := big.NewInt(0)

	// Declare additional inputs for our integer to mutate and select one.
	// Note: We add some basic cases as well.
	inputs := append(g.baseIntegers, big.NewInt(1), min, max)
	g.randomProviderLock.Lock()
	input := inputs[g.randomProvider.Int() % len(inputs)]

	// Determine how many times to mutate our integer and begin each pass
	mutationCount := g.randomProvider.Int() % (g.maxMutationRounds + 1)
	g.randomProviderLock.Unlock()
	for i := 0; i < mutationCount; i++ {
		// Mutate input
		g.randomProviderLock.Lock()
		input = mutationMethods[g.randomProvider.Int() % len(mutationMethods)](g, input, inputs...)
		g.randomProviderLock.Unlock()

		// Correct value boundaries (underflow/overflow)
		input = g.constrainIntegerToBounds(input, min, max)
	}
	return input
}