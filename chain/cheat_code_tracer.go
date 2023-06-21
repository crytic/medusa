package chain

import (
	"github.com/crytic/medusa/chain/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
)

// cheatCodeTracer represents an EVM.Logger which tracks and patches EVM execution state to enable extended
// testing functionality on-chain.
type cheatCodeTracer struct {
	// chain refers to the TestChain which this tracer is bound to. This is nil when the tracer is first created,
	// but is set to TestChain which created it, after it is added.
	chain *TestChain

	// callDepth refers to the current EVM depth during tracing.
	callDepth uint64

	// evm refers to the EVM instance last captured.
	evm *vm.EVM

	// callFrames represents per-call-frame data deployment information being captured by the tracer.
	callFrames []*cheatCodeTracerCallFrame

	// results stores the tracer output after a transaction has concluded.
	results *cheatCodeTracerResults
}

// cheatCodeTracerCallFrame represents per-call-frame data traced by a cheatCodeTracer.
type cheatCodeTracerCallFrame struct {
	// onNextFrameEnterHooks describes hooks which will be executed the next time this call frame executes a call,
	// creating "the next call frame".
	// The hooks are executed as a queue on entry.
	onNextFrameEnterHooks types.GenericHookFuncs
	// onNextFrameExitRestoreHooks describes hooks which will be executed the next time this call frame executes a call,
	// and exits it, "exiting the next call frame".
	// The hooks are executed as a stack on exit (to support revert operations).
	onNextFrameExitRestoreHooks types.GenericHookFuncs
	// onFrameExitRestoreHooks describes hooks which are executed when this call frame is exited.
	// The hooks are executed as a stack on exit (to support revert operations).
	onFrameExitRestoreHooks types.GenericHookFuncs

	// onTopFrameExitRestoreHooks describes hooks which are executed when this scope, or a parent scope reverts, or the
	// top call frame is exiting.
	// The hooks are executed as a stack (to support revert operations).
	onTopFrameExitRestoreHooks types.GenericHookFuncs

	// onChainRevertRestoreHooks describes hooks which are executed when this scope, a parent scope, or the chain reverts.
	// This is propagated up the call stack, only triggering if a call frame reverts. If it does not revert,
	// it is stored in a block and only called when the block is reverted.
	// The hooks are executed as a stack (to support revert operations).
	onChainRevertRestoreHooks types.GenericHookFuncs

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

type cheatCodeTracerResults struct {
	// onChainRevertHooks describes hooks which are to be executed when the chain reverts.
	onChainRevertHooks types.GenericHookFuncs
}

// newCheatCodeTracer creates a cheatCodeTracer and returns it.
func newCheatCodeTracer() *cheatCodeTracer {
	tracer := &cheatCodeTracer{}
	return tracer
}

// bindToChain is called by the TestChain which created the tracer to set its reference.
// Note: This is done because of the cheat code system's dependency on the genesis block, as well as chain's dependency
// on it, which prevents the chain being set in the tracer on initialization.
func (t *cheatCodeTracer) bindToChain(chain *TestChain) {
	t.chain = chain
}

// PreviousCallFrame returns the previous call frame of the current EVM execution, or nil if there is no previous.
func (t *cheatCodeTracer) PreviousCallFrame() *cheatCodeTracerCallFrame {
	if len(t.callFrames) < 2 {
		return nil
	}
	return t.callFrames[t.callDepth-1]
}

// CurrentCallFrame returns the current call frame of the EVM execution, or nil if there is none.
func (t *cheatCodeTracer) CurrentCallFrame() *cheatCodeTracerCallFrame {
	if len(t.callFrames) == 0 {
		return nil
	}
	return t.callFrames[t.callDepth]
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureTxStart(gasLimit uint64) {
	// Reset our capture state
	t.callDepth = 0
	t.callFrames = make([]*cheatCodeTracerCallFrame, 0)
	t.results = &cheatCodeTracerResults{
		onChainRevertHooks: nil,
	}
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
	t.callFrames = append(t.callFrames, callFrameData)
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	// Execute all current call frame exit hooks
	exitingCallFrame := t.callFrames[t.callDepth]
	exitingCallFrame.onFrameExitRestoreHooks.Execute(false, true)
	exitingCallFrame.onTopFrameExitRestoreHooks.Execute(false, true)

	// If we didn't encounter an error in this call frame, we push our upward propagating revert events up one frame.
	if err == nil {
		// Store these revert hooks in our results.
		t.results.onChainRevertHooks = append(t.results.onChainRevertHooks, exitingCallFrame.onChainRevertRestoreHooks...)
	} else {
		// We hit an error, so a revert occurred before this tx was committed.
		exitingCallFrame.onChainRevertRestoreHooks.Execute(false, true)
	}

	// We're exiting the current frame, so remove our frame data.
	t.callFrames = t.callFrames[:t.callDepth]
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
		onFrameExitRestoreHooks: previousCallFrame.onNextFrameExitRestoreHooks,
	}
	previousCallFrame.onNextFrameExitRestoreHooks = nil
	t.callFrames = append(t.callFrames, callFrameData)

	// Note: We do not execute events for "next frame enter" here, as we do not yet have scope information.
	// Those events are executed when the first EVM instruction is executed in the new scope.
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// Execute all current call frame exit hooks
	exitingCallFrame := t.callFrames[t.callDepth]
	exitingCallFrame.onFrameExitRestoreHooks.Execute(false, true)
	parentCallFrame := t.callFrames[t.callDepth-1]

	// If we didn't encounter an error in this call frame, we push our upward propagating revert events up one frame.
	if err == nil {
		parentCallFrame.onTopFrameExitRestoreHooks = append(parentCallFrame.onTopFrameExitRestoreHooks, exitingCallFrame.onTopFrameExitRestoreHooks...)
		parentCallFrame.onChainRevertRestoreHooks = append(parentCallFrame.onChainRevertRestoreHooks, exitingCallFrame.onChainRevertRestoreHooks...)
	} else {
		// We hit an error, so a revert occurred before this tx was committed.
		exitingCallFrame.onChainRevertRestoreHooks.Execute(false, true)
	}

	// We're exiting the current frame, so remove our frame data.
	t.callFrames = t.callFrames[:t.callDepth]

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
		t.callFrames[t.callDepth-1].onNextFrameEnterHooks.Execute(true, true)
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *cheatCodeTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {

}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *cheatCodeTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Add our revert operations we collected for this transaction.
	results.OnRevertHookFuncs = append(results.OnRevertHookFuncs, t.results.onChainRevertHooks...)
}
