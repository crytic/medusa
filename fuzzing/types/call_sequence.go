package types

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
)

// CallSequence describes a sequence of calls sent to a chain.
type CallSequence []*CallSequenceElement

// NewCallSequence returns a new CallSequence struct to track a sequence of calls made to a chain.
func NewCallSequence() *CallSequence {
	callSequence := make(CallSequence, 0)
	return &callSequence
}

// String returns a displayable string representing the CallSequence.
func (cs CallSequence) String() string {
	// Construct a list of strings for each CallSequenceElement.
	elementStrings := make([]string, len(cs))
	for i := 0; i < len(elementStrings); i++ {
		elementStrings[i] = cs[i].String()
	}

	// Join each element with new lines and return it.
	return strings.Join(elementStrings, "\n")
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

// String returns a displayable string representing the CallSequenceElement.
func (cse *CallSequenceElement) String() string {
	// Obtain our tx and decode our method from this.
	method, err := cse.Method()
	if err != nil || method == nil {
		panic("failed to evaluate failed test method from call sequence data")
	}

	// Next decode our arguments (we jump four bytes to skip the function selector)
	args, err := method.Inputs.Unpack(cse.Call().Data()[4:])
	if err != nil {
		panic("failed to unpack method args from transaction data")
	}

	// Serialize our args to a JSON string and set it as our tx method name for this index.
	// TODO: Byte arrays are encoded as base64 strings, so this should be represented another way in the future:
	//  Reference: https://stackoverflow.com/questions/14177862/how-to-marshal-a-byte-uint8-array-as-json-array-in-go
	b, err := json.Marshal(args)
	if err != nil {
		b = []byte("<error resolving args>")
	}

	return fmt.Sprintf(
		"%s.%s(%s) (gas=%d, gasprice=%s, value=%s, sender=%s)",
		cse.Contract().Name(),
		method.Name,
		string(b),
		cse.Call().Gas(),
		cse.Call().GasPrice().String(),
		cse.Call().Value().String(),
		cse.Call().From(),
	)
}
