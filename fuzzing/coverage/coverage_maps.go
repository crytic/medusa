package coverage

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

// CoverageMaps represents a data structure used to identify instruction execution coverage of various smart contracts
// across a transaction or multiple transactions.
type CoverageMaps struct {
	// maps represents a structure used to track every codeCoverageData by a given deployed address/code hash.
	maps map[common.Address]map[common.Hash]*codeCoverageData

	// cachedCodeAddress represents the last code address which coverage was updated for. This is used to prevent an
	// expensive lookup in maps. If cachedCodeHash does not match the current code address for which we are updating
	// coverage for, it, along with other cache variables are updated.
	cachedCodeAddress common.Address

	// cachedCodeHash represents the last code hash which coverage was updated for. This is used to prevent an expensive
	// lookup in maps. If cachedCodeHash does not match the current code hash for which we are updating coverage for,
	// it, along with other cache variables are updated.
	cachedCodeHash common.Hash

	// cachedMap represents the last coverage map which was updated. If the coverage to update resides at the
	// cachedCodeAddress and matches the cachedCodeHash, then this map is used to avoid an expensive lookup into maps.
	cachedMap *codeCoverageData

	// updateLock is a lock to offer concurrent thread safety for map accesses.
	updateLock sync.Mutex
}

// NewCoverageMaps initializes a new CoverageMaps object.
func NewCoverageMaps() *CoverageMaps {
	maps := &CoverageMaps{}
	maps.Reset()
	return maps
}

// Reset clears the coverage state for the CoverageMaps.
func (cm *CoverageMaps) Reset() {
	cm.maps = make(map[common.Address]map[common.Hash]*codeCoverageData)
}

// Update updates the current coverage maps with the provided ones. It returns a boolean indicating whether
// new coverage was achieved, or an error if one was encountered.
func (cm *CoverageMaps) Update(coverageMaps *CoverageMaps) (bool, error) {
	// If our maps provided are nil, do nothing
	if coverageMaps == nil {
		return false, nil
	}

	// Acquire our thread lock and defer our unlocking for when we exit this method
	cm.updateLock.Lock()
	defer cm.updateLock.Unlock()

	// Create a boolean indicating whether we achieved new coverage
	changed := false

	// Loop for each coverage map provided
	for codeAddressToMerge, mapsByCodeHashToMerge := range coverageMaps.maps {
		for codeHashToMerge, coverageMapToMerge := range mapsByCodeHashToMerge {
			// If a coverage map lookup for this code address doesn't exist, create the mapping.
			mapsByCodeHash, codeAddressExists := cm.maps[codeAddressToMerge]
			if !codeAddressExists {
				mapsByCodeHash = make(map[common.Hash]*codeCoverageData)
				cm.maps[codeAddressToMerge] = mapsByCodeHash
			}

			// If a coverage map for this code hash already exists in our current mapping, update it with the one
			// to merge. If it doesn't exist, set it to the one to merge.
			if existingCoverageMap, codeHashExists := mapsByCodeHash[codeHashToMerge]; codeHashExists {
				coverageMapChanged, err := existingCoverageMap.updateCodeCoverageData(coverageMapToMerge)
				changed = changed || coverageMapChanged
				if err != nil {
					return changed, err
				}
			} else {
				mapsByCodeHash[codeHashToMerge] = coverageMapToMerge
				changed = true
			}
		}
	}

	// Return our results
	return changed, nil
}

// SetCoveredAt sets the coverage state of a given program counter location within a codeCoverageData.
func (cm *CoverageMaps) SetCoveredAt(codeAddress common.Address, codeHash common.Hash, init bool, codeSize int, pc uint64) (bool, error) {
	// If the code size is zero, do nothing
	if codeSize == 0 {
		return false, nil
	}

	// Define variables used to update coverage maps and track changes.
	var (
		addedNewMap  bool
		changedInMap bool
		coverageMap  *codeCoverageData
		err          error
	)

	// Try to obtain a coverage map for the given code hash from our cache
	if cm.cachedMap != nil && cm.cachedCodeAddress == codeAddress && cm.cachedCodeHash == codeHash {
		coverageMap = cm.cachedMap
	} else {
		// If a coverage map lookup for this code address doesn't exist, create the mapping.
		coverageMapsByCodeHash, codeAddressExists := cm.maps[codeAddress]
		if !codeAddressExists {
			coverageMapsByCodeHash = make(map[common.Hash]*codeCoverageData)
			cm.maps[codeAddress] = coverageMapsByCodeHash
		}

		// Obtain the coverage map for this code hash if it already exists. If it does not, create a new one.
		if existingCoverageMap, codeHashExists := coverageMapsByCodeHash[codeHash]; codeHashExists {
			coverageMap = existingCoverageMap
		} else {
			coverageMap = &codeCoverageData{
				initBytecodeCoverageData:     nil,
				deployedBytecodeCoverageData: nil,
			}
			cm.maps[codeAddress][codeHash] = coverageMap
			addedNewMap = true
		}

		// Set our cached variables for faster coverage setting next time this method is called.
		cm.cachedMap = coverageMap
		cm.cachedCodeHash = codeHash
		cm.cachedCodeAddress = codeAddress
	}

	// Set our coverage in the map and return our change state
	changedInMap, err = coverageMap.setCodeCoverageDataAt(init, codeSize, pc)
	return addedNewMap || changedInMap, err
}

// Equals checks whether two coverage maps are the same. Equality is determined if the keys and values are all the same.
func (a *CoverageMaps) Equals(b *CoverageMaps) bool {
	// Note: the `map` field is what is being tested for equality. Not the cached values

	// Iterate through all maps
	for addr, aHashToCoverage := range a.maps {
		bHashToCoverage, ok := b.maps[addr]
		// Address is not in b - we're done
		if !ok {
			return false
		}
		for hash, aCoverage := range aHashToCoverage {
			bCoverage, ok := bHashToCoverage[hash]
			// Hash is not in b - we're done
			if !ok {
				return false
			}
			// Compare that the deployed bytecode coverages are the same
			equal := bytes.Compare(aCoverage.deployedBytecodeCoverageData, bCoverage.deployedBytecodeCoverageData)
			if equal != 0 {
				return false
			}
			// Compare that the init bytecode coverages are the same
			equal = bytes.Compare(aCoverage.initBytecodeCoverageData, bCoverage.initBytecodeCoverageData)
			if equal != 0 {
				return false
			}
		}
	}
	return true
}

// codeCoverageData represents a data structure used to identify instruction execution coverage of contract byte code.
type codeCoverageData struct {
	// initBytecodeCoverageData represents a list of bytes for each byte of a contract's init bytecode. Non-zero values
	// indicate the program counter executed an instruction at that offset.
	initBytecodeCoverageData []byte
	// deployedBytecodeCoverageData represents a list of bytes for each byte of a contract's deployed bytecode. Non-zero
	// values indicate the program counter executed an instruction at that offset.
	deployedBytecodeCoverageData []byte
}

// updateCodeCoverageData creates updates the current coverage map with the provided one. It returns a boolean indicating whether
// new coverage was achieved, or an error if one was encountered.
func (cm *codeCoverageData) updateCodeCoverageData(coverageMap *codeCoverageData) (bool, error) {
	// Define our return variable
	changed := false

	// Update our init bytecode coverage data.
	if coverageMap.initBytecodeCoverageData != nil {
		if cm.initBytecodeCoverageData == nil {
			cm.initBytecodeCoverageData = coverageMap.initBytecodeCoverageData
			changed = true
		} else {
			// Update each byte which represents a position in the bytecode which was covered. We ignore any size
			// differences as init bytecode can have arbitrary length arguments appended.
			for i := 0; i < len(cm.initBytecodeCoverageData) || i < len(coverageMap.initBytecodeCoverageData); i++ {
				if cm.initBytecodeCoverageData[i] == 0 && coverageMap.initBytecodeCoverageData[i] != 0 {
					cm.initBytecodeCoverageData[i] = 1
					changed = true
				}
			}
		}
	}

	// Update our deployed bytecode coverage data.
	if coverageMap.deployedBytecodeCoverageData != nil {
		if cm.deployedBytecodeCoverageData == nil {
			cm.deployedBytecodeCoverageData = coverageMap.deployedBytecodeCoverageData
			changed = true
		} else {
			// Update each byte which represents a position in the bytecode which was covered.
			for i := 0; i < len(cm.deployedBytecodeCoverageData); i++ {
				if cm.deployedBytecodeCoverageData[i] == 0 && coverageMap.deployedBytecodeCoverageData[i] != 0 {
					cm.deployedBytecodeCoverageData[i] = 1
					changed = true
				}
			}
		}
	}

	return changed, nil
}

// setCodeCoverageDataAt sets the coverage state of a given program counter location within a codeCoverageData.
func (cm *codeCoverageData) setCodeCoverageDataAt(init bool, codeSize int, pc uint64) (bool, error) {
	// Obtain our coverage data depending on if we're initializing/deploying a contract now. If coverage data doesn't
	// exist, we create it.
	var coverageData []byte
	if init {
		if cm.initBytecodeCoverageData == nil {
			cm.initBytecodeCoverageData = make([]byte, codeSize)
		}
		coverageData = cm.initBytecodeCoverageData
	} else {
		if cm.deployedBytecodeCoverageData == nil {
			cm.deployedBytecodeCoverageData = make([]byte, codeSize)
		}
		coverageData = cm.deployedBytecodeCoverageData
	}

	// If our program counter is in range, determine if we achieved new coverage for the first time, and update it.
	if pc < uint64(len(coverageData)) {
		if coverageData[pc] == 0 {
			coverageData[pc] = 1
			return true, nil
		}
		return false, nil
	}
	return false, fmt.Errorf("tried to set coverage map out of bounds (pc: %d, code size %d)", pc, len(coverageData))
}
