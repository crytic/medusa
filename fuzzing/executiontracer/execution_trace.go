package executiontracer

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/compilation/abiutils"
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
	)

	// If this is a contract creation, use a different prefix
	if callFrame.IsContractCreation() {
		action = "creation"
	} else if callFrame.IsProxyCall() {
		action = "proxy call"
	}

	// Resolve our contract names, as well as our method and its name from the code contract.
	if callFrame.ToContractAbi != nil {
		toContractName = callFrame.ToContractName
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
	if inputArgumentsDisplayText == nil {
		temp := fmt.Sprintf("msg_data=%v", hex.EncodeToString(callFrame.InputData))
		inputArgumentsDisplayText = &temp
	}
	if outputArgumentsDisplayText == nil {
		var temp string
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
		// If we executed code, attach additional context such as the contract name, method, etc.
		if callFrame.ExecutedCode {
			currentFrameLine = fmt.Sprintf("%v[%v: %v, %v: %v] %v -> %v.%v(%v)", prefix, action, callFrame.ToAddress.String(), "value", callFrame.CallValue, toContractName, codeContractName, methodName, *inputArgumentsDisplayText)
		} else {
			currentFrameLine = fmt.Sprintf("%v[%v: %v, %v: %v]", prefix, action, callFrame.ToAddress.String(), "value", callFrame.CallValue)
		}
	} else {
		// If we executed code, attach additional context such as the contract name, method, etc.
		if callFrame.ExecutedCode {
			currentFrameLine = fmt.Sprintf("%v[%v: %v, %v: %v] %v.%v(%v)", prefix, action, callFrame.ToAddress.String(), "value", callFrame.CallValue, codeContractName, methodName, *inputArgumentsDisplayText)
		} else {
			currentFrameLine = fmt.Sprintf("%v[%v: %v, %v: %v]", prefix, action, callFrame.ToAddress.String(), "value", callFrame.CallValue)
		}
	}

	// If we did not execute code then just show the address and the ETH value (we do not care about a zero ETH value here and will show it regardless)
	if !callFrame.ExecutedCode {
		currentFrameLine = fmt.Sprintf("%v[%v: %v, %v: %v]", prefix, action, callFrame.ToAddress.String(), "value", callFrame.CallValue)
	}
	outputLines = append(outputLines, currentFrameLine)

	// Now that the header (call) message has been printed, create our indent level to express everything that
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

			// Try to unpack our event data
			event, eventInputValues := abiutils.UnpackEventAndValues(callFrame.CodeContractAbi, eventLog)
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
			outputLines = append(outputLines, fmt.Sprintf("%v[event] %v", prefix, *eventDisplayText))
		}
	}

	// If we self-destructed, add a string for it.
	if callFrame.SelfDestructed {
		outputLines = append(outputLines, fmt.Sprintf("%v[selfdestruct]", prefix))
	}

	// Wrap our return message and output it at the end.
	var exitScopeDisplayText string
	if callFrame.ReturnError == nil {
		exitScopeDisplayText = fmt.Sprintf("%v[return (%v)]", prefix, *outputArgumentsDisplayText)
	} else {
		// Try to extract a panic message out of this, as well as a custom revert reason.
		panicCode := abiutils.GetSolidityPanicCode(callFrame.ReturnError, callFrame.ReturnData, true)
		errorMessage := abiutils.GetSolidityRevertErrorString(callFrame.ReturnError, callFrame.ReturnData)

		// Try to resolve an assertion failed panic code
		if panicCode != nil && panicCode.Uint64() == abiutils.PanicCodeAssertFailed {
			exitScopeDisplayText = fmt.Sprintf("%v[assertion failed]", prefix)
		} else if errorMessage != nil {
			// Try to resolve a generic revert reason
			exitScopeDisplayText = fmt.Sprintf("%v[revert (reason: '%v')]", prefix, *errorMessage)
		} else {
			// Try to unpack a custom Solidity error from the return values.
			matchedCustomError, unpackedCustomErrorArgs := abiutils.GetSolidityCustomRevertError(callFrame.CodeContractAbi, callFrame.ReturnError, callFrame.ReturnData)
			if matchedCustomError != nil {
				customErrorArgsDisplayText, err := valuegeneration.EncodeABIArgumentsToString(matchedCustomError.Inputs, unpackedCustomErrorArgs)
				if err == nil {
					exitScopeDisplayText = fmt.Sprintf("%v[custom error] %v(%v)", prefix, matchedCustomError.Name, customErrorArgsDisplayText)
				}
			}

			// If we could not resolve any custom error, we simply print out the generic VM error message.
			if len(exitScopeDisplayText) == 0 {
				exitScopeDisplayText = fmt.Sprintf("%v[vm error: %s]", prefix, callFrame.ReturnError.Error())
			}
		}
	}
	outputLines = append(outputLines, exitScopeDisplayText)

	// Return our output lines
	return outputLines
}

// String returns a string representation of the execution trace.
func (t *ExecutionTrace) String() string {
	outputLines := t.generateStringsForCallFrame(0, t.TopLevelCallFrame)
	return strings.Join(outputLines, "\n")
}
