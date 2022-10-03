package types

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/compilation/types"
)

// DeployedContract describes a contract which is actively deployed on-chain at a given address.
type DeployedContract struct {
	// Address represents the Ethereum address where the deployed contract containing the method exists.
	Address common.Address

	// InitBytecode describes the bytecode used to deploy the contract.
	InitBytecode []byte

	// RuntimeBytecode describes the bytecode that exists after deployment of the contract.
	RuntimeBytecode []byte
}

// IsMatch returns a boolean indicating whether the deployed contract is a match with the provided compiled contract.
func (c *DeployedContract) IsMatch(contract *types.CompiledContract) bool {
	// TODO: Matching logic
	// Obtain the contract byte code as a byte slice.
	contractInitBytecode, err := contract.InitBytecodeBytes()
	if err != nil {
		return false
	}

	// If the byte code matches exactly, we treat this as a match.
	if bytes.Compare(c.InitBytecode, contractInitBytecode) == 0 {
		return true
	}

	return false
}
