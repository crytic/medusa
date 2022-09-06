package types

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// CallSequence describes a sequence of calls sent to a chain.
type CallSequence []*CallSequenceElement

// NewCallSequence returns a new CallSequence struct to track a sequence of calls made to a chain.
func NewCallSequence() *CallSequence {
	callSequence := make(CallSequence, 0)
	return &callSequence
}

// CallSequenceElement describes a single call in a call sequence (tx sequence) targeting a specific contract.
// It contains the information regarding the contract/method being called as well as the call message data itself.
type CallSequenceElement struct {
	// contract describes the contract which was targeted by a transaction.
	contract *Contract

	// call represents the underlying message call.
	call *CallMessage
}

// NewCallSequenceElement returns a new CallSequenceElement struct to track a single call made within a CallSequence.
func NewCallSequenceElement(contract *Contract, call *CallMessage) *CallSequenceElement {
	failedTx := &CallSequenceElement{
		contract: contract,
		call:     call,
	}
	return failedTx
}

// Contract obtains the Contract instance which is being targeted by CallSequenceElement.Call.
func (cse *CallSequenceElement) Contract() *Contract {
	return cse.contract
}

// Call obtains the CallMessage used in this CallSequenceElement.
func (cse *CallSequenceElement) Call() *CallMessage {
	return cse.call
}

// Method obtains the abi.Method targeted by the CallSequenceElement.Call, or an error if one occurred while obtaining
// it.
func (cse *CallSequenceElement) Method() (*abi.Method, error) {
	// If there is no contract reference, we return no method.
	if cse.contract == nil {
		return nil, nil
	}
	return cse.contract.CompiledContract().Abi.MethodById(cse.call.Data())
}
