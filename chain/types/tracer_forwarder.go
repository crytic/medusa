package types

import (
	"golang.org/x/exp/slices"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// TracerForwarder implements vm.EVMLogger and forwards all underlying calls to vm.EVMLoggers registered with it.
type TracerForwarder struct {
	// tracers refers to the internally recorded vm.EVMLogger instances to forward all calls to.
	tracers []vm.EVMLogger
}

// NewTracerForwarder returns a new TracerForwarder instance with no registered tracers.
func NewTracerForwarder() *TracerForwarder {
	return NewTracerForwarderWithTracers(make([]vm.EVMLogger, 0))
}

// NewTracerForwarderWithTracers returns a NewTracerForwarder instance with the provided tracers registered upon
// initialization.
func NewTracerForwarderWithTracers(tracers []vm.EVMLogger) *TracerForwarder {
	return &TracerForwarder{
		tracers: tracers,
	}
}

// AddTracer adds a vm.EVMLogger implementation to the TracerForwarder so all other method calls are forwarded to it.
func (t *TracerForwarder) AddTracer(tracer vm.EVMLogger) {
	t.AddTracers(tracer)
}

// AddTracers adds vm.EVMLogger implementations to the TracerForwarder so all other method calls are forwarded to them.
func (t *TracerForwarder) AddTracers(tracers ...vm.EVMLogger) {
	t.tracers = append(t.tracers, tracers...)
}

// Tracers returns the vm.EVMLogger instances added to the TracerForwarder.
func (t *TracerForwarder) Tracers() []vm.EVMLogger {
	return slices.Clone(t.tracers)
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *TracerForwarder) CaptureTxStart(gasLimit uint64) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureTxStart(gasLimit)
	}
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *TracerForwarder) CaptureTxEnd(restGas uint64) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureTxEnd(restGas)
	}
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *TracerForwarder) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureStart(env, from, to, create, input, gas, value)
	}
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *TracerForwarder) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureEnd(output, gasUsed, d, err)
	}
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *TracerForwarder) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureEnter(typ, from, to, input, gas, value)
	}
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *TracerForwarder) CaptureExit(output []byte, gasUsed uint64, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureExit(output, gasUsed, err)
	}
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *TracerForwarder) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureState(pc, op, gas, cost, scope, rData, depth, vmErr)
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *TracerForwarder) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		tracer.CaptureFault(pc, op, gas, cost, scope, depth, err)
	}
}
