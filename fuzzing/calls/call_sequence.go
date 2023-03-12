package calls

import (
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trailofbits/medusa/utils"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	fuzzingTypes "github.com/trailofbits/medusa/fuzzing/contracts"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
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

// Clone creates a copy of the underlying CallSequence.
func (cs CallSequence) Clone() (CallSequence, error) {
	var err error
	r := make(CallSequence, len(cs))
	for i := 0; i < len(r); i++ {
		r[i], err = cs[i].Clone()
		if err != nil {
			return nil, nil
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

		// Hash the call message
		_, err = hashProvider.Write(utils.MessageToTransaction(cse.Call).Hash().Bytes())
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
}

// NewCallSequenceElement returns a new CallSequenceElement struct to track a single call made within a CallSequence.
func NewCallSequenceElement(contract *fuzzingTypes.Contract, call *CallMessage, blockNumberDelay uint64, blockTimestampDelay uint64) *CallSequenceElement {
	callSequenceElement := &CallSequenceElement{
		Contract:            contract,
		Call:                call,
		BlockNumberDelay:    blockNumberDelay,
		BlockTimestampDelay: blockTimestampDelay,
		ChainReference:      nil,
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
	Block *chainTypes.Block

	// TransactionIndex describes the index at which the transaction was included into the Block.
	TransactionIndex int
}

// MessageResults obtains the results of executing the CallSequenceElement.
func (cr *CallSequenceElementChainReference) MessageResults() *chainTypes.MessageResults {
	return cr.Block.MessageResults[cr.TransactionIndex]
}
