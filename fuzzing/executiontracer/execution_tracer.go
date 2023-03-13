package executiontracer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/trailofbits/medusa/chain/types"
	"github.com/trailofbits/medusa/fuzzing/contracts"
	"golang.org/x/exp/slices"
	"math/big"
)

// executionTracerResultsKey describes the key to use when storing tracer results in call message results, or when
// querying them.
const executionTracerResultsKey = "ExecutionTracerResults"

// GetExecutionTracerResults obtains an ExecutionTrace stored by a ExecutionTracer from message results. This is nil if
// no ExecutionTrace was recorded by a tracer (e.g. ExecutionTracer was not attached during this message execution).
func GetExecutionTracerResults(messageResults *types.MessageResults) *ExecutionTrace {
	// Try to obtain the results the tracer should've stored.
	if genericResult, ok := messageResults.AdditionalResults[executionTracerResultsKey]; ok {
		if castedResult, ok := genericResult.(*ExecutionTrace); ok {
			return castedResult
		}
	}

	// If we could not obtain them, return nil.
	return nil
}

// RemoveExecutionTracerResults removes an ExecutionTrace stored by an ExecutionTracer, from message results.
func RemoveExecutionTracerResults(messageResults *types.MessageResults) {
	delete(messageResults.AdditionalResults, executionTracerResultsKey)
}

// ExecutionTracer records execution information into an ExecutionTrace, containing information about each call
// scope entered and exited.
type ExecutionTracer struct {
	// callDepth refers to the current EVM depth during tracing.
	callDepth uint64

	// evm refers to the EVM instance last captured.
	evm *vm.EVM

	// trace represents the current execution trace captured by this tracer.
	trace *ExecutionTrace

	// currentCallFrame references the current call frame being traced.
	currentCallFrame *CallFrame

	// contractDefinitions represents the contract definitions to match for execution traces.
	contractDefinitions contracts.Contracts
}

// NewExecutionTracer creates a ExecutionTracer and returns it.
func NewExecutionTracer(contractDefinitions contracts.Contracts) *ExecutionTracer {
	tracer := &ExecutionTracer{
		contractDefinitions: contractDefinitions,
	}
	return tracer
}

// Trace returns the currently recording or last recorded execution trace by the tracer.
func (t *ExecutionTracer) Trace() *ExecutionTrace {
	return t.trace
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureTxStart(gasLimit uint64) {
	// Reset our capture state
	t.callDepth = 0
	t.trace = newExecutionTrace()
	t.currentCallFrame = nil
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureTxEnd(restGas uint64) {

}

// resolveCallFrameContractDefinitions resolves previously unresolved contract definitions for the To and Code addresses
// used within the provided call frame.
func (t *ExecutionTracer) resolveCallFrameContractDefinitions(callFrame *CallFrame) {
	// If we didn't resolve contract definitions, do so now.
	if callFrame.ToContract == nil {
		callFrame.ToContract = t.contractDefinitions.MatchBytecode(callFrame.ToInitBytecode, callFrame.ToRuntimeBytecode)
	}
	if callFrame.CodeContract == nil {
		callFrame.CodeContract = t.contractDefinitions.MatchBytecode(nil, callFrame.CodeRuntimeBytecode)
	}
}

// captureEnteredCallFrame is a helper method used when a new call frame is entered to record information about it.
func (t *ExecutionTracer) captureEnteredCallFrame(fromAddress common.Address, toAddress common.Address, codeAddress common.Address, inputData []byte, isContractCreation bool) {
	// Create our call frame struct to track data for this call frame we entered.
	callFrameData := &CallFrame{
		SenderAddress:       fromAddress,
		ToAddress:           toAddress,
		ToContract:          nil,
		ToInitBytecode:      nil,
		ToRuntimeBytecode:   nil,
		CodeAddress:         codeAddress,
		CodeContract:        nil,
		CodeRuntimeBytecode: nil,
		InputData:           inputData,
		ReturnData:          nil,
		ReturnError:         nil,
		ChildCallFrames:     make(CallFrames, 0),
		ParentCallFrame:     t.currentCallFrame,
	}

	// Set our known information about the code we're executing.
	if isContractCreation {
		callFrameData.ToInitBytecode = inputData
	} else {
		callFrameData.ToRuntimeBytecode = t.evm.StateDB.GetCode(callFrameData.ToAddress)
	}

	// If we are performing a proxy call, we use code from another address, fetch that code, so we can later match the
	// contract definition. Otherwise, we record the same bytecode.
	if callFrameData.CodeAddress != callFrameData.ToAddress {
		callFrameData.CodeRuntimeBytecode = t.evm.StateDB.GetCode(callFrameData.CodeAddress)
	} else {
		callFrameData.CodeRuntimeBytecode = callFrameData.ToRuntimeBytecode
	}

	// Resolve our contract definitions on the call frame data, if they have not been.
	t.resolveCallFrameContractDefinitions(callFrameData)

	// Set our current call frame in our trace
	if t.trace.TopLevelCallFrame == nil {
		t.trace.TopLevelCallFrame = callFrameData
	} else {
		t.currentCallFrame.ChildCallFrames = append(t.currentCallFrame.ChildCallFrames, callFrameData)
	}
	t.currentCallFrame = callFrameData
}

// captureExitedCallFrame is a helper method used when a call frame is exited, to record information about it.
func (t *ExecutionTracer) captureExitedCallFrame(output []byte, err error) {
	// If this was an initial deployment, now that we're exiting, we'll want to record the finally deployed bytecode.
	callFrameData := t.currentCallFrame
	if callFrameData.ToInitBytecode != nil && err == nil {
		callFrameData.CodeRuntimeBytecode = t.evm.StateDB.GetCode(callFrameData.CodeAddress)
	}

	// Resolve our contract definitions on the call frame data, if they have not been.
	t.resolveCallFrameContractDefinitions(callFrameData)

	// Set our information for this call frame
	callFrameData.ReturnData = slices.Clone(output)
	callFrameData.ReturnError = err

	// We're exiting the current frame, so set our current call frame to the parent
	t.currentCallFrame = t.currentCallFrame.ParentCallFrame
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Store our evm reference
	t.evm = env

	// Capture that a new call frame was entered.
	t.captureEnteredCallFrame(from, to, to, input, create)
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	// Capture that the call frame was exited.
	t.captureExitedCallFrame(output, err)
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Increase our call depth now that we're entering a new call frame.
	t.callDepth++

	// Capture that a new call frame was entered.
	t.captureEnteredCallFrame(from, to, to, input, typ == vm.CREATE || typ == vm.CREATE2)
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// Capture that the call frame was exited.
	t.captureExitedCallFrame(output, err)

	// Decrease our call depth now that we've exited a call frame.
	t.callDepth--
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// Obtain our current call frame.
	callFrameData := t.currentCallFrame

	// If we encounter a SELFDESTRUCT operation, record the operation.
	if op == vm.SELFDESTRUCT {
		destructedAddress := scope.Contract.Address()
		runtimeBytecode := t.evm.StateDB.GetCode(destructedAddress)

		// TODO: Implement self-destruct tracking in the call frame.
		_, _ = callFrameData, runtimeBytecode
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {

}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *ExecutionTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Store our tracer results.
	results.AdditionalResults[executionTracerResultsKey] = t.trace
}
