package valuegenerationtracer

import (
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	coretypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"golang.org/x/exp/slices"
	"math/big"
)

// valueGenerationTracerResultsKey describes the key to use when storing tracer results in call message results, or when
// querying them.
const valueGenerationTracerResultsKey = "ValueGenerationTracerResults"

// ValueGenerationTrace contains information about the values generated during the execution of a given message on the
// EVM.
type ValueGenerationTrace struct {
	// transactionOutputValues holds interesting values that were generated during EVM execution
	transactionOutputValues []any
}

// ValueGenerationTracer records value information into a ValueGenerationTrace. It contains information about each
// call frame that was entered and exited, and their associated contract definitions.
type ValueGenerationTracer struct {
	// evm refers to the EVM instance last captured.
	evmContext *tracing.VMContext

	// trace represents the current execution trace captured by this tracer.
	trace *ValueGenerationTrace

	// currentCallFrame references the current call frame being traced.
	currentCallFrame *CallFrame

	// contractDefinitions represents the contract definitions to match for execution traces.
	contractDefinitions contracts.Contracts

	// nativeTracer is the underlying tracer used to capture EVM execution.
	nativeTracer *chain.TestChainTracer
}

// NativeTracer returns the underlying TestChainTracer.
func (t *ValueGenerationTracer) NativeTracer() *chain.TestChainTracer {
	return t.nativeTracer
}

// NewValueGenerationTracer creates a new ValueGenerationTracer and returns it
func NewValueGenerationTracer(contractDefinitions contracts.Contracts) *ValueGenerationTracer {
	tracer := &ValueGenerationTracer{
		contractDefinitions: contractDefinitions,
	}

	innerTracer := &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart: tracer.OnTxStart,
			OnEnter:   tracer.OnEnter,
			OnTxEnd:   tracer.OnTxEnd,
			OnExit:    tracer.OnExit,
			OnOpcode:  tracer.OnOpcode,
			OnLog:     tracer.OnLog,
		},
	}
	tracer.nativeTracer = &chain.TestChainTracer{Tracer: innerTracer, CaptureTxEndSetAdditionalResults: nil}
	return tracer
}

// newValueGenerationTrace creates a new ValueGenerationTrace and returns it
func newValueGenerationTrace() *ValueGenerationTrace {
	return &ValueGenerationTrace{
		transactionOutputValues: make([]any, 0),
	}
}

// OnTxStart is called upon the start of transaction execution, as defined by tracers.Tracer.
func (t *ValueGenerationTracer) OnTxStart(vm *tracing.VMContext, tx *coretypes.Transaction, from common.Address) {
	t.trace = newValueGenerationTrace()
	t.currentCallFrame = nil
	// Store our evm reference
	t.evmContext = vm
}

// OnTxEnd is called upon the end of transaction execution, as defined by tracers.Tracer.
func (t *ValueGenerationTracer) OnTxEnd(receipt *coretypes.Receipt, err error) {

}

// OnEnter initializes the tracing operation for the top of a call frame, as defined by tracers.Tracer.
func (t *ValueGenerationTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.onEnteredCallFrame(to, input, typ == byte(vm.CREATE) || typ == byte(vm.CREATE2), value)
}

// OnExit is called after a call to finalize tracing completes for the top of a call frame, as defined by tracers.Tracer.
func (t *ValueGenerationTracer) OnExit(depth int, output []byte, used uint64, err error, reverted bool) {
	// Update call frame information and capture any emitted event and/or return values from the call frame
	t.onExitedCallFrame(output, err)
}

// OnOpcode records data from an EVM state update, as defined by tracers.Tracer.
func (t *ValueGenerationTracer) OnOpcode(pc uint64, op byte, gas uint64, cost uint64, scope tracing.OpContext, data []byte, depth int, err error) {
	// Now that we have executed some code, we have access to the VM scope. From this, we can populate more
	// information about our call frame. If this is a delegate or proxy call, the to/code addresses should
	// be appropriately represented in this structure. The information populated earlier on frame enter represents
	// the raw call data, before delegate transformations are applied, etc.
	if !t.currentCallFrame.ExecutedCode {
		// This is not always the "to" address, but the current address e.g. for delegatecall.
		t.currentCallFrame.ToAddress = scope.Address()
		// Mark code as having executed in this scope, so we don't set these values again (as cheat codes may affect it).
		// We also want to know if a given call scope executed code, or simply represented a value transfer call.
		t.currentCallFrame.ExecutedCode = true
	}

	// TODO: look for RET opcode to get runtime values
}

// OnLog is triggered when a LOG operation is encountered during EVM execution, as defined by tracers.Tracer.
func (t *ValueGenerationTracer) OnLog(log *coretypes.Log) {
	// Append log to list of operations for this call frame
	t.currentCallFrame.Logs = append(t.currentCallFrame.Logs, log)
}

// onEnteredCallFrame is a helper method used when a new call frame is entered to record information about it.
func (t *ValueGenerationTracer) onEnteredCallFrame(toAddress common.Address, inputData []byte, isContractCreation bool, value *big.Int) {
	// Create our call frame struct to track data for this call frame we entered.
	callFrameData := &CallFrame{
		ToAddress:           toAddress,
		ToContractAbi:       nil,
		ToInitBytecode:      nil,
		ToRuntimeBytecode:   nil,
		CodeAddress:         toAddress, // Note: Set temporarily, overwritten if code executes (in CaptureState).
		CodeContractAbi:     nil,
		CodeRuntimeBytecode: nil,
		Logs:                make([]*coretypes.Log, 0),
		InputData:           slices.Clone(inputData),
		ReturnData:          nil,
		ExecutedCode:        false,
		ParentCallFrame:     t.currentCallFrame,
	}

	// If this is a contract creation, set the init bytecode for this call frame to the input data.
	if isContractCreation {
		callFrameData.ToInitBytecode = inputData
	}

	// Update our current call frame
	t.currentCallFrame = callFrameData
}

// onExitedCallFrame is a helper method used when a call frame is exited, to record information about it.
func (t *ValueGenerationTracer) onExitedCallFrame(output []byte, err error) {
	// If this was an initial deployment, now that we're exiting, we'll want to record the finally deployed bytecodes.
	if t.currentCallFrame.ToRuntimeBytecode == nil {
		// As long as this isn't a failed contract creation, we should be able to fetch "to" byte code on exit.
		if !t.currentCallFrame.IsContractCreation() || err == nil {
			t.currentCallFrame.ToRuntimeBytecode = t.evmContext.StateDB.GetCode(t.currentCallFrame.ToAddress)
		}
	}
	if t.currentCallFrame.CodeRuntimeBytecode == nil {
		// Optimization: If the "to" and "code" addresses match, we can simply set our "code" already fetched "to"
		// runtime bytecode.
		if t.currentCallFrame.CodeAddress == t.currentCallFrame.ToAddress {
			t.currentCallFrame.CodeRuntimeBytecode = t.currentCallFrame.ToRuntimeBytecode
		} else {
			t.currentCallFrame.CodeRuntimeBytecode = t.evmContext.StateDB.GetCode(t.currentCallFrame.CodeAddress)
		}
	}

	// Resolve our contract definitions on the call frame data, if they have not been.
	t.resolveCallFrameContractDefinitions(t.currentCallFrame)

	// Set return data for this call frame
	t.currentCallFrame.ReturnData = slices.Clone(output)

	// Append any event and return values from the call frame only if the code contract ABI is nil
	// TODO: Note this won't work if the value/event is returned/emitted from something like a library or cheatcode
	codeContractAbi := t.currentCallFrame.CodeContractAbi
	if codeContractAbi != nil {
		// Append event values. Note that we are appending event values even if an error was thrown
		for _, log := range t.currentCallFrame.Logs {
			if _, eventInputValues := abiutils.UnpackEventAndValues(codeContractAbi, log); len(eventInputValues) > 0 {
				t.trace.transactionOutputValues = append(t.trace.transactionOutputValues, eventInputValues...)
			}
		}

		// Append return values assuming no error was returned
		if method, _ := t.currentCallFrame.CodeContractAbi.MethodById(t.currentCallFrame.InputData); method != nil && err != nil {
			if outputValues, decodingError := method.Outputs.Unpack(t.currentCallFrame.ReturnData); decodingError != nil {
				t.trace.transactionOutputValues = append(t.trace.transactionOutputValues, outputValues...)
			}
		}
	}

	// We're exiting the current frame, so set our current call frame to the parent
	t.currentCallFrame = t.currentCallFrame.ParentCallFrame
}

// resolveCallFrameContractDefinitions resolves previously unresolved contract definitions for the To and Code addresses
// used within the provided call frame.
func (t *ValueGenerationTracer) resolveCallFrameContractDefinitions(callFrame *CallFrame) {
	// Try to resolve contract definitions for "to" address
	if callFrame.ToContractAbi == nil {
		// Try to resolve definitions from compiled contracts
		toContract := t.contractDefinitions.MatchBytecode(callFrame.ToInitBytecode, callFrame.ToRuntimeBytecode)
		if toContract != nil {
			callFrame.ToContractAbi = &toContract.CompiledContract().Abi

			// If this is a contract creation, set the code address to the address of the contract we just deployed.
			if callFrame.IsContractCreation() {
				callFrame.CodeContractAbi = &toContract.CompiledContract().Abi
			}
		}
	}

	// Try to resolve contract definitions for "code" address
	if callFrame.CodeContractAbi == nil {
		codeContract := t.contractDefinitions.MatchBytecode(nil, callFrame.CodeRuntimeBytecode)
		if codeContract != nil {
			callFrame.CodeContractAbi = &codeContract.CompiledContract().Abi
			callFrame.ExecutedCode = true
		}

	}
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *ValueGenerationTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	results.AdditionalResults[valueGenerationTracerResultsKey] = t.trace.transactionOutputValues
}
