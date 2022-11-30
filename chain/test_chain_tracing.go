package chain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/trailofbits/medusa/chain/types"
	"golang.org/x/exp/slices"
	"math/big"
	"time"
)

// TestChainTracer is an extended vm.EVMLogger which can be used with a TestChain to store any captured
// information within call results, recorded in each block produced.
type TestChainTracer interface {
	// EVMLogger is extended by this logger.
	vm.EVMLogger

	// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
	// tracer is used during transaction execution (block creation), the results can later be queried from the block.
	// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
	CaptureTxEndSetAdditionalResults(results *types.CallMessageResults)
}

// TestChainTracerRouter acts as a vm.EVMLogger or TestChainTracer splitter, allowing multiple tracers to be used in
// place of one. When this tracer receives callback, it calls upon its underlying tracers.
type TestChainTracerRouter struct {
	// tracers refers to the internally recorded vm.EVMLogger instances to route all calls to.
	tracers []vm.EVMLogger
}

// NewTestChainTracerRouter returns a new TestChainTracerRouter instance with no registered tracers.
func NewTestChainTracerRouter() *TestChainTracerRouter {
	return &TestChainTracerRouter{
		tracers: make([]vm.EVMLogger, 0),
	}
}

// AddTracer adds a vm.EVMLogger or TestChainTracer to the TestChainTracerRouter, so all vm.EVMLogger relates calls
// are forwarded to it.
func (t *TestChainTracerRouter) AddTracer(tracer vm.EVMLogger) {
	t.AddTracers(tracer)
}

// AddTracers adds vm.EVMLogger implementations to the TestChainTracerRouter so all other method calls are forwarded
// to  them.
func (t *TestChainTracerRouter) AddTracers(tracers ...vm.EVMLogger) {
	t.tracers = append(t.tracers, tracers...)
}

// Tracers returns the vm.EVMLogger instances added to the TestChainTracerRouter.
func (t *TestChainTracerRouter) Tracers() []vm.EVMLogger {
	return slices.Clone(t.tracers)
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *TestChainTracerRouter) CaptureTxStart(gasLimit uint64) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureTxStart(gasLimit)
	}
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *TestChainTracerRouter) CaptureTxEnd(restGas uint64) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureTxEnd(restGas)
	}
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *TestChainTracerRouter) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureStart(env, from, to, create, input, gas, value)
	}
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *TestChainTracerRouter) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureEnd(output, gasUsed, d, err)
	}
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *TestChainTracerRouter) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureEnter(typ, from, to, input, gas, value)
	}
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *TestChainTracerRouter) CaptureExit(output []byte, gasUsed uint64, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureExit(output, gasUsed, err)
	}
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *TestChainTracerRouter) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureState(pc, op, gas, cost, scope, rData, depth, vmErr)
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *TestChainTracerRouter) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureFault(pc, op, gas, cost, scope, depth, err)
	}
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *TestChainTracerRouter) CaptureTxEndSetAdditionalResults(results *types.CallMessageResults) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		// Try to cast each tracer to a TestChainTracer and forward the call to it.
		if testChainTracer, ok := tracer.(TestChainTracer); ok {
			testChainTracer.CaptureTxEndSetAdditionalResults(results)
		}
	}
}
