package valuegenerationtracer

import (
	"fmt"
	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
	"github.com/crytic/medusa/utils"
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
	v.evm = env
	v.captureEnteredCallFrame(from, to, input, create, value)
	return
	//panic("implement me")
}

func (v *ValueGenerationTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	//TODO implement me
	//panic("implement me")
}

func (v *ValueGenerationTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	//TODO implement me
}

func (v *ValueGenerationTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	//TODO implement me
	//panic("implement me")
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
		}
	}
}
