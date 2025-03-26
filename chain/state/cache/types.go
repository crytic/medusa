package cache

import (
	"github.com/crytic/medusa-geth/common"
	"github.com/holiman/uint256"
)

// StateObject gives us a way to store state objects without the overhead of using geth's stateObject
type StateObject struct {
	Balance *uint256.Int
	Nonce   uint64
	Code    []byte
}

type StateCache interface {
	GetStateObject(addr common.Address) (*StateObject, error)
	WriteStateObject(addr common.Address, data StateObject) error

	GetSlotData(addr common.Address, slot common.Hash) (common.Hash, error)
	WriteSlotData(addr common.Address, slot common.Hash, data common.Hash) error
}
