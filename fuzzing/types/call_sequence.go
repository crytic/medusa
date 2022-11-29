package types

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/chain/types"
	"strconv"
	"strings"
)

// CallSequence describes a sequence of calls sent to a chain.
type CallSequence []*CallSequenceElement

// CallSequenceExecuteStepFunc describes a function that should be called at each step of call sequence execution
// to determine if we should continue. It takes the current index of the element we're processing, and returns a
// boolean indicating whether the execution should break, or an error if one occurs.
type CallSequenceExecuteStepFunc func(index int) (bool, error)

// ExecuteOnChain executes the CallSequence upon a provided chain. It ensures calls are included in blocks which
// adhere to the CallSequence properties (such as delays) as much as possible. If indicated, the last pending block
// will be committed to the chain. A step function is provided to track the last processed index in the call sequence.
// The step function returns a boolean which is true if the call sequence processing should stop at this index, or an
// error if one occurs.
// ExecuteOnChain returns the amount of elements executed in the sequence and an error if one occurs.
func (cs CallSequence) ExecuteOnChain(chain *chain.TestChain, commitLastPendingBlock bool, preStepFunc CallSequenceExecuteStepFunc, postStepFunc CallSequenceExecuteStepFunc) (int, error) {
	// Define a variable to indicate if the step function requested the call sequence processing to exit.
	stepRequestedExit := false

	// Loop through each sequence element in our sequence we'll want to execute.
	executedCount := 0
	for i := 0; i < len(cs); i++ {
		// We try to add the transaction with our call more than once. If the pending block is too full, we may hit a
		// block gas limit, which we handle by committing the pending block without this tx, and creating a new pending
		// block that is empty to try again. If we encounter an error an empty block, we throw the error.
		for {
			// We are trying to add a call to a block as a transaction. Call our step function with the update and check
			// if it returned an error.
			var err error
			if preStepFunc != nil {
				stepRequestedExit, err = preStepFunc(i)
				if err != nil {
					return executedCount, err
				}

				// Stop if the step function indicated we should.
				if stepRequestedExit {
					break
				}
			}

			// If we have a pending block, but we intend to delay this call from the last, we commit that block.
			if chain.PendingBlock() != nil && cs[i].BlockNumberDelay > 0 {
				err := chain.PendingBlockCommit()
				if err != nil {
					return executedCount, err
				}
			}

			// If we have no pending block to add a tx containing our call to, we must create one.
			if chain.PendingBlock() == nil {
				// The minimum step between blocks must be 1 in block number and timestamp, so we ensure this is the
				// case.
				numberDelay := cs[i].BlockNumberDelay
				timeDelay := cs[i].BlockTimestampDelay
				if numberDelay == 0 {
					numberDelay = 1
				}
				if timeDelay == 0 {
					timeDelay = 1
				}

				// Each timestamp/block number must be unique as well, so we cannot jump more block numbers than time.
				if numberDelay > timeDelay {
					numberDelay = timeDelay
				}
				_, err := chain.PendingBlockCreateWithParameters(chain.Head().Header.Number.Uint64()+numberDelay, chain.Head().Header.Time+timeDelay, nil)
				if err != nil {
					return executedCount, err
				}
			}

			// Try to add our transaction to this block.
			err = chain.PendingBlockAddTx(cs[i].Call)
			if err != nil {
				// If we encountered a block gas limit error, this tx is too expensive to fit in this block.
				// If there are other transactions in the block, this makes sense. The block is "full".
				// In that case, we commit the pending block without this tx, and create a new pending block to add
				// our tx to, and iterate to try and add it again.
				// TODO: This should also check the condition that this is a block gas error specifically. For now, we
				//  simply assume it is and try processing in an empty block (if that fails, that error will be
				//  returned).
				if len(chain.PendingBlock().Messages) > 0 {
					err := chain.PendingBlockCommit()
					if err != nil {
						return executedCount, err
					}
					continue
				}

				// If there are no transactions in our block, and we failed to add this one, return the error
				return executedCount, err
			}

			// Update our chain reference for this element.
			cs[i].ChainReference = &CallSequenceElementChainReference{
				Block:            chain.PendingBlock(),
				TransactionIndex: len(chain.PendingBlock().Messages) - 1,
			}

			// Add to our executed count
			executedCount++

			// We added our call to the block as a transaction. Call our step function with the update and check
			// if it returned an error.
			if postStepFunc != nil {
				stepRequestedExit, err = postStepFunc(i)
				if err != nil {
					return executedCount, err
				}

				// Stop if the step function indicated we should.
				if stepRequestedExit {
					break
				}
			}

			// Exit this loop attempting to add the current element, as we've been successful.
			// This moves onto processing the next element.
			break
		}

		// Stop if the step function indicated we should.
		if stepRequestedExit {
			break
		}
	}

	// Commit the last pending block if we're instructed to.
	if commitLastPendingBlock && chain.PendingBlock() != nil {
		err := chain.PendingBlockCommit()
		if err != nil {
			return executedCount, err
		}
	}
	return executedCount, nil
}

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

// Clone creates a copy of the underlying CallSequence.
func (cs CallSequence) Clone() CallSequence {
	r := make(CallSequence, len(cs))
	for i := 0; i < len(r); i++ {
		r[i] = cs[i].Clone()
	}
	return r
}

// CallSequenceElement describes a single call in a call sequence (tx sequence) targeting a specific contract.
// It contains the information regarding the contract/method being called as well as the call message data itself.
type CallSequenceElement struct {
	// Contract describes the contract which was targeted by a transaction.
	Contract *Contract

	// Call represents the underlying message call.
	Call *types.CallMessage `json:"call"`

	// BlockNumberDelay defines how much the block number should advance when executing this transaction, compared to
	// the last executed transaction. If zero, this indicates the call should be included in the current pending block.
	// This number is *suggestive*: if delay specifies we should add a tx to a block which is full, it will be added to
	// a new block instead. If BlockNumberDelay is greater than BlockTimestampDelay and both are non-zero (we want to
	// create a new block), BlockNumberDelay will be capped to BlockTimestampDelay, as each block must have a unique
	// time stamp for chain semantics.
	BlockNumberDelay uint64 `json:"blockNumberDelay"`

	// BlockTimestampDelay defines how much the block timestamp should advance when executing this transaction,
	// compared to the last executed transaction.
	// This number is *suggestive*: if BlockNumberDelay is non-zero (indicating to add to the existing block), this
	// value will not be used.
	BlockTimestampDelay uint64 `json:"blockTimestampDelay"`

	// ChainReference describes the inclusion of the Call as a transaction in a block. This block may not yet be
	// committed to its underlying chain if this is a CallSequenceElement was just executed. Additional transactions
	// may be included before the block is committed. This reference will remain compatible after the block finalizes.
	ChainReference *CallSequenceElementChainReference
}

// NewCallSequenceElement returns a new CallSequenceElement struct to track a single call made within a CallSequence.
func NewCallSequenceElement(contract *Contract, call *types.CallMessage, blockNumberDelay uint64, blockTimestampDelay uint64) *CallSequenceElement {
	callSequenceElement := &CallSequenceElement{
		Contract:            contract,
		Call:                call,
		BlockNumberDelay:    blockNumberDelay,
		BlockTimestampDelay: blockTimestampDelay,
		ChainReference:      nil,
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

	// If we have runtime info, populate it
	blockNumberStr := "n/a"
	blockTimeStr := "n/a"
	if cse.ChainReference != nil {
		blockNumberStr = cse.ChainReference.Block.Header.Number.String()
		blockTimeStr = strconv.FormatUint(cse.ChainReference.Block.Header.Time, 10)
	}

	// Return a formatted string representing this element.
	return fmt.Sprintf(
		"%s.%s(%s) (block=%s, time=%s, gas=%d, gasprice=%s, value=%s, sender=%s)",
		contractName,
		methodName,
		argsText,
		blockNumberStr,
		blockTimeStr,
		cse.Call.Gas(),
		cse.Call.GasPrice().String(),
		cse.Call.Value().String(),
		cse.Call.From(),
	)
}

// MarshalJSON provides the default serialization routine for a CallSequenceElement.
func (cse *CallSequenceElement) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Call                *types.CallMessage `json:"call"`
		BlockNumberDelay    uint64             `json:"blockNumberDelay"`
		BlockTimestampDelay uint64             `json:"blockTimestampDelay"`
	}{
		Call:                cse.Call,
		BlockNumberDelay:    cse.BlockNumberDelay,
		BlockTimestampDelay: cse.BlockTimestampDelay,
	})
}

// UnmarshalJSON provides the default deserialization routine for a CallSequenceElement.
func (cse *CallSequenceElement) UnmarshalJSON(data []byte) error {
	// Define our definition and deserialize our data.
	type jsonDefinition struct {
		Call                *types.CallMessage `json:"call"`
		BlockNumberDelay    uint64             `json:"blockNumberDelay"`
		BlockTimestampDelay uint64             `json:"blockTimestampDelay"`
	}
	var def jsonDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return err
	}
	cse.Call = def.Call
	cse.BlockNumberDelay = def.BlockNumberDelay
	cse.BlockTimestampDelay = def.BlockTimestampDelay
	return nil
}

// Clone creates a copy of the underlying CallSequenceElement.
func (cse *CallSequenceElement) Clone() *CallSequenceElement {
	return &CallSequenceElement{
		Contract:            cse.Contract,
		Call:                cse.Call,
		BlockNumberDelay:    cse.BlockNumberDelay,
		BlockTimestampDelay: cse.BlockTimestampDelay,
		ChainReference:      cse.ChainReference,
	}
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
