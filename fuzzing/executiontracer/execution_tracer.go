package executiontracer

import (
	"math/big"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/slices"
)

// CallWithExecutionTrace obtains an execution trace for a given call, on the provided chain, using the state
// provided. If a nil state is provided, the current chain state will be used.
// Returns the ExecutionTrace for the call or an error if one occurs.
func CallWithExecutionTrace(chain *chain.TestChain, contractDefinitions contracts.Contracts, msg *core.Message, state *state.StateDB) (*core.ExecutionResult, *ExecutionTrace, error) {
	// Create an execution tracer
	executionTracer := NewExecutionTracer(contractDefinitions, chain.CheatCodeContracts())

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

	// cheatCodeContracts  represents the cheat code contract definitions to match for execution traces.
	cheatCodeContracts map[common.Address]*chain.CheatCodeContract

	// onNextCaptureState refers to methods which should be executed the next time CaptureState executes.
	// CaptureState is called prior to execution of an instruction. This allows actions to be performed
	// after some state is captured, on the next state capture (e.g. detecting a log instruction, but
	// using this structure to execute code later once the log is committed).
	onNextCaptureState []func()
}

// NewExecutionTracer creates a ExecutionTracer and returns it.
func NewExecutionTracer(contractDefinitions contracts.Contracts, cheatCodeContracts map[common.Address]*chain.CheatCodeContract) *ExecutionTracer {
	tracer := &ExecutionTracer{
		contractDefinitions: contractDefinitions,
		cheatCodeContracts:  cheatCodeContracts,
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
	t.trace = newExecutionTrace(t.contractDefinitions)
	t.currentCallFrame = nil
	t.onNextCaptureState = nil
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureTxEnd(restGas uint64) {

}

// resolveConstructorArgs resolves previously unresolved constructor argument ABI data from the call data, if
// the call frame provided represents a contract deployment.
func (t *ExecutionTracer) resolveCallFrameConstructorArgs(callFrame *CallFrame, contract *contracts.Contract) {
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
func (t *ExecutionTracer) resolveCallFrameContractDefinitions(callFrame *CallFrame) {
	// Try to resolve contract definitions for "to" address
	if callFrame.ToContractAbi == nil {
		// Try to resolve definitions from cheat code contracts
		if cheatCodeContract, ok := t.cheatCodeContracts[callFrame.ToAddress]; ok {
			callFrame.ToContractName = cheatCodeContract.Name()
			callFrame.ToContractAbi = cheatCodeContract.Abi()
			callFrame.ExecutedCode = true
		} else {
			// Try to resolve definitions from compiled contracts
			toContract := t.contractDefinitions.MatchBytecode(callFrame.ToInitBytecode, callFrame.ToRuntimeBytecode)
			if toContract != nil {
				callFrame.ToContractName = toContract.Name()
				callFrame.ToContractAbi = &toContract.CompiledContract().Abi
				t.resolveCallFrameConstructorArgs(callFrame, toContract)

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
		if cheatCodeContract, ok := t.cheatCodeContracts[callFrame.CodeAddress]; ok {
			callFrame.CodeContractName = cheatCodeContract.Name()
			callFrame.CodeContractAbi = cheatCodeContract.Abi()
			callFrame.ExecutedCode = true
		} else {
			// Try to resolve definitions from compiled contracts
			codeContract := t.contractDefinitions.MatchBytecode(nil, callFrame.CodeRuntimeBytecode)
			if codeContract != nil {
				callFrame.CodeContractName = codeContract.Name()
				callFrame.CodeContractAbi = &codeContract.CompiledContract().Abi
			}
		}
	}
}

// captureEnteredCallFrame is a helper method used when a new call frame is entered to record information about it.
func (t *ExecutionTracer) captureEnteredCallFrame(fromAddress common.Address, toAddress common.Address, inputData []byte, isContractCreation bool, value *big.Int) {
	// Create our call frame struct to track data for this call frame we entered.
	callFrameData := &CallFrame{
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
		ParentCallFrame:     t.currentCallFrame,
	}

	// If this is a contract creation, set the init bytecode for this call frame to the input data.
	if isContractCreation {
		callFrameData.ToInitBytecode = inputData
	}

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
	if t.currentCallFrame.ToRuntimeBytecode == nil {
		// As long as this isn't a failed contract creation, we should be able to fetch "to" byte code on exit.
		if !t.currentCallFrame.IsContractCreation() || err == nil {
			t.currentCallFrame.ToRuntimeBytecode = t.evm.StateDB.GetCode(t.currentCallFrame.ToAddress)
		}
	}
	if t.currentCallFrame.CodeRuntimeBytecode == nil {
		// Optimization: If the "to" and "code" addresses match, we can simply set our "code" already fetched "to"
		// runtime bytecode.
		if t.currentCallFrame.CodeAddress == t.currentCallFrame.ToAddress {
			t.currentCallFrame.CodeRuntimeBytecode = t.currentCallFrame.ToRuntimeBytecode
		} else {
			t.currentCallFrame.CodeRuntimeBytecode = t.evm.StateDB.GetCode(t.currentCallFrame.CodeAddress)
		}
	}

	// Resolve our contract definitions on the call frame data, if they have not been.
	t.resolveCallFrameContractDefinitions(t.currentCallFrame)

	// Set our information for this call frame
	t.currentCallFrame.ReturnData = slices.Clone(output)
	t.currentCallFrame.ReturnError = err

	// We're exiting the current frame, so set our current call frame to the parent
	t.currentCallFrame = t.currentCallFrame.ParentCallFrame
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *ExecutionTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Store our evm reference
	t.evm = env

	// Capture that a new call frame was entered.
	t.captureEnteredCallFrame(from, to, input, create, value)
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
	t.captureEnteredCallFrame(from, to, input, typ == vm.CREATE || typ == vm.CREATE2, value)
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

	// Now that we have executed some code, we have access to the VM scope. From this, we can populate more
	// information about our call frame. If this is a delegate or proxy call, the sender/to/code addresses should
	// be appropriately represented in this structure. The information populated earlier on frame enter represents
	// the raw call data, before delegate transformations are applied, etc.
	if !t.currentCallFrame.ExecutedCode {
		t.currentCallFrame.SenderAddress = scope.Contract.CallerAddress
		t.currentCallFrame.ToAddress = scope.Contract.Address()
		if scope.Contract.CodeAddr != nil {
			t.currentCallFrame.CodeAddress = *scope.Contract.CodeAddr
		}

		// Mark code as having executed in this scope, so we don't set these values again (as cheat codes may affect it).
		// We also want to know if a given call scope executed code, or simply represented a value transfer call.
		t.currentCallFrame.ExecutedCode = true
	}

	// If we encounter a SELFDESTRUCT operation, record the operation.
	if op == vm.SELFDESTRUCT {
		t.currentCallFrame.SelfDestructed = true
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
