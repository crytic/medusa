package state

import (
	"github.com/crytic/medusa/chain/state/cache"
	"testing"

	"github.com/crytic/medusa-geth/common"
	gethstate "github.com/crytic/medusa-geth/core/state"
	"github.com/crytic/medusa-geth/core/tracing"
	"github.com/crytic/medusa-geth/core/types"
	types2 "github.com/crytic/medusa/chain/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

// TestForkedStateDB provides unit testing for medusa-geth's ForkedStateDb
func TestForkedStateDB(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	factory := NewForkedStateFactory(fixture.Backend)

	cachingDb := gethstate.NewDatabaseForTesting()

	stateDb1, err := factory.New(types.EmptyRootHash, cachingDb)
	assert.NoError(t, err)
	genesisSnap := stateDb1.Snapshot()

	// ensure the statedb is hitting the backend
	assert.True(t, stateDb1.Exist(fixture.StateObjectContractAddress))
	assert.True(t, stateDb1.Exist(fixture.StateObjectEOAAddress))
	assert.False(t, stateDb1.Exist(fixture.StateObjectEmptyAddress))

	fixture.verifyAgainstState(t, stateDb1)

	// write some new data and make sure it's readable
	newAccount := common.BytesToAddress([]byte{1, 2, 3, 4, 5, 6})
	newAccountData := cache.StateObject{
		Balance: uint256.NewInt(5),
		Nonce:   99,
		Code:    []byte{1, 2, 3},
	}

	stateDb1.SetNonce(newAccount, newAccountData.Nonce, tracing.NonceChangeUnspecified)
	assert.True(t, stateDb1.Exist(newAccount))
	stateDb1.SetCode(newAccount, newAccountData.Code)
	stateDb1.SetBalance(newAccount, newAccountData.Balance, tracing.BalanceChangeUnspecified)
	checkAccountAgainstFixture(t, stateDb1, newAccount, newAccountData)

	// roll back to snapshot, ensure fork data still queryable and newly added data was purged
	stateDb1.Snapshot()
	stateDb1.RevertToSnapshot(genesisSnap)
	fixture.verifyAgainstState(t, stateDb1)
	assert.False(t, stateDb1.Exist(newAccount))

	// now we want to test to verify our fork-populated data is being persisted
	root, err := stateDb1.Commit(1, true, true)
	assert.NoError(t, err)
	stateDb2, err := factory.New(root, cachingDb)
	assert.NoError(t, err)

	fixture.verifyAgainstState(t, stateDb2)
}

// TestForkedStateFactory verifies the various independence/shared properties of each forkedStateDB created by the
// factory. This is because the underlying RPC/caching layer is shared between all TestChain instances globally,
// but this sharing relationship should not cause state to leak from one forkedStateDb to another.
func TestForkedStateFactory(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	factory := NewForkedStateFactory(fixture.Backend)

	stateDb1, err := createEmptyStateDb(factory)
	assert.NoError(t, err)
	stateDb1.Snapshot()

	stateDb2, err := createEmptyStateDb(factory)
	assert.NoError(t, err)
	stateDb2.Snapshot()

	// naive check to ensure they're both pulling from the same remote
	fixture.verifyAgainstState(t, stateDb1)
	fixture.verifyAgainstState(t, stateDb2)

	// snapshot and roll em back
	stateDb1.Snapshot()
	stateDb2.Snapshot()
	stateDb1.RevertToSnapshot(0)
	stateDb2.RevertToSnapshot(0)

	// now we'll mutate a cold account in one stateDB and ensure the mutation doesn't propagate
	valueAdded := uint256.NewInt(100)
	expectedSum := uint256.NewInt(0).Add(fixture.StateObjectEOA.Balance, valueAdded)
	stateDb1.AddBalance(fixture.StateObjectEOAAddress, valueAdded, tracing.BalanceChangeUnspecified)
	bal := stateDb1.GetBalance(fixture.StateObjectEOAAddress)
	assert.Equal(t, expectedSum, bal)

	// check the other statedb
	bal = stateDb2.GetBalance(fixture.StateObjectEOAAddress)
	assert.Equal(t, bal, fixture.StateObjectEOA.Balance)

	// just in case there's some weird pointer issue that was introduced, create a new stateDB and check it as well
	stateDb3, err := createEmptyStateDb(factory)
	assert.NoError(t, err)
	bal = stateDb3.GetBalance(fixture.StateObjectEOAAddress)
	assert.Equal(t, bal, fixture.StateObjectEOA.Balance)

	// now we'll emulate one stateDB obtaining a new piece of data from RPC and ensuring the other stateDB loads
	// the same data
	newAccount := common.BytesToAddress([]byte{1, 2, 3, 4, 5, 6})
	slotKey := common.BytesToHash([]byte{5, 5, 5, 5, 5, 5, 5})
	slotData := common.BytesToHash([]byte{6, 6, 6, 6, 6, 6, 6})

	fixture.Backend.SetStorageAt(newAccount, slotKey, slotData)
	data := stateDb1.GetState(newAccount, slotKey)
	assert.EqualValues(t, slotData, data)

	// do it again with a fresh stateDB
	stateDb4, err := createEmptyStateDb(factory)
	assert.NoError(t, err)
	data = stateDb4.GetState(newAccount, slotKey)
	assert.EqualValues(t, slotData, data)
}

// TestEmptyBackendFactoryDifferential tests the differential properties between a stateDB using an empty forked backend
// versus directly using geth's statedb.
func TestEmptyBackendFactoryDifferential(t *testing.T) {
	gethFactory := &gethStateFactory{}
	unbackedFactory := NewUnbackedStateFactory()

	gethStateDb, err := createEmptyStateDb(gethFactory)
	assert.NoError(t, err)

	unbackedStateDb, err := createEmptyStateDb(unbackedFactory)
	assert.NoError(t, err)

	// start with existence/empty of an existing object. should be identical.
	addr := common.BytesToAddress([]byte{1})
	gethStateDb.SetNonce(addr, 5, tracing.NonceChangeUnspecified)
	unbackedStateDb.SetNonce(addr, 5, tracing.NonceChangeUnspecified)
	assert.EqualValues(t, gethStateDb.Exist(addr), unbackedStateDb.Exist(addr))
	assert.EqualValues(t, gethStateDb.Empty(addr), unbackedStateDb.Empty(addr))

	// existence/empty of a non-existing object, should be identical.
	nonExistentStateObjAddr := common.BytesToAddress([]byte{5, 5, 5, 5, 5})
	assert.EqualValues(t, gethStateDb.Exist(nonExistentStateObjAddr), unbackedStateDb.Exist(nonExistentStateObjAddr))
	assert.EqualValues(t, gethStateDb.Empty(nonExistentStateObjAddr), unbackedStateDb.Empty(nonExistentStateObjAddr))

	emptyStateObjectAddr := common.BytesToAddress([]byte{6, 7, 8, 9, 10})
	value := uint256.NewInt(5000)
	gethStateDb.SetBalance(emptyStateObjectAddr, value, tracing.BalanceChangeUnspecified)
	unbackedStateDb.SetBalance(emptyStateObjectAddr, value, tracing.BalanceChangeUnspecified)

	// existence/empty of an empty object, should be identical.
	gethStateDb.SubBalance(emptyStateObjectAddr, value, tracing.BalanceChangeUnspecified)
	unbackedStateDb.SubBalance(emptyStateObjectAddr, value, tracing.BalanceChangeUnspecified)
	assert.EqualValues(t, gethStateDb.Exist(emptyStateObjectAddr), unbackedStateDb.Exist(emptyStateObjectAddr))
	assert.EqualValues(t, gethStateDb.Empty(emptyStateObjectAddr), unbackedStateDb.Empty(emptyStateObjectAddr))
}

// TestForkedBackendDifferential tests the differential properties between a stateDB using a forked backend
// versus directly using geth's statedb. Consider this test a canonical definition of how our forked stateDB acts
// differently from geth's.
// Good place for future fuzz testing if we run into issues.
func TestForkedBackendDifferential(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	factory := NewForkedStateFactory(fixture.Backend)
	forkedStateDb, err := createEmptyStateDb(factory)
	assert.NoError(t, err)

	gethFactory := &gethStateFactory{}
	gethStateDb, err := createEmptyStateDb(gethFactory)
	assert.NoError(t, err)

	// modify the geth statedb to reflect the fixture's different accounts
	// contract
	gethStateDb.SetBalance(
		fixture.StateObjectContractAddress,
		fixture.StateObjectContract.Balance,
		tracing.BalanceChangeUnspecified)
	gethStateDb.SetNonce(fixture.StateObjectContractAddress, fixture.StateObjectContract.Nonce, tracing.NonceChangeUnspecified)
	gethStateDb.SetCode(fixture.StateObjectContractAddress, fixture.StateObjectContract.Code)
	// eoa
	gethStateDb.SetBalance(
		fixture.StateObjectEOAAddress,
		fixture.StateObjectEOA.Balance,
		tracing.BalanceChangeUnspecified)
	gethStateDb.SetNonce(fixture.StateObjectEOAAddress, fixture.StateObjectEOA.Nonce, tracing.NonceChangeUnspecified)
	// do not set the empty account. On a live geth node, the empty account will be pruned.

	// check exist/empty equivalence for the contract account
	assert.EqualValues(
		t,
		gethStateDb.Exist(fixture.StateObjectContractAddress),
		forkedStateDb.Exist(fixture.StateObjectContractAddress))
	assert.EqualValues(
		t,
		gethStateDb.Empty(fixture.StateObjectContractAddress),
		forkedStateDb.Empty(fixture.StateObjectContractAddress))

	// check exist/empty equivalence for the eoa account
	assert.EqualValues(
		t,
		gethStateDb.Exist(fixture.StateObjectEOAAddress),
		forkedStateDb.Exist(fixture.StateObjectEOAAddress))
	assert.EqualValues(
		t,
		gethStateDb.Empty(fixture.StateObjectEOAAddress),
		forkedStateDb.Empty(fixture.StateObjectEOAAddress))

	// check exist/empty equivalence for the empty account
	assert.EqualValues(
		t,
		gethStateDb.Empty(fixture.StateObjectEmptyAddress),
		forkedStateDb.Empty(fixture.StateObjectEmptyAddress))
	// note how this is _not_ EqualValues. As far as we know, this is the only place where the forked state provider
	// diverges from geth's behavior.
	assert.NotEqualValues(
		t,
		gethStateDb.Exist(fixture.StateObjectEmptyAddress),
		forkedStateDb.Exist(fixture.StateObjectEmptyAddress))
}

// createEmptyStateDb creates an empty stateDB using the provided factory. Intended for tests only.
func createEmptyStateDb(factory MedusaStateFactory) (types2.MedusaStateDB, error) {
	cachingDb := gethstate.NewDatabaseForTesting()
	return factory.New(types.EmptyRootHash, cachingDb)
}

// GethStateFactory is used to build vanilla StateDBs that perfectly reproduce geth's statedb behavior. Only intended
// to be used for differential testing against the unbacked state factory.
type gethStateFactory struct{}

func (f *gethStateFactory) New(root common.Hash, db gethstate.Database) (types2.MedusaStateDB, error) {
	return gethstate.New(root, db)
}
