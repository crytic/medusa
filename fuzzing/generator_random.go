package fuzzing

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"math/rand"
	"sync"
	"time"
)

type txGeneratorRandom struct {
	// randomProvider offers a source of random data.
	randomProvider *rand.Rand
	// randomProviderLock is a lock to offer thread safety to the random number generator.
	randomProviderLock sync.Mutex
}

func newTxGeneratorRandom() *txGeneratorRandom {
	// Create and return our generator
	generator := &txGeneratorRandom{
		randomProvider: rand.New(rand.NewSource(time.Now().Unix())),
	}
	return generator
}

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

func (g *txGeneratorRandom) generateAddress(worker *fuzzerWorker) common.Address {
	// Generate random bytes of the address length, then convert it to an address.
	addressBytes := make([]byte, common.AddressLength)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(addressBytes)
	g.randomProviderLock.Unlock()
	return common.BytesToAddress(addressBytes)
}

func (g *txGeneratorRandom) generateBool(worker *fuzzerWorker) bool {
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return g.randomProvider.Uint32() % 2 == 0
}

func (g *txGeneratorRandom) generateBytes(worker *fuzzerWorker) []byte {
	g.randomProviderLock.Lock()
	b := make([]byte, rand.Uint64() % 100) // TODO: Right now we only generate 0-100 bytes, make this configurable.
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()
	return b
}

func (g *txGeneratorRandom) generateString(worker *fuzzerWorker) string {
	return string(g.generateBytes(worker))
}

func (g *txGeneratorRandom) generateFixedBytes(worker *fuzzerWorker, length int) []byte {
	g.randomProviderLock.Lock()
	b := make([]byte, length)
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()
	return b
}

func (g *txGeneratorRandom) generateArbitraryUint(worker *fuzzerWorker, bitWidth int) *big.Int {
	// Fill a byte array of the appropriate size with random bytes
	b := make([]byte, bitWidth / 8)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()

	// Convert to a big integer and return
	res := big.NewInt(0)
	return res.SetBytes(b)
}

func (g *txGeneratorRandom) generateUint64(worker *fuzzerWorker) uint64 {
	// Return a random uint64
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return g.randomProvider.Uint64()
}

func (g *txGeneratorRandom) generateUint32(worker *fuzzerWorker) uint32 {
	// Return a random uint32
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return g.randomProvider.Uint32()
}

func (g *txGeneratorRandom) generateUint16(worker *fuzzerWorker) uint16 {
	// Return a random uint16
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return uint16(g.randomProvider.Uint32())
}

func (g *txGeneratorRandom) generateUint8(worker *fuzzerWorker) uint8 {
	// Return a random uint8
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return uint8(g.randomProvider.Uint32())
}

func (g *txGeneratorRandom) generateArbitraryInt(worker *fuzzerWorker, bitWidth int) *big.Int {
	// Fill a byte array of the appropriate size with random bytes
	b := make([]byte, bitWidth / 8)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()

	// Convert to a big integer and return
	res := big.NewInt(0)
	return res.SetBytes(b)
}

func (g *txGeneratorRandom) generateInt64(worker *fuzzerWorker) int64 {
	// Return a random int64
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return int64(g.randomProvider.Uint64())
}

func (g *txGeneratorRandom) generateInt32(worker *fuzzerWorker) int32 {
	// Return a random int32
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return int32(g.randomProvider.Uint32())
}

func (g *txGeneratorRandom) generateInt16(worker *fuzzerWorker) int16 {
	// Return a random int16
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return int16(g.randomProvider.Uint32())
}

func (g *txGeneratorRandom) generateInt8(worker *fuzzerWorker) int8 {
	// Return a random int8
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return int8(g.randomProvider.Uint32())
}