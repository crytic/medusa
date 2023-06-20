package executiontracer

import (
	"encoding/hex"
	"fmt"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
	"github.com/ethereum/go-ethereum/accounts/abi"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"strings"
)

// ExecutionTrace contains information recorded by an ExecutionTracer. It contains information about each call
// scope entered and exited, and their associated contract definitions.
type ExecutionTrace struct {
	// TopLevelCallFrame refers to the root call frame, the first EVM call scope entered when an externally-owned
	// address calls upon a contract.
	TopLevelCallFrame *CallFrame

	// contractDefinitions represents the known contract definitions at the time of tracing. This is used to help
	// obtain any additional information regarding execution.
	contractDefinitions contracts.Contracts
}

// newExecutionTrace creates and returns a new ExecutionTrace, to be used by the ExecutionTracer.
func newExecutionTrace(contracts contracts.Contracts) *ExecutionTrace {
	return &ExecutionTrace{
		TopLevelCallFrame:   nil,
		contractDefinitions: contracts,
	}
}

// generateCallFrameEnterString generates a header string to print for the given call frame. It contains
// information about the invoked call.
// Returns the header string
func (t *ExecutionTrace) generateCallFrameEnterString(callFrame *CallFrame) string {
	// Define some strings that represent our current call frame
	var (
		callType          = "call"
		proxyContractName = "<unresolved proxy>"
		codeContractName  = "<unresolved contract>"
		methodName        = "<unresolved method>"
		method            *abi.Method
		err               error
	)

	// If this is a contract creation, use a different prefix
	if callFrame.IsContractCreation() {
		callType = "creation"
	} else if callFrame.IsProxyCall() {
		callType = "proxy call"
	}

	// Resolve our contract names, as well as our method and its name from the code contract.
	if callFrame.ToContractAbi != nil {
		proxyContractName = callFrame.ToContractName
	}
	if callFrame.CodeContractAbi != nil {
		codeContractName = callFrame.CodeContractName
		if callFrame.IsContractCreation() {
			methodName = "constructor"
			method = &callFrame.CodeContractAbi.Constructor
		} else {
			method, err = callFrame.CodeContractAbi.MethodById(callFrame.InputData)
			if err == nil {
				methodName = method.Name
			}
		}
	}

	// Next we attempt to obtain a display string for the input and output arguments.
	var inputArgumentsDisplayText *string
	if method != nil {
		// Determine what buffer will hold our ABI data.
		// - If this a contract deployment, constructor argument data follows code, so we use a different buffer the
		//   tracer provides.
		// - If this is a normal call, the input data for the call is used, with the 32-bit function selector sliced off.
		abiDataInputBuffer := make([]byte, 0)
		if callFrame.IsContractCreation() {
			abiDataInputBuffer = callFrame.ConstructorArgsData
		} else if len(callFrame.InputData) >= 4 {
			abiDataInputBuffer = callFrame.InputData[4:]
		}

		// Unpack our input values and obtain a string to represent them
		inputValues, err := method.Inputs.Unpack(abiDataInputBuffer)
		if err == nil {
			encodedInputString, err := valuegeneration.EncodeABIArgumentsToString(method.Inputs, inputValues)
			if err == nil {
				inputArgumentsDisplayText = &encodedInputString
			}
		}
	}

	// If we could not correctly obtain the unpacked arguments in a nice display string (due to not having a resolved
	// contract or method definition, or failure to unpack), we display as raw data in the worst case.
	if inputArgumentsDisplayText == nil {
		temp := fmt.Sprintf("msg_data=%v", hex.EncodeToString(callFrame.InputData))
		inputArgumentsDisplayText = &temp
	}

	// Generate the message we wish to output finally, using all these display string components.
	// If we executed code, attach additional context such as the contract name, method, etc.
	if callFrame.IsProxyCall() {
		if callFrame.ExecutedCode {
			return fmt.Sprintf("[%v] %v -> %v.%v(%v) (addr=%v, code=%v, value=%v, sender=%v)", callType, proxyContractName, codeContractName, methodName, *inputArgumentsDisplayText, callFrame.ToAddress.String(), callFrame.CodeAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		} else {
			return fmt.Sprintf("[%v] (addr=%v, value=%v, sender=%v)", callType, callFrame.ToAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		}
	} else {
		if callFrame.ExecutedCode {
			return fmt.Sprintf("[%v] %v.%v(%v) (addr=%v, value=%v, sender=%v)", callType, codeContractName, methodName, *inputArgumentsDisplayText, callFrame.ToAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		} else {
			return fmt.Sprintf("[%v] (addr=%v, value=%v, sender=%v)", callType, callFrame.ToAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		}
	}
}

// generateCallFrameExitString generates a footer string to print for the given call frame. It contains
// result information about the call.
// Returns the footer string.
func (t *ExecutionTrace) generateCallFrameExitString(callFrame *CallFrame) string {
	// Define some strings that represent our current call frame
	var method *abi.Method

	// Resolve our method definition
	if callFrame.CodeContractAbi != nil {
		if callFrame.IsContractCreation() {
			method = &callFrame.CodeContractAbi.Constructor
		} else {
			method, _ = callFrame.CodeContractAbi.MethodById(callFrame.InputData)
		}
	}

	// Next we attempt to obtain a display string for the input and output arguments.
	var outputArgumentsDisplayText *string
	if method != nil {
		// Unpack our output values and obtain a string to represent them, only if we didn't encounter an error.
		if callFrame.ReturnError == nil {
			outputValues, err := method.Outputs.Unpack(callFrame.ReturnData)
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
	if outputArgumentsDisplayText == nil {
		temp := fmt.Sprintf("return_data=%v", hex.EncodeToString(callFrame.ReturnData))
		outputArgumentsDisplayText = &temp
	}

	// Wrap our return message and output it at the end.
	if callFrame.ReturnError == nil {
		return fmt.Sprintf("[return (%v)]", *outputArgumentsDisplayText)
	}

	// Try to resolve a panic message and check if it signals a failed assertion.
	panicCode := abiutils.GetSolidityPanicCode(callFrame.ReturnError, callFrame.ReturnData, true)
	if panicCode != nil && panicCode.Uint64() == abiutils.PanicCodeAssertFailed {
		return "[assertion failed]"
	}

	// Try to resolve an assertion failed panic code.
	errorMessage := abiutils.GetSolidityRevertErrorString(callFrame.ReturnError, callFrame.ReturnData)
	if errorMessage != nil {
		return fmt.Sprintf("[revert ('%v')]", *errorMessage)
	}

	// Try to unpack a custom Solidity error from the return values.
	matchedCustomError, unpackedCustomErrorArgs := abiutils.GetSolidityCustomRevertError(callFrame.CodeContractAbi, callFrame.ReturnError, callFrame.ReturnData)
	if matchedCustomError != nil {
		customErrorArgsDisplayText, err := valuegeneration.EncodeABIArgumentsToString(matchedCustomError.Inputs, unpackedCustomErrorArgs)
		if err == nil {
			return fmt.Sprintf("[revert (error: %v(%v))]", matchedCustomError.Name, customErrorArgsDisplayText)
		}
	}

	// Check if this is a generic revert.
	if callFrame.ReturnError == vm.ErrExecutionReverted {
		return "[revert]"
	}

	// If we could not resolve any custom error, we simply print out the generic VM error message.
	return fmt.Sprintf("[vm error ('%v')]", callFrame.ReturnError.Error())
}

// generateEventEmittedString generates a string used to express an event emission. It contains information about an
// event log.
// Returns a string representing an event emission.
func (t *ExecutionTrace) generateEventEmittedString(callFrame *CallFrame, eventLog *coreTypes.Log) string {
	// If this is an event log, match it in our contract's ABI.
	var eventDisplayText *string

	// Try to unpack our event data
	event, eventInputValues := abiutils.UnpackEventAndValues(callFrame.CodeContractAbi, eventLog)
	if event == nil {
		// If we couldn't resolve the event from our immediate contract ABI, it may come from a library.
		// TODO: Temporarily, we fix this by trying to resolve the event from any contracts definition. A future
		//  fix should include only checking relevant libraries associated with the contract.
		for _, contract := range t.contractDefinitions {
			event, eventInputValues = abiutils.UnpackEventAndValues(&contract.CompiledContract().Abi, eventLog)
			if event != nil {
				break
			}
		}
	}

	// If we resolved an event definition and unpacked data.
	if event != nil {
		// Format the values as a comma-separated string
		encodedEventValuesString, err := valuegeneration.EncodeABIArgumentsToString(event.Inputs, eventInputValues)
		if err == nil {
			// Format our event display text finally, with the event name.
			temp := fmt.Sprintf("%v(%v)", event.Name, encodedEventValuesString)
			eventDisplayText = &temp
		}
	}

	// If we could not resolve the event, print the raw event data
	if eventDisplayText == nil {
		var topicsStrings []string
		for _, topic := range eventLog.Topics {
			topicsStrings = append(topicsStrings, hex.EncodeToString(topic.Bytes()))
		}

		temp := fmt.Sprintf("<unresolved(topics=[%v], data=%v)>", strings.Join(topicsStrings, ", "), hex.EncodeToString(eventLog.Data))
		eventDisplayText = &temp
	}

	// Finally, add our output line with this event data to it.
	return fmt.Sprintf("[event] %v", *eventDisplayText)
}

// generateStringsForCallFrame generates indented strings for a given call frame and its children.
// Returns the list of strings, to be joined by new line separators.
func (t *ExecutionTrace) generateStringsForCallFrame(currentDepth int, callFrame *CallFrame) []string {
	// Create our resulting strings array
	var outputLines []string

	// Create our current call line prefix (indented by call depth)
	prefix := strings.Repeat("\t", currentDepth) + " -> "

	// If we're printing the root frame, add the overall execution trace header.
	if currentDepth == 0 {
		outputLines = append(outputLines, prefix+"[Execution Trace]")
	}

	// Add the call frame enter header
	header := prefix + t.generateCallFrameEnterString(callFrame)
	outputLines = append(outputLines, header)

	// Now that the header has been printed, create our indent level to express everything that
	// happened under it.
	prefix = "\t" + prefix

	// If we executed some code underneath this frame, we'll output additional information. If we did not,
	// we shorten our trace by skipping over blank call scope returns, etc.
	if callFrame.ExecutedCode {
		// Loop for each operation performed in the call frame, to provide a chronological history of operations in the
		// frame.
		for _, operation := range callFrame.Operations {
			if childCallFrame, ok := operation.(*CallFrame); ok {
				// If this is a call frame being entered, generate information recursively.
				childOutputLines := t.generateStringsForCallFrame(currentDepth+1, childCallFrame)
				outputLines = append(outputLines, childOutputLines...)
			} else if eventLog, ok := operation.(*coreTypes.Log); ok {
				// If an event log was emitted, add a message for it.
				eventMessage := prefix + t.generateEventEmittedString(callFrame, eventLog)
				outputLines = append(outputLines, eventMessage)
			}
		}

		// If we self-destructed, add a message for it before our footer.
		if callFrame.SelfDestructed {
			outputLines = append(outputLines, fmt.Sprintf("%v[selfdestruct]", prefix))
		}

		// Add the call frame exit footer
		footer := prefix + t.generateCallFrameExitString(callFrame)
		outputLines = append(outputLines, footer)
	}

	// Return our output lines
	return outputLines
}

// String returns a string representation of the execution trace.
func (t *ExecutionTrace) String() string {
	outputLines := t.generateStringsForCallFrame(0, t.TopLevelCallFrame)
	return strings.Join(outputLines, "\n")
}
