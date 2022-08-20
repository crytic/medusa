package fuzzing

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/compilation/types"
)

// deployedMethod describes a method which is accessible through the contract actively deployed on the test node.
type deployedMethod struct {
	// address represents the Ethereum address where the deployed contract containing the method exists.
	address common.Address

	// contract describes the contract which was deployed and contains the target method.
	contract types.CompiledContract

	// method describes the method which is available through the deployed contract.
	method abi.Method
}

// txSequenceElement describes an element of a transaction sequence.
type txSequenceElement struct {
	// tx represents the actual transaction sent to the TestNode in a fuzzing transaction sequence.
	tx *coreTypes.LegacyTx
	// sender represents the account which was selected to send the tx on the TestNode.
	sender *fuzzerAccount
}

// newTxSequenceElement creates a new txSequenceElement, which represents a single transaction in a transaction sequence
// tested during the fuzzing campaign.
func newTxSequenceElement(tx *coreTypes.LegacyTx, sender *fuzzerAccount) *txSequenceElement {
	// Create a sequence element and return it.
	elem := &txSequenceElement{
		tx:     tx,
		sender: sender,
	}
	return elem
}
