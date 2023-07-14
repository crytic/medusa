package executiontracer

import (
	"encoding/hex"
	"fmt"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/logging/colors"
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

// generateCallFrameEnterElements generates a list of elements describing top level information about this call frame.
// This list of elements will hold information about what kind of call it is, wei sent, what method is called, and more.
// Additionally, the list may also hold formatting options for console output.
func (t *ExecutionTrace) generateCallFrameEnterElements(callFrame *CallFrame) []any {
	// Create list of elements
	elements := make([]any, 0)

	// Define some strings and objects that represent our current call frame
	var (
		callType          = []any{colors.BlueBold, "[call] ", colors.Reset}
		proxyContractName = "<unresolved proxy>"
		codeContractName  = "<unresolved contract>"
		methodName        = "<unresolved method>"
		method            *abi.Method
		err               error
	)

	// If this is a contract creation or proxy call, use different formatting for call type
	if callFrame.IsContractCreation() {
		callType = []any{colors.YellowBold, "[creation] ", colors.Reset}
	} else if callFrame.IsProxyCall() {
		callType = []any{colors.CyanBold, "[proxy call] ", colors.Reset}
	}

	// Append the formatted call type information to the list of elements
	elements = append(elements, callType...)

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
	var callInfo string
	if callFrame.IsProxyCall() {
		if callFrame.ExecutedCode {
			callInfo = fmt.Sprintf("%v -> %v.%v(%v) (addr=%v, code=%v, value=%v, sender=%v)", proxyContractName, codeContractName, methodName, *inputArgumentsDisplayText, callFrame.ToAddress.String(), callFrame.CodeAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		} else {
			callInfo = fmt.Sprintf("(addr=%v, value=%v, sender=%v)", callFrame.ToAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		}
	} else {
		if callFrame.ExecutedCode {
			callInfo = fmt.Sprintf("%v.%v(%v) (addr=%v, value=%v, sender=%v)", codeContractName, methodName, *inputArgumentsDisplayText, callFrame.ToAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		} else {
			callInfo = fmt.Sprintf("(addr=%v, value=%v, sender=%v)", callFrame.ToAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		}
	}

	// Add call information to the elements
	elements = append(elements, callInfo, "\n")

	return elements
}

// generateCallFrameExitElements generates a list of elements describing the return data of the call frame (e.g.
// traditional return data, assertion failure, revert data, etc.). Additionally, the list may also hold formatting options for console output.
func (t *ExecutionTrace) generateCallFrameExitElements(callFrame *CallFrame) []any {
	// Create list of elements
	elements := make([]any, 0)

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
		elements = append(elements, colors.GreenBold, fmt.Sprintf("[return (%v)]", *outputArgumentsDisplayText), colors.Reset, "\n")
		return elements
	}

	// Try to resolve a panic message and check if it signals a failed assertion.
	panicCode := abiutils.GetSolidityPanicCode(callFrame.ReturnError, callFrame.ReturnData, true)
	if panicCode != nil {
		elements = append(elements, colors.RedBold, fmt.Sprintf("[%v]", abiutils.GetPanicReason(panicCode.Uint64())), colors.Reset, "\n")
		return elements
	}

	// Try to resolve an assertion failed panic code.
	errorMessage := abiutils.GetSolidityRevertErrorString(callFrame.ReturnError, callFrame.ReturnData)
	if errorMessage != nil {
		elements = append(elements, colors.RedBold, fmt.Sprintf("[revert ('%v')]", *errorMessage), colors.Reset, "\n")
		return elements
	}

	// Try to unpack a custom Solidity error from the return values.
	matchedCustomError, unpackedCustomErrorArgs := abiutils.GetSolidityCustomRevertError(callFrame.CodeContractAbi, callFrame.ReturnError, callFrame.ReturnData)
	if matchedCustomError != nil {
		customErrorArgsDisplayText, err := valuegeneration.EncodeABIArgumentsToString(matchedCustomError.Inputs, unpackedCustomErrorArgs)
		if err == nil {
			elements = append(elements, colors.RedBold, fmt.Sprintf("[revert (error: %v(%v))]", matchedCustomError.Name, customErrorArgsDisplayText), colors.Reset, "\n")
			return elements
		}
	}

	// Check if this is a generic revert.
	if callFrame.ReturnError == vm.ErrExecutionReverted {
		elements = append(elements, colors.RedBold, "[revert]", colors.Reset, "\n")
		return elements
	}

	// If we could not resolve any custom error, we simply print out the generic VM error message.
	elements = append(elements, colors.RedBold, fmt.Sprintf("[vm error ('%v')]", callFrame.ReturnError.Error()), colors.Reset, "\n")
	return elements
}

// generateEventEmittedElements generates a list of elements used to express an event emission. It contains information about an
// event log such as the topics and the event data. Additionally, the list may also hold formatting options for console output.
func (t *ExecutionTrace) generateEventEmittedElements(callFrame *CallFrame, eventLog *coreTypes.Log) []any {
	// Create list of elements
	elements := make([]any, 0)

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
	elements = append(elements, colors.MagentaBold, "[event] ", colors.Reset, *eventDisplayText, "\n")
	return elements
}

// generateElementsForCallFrame generates a list of elements for a given call frame and its children. Additionally,
// the list may also hold formatting options for console output.
func (t *ExecutionTrace) generateElementsForCallFrame(currentDepth int, callFrame *CallFrame) []any {
	// Create list of elements
	elements := make([]any, 0)

	// Create our current call line prefix (indented by call depth)
	prefix := strings.Repeat("\t", currentDepth) + " => "

	// If we're printing the root frame, add the overall execution trace header.
	if currentDepth == 0 {
		elements = append(elements, colors.Bold, "[Execution Trace]", colors.Reset, "\n")
	}

	// Add the call frame enter header elements
	elements = append(elements, prefix)
	elements = append(elements, t.generateCallFrameEnterElements(callFrame)...)

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
				childOutputLines := t.generateElementsForCallFrame(currentDepth+1, childCallFrame)
				elements = append(elements, childOutputLines...)
			} else if eventLog, ok := operation.(*coreTypes.Log); ok {
				// If an event log was emitted, add a message for it.
				elements = append(elements, prefix)
				elements = append(elements, t.generateEventEmittedElements(callFrame, eventLog)...)
			}
		}

		// If we self-destructed, add a message for it before our footer.
		if callFrame.SelfDestructed {
			elements = append(elements, prefix, colors.MagentaBold, "[selfdestruct]", colors.Reset, "\n")
		}

		// Add the call frame exit footer
		elements = append(elements, prefix)
		elements = append(elements, t.generateCallFrameExitElements(callFrame)...)
	}

	// Return our elements
	return elements
}

// Log returns a logging.LogBuffer that represents this execution trace. This buffer will be passed to the underlying
// logger which will format it accordingly for console or file.
func (t *ExecutionTrace) Log() *logging.LogBuffer {
	buffer := logging.NewLogBuffer()
	buffer.Append(t.generateElementsForCallFrame(0, t.TopLevelCallFrame)...)
	return buffer
}

// String returns the string representation of this execution trace
func (t *ExecutionTrace) String() string {
	// Internally, we just call the log function, get the list of elements and create their non-colorized string representation
	// Might be useful for 3rd party apps
	return t.Log().String()
}
