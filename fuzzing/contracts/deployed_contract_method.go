package contracts

import (
	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/common"
)

// DeployedContractMethod describes a method which is accessible through a contract actively deployed on-chain.
type DeployedContractMethod struct {
	// Address represents the Ethereum address where the deployed contract containing the method exists.
	Address common.Address

	// Contract describes the contract which was deployed and contains the target method.
	Contract *Contract

	// Method describes the method which is available through the deployed contract.
	Method abi.Method
}
