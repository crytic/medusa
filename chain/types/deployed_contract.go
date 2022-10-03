package types

import "github.com/ethereum/go-ethereum/common"

// DeployedContract describes a contract which is actively deployed on-chain at a given address.
type DeployedContract struct {
	// Address represents the Ethereum address where the deployed contract containing the method exists.
	Address common.Address

	// InitBytecode describes the bytecode used to deploy the contract.
	InitBytecode []byte

	// RuntimeBytecode describes the bytecode that exists after deployment of the contract.
	RuntimeBytecode []byte
}
