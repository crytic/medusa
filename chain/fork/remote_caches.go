package fork

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"sync"
)

type remoteStateObject struct {
	Balance *uint256.Int
	Nonce   uint64
	Code    []byte
}

type stateObjectCacheThreadSafe struct {
	lock  sync.RWMutex
	cache map[common.Address]*remoteStateObject
}

func newStateObjectCache() *stateObjectCacheThreadSafe {
	return &stateObjectCacheThreadSafe{
		lock:  sync.RWMutex{},
		cache: make(map[common.Address]*remoteStateObject),
	}
}

func (s *stateObjectCacheThreadSafe) GetStateObject(addr common.Address) (*remoteStateObject, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if obj, ok := s.cache[addr]; !ok {
		return nil, fmt.Errorf("cache miss")
	} else {
		return obj, nil
	}
}

func (s *stateObjectCacheThreadSafe) WriteStateObject(addr common.Address, data remoteStateObject) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.cache[addr] = &data
}

type slotCacheThreadSafe struct {
	lock  sync.RWMutex
	cache map[common.Address]map[common.Hash]common.Hash
}

func newSlotCache() *slotCacheThreadSafe {
	return &slotCacheThreadSafe{
		lock:  sync.RWMutex{},
		cache: make(map[common.Address]map[common.Hash]common.Hash),
	}
}

func (s *slotCacheThreadSafe) GetSlotData(addr common.Address, slot common.Hash) (common.Hash, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if slotLookup, ok := s.cache[addr]; ok {
		if data, ok := slotLookup[slot]; ok {
			return data, nil
		}
	}
	return common.Hash{}, fmt.Errorf("cache miss")
}

func (s *slotCacheThreadSafe) WriteSlotData(addr common.Address, slot common.Hash, data common.Hash) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.cache[addr]; !ok {
		s.cache[addr] = make(map[common.Hash]common.Hash)
	}

	s.cache[addr][slot] = data
}
