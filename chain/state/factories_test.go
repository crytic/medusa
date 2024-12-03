package state

import (
	types2 "github.com/crytic/medusa/chain/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	gethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"testing"
)

/* TestForkedStateDB provides unit testing for medusa-geth's ForkedStateDb */
func TestForkedStateDB(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	factory := NewForkedStateFactory(fixture.Backend)

	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	cachingDb := gethstate.NewDatabaseWithNodeDB(db, tdb)

	stateDb1, err := factory.New(types.EmptyRootHash, cachingDb)
	assert.NoError(t, err)
	genesisSnap := stateDb1.Snapshot()

	/* ensure the statedb is hitting the backend */
	assert.True(t, stateDb1.Exist(fixture.StateObjectContractAddress))
	assert.True(t, stateDb1.Exist(fixture.StateObjectEOAAddress))
	assert.False(t, stateDb1.Exist(fixture.StateObjectEmptyAddress))

	checkAccountAgainstFixture := func(stateDb types2.MedusaStateDB, addr common.Address, fixture remoteStateObject) {
		bal := stateDb.GetBalance(addr)
		assert.NoError(t, stateDb.Error())
		assert.EqualValues(t, bal, fixture.Balance)
		nonce := stateDb.GetNonce(addr)
		assert.NoError(t, stateDb.Error())
		assert.EqualValues(t, nonce, fixture.Nonce)
		code := stateDb.GetCode(addr)
		assert.NoError(t, stateDb.Error())
		assert.EqualValues(t, code, fixture.Code)
	}

	checkAccountAgainstFixture(stateDb1, fixture.StateObjectContractAddress, fixture.StateObjectContract)
	checkAccountAgainstFixture(stateDb1, fixture.StateObjectEOAAddress, fixture.StateObjectEOA)
	checkAccountAgainstFixture(stateDb1, fixture.StateObjectEmptyAddress, fixture.StateObjectEmpty)

	/* write some new data and make sure it's readable */
	newAccount := common.BytesToAddress([]byte{1, 2, 3, 4, 5, 6})
	newAccountData := remoteStateObject{
		Balance: uint256.NewInt(5),
		Nonce:   99,
		Code:    []byte{1, 2, 3},
	}

	stateDb1.SetNonce(newAccount, newAccountData.Nonce)
	assert.True(t, stateDb1.Exist(newAccount))
	stateDb1.SetCode(newAccount, newAccountData.Code)
	stateDb1.SetBalance(newAccount, newAccountData.Balance, tracing.BalanceChangeUnspecified)
	checkAccountAgainstFixture(stateDb1, newAccount, newAccountData)

	/* roll back to snapshot, ensure fork data still queryable and newly added data was purged */
	stateDb1.Snapshot()
	stateDb1.RevertToSnapshot(genesisSnap)
	checkAccountAgainstFixture(stateDb1, fixture.StateObjectContractAddress, fixture.StateObjectContract)
	checkAccountAgainstFixture(stateDb1, fixture.StateObjectEOAAddress, fixture.StateObjectEOA)
	checkAccountAgainstFixture(stateDb1, fixture.StateObjectEmptyAddress, fixture.StateObjectEmpty)
	assert.False(t, stateDb1.Exist(newAccount))

	/* now we want to test to verify our fork-populated data is being persisted */
	root, err := stateDb1.Commit(1, true)
	assert.NoError(t, err)
	stateDb2, err := factory.New(root, cachingDb)
	assert.NoError(t, err)

	checkAccountAgainstFixture(stateDb2, fixture.StateObjectContractAddress, fixture.StateObjectContract)
	checkAccountAgainstFixture(stateDb2, fixture.StateObjectEOAAddress, fixture.StateObjectEOA)
	checkAccountAgainstFixture(stateDb2, fixture.StateObjectEmptyAddress, fixture.StateObjectEmpty)
}

/*
TestForkedStateFactory verifies the various independence/shared properties of each forkedStateDB created by the
factory. This is because the underlying RPC/caching layer is shared between all TestChain instances globally,
but this sharing relationship should not cause state to leak from one forkedStateDb to another.
*/
func TestForkedStateFactory(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	factory := NewForkedStateFactory(fixture.Backend)

	db1 := rawdb.NewMemoryDatabase()
	tdb1 := triedb.NewDatabase(db1, nil)
	cachingDb1 := gethstate.NewDatabaseWithNodeDB(db1, tdb1)

	stateDb1, err := factory.New(types.EmptyRootHash, cachingDb1)
	assert.NoError(t, err)
	_ = stateDb1
}
