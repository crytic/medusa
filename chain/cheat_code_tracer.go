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
	onNextFrameEnterHooks   cheatCodeTracerHooks
	onNextFrameExitHooks    cheatCodeTracerHooks
	onCurrentFrameExitHooks cheatCodeTracerHooks
}

// cheatCodeTracerHook defines a function to be called when some tracer event occurs. This is used to trigger "undo"
// operations to ensure some changes only take effect in certain call scopes/depths.
type cheatCodeTracerHook func()

// cheatCodeTracerHooks wraps a list of cheatCodeTracerHook items. It acts as a stack of undo operations.
type cheatCodeTracerHooks []cheatCodeTracerHook

// Execute pops each hook off the stack and executes it until there are no more hooks left to execute.
func (t *cheatCodeTracerHooks) Execute() {
	for i := len(*t) - 1; i >= 0; i-- {
		(*t)[i]()
	}
	*t = nil
}

func (t *cheatCodeTracerHooks) Push(f cheatCodeTracerHook) {
	*t = append(*t, f)
}

// newCheatCodeTracer creates a cheatCodeTracer
func newCheatCodeTracer() *cheatCodeTracer {
	tracer := &cheatCodeTracer{}
	return tracer
}

func (t *cheatCodeTracer) TopCallFrame() *cheatCodeTracerCallFrame {
	if len(t.pendingCallFrames) == 0 {
		return nil
	}
	return t.pendingCallFrames[0]
}

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
	t.pendingCallFrames[t.callDepth].onCurrentFrameExitHooks.Execute()

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
		onCurrentFrameExitHooks: previousCallFrame.onNextFrameExitHooks,
	}
	previousCallFrame.onNextFrameExitHooks = nil
	t.pendingCallFrames = append(t.pendingCallFrames, callFrameData)

	// Execute our hooks for entering this call depth.
	previousCallFrame.onNextFrameEnterHooks.Execute()
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// Execute all current call frame exit hooks
	exitingCallFrame := t.pendingCallFrames[t.callDepth]
	exitingCallFrame.onCurrentFrameExitHooks.Execute()

	// We're exiting the current frame, so remove our frame data.
	t.pendingCallFrames = t.pendingCallFrames[:t.callDepth]

	// Decrease our call depth now that we've exited a call frame.
	t.callDepth--
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {

}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {

}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *cheatCodeTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {

}
