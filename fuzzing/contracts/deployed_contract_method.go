package contracts

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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
