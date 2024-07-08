package chain

import (
	"math/big"

	"github.com/crytic/medusa/chain/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	coretypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params/forks"
)

// testChainDeploymentsTracer implements TestChainTracer, capturing information regarding contract deployments and
// self-destructs. It is a special tracer that is used internally by each TestChain. It subscribes to its block-related
// events in order to power the TestChain's contract deployment related events.
type testChainDeploymentsTracer struct {
	// results describes the results being currently captured.
	results []types.DeployedContractBytecodeChange

	// callDepth refers to the current EVM depth during tracing.
	callDepth int

	// evm refers to the last tracing.VMContext captured.
	evmContext *tracing.VMContext

	// pendingCallFrames represents per-call-frame data deployment information being captured by the tracer.
	// This is committed as each call frame succeeds, so that contract deployments which later encountered an error
	// and reverted are not considered. The index of each element in the array represents its call frame depth.
	pendingCallFrames []*testChainDeploymentsTracerCallFrame
}

// testChainDeploymentsTracerCallFrame represents per-call-frame data traced by a testChainDeploymentsTracer.
type testChainDeploymentsTracerCallFrame struct {
	// results describes the results being currently captured.
	results []types.DeployedContractBytecodeChange
}

// newTestChainDeploymentsTracer creates a testChainDeploymentsTracer
func newTestChainDeploymentsTracer() *TestChainTracer {
	t := &testChainDeploymentsTracer{}
	tracer := &TestChainTracer{&tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart: t.OnTxStart,
			// OnTxEnd:   t.OnTxEnd,
			// OnEnter:      t.OnEnter,
			// OnExit:       t.OnExit,
			OnOpcode:     t.OnOpcode,
			OnCodeChange: t.OnCodeChange,
		},
	}, t.CaptureTxEndSetAdditionalResults}
	return tracer
}

// CaptureTxStart is called upon the start of transaction execution, as defined by tracers.Tracer.
func (t *testChainDeploymentsTracer) OnTxStart(vm *tracing.VMContext, tx *coretypes.Transaction, from common.Address) {
	// Reset our capture state
	t.callDepth = 0
	t.results = make([]types.DeployedContractBytecodeChange, 0)
	t.pendingCallFrames = make([]*testChainDeploymentsTracerCallFrame, 0)
	// Store our evm reference
	t.evmContext = vm
}

// // OnCodeChange initializes the tracing operation for the top of a call frame, as defined by tracers.Tracer.
func (t *testChainDeploymentsTracer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	// Create our call frame struct to track data for this initial entry call frame.
	callFrameData := &testChainDeploymentsTracerCallFrame{}
	t.pendingCallFrames = append(t.pendingCallFrames, callFrameData)
	// TODO check that we aren't double counting dynamic creations
	t.results = append(t.results, types.DeployedContractBytecodeChange{
		Contract: &types.DeployedContractBytecode{
			Address:         a,
			InitBytecode:    nil,
			RuntimeBytecode: code,
		},
		Creation:        true,
		DynamicCreation: false,
		SelfDestructed:  false,
		Destroyed:       false,
	})
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by tracers.Tracer.
func (t *testChainDeploymentsTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	// // Fetch runtime bytecode for all deployments in this frame which did not record one, before exiting.
	// // We had to fetch it upon exit as it does not exist during creation of course.
	// for _, contractChange := range t.pendingCallFrames[t.callDepth].results {
	// 	if contractChange.Creation && contractChange.Contract.RuntimeBytecode == nil {
	// 		contractChange.Contract.RuntimeBytecode = t.evmContext.StateDB.GetCode(contractChange.Contract.Address)
	// 	}
	// }

	// If we didn't encounter an error in this call frame, we're at the end, so we commit all results.
	if err == nil && !reverted {
		t.results = append(t.results, t.pendingCallFrames[t.callDepth].results...)
	}

	// // We're exiting the current frame, so remove our frame data.
	// t.pendingCallFrames = t.pendingCallFrames[:t.callDepth]
}

// CaptureEnter is called upon entering of the call frame, as defined by tracers.Tracer.
func (t *testChainDeploymentsTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Increase our call depth now that we're entering a new call frame.
	if depth == 0 {
		return
	}
	t.callDepth = depth

	// Create our call frame struct to track data for this initial entry call frame.
	// callFrameData := &testChainDeploymentsTracerCallFrame{}
	// t.pendingCallFrames = append(t.pendingCallFrames, callFrameData)

	// If this is a contract creation, record the `to` address as a pending deployment (if it succeeds upon exit,
	// we commit it).
	if typ == byte(vm.CREATE) || typ == byte(vm.CREATE2) {
		t.results = append(t.results, types.DeployedContractBytecodeChange{
			Contract: &types.DeployedContractBytecode{
				Address:         to,
				InitBytecode:    input,
				RuntimeBytecode: nil,
			},
			Creation:        true,
			DynamicCreation: true,
			SelfDestructed:  false,
			Destroyed:       false,
		})
	}
}

// // CaptureExit is called upon exiting of the call frame, as defined by tracers.Tracer.
// func (t *testChainDeploymentsTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
// 	if depth == 0 {
// 		return
// 	}

// 	// // Fetch runtime bytecode for all deployments in this frame which did not record one, before exiting.
// 	// // We had to fetch it upon exit as it does not exist during creation of course.
// 	// for _, contractChange := range t.pendingCallFrames[t.callDepth].results {
// 	// 	if contractChange.Creation && contractChange.Contract.RuntimeBytecode == nil {
// 	// 		contractChange.Contract.RuntimeBytecode = t.evmContext.StateDB.GetCode(contractChange.Contract.Address)
// 	// 	}
// 	// }

// 	// If we didn't encounter an error in this call frame, we push our captured data up one frame.
// 	if err == nil {
// 		t.pendingCallFrames[t.callDepth-1].results = append(t.pendingCallFrames[t.callDepth-1].results, t.pendingCallFrames[t.callDepth].results...)
// 	}

// 	// We're exiting the current frame, so remove our frame data.
// 	t.pendingCallFrames = t.pendingCallFrames[:t.callDepth]

// 	// Decrease our call depth now that we've exited a call frame.
// 	t.callDepth--
// 	fmt.Println("OnExit", t.callDepth, depth)

// }

// CaptureState records data from an EVM state update, as defined by tracers.Tracer.
func (t *testChainDeploymentsTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	// If we encounter a SELFDESTRUCT operation, record the change to our contract in our results.
	if op == byte(vm.SELFDESTRUCT) {
		addr := scope.Address()
		code := t.evmContext.StateDB.GetCode(addr)
		t.results = append(t.results, types.DeployedContractBytecodeChange{
			Contract: &types.DeployedContractBytecode{
				Address:         addr,
				InitBytecode:    nil,
				RuntimeBytecode: code,
			},
			Creation:        false,
			DynamicCreation: false,
			SelfDestructed:  true,
			// Check if this is a new contract (not previously deployed and self destructed).
			// https://github.com/ethereum/go-ethereum/blob/8d42e115b1cae4f09fd02b71c06ec9c85f22ad4f/core/state/statedb.go#L504-L506
			Destroyed: t.evmContext.ChainConfig.LatestFork(t.evmContext.Time) < forks.Cancun || !t.evmContext.StateDB.Exist(addr),
		})
	}
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *testChainDeploymentsTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Set our results. This is an internal tracer used by the test chain, so we don't need to use the
	// "additional results" field as other tracers might, we instead populate the field explicitly defined.
	results.ContractDeploymentChanges = t.results
}
