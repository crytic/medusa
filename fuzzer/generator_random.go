package fuzzer

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

func (g *txGeneratorRandom) generateUint(worker *fuzzerWorker) *big.Int {
	uintBytes := make([]byte, 32)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(uintBytes)
	g.randomProviderLock.Unlock()

	res := big.NewInt(0)
	return res.SetBytes(uintBytes)
}
