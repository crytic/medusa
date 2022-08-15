package tracing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
)

// CoverageMap represents a data structure used to identify instruction execution coverage of a smart contract.
type CoverageMap struct {
	// codeHash represents the keccak256 hash of the code we're collecting coverage for. This can be used to match
	// a coverage map to its respective byte code.
	codeHash common.Hash

	// executedBytes represents a list of bytes for each byte of a deployed smart contract where zero values indicate
	// execution did not occur at the given position, while any other value indicates code execution occurred at the
	// given smart contract offset.
	executedBytes []byte
}

// NewCoverageMap initializes a new CoverageMap for a given types.CompiledContract.
func NewCoverageMap(codeHash common.Hash, size int) (*CoverageMap, error) {
	// If the size is negative, throw an error
	if size < 0 {
		return nil, fmt.Errorf("cannot create a coverage map with a negative byte code size (%d)", size)
	}

	// Create a coverage map of the requested size
	coverageMap := &CoverageMap{
		codeHash:      codeHash,
		executedBytes: make([]byte, size),
	}
	return coverageMap, nil
}

func (cm *CoverageMap) Add(coverageMap *CoverageMap) (int, error) {
	// Ensure our coverage maps match in size
	if len(cm.executedBytes) != len(coverageMap.executedBytes) {
		return 0, fmt.Errorf("failed to add/merge coverage maps. Map of size %d cannot be merged into map of size %d", len(coverageMap.executedBytes), len(cm.executedBytes))
	}

	// OR each byte of our coverage map with the provided one. This means any non-zero bytes indicating coverage
	// will be merged into the current map.
	changeCount := 0
	for i := 0; i < len(cm.executedBytes); i++ {
		if cm.executedBytes[i] == 0 && coverageMap.executedBytes[i] != 0 {
			cm.executedBytes[i] = 1
			changeCount++
		}
	}
	return changeCount, nil
}
