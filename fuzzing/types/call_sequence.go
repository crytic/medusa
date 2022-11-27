package types

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trailofbits/medusa/chain/types"
	"strings"
)

// CallSequence describes a sequence of calls sent to a chain.
type CallSequence []*CallSequenceElement

// String returns a displayable string representing the CallSequence.
func (cs CallSequence) String() string {
	// If we have an empty call sequence, return a special string
	if len(cs) == 0 {
		return "<none>"
	}

	// Construct a list of strings for each CallSequenceElement.
	elementStrings := make([]string, len(cs))
	for i := 0; i < len(elementStrings); i++ {
		elementStrings[i] = fmt.Sprintf("%d) %s", i+1, cs[i].String())
	}

	// Join each element with new lines and return it.
	return strings.Join(elementStrings, "\n")
}

// CallSequenceElement describes a single call in a call sequence (tx sequence) targeting a specific contract.
// It contains the information regarding the contract/method being called as well as the call message data itself.
type CallSequenceElement struct {
	// contract describes the contract which was targeted by a transaction.
	Contract *Contract

	// call represents the underlying message call.
	Call *types.CallMessage

	// ChainReference describes the inclusion of the Call as a transaction in a block. This block may not yet be
	// committed to its underlying chain if this is a CallSequenceElement was just executed. Additional transactions
	// may be included before the block is committed. This reference will remain compatible after the block finalizes.
	ChainReference *CallSequenceElementChainReference
}

// NewCallSequenceElement returns a new CallSequenceElement struct to track a single call made within a CallSequence.
func NewCallSequenceElement(contract *Contract, call *types.CallMessage) *CallSequenceElement {
	callSequenceElement := &CallSequenceElement{
		Contract:       contract,
		Call:           call,
		ChainReference: nil,
	}
	return callSequenceElement
}

// Method obtains the abi.Method targeted by the CallSequenceElement.Call, or an error if one occurred while obtaining
// it.
func (cse *CallSequenceElement) Method() (*abi.Method, error) {
	// If there is no resolved contract definition, we return no method.
	if cse.Contract == nil {
		return nil, nil
	}
	return cse.Contract.CompiledContract().Abi.MethodById(cse.Call.Data())
}

// String returns a displayable string representing the CallSequenceElement.
func (cse *CallSequenceElement) String() string {
	// Obtain our contract name
	contractName := "<unresolved contract>"
	if cse.Contract != nil {
		contractName = cse.Contract.Name()
	}

	// Obtain our method name
	method, err := cse.Method()
	methodName := "<unresolved method>"
	if err == nil && method != nil {
		methodName = method.Name
	}

	// Next decode our arguments (we jump four bytes to skip the function selector)
	args, err := method.Inputs.Unpack(cse.Call.Data()[4:])
	argsText := "<unresolved args>"
	if err == nil {
		// Serialize our args to a JSON string and set it as our method name if we succeeded.
		// TODO: Byte arrays are encoded as base64 strings, so this should be represented another way in the future:
		//  Reference: https://stackoverflow.com/questions/14177862/how-to-marshal-a-byte-uint8-array-as-json-array-in-go
		var argsJson []byte
		argsJson, err = json.Marshal(args)
		if err == nil {
			argsText = string(argsJson)
		}
	}

	// Return a formatted string representing this element.
	return fmt.Sprintf(
		"%s.%s(%s) (gas=%d, gasprice=%s, value=%s, sender=%s)",
		contractName,
		methodName,
		argsText,
		cse.Call.Gas(),
		cse.Call.GasPrice().String(),
		cse.Call.Value().String(),
		cse.Call.From(),
	)
}

// CallSequenceElementChainReference references the inclusion of a CallSequenceElement's underlying call being
// included in a block as a transaction.
type CallSequenceElementChainReference struct {
	// Block describes the block the CallSequenceElement.Call was included in as a transaction. This block may be
	// pending commitment to the chain, or already committed.
	Block *types.Block

	// TransactionIndex describes the index at which the transaction was included into the Block.
	TransactionIndex int
}

// MessageResults obtains the results of executing the CallSequenceElement.
func (cr *CallSequenceElementChainReference) MessageResults() *types.CallMessageResults {
	return cr.Block.MessageResults[cr.TransactionIndex]
}
