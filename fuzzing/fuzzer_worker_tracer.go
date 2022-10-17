package fuzzing

import (
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// fuzzerWorkerTracer implements vm.EVMLogger to collect information such as coverage maps
// for fuzzing campaigns from EVM execution traces.
type fuzzerWorkerTracer struct {
	// fuzzerWorker describes the parent worker which this tracer belongs to.
	fuzzerWorker *FuzzerWorker

	// tracing results
	capturedTransactionInfo []*fuzzerTracerTransactionInfo

	// cachedCodeHashOriginal describes the code hash used to last store coverage.
	cachedCodeHashOriginal common.Hash
	// cachedCodeHashResolved describes the code hash used to store the last coverage map. If the contract metadata
	// code hash is embedded, then it is used. Otherwise, this refers to cachedCodeHashOriginal.
	cachedCodeHashResolved common.Hash
}

type fuzzerTracerTransactionInfo struct {
	coverageMaps *types.CoverageMaps
	returnData   []byte
	err          error
	gasLimit     uint64
	gasUsed      uint64
}

// newFuzzerWorkerTracer returns a new execution tracer for the fuzzer
func newFuzzerWorkerTracer(fuzzerWorker *FuzzerWorker) *fuzzerWorkerTracer {
	tracer := &fuzzerWorkerTracer{
		fuzzerWorker:            fuzzerWorker,
		capturedTransactionInfo: make([]*fuzzerTracerTransactionInfo, 0),
	}
	return tracer
}

// ClearCoverageMaps clears the state of the fuzzerWorkerTracer.
func (t *fuzzerWorkerTracer) ClearCoverageMaps() {
	t.capturedTransactionInfo = make([]*fuzzerTracerTransactionInfo, 0)
	t.cachedCodeHashOriginal = common.Hash{}
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *fuzzerWorkerTracer) CaptureTxStart(gasLimit uint64) {
	transactionInfo := &fuzzerTracerTransactionInfo{
		coverageMaps: types.NewCoverageMaps(),
		returnData:   nil,
		err:          nil,
		gasLimit:     gasLimit,
		gasUsed:      0,
	}
	t.capturedTransactionInfo = append(t.capturedTransactionInfo, transactionInfo)
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *fuzzerWorkerTracer) CaptureTxEnd(restGas uint64) {
	// Obtain our current tracer transaction info
	txInfo := t.capturedTransactionInfo[len(t.capturedTransactionInfo)-1]

	// Update our gas used.
	txInfo.gasUsed = txInfo.gasLimit - restGas
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *fuzzerWorkerTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *fuzzerWorkerTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	// Obtain our current tracer transaction info
	txInfo := t.capturedTransactionInfo[len(t.capturedTransactionInfo)-1]

	// Update our vm return data and error.
	txInfo.returnData = output
	txInfo.err = err
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *fuzzerWorkerTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *fuzzerWorkerTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *fuzzerWorkerTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// Obtain our current tracer transaction info
	txInfo := t.capturedTransactionInfo[len(t.capturedTransactionInfo)-1]

	// If coverage is enabled, there is code we're executing, collect coverage.
	if t.fuzzerWorker.Fuzzer().Config().Fuzzing.CoverageEnabled && len(scope.Contract.Code) > 0 {
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
			_, err := txInfo.coverageMaps.SetCoveredAt(t.cachedCodeHashResolved, len(scope.Contract.Code), pc)
			if err != nil {
				panic("error occurred when setting coverage during execution trace: " + err.Error())
			}
		}
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *fuzzerWorkerTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}
