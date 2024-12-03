package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

/* This file is exclusively for test fixtures. */

var _ StateBackend = (*prePopulatedBackend)(nil)

// prePopulatedBackend is an offline-only backend used for testing.
type prePopulatedBackend struct {
	storageSlots map[common.Address]map[common.Hash]common.Hash
	stateObjects map[common.Address]remoteStateObject
}

func newPrepopulatedBackend(
	storageSlots map[common.Address]map[common.Hash]common.Hash,
	stateObjects map[common.Address]remoteStateObject,
) *prePopulatedBackend {
	return &prePopulatedBackend{
		storageSlots: storageSlots,
		stateObjects: stateObjects,
	}
}

func (p *prePopulatedBackend) GetStorageAt(address common.Address, hash common.Hash) (common.Hash, error) {
	if c, exists := p.storageSlots[address]; exists {
		if data, exists := c[hash]; exists {
			return data, nil
		}
	}
	return common.Hash{}, nil
}

func (p *prePopulatedBackend) GetStateObject(address common.Address) (*uint256.Int, uint64, []byte, error) {
	if s, exists := p.stateObjects[address]; exists {
		return s.Balance, s.Nonce, s.Code, nil
	}
	return uint256.NewInt(0), uint64(0), []byte{}, nil
}

func (p *prePopulatedBackend) SetStorageAt(address common.Address, slotKey common.Hash, value common.Hash) {
	if _, exists := p.storageSlots[address]; !exists {
		p.storageSlots[address] = make(map[common.Hash]common.Hash)
	}
	p.storageSlots[address][slotKey] = value
}

// prepopulatedBackendFixture is a test fixture for a pre-populated backend
type prepopulatedBackendFixture struct {
	Backend *prePopulatedBackend

	StateObjectContractAddress common.Address
	StateObjectContract        remoteStateObject

	StorageSlotPopulatedKey  common.Hash
	StorageSlotPopulatedData common.Hash

	StorageSlotEmptyKey common.Hash
	StorageSlotEmpty    common.Hash

	StateObjectEOAAddress common.Address
	StateObjectEOA        remoteStateObject

	StateObjectEmptyAddress common.Address
	StateObjectEmpty        remoteStateObject
}

func newPrePopulatedBackendFixture() *prepopulatedBackendFixture {
	stateObjectContract := remoteStateObject{
		Balance: uint256.NewInt(1000),
		Nonce:   5,
		Code:    []byte{1, 2, 3},
	}
	stateObjectEOA := remoteStateObject{
		Balance: uint256.NewInt(5000),
		Nonce:   1,
		Code:    nil,
	}

	stateObjectEmpty := remoteStateObject{
		Balance: uint256.NewInt(0),
		Nonce:   0,
		Code:    nil,
	}

	contractAddress := common.BytesToAddress([]byte{5, 5, 5, 5})
	eoaAddress := common.BytesToAddress([]byte{6, 6, 6, 6})
	emptyAddress := common.BytesToAddress([]byte{0, 0, 0, 1})

	storageSlotPopulated := common.HexToHash("0xdeadbeef")
	storageSlotPopulatedAddress := common.HexToHash("0xaaaaaaaa")

	storageSlotEmpty := common.Hash{}
	storageSlotEmptyAddress := common.HexToHash("0xbbbbbbbbb")

	stateObjects := make(map[common.Address]remoteStateObject)
	stateObjects[contractAddress] = stateObjectContract
	stateObjects[eoaAddress] = stateObjectEOA
	stateObjects[emptyAddress] = stateObjectEmpty

	storageObjects := make(map[common.Address]map[common.Hash]common.Hash)
	storageObjects[contractAddress] = make(map[common.Hash]common.Hash)
	storageObjects[contractAddress][storageSlotPopulatedAddress] = storageSlotPopulated
	storageObjects[contractAddress][storageSlotEmptyAddress] = storageSlotEmpty

	prepopulatedBackend := newPrepopulatedBackend(storageObjects, stateObjects)

	return &prepopulatedBackendFixture{
		Backend:                    prepopulatedBackend,
		StateObjectContractAddress: contractAddress,
		StateObjectContract:        stateObjectContract,
		StorageSlotPopulatedKey:    storageSlotPopulatedAddress,
		StorageSlotPopulatedData:   storageSlotPopulated,
		StorageSlotEmptyKey:        storageSlotEmptyAddress,
		StorageSlotEmpty:           storageSlotEmpty,
		StateObjectEOAAddress:      eoaAddress,
		StateObjectEOA:             stateObjectEOA,
		StateObjectEmpty:           stateObjectEmpty,
		StateObjectEmptyAddress:    emptyAddress,
	}
}
