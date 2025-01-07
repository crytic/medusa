package chain

import (
	"math/big"

	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	coretypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
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
	evmContext *tracing.VMContext

	// callFrames represents per-call-frame data deployment information being captured by the tracer.
	callFrames []*cheatCodeTracerCallFrame

	// results stores the tracer output after a transaction has concluded.
	results *cheatCodeTracerResults

	// nativeTracer is the underlying tracer interface that the cheatcode tracer follows
	nativeTracer *TestChainTracer
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
	vmScope tracing.OpContext
	// vmReturnData describes the current call frame's return data (set on exit).
	vmReturnData []byte
	// vmErr describes the current call frame's returned error (set on exit), nil if no error.
	vmErr error
}

// cheatCodeTracerResults holds the hooks that need to be executed when the chain reverts.
type cheatCodeTracerResults struct {
	// executionResult describes the results for the most recently concluded transaction
	executionResult *core.ExecutionResult

	// onChainRevertHooks describes hooks which are to be executed when the chain reverts.
	onChainRevertHooks types.GenericHookFuncs
}

// newCheatCodeTracer creates a cheatCodeTracer and returns it.
func newCheatCodeTracer() *cheatCodeTracer {
	tracer := &cheatCodeTracer{}
	innerTracer := &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart: tracer.OnTxStart,
			OnTxEnd:   tracer.OnTxEnd,
			OnEnter:   tracer.OnEnter,
			OnExit:    tracer.OnExit,
			OnOpcode:  tracer.OnOpcode,
		},
	}
	tracer.nativeTracer = &TestChainTracer{Tracer: innerTracer, CaptureTxEndSetAdditionalResults: tracer.CaptureTxEndSetAdditionalResults}

	return tracer
}

// NativeTracer returns the underlying TestChainTracer.
func (t *cheatCodeTracer) NativeTracer() *TestChainTracer {
	return t.nativeTracer
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

// OnTxStart is called upon the start of transaction execution, as defined by tracers.Tracer.
func (t *cheatCodeTracer) OnTxStart(vm *tracing.VMContext, tx *coretypes.Transaction, from common.Address) {
	// Reset our capture state
	t.callDepth = 0
	t.callFrames = make([]*cheatCodeTracerCallFrame, 0)
	t.results = &cheatCodeTracerResults{
		executionResult:    &core.ExecutionResult{},
		onChainRevertHooks: nil,
	}
	// Store our evm reference
	t.evmContext = vm
}

// OnTxEnd is called upon the end of transaction execution, as defined by tracers.Tracer
func (t *cheatCodeTracer) OnTxEnd(*coretypes.Receipt, error) {

}

// OnEnter initializes the tracing operation for the top of a call frame, as defined by tracers.Tracer.
func (t *cheatCodeTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Check to see if this is the top level call frame
	isTopLevelFrame := depth == 0
	var callFrameData *cheatCodeTracerCallFrame
	if isTopLevelFrame {
		// Create our call frame struct to track data for this initial entry call frame.
		callFrameData = &cheatCodeTracerCallFrame{}
	} else {
		// We haven't updated our call depth yet, so obtain the "previous" call frame (current for now)
		previousCallFrame := t.CurrentCallFrame()

		// Create our call frame struct to track data for this initial entry call frame.
		// We forward our "next frame hooks" to this frame, then clear them from the previous frame.
		callFrameData = &cheatCodeTracerCallFrame{
			onFrameExitRestoreHooks: previousCallFrame.onNextFrameExitRestoreHooks,
		}
		previousCallFrame.onNextFrameExitRestoreHooks = nil

		// Increase our call depth now that we're entering a new call frame.
		t.callDepth++
	}

	// Append our new call frame
	t.callFrames = append(t.callFrames, callFrameData)

	// Note: We do not execute events for "next frame enter" here, as we do not yet have scope information.
	// Those events are executed when the first EVM instruction is executed in the new scope.
}

// OnExit is called after a call to finalize tracing completes for the top of a call frame, as defined by tracers.Tracer.
func (t *cheatCodeTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	// Execute all current call frame exit hooks
	exitingCallFrame := t.callFrames[t.callDepth]

	exitingCallFrame.vmReturnData = output
	exitingCallFrame.vmErr = err

	exitingCallFrame.onFrameExitRestoreHooks.Execute(false, true)

	var parentCallFrame *cheatCodeTracerCallFrame
	if depth == 0 {
		// If this is the top-level call frame, execute all of its exit hooks
		exitingCallFrame.onTopFrameExitRestoreHooks.Execute(false, true)
	} else {
		// If not, retrieve the parent call frame
		parentCallFrame = t.callFrames[t.callDepth-1]
	}

	// We're exiting the current frame, so remove our frame data.
	t.callFrames = t.callFrames[:t.callDepth]

	// If we didn't encounter an error in this call frame, we push our upward propagating restore events up one frame.
	if err == nil && depth == 0 {
		// Since this is the top call frame, we add the revert events to the results of the tracer and return early
		t.results.onChainRevertHooks = append(t.results.onChainRevertHooks, exitingCallFrame.onChainRevertRestoreHooks...)
		return
	} else if err == nil {
		// Propagate hooks up to the parent call frame
		parentCallFrame.onTopFrameExitRestoreHooks = append(parentCallFrame.onTopFrameExitRestoreHooks, exitingCallFrame.onTopFrameExitRestoreHooks...)
		parentCallFrame.onChainRevertRestoreHooks = append(parentCallFrame.onChainRevertRestoreHooks, exitingCallFrame.onChainRevertRestoreHooks...)
	} else {
		// We hit an error, so a revert occurred before this tx was committed.
		exitingCallFrame.onChainRevertRestoreHooks.Execute(false, true)
	}

	// Decrease our call depth now that we've exited a call frame.
	t.callDepth--
}

// OnOpcode records data from an EVM state update, as defined by tracers.Tracer.
func (t *cheatCodeTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	// Set our current frame information.
	currentCallFrame := t.CurrentCallFrame()
	currentCallFrame.vmPc = pc
	currentCallFrame.vmOp = vm.OpCode(op)
	currentCallFrame.vmScope = scope
	currentCallFrame.vmReturnData = rData
	currentCallFrame.vmErr = err

	// We execute our entered next frame hooks here (from our previous call frame), as we now have scope information.
	if t.callDepth > 0 {
		t.callFrames[t.callDepth-1].onNextFrameEnterHooks.Execute(true, true)
	}
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *cheatCodeTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Add our revert operations we collected for this transaction.
	results.OnRevertHookFuncs = append(results.OnRevertHookFuncs, t.results.onChainRevertHooks...)
	results.ExecutionResult = t.results.executionResult
}

// ThrowAssertionError is used to trigger an assertion failure from within a cheatcode
func (t *cheatCodeTracer) ThrowAssertionError() {
	// Define ABI types
	uintType, _ := abi.NewType("uint256", "", nil)

	// Create Panic ABI method
	panicReturnDataAbi := abi.NewMethod("Panic", "Panic", abi.Function, "", false, false, []abi.Argument{
		{Name: "", Type: uintType, Indexed: false},
	}, abi.Arguments{})

	// Set panic code to 1 which represents an assertion failure
	panicCode := big.NewInt(abiutils.PanicCodeAssertFailed)

	// Pack the values into ABI encoded data
	packedData, err := panicReturnDataAbi.Inputs.Pack(panicCode)
	if err != nil {
		panic(err)
	}

	// Add selector to the packed data
	selector := panicReturnDataAbi.ID
	returnData := append(selector, packedData...)

	// Override the tracer's return error and data
	t.results.executionResult.Err = vm.ErrExecutionReverted
	t.results.executionResult.ReturnData = returnData
}
