package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// MessageResults represents metadata obtained from the execution of a CallMessage in a Block.
// This contains results such as contracts deployed, and other variables tracked by a chain.TestChain.
type MessageResults struct {

	// PostStateRoot refers to the state root hash after the execution of this transaction.
	PostStateRoot common.Hash

	// ExecutionResult describes the core.ExecutionResult returned after processing a given call.
	ExecutionResult *core.ExecutionResult

	// Receipt represents the transaction receipt
	Receipt *types.Receipt

	// ContractDeploymentChanges describes changes made to deployed contracts, such as creation and destruction.
	ContractDeploymentChanges []DeployedContractBytecodeChange

	// AdditionalResults represents results of arbitrary types which can be stored by any part of the application,
	// such as a tracers.
	AdditionalResults map[string]any

	// OnRevertHookFuncs refers hook functions that should be executed when this transaction is reverted.
	// This is to be used when a non-vm safe operation occurs, such as patching chain ID mid-execution, to ensure
	// that when the transaction is reverted, the value is also restored.
	// The hooks are executed as a stack (to support revert operations).
	OnRevertHookFuncs GenericHookFuncs
}
