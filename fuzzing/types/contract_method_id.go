package types

import "github.com/ethereum/go-ethereum/accounts/abi"

// ContractMethodID is a string-based identifier which is meant to be unique for any contract-method definition
// pair, such that it can be used to index a specific contract method.
type ContractMethodID string

// GetContractMethodID returns a unique string for the contract and method definition provided. This is used to
// identify a contract and method definition and generate keys for mappings for a given method.
func GetContractMethodID(contract *Contract, method *abi.Method) ContractMethodID {
	return ContractMethodID(contract.Name() + "." + method.Sig)
}
