package executiontracer

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
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
		inputValues, err := method.Inputs.Unpack(callFrame.InputData)
		if err == nil {
			encodedInputString, err := valuegeneration.EncodeABIArgumentsToString(method.Inputs, inputValues)
			if err == nil {
				inputArgumentsDisplayText = &encodedInputString
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
		temp = fmt.Sprintf("msg_data='%v'", hex.EncodeToString(callFrame.InputData))
		inputArgumentsDisplayText = &temp
	}
	if outputArgumentsDisplayText == nil {
		temp = fmt.Sprintf("return_data='%v'", hex.EncodeToString(callFrame.ReturnData))
		outputArgumentsDisplayText = &temp
	}

	// Generate the message we wish to output finally, using all these display string components.
	var currentFrameLine string
	if callFrame.IsProxyCall() {
		currentFrameLine = fmt.Sprintf("%v[%v: %v] %v->%v.%v(%v)", prefix, action, callFrame.ToAddress.String(), toContractName, codeContractName, methodName, *inputArgumentsDisplayText)
	} else {
		currentFrameLine = fmt.Sprintf("%v[%v: %v] %v.%v(%v)", prefix, action, callFrame.ToAddress.String(), codeContractName, methodName, *inputArgumentsDisplayText)
	}
	outputLines = append(outputLines, currentFrameLine)

	// Loop for each call frame under this call frame and print that as well.
	for _, childCallFrame := range callFrame.ChildCallFrames {
		childOutputLines := t.generateStringsForCallFrame(currentDepth+1, childCallFrame)
		outputLines = append(outputLines, childOutputLines...)
	}

	// Wrap our return message and output it at the end.
	if callFrame.ReturnError == nil {
		*outputArgumentsDisplayText = fmt.Sprintf("%v[return (%v)]", prefix, *outputArgumentsDisplayText)
	} else {
		*outputArgumentsDisplayText = fmt.Sprintf("%v[error (%v)]", prefix, *outputArgumentsDisplayText)
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
