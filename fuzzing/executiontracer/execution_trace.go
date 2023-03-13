package executiontracer

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/chain/types"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"strings"
)

// ExecutionTrace contains information recorded by an ExecutionTracer. It contains information about each call
// scope entered and exited, and their associated contract definitions.
type ExecutionTrace struct {
	// TopLevelCallFrame refers to the root call frame, the first EVM call scope entered when an externally-owned
	// address calls upon a contract.
	TopLevelCallFrame *CallFrame
}

// newExecutionTrace creates and returns a new ExecutionTrace, to be used by the ExecutionTracer.
func newExecutionTrace() *ExecutionTrace {
	return &ExecutionTrace{
		TopLevelCallFrame: nil,
	}
}

// unpackEventValues unpacks the input values of events from an event log. It does so by unpacking indexed arguments
// from the event log topics, while unpacking un-indexed arguments from the event log data. It then merges the
// unpacked values, so they retain their original order.
// Returns the unpacked event input argument values, or an error if one occurred.
func (t *ExecutionTrace) unpackEventValues(event *abi.Event, eventLog *coreTypes.Log) ([]any, error) {
	// First, split our indexed and non-indexed arguments.
	var (
		unindexedInputArguments abi.Arguments
		indexedInputArguments   abi.Arguments
	)
	for _, arg := range event.Inputs {
		if arg.Indexed {
			// We have to re-create indexed items, as go-ethereum's ABI API does not typically support events.
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
		return nil, err
	}

	// Unpack our indexed values.
	indexedInputValues, err := indexedInputArguments.Unpack(indexedInputData)
	if err != nil {
		return nil, err
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

	return inputValues, nil
}

// generateStringsForCallFrame generates indented strings for a given call frame and its children.
// Returns the list of strings, to be joined by new line separators.
func (t *ExecutionTrace) generateStringsForCallFrame(currentDepth int, callFrame *CallFrame) []string {
	// Create our resulting strings array
	var outputLines []string

	// Create our current call line prefix (indented by call depth)
	prefix := strings.Repeat("\t", currentDepth) + " -> "

	if currentDepth == 0 {
		outputLines = append(outputLines, prefix+"[Execution Trace]")
	}

	// Define some strings that represent our current call frame
	var (
		action           = "call"
		toContractName   = "<unresolved contract>"
		codeContractName = "<unresolved contract>"
		methodName       = "<unresolved method>"
		method           *abi.Method
		err              error
		temp             string
	)

	// If this is a contract creation, use a different prefix
	if callFrame.IsContractCreation() {
		action = "creation"
	} else if callFrame.IsProxyCall() {
		action = "proxy call"
	}

	// Resolve our contract names, as well as our method and its name from the code contract.
	if callFrame.ToContract != nil {
		toContractName = callFrame.ToContract.Name()
	}
	if callFrame.CodeContract != nil {
		codeContractName = callFrame.CodeContract.Name()
		if callFrame.IsContractCreation() {
			methodName = "constructor"
			method = &callFrame.CodeContract.CompiledContract().Abi.Constructor
		} else {
			method, err = callFrame.CodeContract.CompiledContract().Abi.MethodById(callFrame.InputData)
			if err == nil {
				methodName = method.Name
			}
		}
	}

	// Next we attempt to obtain a display string for the input and output arguments.
	var (
		inputArgumentsDisplayText  *string
		outputArgumentsDisplayText *string
	)
	if method != nil {
		// Unpack our input values and obtain a string to represent them
		if len(callFrame.InputData) >= 4 {
			inputValues, err := method.Inputs.Unpack(callFrame.InputData[4:])
			if err == nil {
				encodedInputString, err := valuegeneration.EncodeABIArgumentsToString(method.Inputs, inputValues)
				if err == nil {
					inputArgumentsDisplayText = &encodedInputString
				}
			}
		}

		// Unpack our output values and obtain a string to represent them, only if we didn't encounter an error.
		if callFrame.ReturnError == nil && len(callFrame.ReturnData) >= 4 {
			outputValues, err := method.Outputs.Unpack(callFrame.ReturnData[4:])
			if err == nil {
				encodedOutputString, err := valuegeneration.EncodeABIArgumentsToString(method.Outputs, outputValues)
				if err == nil {
					outputArgumentsDisplayText = &encodedOutputString
				}
			}
		}
	}

	// If we could not correctly obtain the unpacked arguments in a nice display string (due to not having a resolved
	// contract or method definition, or failure to unpack), we display as raw data in the worst case.
	if inputArgumentsDisplayText == nil {
		temp = fmt.Sprintf("msg_data=%v", hex.EncodeToString(callFrame.InputData))
		inputArgumentsDisplayText = &temp
	}
	if outputArgumentsDisplayText == nil {
		if len(callFrame.ReturnData) > 0 {
			temp = fmt.Sprintf("return_data=%v", hex.EncodeToString(callFrame.ReturnData))
		} else {
			// If there was no return data, we display nothing for output arguments.
			temp = ""
		}
		outputArgumentsDisplayText = &temp
	}

	// Generate the message we wish to output finally, using all these display string components.
	var currentFrameLine string
	if callFrame.IsProxyCall() {
		currentFrameLine = fmt.Sprintf("%v[%v: %v] %v -> %v.%v(%v)", prefix, action, callFrame.ToAddress.String(), toContractName, codeContractName, methodName, *inputArgumentsDisplayText)
	} else {
		currentFrameLine = fmt.Sprintf("%v[%v: %v] %v.%v(%v)", prefix, action, callFrame.ToAddress.String(), codeContractName, methodName, *inputArgumentsDisplayText)
	}
	outputLines = append(outputLines, currentFrameLine)

	// Now that the header (call) message has been printed, increate our indent level to express everything that
	// happened under it.
	prefix = "\t" + prefix

	// Loop for each operation performed in the call frame, to provide a chronological history of operations in the
	// frame.
	for _, operation := range callFrame.Operations {
		// If this is a call frame being entered, generate information recursively.
		if childCallFrame, ok := operation.(*CallFrame); ok {
			childOutputLines := t.generateStringsForCallFrame(currentDepth+1, childCallFrame)
			outputLines = append(outputLines, childOutputLines...)
		} else if eventLog, ok := operation.(*coreTypes.Log); ok {
			// If this is an event log, match it in our contract's ABI.
			var (
				eventDisplayText *string
			)
			if callFrame.CodeContract != nil {
				event, err := callFrame.CodeContract.CompiledContract().Abi.EventByID(eventLog.Topics[0])
				if err == nil {
					// Next, unpack our values
					eventInputValues, err := t.unpackEventValues(event, eventLog)
					if err == nil {
						// Format the values as a comma-separated string
						encodedEventValuesString, err := valuegeneration.EncodeABIArgumentsToString(event.Inputs, eventInputValues)
						if err == nil {
							// Format our event display text finally, with the event name.
							temp = fmt.Sprintf("%v(%v)", event.Name, encodedEventValuesString)
							eventDisplayText = &temp
						}
					}
				}
			}

			// If we could not resolve the event, print the raw event data
			if eventDisplayText == nil {
				var topicsStrings []string
				for _, topic := range eventLog.Topics {
					topicsStrings = append(topicsStrings, hex.EncodeToString(topic.Bytes()))
				}

				temp = fmt.Sprintf("<unresolved(topics=[%v], data=%v)>", strings.Join(topicsStrings, ", "), hex.EncodeToString(eventLog.Data))
				eventDisplayText = &temp
			}

			// Finally, add our output line with this event data to it.
			outputLines = append(outputLines, fmt.Sprintf("%v[event] %v", prefix, *eventDisplayText))
		}
	}

	// If we self-destructed, add a string for it.
	if callFrame.SelfDestructed {
		outputLines = append(outputLines, fmt.Sprintf("%v[selfdestruct]", prefix))
	}

	// Wrap our return message and output it at the end.
	if callFrame.ReturnError == nil {
		*outputArgumentsDisplayText = fmt.Sprintf("%v[return (%v)]", prefix, *outputArgumentsDisplayText)
	} else {
		// Try to extract a panic message out of this
		panicCode := types.GetSolidityPanicCode(callFrame.ReturnError, callFrame.ReturnData, true)
		errorMessage := types.GetSolidityRevertErrorString(callFrame.ReturnError, callFrame.ReturnData)
		if panicCode != nil && panicCode.Uint64() == types.PanicCodeAssertFailed {
			*outputArgumentsDisplayText = fmt.Sprintf("%v[assertion failed]", prefix)
		} else if errorMessage != nil {
			*outputArgumentsDisplayText = fmt.Sprintf("%v[revert (reason: '%v')]", prefix, *errorMessage)
		}
	}
	outputLines = append(outputLines, *outputArgumentsDisplayText)

	// Return our output lines
	return outputLines
}

// String returns a string representation of the execution trace.
func (t *ExecutionTrace) String() string {
	outputLines := t.generateStringsForCallFrame(0, t.TopLevelCallFrame)
	return strings.Join(outputLines, "\n")
}
