package types

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// CallMessageResults represents metadata obtained from the execution of a CallMessage in a Block.
// This contains results such as contracts deployed, and other variables tracked by a chain.TestChain.
type CallMessageResults struct {
	// ExecutionResult describes the core.ExecutionResult returned after processing a given call.
	ExecutionResult *core.ExecutionResult

	// Receipt represents the transaction receipt
	Receipt *types.Receipt

	// ContractDeploymentChanges describes changes made to deployed contracts, such as creation and destruction.
	ContractDeploymentChanges []DeployedContractBytecodeChange

	// AdditionalResults represents results of arbitrary types which can be stored by any part of the application,
	// such as a tracers.
	AdditionalResults map[string]any
}
