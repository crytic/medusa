package value_generation

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/utils"
	"math/big"
	"math/rand"
	"sync"
	"time"
)

// ValueGeneratorRandom represents an interface for a provider used to generate transaction fields and call arguments
// using a random provider. As such it may not be accurate in many test results with tightly-bound pre-conditions.
type ValueGeneratorRandom struct {
	// randomProvider offers a source of random data.
	randomProvider *rand.Rand
	// randomProviderLock is a lock to offer thread safety to the random number generator.
	randomProviderLock sync.Mutex
}

// NewValueGeneratorRandom creates a new ValueGeneratorRandom with a new random provider.
func NewValueGeneratorRandom() *ValueGeneratorRandom {
	// Create and return our generator
	generator := &ValueGeneratorRandom{
		randomProvider: rand.New(rand.NewSource(time.Now().Unix())),
	}
	return generator
}

// GenerateAddress generates a random address to use when populating inputs.
func (g *ValueGeneratorRandom) GenerateAddress() common.Address {
	// Generate random bytes of the address length, then convert it to an address.
	addressBytes := make([]byte, common.AddressLength)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(addressBytes)
	g.randomProviderLock.Unlock()
	return common.BytesToAddress(addressBytes)
}

// GenerateArrayLength generates a random array length to use when populating inputs.
func (g *ValueGeneratorRandom) GenerateArrayLength() int {
	return int(g.GenerateInteger(false, 16).Uint64() % 100) // TODO: Right now we only generate 0-100 elements, make this configurable.
}

// GenerateBool generates a random bool to use when populating inputs.
func (g *ValueGeneratorRandom) GenerateBool() bool {
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return g.randomProvider.Uint32()%2 == 0
}

// GenerateBytes generates a random dynamic-sized byte array to use when populating inputs.
func (g *ValueGeneratorRandom) GenerateBytes() []byte {
	g.randomProviderLock.Lock()
	b := make([]byte, rand.Uint64()%100) // TODO: Right now we only generate 0-100 bytes, make this configurable.
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()
	return b
}

// GenerateFixedBytes generates a random fixed-sized byte array to use when populating inputs.
func (g *ValueGeneratorRandom) GenerateFixedBytes(length int) []byte {
	g.randomProviderLock.Lock()
	b := make([]byte, length)
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()
	return b
}

// GenerateString generates a random dynamic-sized string to use when populating inputs.
func (g *ValueGeneratorRandom) GenerateString() string {
	return string(g.GenerateBytes())
}

// GenerateInteger generates a random integer to use when populating inputs.
func (g *ValueGeneratorRandom) GenerateInteger(signed bool, bitLength int) *big.Int {
	// Fill a byte array of the appropriate size with random bytes
	b := make([]byte, bitLength/8)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()

	// Create an unsigned integer.
	res := big.NewInt(0).SetBytes(b)

	// Constrain our integer bounds
	return utils.ConstrainIntegerToBitLength(res, signed, bitLength)
}
