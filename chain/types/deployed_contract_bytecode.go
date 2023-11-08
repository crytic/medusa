package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// DeployedContractBytecodeChange describes a change made to a given contract addresses' code due to state updates
// (e.g. from a transaction being processed).
type DeployedContractBytecodeChange struct {
	// Contract describes the contract address which was affected as well as the relevant init and runtime bytecode.
	// If this change represents a creation, this is the new bytecode information. If it represents destruction, this
	// is the destroyed bytecode information.
	Contract *DeployedContractBytecode

	// Creation indicates whether the change made was a contract creation. This cannot be true if SelfDestructed or
	// Destroyed are true.
	Creation bool

	// DynamicCreation indicates whether the change made was a _dynamic_ contract creation. This cannot be true if
	// Creation is false.
	DynamicCreation bool

	// SelfDestructed indicates whether the change made was due to a self-destruct instruction being executed. This
	// cannot be true if Creation is true.
	// Note: This may not be indicative of contract removal (as is the case with Destroyed), as proposed changes to
	// the `SELFDESTRUCT` instruction aim to not remove contract code.
	SelfDestructed bool

	// Destroyed indicates whether the contract was destroyed as a result of the operation, indicating the code
	// provided by Contract is no longer available.
	Destroyed bool
}

// DeployedContractBytecode describes the init and runtime bytecode recorded for a given contract address.
type DeployedContractBytecode struct {
	// Address represents the Ethereum address where the deployed contract containing the method exists.
	Address common.Address

	// InitBytecode describes the bytecode used to deploy the contract.
	InitBytecode []byte

	// RuntimeBytecode describes the bytecode which was deployed by the InitBytecode. This is expected to be non-nil.
	RuntimeBytecode []byte
}
