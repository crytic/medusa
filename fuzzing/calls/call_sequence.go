package calls

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/crytic/medusa/chain"
	chainTypes "github.com/crytic/medusa/chain/types"
	fuzzingTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/executiontracer"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// CallSequence describes a sequence of calls sent to a chain.
type CallSequence []*CallSequenceElement

// AttachExecutionTraces takes a given chain which executed the call sequence, and a list of contract definitions,
// and it replays each call of the sequence with an execution tracer attached to it, it then sets each
// CallSequenceElement.ExecutionTrace to the resulting trace. Returns an error if one occurred.
func (cs CallSequence) AttachExecutionTraces(chain *chain.TestChain, contractDefinitions fuzzingTypes.Contracts) error {
	// For each call sequence element, attach an execution trace.
	for _, cse := range cs {
		err := cse.AttachExecutionTrace(chain, contractDefinitions)
		if err != nil {
			return err
		}
	}
	return nil
}

// Log returns a logging.LogBuffer that represents this call sequence. This buffer will be passed to the underlying
// logger which will format it accordingly for console or file.
func (cs CallSequence) Log() *logging.LogBuffer {
	buffer := logging.NewLogBuffer()
	// If we have an empty call sequence, return a special string
	if len(cs) == 0 {
		buffer.Append("<none>")
		return buffer
	}

	// Construct the buffer for each call made in the sequence
	for i := 0; i < len(cs); i++ {
		// Add the string representing the call
		buffer.Append(fmt.Sprintf("%d) %s\n", i+1, cs[i].String()))

		// If we have an execution trace attached, print information about it.
		if cs[i].ExecutionTrace != nil {
			buffer.Append(cs[i].ExecutionTrace.Log().Elements()...)
			buffer.Append("\n")
		}
	}

	// Return the buffer
	return buffer
}

// String returns the string representation of this call sequence
func (cs CallSequence) String() string {
	// Internally, we just call the log function, get the list of elements and create their non-colorized string representation
	// Might be useful for 3rd party apps
	return cs.Log().String()
}

// Clone creates a copy of the underlying CallSequence.
func (cs CallSequence) Clone() (CallSequence, error) {
	var err error
	r := make(CallSequence, len(cs))
	for i := 0; i < len(r); i++ {
		r[i], err = cs[i].Clone()
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Hash calculates a unique hash which represents the uniqueness of the call sequence and each element in it. It does
// not hash execution/result data.
// Returns the calculated hash, or an error if one occurs.
func (cs CallSequence) Hash() (common.Hash, error) {
	// Create our hash provider
	hashProvider := crypto.NewKeccakState()
	hashProvider.Reset()
	for _, cse := range cs {
		var temp [8]byte

		// Hash block number delay
		binary.LittleEndian.PutUint64(temp[:], cse.BlockNumberDelay)
		_, err := hashProvider.Write(temp[:])
		if err != nil {
			return common.Hash{}, err
		}

		// Hash block timestamp delay
		binary.LittleEndian.PutUint64(temp[:], cse.BlockTimestampDelay)
		_, err = hashProvider.Write(temp[:])
		if err != nil {
			return common.Hash{}, err
		}

		// Try to pack the call message and obtain a hash for it.
		// This may panic if the ABI changed and the ABI method/function targeted does not resolve or the call
		// could otherwise not be packed/serialized. If it does, we use fixed hash data instead.
		var messageHashData []byte
		func() {
			// If the below operations to obtain a message/call hash fail, we instead substitute it with hardcoded
			// hash data.
			defer func() {
				if r := recover(); r != nil {
					messageHashData = common.Hash{}.Bytes()
				}
			}()

			// Try to obtain a hash for the message/call. If this fails, we will replace it in the deferred panic
			// recovery.
			messageHashData = utils.MessageToTransaction(cse.Call.ToCoreMessage()).Hash().Bytes()
		}()

		// Hash the message hash data.
		_, err = hashProvider.Write(messageHashData)
		if err != nil {
			return common.Hash{}, err
		}
	}

	// Obtain the output hash and return it
	hash := hashProvider.Sum(nil)
	return common.BytesToHash(hash), nil
}

// CallSequenceElement describes a single call in a call sequence (tx sequence) targeting a specific contract.
// It contains the information regarding the contract/method being called as well as the call message data itself.
type CallSequenceElement struct {
	// Contract describes the contract which was targeted by a transaction.
	Contract *fuzzingTypes.Contract `json:"-"`

	// Call represents the underlying message call.
	Call *CallMessage `json:"call"`

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
	ChainReference *CallSequenceElementChainReference `json:"-"`

	// ExecutionTrace represents a verbose execution trace collected. Nil if an execution trace was not collected.
	ExecutionTrace *executiontracer.ExecutionTrace `json:"-"`
}

// NewCallSequenceElement returns a new CallSequenceElement struct to track a single call made within a CallSequence.
func NewCallSequenceElement(contract *fuzzingTypes.Contract, call *CallMessage, blockNumberDelay uint64, blockTimestampDelay uint64) *CallSequenceElement {
	callSequenceElement := &CallSequenceElement{
		Contract:            contract,
		Call:                call,
		BlockNumberDelay:    blockNumberDelay,
		BlockTimestampDelay: blockTimestampDelay,
		ChainReference:      nil,
		ExecutionTrace:      nil,
	}
	return callSequenceElement
}

// Clone creates a copy of the underlying CallSequenceElement.
func (cse *CallSequenceElement) Clone() (*CallSequenceElement, error) {
	// Clone our call
	clonedCall, err := cse.Call.Clone()
	if err != nil {
		return nil, err
	}

	// Clone the element
	clone := &CallSequenceElement{
		Contract:            cse.Contract,
		Call:                clonedCall,
		BlockNumberDelay:    cse.BlockNumberDelay,
		BlockTimestampDelay: cse.BlockTimestampDelay,
		ChainReference:      cse.ChainReference,
		ExecutionTrace:      cse.ExecutionTrace,
	}
	return clone, nil
}

// Method obtains the abi.Method targeted by the CallSequenceElement.Call, or an error if one occurred while obtaining
// it.
func (cse *CallSequenceElement) Method() (*abi.Method, error) {
	// If there is no resolved contract definition, we return no method.
	if cse.Contract == nil {
		return nil, nil
	}

	// If we have a method resolved, return it.
	if cse.Call != nil && cse.Call.DataAbiValues != nil {
		if cse.Call.DataAbiValues.Method != nil {
			return cse.Call.DataAbiValues.Method, nil
		}
	}

	// Try to resolve the method by ID from the call data.
	method, err := cse.Contract.CompiledContract().Abi.MethodById(cse.Call.Data)
	return method, err
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
		methodName = method.Sig
	}

	// Next decode our arguments (we jump four bytes to skip the function selector)
	args, err := method.Inputs.Unpack(cse.Call.Data[4:])
	argsText := "<unable to unpack args>"
	if err == nil {
		argsText, err = valuegeneration.EncodeABIArgumentsToString(method.Inputs, args)
		if err != nil {
			argsText = "<unresolved args>"
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
		cse.Call.GasLimit,
		cse.Call.GasPrice.String(),
		cse.Call.Value.String(),
		cse.Call.From,
	)
}

// AttachExecutionTrace takes a given chain which executed the call sequence element, and a list of contract definitions,
// and it replays the call with an execution tracer attached to it, it then sets CallSequenceElement.ExecutionTrace to
// the resulting trace.
// Returns an error if one occurred.
func (cse *CallSequenceElement) AttachExecutionTrace(chain *chain.TestChain, contractDefinitions fuzzingTypes.Contracts) error {
	// Verify the element has been executed before.
	if cse.ChainReference == nil {
		return fmt.Errorf("failed to resolve execution trace as the chain reference is nil, indicating the call sequence element has never been executed")
	}

	var err error
	// Perform our call with the given trace
	_, cse.ExecutionTrace, err = executiontracer.CallWithExecutionTrace(chain, contractDefinitions, cse.Call.ToCoreMessage(), nil)
	if err != nil {
		return fmt.Errorf("failed to resolve execution trace due to error replaying the call: %v", err)
	}
	return nil
}

// CallSequenceElementChainReference references the inclusion of a CallSequenceElement's underlying call being
// included in a block as a transaction.
type CallSequenceElementChainReference struct {
	// Block describes the block the CallSequenceElement.Call was included in as a transaction. This block may be
	// pending commitment to the chain, or already committed.
	Block *chainTypes.Block

	// TransactionIndex describes the index at which the transaction was included into the Block.
	TransactionIndex int
}

// MessageResults obtains the results of executing the CallSequenceElement.
func (cr *CallSequenceElementChainReference) MessageResults() *chainTypes.MessageResults {
	return cr.Block.MessageResults[cr.TransactionIndex]
}
