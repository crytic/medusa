package fork

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type RemoteStateCache interface {
	GetStorageAt(common.Address, common.Hash) (common.Hash, error)
	GetStateObject(common.Address) (*uint256.Int, uint64, []byte, error)
}

var _ RemoteStateCache = (*EmptyRemoteStateCache)(nil)

type EmptyRemoteStateCache struct{}

func (d EmptyRemoteStateCache) GetStorageAt(address common.Address, hash common.Hash) (common.Hash, error) {
	return common.Hash{}, nil
}

func (d EmptyRemoteStateCache) GetStateObject(address common.Address) (*uint256.Int, uint64, []byte, error) {
	return uint256.NewInt(0), 0, nil, nil
}
