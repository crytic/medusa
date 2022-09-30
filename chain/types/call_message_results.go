package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// CallMessageResults represents metadata obtained from the execution of a CallMessage in a Block.
// This contains results such as contracts deployed, and other variables tracked by a chain.TestChain.
type CallMessageResults struct {
	// DeployedContractAddresses refers to addresses which hold newly deployed contracts as a result of the transaction
	// this metadata belongs to.
	DeployedContractAddresses []common.Address
}
