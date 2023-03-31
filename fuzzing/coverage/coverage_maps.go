package coverage

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/utils"
	"sync"
)

// CoverageMaps represents a data structure used to identify instruction execution coverage of various smart contracts
// across a transaction or multiple transactions.
type CoverageMaps struct {
	// maps represents a structure used to track every ContractCoverageMap by a given deployed address/code hash.
	maps map[common.Hash]map[common.Address]*ContractCoverageMap

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
	cachedMap *ContractCoverageMap

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
	cm.maps = make(map[common.Hash]map[common.Address]*ContractCoverageMap)
}

// GetContractCoverageMap obtains a total coverage map representing coverage for the provided bytecode.
// The bytecode matching is done first through an embedded metadata hash, then as a fallback option it hashes the
// byte code data. The former is seen as "safe", while the latter is subject to various failures.
// If the provided bytecode could not find coverage maps, nil is returned.
// Returns the total coverage map, or an error if one occurs.
func (cm *CoverageMaps) GetContractCoverageMap(bytecode []byte) (*ContractCoverageMap, error) {
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
		totalCoverage := newContractCoverageMap()
		for _, coverage := range coverageByAddresses {
			_, err := totalCoverage.update(coverage)
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
				mapsByAddress = make(map[common.Address]*ContractCoverageMap)
				cm.maps[codeHash] = mapsByAddress
			}

			// If a coverage map for this address already exists in our current mapping, update it with the one
			// to merge. If it doesn't exist, set it to the one to merge.
			if existingCoverageMap, codeAddressExists := mapsByAddress[codeAddress]; codeAddressExists {
				coverageMapChanged, err := existingCoverageMap.update(coverageMapToMerge)
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
		coverageMap  *ContractCoverageMap
		err          error
	)

	// Try to obtain a coverage map from our cache
	if cm.cachedMap != nil && cm.cachedCodeAddress == codeAddress && cm.cachedCodeHash == codeHash {
		coverageMap = cm.cachedMap
	} else {
		// If a coverage map lookup for this code hash doesn't exist, create the mapping.
		mapsByCodeAddress, codeHashExists := cm.maps[codeHash]
		if !codeHashExists {
			mapsByCodeAddress = make(map[common.Address]*ContractCoverageMap)
			cm.maps[codeHash] = mapsByCodeAddress
		}

		// Obtain the coverage map for this code address if it already exists. If it does not, create a new one.
		if existingCoverageMap, codeAddressExists := mapsByCodeAddress[codeAddress]; codeAddressExists {
			coverageMap = existingCoverageMap
		} else {
			coverageMap = newContractCoverageMap()
			cm.maps[codeHash][codeAddress] = coverageMap
			addedNewMap = true
		}

		// Set our cached variables for faster coverage setting next time this method is called.
		cm.cachedMap = coverageMap
		cm.cachedCodeHash = codeHash
		cm.cachedCodeAddress = codeAddress
	}

	// Set our coverage in the map and return our change state
	changedInMap, err = coverageMap.setCoveredAt(init, codeSize, pc)
	return addedNewMap || changedInMap, err
}

// Equal checks whether two coverage maps are the same. Equality is determined if the keys and values are all the same.
func (cm *CoverageMaps) Equal(b *CoverageMaps) bool {
	// Note: the `map` field is what is being tested for equality. Not the cached values

	// Iterate through all maps
	for codeHash, mapsByAddressA := range cm.maps {
		mapsByAddressB, ok := b.maps[codeHash]
		// Hash is not in b - we're done
		if !ok {
			return false
		}
		for codeAddress, coverageMapA := range mapsByAddressA {
			coverageMapB, ok := mapsByAddressB[codeAddress]
			// Address is not in b - we're done
			if !ok {
				return false
			}

			// Verify the equality of the map data.
			if !coverageMapA.Equal(coverageMapB) {
				return false
			}
		}
	}
	return true
}

// ContractCoverageMap represents a data structure used to identify instruction execution coverage of a contract.
type ContractCoverageMap struct {
	// initBytecodeCoverage represents a list of bytes for each byte of a contract's init bytecode. Non-zero values
	// indicate the program counter executed an instruction at that offset.
	initBytecodeCoverage *CoverageMapBytecodeData
	// deployedBytecodeCoverage represents a list of bytes for each byte of a contract's deployed bytecode. Non-zero
	// values indicate the program counter executed an instruction at that offset.
	deployedBytecodeCoverage *CoverageMapBytecodeData
}

// newContractCoverageMap creates and returns a new ContractCoverageMap.
func newContractCoverageMap() *ContractCoverageMap {
	return &ContractCoverageMap{
		initBytecodeCoverage:     &CoverageMapBytecodeData{},
		deployedBytecodeCoverage: &CoverageMapBytecodeData{},
	}
}

// update creates updates the current ContractCoverageMap with the provided one.
// Returns a boolean indicating whether new coverage was achieved, or an error if one was encountered.
func (cm *ContractCoverageMap) update(coverageMap *ContractCoverageMap) (bool, error) {
	// Define our return variable
	changed := false

	// Update our init bytecode coverage data
	c, err := cm.initBytecodeCoverage.update(coverageMap.initBytecodeCoverage)
	if err != nil {
		return c, err
	}
	changed = changed || c

	// Update our deployed bytecode coverage data
	c, err = cm.deployedBytecodeCoverage.update(coverageMap.deployedBytecodeCoverage)
	if err != nil {
		return c, err
	}
	changed = changed || c

	return changed, nil
}

// setCoveredAt sets the coverage state at a given program counter location within a ContractCoverageMap.
// Returns a boolean indicating whether new coverage was achieved, or an error if one occurred.
func (cm *ContractCoverageMap) setCoveredAt(init bool, codeSize int, pc uint64) (bool, error) {
	// Set our coverage data for the appropriate map.
	if init {
		return cm.initBytecodeCoverage.setCoveredAt(codeSize, pc)
	} else {
		return cm.deployedBytecodeCoverage.setCoveredAt(codeSize, pc)
	}
}

// Equal checks whether the provided ContractCoverageMap contains the same data as the current one.
// Returns a boolean indicating whether the two maps match.
func (cm *ContractCoverageMap) Equal(b *ContractCoverageMap) bool {
	// Compare that the deployed bytecode coverages are the same
	if !cm.deployedBytecodeCoverage.Equal(b.deployedBytecodeCoverage) {
		return false
	}
	// Compare that the init bytecode coverages are the same
	if !cm.initBytecodeCoverage.Equal(b.initBytecodeCoverage) {
		return false
	}
	return true
}

// CoverageMapBytecodeData represents a data structure used to identify instruction execution coverage of some init
// or runtime bytecode.
type CoverageMapBytecodeData struct {
	executedFlags []byte
}

// isCovered checks if a given program counter location is covered by the map.
// Returns a boolean indicating if the program counter was executed on this map.
func (cm *CoverageMapBytecodeData) isCovered(pc int) bool {
	// If this map has no execution data or is out of bounds, it is not covered.
	if cm.executedFlags == nil || len(cm.executedFlags) <= pc {
		return false
	}

	// Otherwise, return the execution flag
	return cm.executedFlags[pc] != 0
}

// update creates updates the current CoverageMapBytecodeData with the provided one.
// Returns a boolean indicating whether new coverage was achieved, or an error if one was encountered.
func (cm *CoverageMapBytecodeData) update(coverageMap *CoverageMapBytecodeData) (bool, error) {
	// If the coverage map execution data provided is nil, exit early
	if coverageMap.executedFlags == nil {
		return false, nil
	}

	// If the current map has no execution data, simply set it to the provided one.
	if cm.executedFlags == nil {
		cm.executedFlags = coverageMap.executedFlags
		return true, nil
	}

	// Update each byte which represents a position in the bytecode which was covered. We ignore any size
	// differences as init bytecode can have arbitrary length arguments appended.
	changed := false
	for i := 0; i < len(cm.executedFlags) || i < len(coverageMap.executedFlags); i++ {
		if cm.executedFlags[i] == 0 && coverageMap.executedFlags[i] != 0 {
			cm.executedFlags[i] = 1
			changed = true
		}
	}
	return changed, nil
}

// setCoveredAt sets the coverage state at a given program counter location within a CoverageMapBytecodeData.
// Returns a boolean indicating whether new coverage was achieved, or an error if one occurred.
func (cm *CoverageMapBytecodeData) setCoveredAt(codeSize int, pc uint64) (bool, error) {
	// If the execution flags don't exist, create them for this code size.
	if cm.executedFlags == nil {
		cm.executedFlags = make([]byte, codeSize)
	}

	// If our program counter is in range, determine if we achieved new coverage for the first time, and update it.
	if pc < uint64(len(cm.executedFlags)) {
		if cm.executedFlags[pc] == 0 {
			cm.executedFlags[pc] = 1
			return true, nil
		}
		return false, nil
	}
	return false, fmt.Errorf("tried to set coverage map out of bounds (pc: %d, code size %d)", pc, len(cm.executedFlags))
}

// Equal checks whether the provided CoverageMapBytecodeData contains the same data as the current one.
// Returns a boolean indicating whether the two maps match.
func (cm *CoverageMapBytecodeData) Equal(b *CoverageMapBytecodeData) bool {
	// Return an equality comparison on the data, ignoring size checks by stopping at the end of the shortest slice.
	// We do this to avoid comparing arbitrary length constructor arguments appended to init bytecode.
	smallestSize := utils.Min(len(cm.executedFlags), len(b.executedFlags))
	return bytes.Equal(cm.executedFlags[:smallestSize], b.executedFlags[:smallestSize])
}
