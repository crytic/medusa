package object

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"sync"
)

// StateObject gives us a way to store state objects without the overhead of using geth's stateObject
type StateObject struct {
	Balance *uint256.Int
	Nonce   uint64
	Code    []byte
}

// StateObjectCacheThreadSafe provides a thread-safe cache for storing state objects.
type StateObjectCacheThreadSafe struct {
	lock  sync.RWMutex
	cache map[common.Address]*StateObject
}

func NewStateObjectCache() *StateObjectCacheThreadSafe {
	return &StateObjectCacheThreadSafe{
		lock:  sync.RWMutex{},
		cache: make(map[common.Address]*StateObject),
	}
}

// GetStateObject checks if the addr is present in the cache, and if not, returns an error
func (s *StateObjectCacheThreadSafe) GetStateObject(addr common.Address) (*StateObject, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if obj, ok := s.cache[addr]; !ok {
		return nil, fmt.Errorf("cache miss")
	} else {
		return obj, nil
	}
}

func (s *StateObjectCacheThreadSafe) WriteStateObject(addr common.Address, data StateObject) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.cache[addr] = &data
}

// SlotCacheThreadSafe provides a thread-safe cache for storing data in an account's storage
type SlotCacheThreadSafe struct {
	lock  sync.RWMutex
	cache map[common.Address]map[common.Hash]common.Hash
}

func NewSlotCache() *SlotCacheThreadSafe {
	return &SlotCacheThreadSafe{
		lock:  sync.RWMutex{},
		cache: make(map[common.Address]map[common.Hash]common.Hash),
	}
}

// GetSlotData checks if the specified data is stored in the cache, and if not, returns an error.
func (s *SlotCacheThreadSafe) GetSlotData(addr common.Address, slot common.Hash) (common.Hash, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if slotLookup, ok := s.cache[addr]; ok {
		if data, ok := slotLookup[slot]; ok {
			return data, nil
		}
	}
	return common.Hash{}, fmt.Errorf("cache miss")
}

func (s *SlotCacheThreadSafe) WriteSlotData(addr common.Address, slot common.Hash, data common.Hash) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.cache[addr]; !ok {
		s.cache[addr] = make(map[common.Hash]common.Hash)
	}

	s.cache[addr][slot] = data
}
