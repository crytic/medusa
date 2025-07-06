package cache

import (
	"context"
	"math/rand"
	"os"
	"sync"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/stretchr/testify/assert"
)

// TestNonPersistentStateObjectCacheRace tests for race conditions
func TestNonPersistentStateObjectCacheRace(t *testing.T) {
	cache := newNonPersistentStateCache()
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
			stateObject := StateObject{
				Nonce: r.Uint64(),
			}
			err := cache.WriteStateObject(addr, stateObject)
			assert.NoError(t, err)
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

// TestNonPersistentSlotCacheRace tests for race conditions
func TestNonPersistentSlotCacheRace(t *testing.T) {
	cache := newNonPersistentStateCache()
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

			err := cache.WriteSlotData(addr, objHash, dataHash)
			assert.NoError(t, err)
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

// TestPersistentCache tests read/write capability of the persistent cache, along with persistence itself.
func TestPersistentCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	rpcAddr := "www.rpc.net/ethereum/etc"
	blockHeight := uint64(55555)
	tmpDir, err := os.MkdirTemp("", "test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pc, err := newPersistentCache(ctx, tmpDir, rpcAddr, blockHeight)
	assert.NoError(t, err)

	stateObjectAddr := common.Address{0x55}
	stateObjectData := &StateObject{
		Nonce: rand.Uint64(),
	}
	// try reading from a state cache that doesn't exist
	_, err = pc.GetStateObject(stateObjectAddr)
	assert.Error(t, err)
	assert.Equal(t, err, ErrCacheMiss)

	// write the state cache, then make sure we can read it
	err = pc.WriteStateObject(stateObjectAddr, *stateObjectData)
	assert.NoError(t, err)

	so, err := pc.GetStateObject(stateObjectAddr)
	assert.NoError(t, err)
	assert.Equal(t, *stateObjectData, *so)

	// repeat the above for slots
	stateSlotAddress := common.Hash{0x66, 0x01}
	stateSlotData := common.Hash{0x81}

	// try reading from a slot that doesn't exist
	_, err = pc.GetSlotData(stateObjectAddr, stateSlotAddress)
	assert.Error(t, err)
	assert.Equal(t, err, ErrCacheMiss)

	// write the slot, then make sure we can read it
	err = pc.WriteSlotData(stateObjectAddr, stateSlotAddress, stateSlotData)
	assert.NoError(t, err)

	data, err := pc.GetSlotData(stateObjectAddr, stateSlotAddress)
	assert.NoError(t, err)
	assert.Equal(t, stateSlotData, data)

	// now terminate our cache to test persistence
	cancel()

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	pc, err = newPersistentCache(ctx, tmpDir, rpcAddr, blockHeight)
	assert.NoError(t, err)

	// state cache matches
	so, err = pc.GetStateObject(stateObjectAddr)
	assert.NoError(t, err)
	assert.Equal(t, *stateObjectData, *so)

	// slot matches
	data, err = pc.GetSlotData(stateObjectAddr, stateSlotAddress)
	assert.NoError(t, err)
	assert.Equal(t, stateSlotData, data)
}
