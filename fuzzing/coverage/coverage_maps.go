package coverage

import (
	"bytes"
	"fmt"
	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"sync"
)

// CoverageMaps represents a data structure used to identify instruction execution coverage of various smart contracts
// across a transaction or multiple transactions.
type CoverageMaps struct {
	// maps represents a structure used to track every ContractCoverageMap by a given deployed address/lookup hash.
	maps map[common.Hash]map[common.Address]*ContractCoverageMap

	// cachedCodeAddress represents the last code address which coverage was updated for. This is used to prevent an
	// expensive lookup in maps. If cachedCodeHash does not match the current code address for which we are updating
	// coverage for, it, along with other cache variables are updated.
	cachedCodeAddress common.Address

	// cachedCodeHash represents the last lookup hash which coverage was updated for. This is used to prevent an
	// expensive lookup in maps. If cachedCodeHash does not match the current code hash which we are updating
	// coverage for, it, along with other cache variables are updated.
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
	cm.cachedCodeAddress = common.Address{}
	cm.cachedCodeHash = common.Hash{}
	cm.cachedMap = nil
}

// Equal checks whether two coverage maps are the same. Equality is determined if the keys and values are all the same.
func (cm *CoverageMaps) Equal(b *CoverageMaps) bool {
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

// getContractCoverageMapHash obtain the hash used to look up a given contract's ContractCoverageMap.
// If this is init bytecode, metadata and abi arguments will attempt to be stripped, then a hash is computed.
// If this is runtime bytecode, the metadata ipfs/swarm hash will be used if available, otherwise the bytecode
// is hashed.
// Returns the resulting lookup hash.
func getContractCoverageMapHash(bytecode []byte, init bool) common.Hash {
	// If available, the metadata code hash should be unique and reliable to use above all (for runtime bytecode).
	if !init {
		metadata := compilationTypes.ExtractContractMetadata(bytecode)
		if metadata != nil {
			metadataHash := metadata.ExtractBytecodeHash()
			if metadataHash != nil {
				return common.BytesToHash(metadataHash)
			}
		}
	}

	// Otherwise, we use the hash of the bytecode after attempting to strip metadata (and constructor args).
	strippedBytecode := compilationTypes.RemoveContractMetadata(bytecode)
	return crypto.Keccak256Hash(strippedBytecode)
}

// GetContractCoverageMap obtains a total coverage map representing coverage for the provided bytecode.
// If the provided bytecode could not find coverage maps, nil is returned.
// Returns the total coverage map, or an error if one occurs.
func (cm *CoverageMaps) GetContractCoverageMap(bytecode []byte, init bool) (*ContractCoverageMap, error) {
	// Obtain the lookup hash
	hash := getContractCoverageMapHash(bytecode, init)

	// Acquire our thread lock and defer our unlocking for when we exit this method
	cm.updateLock.Lock()
	defer cm.updateLock.Unlock()

	// Loop through all coverage maps for this hash and collect our total coverage.
	if coverageByAddresses, ok := cm.maps[hash]; ok {
		totalCoverage := newContractCoverageMap()
		for _, coverage := range coverageByAddresses {
			_, _, err := totalCoverage.update(coverage)
			if err != nil {
				return nil, err
			}
		}
		return totalCoverage, nil
	} else {
		return nil, nil
	}
}

// Update updates the current coverage maps with the provided ones.
// Returns two booleans indicating whether successful or reverted coverage changed, or an error if one occurred.
func (cm *CoverageMaps) Update(coverageMaps *CoverageMaps) (bool, bool, error) {
	// If our maps provided are nil, do nothing
	if coverageMaps == nil {
		return false, false, nil
	}

	// Acquire our thread lock and defer our unlocking for when we exit this method
	cm.updateLock.Lock()
	defer cm.updateLock.Unlock()

	// Create a boolean indicating whether we achieved new coverage
	successCoverageChanged := false
	revertedCoverageChanged := false

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
				sChanged, rChanged, err := existingCoverageMap.update(coverageMapToMerge)
				successCoverageChanged = successCoverageChanged || sChanged
				revertedCoverageChanged = revertedCoverageChanged || rChanged
				if err != nil {
					return successCoverageChanged, revertedCoverageChanged, err
				}
			} else {
				mapsByAddress[codeAddress] = coverageMapToMerge
				successCoverageChanged = coverageMapToMerge.successfulCoverage != nil
				revertedCoverageChanged = coverageMapToMerge.revertedCoverage != nil
			}
		}
	}

	// Return our results
	return successCoverageChanged, revertedCoverageChanged, nil
}

// SetAt sets the coverage state of a given program counter location within code coverage data.
func (cm *CoverageMaps) SetAt(codeAddress common.Address, codeLookupHash common.Hash, codeSize int, pc uint64) (bool, error) {
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
	if cm.cachedMap != nil && cm.cachedCodeAddress == codeAddress && cm.cachedCodeHash == codeLookupHash {
		coverageMap = cm.cachedMap
	} else {
		// If a coverage map lookup for this code hash doesn't exist, create the mapping.
		mapsByCodeAddress, codeHashExists := cm.maps[codeLookupHash]
		if !codeHashExists {
			mapsByCodeAddress = make(map[common.Address]*ContractCoverageMap)
			cm.maps[codeLookupHash] = mapsByCodeAddress
		}

		// Obtain the coverage map for this code address if it already exists. If it does not, create a new one.
		if existingCoverageMap, codeAddressExists := mapsByCodeAddress[codeAddress]; codeAddressExists {
			coverageMap = existingCoverageMap
		} else {
			coverageMap = newContractCoverageMap()
			cm.maps[codeLookupHash][codeAddress] = coverageMap
			addedNewMap = true
		}

		// Set our cached variables for faster coverage setting next time this method is called.
		cm.cachedMap = coverageMap
		cm.cachedCodeHash = codeLookupHash
		cm.cachedCodeAddress = codeAddress
	}

	// Set our coverage in the map and return our change state
	changedInMap, err = coverageMap.setCoveredAt(codeSize, pc)
	return addedNewMap || changedInMap, err
}

// RevertAll sets all coverage in the coverage map as reverted coverage. Reverted coverage is updated with successful
// coverage, the successful coverage is cleared.
// Returns a boolean indicating whether reverted coverage increased, and an error if one occurred.
func (cm *CoverageMaps) RevertAll() (bool, error) {
	// Acquire our thread lock and defer our unlocking for when we exit this method
	cm.updateLock.Lock()
	defer cm.updateLock.Unlock()

	// Define a variable to track if our reverted coverage changed.
	revertedCoverageChanged := false

	// Loop for each coverage map provided
	for _, mapsByAddressToMerge := range cm.maps {
		for _, contractCoverageMap := range mapsByAddressToMerge {
			// Update our reverted coverage with the (previously thought to be) successful coverage.
			changed, err := contractCoverageMap.revertedCoverage.update(contractCoverageMap.successfulCoverage)
			revertedCoverageChanged = revertedCoverageChanged || changed
			if err != nil {
				return revertedCoverageChanged, err
			}

			// Clear our successful coverage, as these maps were marked as reverted.
			contractCoverageMap.successfulCoverage.Reset()
		}
	}
	return revertedCoverageChanged, nil
}

// ContractCoverageMap represents a data structure used to identify instruction execution coverage of a contract.
type ContractCoverageMap struct {
	// successfulCoverage represents coverage for the contract bytecode, which did not encounter a revert and was
	// deemed successful.
	successfulCoverage *CoverageMapBytecodeData

	// revertedCoverage represents coverage for the contract bytecode, which encountered a revert.
	revertedCoverage *CoverageMapBytecodeData
}

// newContractCoverageMap creates and returns a new ContractCoverageMap.
func newContractCoverageMap() *ContractCoverageMap {
	return &ContractCoverageMap{
		successfulCoverage: &CoverageMapBytecodeData{},
		revertedCoverage:   &CoverageMapBytecodeData{},
	}
}

// Equal checks whether the provided ContractCoverageMap contains the same data as the current one.
// Returns a boolean indicating whether the two maps match.
func (cm *ContractCoverageMap) Equal(b *ContractCoverageMap) bool {
	// Compare both our underlying bytecode coverage maps.
	return cm.successfulCoverage.Equal(b.successfulCoverage) && cm.revertedCoverage.Equal(b.revertedCoverage)
}

// update creates updates the current ContractCoverageMap with the provided one.
// Returns two booleans indicating whether successful or reverted coverage changed, or an error if one was encountered.
func (cm *ContractCoverageMap) update(coverageMap *ContractCoverageMap) (bool, bool, error) {
	// Update our success coverage data
	successfulCoverageChanged, err := cm.successfulCoverage.update(coverageMap.successfulCoverage)
	if err != nil {
		return false, false, err
	}

	// Update our reverted coverage data
	revertedCoverageChanged, err := cm.revertedCoverage.update(coverageMap.revertedCoverage)
	if err != nil {
		return successfulCoverageChanged, false, err
	}

	return successfulCoverageChanged, revertedCoverageChanged, nil
}

// setCoveredAt sets the coverage state at a given program counter location within a ContractCoverageMap used for
// "successful" coverage (non-reverted).
// Returns a boolean indicating whether new coverage was achieved, or an error if one occurred.
func (cm *ContractCoverageMap) setCoveredAt(codeSize int, pc uint64) (bool, error) {
	// Set our coverage data for the successful path.
	return cm.successfulCoverage.setCoveredAt(codeSize, pc)
}

// CoverageMapBytecodeData represents a data structure used to identify instruction execution coverage of some init
// or runtime bytecode.
type CoverageMapBytecodeData struct {
	executedFlags []byte
}

// Reset resets the bytecode coverage map data to be empty.
func (cm *CoverageMapBytecodeData) Reset() {
	cm.executedFlags = nil
}

// Equal checks whether the provided CoverageMapBytecodeData contains the same data as the current one.
// Returns a boolean indicating whether the two maps match.
func (cm *CoverageMapBytecodeData) Equal(b *CoverageMapBytecodeData) bool {
	// Return an equality comparison on the data, ignoring size checks by stopping at the end of the shortest slice.
	// We do this to avoid comparing arbitrary length constructor arguments appended to init bytecode.
	smallestSize := utils.Min(len(cm.executedFlags), len(b.executedFlags))
	return bytes.Equal(cm.executedFlags[:smallestSize], b.executedFlags[:smallestSize])
}

// IsCovered checks if a given program counter location is covered by the map.
// Returns a boolean indicating if the program counter was executed on this map.
func (cm *CoverageMapBytecodeData) IsCovered(pc int) bool {
	// If the coverage map bytecode data is nil, this is not covered.
	if cm == nil {
		return false
	}

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
