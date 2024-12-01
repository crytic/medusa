package fork

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var _ RemoteStateQuery = (*DummyRemoteStateQuery)(nil)

type DummyRemoteStateQuery struct{}

func (d DummyRemoteStateQuery) GetStorageAt(address common.Address, hash common.Hash) (common.Hash, error) {
	return common.Hash{}, nil
}

func (d DummyRemoteStateQuery) GetStateObject(address common.Address) (*uint256.Int, uint64, []byte, error) {
	return uint256.NewInt(0), 0, nil, nil
}
