package chain

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// testChainTracer implements vm.EVMLogger and captures information used in the TestChain API.
type testChainTracer struct {
	// callDepth refers to the current EVM depth during tracing.
	callDepth uint64

	// deployedContractAddresses is a list of all contract deployments found from the current/last transaction
	// executed.
	deployedContractAddresses []common.Address

	// pendingContractAddresses is a stack of contract deployments that were made in a given call frame. They are
	// added to the stack when deployment is invoked, and moved up the stack as each call frame succeeds, before
	// finally being committed to deployedContractAddresses if the entire call stack succeeded.
	pendingContractAddresses [][]common.Address
}

// newTestChainTracer returns a new testChainTracer.
func newTestChainTracer() *testChainTracer {
	return &testChainTracer{}
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *testChainTracer) CaptureTxStart(gasLimit uint64) {
	// Reset our capture state
	t.callDepth = 0
	t.deployedContractAddresses = make([]common.Address, 0)
	t.pendingContractAddresses = make([][]common.Address, 0)
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *testChainTracer) CaptureTxEnd(restGas uint64) {

}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *testChainTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Create our stack item for pending deployments discovered at this call frame depth.
	t.pendingContractAddresses = append(t.pendingContractAddresses, make([]common.Address, 0))

	// If this is a contract creation, record the `to` address as a pending deployment (if it succeeds upon exit,
	// we commit it).
	if create {
		t.pendingContractAddresses[t.callDepth] = append(t.pendingContractAddresses[t.callDepth], to)
	}
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *testChainTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	// If we didn't encounter an error in the end, we commit all contract deployments, as we know they'll be committed.
	// If we encountered an error, we reverted, so we don't consider the deployments.
	if err == nil {
		t.deployedContractAddresses = append(t.deployedContractAddresses, t.pendingContractAddresses[t.callDepth]...)
	}

	// Pop the pending contracts for this frame off the stack.
	t.pendingContractAddresses = t.pendingContractAddresses[:len(t.pendingContractAddresses)-1]
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *testChainTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Increase our call depth now that we're entering a new call frame.
	t.callDepth++

	// Create our stack item for pending deployments discovered at this call frame depth.
	t.pendingContractAddresses = append(t.pendingContractAddresses, make([]common.Address, 0))

	// If this is a contract creation, record the `to` address as a pending deployment (if it succeeds upon exit,
	// we commit it).
	if typ == vm.CREATE || typ == vm.CREATE2 {
		t.pendingContractAddresses[t.callDepth] = append(t.pendingContractAddresses[t.callDepth], to)
	}
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *testChainTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// If we didn't encounter an error in this call frame, we're one step closer to this deployment being committed.
	// We push the responsibility up one call frame, as if the parent call succeeds, then this deployment won't be
	// reverted.
	if err == nil {
		t.pendingContractAddresses[t.callDepth-1] = append(t.pendingContractAddresses[t.callDepth-1], t.pendingContractAddresses[t.callDepth]...)
	}

	// Pop the pending contracts for this frame off the stack.
	t.pendingContractAddresses = t.pendingContractAddresses[:t.callDepth]

	// Decrease our call depth now that we've exited a call frame.
	t.callDepth--
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *testChainTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {

}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *testChainTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {

}
