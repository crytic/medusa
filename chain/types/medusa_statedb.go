package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

var _ MedusaStateDB = (*state.StateDB)(nil)
var _ MedusaStateDB = (*state.ForkStateDb)(nil)

type MedusaStateDB interface {
	vm.StateDB
	// geth's built-in statedb interface is not complete.
	// We need to add the extra methods that Medusa uses.
	IntermediateRoot(bool) common.Hash
	Finalise(bool)
	Logs() []*types.Log
	GetLogs(common.Hash, uint64, common.Hash) []*types.Log
	TxIndex() int
	SetBalance(common.Address, *uint256.Int, tracing.BalanceChangeReason)
	SetTxContext(common.Hash, int)
	Commit(uint64, bool) (common.Hash, error)
	SetLogger(*tracing.Hooks)
}
