package coverage

import (
	"os"
	"sync"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/crypto"
	compilationTypes "github.com/crytic/medusa/compilation/types"
)

// CoverageMaps represents a data structure used to identify branch coverage of various smart contracts
// across a transaction or multiple transactions. Branch coverage includes jumps, returns, reverts, and contract entrance.
type CoverageMaps struct {
	// maps represents a structure used to track every ContractCoverageMap by a given deployed address/lookup hash.
	maps map[common.Hash]map[common.Address]*ContractCoverageMap

	// cachedCodeAddress represents the last code address which coverage was updated for. This is used to prevent an
	// expensive lookup in maps. If cachedCodeAddress does not match the current code address for which we are updating
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

// A bit of a hack, but a pretty harmless one.
// Some builds produce multiple contracts with identical cbor metadatas.
// This results in getContractCoverageMapHash returning the same result for different contracts,
// which cause problems when generating coverage reports.
// Those people should set USE_FULL_BYTECODE=1, which causes getContractCoverageMapHash calculation
// to look at the full bytecode, not just the metadata.
// We don't want to use this option universally because it gets screwed up when immutables are involved,
// and is probably slower.
var useFullBytecode func() bool = sync.OnceValue(func() bool {
	return os.Getenv("USE_FULL_BYTECODE") != ""
})

// getContractCoverageMapHash obtain the hash used to look up a given contract's ContractCoverageMap.
// If this is init bytecode, metadata and abi arguments will attempt to be stripped, then a hash is computed.
// If this is runtime bytecode, the metadata ipfs/swarm hash will be used if available, otherwise the bytecode
// is hashed.
// Returns the resulting lookup hash.
func getContractCoverageMapHash(bytecode []byte, init bool) common.Hash {
	// If available, the metadata code hash should be unique and reliable to use above all (for runtime bytecode).
	if !init && !useFullBytecode() {
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

// Update updates the current coverage maps with the provided ones.
// Returns a boolean indicating whether coverage changed, or an error if one occurred.
func (cm *CoverageMaps) Update(coverageMaps *CoverageMaps) (bool, error) {
	// If our maps provided are nil, do nothing
	if coverageMaps == nil {
		return false, nil
	}

	// Acquire our thread lock and defer our unlocking for when we exit this method
	cm.updateLock.Lock()
	defer cm.updateLock.Unlock()

	// Create a boolean indicating whether we achieved new coverage
	coverageChanged := false

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
				changed, err := existingCoverageMap.update(coverageMapToMerge)
				coverageChanged = coverageChanged || changed
				if err != nil {
					return coverageChanged, err
				}
			} else {
				mapsByAddress[codeAddress] = coverageMapToMerge
				coverageChanged = coverageMapToMerge.executedMarkers.entries != nil
			}
		}
	}

	// Return our results
	return coverageChanged, nil
}

// UpdateAt updates the hit count of a given marker within code coverage data.
func (cm *CoverageMaps) UpdateAt(codeAddress common.Address, codeLookupHash common.Hash, marker uint64) (bool, error) {
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
	changedInMap, err = coverageMap.updateCoveredAt(marker)

	return addedNewMap || changedInMap, err
}

// BranchesHit is a function that returns the total number of unique markers.
// Technically these markers can also refer to contract entrance, return, and revert,
// but calling it "branches" is close enough.
func (cm *CoverageMaps) BranchesHit() uint64 {
	// Acquire our thread lock and defer our unlocking for when we exit this method
	cm.updateLock.Lock()
	defer cm.updateLock.Unlock()

	branchesHit := uint64(0)
	// Iterate across each contract deployment
	for _, mapsByAddress := range cm.maps {
		// Consider the coverage of all of the different deployments of this codehash as a set
		// And mark a marker as hit if any of the instances has a hit for it
		uniqueMarkersForHash := make(map[uint64]struct{})

		for _, contractCoverageMap := range mapsByAddress {
			for branch := range contractCoverageMap.executedMarkers.all() {
				uniqueMarkersForHash[branch] = struct{}{}
			}
		}
		branchesHit += uint64(len(uniqueMarkersForHash))
	}
	return branchesHit
}

// ContractCoverageMap represents a data structure used to identify execution coverage of a contract.
type ContractCoverageMap struct {
	// executedMarkers is a map of marker to number of times this marker has been hit.
	// A marker can represent one of four things: jump instruction, revert, return, or entering a contract.
	// Markers are 64-bit integers, storing two values: "source" in the upper 32 bits, and "destination" in lower 32 bits.
	// For markers corresponding to jump instructions, source and dest mean the source and dest of the jump.
	// For revert and return, the "source" is the PC of the opcode causing the revert/return, and the "destination" is REVERT_MARKER_XOR or RETURN_MARKER_XOR.
	// For contract entrance, the "source" is ENTER_MARKER_XOR and the "destination" is the first PC executed.
	executedMarkers markerTable
}

// Constants used in markers. See comments on ContractCoverageMap.executedMarkers for more details on how these are used.
// These are high (> 1 billion) so that it should never overlap with real PCs, ie so that these markers should never overlap with jump markers.
const (
	REVERT_MARKER_XOR = 0x40000000
	RETURN_MARKER_XOR = 0x80000000
	ENTER_MARKER_XOR  = 0xC0000000
)

// newContractCoverageMap creates and returns a new ContractCoverageMap.
func newContractCoverageMap() *ContractCoverageMap {
	return &ContractCoverageMap{}
}

// Equal checks whether the provided ContractCoverageMap contains the same data as the current one.
// Returns a boolean indicating whether the two maps match.
func (cm *ContractCoverageMap) Equal(b *ContractCoverageMap) bool {
	for marker, count := range cm.executedMarkers.all() {
		if b.executedMarkers.get(marker) != count {
			return false
		}
	}
	for marker, count := range b.executedMarkers.all() {
		if cm.executedMarkers.get(marker) != count {
			return false
		}
	}
	return true
}

// update merges exact hit counts, including uint64 wraparound as before.
func (cm *ContractCoverageMap) update(other *ContractCoverageMap) (bool, error) {
	if other.executedMarkers.entries == nil {
		return false, nil
	}
	cm.executedMarkers.init()
	changed := false
	for marker, count := range other.executedMarkers.all() {
		if count != 0 {
			added := cm.executedMarkers.add(marker, count)
			changed = changed || added
		}
	}
	return changed, nil
}

// updateCoveredAt updates one marker with a single hash-table probe.
func (cm *ContractCoverageMap) updateCoveredAt(marker uint64) (bool, error) {
	return cm.executedMarkers.add(marker, 1), nil
}

// HitCount returns the number of times a marker was hit, or zero if absent.
func (cm *ContractCoverageMap) HitCount(marker uint64) uint64 {
	if cm == nil {
		return 0
	}
	return cm.executedMarkers.get(marker)
}
