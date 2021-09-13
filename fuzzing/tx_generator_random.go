package fuzzing

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/utils"
	"math/big"
	"math/rand"
	"sync"
	"time"
)

// txGeneratorRandom represents an interface for a provider used to generate transaction fields and call arguments
// using a random provider. As such it may not be accurate in many test results with tightly-bound pre-conditions.
type txGeneratorRandom struct {
	// randomProvider offers a source of random data.
	randomProvider *rand.Rand
	// randomProviderLock is a lock to offer thread safety to the random number generator.
	randomProviderLock sync.Mutex
}

// newTxGeneratorRandom creates a new txGeneratorRandom with a new random provider.
func newTxGeneratorRandom() *txGeneratorRandom {
	// Create and return our generator
	generator := &txGeneratorRandom{
		randomProvider: rand.New(rand.NewSource(time.Now().Unix())),
	}
	return generator
}

// chooseMethod selects a random state-changing deployed contract method to target with a transaction.
func (g *txGeneratorRandom) chooseMethod(worker *fuzzerWorker) *deployedMethod {
	// If we have no state changing methods, return nil immediately.
	if len(worker.stateChangingMethods) == 0 {
		return nil
	}

	// Otherwise, we obtain a random state changing method to target.
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return &worker.stateChangingMethods[g.randomProvider.Int() % len(worker.stateChangingMethods)]
}

// chooseSender selects a random account address to send the transaction from.
func (g *txGeneratorRandom) chooseSender(worker *fuzzerWorker) *fuzzerAccount {
	// If we have no state changing methods, return nil immediately.
	if len(worker.fuzzer.accounts) == 0 {
		return nil
	}

	// Otherwise, we obtain a random state changing method to target.
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return &worker.fuzzer.accounts[g.randomProvider.Int() % len(worker.fuzzer.accounts)]
}

// generateAddress generates a random address to use when populating transaction fields.
func (g *txGeneratorRandom) generateAddress(worker *fuzzerWorker) common.Address {
	// Generate random bytes of the address length, then convert it to an address.
	addressBytes := make([]byte, common.AddressLength)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(addressBytes)
	g.randomProviderLock.Unlock()
	return common.BytesToAddress(addressBytes)
}

// generateArrayLength generates a random array length to use when populating transaction fields.
func (g *txGeneratorRandom) generateArrayLength(worker *fuzzerWorker) int {
	return int(g.generateInteger(worker, false, 16).Uint64() % 100)  // TODO: Right now we only generate 0-100 elements, make this configurable.
}

// generateBool generates a random bool to use when populating transaction fields.
func (g *txGeneratorRandom) generateBool(worker *fuzzerWorker) bool {
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return g.randomProvider.Uint32() % 2 == 0
}

// generateBytes generates a random dynamic-sized byte array to use when populating transaction fields.
func (g *txGeneratorRandom) generateBytes(worker *fuzzerWorker) []byte {
	g.randomProviderLock.Lock()
	b := make([]byte, rand.Uint64() % 100) // TODO: Right now we only generate 0-100 bytes, make this configurable.
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()
	return b
}

// generateFixedBytes generates a random fixed-sized byte array to use when populating transaction fields.
func (g *txGeneratorRandom) generateFixedBytes(worker *fuzzerWorker, length int) []byte {
	g.randomProviderLock.Lock()
	b := make([]byte, length)
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()
	return b
}

// generateString generates a random dynamic-sized string to use when populating transaction fields.
func (g *txGeneratorRandom) generateString(worker *fuzzerWorker) string {
	return string(g.generateBytes(worker))
}

// generateUint generates a random unsigned-integer to use when populating transaction fields.
func (g *txGeneratorRandom) generateInteger(worker *fuzzerWorker, signed bool, bitLength int) *big.Int {
	// Fill a byte array of the appropriate size with random bytes
	b := make([]byte, bitLength / 8)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()

	// Create an unsigned integer.
	res := big.NewInt(0).SetBytes(b)

	// Constrain our integer bounds
	return utils.ConstrainIntegerToBitLength(res, signed, bitLength)
}
