package chain

import (
	"github.com/trailofbits/medusa/chain/types"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// testChainDeploymentsTracer implements TestChainTracer, capturing information regarding contract deployments and
// self-destructs. It is a special tracer that is used internally by each TestChain. It subscribes to its block-related
// events in order to power the TestChain's contract deployment related events.
type testChainDeploymentsTracer struct {
	// chain describes the TestChain to drive analysis and events for.
	chain *TestChain

	// results describes the results being currently captured.
	results []types.DeployedContractBytecodeChange

	// callDepth refers to the current EVM depth during tracing.
	callDepth uint64

	// evm refers to the EVM instance last captured.
	evm *vm.EVM

	// pendingCallFrames represents per-call-frame data deployment information being captured by the tracer.
	// This is committed as each call frame succeeds, so that contract deployments which later encountered an error
	// and reverted are not considered. The index of each element in the array represents its call frame depth.
	pendingCallFrames []*testChainDeploymentsTracerCallFrame

	// selfDestructDestroysCode indicates whether the SELFDESTRUCT opcode is configured to remove contract code.
	selfDestructDestroysCode bool
}

// testChainDeploymentsTracerCallFrame represents per-call-frame data traced by a testChainDeploymentsTracer.
type testChainDeploymentsTracerCallFrame struct {
	// results describes the results being currently captured.
	results []types.DeployedContractBytecodeChange
}

// newTestChainDeploymentsTracer creates a testChainDeploymentsTracer. It takes a TestChain, adds itself as a tracer for
// it, and powers deployed contract events.
func newTestChainDeploymentsTracer(chain *TestChain) *testChainDeploymentsTracer {
	tracer := &testChainDeploymentsTracer{
		chain:                    chain,
		selfDestructDestroysCode: true,
	}
	chain.AddTracer(tracer, true, false)
	chain.BlockMinedEventEmitter.Subscribe(tracer.onChainBlockMinedEvent)
	chain.BlocksRemovedEventEmitter.Subscribe(tracer.onChainBlocksRemovedEvent)
	return tracer
}

// onChainBlockMinedEvent is the event callback used when the chain mined a new block.
func (t *testChainDeploymentsTracer) onChainBlockMinedEvent(event BlockMinedEvent) error {
	// Obtain our message results for our mined block, and derive an event.
	results := event.Block.MessageResults()

	// Loop through all deployment changes stored in our call results and emit our TestChain event.
	var err error
	for i := 0; i < len(results); i++ {
		for _, deploymentChange := range results[i].ContractDeploymentChanges {
			// We emit the relevant event depending on the contract deployment change, as a block with
			// this execution result is being committed to chain.
			if deploymentChange.Creation {
				err = t.chain.ContractDeploymentAddedEventEmitter.Publish(ContractDeploymentsAddedEvent{
					Chain:    t.chain,
					Contract: deploymentChange.Contract,
				})
			} else if deploymentChange.Destroyed {
				err = t.chain.ContractDeploymentRemovedEventEmitter.Publish(ContractDeploymentsRemovedEvent{
					Chain:    t.chain,
					Contract: deploymentChange.Contract,
				})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// onChainBlocksRemovedEvent is the event callback used when the chain removes a block.
func (t *testChainDeploymentsTracer) onChainBlocksRemovedEvent(event BlocksRemovedEvent) error {
	// Loop through all deployment changes stored in our call results in revert order, as we are reverting and wish
	// to invert the events we emitted when these blocks were mined.
	var err error
	for i := len(event.Blocks) - 1; i >= 0; i-- {
		results := event.Blocks[i].MessageResults()
		for j := len(results) - 1; j >= 0; j-- {
			result := results[j]
			for k := len(result.ContractDeploymentChanges) - 1; k >= 0; k-- {
				deploymentChange := result.ContractDeploymentChanges[k]

				// We emit the *opposite* event depending on the contract deployment change, as a block with
				// this execution result is being reverted/removed from the chain.
				if deploymentChange.Creation {
					err = t.chain.ContractDeploymentRemovedEventEmitter.Publish(ContractDeploymentsRemovedEvent{
						Chain:    t.chain,
						Contract: deploymentChange.Contract,
					})
				} else if deploymentChange.Destroyed {
					err = t.chain.ContractDeploymentAddedEventEmitter.Publish(ContractDeploymentsAddedEvent{
						Chain:    t.chain,
						Contract: deploymentChange.Contract,
					})
				}
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *testChainDeploymentsTracer) CaptureTxStart(gasLimit uint64) {
	// Reset our capture state
	t.callDepth = 0
	t.results = make([]types.DeployedContractBytecodeChange, 0)
	t.pendingCallFrames = make([]*testChainDeploymentsTracerCallFrame, 0)
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *testChainDeploymentsTracer) CaptureTxEnd(restGas uint64) {

}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *testChainDeploymentsTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Store our evm reference
	t.evm = env

	// Create our call frame struct to track data for this initial entry call frame.
	callFrameData := &testChainDeploymentsTracerCallFrame{}
	t.pendingCallFrames = append(t.pendingCallFrames, callFrameData)

	// If this is a contract creation, record the `to` address as a pending deployment (if it succeeds upon exit,
	// we commit it).
	if create {
		callFrameData.results = append(callFrameData.results, types.DeployedContractBytecodeChange{
			Contract: &types.DeployedContractBytecode{
				Address:         to,
				InitBytecode:    input,
				RuntimeBytecode: nil,
			},
			Creation:       true,
			SelfDestructed: false,
			Destroyed:      false,
		})
	}
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *testChainDeploymentsTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	// Fetch runtime bytecode for all deployments in this frame which did not record one, before exiting.
	// We had to fetch it upon exit as it does not exist during creation of course.
	for _, contractChange := range t.pendingCallFrames[t.callDepth].results {
		if contractChange.Creation && contractChange.Contract.RuntimeBytecode == nil {
			contractChange.Contract.RuntimeBytecode = t.evm.StateDB.GetCode(contractChange.Contract.Address)
		}
	}

	// If we didn't encounter an error in this call frame, we're at the end, so we commit all results.
	if err == nil {
		t.results = append(t.results, t.pendingCallFrames[t.callDepth].results...)
	}

	// We're exiting the current frame, so remove our frame data.
	t.pendingCallFrames = t.pendingCallFrames[:t.callDepth]
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *testChainDeploymentsTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Increase our call depth now that we're entering a new call frame.
	t.callDepth++

	// Create our call frame struct to track data for this initial entry call frame.
	callFrameData := &testChainDeploymentsTracerCallFrame{}
	t.pendingCallFrames = append(t.pendingCallFrames, callFrameData)

	// If this is a contract creation, record the `to` address as a pending deployment (if it succeeds upon exit,
	// we commit it).
	if typ == vm.CREATE || typ == vm.CREATE2 {
		callFrameData.results = append(callFrameData.results, types.DeployedContractBytecodeChange{
			Contract: &types.DeployedContractBytecode{
				Address:         to,
				InitBytecode:    input,
				RuntimeBytecode: nil,
			},
			Creation:       true,
			SelfDestructed: false,
			Destroyed:      false,
		})
	}
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *testChainDeploymentsTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// Fetch runtime bytecode for all deployments in this frame which did not record one, before exiting.
	// We had to fetch it upon exit as it does not exist during creation of course.
	for _, contractChange := range t.pendingCallFrames[t.callDepth].results {
		if contractChange.Creation && contractChange.Contract.RuntimeBytecode == nil {
			contractChange.Contract.RuntimeBytecode = t.evm.StateDB.GetCode(contractChange.Contract.Address)
		}
	}

	// If we didn't encounter an error in this call frame, we push our captured data up one frame.
	if err == nil {
		t.pendingCallFrames[t.callDepth-1].results = append(t.pendingCallFrames[t.callDepth-1].results, t.pendingCallFrames[t.callDepth].results...)
	}

	// We're exiting the current frame, so remove our frame data.
	t.pendingCallFrames = t.pendingCallFrames[:t.callDepth]

	// Decrease our call depth now that we've exited a call frame.
	t.callDepth--
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *testChainDeploymentsTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// If we encounter a SELFDESTRUCT operation, record the change to our contract in our results.
	if op == vm.SELFDESTRUCT {
		callFrameData := t.pendingCallFrames[t.callDepth]
		callFrameData.results = append(callFrameData.results, types.DeployedContractBytecodeChange{
			Contract: &types.DeployedContractBytecode{
				Address:         *scope.Contract.CodeAddr,
				InitBytecode:    nil,
				RuntimeBytecode: t.evm.StateDB.GetCode(*scope.Contract.CodeAddr),
			},
			Creation:       false,
			SelfDestructed: true,
			Destroyed:      t.selfDestructDestroysCode,
		})
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *testChainDeploymentsTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {

}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *testChainDeploymentsTracer) CaptureTxEndSetAdditionalResults(results *types.CallMessageResults) {
	// Set our results. This is an internal tracer used by the test chain, so we don't need to use the
	// "additional results" field as other tracers might, we instead populate the field explicitly defined.
	results.ContractDeploymentChanges = t.results
}
