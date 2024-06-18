package valuegenerationtracer

import (
	"encoding/hex"
	"fmt"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
	"github.com/crytic/medusa/logging/colors"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/slices"
	"math/big"
	"regexp"
)

// valueGenerationTracerResultsKey describes the key to use when storing tracer results in call message results, or when
// querying them.
const valueGenerationTracerResultsKey = "ValueGenerationTracerResults"

type ValueGenerationTrace struct {
	TopLevelCallFrame *utils.CallFrame

	contractDefinitions contracts.Contracts
}

// TODO: Sanan
type ValueGenerationTracer struct {
	// emittedValue describes emitted event values during the execution of the contract.
	emittedValues []any

	// functionReturnValues indicates the return value of executed functions in one sequence.
	functionReturnValues []any

	// evm refers to the EVM instance last captured.
	evm *vm.EVM

	// trace represents the current execution trace captured by this tracer.
	trace *ValueGenerationTrace

	// currentCallFrame references the current call frame being traced.
	currentCallFrame *utils.CallFrame

	// contractDefinitions represents the contract definitions to match for execution traces.
	contractDefinitions contracts.Contracts

	// cheatCodeContracts  represents the cheat code contract definitions to match for execution traces.
	cheatCodeContracts map[common.Address]*chain.CheatCodeContract

	// onNextCaptureState refers to methods which should be executed the next time CaptureState executes.
	// CaptureState is called prior to execution of an instruction. This allows actions to be performed
	// after some state is captured, on the next state capture (e.g. detecting a log instruction, but
	// using this structure to execute code later once the log is committed).
	onNextCaptureState []func()
}

func (v *ValueGenerationTracer) CaptureTxStart(gasLimit uint64) {
	// Sanan: start fresh
	//v.callDepth = 0
	v.trace = newValueGenerationTrace(v.contractDefinitions)
	v.currentCallFrame = nil
	v.onNextCaptureState = nil
	v.emittedValues = make([]any, 0)
	v.functionReturnValues = make([]any, 0)
}

func (v *ValueGenerationTracer) CaptureTxEnd(restGas uint64) {
	//TODO implement me
	//panic("implement me")
}

func (v *ValueGenerationTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	//TODO implement me
	fmt.Println("Called CaptureStart")
	v.evm = env
	v.captureEnteredCallFrame(from, to, input, create, value)
	//panic("implement me")
}

func (v *ValueGenerationTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	//TODO implement me
	//panic("implement me")
	v.captureExitedCallFrame(output, err)
}

func (v *ValueGenerationTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	//TODO implement me
	v.captureEnteredCallFrame(from, to, input, typ == vm.CREATE || typ == vm.CREATE2, value)
}

func (v *ValueGenerationTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	//TODO implement me
	//panic("implement me")
	v.captureExitedCallFrame(output, err)
}

func newValueGenerationTrace(contracts contracts.Contracts) *ValueGenerationTrace {
	return &ValueGenerationTrace{
		TopLevelCallFrame:   nil,
		contractDefinitions: contracts,
	}
}

// captureEnteredCallFrame is a helper method used when a new call frame is entered to record information about it.
func (v *ValueGenerationTracer) captureEnteredCallFrame(fromAddress common.Address, toAddress common.Address, inputData []byte, isContractCreation bool, value *big.Int) {
	// Create our call frame struct to track data for this call frame we entered.
	fmt.Println("Entered captureEnteredCallFrame")
	callFrameData := &utils.CallFrame{
		SenderAddress:       fromAddress,
		ToAddress:           toAddress,
		ToContractName:      "",
		ToContractAbi:       nil,
		ToInitBytecode:      nil,
		ToRuntimeBytecode:   nil,
		CodeAddress:         toAddress, // Note: Set temporarily, overwritten if code executes (in CaptureState).
		CodeContractName:    "",
		CodeContractAbi:     nil,
		CodeRuntimeBytecode: nil,
		Operations:          make([]any, 0),
		SelfDestructed:      false,
		InputData:           slices.Clone(inputData),
		ConstructorArgsData: nil,
		ReturnData:          nil,
		ExecutedCode:        false,
		CallValue:           value,
		ReturnError:         nil,
		ParentCallFrame:     v.currentCallFrame,
	}

	// If this is a contract creation, set the init bytecode for this call frame to the input data.
	if isContractCreation {
		callFrameData.ToInitBytecode = inputData
	}

	// Set our current call frame in our trace
	if v.trace.TopLevelCallFrame == nil {
		v.trace.TopLevelCallFrame = callFrameData
	} else {
		v.currentCallFrame.Operations = append(v.currentCallFrame.Operations, callFrameData)
	}
	v.currentCallFrame = callFrameData
}

// generateCallFrameEnterElements generates a list of elements describing top level information about this call frame.
// This list of elements will hold information about what kind of call it is, wei sent, what method is called, and more.
// Additionally, the list may also hold formatting options for console output. This function also returns a non-empty
// string in case this call frame represents a call to the console.log precompile contract.
func (v *ValueGenerationTracer) generateCallFrameEnterElements(callFrame *utils.CallFrame) ([]any, string) {
	// Create list of elements and console log string
	elements := make([]any, 0)
	var consoleLogString string

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
				methodName = method.Sig
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
			// Encode the ABI arguments into strings
			encodedInputString, err := valuegeneration.EncodeABIArgumentsToString(method.Inputs, inputValues)
			if err == nil {
				inputArgumentsDisplayText = &encodedInputString
			}

			// If the call was made to the console log precompile address, let's retrieve the log and format it
			if callFrame.ToAddress == chain.ConsoleLogContractAddress {
				// First, attempt to do string formatting if the first element is a string, has a percent sign in it,
				// and there is at least one argument provided for formatting.
				exp := regexp.MustCompile(`%`)
				stringInput, isString := inputValues[0].(string)
				if isString && exp.MatchString(stringInput) && len(inputValues) > 1 {
					// Format the string and add it to the list of logs
					consoleLogString = fmt.Sprintf(inputValues[0].(string), inputValues[1:]...)
				} else {
					// The string does not need to be formatted, and we can just use the encoded input string
					consoleLogString = encodedInputString
				}

				// Add a bullet point before the string and a new line after the string
				if len(consoleLogString) > 0 {
					consoleLogString = colors.BULLET_POINT + " " + consoleLogString + "\n"
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
			if callFrame.ToAddress == chain.ConsoleLogContractAddress {
				callInfo = fmt.Sprintf("%v.%v(%v)", codeContractName, methodName, *inputArgumentsDisplayText)
			} else {
				callInfo = fmt.Sprintf("%v.%v(%v) (addr=%v, value=%v, sender=%v)", codeContractName, methodName, *inputArgumentsDisplayText, callFrame.ToAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
			}
		} else {
			callInfo = fmt.Sprintf("(addr=%v, value=%v, sender=%v)", callFrame.ToAddress.String(), callFrame.CallValue, callFrame.SenderAddress.String())
		}
	}

	// Add call information to the elements
	elements = append(elements, callInfo, "\n")

	return elements, consoleLogString
}

// resolveConstructorArgs resolves previously unresolved constructor argument ABI data from the call data, if
// the call frame provided represents a contract deployment.
func (v *ValueGenerationTracer) resolveCallFrameConstructorArgs(callFrame *utils.CallFrame, contract *contracts.Contract) {
	// If this is a contract creation and the constructor ABI argument data has not yet been resolved, do so now.
	if callFrame.ConstructorArgsData == nil && callFrame.IsContractCreation() {
		// We simply slice the compiled bytecode leading the input data off, and we are left with the constructor
		// arguments ABI data.
		compiledInitBytecode := contract.CompiledContract().InitBytecode
		if len(compiledInitBytecode) <= len(callFrame.InputData) {
			callFrame.ConstructorArgsData = callFrame.InputData[len(compiledInitBytecode):]
		}
	}
}

// resolveCallFrameContractDefinitions resolves previously unresolved contract definitions for the To and Code addresses
// used within the provided call frame.
func (v *ValueGenerationTracer) resolveCallFrameContractDefinitions(callFrame *utils.CallFrame) {
	// Try to resolve contract definitions for "to" address
	if callFrame.ToContractAbi == nil {
		// Try to resolve definitions from cheat code contracts
		if cheatCodeContract, ok := v.cheatCodeContracts[callFrame.ToAddress]; ok {
			callFrame.ToContractName = cheatCodeContract.Name()
			callFrame.ToContractAbi = cheatCodeContract.Abi()
			callFrame.ExecutedCode = true
		} else {
			// Try to resolve definitions from compiled contracts
			toContract := v.contractDefinitions.MatchBytecode(callFrame.ToInitBytecode, callFrame.ToRuntimeBytecode)
			if toContract != nil {
				callFrame.ToContractName = toContract.Name()
				callFrame.ToContractAbi = &toContract.CompiledContract().Abi
				v.resolveCallFrameConstructorArgs(callFrame, toContract)

				// If this is a contract creation, set the code address to the address of the contract we just deployed.
				if callFrame.IsContractCreation() {
					callFrame.CodeContractName = toContract.Name()
					callFrame.CodeContractAbi = &toContract.CompiledContract().Abi
				}
			}
		}
	}

	// Try to resolve contract definitions for "code" address
	if callFrame.CodeContractAbi == nil {
		// Try to resolve definitions from cheat code contracts
		if cheatCodeContract, ok := v.cheatCodeContracts[callFrame.CodeAddress]; ok {
			callFrame.CodeContractName = cheatCodeContract.Name()
			callFrame.CodeContractAbi = cheatCodeContract.Abi()
			callFrame.ExecutedCode = true
		} else {
			// Try to resolve definitions from compiled contracts
			codeContract := v.contractDefinitions.MatchBytecode(nil, callFrame.CodeRuntimeBytecode)
			if codeContract != nil {
				callFrame.CodeContractName = codeContract.Name()
				callFrame.CodeContractAbi = &codeContract.CompiledContract().Abi
			}
		}
	}
}

// captureExitedCallFrame is a helper method used when a call frame is exited, to record information about it.
func (v *ValueGenerationTracer) captureExitedCallFrame(output []byte, err error) {
	fmt.Println("Called captureExitedCallFrame")
	// If this was an initial deployment, now that we're exiting, we'll want to record the finally deployed bytecodes.
	if v.currentCallFrame.ToRuntimeBytecode == nil {
		// As long as this isn't a failed contract creation, we should be able to fetch "to" byte code on exit.
		if !v.currentCallFrame.IsContractCreation() || err == nil {
			v.currentCallFrame.ToRuntimeBytecode = v.evm.StateDB.GetCode(v.currentCallFrame.ToAddress)
		}
	}
	if v.currentCallFrame.CodeRuntimeBytecode == nil {
		// Optimization: If the "to" and "code" addresses match, we can simply set our "code" already fetched "to"
		// runtime bytecode.
		if v.currentCallFrame.CodeAddress == v.currentCallFrame.ToAddress {
			v.currentCallFrame.CodeRuntimeBytecode = v.currentCallFrame.ToRuntimeBytecode
		} else {
			v.currentCallFrame.CodeRuntimeBytecode = v.evm.StateDB.GetCode(v.currentCallFrame.CodeAddress)
		}
	}

	// Resolve our contract definitions on the call frame data, if they have not been.
	v.resolveCallFrameContractDefinitions(v.currentCallFrame)

	// Set our information for this call frame
	v.currentCallFrame.ReturnData = slices.Clone(output)
	v.currentCallFrame.ReturnError = err

	// We're exiting the current frame, so set our current call frame to the parent
	v.currentCallFrame = v.currentCallFrame.ParentCallFrame
}

func (v *ValueGenerationTracer) GetEventsGenerated(callFrame *utils.CallFrame, eventLog *coreTypes.Log) {
	// Try to unpack our event data
	event, eventInputValues := abiutils.UnpackEventAndValues(callFrame.CodeContractAbi, eventLog)

	if event == nil {
		// If we couldn't resolve the event from our immediate contract ABI, it may come from a library.
		for _, contract := range v.contractDefinitions {
			event, eventInputValues = abiutils.UnpackEventAndValues(&contract.CompiledContract().Abi, eventLog)
			if event != nil {
				break
			}
		}
	}

	// If we resolved an event definition and unpacked data.
	if event != nil {
		// Format the values as a comma-separated string
		encodedEventValuesString, _ := valuegeneration.EncodeABIArgumentsToString(event.Inputs, eventInputValues)
		myEncodedEventValuesString := encodedEventValuesString
		fmt.Println(myEncodedEventValuesString)
	}

	myEventLogData := eventLog.Data
	fmt.Printf("eventLog.Data: %+v\n", myEventLogData)
}

func (v *ValueGenerationTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	fmt.Println("Called CaptureState")
	// Execute all "on next capture state" events and clear them.
	for _, eventHandler := range v.onNextCaptureState {
		eventHandler()
	}
	v.onNextCaptureState = nil

	// If a log operation occurred, add a deferred operation to capture it.
	if op == vm.LOG0 || op == vm.LOG1 || op == vm.LOG2 || op == vm.LOG3 || op == vm.LOG4 {
		v.onNextCaptureState = append(v.onNextCaptureState, func() {
			logs := v.evm.StateDB.(*state.StateDB).Logs()
			if len(logs) > 0 {
				v.currentCallFrame.Operations = append(v.currentCallFrame.Operations, logs[len(logs)-1])
			}
		})
	}
}

func (v *ValueGenerationTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	//TODO implement me
	//panic("implement me")
}
func NewValueGenerationTracer() *ValueGenerationTracer {
	fmt.Println("Called NewValueGenerationTracer")
	// TODO: Sanan
	tracer := &ValueGenerationTracer{
		emittedValues:        make([]any, 0),
		functionReturnValues: make([]any, 0),
	}
	return tracer
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (v *ValueGenerationTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Store our tracer results.
	//results.Receipt.Logs = v.currentCallFrame.Operations
	//var eventLogs []*coreTypes.Log
	for _, operation := range v.currentCallFrame.Operations {
		if _, ok := operation.(*utils.CallFrame); ok {
			// If this is a call frame being entered, generate information recursively.
			fmt.Printf("CallFrame Operation: %+v\n", operation)
		} else if eventLog, ok := operation.(*coreTypes.Log); ok {
			// If an event log was emitted, add a message for it.
			fmt.Printf("Event Operation: %+v\n", eventLog)
			v.GetEventsGenerated(v.currentCallFrame, eventLog)
			//eventLogs = append(eventLogs, eventLog)
			results.Receipt.Logs = append(results.Receipt.Logs, eventLog)
		}
	}

}
