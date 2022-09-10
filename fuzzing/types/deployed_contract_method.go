package types

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// DeployedContractMethod describes a method which is accessible through a contract actively deployed on a fuzzing.TestNode.
type DeployedContractMethod struct {
	// Address represents the Ethereum address where the deployed contract containing the method exists.
	Address common.Address

	// Contract describes the contract which was deployed and contains the target method.
	Contract *Contract

	// Method describes the method which is available through the deployed contract.
	Method abi.Method
}

// ContractMethodID is a string-based identifier which is meant to be unique for any contract-method definition
// pair, such that it can be used to index a specific contract method.
type ContractMethodID string

// GetContractMethodID returns a unique string for the contract and method definition provided. This is used to
// identify a contract and method definition and generate keys for mappings for a given method.
func GetContractMethodID(contract *Contract, method *abi.Method) ContractMethodID {
	return ContractMethodID(contract.Name() + "." + method.Sig)
}
