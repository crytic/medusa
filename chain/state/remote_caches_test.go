package state

import (
	"testing"
)

func TestRemoteStateObjectCache(t *testing.T) {
	/*
		cache := newStateObjectCache()
		r := rand.New(rand.NewSource(1337))

		numObjects := 5
		writers := 10
		numWrites := 10_000
		readers := 10
		numReads := 10_000

		var wg sync.WaitGroup
		wg.Add(writers+readers)

		write := func(r *rand.Rand, writesRem int) {
			objId := r.Uint32() % uint32(numObjects)
			addr := common.BytesToAddress([]byte{byte(objId)})
			stateObject := remoteStateObject{
				Nonce: r.Uint64(),
			}
			cache.WriteStateObject(addr, stateObject)
			writesRem--
			if writesRem == 0 {
				wg.Add(-1)
			} else {
				write(r, writesRem)
			}
		}


		g := go write(r, 5)
		g
	*/
}
