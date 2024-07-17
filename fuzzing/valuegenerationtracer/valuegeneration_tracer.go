package valuegenerationtracer

import (
	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/slices"
	"math/big"
)

// valueGenerationTracerResultsKey describes the key to use when storing tracer results in call message results, or when
// querying them.
const valueGenerationTracerResultsKey = "ValueGenerationTracerResults"

type ValueGenerationTrace struct {
	// TopLevelCallFrame refers to the root call frame, the first EVM call scope entered when an externally-owned
	// address calls upon a contract.
	TopLevelCallFrame *utils.CallFrame

	// contractDefinitions represents the known contract definitions at the time of tracing. This is used to help
	// obtain any additional information regarding execution.
	contractDefinitions     contracts.Contracts
	transactionOutputValues []any
}

type ValueGenerationTracer struct {
	// evm refers to the EVM instance last captured.
	evm *vm.EVM

	// trace represents the current execution trace captured by this tracer.
	trace *ValueGenerationTrace

	// currentCallFrame references the current call frame being traced.
	currentCallFrame *utils.CallFrame

	// contractDefinitions represents the contract definitions to match for execution traces.
	contractDefinitions contracts.Contracts

	// onNextCaptureState refers to methods which should be executed the next time CaptureState executes.
	// CaptureState is called prior to execution of an instruction. This allows actions to be performed
	// after some state is captured, on the next state capture (e.g. detecting a log instruction, but
	// using this structure to execute code later once the log is committed).
	onNextCaptureState []func()
}

func NewValueGenerationTracer(contractDefinitions contracts.Contracts) *ValueGenerationTracer {
	tracer := &ValueGenerationTracer{
		contractDefinitions: contractDefinitions,
	}
	return tracer
}

func (v *ValueGenerationTracer) CaptureTxStart(gasLimit uint64) {
	v.trace = newValueGenerationTrace(v.contractDefinitions)
	v.currentCallFrame = nil
	v.onNextCaptureState = nil
}

func (v *ValueGenerationTracer) CaptureTxEnd(restGas uint64) {
}

func (v *ValueGenerationTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	v.evm = env
	v.captureEnteredCallFrame(from, to, input, create, value)
}

func (v *ValueGenerationTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	v.captureExitedCallFrame(output, err)
}

func (v *ValueGenerationTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	v.captureEnteredCallFrame(from, to, input, typ == vm.CREATE || typ == vm.CREATE2, value)
}

func (v *ValueGenerationTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
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

	// Try to resolve contract definitions for "code" address
	if callFrame.CodeContractAbi == nil {
		codeContract := v.contractDefinitions.MatchBytecode(nil, callFrame.CodeRuntimeBytecode)
		if codeContract != nil {
			callFrame.CodeContractName = codeContract.Name()
			callFrame.CodeContractAbi = &codeContract.CompiledContract().Abi
		}

	}
}

// getCallFrameReturnValue generates a list of elements describing the return value of the call frame
func (t *ValueGenerationTracer) getCallFrameReturnValue() any {
	// Define some strings that represent our current call frame
	var method *abi.Method

	// Define a slice of any to represent return values of the current call frame
	//var outputValues TransactionOutputValues
	var outputValue any

	// Resolve our method definition
	if t.currentCallFrame.CodeContractAbi != nil {
		method, _ = t.currentCallFrame.CodeContractAbi.MethodById(t.currentCallFrame.InputData)
	}

	if method != nil {
		// Unpack our output values and obtain a string to represent them, only if we didn't encounter an error.
		if t.currentCallFrame.ReturnError == nil {
			outputValue, _ := method.Outputs.Unpack(t.currentCallFrame.ReturnData)
			outputValues = append(outputValues, outputValue)
		}
	}

	return outputValues
}

// captureExitedCallFrame is a helper method used when a call frame is exited, to record information about it.
func (v *ValueGenerationTracer) captureExitedCallFrame(output []byte, err error) {
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

	if v.currentCallFrame.ReturnError == nil {
		// Grab return data of the call frame
		v.trace.transactionOutputValues = append(v.trace.transactionOutputValues, v.getCallFrameReturnValue())
	}

	// We're exiting the current frame, so set our current call frame to the parent
	v.currentCallFrame = v.currentCallFrame.ParentCallFrame
}

func (v *ValueGenerationTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	// TODO: look for RET opcode (for now try getting them from currentCallFrame.ReturnData)
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
}

func AddTransactionOutputValuesToValueSet(results *types.MessageResults, valueSet *valuegeneration.ValueSet) {
	valueGenerationTracerResults := results.AdditionalResults["ValueGenerationTracerResults"]

	if transactionOutputValues, ok := valueGenerationTracerResults.(TransactionOutputValues); ok {

		for _, eventOrReturnValue := range transactionOutputValues {
			switch v := eventOrReturnValue.(type) {
			case *big.Int:
				valueSet.AddInteger(v)
			case common.Address:
				valueSet.AddAddress(v)
			case string:
				valueSet.AddString(v)
			case []byte:
				valueSet.AddBytes(v)
			default:
				continue

			}
		}
	}

	fmt.Printf("ValueSet after modification: %v\n", valueSet)
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (v *ValueGenerationTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Collect generated event and return values of the current transaction
	eventAndReturnValues := make([]any, 0)
	eventAndReturnValues = v.trace.generateEvents(v.trace.TopLevelCallFrame, eventAndReturnValues)

	v.trace.transactionOutputValues = append(v.trace.transactionOutputValues, eventAndReturnValues)

	results.AdditionalResults[valueGenerationTracerResultsKey] = v.trace.transactionOutputValues

}

func (t *ValueGenerationTrace) generateEvents(currentCallFrame *utils.CallFrame, events []any) []any {
	for _, operation := range currentCallFrame.Operations {
		if childCallFrame, ok := operation.(*utils.CallFrame); ok {
			// If this is a call frame being entered, generate information recursively.
			t.generateEvents(childCallFrame, events)
		} else if eventLog, ok := operation.(*coreTypes.Log); ok {
			// If an event log was emitted, add a message for it.
			events = append(events, t.getEventsGenerated(currentCallFrame, eventLog)...)
			//t.getEventsGenerated(currentCallFrame, eventLog)
			//eventLogs = append(eventLogs, eventLog)
		}
	}
	return events
}

func (t *ValueGenerationTrace) getEventsGenerated(callFrame *utils.CallFrame, eventLog *coreTypes.Log) []any {
	// Try to unpack our event data
	eventInputs := t.transactionOutputValues
	event, eventInputValues := abiutils.UnpackEventAndValues(callFrame.CodeContractAbi, eventLog)

	if event == nil {
		// If we couldn't resolve the event from our immediate contract ABI, it may come from a library.
		for _, contract := range t.contractDefinitions {
			event, eventInputValues = abiutils.UnpackEventAndValues(&contract.CompiledContract().Abi, eventLog)
			if event != nil {
				break
			}
		}
	}

	if event != nil {
		for _, value := range eventInputValues {
			eventInputs = append(eventInputs, value)
		}
	}

	return eventInputs
}
