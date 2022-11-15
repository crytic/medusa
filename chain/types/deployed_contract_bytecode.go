package types

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/compilation/types"
)

// DeployedContractBytecode describes contract bytecode which is actively deployed on-chain at a given address.
type DeployedContractBytecode struct {
	// Address represents the Ethereum address where the deployed contract containing the method exists.
	Address common.Address

	// InitBytecode describes the bytecode used to deploy the contract.
	InitBytecode []byte

	// RuntimeBytecode describes the bytecode which was deployed by the InitBytecode.
	RuntimeBytecode []byte
}

// IsMatch returns a boolean indicating whether the deployed contract bytecode is a match with the provided compiled
// contract.
func (c *DeployedContractBytecode) IsMatch(contract *types.CompiledContract) bool {
	// Obtain the provided contract definition's runtime byte code as a byte slice.
	contractRuntimeBytecode, err := contract.RuntimeBytecodeBytes()
	if err != nil {
		return false
	}

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

	// Since we could not match with runtime bytecode's metadata hashes, we try to match based on init code. To do this,
	// we anticipate our init bytecode might contain appended arguments, so we'll be slicing it down to size and trying
	// to match as a last ditch effort.

	// Obtain the provided contract definition's init byte code as a byte slice.
	contractInitBytecode, err := contract.InitBytecodeBytes()
	if err != nil {
		return false
	}

	// If the init byte code size is larger than what we initialized with, it is not a match.
	if len(contractInitBytecode) > len(c.InitBytecode) {
		return false
	}

	// Cut down the contract init bytecode to the size of the definition's to attempt to strip away constructor
	// arguments before performing a direct compare.
	cutDeployedInitBytecode := c.InitBytecode[:len(contractInitBytecode)]

	// If the byte code matches exactly, we treat this as a match.
	if bytes.Compare(cutDeployedInitBytecode, contractInitBytecode) == 0 {
		return true
	}

	// Otherwise return our failed match status.
	return false
}
