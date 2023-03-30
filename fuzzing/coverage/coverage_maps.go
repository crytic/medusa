package coverage

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"sync"
)

// CoverageMaps represents a data structure used to identify instruction execution coverage of various smart contracts
// across a transaction or multiple transactions.
type CoverageMaps struct {
	// maps represents a structure used to track every CoverageMapData by a given deployed address/code hash.
	maps map[common.Hash]map[common.Address]*CoverageMapData

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
	cachedMap *CoverageMapData

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
	cm.maps = make(map[common.Hash]map[common.Address]*CoverageMapData)
}

// GetCoverageMapData obtains a total coverage map representing coverage for the provided bytecode.
// The bytecode matching is done first through an embedded metadata hash, then as a fallback option it hashes the
// byte code data. The former is seen as "safe", while the latter is subject to various failures.
// If the provided bytecode could not find coverage maps, nil is returned.
// Returns the total coverage map, or an error if one occurs.
func (cm *CoverageMaps) GetCoverageMapData(bytecode []byte) (*CoverageMapData, error) {
	// Try to extract the embedded contract metadata and its underlying bytecode hash.
	var hash common.Hash
	hashSet := false
	metadata := compilationTypes.ExtractContractMetadata(bytecode)
	if metadata != nil {
		metadataHash := metadata.ExtractBytecodeHash()
		if metadataHash != nil {
			hash = common.BytesToHash(metadataHash)
			hashSet = true
		}
	}

	// Otherwise, obtain a hash of the compiled byte code itself.
	if !hashSet {
		hash = crypto.Keccak256Hash(bytecode)
	}

	// Acquire our thread lock and defer our unlocking for when we exit this method
	cm.updateLock.Lock()
	defer cm.updateLock.Unlock()

	// Loop through all coverage maps for this hash and collect our total coverage.
	if coverageByAddresses, ok := cm.maps[hash]; ok {
		totalCoverage := &CoverageMapData{}
		for _, coverage := range coverageByAddresses {
			_, err := totalCoverage.updateCodeCoverageData(coverage)
			if err != nil {
				return nil, err
			}
		}
		return totalCoverage, nil
	} else {
		return nil, nil
	}
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
	for codeHash, mapsByAddressToMerge := range coverageMaps.maps {
		for codeAddress, coverageMapToMerge := range mapsByAddressToMerge {
			// If a coverage map lookup for this code hash doesn't exist, create the mapping.
			mapsByAddress, codeHashExists := cm.maps[codeHash]
			if !codeHashExists {
				mapsByAddress = make(map[common.Address]*CoverageMapData)
				cm.maps[codeHash] = mapsByAddress
			}

			// If a coverage map for this address already exists in our current mapping, update it with the one
			// to merge. If it doesn't exist, set it to the one to merge.
			if existingCoverageMap, codeAddressExists := mapsByAddress[codeAddress]; codeAddressExists {
				coverageMapChanged, err := existingCoverageMap.updateCodeCoverageData(coverageMapToMerge)
				changed = changed || coverageMapChanged
				if err != nil {
					return changed, err
				}
			} else {
				mapsByAddress[codeAddress] = coverageMapToMerge
				changed = true
			}
		}
	}

	// Return our results
	return changed, nil
}

// SetCoveredAt sets the coverage state of a given program counter location within code coverage data.
func (cm *CoverageMaps) SetCoveredAt(codeAddress common.Address, codeHash common.Hash, init bool, codeSize int, pc uint64) (bool, error) {
	// If the code size is zero, do nothing
	if codeSize == 0 {
		return false, nil
	}

	// Define variables used to update coverage maps and track changes.
	var (
		addedNewMap  bool
		changedInMap bool
		coverageMap  *CoverageMapData
		err          error
	)

	// Try to obtain a coverage map from our cache
	if cm.cachedMap != nil && cm.cachedCodeAddress == codeAddress && cm.cachedCodeHash == codeHash {
		coverageMap = cm.cachedMap
	} else {
		// If a coverage map lookup for this code hash doesn't exist, create the mapping.
		mapsByCodeAddress, codeHashExists := cm.maps[codeHash]
		if !codeHashExists {
			mapsByCodeAddress = make(map[common.Address]*CoverageMapData)
			cm.maps[codeHash] = mapsByCodeAddress
		}

		// Obtain the coverage map for this code address if it already exists. If it does not, create a new one.
		if existingCoverageMap, codeAddressExists := mapsByCodeAddress[codeAddress]; codeAddressExists {
			coverageMap = existingCoverageMap
		} else {
			coverageMap = &CoverageMapData{
				initBytecodeCoverageData:     nil,
				deployedBytecodeCoverageData: nil,
			}
			cm.maps[codeHash][codeAddress] = coverageMap
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
func (cm *CoverageMaps) Equals(b *CoverageMaps) bool {
	// Note: the `map` field is what is being tested for equality. Not the cached values

	// Iterate through all maps
	for codeHash, mapsByAddressA := range cm.maps {
		mapsByAddressB, ok := b.maps[codeHash]
		// Hash is not in b - we're done
		if !ok {
			return false
		}
		for codeAddress, coverageMap := range mapsByAddressA {
			bCoverage, ok := mapsByAddressB[codeAddress]
			// Address is not in b - we're done
			if !ok {
				return false
			}
			// Compare that the deployed bytecode coverages are the same
			equal := bytes.Compare(coverageMap.deployedBytecodeCoverageData, bCoverage.deployedBytecodeCoverageData)
			if equal != 0 {
				return false
			}
			// Compare that the init bytecode coverages are the same
			equal = bytes.Compare(coverageMap.initBytecodeCoverageData, bCoverage.initBytecodeCoverageData)
			if equal != 0 {
				return false
			}
		}
	}
	return true
}

// CoverageMapData represents a data structure used to identify instruction execution coverage of contract byte code.
type CoverageMapData struct {
	// initBytecodeCoverageData represents a list of bytes for each byte of a contract's init bytecode. Non-zero values
	// indicate the program counter executed an instruction at that offset.
	initBytecodeCoverageData []byte
	// deployedBytecodeCoverageData represents a list of bytes for each byte of a contract's deployed bytecode. Non-zero
	// values indicate the program counter executed an instruction at that offset.
	deployedBytecodeCoverageData []byte
}

// updateCodeCoverageData creates updates the current coverage map with the provided one. It returns a boolean indicating whether
// new coverage was achieved, or an error if one was encountered.
func (cm *CoverageMapData) updateCodeCoverageData(coverageMap *CoverageMapData) (bool, error) {
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

// setCodeCoverageDataAt sets the coverage state of a given program counter location within a CoverageMapData.
func (cm *CoverageMapData) setCodeCoverageDataAt(init bool, codeSize int, pc uint64) (bool, error) {
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
