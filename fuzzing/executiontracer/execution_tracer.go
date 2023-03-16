package executiontracer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/fuzzing/contracts"
	"golang.org/x/exp/slices"
	"math/big"
)

// CallWithExecutionTrace obtains an execution trace for a given call, on the provided chain, using the state
// provided. If a nil state is provided, the current chain state will be used.
// Returns the ExecutionTrace for the call or an error if one occurs.
func CallWithExecutionTrace(chain *chain.TestChain, contractDefinitions contracts.Contracts, msg core.Message, state *state.StateDB) (*core.ExecutionResult, *ExecutionTrace, error) {
	// Create an execution tracer
	executionTracer := NewExecutionTracer(contractDefinitions)

	// Call the contract on our chain with the provided state.
	executionResult, err := chain.CallContract(msg, state, executionTracer)
	if err != nil {
		return nil, nil, err
	}

	// Obtain our trace
	trace := executionTracer.Trace()

	// Return the trace
	return executionResult, trace, nil
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

	// onNextCaptureState refers to methods which should be executed the next time CaptureState executes.
	// CaptureState is called prior to execution of an instruction. This allows actions to be performed
	// after some state is captured, on the next state capture (e.g. detecting a log instruction, but
	// using this structure to execute code later once the log is committed).
	onNextCaptureState []func()
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
	t.onNextCaptureState = nil
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
func (t *ExecutionTracer) captureEnteredCallFrame(fromAddress common.Address, toAddress common.Address, codeAddress common.Address, inputData []byte, isContractCreation bool, value *big.Int) {
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
		Operations:          make([]any, 0),
		SelfDestructed:      false,
		InputData:           inputData,
		ReturnData:          nil,
		ExecutedCode:        false,
		CallValue:           value,
		ReturnError:         nil,
		ParentCallFrame:     t.currentCallFrame,
	}

	// Set our known information about the code we're executing.
	if isContractCreation {
		callFrameData.ToInitBytecode = inputData
	} else {
		callFrameData.ToRuntimeBytecode = t.evm.StateDB.GetCode(callFrameData.ToAddress)
	}

	// If we are performing a proxy call, we use code from another address (which we can expect to exist), and fetch
	// that code, so we can later match the contract definition. Otherwise, we record the same bytecode as "to".
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
		t.currentCallFrame.Operations = append(t.currentCallFrame.Operations, callFrameData)
	}
	t.currentCallFrame = callFrameData
}

// captureExitedCallFrame is a helper method used when a call frame is exited, to record information about it.
func (t *ExecutionTracer) captureExitedCallFrame(output []byte, err error) {
	// If this was an initial deployment, now that we're exiting, we'll want to record the finally deployed bytecodes.
	callFrameData := t.currentCallFrame
	if err == nil {
		if callFrameData.ToRuntimeBytecode == nil {
			callFrameData.ToRuntimeBytecode = t.evm.StateDB.GetCode(callFrameData.ToAddress)
		}
		if callFrameData.CodeRuntimeBytecode == nil {
			// Optimization: If the "to" and "code" addresses match, we can simply set our "code" already fetched "to"
			// runtime bytecode.
			if callFrameData.CodeAddress == callFrameData.ToAddress {
				callFrameData.CodeRuntimeBytecode = callFrameData.ToRuntimeBytecode
			} else {
				callFrameData.CodeRuntimeBytecode = t.evm.StateDB.GetCode(callFrameData.CodeAddress)
			}
		}
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
	t.captureEnteredCallFrame(from, to, to, input, create, value)
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
	t.captureEnteredCallFrame(from, to, to, input, typ == vm.CREATE || typ == vm.CREATE2, value)
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
	// Execute all "on next capture state" events and clear them.
	for _, eventHandler := range t.onNextCaptureState {
		eventHandler()
	}
	t.onNextCaptureState = nil

	// Obtain our current call frame.
	callFrameData := t.currentCallFrame

	// Since we are executing an opcode, we can mark that we have executed code in this call frame
	callFrameData.ExecutedCode = true

	// If we encounter a SELFDESTRUCT operation, record the operation.
	if op == vm.SELFDESTRUCT {
		callFrameData.SelfDestructed = true
	}

	// If a log operation occurred, add a deferred operation to capture it.
	if op == vm.LOG0 || op == vm.LOG1 || op == vm.LOG2 || op == vm.LOG3 || op == vm.LOG4 {
		t.onNextCaptureState = append(t.onNextCaptureState, func() {
			logs := t.evm.StateDB.(*state.StateDB).Logs()
			if len(logs) > 0 {
				t.currentCallFrame.Operations = append(t.currentCallFrame.Operations, logs[len(logs)-1])
			}
		})
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {

}
