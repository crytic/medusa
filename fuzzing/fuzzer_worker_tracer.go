package fuzzing

import (
	"github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// fuzzerTracer implements vm.EVMLogger to collect information such as coverage maps
// for fuzzing campaigns from EVM execution traces.
type fuzzerTracer struct {
	// config options
	CoverageEnabled bool // enables coverage collection

	// tracing results
	coverageMaps *types.CoverageMaps
	vmReturnData []byte
	vmError      error
	gasLimit     uint64
	gasUsed      uint64
}

// newFuzzerTracer returns a new execution tracer for the fuzzer
func newFuzzerTracer(coverageEnabled bool) *fuzzerTracer {
	tracer := &fuzzerTracer{
		CoverageEnabled: coverageEnabled,
		coverageMaps:    types.NewCoverageMaps(),
	}
	return tracer
}

// VMError returns any EVM error which occurred during execution tracing.
func (t *fuzzerTracer) VMError() error {
	return t.vmError
}

// VMReturnData returns any EVM return data obtained from execution tracing.
func (t *fuzzerTracer) VMReturnData() []byte {
	return t.vmReturnData
}

// CoverageMaps returns the coverage maps collected by this tracer.
func (t *fuzzerTracer) CoverageMaps() *types.CoverageMaps {
	return t.coverageMaps
}

// Reset clears the state of the fuzzerTracer.
func (t *fuzzerTracer) Reset() {
	t.coverageMaps.Reset()
	t.vmReturnData = nil
	t.vmError = nil
	t.gasLimit = 0
	t.gasUsed = 0
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *fuzzerTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *fuzzerTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// If coverage is enabled and the code is not nil, collect the coverage.
	if t.CoverageEnabled && scope.Contract.Code != nil {
		// Ensure we have a coverage map for this code address, otherwise create one.
		_, err := t.coverageMaps.SetCoveredAt(scope.Contract.CodeHash, len(scope.Contract.Code), pc)
		if err != nil {
			panic("error occurred when setting coverage during execution trace: " + err.Error())
		}
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *fuzzerTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *fuzzerTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	t.vmReturnData = output
	t.vmError = err
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *fuzzerTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *fuzzerTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *fuzzerTracer) CaptureTxStart(gasLimit uint64) {
	t.gasLimit = gasLimit
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *fuzzerTracer) CaptureTxEnd(restGas uint64) {
	t.gasUsed = t.gasLimit - restGas
}
