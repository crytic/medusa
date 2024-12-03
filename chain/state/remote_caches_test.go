package state

import (
	"github.com/ethereum/go-ethereum/common"
	"math/rand"
	"sync"
	"testing"
)

// TestRemoteStateObjectCache tests for race conditions in stateObjectCacheThreadSafe
func TestRemoteStateObjectCache(t *testing.T) {
	cache := newStateObjectCache()
	numObjects := 5
	writers := 10
	numWrites := 10_000
	readers := 10
	numReads := 10_000

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	write := func(r *rand.Rand, writesRem int) {
		for writesRem > 0 {
			objId := r.Uint32() % uint32(numObjects)
			addr := common.BytesToAddress([]byte{byte(objId)})
			stateObject := remoteStateObject{
				Nonce: r.Uint64(),
			}
			cache.WriteStateObject(addr, stateObject)
			writesRem--
		}
		wg.Add(-1)
	}

	read := func(r *rand.Rand, readsRem int) {
		for readsRem > 0 {
			objId := r.Uint32() % uint32(numObjects)
			addr := common.BytesToAddress([]byte{byte(objId)})
			_, _ = cache.GetStateObject(addr)
			readsRem--
		}
		wg.Add(-1)
	}

	for i := 0; i < readers; i++ {
		go read(rand.New(rand.NewSource(int64(i))), numReads)
	}

	for i := 0; i < writers; i++ {
		go write(rand.New(rand.NewSource(int64(i))), numWrites)
	}
	wg.Wait()
}

// TestRemoteStateObjectCache tests for race conditions in slotCacheThreadSafe
func TestRemoteStateSlotCache(t *testing.T) {
	cache := newSlotCache()
	numContracts := 3
	numObjects := 5
	writers := 10
	numWrites := 10_000
	readers := 10
	numReads := 10_000

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	write := func(r *rand.Rand, writesRem int) {
		for writesRem > 0 {
			addrId := r.Uint32() % uint32(numContracts)
			addr := common.BytesToAddress([]byte{byte(addrId)})

			objId := r.Uint32() % uint32(numObjects)
			objHash := common.BytesToHash([]byte{byte(objId)})

			data := r.Uint32() % 255
			dataHash := common.BytesToHash([]byte{byte(data)})

			cache.WriteSlotData(addr, objHash, dataHash)
			writesRem--
		}
		wg.Add(-1)
	}

	read := func(r *rand.Rand, readsRem int) {
		for readsRem > 0 {
			addrId := r.Uint32() % uint32(numContracts)
			addr := common.BytesToAddress([]byte{byte(addrId)})

			objId := r.Uint32() % uint32(numObjects)
			objHash := common.BytesToHash([]byte{byte(objId)})
			_, _ = cache.GetSlotData(addr, objHash)
			readsRem--
		}
		wg.Add(-1)
	}

	for i := 0; i < readers; i++ {
		go read(rand.New(rand.NewSource(int64(i))), numReads)
	}

	for i := 0; i < writers; i++ {
		go write(rand.New(rand.NewSource(int64(i))), numWrites)
	}
	wg.Wait()
}
