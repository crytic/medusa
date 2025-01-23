package state

import (
	"github.com/crytic/medusa/chain/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

/*
MedusaStateFactory defines a thread-safe interface for creating new state databases. This abstraction allows globally
shared data like RPC caches to be shared across all TestChain instances.
*/
type MedusaStateFactory interface {
	// New initializes a new state
	New(root common.Hash, db state.Database) (types.MedusaStateDB, error)
}

var _ MedusaStateFactory = (*UnbackedStateFactory)(nil)
var _ MedusaStateFactory = (*ForkedStateFactory)(nil)

// ForkedStateFactory is used to build StateDBs that are backed by a remote RPC
type ForkedStateFactory struct {
	globalRemoteStateQuery stateBackend
}

func NewForkedStateFactory(globalCache stateBackend) *ForkedStateFactory {
	return &ForkedStateFactory{globalCache}
}

func (f *ForkedStateFactory) New(root common.Hash, db state.Database) (types.MedusaStateDB, error) {
	remoteStateProvider := newRemoteStateProvider(f.globalRemoteStateQuery)
	return state.NewForkedStateDb(root, db, remoteStateProvider)
}

// UnbackedStateFactory is used to build StateDBs that are not backed by any remote state, but still use the custom
// forked stateDB logic around state object existence checks.
type UnbackedStateFactory struct{}

func NewUnbackedStateFactory() *UnbackedStateFactory {
	return &UnbackedStateFactory{}
}

func (f *UnbackedStateFactory) New(root common.Hash, db state.Database) (types.MedusaStateDB, error) {
	remoteStateProvider := newRemoteStateProvider(EmptyBackend{})
	return state.NewForkedStateDb(root, db, remoteStateProvider)
}
