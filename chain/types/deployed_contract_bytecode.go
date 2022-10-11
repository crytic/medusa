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
}

// IsMatch returns a boolean indicating whether the deployed contract bytecode is a match with the provided compiled
// contract.
func (c *DeployedContractBytecode) IsMatch(contract *types.CompiledContract) bool {
	// Obtain the contract byte code as a byte slice.
	contractInitBytecode, err := contract.InitBytecodeBytes()
	if err != nil {
		return false
	}

	// First we try to match contracts with contract metadata embedded within the smart contract.
	deploymentMetadata := types.ExtractContractMetadata(c.InitBytecode)
	definitionMetadata := types.ExtractContractMetadata(contractInitBytecode)
	if deploymentMetadata != nil && definitionMetadata != nil {
		deploymentBytecodeHash := deploymentMetadata.ExtractBytecodeHash()
		definitionBytecodeHash := definitionMetadata.ExtractBytecodeHash()
		if deploymentBytecodeHash != nil && definitionBytecodeHash != nil {
			return bytes.Equal(deploymentBytecodeHash, definitionBytecodeHash)
		}
	}

	// If the init byte code size is larger than what we initialized with, it is not a match.
	if len(c.InitBytecode) > len(contractInitBytecode) {
		return false
	}

	// As a last ditch effort, cut down the contract init bytecode to the size of the definition's to attempt to strip
	// away constructor arguments before performing a direct compare.
	contractInitBytecode = contractInitBytecode[:len(c.InitBytecode)]

	// If the byte code matches exactly, we treat this as a match.
	if bytes.Compare(c.InitBytecode, contractInitBytecode) == 0 {
		return true
	}

	// Otherwise return our failed match status.
	return false
}
