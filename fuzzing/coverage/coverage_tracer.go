package coverage

import (
	"math/big"
	"math/bits"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/logging"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	coretypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

// coverageTracerResultsKey describes the key to use when storing tracer results in call message results, or when
// querying them.
const coverageTracerResultsKey = "CoverageTracerResults"

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

// CoverageTracer implements tracers.Tracer to collect information such as coverage maps
// for fuzzing campaigns from EVM execution traces.
type CoverageTracer struct {
	// coverageMaps describes the execution coverage recorded. Call frames which errored are not recorded.
	coverageMaps *CoverageMaps

	// callFrameStates describes the state tracked by the tracer per call frame.
	callFrameStates []*coverageTracerCallFrameState

	// callDepth refers to the current EVM depth during tracing.
	callDepth int

	evmContext *tracing.VMContext

	// nativeTracer is the underlying tracer used to capture EVM execution.
	nativeTracer *chain.TestChainTracer

	// codeHashCache is a cache for values returned by getContractCoverageMapHash,
	// so that this expensive calculation doesn't need to be done every opcode.
	// The [2] array is to differentiate between contract init (0) vs runtime (1),
	// since init vs runtime produces different results from getContractCoverageMapHash.
	// The Hash key is a contract's codehash, which uniquely identifies it.
	codeHashCache [2]map[common.Hash]common.Hash
}

// coverageTracerCallFrameState tracks state across call frames in the tracer.
type coverageTracerCallFrameState struct {
	// Some fields, such as address, are not initialized until OnOpcode is called.
	// initialized tracks whether or not this has happened yet.
	initialized bool

	// create indicates whether the current call frame is executing on init bytecode (deploying a contract).
	create bool

	// pendingCoverageMap describes the coverage maps recorded for this call frame.
	pendingCoverageMap *CoverageMaps

	// lookupHash describes the hash used to look up the ContractCoverageMap being updated in this frame.
	lookupHash *common.Hash

	// lastPC is the most recent PC that has been executed. Used for coverage tracking.
	lastPC uint64

	// address is used by OnOpcode to cache the result of scope.Address(), which is slow.
	// It records the address of the current contract.
	address common.Address

	// justJumped indicates whether or not the most recent instruction (the one indicated by lastPC) was JUMP/JUMPI.
	justJumped bool
}

// NewCoverageTracer returns a new CoverageTracer.
func NewCoverageTracer() *CoverageTracer {
	tracer := &CoverageTracer{
		coverageMaps:    NewCoverageMaps(),
		callFrameStates: make([]*coverageTracerCallFrameState, 0),
		codeHashCache:   [2]map[common.Hash]common.Hash{make(map[common.Hash]common.Hash), make(map[common.Hash]common.Hash)},
	}
	nativeTracer := &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart: tracer.OnTxStart,
			OnEnter:   tracer.OnEnter,
			OnExit:    tracer.OnExit,
			OnOpcode:  tracer.OnOpcode,
		},
	}
	tracer.nativeTracer = &chain.TestChainTracer{Tracer: nativeTracer, CaptureTxEndSetAdditionalResults: tracer.CaptureTxEndSetAdditionalResults}

	return tracer
}

// NativeTracer returns the underlying TestChainTracer.
func (t *CoverageTracer) NativeTracer() *chain.TestChainTracer {
	return t.nativeTracer
}

// OnTxStart is called upon the start of transaction execution, as defined by tracers.Tracer.
func (t *CoverageTracer) OnTxStart(vm *tracing.VMContext, tx *coretypes.Transaction, from common.Address) {
	// Reset our call frame states
	t.callDepth = 0
	t.coverageMaps = NewCoverageMaps()
	t.callFrameStates = make([]*coverageTracerCallFrameState, 0)
	t.evmContext = vm
}

// OnEnter initializes the tracing operation for the top of a call frame, as defined by tracers.Tracer.
func (t *CoverageTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Check to see if this is the top level call frame
	isTopLevelFrame := depth == 0

	// Increment call frame depth if it is not the top level call frame
	if !isTopLevelFrame {
		t.callDepth++
	}

	// Create our state tracking struct for this frame.
	t.callFrameStates = append(t.callFrameStates, &coverageTracerCallFrameState{
		create:             typ == byte(vm.CREATE) || typ == byte(vm.CREATE2),
		pendingCoverageMap: NewCoverageMaps(),
	})
}

// OnExit is called after a call to finalize tracing completes for the top of a call frame, as defined by tracers.Tracer.
func (t *CoverageTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	currentCallFrameState := t.callFrameStates[t.callDepth]
	currentCoverageMap := currentCallFrameState.pendingCoverageMap

	// Record the exit in our coverage map
	// We should always be initialized here, but if we aren't then fields like address will be messed up, so we check to be sure
	if currentCallFrameState.initialized && currentCallFrameState.lookupHash != nil {
		var markerXor uint64
		if reverted {
			markerXor = REVERT_MARKER_XOR
		} else {
			markerXor = RETURN_MARKER_XOR
		}
		marker := bits.RotateLeft64(currentCallFrameState.lastPC, 32) ^ markerXor
		_, coverageUpdateErr := currentCoverageMap.UpdateAt(currentCallFrameState.address, *currentCallFrameState.lookupHash, marker)
		if coverageUpdateErr != nil {
			logging.GlobalLogger.Panic("Coverage tracer failed to update coverage map while tracing state", coverageUpdateErr)
		}
	}

	// Check to see if this is the top level call frame
	isTopLevelFrame := depth == 0

	// Commit all our coverage maps up one call frame.
	var coverageUpdateErr error
	if isTopLevelFrame {
		// Update the final coverage map if this is the top level call frame
		_, coverageUpdateErr = t.coverageMaps.Update(currentCoverageMap)
	} else {
		// Move coverage up one call frame
		_, coverageUpdateErr = t.callFrameStates[t.callDepth-1].pendingCoverageMap.Update(currentCoverageMap)

		// Pop the state tracking struct for this call frame off the stack and decrement the call depth
		t.callFrameStates = t.callFrameStates[:t.callDepth]
		t.callDepth--
	}
	if coverageUpdateErr != nil {
		logging.GlobalLogger.Panic("Coverage tracer failed to update coverage map during capture end", coverageUpdateErr)
	}

}

// OnOpcode records data from an EVM state update, as defined by tracers.Tracer.
func (t *CoverageTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	// Obtain our call frame state tracking struct
	callFrameState := t.callFrameStates[t.callDepth]

	// Back up these values before we overwrite them
	initialized := callFrameState.initialized
	justJumped := callFrameState.justJumped
	lastPC := callFrameState.lastPC

	// Record some info about where we are
	callFrameState.lastPC = pc
	callFrameState.justJumped = vm.OpCode(op) == vm.JUMP || vm.OpCode(op) == vm.JUMPI
	if !initialized {
		callFrameState.initialized = true
		callFrameState.address = scope.Address()
	}

	// Now record coverage, if applicable. Otherwise return

	var marker uint64
	if !initialized { // first opcode
		marker = bits.RotateLeft64(ENTER_MARKER_XOR, 32) ^ pc
	} else if justJumped {
		marker = bits.RotateLeft64(lastPC, 32) ^ pc
	} else {
		return
	}

	// We can cast OpContext to ScopeContext because that is the type passed to OnOpcode.
	scopeContext := scope.(*vm.ScopeContext)
	code := scopeContext.Contract.Code
	isCreate := callFrameState.create
	gethCodeHash := scopeContext.Contract.CodeHash

	cacheArrayKey := 1
	if isCreate {
		cacheArrayKey = 0
	}

	// Obtain our contract coverage map lookup hash.
	if callFrameState.lookupHash == nil {
		if isCreate {
			lookupHash := getContractCoverageMapHash(code, isCreate)
			callFrameState.lookupHash = &lookupHash
		} else {
			lookupHash, cacheHit := t.codeHashCache[cacheArrayKey][gethCodeHash]
			if !cacheHit {
				lookupHash = getContractCoverageMapHash(code, isCreate)
				t.codeHashCache[cacheArrayKey][gethCodeHash] = lookupHash
			}
			callFrameState.lookupHash = &lookupHash
		}
	}

	// Record coverage for this location in our map.
	_, coverageUpdateErr := callFrameState.pendingCoverageMap.UpdateAt(callFrameState.address, *callFrameState.lookupHash, marker)
	if coverageUpdateErr != nil {
		logging.GlobalLogger.Panic("Coverage tracer failed to update coverage map while tracing state", coverageUpdateErr)
	}
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *CoverageTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Store our tracer results.
	results.AdditionalResults[coverageTracerResultsKey] = t.coverageMaps
}
