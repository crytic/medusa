package valuegeneration

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/crytic/medusa/utils"
	"golang.org/x/exp/slices"
	"math/big"
	"math/rand"
)

// MutatingValueGenerator is a provider used to generate function inputs and call arguments using mutation-based
// approaches against items within a base_value_set.ValueSet, such as AST literals.
type MutatingValueGenerator struct {
	// config describes the configuration defining value generation parameters.
	config *MutatingValueGeneratorConfig

	// ValueSet contains a set of values which the ValueGenerator may use to aid in value generation and mutation
	// operations.
	valueSet *ValueSet

	// RandomValueGenerator is included to inherit from the random generator
	*RandomValueGenerator
}

// MutatingValueGeneratorConfig defines the operating parameters for a MutatingValueGenerator.
type MutatingValueGeneratorConfig struct {
	// MinMutationRounds describes the minimum amount of mutations which should occur when generating a value.
	// This parameter is used when generating a new value by mutating a value in the value set, or when mutating
	// an existing value.
	MinMutationRounds int
	// MaxMutationRounds describes the maximum amount of mutations which should occur when generating a value.
	// This parameter is used when generating a new value by mutating a value in the value set, or when mutating
	// an existing value.
	MaxMutationRounds int

	// GenerateRandomIntegerBias defines the probability in which an address generated by the value generator is
	// entirely random, rather than selected from the ValueSet provided by MutatingValueGenerator.SetValueSet. Value
	// range is [0.0, 1.0].
	GenerateRandomAddressBias float32
	// GenerateRandomIntegerBias defines the probability in which an integer generated by the value generator is
	// entirely random, rather than mutated. Value range is [0.0, 1.0].
	GenerateRandomIntegerBias float32
	// GenerateRandomStringBias defines the probability in which a string generated by the value generator is entirely
	// random, rather than mutated. Value range is [0.0, 1.0].
	GenerateRandomStringBias float32
	// GenerateRandomStringBias defines the probability in which a byte array generated by the value generator is
	// entirely random, rather than mutated. Value range is [0.0, 1.0].
	GenerateRandomBytesBias float32

	// MutateAddressProbability defines the probability in which an existing address value will be mutated by
	// the value generator. Value range is [0.0, 1.0].
	MutateAddressProbability float32
	// MutateArrayStructureProbability defines the probability in which an existing array value will be mutated by
	// the value generator. Value range is [0.0, 1.0].
	MutateArrayStructureProbability float32
	// MutateAddressProbability defines the probability in which an existing boolean value will be mutated by
	// the value generator. Value range is [0.0, 1.0].
	MutateBoolProbability float32
	// MutateBytesProbability defines the probability in which an existing dynamic-sized byte array value will be
	// mutated by the value generator. Value range is [0.0, 1.0].
	MutateBytesProbability float32
	// MutateBytesGenerateNewBias defines the probability that when an existing dynamic-sized byte array will be
	// mutated, it is done so by being replaced with a newly generated one instead. Value range is [0.0, 1.0].
	MutateBytesGenerateNewBias float32
	// MutateFixedBytesProbability defines the probability in which an existing fixed-sized byte array value will be
	// mutated by the value generator. Value range is [0.0, 1.0].
	MutateFixedBytesProbability float32
	// MutateStringProbability defines the probability in which an existing string value will be mutated by
	// the value generator. Value range is [0.0, 1.0].
	MutateStringProbability float32
	// MutateStringGenerateNewBias defines the probability that when an existing string will be mutated,
	// it is done so by being replaced with a newly generated one instead. Value range is [0.0, 1.0].
	MutateStringGenerateNewBias float32
	// MutateIntegerProbability defines the probability in which an existing integer value will be mutated by
	// the value generator. Value range is [0.0, 1.0].
	MutateIntegerProbability float32
	// MutateIntegerGenerateNewBias defines the probability that when an existing integer will be mutated,
	// it is done so by being replaced with a newly generated one instead. Value range is [0.0, 1.0].
	MutateIntegerGenerateNewBias float32

	// RandomValueGeneratorConfig is adhered to in this structure, to power the underlying RandomValueGenerator.
	*RandomValueGeneratorConfig
}

// NewMutatingValueGenerator creates a new MutatingValueGenerator using a provided base_value_set.ValueSet to seed base-values for mutation.
func NewMutatingValueGenerator(config *MutatingValueGeneratorConfig, valueSet *ValueSet, randomProvider *rand.Rand) *MutatingValueGenerator {
	// Create and return our generator
	generator := &MutatingValueGenerator{
		config:               config,
		valueSet:             valueSet,
		RandomValueGenerator: NewRandomValueGenerator(config.RandomValueGeneratorConfig, randomProvider),
	}

	// Ensure some initial values this mutator will depend on for basic mutations to the set.
	generator.valueSet.AddInteger(big.NewInt(0))
	generator.valueSet.AddInteger(big.NewInt(1))
	generator.valueSet.AddInteger(big.NewInt(-1))
	generator.valueSet.AddInteger(big.NewInt(2))
	return generator
}

// getMutationParams takes a length of inputs and returns an initial input index to start with as a base value, as well
// as a random number of mutations which should be performed (within the mutation range specified by the
// ValueGeneratorConfig).
func (g *MutatingValueGenerator) getMutationParams(inputsLen int) (int, int) {
	inputIdx := g.randomProvider.Intn(inputsLen)
	mutationCount := g.randomProvider.Intn(((g.config.MaxMutationRounds - g.config.MinMutationRounds) + 1) + g.config.MinMutationRounds)
	return inputIdx, mutationCount
}

// integerMutationMethods define methods which take a big integer and a set of inputs and
// transform the integer with a random input and operation. This is used in a loop to create
// mutated integer values.
var integerMutationMethods = []func(*MutatingValueGenerator, *big.Int, ...*big.Int) *big.Int{
	func(g *MutatingValueGenerator, x *big.Int, inputs ...*big.Int) *big.Int {
		// Add a random input
		return big.NewInt(0).Add(x, inputs[g.randomProvider.Intn(len(inputs))])
	},
	func(g *MutatingValueGenerator, x *big.Int, inputs ...*big.Int) *big.Int {
		// Subtract a random input
		return big.NewInt(0).Sub(x, inputs[g.randomProvider.Intn(len(inputs))])
	},
	func(g *MutatingValueGenerator, x *big.Int, inputs ...*big.Int) *big.Int {
		// Multiply a random input
		return big.NewInt(0).Mul(x, inputs[g.randomProvider.Intn(len(inputs))])
	},
	func(g *MutatingValueGenerator, x *big.Int, inputs ...*big.Int) *big.Int {
		// Divide a random input
		divisor := inputs[g.randomProvider.Intn(len(inputs))]
		if divisor.Cmp(big.NewInt(0)) == 0 {
			return big.NewInt(1) // leave unchanged if divisor was zero (would've caused panic)
		}
		return big.NewInt(0).Div(x, divisor)
	},
	func(g *MutatingValueGenerator, x *big.Int, inputs ...*big.Int) *big.Int {
		// Modulo divide a random input
		divisor := inputs[g.randomProvider.Intn(len(inputs))]
		if divisor.Cmp(big.NewInt(0)) == 0 {
			return big.NewInt(0).Set(x) // leave unchanged if divisor was zero (would've caused panic)
		}
		return big.NewInt(0).Mod(x, divisor)
	},
}

// mutateIntegerInternal takes an integer input and returns either a random new integer, or a mutated value based off the input.
// If a nil input is provided, this method uses an existing base value set value as the starting point for mutation.
func (g *MutatingValueGenerator) mutateIntegerInternal(i *big.Int, signed bool, bitLength int) *big.Int {
	// If our bias directs us to, use the random generator instead
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.GenerateRandomIntegerBias {
		return g.RandomValueGenerator.GenerateInteger(signed, bitLength)
	}

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

	// Determine which value we'll use as an initial input, and how many mutations we will perform.
	inputIdx, mutationCount := g.getMutationParams(len(inputs))
	input := new(big.Int)
	if i != nil {
		input.Set(i)
	} else {
		input.Set(inputs[inputIdx])
	}
	input = utils.ConstrainIntegerToBounds(input, min, max)

	// Perform the appropriate number of mutations.
	for i := 0; i < mutationCount; i++ {
		// Mutate input
		input = integerMutationMethods[g.randomProvider.Intn(len(integerMutationMethods))](g, input, inputs...)

		// Correct value boundaries (underflow/overflow)
		input = utils.ConstrainIntegerToBounds(input, min, max)
	}
	return input
}

// bytesMutationMethods define methods which take an initial bytes and a set of inputs to transform the input. The
// transformed input is returned. This is used in a loop to mutate byte slices.
var bytesMutationMethods = []func(*MutatingValueGenerator, []byte, ...[]byte) []byte{
	// Replace a random index with a random byte
	func(g *MutatingValueGenerator, b []byte, inputs ...[]byte) []byte {
		// Generate a random byte and replace an existing byte in our array with it. If our array has no bytes, we add
		// it.
		randomByteValue := byte(g.randomProvider.Intn(256))
		if len(b) > 0 {
			b[g.randomProvider.Intn(len(b))] = randomByteValue
		} else {
			b = append(b, randomByteValue)
		}
		return b
	},
	// Flip a random bit in it.
	func(g *MutatingValueGenerator, b []byte, inputs ...[]byte) []byte {
		// If we have bytes in our array, flip a random bit in a random byte. Otherwise, we add a random byte.
		if len(b) > 0 {
			i := g.randomProvider.Intn(len(b))
			b[i] = b[i] ^ (1 << (g.randomProvider.Intn(8)))
		} else {
			b = append(b, byte(g.randomProvider.Intn(256)))
		}
		return b
	},
	// Add a random byte at a random position
	func(g *MutatingValueGenerator, b []byte, inputs ...[]byte) []byte {
		// Generate a random byte to insert
		by := byte(g.randomProvider.Intn(256))

		// If our provided byte array has no bytes, simply return a new array with this byte.
		if len(b) == 0 {
			return []byte{by}
		}

		// Determine the index to insert our byte into and insert it accordingly. We add +1 here as we allow appending
		// to the end here.
		i := g.randomProvider.Intn(len(b) + 1)
		if i >= len(b) {
			return append(b, by)
		} else {
			return append(b[:i], append([]byte{by}, b[i:]...)...)
		}
	},
	// Remove a random byte
	func(g *MutatingValueGenerator, b []byte, inputs ...[]byte) []byte {
		// If we have no bytes to remove, do nothing.
		if len(b) == 0 {
			return b
		}

		i := g.randomProvider.Intn(len(b))
		return append(b[:i], b[i+1:]...)
	},
}

// mutateBytesInternal takes a byte array and returns either a random new byte array, or a mutated value based off the
// input.
// If a nil input is provided, this method uses an existing base value set value as the starting point for mutation.
func (g *MutatingValueGenerator) mutateBytesInternal(b []byte) []byte {
	// If we have no inputs or our bias directs us to, use the random generator instead
	inputs := g.valueSet.Bytes()
	randomGeneratorDecision := g.randomProvider.Float32()
	if len(inputs) == 0 || randomGeneratorDecision < g.config.GenerateRandomBytesBias {
		return g.RandomValueGenerator.GenerateBytes()
	}

	// Determine which value we'll use as an initial input, and how many mutations we will perform.
	inputIdx, mutationCount := g.getMutationParams(len(inputs))
	var input []byte
	if b != nil {
		input = slices.Clone(b)
	} else {
		input = slices.Clone(inputs[inputIdx])
	}

	// Mutate the data for our desired number of rounds
	for i := 0; i < mutationCount; i++ {
		input = bytesMutationMethods[g.randomProvider.Intn(len(bytesMutationMethods))](g, input, inputs...)
	}

	return input
}

// stringMutationMethods define methods which take an initial string and a set of inputs to transform the input. The
// transformed input is returned. This is used in a loop to mutate strings.
var stringMutationMethods = []func(*MutatingValueGenerator, string, ...string) string{
	// Replace a random index with a random character
	func(g *MutatingValueGenerator, s string, inputs ...string) string {
		// Generate a random rune
		randomRune := rune(32 + g.randomProvider.Intn(95))

		// If the string is empty, we can simply return a new string with just the rune in it.
		r := []rune(s)
		if len(r) == 0 {
			return string(randomRune)
		}

		// Otherwise, we replace a rune in it and return it.
		r[g.randomProvider.Intn(len(r))] = randomRune
		return string(r)
	},
	// Flip a random bit
	func(g *MutatingValueGenerator, s string, inputs ...string) string {
		// If the string is empty, simply return a new one with a randomly added character.
		r := []rune(s)
		if len(r) == 0 {
			return string(rune(32 + g.randomProvider.Int()%95))
		}

		// Otherwise, flip a random bit in it and return it.
		i := g.randomProvider.Intn(len(r))
		r[i] = r[i] ^ (1 << (g.randomProvider.Int() % 8))
		return string(r)
	},
	// Insert a random character at a random position
	func(g *MutatingValueGenerator, s string, inputs ...string) string {
		// Create a random character.
		c := string(rune(32 + g.randomProvider.Intn(95)))

		// If we have an empty string, simply return it
		if len(s) == 0 {
			return c
		}

		// Otherwise we insert it into a random position in the string.
		i := g.randomProvider.Intn(len(s))
		return s[:i] + c + s[i+1:]
	},
	// Remove a random character
	func(g *MutatingValueGenerator, s string, inputs ...string) string {
		// If we have no characters to remove, do nothing
		if len(s) == 0 {
			return s
		}

		// Otherwise, remove a random character.
		i := g.randomProvider.Intn(len(s))
		return s[:i] + s[i+1:]
	},
}

// mutateStringInternal takes a string and returns either a random new string, or a mutated value based off the input.
// If a nil input is provided, this method uses an existing base value set value as the starting point for mutation.
func (g *MutatingValueGenerator) mutateStringInternal(s *string) string {
	// If we have no inputs or our bias directs us to, use the random generator instead
	inputs := g.valueSet.Strings()
	randomGeneratorDecision := g.randomProvider.Float32()
	if len(inputs) == 0 || randomGeneratorDecision < g.config.GenerateRandomStringBias {
		return g.RandomValueGenerator.GenerateString()
	}

	// Obtain a random input to mutate
	inputIdx, mutationCount := g.getMutationParams(len(inputs))
	var input string
	if s != nil {
		input = *s
	} else {
		input = inputs[inputIdx]
	}

	// Mutate the data for our desired number of rounds
	for i := 0; i < mutationCount; i++ {
		input = stringMutationMethods[g.randomProvider.Intn(len(stringMutationMethods))](g, input, inputs...)
	}

	return input
}

// GenerateAddress obtains an existing address from its underlying value set or generates a random one.
func (g *MutatingValueGenerator) GenerateAddress() common.Address {
	// If our bias directs us to, use the random generator instead
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.GenerateRandomAddressBias {
		return g.RandomValueGenerator.GenerateAddress()
	}

	// Obtain our addresses from our value set. If we have none, generate a random one instead.
	addresses := g.valueSet.Addresses()
	if len(addresses) == 0 {
		return g.RandomValueGenerator.GenerateAddress()
	}

	// Select a random address from our set of addresses.
	address := addresses[g.randomProvider.Intn(len(addresses))]
	return address
}

// MutateAddress takes an address input and sometimes returns a mutated value based off the input.
func (g *MutatingValueGenerator) MutateAddress(addr common.Address) common.Address {
	// Determine whether to perform mutations against this input or just return it as-is.
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.MutateAddressProbability {
		return g.RandomValueGenerator.GenerateAddress()
	}
	return addr
}

// MutateArray takes a dynamic or fixed sized array as input, and returns a mutated value based off of the input.
// Returns the mutated value. If any element of the returned array is nil, the value generator will be called upon
// to generate it new.
func (g *MutatingValueGenerator) MutateArray(value []any, fixedLength bool) []any {
	// Determine whether to perform mutations against this input or just return it as-is.
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.MutateArrayStructureProbability {
		// Determine how many mutations we'll apply
		_, mutationCount := g.getMutationParams(1)
		for i := 0; i < mutationCount; i++ {
			// TODO: Apply array structure mutations (swap, insert, delete)
		}
		return value
	}
	return value
}

// MutateBool takes a boolean input and returns a mutated value based off the input.
func (g *MutatingValueGenerator) MutateBool(bl bool) bool {
	// Determine whether to perform mutations against this input or just return it as-is.
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.MutateBoolProbability {
		return g.RandomValueGenerator.GenerateBool()
	}
	return bl
}

// GenerateBytes generates bytes and returns them.
func (g *MutatingValueGenerator) GenerateBytes() []byte {
	return g.mutateBytesInternal(nil)
}

// MutateBytes takes a dynamic-sized byte array input and returns a mutated value based off the input.
func (g *MutatingValueGenerator) MutateBytes(b []byte) []byte {
	// Determine whether to perform mutations against this input or just return it as-is.
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.MutateBytesProbability {
		// Determine whether to return a newly generated value, or mutate our existing.
		randomGeneratorDecision = g.randomProvider.Float32()
		if randomGeneratorDecision < g.config.MutateBytesGenerateNewBias {
			return g.GenerateBytes()
		} else {
			return g.mutateBytesInternal(b)
		}
	}
	return b
}

// MutateFixedBytes takes a fixed-sized byte array input and returns a mutated value based off the input.
func (g *MutatingValueGenerator) MutateFixedBytes(b []byte) []byte {
	// Determine whether to perform mutations against this input or just return it as-is.
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.MutateFixedBytesProbability {
		return g.GenerateFixedBytes(len(b))
	}
	return b
}

// GenerateString generates strings and returns them.
func (g *MutatingValueGenerator) GenerateString() string {
	return g.mutateStringInternal(nil)
}

// MutateString takes a string input and returns a mutated value based off the input.
func (g *MutatingValueGenerator) MutateString(s string) string {
	// Determine whether to perform mutations against this input or just return it as-is.
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.MutateStringProbability {
		// Determine whether to return a newly generated value, or mutate our existing.
		randomGeneratorDecision = g.randomProvider.Float32()
		if randomGeneratorDecision < g.config.MutateStringGenerateNewBias {
			return g.GenerateString()
		} else {
			return g.mutateStringInternal(&s)
		}
	}
	return s
}

// GenerateInteger generates an integer of the provided properties and returns a big.Int representing it.
func (g *MutatingValueGenerator) GenerateInteger(signed bool, bitLength int) *big.Int {
	// Call our internal mutation method with no starting input. This will generate a new input.
	return g.mutateIntegerInternal(nil, signed, bitLength)
}

// MutateInteger takes an integer input and applies optional mutations to the provided value.
// Returns an optionally mutated copy of the input.
func (g *MutatingValueGenerator) MutateInteger(i *big.Int, signed bool, bitLength int) *big.Int {
	// Determine whether to perform mutations against this input or just return it as-is.
	randomGeneratorDecision := g.randomProvider.Float32()
	if randomGeneratorDecision < g.config.MutateIntegerProbability {
		// Determine whether to return a newly generated value, or mutate our existing.
		randomGeneratorDecision = g.randomProvider.Float32()
		if randomGeneratorDecision < g.config.MutateIntegerGenerateNewBias {
			return g.GenerateInteger(signed, bitLength)
		} else {
			return g.mutateIntegerInternal(i, signed, bitLength)
		}
	}
	return i
}
