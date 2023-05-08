package abiutils

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
)

// UnpackEventAndValues takes a given contract ABI, and an emitted event log from VM, and attempts to find an
// event definition for the log, and unpack its input values.
// Returns the event definition and unpacked event input values, or nil for both if an event definition could not
// be resolved, or values could not be unpacked.
func UnpackEventAndValues(contractAbi *abi.ABI, eventLog *coreTypes.Log) (*abi.Event, []any) {
	// If no ABI was given, no event data can be extracted.
	if contractAbi == nil {
		return nil, nil
	}

	// Obtain an event definition matching this event log's first topic.
	event, err := contractAbi.EventByID(eventLog.Topics[0])
	if err != nil {
		return nil, nil
	}

	// So now we can begin to unpack. For context, what we will be doing below is because go-ethereum's ABI API does
	// not support unpacking events (or indexed items). So we simply split indexed/un-indexed items, create new
	// argument definitions for indexed items to seem un-indexed, and unpack them from log topics (indexed) or
	// data (un-indexed), accordingly. Then we merge the results back to be in the original order of argument
	// definitions and return them, so unpacking here appears consistent with abi.Method unpacking.

	// First, split our indexed and non-indexed arguments.
	var (
		unindexedInputArguments abi.Arguments
		indexedInputArguments   abi.Arguments
	)
	for _, arg := range event.Inputs {
		if arg.Indexed {
			// We have to re-create indexed items, as go-ethereum's ABI API does not typically support indexed data.
			// TODO: See if we can upstream something to go-ethereum here before replacing the ABI API in the future.
			indexedInputArguments = append(indexedInputArguments, abi.Argument{
				Name:    arg.Name,
				Type:    arg.Type,
				Indexed: false,
			})
		} else {
			unindexedInputArguments = append(unindexedInputArguments, arg)
		}
	}

	// Next, aggregate all topics into a single buffer, so we can treat it like data to unpack from.
	var indexedInputData []byte
	for i := range indexedInputArguments {
		indexedInputData = append(indexedInputData, eventLog.Topics[i+1].Bytes()...)
	}

	// Unpacked our un-indexed values.
	unindexedInputValues, err := unindexedInputArguments.Unpack(eventLog.Data)
	if err != nil {
		return nil, nil
	}

	// Unpack our indexed values.
	indexedInputValues, err := indexedInputArguments.Unpack(indexedInputData)
	if err != nil {
		return nil, nil
	}

	// Now merge our indexed and non-indexed values according to the original order we had for event input arguments.
	var (
		currentIndexed   int
		currentUnindexed int
		inputValues      []any
	)
	for _, arg := range event.Inputs {
		if arg.Indexed {
			inputValues = append(inputValues, indexedInputValues[currentIndexed])
			currentIndexed++
		} else {
			inputValues = append(inputValues, unindexedInputValues[currentUnindexed])
			currentUnindexed++
		}
	}

	// Return our definition and data
	return event, inputValues
}
