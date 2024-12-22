package state

import (
	"github.com/crytic/medusa/chain/state/object"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoteStateProvider_ImportStateObject(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	stateProvider := newRemoteStateProvider(fixture.Backend)

	snapId := 5
	importTest := func(objectAddr common.Address, expectedObjectData object.StateObject) {
		/* test a basic state object read */
		bal, nonce, code, err := stateProvider.ImportStateObject(objectAddr, snapId)
		assert.Nil(t, err)
		assert.EqualValues(t, bal, expectedObjectData.Balance)
		assert.EqualValues(t, nonce, expectedObjectData.Nonce)
		assert.EqualValues(t, code, expectedObjectData.Code)

		/* reading the same state object twice should return dirty error */
		_, _, _, err = stateProvider.ImportStateObject(objectAddr, snapId)
		assert.True(t, err.CannotQueryDirtyAccount)
		assert.NotNil(t, err)
		assert.Error(t, err.Error)

		/* reverting to a snapshot equal to that of the imported value should still result in a dirty error */
		stateProvider.NotifyRevertedToSnapshot(snapId)
		_, _, _, err = stateProvider.ImportStateObject(objectAddr, snapId)
		assert.True(t, err.CannotQueryDirtyAccount)
		assert.NotNil(t, err.Error)

		/* reverting to a snapshot before that of the imported value should result in the value being returned */
		stateProvider.NotifyRevertedToSnapshot(snapId - 1)
		bal, nonce, code, err = stateProvider.ImportStateObject(objectAddr, snapId)
		assert.Nil(t, err)
		assert.EqualValues(t, bal, expectedObjectData.Balance)
		assert.EqualValues(t, nonce, expectedObjectData.Nonce)
		assert.EqualValues(t, code, expectedObjectData.Code)
	}

	// run importTest for a contract
	importTest(fixture.StateObjectContractAddress, fixture.StateObjectContract)
	// run importTest for an EOA
	importTest(fixture.StateObjectEOAAddress, fixture.StateObjectEOA)
	// run importTest for an empty/non-existent account
	importTest(fixture.StateObjectEmptyAddress, fixture.StateObjectEmpty)
}

func TestRemoteStateProvider_ImportStorageAt(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	stateProvider := newRemoteStateProvider(fixture.Backend)

	snapId := 5
	importTest := func(contractAddr common.Address, slotKey common.Hash, expectedData common.Hash) {
		/* test a basic state slot read */
		data, err := stateProvider.ImportStorageAt(contractAddr, slotKey, snapId)
		assert.Nil(t, err)
		assert.EqualValues(t, expectedData, data)

		/* reading the same slot twice should result in an error */
		_, err = stateProvider.ImportStorageAt(contractAddr, slotKey, snapId)
		assert.NotNil(t, err)
		assert.True(t, err.CannotQueryDirtySlot)

		/* reverting to a snapshot equal to that of the imported value should still result in a dirty error */
		stateProvider.NotifyRevertedToSnapshot(snapId)
		_, err = stateProvider.ImportStorageAt(contractAddr, slotKey, snapId)
		assert.NotNil(t, err)
		assert.True(t, err.CannotQueryDirtySlot)

		/* reverting to a snapshot before that of the imported value should result in the value being returned */
		stateProvider.NotifyRevertedToSnapshot(snapId - 1)
		data, err = stateProvider.ImportStorageAt(contractAddr, slotKey, snapId)
		assert.Nil(t, err)
		assert.EqualValues(t, expectedData, data)
	}

	/* test for populated slot */
	importTest(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, fixture.StorageSlotPopulatedData)
	/* test for empty slot */
	importTest(fixture.StateObjectContractAddress, fixture.StorageSlotEmptyKey, fixture.StorageSlotEmpty)
}

func TestRemoteStateProvider_MarkSlotWritten(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	stateProvider := newRemoteStateProvider(fixture.Backend)
	snapId := 5

	/* marking a slot as dirty should result in a dirty error when it's read */
	stateProvider.MarkSlotWritten(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, snapId)
	_, err := stateProvider.ImportStorageAt(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, snapId)
	assert.NotNil(t, err)
	assert.Error(t, err.Error)
	assert.True(t, err.CannotQueryDirtySlot)

	/* reverting to a snapshot before the mark should allow the value to be read */
	stateProvider.NotifyRevertedToSnapshot(snapId - 1)
	_, err = stateProvider.ImportStorageAt(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, snapId)
	assert.Nil(t, err)

	/*
		If a slot is written to twice in two successive snapshots, reverting one of the snapshots should not make the
		value readable from the remoteStateProvider.
	*/
	initSnapId := snapId - 2
	stateProvider.NotifyRevertedToSnapshot(initSnapId)
	_, err = stateProvider.ImportStorageAt(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, initSnapId)
	assert.Nil(t, err)

	// first write
	stateProvider.MarkSlotWritten(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, initSnapId)
	_, err = stateProvider.ImportStorageAt(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, initSnapId)
	assert.NotNil(t, err)
	assert.Error(t, err.Error)
	assert.True(t, err.CannotQueryDirtySlot)

	// second write
	secondSnapId := initSnapId + 1
	stateProvider.MarkSlotWritten(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, secondSnapId)
	_, err = stateProvider.ImportStorageAt(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, secondSnapId)
	assert.NotNil(t, err)
	assert.Error(t, err.Error)
	assert.True(t, err.CannotQueryDirtySlot)

	// revert to initSnapId
	stateProvider.NotifyRevertedToSnapshot(initSnapId)

	// ensure it's still dirty
	_, err = stateProvider.ImportStorageAt(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, initSnapId)
	assert.NotNil(t, err)
	assert.Error(t, err.Error)
	assert.True(t, err.CannotQueryDirtySlot)
}

func TestRemoteStateProvider_MarkContractDeployed(t *testing.T) {
	fixture := newPrePopulatedBackendFixture()
	stateProvider := newRemoteStateProvider(fixture.Backend)
	snapId := 5

	/* marking a contract as deployed should result in a dirty error when we try to read its slots */
	stateProvider.MarkContractDeployed(fixture.StateObjectContractAddress, snapId)
	_, err := stateProvider.ImportStorageAt(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, snapId)
	assert.NotNil(t, err)
	assert.Error(t, err.Error)
	assert.True(t, err.CannotQueryDirtySlot)

	/* reverting to a snapshot before the mark should allow the value to be read */
	stateProvider.NotifyRevertedToSnapshot(snapId - 1)
	_, err = stateProvider.ImportStorageAt(fixture.StateObjectContractAddress, fixture.StorageSlotPopulatedKey, snapId)
	assert.Nil(t, err)
}
