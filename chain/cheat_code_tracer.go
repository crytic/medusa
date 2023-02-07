package chain

import (
	"github.com/trailofbits/medusa/chain/types"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// cheatCodeTracer represents an EVM.Logger which tracks and patches EVM execution state to enable extended
// testing functionality on-chain.
type cheatCodeTracer struct {
	// callDepth refers to the current EVM depth during tracing.
	callDepth uint64

	// evm refers to the EVM instance last captured.
	evm *vm.EVM

	// pendingCallFrames represents per-call-frame data deployment information being captured by the tracer.
	pendingCallFrames []*cheatCodeTracerCallFrame
}

// cheatCodeTracerCallFrame represents per-call-frame data traced by a cheatCodeTracer.
type cheatCodeTracerCallFrame struct {
	// onNextFrameEnterHooks describes hooks which will be executed the next time this call frame executes a call,
	// creating "the next call frame".
	onNextFrameEnterHooks cheatCodeTracerHooks
	// onNextFrameExitHooks describes hooks which will be executed the next time this call frame executes a call,
	// and exits it, "exiting the next call frame".
	onNextFrameExitHooks cheatCodeTracerHooks
	// onFrameExitHooks describes hooks when are executed when this call frame is exited.
	onFrameExitHooks cheatCodeTracerHooks

	// vmPc describes the current call frame's program counter.
	vmPc uint64
	// vmOp describes the current call frame's last instruction executed.
	vmOp vm.OpCode
	// vmScope describes the current call frame's scope context.
	vmScope *vm.ScopeContext
	// vmReturnData describes the current call frame's return data (set on exit).
	vmReturnData []byte
	// vmErr describes the current call frame's returned error (set on exit), nil if no error.
	vmErr error
}

// cheatCodeTracerHook defines a function to be called when some tracer event occurs. This is used to trigger "undo"
// operations to ensure some changes only take effect in certain call scopes/depths.
type cheatCodeTracerHook func()

// cheatCodeTracerHooks wraps a list of cheatCodeTracerHook items. It acts as a stack of undo operations.
type cheatCodeTracerHooks []cheatCodeTracerHook

// Execute pops each hook off the stack and executes it until there are no more hooks left to execute.
func (t *cheatCodeTracerHooks) Execute() {
	// If the hooks aren't set yet, do nothing.
	if t == nil {
		return
	}

	// Otherwise execute every hook in reverse order (as this is a stack)
	for i := len(*t) - 1; i >= 0; i-- {
		(*t)[i]()
	}

	// And set the hook to nil.
	*t = nil
}

// Push pushes a provided hook onto the stack.
func (t *cheatCodeTracerHooks) Push(f cheatCodeTracerHook) {
	// Push the provided hook onto the stack.
	*t = append(*t, f)
}

// newCheatCodeTracer creates a cheatCodeTracer and returns it.
func newCheatCodeTracer() *cheatCodeTracer {
	tracer := &cheatCodeTracer{}
	return tracer
}

// TopCallFrame returns the top call frame (initial call produced by EOA account) of the current EVM execution,
// or nil if no frame has been entered.
func (t *cheatCodeTracer) TopCallFrame() *cheatCodeTracerCallFrame {
	if len(t.pendingCallFrames) == 0 {
		return nil
	}
	return t.pendingCallFrames[0]
}

// PreviousCallFrame returns the previous call frame of the current EVM execution, or nil if there is no previous.
func (t *cheatCodeTracer) PreviousCallFrame() *cheatCodeTracerCallFrame {
	if len(t.pendingCallFrames) < 2 {
		return nil
	}
	return t.pendingCallFrames[t.callDepth-1]
}

// CurrentCallFrame returns the current call frame of the EVM execution, or nil if there is none.
func (t *cheatCodeTracer) CurrentCallFrame() *cheatCodeTracerCallFrame {
	if len(t.pendingCallFrames) == 0 {
		return nil
	}
	return t.pendingCallFrames[t.callDepth]
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureTxStart(gasLimit uint64) {
	// Reset our capture state
	t.callDepth = 0
	t.pendingCallFrames = make([]*cheatCodeTracerCallFrame, 0)
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureTxEnd(restGas uint64) {

}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Store our evm reference
	t.evm = env

	// Create our call frame struct to track data for this initial entry call frame.
	callFrameData := &cheatCodeTracerCallFrame{}
	t.pendingCallFrames = append(t.pendingCallFrames, callFrameData)
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	// Execute all current call frame exit hooks
	t.pendingCallFrames[t.callDepth].onFrameExitHooks.Execute()

	// We're exiting the current frame, so remove our frame data.
	t.pendingCallFrames = t.pendingCallFrames[:t.callDepth]
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// We haven't updated our call depth yet, so obtain the "previous" call frame (current for now)
	previousCallFrame := t.CurrentCallFrame()

	// Increase our call depth now that we're entering a new call frame.
	t.callDepth++

	// Create our call frame struct to track data for this initial entry call frame.
	// We forward our "next frame hooks" to this frame, then clear them from the previous frame.
	callFrameData := &cheatCodeTracerCallFrame{
		onFrameExitHooks: previousCallFrame.onNextFrameExitHooks,
	}
	previousCallFrame.onNextFrameExitHooks = nil
	t.pendingCallFrames = append(t.pendingCallFrames, callFrameData)

	// Note: We do not execute events for "next frame enter" here, as we do not yet have scope information.
	// Those events are executed when the first EVM instruction is executed in the new scope.
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// Execute all current call frame exit hooks
	exitingCallFrame := t.pendingCallFrames[t.callDepth]
	exitingCallFrame.onFrameExitHooks.Execute()

	// We're exiting the current frame, so remove our frame data.
	t.pendingCallFrames = t.pendingCallFrames[:t.callDepth]

	// Decrease our call depth now that we've exited a call frame.
	t.callDepth--
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// Set our current frame information.
	currentCallFrame := t.CurrentCallFrame()
	currentCallFrame.vmPc = pc
	currentCallFrame.vmOp = op
	currentCallFrame.vmScope = scope
	currentCallFrame.vmReturnData = rData
	currentCallFrame.vmErr = vmErr

	// We execute our entered next frame hooks here (from our previous call frame), as we now have scope information.
	if t.callDepth > 0 {
		t.pendingCallFrames[t.callDepth-1].onNextFrameEnterHooks.Execute()
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {

}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *cheatCodeTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {

}
