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
	// create indicates whether the current call frame is executing on init bytecode (deploying a contract).
	create bool

	// pendingCoverageMap describes the coverage maps recorded for this call frame.
	pendingCoverageMap *CoverageMaps

	// lookupHash describes the hash used to look up the ContractCoverageMap being updated in this frame.
	lookupHash *common.Hash

	lastPC uint64
	lastOp byte
	address *common.Address
	codeSize int
	lastOpJmp bool
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

// CaptureTxStart is called upon the start of transaction execution, as defined by tracers.Tracer.
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

func (t *CoverageTracer) recordExit(reverted bool) {
	callFrameState := t.callFrameStates[t.callDepth]
	var markerXor uint64
	if reverted {
		markerXor = 0x40000000
	} else {
		markerXor = 0x80000000
	}
	marker := bits.RotateLeft64(callFrameState.lastPC, 32) ^ markerXor
	if callFrameState != nil && callFrameState.lookupHash != nil { // TODO necessary?
		callFrameState.pendingCoverageMap.UpdateAt(*callFrameState.address, *callFrameState.lookupHash, callFrameState.codeSize, marker)
	}
}

// OnExit is called after a call to finalize tracing completes for the top of a call frame, as defined by tracers.Tracer.
func (t *CoverageTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	t.recordExit(reverted)

	// Check to see if this is the top level call frame
	isTopLevelFrame := depth == 0

	// Commit all our coverage maps up one call frame.
	var coverageUpdateErr error
	if isTopLevelFrame {
		// Update the final coverage map if this is the top level call frame
		_, coverageUpdateErr = t.coverageMaps.Update(t.callFrameStates[t.callDepth].pendingCoverageMap)
	} else {
		// Move coverage up one call frame
		_, coverageUpdateErr = t.callFrameStates[t.callDepth-1].pendingCoverageMap.Update(t.callFrameStates[t.callDepth].pendingCoverageMap)

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

	firstOpcode := callFrameState.codeSize == 0

	old_lastOp := vm.OpCode(callFrameState.lastOp)
	old_lastOpJmp := old_lastOp == vm.JUMPI || old_lastOp == vm.JUMP
	old_lastPC := callFrameState.lastPC

	callFrameState.lastPC = pc
	callFrameState.lastOp = op
	if callFrameState.address == nil {
		add := scope.Address() // TODO
		callFrameState.address = &add // TODO
		callFrameState.codeSize = len(scope.(*vm.ScopeContext).Contract.Code) // TODO
	}

	if firstOpcode {
		address := callFrameState.address
		// TODO need to refactor

		// If there is code we're executing, collect coverage.
		// address := scope.Address()
		// We can cast OpContext to ScopeContext because that is the type passed to OnOpcode.
		scopeContext := scope.(*vm.ScopeContext)
		code := scopeContext.Contract.Code
		codeSize := len(code)
		isCreate := callFrameState.create
		gethCodeHash := scopeContext.Contract.CodeHash

		cacheArrayKey := 1
		if isCreate {
			cacheArrayKey = 0
		}

		// TODO used to have a codeSize > 0 check here

		// Obtain our contract coverage map lookup hash.
		if callFrameState.lookupHash == nil {
			if isCreate { // TODO
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
		callFrameState.pendingCoverageMap.UpdateAt(*address, *callFrameState.lookupHash, codeSize, uint64(pc))
	}

	if !old_lastOpJmp { return }

	address := callFrameState.address
	marker := bits.RotateLeft64(old_lastPC, 32) ^ pc

	// If there is code we're executing, collect coverage.
	//address := scope.Address()
	// We can cast OpContext to ScopeContext because that is the type passed to OnOpcode.
	scopeContext := scope.(*vm.ScopeContext)
	code := scopeContext.Contract.Code
	codeSize := len(code)
	isCreate := callFrameState.create
	gethCodeHash := scopeContext.Contract.CodeHash

	cacheArrayKey := 1
	if isCreate {
		cacheArrayKey = 0
	}

	// TODO used to have a codeSize > 0 check here

	// Obtain our contract coverage map lookup hash.
	if callFrameState.lookupHash == nil {
		if isCreate { // TODO
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
	callFrameState.pendingCoverageMap.UpdateAt(*address, *callFrameState.lookupHash, codeSize, marker)
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *CoverageTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Store our tracer results.
	results.AdditionalResults[coverageTracerResultsKey] = t.coverageMaps
}
