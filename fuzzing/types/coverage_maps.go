package types

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

// CoverageMaps represents a data structure used to identify instruction execution coverage of various smart contracts
// across a transaction or multiple transactions.
type CoverageMaps struct {
	maps map[common.Hash]*CoverageMap

	cachedCodeHash common.Hash
	cachedMap      *CoverageMap

	// updateLock is a lock to offer concurrent thread safety for map accesses.
	updateLock sync.Mutex
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
	// Acquire our thread lock and defer our unlocking for when we exit this method
	cm.updateLock.Lock()
	defer cm.updateLock.Unlock()

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
			coverageMap, err = NewCoverageMap(codeSize)
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

// MarshalJSON encodes the current CoverageMaps as JSON data.
func (cm *CoverageMaps) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Maps map[common.Hash]*CoverageMap `json:"maps"`
	}{
		Maps: cm.maps,
	})
}

// UnmarshalJSON parses JSON encoded data from the provided bytes into the current CoverageMaps.
func (cm *CoverageMaps) UnmarshalJSON(b []byte) error {
	// Unmarshal our data into our temp struct.
	var tmp struct {
		Maps map[common.Hash]*CoverageMap `json:"maps"`
	}
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}

	// If we succeeded, set our data.
	cm.maps = tmp.Maps
	return nil
}

// CoverageMap represents a data structure used to identify instruction execution coverage of smart contract byte code.
type CoverageMap struct {
	// mapData represents a list of bytes for each byte of a deployed smart contract where zero values indicate
	// execution did not occur at the given position, while any other value indicates code execution occurred at the
	// given smart contract offset.
	mapData []byte
}

// NewCoverageMap initializes a new CoverageMap object.
func NewCoverageMap(size int) (*CoverageMap, error) {
	// If the size is negative, throw an error
	if size < 0 {
		return nil, fmt.Errorf("cannot create a coverage map with a negative byte code size (%d)", size)
	}

	// Create a coverage map of the requested size
	coverageMap := &CoverageMap{
		mapData: make([]byte, size),
	}
	return coverageMap, nil
}

// Update creates updates the current coverage map with the provided one. It returns a boolean indicating whether
// new coverage was achieved, or an error if one was encountered.
func (cm *CoverageMap) Update(coverageMap *CoverageMap) (bool, error) {
	// Ensure our coverage maps match in size
	if len(cm.mapData) != len(coverageMap.mapData) {
		return false, fmt.Errorf("failed to add/merge coverage maps. Map of size %d cannot be merged into map of size %d", len(coverageMap.mapData), len(cm.mapData))
	}

	// Update each byte which represents a position in the bytecode which was covered.
	changed := false
	for i := 0; i < len(cm.mapData); i++ {
		if cm.mapData[i] == 0 && coverageMap.mapData[i] != 0 {
			cm.mapData[i] = 1
			changed = true
		}
	}
	return changed, nil
}

// SetCoveredAt sets the coverage state of a given program counter location within a CoverageMap.
func (cm *CoverageMap) SetCoveredAt(pc uint64) (bool, error) {
	if pc < uint64(len(cm.mapData)) {
		if cm.mapData[pc] == 0 {
			cm.mapData[pc] = 1
			return true, nil
		}
		return false, nil
	}
	return false, fmt.Errorf("tried to set coverage map out of bounds (pc: %d, code size %d)", pc, len(cm.mapData))
}

// Reset clears the coverage state for the CoverageMap.
func (cm *CoverageMap) Reset() {
	cm.mapData = make([]byte, len(cm.mapData))
}

// MarshalJSON encodes the current CoverageMap as JSON data.
func (cm *CoverageMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		MapData []byte `json:"mapData"`
	}{
		MapData: cm.mapData,
	})
}

// UnmarshalJSON parses JSON encoded data from the provided bytes into the current CoverageMap.
func (cm *CoverageMap) UnmarshalJSON(b []byte) error {
	// Unmarshal our data into our temp struct.
	var tmp struct {
		MapData []byte `json:"mapData"`
	}
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}

	// If we succeeded, set our data.
	cm.mapData = tmp.MapData
	return nil
}
