package valuegeneration

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/utils"
	"math/big"
	"math/rand"
)

// RandomValueGenerator represents an interface for a provider used to generate transaction fields and call arguments
// using a random provider. As such it may not be accurate in many test results with tightly-bound pre-conditions.
type RandomValueGenerator struct {
	// config describes the configuration defining value generation parameters.
	config *RandomValueGeneratorConfig

	// randomProvider offers a source of random data.
	randomProvider *rand.Rand
}

// RandomValueGeneratorConfig defines the parameters for a RandomValueGenerator.
type RandomValueGeneratorConfig struct {
	// RandomArrayMinSize defines the minimum size which a generated array should be.
	RandomArrayMinSize int
	// RandomArrayMaxSize defines the maximum size which a generated array should be.
	RandomArrayMaxSize int
	// RandomBytesMinSize defines the minimum size which a generated byte slice should be.
	RandomBytesMinSize int
	// RandomBytesMaxSize defines the maximum size which a generated byte slice should be.
	RandomBytesMaxSize int
	// RandomStringMinSize defines the minimum size which a generated string should be.
	RandomStringMinSize int
	// RandomStringMaxSize defines the maximum size which a generated string should be.
	RandomStringMaxSize int
}

// NewRandomValueGenerator creates a new RandomValueGenerator with a new random provider.
func NewRandomValueGenerator(config *RandomValueGeneratorConfig, randomProvider *rand.Rand) *RandomValueGenerator {
	// Create and return our generator
	generator := &RandomValueGenerator{
		config:         config,
		randomProvider: randomProvider,
	}
	return generator
}

// GenerateAddress generates a random address to use when populating inputs.
func (g *RandomValueGenerator) GenerateAddress() common.Address {
	// Generate random bytes of the address length, then convert it to an address.
	addressBytes := make([]byte, common.AddressLength)
	g.randomProvider.Read(addressBytes)
	return common.BytesToAddress(addressBytes)
}

// GenerateArrayLength generates a random array length to use when populating inputs. This is used to determine how
// many elements a non-byte, non-string array should have.
func (g *RandomValueGenerator) GenerateArrayLength() int {
	rangeSize := uint64(g.config.RandomArrayMaxSize-g.config.RandomArrayMinSize) + 1
	return int(g.GenerateInteger(false, 16).Uint64()%rangeSize) + g.config.RandomArrayMinSize
}

// GenerateBool generates a random bool to use when populating inputs.
func (g *RandomValueGenerator) GenerateBool() bool {
	return g.randomProvider.Uint32()%2 == 0
}

// GenerateBytes generates a random dynamic-sized byte array to use when populating inputs.
func (g *RandomValueGenerator) GenerateBytes() []byte {
	rangeSize := uint64(g.config.RandomBytesMaxSize-g.config.RandomBytesMinSize) + 1
	b := make([]byte, int(g.randomProvider.Uint64()%rangeSize)+g.config.RandomBytesMinSize)
	g.randomProvider.Read(b)
	return b
}

// GenerateFixedBytes generates a random fixed-sized byte array to use when populating inputs.
func (g *RandomValueGenerator) GenerateFixedBytes(length int) []byte {
	b := make([]byte, length)
	g.randomProvider.Read(b)
	return b
}

// GenerateString generates a random dynamic-sized string to use when populating inputs.
func (g *RandomValueGenerator) GenerateString() string {
	rangeSize := uint64(g.config.RandomStringMaxSize-g.config.RandomStringMinSize) + 1
	b := make([]byte, int(g.randomProvider.Uint64()%rangeSize)+g.config.RandomStringMinSize)
	g.randomProvider.Read(b)
	return string(b)
}

// GenerateInteger generates a random integer to use when populating inputs.
func (g *RandomValueGenerator) GenerateInteger(signed bool, bitLength int) *big.Int {
	// Fill a byte array of the appropriate size with random bytes
	b := make([]byte, bitLength/8)
	g.randomProvider.Read(b)

	// Create an unsigned integer.
	res := big.NewInt(0).SetBytes(b)

	// Constrain our integer bounds
	return utils.ConstrainIntegerToBitLength(res, signed, bitLength)
}

// RandomProvider returns the internal random provider used for value generation.
func (g *RandomValueGenerator) RandomProvider() *rand.Rand {
	return g.randomProvider
}
