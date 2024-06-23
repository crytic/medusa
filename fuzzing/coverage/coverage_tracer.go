package coverage

import (
	"math/big"

	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/logging"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

// coverageTracerResultsKey describes the key to use when storing tracer results in call message results, or when
// querying them.
const coverageTracerResultsKey = "CoverageTracerResults"
const MAP_SIZE = 4096

// Assuming RW_SKIPPER_PERCT_IDX and RW_SKIPPER_AMT are some predefined constants
var RW_SKIPPER_PERCT_IDX = uint256.NewInt(100)
var RW_SKIPPER_AMT = uint256.NewInt(16)

// asU64 converts a big.Int to a uint64
func asU64(n *uint256.Int) uint64 {
	return n.Uint64()
}

func processRwKey(key *uint256.Int) int {
	if key.Cmp(RW_SKIPPER_PERCT_IDX) > 0 {
		key.Mod(key, RW_SKIPPER_AMT)
		key.Add(key, RW_SKIPPER_PERCT_IDX)
	}
	return int(asU64(key) % MAP_SIZE)
}

// u256ToU8 takes a big.Int, performs a right shift by 4 bits, converts to uint64,
// takes the result modulo 254, and returns it as a uint8.
func u256ToU8(key *uint256.Int) uint8 {
	shiftedKey := new(uint256.Int).Rsh(key, 4) // key >> 4
	asUint64 := shiftedKey.Uint64()            // Convert to uint64
	result := asUint64 % 254                   // Take modulo 254
	return uint8(result)                       // Convert to uint8
}

// GetCoverageTracerResults obtains CoverageMaps stored by a CoverageTracer from message results. This is nil if
// no CoverageMaps were recorded by a tracer (e.g. CoverageTracer was not attached during this message execution).
func GetCoverageTracerResults(messageResults *types.MessageResults) *CoverageMaps {
	// Try to obtain the results the tracer should've stored.
	if genericResult, ok := messageResults.AdditionalResults[coverageTracerResultsKey]; ok {
		if castedResult, ok := genericResult.(*CoverageMaps); ok {
			return castedResult
		}
	}

	// If we could not obtain them, return nil.
	return nil
}

// RemoveCoverageTracerResults removes CoverageMaps stored by a CoverageTracer from message results.
func RemoveCoverageTracerResults(messageResults *types.MessageResults) {
	delete(messageResults.AdditionalResults, coverageTracerResultsKey)
}

// CoverageTracer implements vm.EVMLogger to collect information such as coverage maps
// for fuzzing campaigns from EVM execution traces.
type CoverageTracer struct {
	// coverageMaps describes the execution coverage recorded. Call frames which errored are not recorded.
	coverageMaps *CoverageMaps

	// callFrameStates describes the state tracked by the tracer per call frame.
	callFrameStates []*coverageTracerCallFrameState

	// callDepth refers to the current EVM depth during tracing.
	callDepth uint64

	globalWriteMap [MAP_SIZE][4]bool

	interesting bool
}

// coverageTracerCallFrameState tracks state across call frames in the tracer.
type coverageTracerCallFrameState struct {
	// create indicates whether the current call frame is executing on init bytecode (deploying a contract).
	create bool

	// pendingCoverageMap describes the coverage maps recorded for this call frame.
	pendingCoverageMap *CoverageMaps

	// lookupHash describes the hash used to look up the ContractCoverageMap being updated in this frame.
	lookupHash *common.Hash

	readMap []bool

	writeMap []uint8
}

// NewCoverageTracer returns a new CoverageTracer.
func NewCoverageTracer() *CoverageTracer {
	tracer := &CoverageTracer{
		coverageMaps:    NewCoverageMaps(),
		callFrameStates: make([]*coverageTracerCallFrameState, 0),
	}
	return tracer
}

func (t *CoverageTracer) IsInteresting() bool {
	return t.interesting
}

// CaptureTxStart is called upon the start of transaction execution, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureTxStart(gasLimit uint64) {
	// Reset our call frame states
	t.callDepth = 0
	t.coverageMaps = NewCoverageMaps()
	t.callFrameStates = make([]*coverageTracerCallFrameState, 0)
	t.interesting = false
}

// CaptureTxEnd is called upon the end of transaction execution, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureTxEnd(restGas uint64) {
}

// CaptureStart initializes the tracing operation for the top of a call frame, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Create our state tracking struct for this frame.
	t.callFrameStates = append(t.callFrameStates, &coverageTracerCallFrameState{
		create:             create,
		pendingCoverageMap: NewCoverageMaps(),
		readMap:            make([]bool, MAP_SIZE),
		writeMap:           make([]uint8, MAP_SIZE),
	})
}

// CaptureEnd is called after a call to finalize tracing completes for the top of a call frame, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	// If we encountered an error in this call frame, mark all coverage as reverted.
	if err != nil {
		_, revertCoverageErr := t.callFrameStates[t.callDepth].pendingCoverageMap.RevertAll()
		if revertCoverageErr != nil {
			logging.GlobalLogger.Panic("Coverage tracer failed to update revert coverage map during capture end", revertCoverageErr)
		}
	}

	// Commit all our coverage maps up one call frame.
	_, _, coverageUpdateErr := t.coverageMaps.Update(t.callFrameStates[t.callDepth].pendingCoverageMap)

	interesting := false
	for i := 0; i < MAP_SIZE; i++ {
		if t.callFrameStates[t.callDepth].readMap[i] && t.callFrameStates[t.callDepth].writeMap[i] != 0 {
			category := 0
			if t.callFrameStates[t.callDepth].writeMap[i] < (2 << 2) {
				category = 0
			} else if t.callFrameStates[t.callDepth].writeMap[i] < (2 << 4) {
				category = 1
			} else if t.callFrameStates[t.callDepth].writeMap[i] < (2 << 6) {
				category = 2
			} else {
				category = 3
			}
			if !t.globalWriteMap[i%MAP_SIZE][category] {
				interesting = true
				t.globalWriteMap[i%MAP_SIZE][category] = true
			}
		}
	}
	if interesting {
		t.interesting = interesting
		// logging.GlobalLogger.Info("Interesting read/write pair found")
	}
	if coverageUpdateErr != nil {
		logging.GlobalLogger.Panic("Coverage tracer failed to update coverage map during capture end", coverageUpdateErr)
	}

	// Pop the state tracking struct for this call frame off the stack.
	t.callFrameStates = t.callFrameStates[:t.callDepth]
}

// CaptureEnter is called upon entering of the call frame, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Increase our call depth now that we're entering a new call frame.
	t.callDepth++

	// Create our state tracking struct for this frame.
	t.callFrameStates = append(t.callFrameStates, &coverageTracerCallFrameState{
		create:             typ == vm.CREATE || typ == vm.CREATE2,
		pendingCoverageMap: NewCoverageMaps(),
	})
}

// CaptureExit is called upon exiting of the call frame, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	// If we encountered an error in this call frame, mark all coverage as reverted.
	if err != nil {
		_, revertCoverageErr := t.callFrameStates[t.callDepth].pendingCoverageMap.RevertAll()
		if revertCoverageErr != nil {
			logging.GlobalLogger.Panic("Coverage tracer failed to update revert coverage map during capture exit", revertCoverageErr)
		}
	}

	// Commit all our coverage maps up one call frame.
	_, _, coverageUpdateErr := t.callFrameStates[t.callDepth-1].pendingCoverageMap.Update(t.callFrameStates[t.callDepth].pendingCoverageMap)
	if coverageUpdateErr != nil {
		logging.GlobalLogger.Panic("Coverage tracer failed to update coverage map during capture exit", coverageUpdateErr)
	}

	// Pop the state tracking struct for this call frame off the stack.
	t.callFrameStates = t.callFrameStates[:t.callDepth]

	// Decrease our call depth now that we've exited a call frame.
	t.callDepth--
}

// CaptureState records data from an EVM state update, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, vmErr error) {
	// Obtain our call frame state tracking struct
	callFrameState := t.callFrameStates[t.callDepth]

	// If there is code we're executing, collect coverage.
	if len(scope.Contract.Code) > 0 {
		// Obtain our contract coverage map lookup hash.
		if callFrameState.lookupHash == nil {
			lookupHash := getContractCoverageMapHash(scope.Contract.Code, callFrameState.create)
			callFrameState.lookupHash = &lookupHash
		}
		// Is SSTORE operation
		if op == vm.SSTORE {
			key := scope.Stack.Back(0)
			value := scope.Stack.Back(1)
			// logging.GlobalLogger.Info("SSTORE key: ", key, " value: ", value)
			callFrameState.writeMap[processRwKey(key)] = u256ToU8(value)
		}

		if op == vm.SLOAD {

			key := scope.Stack.Back(0)
			// logging.GlobalLogger.Info("SLOAD key: ", key)
			callFrameState.readMap[processRwKey(key)] = true

		}

		// Record coverage for this location in our map.
		_, coverageUpdateErr := callFrameState.pendingCoverageMap.SetAt(scope.Contract.Address(), *callFrameState.lookupHash, len(scope.Contract.Code), pc)
		if coverageUpdateErr != nil {
			logging.GlobalLogger.Panic("Coverage tracer failed to update coverage map while tracing state", coverageUpdateErr)
		}
	}
}

// CaptureFault records an execution fault, as defined by vm.EVMLogger.
func (t *CoverageTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *CoverageTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Store our tracer results.
	results.AdditionalResults[coverageTracerResultsKey] = t.coverageMaps
}
