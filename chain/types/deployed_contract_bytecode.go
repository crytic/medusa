package types

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/compilation/types"
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

// IsMatch returns a boolean indicating whether the deployed contract bytecode is a match with the provided compiled
// contract.
func (c *DeployedContractBytecode) IsMatch(contract *types.CompiledContract) bool {
	// Obtain the provided contract definition's init and runtime byte code/
	contractInitBytecode, err := contract.InitBytecodeBytes()
	if err != nil {
		return false
	}
	contractRuntimeBytecode, err := contract.RuntimeBytecodeBytes()
	if err != nil {
		return false
	}

	// Check if we can compare init and runtime bytecode
	canCompareInit := len(c.InitBytecode) > 0 && len(contractInitBytecode) > 0
	canCompareRuntime := len(c.RuntimeBytecode) > 0 && len(contractRuntimeBytecode) > 0

	// First try matching runtime bytecode contract metadata.
	if canCompareRuntime {
		// First we try to match contracts with contract metadata embedded within the smart contract.
		// Note: We use runtime bytecode for this because init byte code can have matching metadata hashes for different
		// contracts.
		deploymentMetadata := types.ExtractContractMetadata(c.RuntimeBytecode)
		definitionMetadata := types.ExtractContractMetadata(contractRuntimeBytecode)
		if deploymentMetadata != nil && definitionMetadata != nil {
			deploymentBytecodeHash := deploymentMetadata.ExtractBytecodeHash()
			definitionBytecodeHash := definitionMetadata.ExtractBytecodeHash()
			if deploymentBytecodeHash != nil && definitionBytecodeHash != nil {
				return bytes.Equal(deploymentBytecodeHash, definitionBytecodeHash)
			}
		}
	}

	// Since we could not match with runtime bytecode's metadata hashes, we try to match based on init code. To do this,
	// we anticipate our init bytecode might contain appended arguments, so we'll be slicing it down to size and trying
	// to match as a last ditch effort.
	if canCompareInit {
		// If the init byte code size is larger than what we initialized with, it is not a match.
		if len(contractInitBytecode) > len(c.InitBytecode) {
			return false
		}

		// Cut down the contract init bytecode to the size of the definition's to attempt to strip away constructor
		// arguments before performing a direct compare.
		cutDeployedInitBytecode := c.InitBytecode[:len(contractInitBytecode)]

		// If the byte code matches exactly, we treat this as a match.
		if bytes.Equal(cutDeployedInitBytecode, contractInitBytecode) {
			return true
		}
	}

	// As a final fallback, try to compare the whole runtime byte code (least likely to work, given the deployment
	// process, e.g. smart contract constructor, will change the runtime code in most cases).
	if canCompareRuntime {
		// If the byte code matches exactly, we treat this as a match.
		if bytes.Equal(c.RuntimeBytecode, contractRuntimeBytecode) {
			return true
		}
	}

	// Otherwise return our failed match status.
	return false
}
