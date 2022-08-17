package tracing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// CoverageMaps represents a data structure used to identify instruction execution coverage of various smart contracts
// across a transaction or multiple transactions.
type CoverageMaps struct {
	maps map[common.Hash]*CoverageMap

	cachedCodeHash common.Hash
	cachedMap      *CoverageMap
}

// NewCoverageMaps initializes a new CoverageMaps object.
func NewCoverageMaps() *CoverageMaps {
	return &CoverageMaps{
		maps: make(map[common.Hash]*CoverageMap),
	}
}

// Update updates the current coverage maps with the provided ones. It returns a boolean indicating whether
// new coverage was achieved, or an error if one was encountered.
func (cm *CoverageMaps) Update(coverageMaps *CoverageMaps) (bool, error) {
	// Create a boolean indicating whether we achieved new coverage
	changed := false

	// Loop for each coverage map provided
	for codeHash, coverageMap := range coverageMaps.maps {
		// If the code hash has an existing coverage map, add to it. Otherwise, create a new entry using the provided
		// coverage map.
		if existingCoverageMap, ok := cm.maps[codeHash]; ok {
			coverageMapChanged, err := existingCoverageMap.Update(coverageMap)
			if err != nil {
				return false, err
			}
			changed = changed || coverageMapChanged
		} else {
			cm.maps[codeHash] = coverageMap
			changed = true
		}
	}

	// Return our results
	return changed, nil
}

// SetCoveredAt sets the coverage state of a given program counter location within a CoverageMap.
func (cm *CoverageMaps) SetCoveredAt(codeHash common.Hash, codeSize int, pc uint64) (bool, error) {
	// If we have a zero code hash, this is code being deployed so we don't track coverage
	zeroHash := common.BigToHash(big.NewInt(0))
	if codeHash == zeroHash {
		return false, nil
	}

	// Define variables used to update coverage maps and track changes.
	var (
		addedNewMap  bool
		changedInMap bool
		coverageMap  *CoverageMap
		err          error
	)

	// Try to obtain a coverage map for the given code hash from our cache
	if cm.cachedMap != nil && cm.cachedCodeHash == codeHash {
		coverageMap = cm.cachedMap
	} else {
		// If the cached map is not the one we're looking for, we perform a lookup, or create one if one doesn't
		// exist.
		if m, ok := cm.maps[codeHash]; ok {
			coverageMap = m
		} else {
			coverageMap, err = NewCoverageMap(codeHash, codeSize)
			if err != nil {
				return false, nil
			}
			cm.maps[codeHash] = coverageMap
			addedNewMap = true
		}
		cm.cachedMap = coverageMap
		cm.cachedCodeHash = codeHash
	}

	// Set our coverage in the map and return our change state
	changedInMap, err = coverageMap.SetCoveredAt(pc)
	return addedNewMap || changedInMap, err
}

// Reset clears the coverage state for the CoverageMaps.
func (cm *CoverageMaps) Reset() {
	cm.maps = make(map[common.Hash]*CoverageMap)
}

// CoverageMap represents a data structure used to identify instruction execution coverage of smart contract byte code.
type CoverageMap struct {
	// codeHash represents the keccak256 hash of the code we're collecting coverage for. This can be used to match
	// a coverage map to its respective byte code.
	codeHash common.Hash

	// executedBytes represents a list of bytes for each byte of a deployed smart contract where zero values indicate
	// execution did not occur at the given position, while any other value indicates code execution occurred at the
	// given smart contract offset.
	executedBytes []byte
}

// NewCoverageMap initializes a new CoverageMap object.
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

// Update creates updates the current coverage map with the provided one. It returns a boolean indicating whether
// new coverage was achieved, or an error if one was encountered.
func (cm *CoverageMap) Update(coverageMap *CoverageMap) (bool, error) {
	// Ensure our coverage maps match in size
	if len(cm.executedBytes) != len(coverageMap.executedBytes) {
		return false, fmt.Errorf("failed to add/merge coverage maps. Map of size %d cannot be merged into map of size %d", len(coverageMap.executedBytes), len(cm.executedBytes))
	}

	// Update each byte which represents a position in the bytecode which was covered.
	changed := false
	for i := 0; i < len(cm.executedBytes); i++ {
		if cm.executedBytes[i] == 0 && coverageMap.executedBytes[i] != 0 {
			cm.executedBytes[i] = 1
			changed = true
		}
	}
	return changed, nil
}

// SetCoveredAt sets the coverage state of a given program counter location within a CoverageMap.
func (cm *CoverageMap) SetCoveredAt(pc uint64) (bool, error) {
	if pc < uint64(len(cm.executedBytes)) {
		if cm.executedBytes[pc] == 0 {
			cm.executedBytes[pc] = 1
			return true, nil
		}
		return false, nil
	}
	return false, fmt.Errorf("tried to set coverage map out of bounds (pc: %d, code size %d)", pc, len(cm.executedBytes))
}

// Reset clears the coverage state for the CoverageMap.
func (cm *CoverageMap) Reset() {
	cm.executedBytes = make([]byte, len(cm.executedBytes))
}
