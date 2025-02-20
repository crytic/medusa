package cache

import (
	"sync"

	"github.com/crytic/medusa-geth/common"
)

// nonPersistentStateCache provides a thread-safe cache for storing state objects and slots without persisting to disk.
type nonPersistentStateCache struct {
	stateObjectLock  sync.RWMutex
	stateObjectCache map[common.Address]*StateObject

	slotLock  sync.RWMutex
	slotCache map[common.Address]map[common.Hash]common.Hash
}

func newNonPersistentStateCache() *nonPersistentStateCache {
	return &nonPersistentStateCache{
		stateObjectLock:  sync.RWMutex{},
		slotLock:         sync.RWMutex{},
		stateObjectCache: make(map[common.Address]*StateObject),
		slotCache:        make(map[common.Address]map[common.Hash]common.Hash),
	}
}

// GetStateObject checks if the addr is present in the cache, and if not, returns an error
func (s *nonPersistentStateCache) GetStateObject(addr common.Address) (*StateObject, error) {
	s.stateObjectLock.RLock()
	defer s.stateObjectLock.RUnlock()

	if obj, ok := s.stateObjectCache[addr]; !ok {
		return nil, ErrCacheMiss
	} else {
		return obj, nil
	}
}

func (s *nonPersistentStateCache) WriteStateObject(addr common.Address, data StateObject) error {
	s.stateObjectLock.Lock()
	defer s.stateObjectLock.Unlock()
	s.stateObjectCache[addr] = &data
	return nil
}

// GetSlotData checks if the specified data is stored in the cache, and if not, returns an error.
func (s *nonPersistentStateCache) GetSlotData(addr common.Address, slot common.Hash) (common.Hash, error) {
	s.slotLock.RLock()
	defer s.slotLock.RUnlock()
	if slotLookup, ok := s.slotCache[addr]; ok {
		if data, ok := slotLookup[slot]; ok {
			return data, nil
		}
	}
	return common.Hash{}, ErrCacheMiss
}

func (s *nonPersistentStateCache) WriteSlotData(addr common.Address, slot common.Hash, data common.Hash) error {
	s.slotLock.Lock()
	defer s.slotLock.Unlock()

	if _, ok := s.slotCache[addr]; !ok {
		s.slotCache[addr] = make(map[common.Hash]common.Hash)
	}

	s.slotCache[addr][slot] = data
	return nil
}
