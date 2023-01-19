package randomutils

import (
	"encoding/binary"
	"math/rand"
)

// ForkRandomProvider creates a child random provider from the current random provider by using its random data as
// a seed. This can be leveraged to help increase determinism so multiple go routines can use their own random provider
// derived from an original. Returns the forked child random provider.
func ForkRandomProvider(randomProvider *rand.Rand) *rand.Rand {
	// Create random bytes to use for an int64 random seed.
	b := make([]byte, 8)
	_, err := randomProvider.Read(b)
	if err != nil {
		panic(err)
	}

	// Return a new random provider with our derived seed.
	forkSeed := int64(binary.LittleEndian.Uint64(b))
	return rand.New(rand.NewSource(forkSeed))
}
