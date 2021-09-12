package fuzzing

import (
	"math/big"
	"math/rand"
)

type txGeneratorMutation struct {
	// Inherit the random mutation
	txGeneratorRandom
}

func newTxGeneratorMutation() *txGeneratorMutation {
	// Create and return our generator
	generator := &txGeneratorMutation{

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

func (g *txGeneratorMutation) generateArbitraryUint(worker *fuzzerWorker, bitWidth int) *big.Int {
	// Fill a byte array of the appropriate size with random bytes
	b := make([]byte, bitWidth / 8)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()

	// Convert to a big integer and return
	res := big.NewInt(0)
	return res.SetBytes(b)
}

func (g *txGeneratorMutation) generateUint64(worker *fuzzerWorker) uint64 {
	// Return a random uint64
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return g.randomProvider.Uint64()
}

func (g *txGeneratorMutation) generateUint32(worker *fuzzerWorker) uint32 {
	// Return a random uint32
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return g.randomProvider.Uint32()
}

func (g *txGeneratorMutation) generateUint16(worker *fuzzerWorker) uint16 {
	// Return a random uint16
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return uint16(g.randomProvider.Uint32())
}

func (g *txGeneratorMutation) generateUint8(worker *fuzzerWorker) uint8 {
	// Return a random uint8
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return uint8(g.randomProvider.Uint32())
}

func (g *txGeneratorMutation) generateArbitraryInt(worker *fuzzerWorker, bitWidth int) *big.Int {
	// Fill a byte array of the appropriate size with random bytes
	b := make([]byte, bitWidth / 8)
	g.randomProviderLock.Lock()
	g.randomProvider.Read(b)
	g.randomProviderLock.Unlock()

	// Convert to a big integer and return
	res := big.NewInt(0)
	return res.SetBytes(b)
}

func (g *txGeneratorMutation) generateInt64(worker *fuzzerWorker) int64 {
	// Return a random int64
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return int64(g.randomProvider.Uint64())
}

func (g *txGeneratorMutation) generateInt32(worker *fuzzerWorker) int32 {
	// Return a random int32
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return int32(g.randomProvider.Uint32())
}

func (g *txGeneratorMutation) generateInt16(worker *fuzzerWorker) int16 {
	// Return a random int16
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return int16(g.randomProvider.Uint32())
}

func (g *txGeneratorMutation) generateInt8(worker *fuzzerWorker) int8 {
	// Return a random int8
	g.randomProviderLock.Lock()
	defer g.randomProviderLock.Unlock()
	return int8(g.randomProvider.Uint32())
}