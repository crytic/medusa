package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var _ stateBackend = (*EmptyBackend)(nil)

/*
EmptyBackend defines a backend containing no data. Intended to be used for local-only state databases.
*/
type EmptyBackend struct{}

func (d EmptyBackend) GetStorageAt(address common.Address, hash common.Hash) (common.Hash, error) {
	return common.Hash{}, nil
}

func (d EmptyBackend) GetStateObject(address common.Address) (*uint256.Int, uint64, []byte, error) {
	return uint256.NewInt(0), 0, nil, nil
}
