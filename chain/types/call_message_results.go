package types

import "github.com/ethereum/go-ethereum/core"

// CallMessageResults represents metadata obtained from the execution of a CallMessage in a Block.
// This contains results such as contracts deployed, and other variables tracked by a chain.TestChain.
type CallMessageResults struct {
	// DeployedContractBytecodes describes contracts which were deployed on-chain as a result of the relevant call
	// message.
	DeployedContractBytecodes []*DeployedContractBytecode

	// ExecutionResult describes the core.ExecutionResult returned after processing a given call.
	ExecutionResult *core.ExecutionResult
}
