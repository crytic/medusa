package types

import (
	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/state"
	"github.com/crytic/medusa-geth/core/tracing"
	"github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/holiman/uint256"
)

var _ MedusaStateDB = (*state.StateDB)(nil)
var _ MedusaStateDB = (*state.ForkStateDb)(nil)

/*
MedusaStateDB provides an interface that supersedes the stateDB interface exposed by geth. All of these functions are
implemented by the vanilla geth statedb.
This interface allows the TestChain to use a forked statedb and native geth statedb interoperably.
*/
type MedusaStateDB interface {
	vm.StateDB
	IntermediateRoot(bool) common.Hash
	Finalise(bool)
	Logs() []*types.Log
	GetLogs(common.Hash, uint64, common.Hash) []*types.Log
	TxIndex() int
	SetBalance(common.Address, *uint256.Int, tracing.BalanceChangeReason)
	SetTxContext(common.Hash, int)
	Commit(uint64, bool) (common.Hash, error)
	SetLogger(*tracing.Hooks)
	Error() error
}
