package chain

import (
	"math/big"

	"github.com/crytic/medusa/chain/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	coretypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"golang.org/x/exp/slices"
)

// TestChainTracer is an extended tracers.Tracer which can be used with a TestChain to store any captured
// information within call results, recorded in each block produced.
type TestChainTracer struct {
	// tracers.Tracer is extended by this logger.
	*tracers.Tracer

	// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
	// tracer is used during transaction execution (block creation), the results can later be queried from the block.
	// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
	CaptureTxEndSetAdditionalResults func(results *types.MessageResults)
}

// TestChainTracerRouter acts as a tracers.Tracer, allowing multiple tracers to be used in
// place of one. When this tracer receives callback, it calls upon its underlying tracers.
type TestChainTracerRouter struct {
	// tracers refers to the internally recorded tracers.Tracer instances to route all calls to.
	tracers      []*TestChainTracer
	nativeTracer *TestChainTracer
}

// NewTestChainTracerRouter returns a new TestChainTracerRouter instance with no registered tracers.
func NewTestChainTracerRouter() *TestChainTracerRouter {
	tracer := &TestChainTracerRouter{
		tracers: make([]*TestChainTracer, 0),
	}
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
func (t *TestChainTracerRouter) NativeTracer() *TestChainTracer {
	return t.nativeTracer
}

// AddTracer adds a TestChainTracer to the TestChainTracerRouter so that all other tracing.Hooks calls are forwarded.
// are forwarded to it.
func (t *TestChainTracerRouter) AddTracer(tracer *TestChainTracer) {
	t.AddTracers(tracer)
}

// AddTracers adds TestChainTracers to the TestChainTracerRouter so that all other tracing.Hooks calls are forwarded.
func (t *TestChainTracerRouter) AddTracers(tracers ...*TestChainTracer) {
	t.tracers = append(t.tracers, tracers...)
}

// Tracers returns the tracers.Tracer instances added to the TestChainTracerRouter.
func (t *TestChainTracerRouter) Tracers() []*TestChainTracer {
	return slices.Clone(t.tracers)
}

// OnTxStart is called upon the start of transaction execution, as defined by tracers.Tracer.
func (t *TestChainTracerRouter) OnTxStart(vm *tracing.VMContext, tx *coretypes.Transaction, from common.Address) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		if tracer.OnTxStart != nil {
			tracer.OnTxStart(vm, tx, from)
		}
	}
}

// OnTxEnd is called upon the end of transaction execution, as defined by tracers.Tracer.
func (t *TestChainTracerRouter) OnTxEnd(receipt *coretypes.Receipt, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		if tracer.OnTxEnd != nil {
			tracer.OnTxEnd(receipt, err)
		}
	}
}

// OnEnter initializes the tracing operation for the top of a call frame, as defined by tracers.Tracer.
func (t *TestChainTracerRouter) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		if tracer.OnEnter != nil {
			tracer.OnEnter(depth, typ, from, to, input, gas, value)
		}
	}
}

// OnExit is called after a call to finalize tracing completes for the top of a call frame, as defined by tracers.Tracer.
func (t *TestChainTracerRouter) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		if tracer.OnExit != nil {
			tracer.OnExit(depth, output, gasUsed, err, reverted)
		}
	}
}

// OnOpcode records data from an EVM state update, as defined by tracers.Tracer.
func (t *TestChainTracerRouter) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		if tracer.OnOpcode != nil {

			tracer.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
		}
	}
}

// OnFault records an execution fault, as defined by tracers.Tracer.
func (t *TestChainTracerRouter) OnFault(pc uint64, op byte, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		if tracer.OnFault != nil {
			tracer.OnFault(pc, op, gas, cost, scope, depth, err)
		}
	}
}

func (t *TestChainTracerRouter) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	// Call the underlying method for each registered tracer.

	for _, tracer := range t.tracers {
		if tracer.OnCodeChange != nil {
			tracer.OnCodeChange(a, prevCodeHash, prev, codeHash, code)
		}
	}

}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *TestChainTracerRouter) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Call the underlying method for each registered tracer.
	for _, tracer := range t.tracers {
		if tracer.CaptureTxEndSetAdditionalResults != nil {
			tracer.CaptureTxEndSetAdditionalResults(results)
		}
	}
}
