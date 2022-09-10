package tracing

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// MultiTracer implements vm.EVMLogger and forwards all underlying calls to vm.EVMLoggers registered with it.
type MultiTracer struct {
	tracers []vm.EVMLogger
}

// NewMultiTracer returns a NewMultiTracer instance with no registered tracers.
func NewMultiTracer() *MultiTracer {
	return NewMultiTracerWithTracers(make([]vm.EVMLogger, 0))
}

// NewMultiTracerWithTracers returns a NewMultiTracer instance with the provided tracers registered upon initialization.
func NewMultiTracerWithTracers(tracers []vm.EVMLogger) *MultiTracer {
	return &MultiTracer{
		tracers: tracers,
	}
}

// RegisterTracer adds a vm.EVMLogger implementation to the MultiTracer so all other method calls are forwarded to it.
func (t *MultiTracer) RegisterTracer(tracer vm.EVMLogger) {
	t.tracers = append(t.tracers, tracer)
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *MultiTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureStart(env, from, to, create, input, gas, value)
	}
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *MultiTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureState(pc, op, gas, cost, scope, rData, depth, vmErr)
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *MultiTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureFault(pc, op, gas, cost, scope, depth, err)
	}
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *MultiTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureEnd(output, gasUsed, d, err)
	}
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *MultiTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureEnter(typ, from, to, input, gas, value)
	}
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *MultiTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureExit(output, gasUsed, err)
	}
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *MultiTracer) CaptureTxStart(gasLimit uint64) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureTxStart(gasLimit)
	}
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *MultiTracer) CaptureTxEnd(restGas uint64) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureTxEnd(restGas)
	}
}
