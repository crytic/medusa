package coverage

import (
	"fmt"
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// CoverageTracer implements vm.EVMLogger to collect information such as coverage maps
// for fuzzing campaigns from EVM execution traces.
type CoverageTracer struct {
	// coverageMaps describes the execution coverage recorded. Call frames which errored are not recorded.
	coverageMaps *CoverageMaps

	// pendingCoverageMaps describes the coverage maps per call frame of the transaction during its execution. This is
	// used to filter coverage only for call scopes which did not error/revert.
	pendingCoverageMaps []*CoverageMaps

	// callDepth refers to the current EVM depth during tracing.
	callDepth uint64

	// cachedCodeHashOriginal describes the code hash used to last store coverage.
	cachedCodeHashOriginal common.Hash
	// cachedCodeHashResolved describes the code hash used to store the last coverage map. If the contract metadata
	// code hash is embedded, then it is used. Otherwise, this refers to cachedCodeHashOriginal.
	cachedCodeHashResolved common.Hash
}

// NewCoverageTracer returns a new CoverageTracer.
func NewCoverageTracer() *CoverageTracer {
	tracer := &CoverageTracer{
		coverageMaps:        NewCoverageMaps(),
		pendingCoverageMaps: make([]*CoverageMaps, 0),
	}
	return tracer
}

// CoverageMaps returns the coverage maps describing execution coverage recorded by the tracer.
func (t *CoverageTracer) CoverageMaps() *CoverageMaps {
	return t.coverageMaps
}

// Reset clears the state of the CoverageTracer.
func (t *CoverageTracer) Reset() {
	t.callDepth = 0
	t.coverageMaps = NewCoverageMaps()
	t.pendingCoverageMaps = make([]*CoverageMaps, 0)
	t.cachedCodeHashOriginal = common.Hash{}
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureTxStart(gasLimit uint64) {
	// Reset our capture state
	t.callDepth = 0

	// Reset our list used to track maps we are currently populating, before committing them if the call frame succeeds.
	t.pendingCoverageMaps = make([]*CoverageMaps, 0)
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureTxEnd(restGas uint64) {
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Add a coverage map to track the initial call frame.
	t.pendingCoverageMaps = append(t.pendingCoverageMaps, NewCoverageMaps())
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	// If we didn't encounter an error in the end, we commit all our coverage maps to the final coverage map.
	// If we encountered an error, we reverted, so we don't consider them.
	if err == nil {
		_, coverageUpdateErr := t.coverageMaps.Update(t.pendingCoverageMaps[t.callDepth])
		if coverageUpdateErr != nil {
			panic(fmt.Sprintf("coverage tracer failed to update coverage map during capture end: %v", coverageUpdateErr))
		}
	}

	// Pop the pending maps for this frame off the stack.
	t.pendingCoverageMaps = t.pendingCoverageMaps[:t.callDepth]
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Increase our call depth now that we're entering a new call frame.
	t.callDepth++

	// Add a coverage map to track the newly entered call frame.
	t.pendingCoverageMaps = append(t.pendingCoverageMaps, NewCoverageMaps())
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// If we didn't encounter an error in the end, we commit all our coverage maps up one call frame.
	// If we encountered an error, we reverted, so we don't consider them.
	if err == nil {
		_, coverageUpdateErr := t.pendingCoverageMaps[t.callDepth-1].Update(t.pendingCoverageMaps[t.callDepth])
		if coverageUpdateErr != nil {
			panic(fmt.Sprintf("coverage tracer failed to update coverage map during capture exit: %v", coverageUpdateErr))
		}
	}

	// Pop the pending maps for this frame off the stack.
	t.pendingCoverageMaps = t.pendingCoverageMaps[:len(t.pendingCoverageMaps)-1]

	// Decrease our call depth now that we've exited a call frame.
	t.callDepth--
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// If there is code we're executing, collect coverage.
	if len(scope.Contract.Code) > 0 {
		// Verify the code hash is not zero (this is not a contract deployment being executed), prior to recovering
		// coverage.
		zeroHash := common.BigToHash(big.NewInt(0))
		if scope.Contract.CodeHash != zeroHash {
			// We record coverage maps under a code hash to merge coverage across different deployments of a contract.
			// We rely on the embedded contract metadata code hash if it is available, otherwise the immediate hash
			// for this code. Because this method is called for every instruction executed, we cache the resolved
			// code hash for performance reasons.
			if t.cachedCodeHashOriginal != scope.Contract.CodeHash {
				t.cachedCodeHashOriginal = scope.Contract.CodeHash
				t.cachedCodeHashResolved = t.cachedCodeHashOriginal
				if metadata := compilationTypes.ExtractContractMetadata(scope.Contract.Code); metadata != nil {
					if metadataHash := metadata.ExtractBytecodeHash(); metadataHash != nil {
						t.cachedCodeHashResolved = common.BytesToHash(metadataHash)
					}
				}
			}

			// Record our coverage for this code hash.
			_, coverageUpdateErr := t.pendingCoverageMaps[t.callDepth].SetCoveredAt(t.cachedCodeHashResolved, len(scope.Contract.Code), pc)
			if coverageUpdateErr != nil {
				panic(fmt.Sprintf("coverage tracer failed to update coverage map while tracing state: %v", coverageUpdateErr))
			}
		}
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}
