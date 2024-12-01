package fork

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"sync"
)

type RemoteStateQuery interface {
	GetStorageAt(common.Address, common.Hash) (common.Hash, error)
	GetStateObject(common.Address) (*uint256.Int, uint64, []byte, error)
}

type remoteStateObject struct {
	Balance *uint256.Int
	Nonce   uint64
	Code    []byte
}

type slotCacheThreadSafe struct {
	lock  sync.RWMutex
	cache map[common.Address]map[common.Hash]common.Hash
}

func newSlotCacheThreadSafe() *slotCacheThreadSafe {
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

	// defensive code
	if _, ok := s.cache[addr][slot]; ok {
		panic("This slot was populated by another RPC request.")
	}
	s.cache[addr][slot] = data
}

type LiveRemoteStateQuery struct {
	height uint64

	slotCache        map[common.Address]map[common.Hash]common.Hash
	stateObjectCache map[common.Address]*remoteStateObject
}
