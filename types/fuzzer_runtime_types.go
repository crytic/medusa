package types

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/compilation/types"
)

// TODO: need to move this into types/ folder

// DeployedMethod describes a method which is accessible through the contract actively deployed on the test node.
type DeployedMethod struct {
	// Address represents the Ethereum address where the deployed Contract containing the Method exists.
	Address common.Address

	// Contract describes the contract which was deployed and contains the target Method.
	Contract types.CompiledContract

	// Method describes the method which is available through the deployed Contract.
	Method abi.Method
}

// TxSequenceElement describes an element of a transaction sequence.
type TxSequenceElement struct {
	// Tx represents the actual transaction sent to the testNode in a fuzzing transaction sequence.
	Tx *coreTypes.LegacyTx
	// Sender represents the account which was selected to send the tx on the testNode.
	Sender *FuzzerAccount
}

// NewTxSequenceElement creates a new TxSequenceElement, which represents a single transaction in a transaction sequence
// tested during the fuzzing campaign.
func NewTxSequenceElement(tx *coreTypes.LegacyTx, sender *FuzzerAccount) *TxSequenceElement {
	// Create a sequence element and return it.
	elem := &TxSequenceElement{
		Tx:     tx,
		Sender: sender,
	}
	return elem
}

// FuzzerAccount represents a single keypair generated or derived from settings provided in the Fuzzer.config.
type FuzzerAccount struct {
	// key describes the ecdsa private key of an account used a Fuzzer instance.
	Key *ecdsa.PrivateKey
	// address represents the ethereum address which corresponds to key.
	Address common.Address
}
