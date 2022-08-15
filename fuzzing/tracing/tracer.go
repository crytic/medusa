package tracing

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// FuzzerTracer implements vm.EVMLogger to collect information such as coverage maps
// for fuzzing campaigns from EVM execution traces.
type FuzzerTracer struct {
	// config options
	CoverageEnabled bool // enables coverage collection

	// tracing results
	coverageMaps map[common.Address]*CoverageMap
	returnData   []byte
	vmErr        error
	gasLimit     uint64
	gasUsed      uint64
}

// NewFuzzerTracer returns a new execution tracer for the fuzzer
func NewFuzzerTracer(coverageEnabled bool) *FuzzerTracer {
	tracer := &FuzzerTracer{
		CoverageEnabled: coverageEnabled,
		coverageMaps:    make(map[common.Address]*CoverageMap),
	}
	return tracer
}

// Error returns any EVM error which occurred during execution tracing.
func (t *FuzzerTracer) Error() error {
	return t.vmErr
}

// ReturnData returns any EVM return data obtained from execution tracing.
func (t *FuzzerTracer) ReturnData() []byte {
	return t.returnData
}

// Reset clears the state of the FuzzerTracer.
func (t *FuzzerTracer) Reset() {
	t.coverageMaps = make(map[common.Address]*CoverageMap)
	t.returnData = nil
	t.vmErr = nil
	t.gasLimit = 0
	t.gasUsed = 0
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *FuzzerTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *FuzzerTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	var err error

	// If the code is not nil, we collect coverage
	if t.CoverageEnabled && scope.Contract.Code != nil {

		// Obtain the contract code address we're executing
		codeAddress := scope.Contract.CodeAddr

		// Ensure we have a coverage map for this code address, otherwise create one.
		coverageMap, coverageMapExisted := t.coverageMaps[*codeAddress]
		if !coverageMapExisted {
			// Create a coverage map and set it
			coverageMap, err = NewCoverageMap(scope.Contract.CodeHash, len(scope.Contract.Code))
			if err != nil {
				panic("failed to create coverage map: " + err.Error())
			}
			t.coverageMaps[*codeAddress] = coverageMap
		}

		// TODO: Verify our code has not changed at this address since we started coverage collection
		//  This can happen when a contract is being deployed to an address as the code hash will first match
		//  the init byte code, and will later match the runtime byte code.
		if coverageMap.codeHash != scope.Contract.CodeHash {
			// TODO: Signal that the coverage map has changed.
		}

		// TODO: Update our coverage map and state regarding if new code was executed.
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *FuzzerTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *FuzzerTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	t.returnData = output
	t.vmErr = err
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *FuzzerTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *FuzzerTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *FuzzerTracer) CaptureTxStart(gasLimit uint64) {
	t.gasLimit = gasLimit
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *FuzzerTracer) CaptureTxEnd(restGas uint64) {
	t.gasUsed = t.gasLimit - restGas
}
