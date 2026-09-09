package coverage

import (
	"math/big"
	"math/bits"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/tracing"
	coretypes "github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/logging"
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
	// coverageMaps records all executed branches, including reverted call frames.
	coverageMaps *CoverageMaps

	// callFrameStates describes the state tracked by the tracer per call frame.
	callFrameStates []coverageTracerCallFrameState

	// callDepth refers to the current EVM depth during tracing.
	callDepth int

	// nativeTracer is the underlying tracer used to capture EVM execution.
	nativeTracer *chain.TestChainTracer

	// codeHashCache reuses metadata-aware coverage identities across call frames.
	// Only runtime bytecode is cached: init bytecode may have no geth code hash.
	codeHashCache map[common.Hash]common.Hash

	// initialContractsSet records the set of contract addresses present in the base chain,
	// before any contracts are added by test sequences. Only these addresses will be recorded
	// in coverage; others will be replaced with the zero address to prevent infinitely growing corpus.
	initialContractsSet *map[common.Address]struct{}
}

// coverageTracerCallFrameState tracks state across call frames in the tracer.
type coverageTracerCallFrameState struct {
	// Some fields, such as address, are not initialized until OnOpcode is called.
	// initialized tracks whether or not this has happened yet.
	initialized bool

	// create indicates whether the current call frame is executing on init bytecode (deploying a contract).
	create bool

	// lookupHash describes the hash used to look up the ContractCoverageMap being updated in this frame.
	lookupHash common.Hash

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
		callFrameStates: make([]coverageTracerCallFrameState, 0),
		codeHashCache:   make(map[common.Hash]common.Hash),
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

// SetInitialContractsSet sets the initialContractsSet value (see above).
func (t *CoverageTracer) SetInitialContractsSet(initialContractsSet *map[common.Address]struct{}) {
	t.initialContractsSet = initialContractsSet
}

// BLANK_ADDRESS is an all-zero address; it's a global var so that we don't have to recalculate (and reallocate) it every time.
var BLANK_ADDRESS = common.BytesToAddress([]byte{})

// addressForCoverage modifies an address based on the initialContractsSet value.
// This is applied to all addresses before they are recorded in the coverage map.
// If t.initialContractsSet is nil, we preserve all addresses.
// If t.initialContractsSet is defined, we only preserve addresses present in this set.
// Addresses not present in this set are zeroed to prevent issues with infinitely growing corpus.
func (t *CoverageTracer) addressForCoverage(address common.Address) common.Address {
	if t.initialContractsSet == nil {
		return address
	} else if _, ok := (*t.initialContractsSet)[address]; ok {
		return address
	} else {
		return BLANK_ADDRESS
	}
}

// OnTxStart is called upon the start of transaction execution, as defined by tracers.Tracer.
func (t *CoverageTracer) OnTxStart(vm *tracing.VMContext, tx *coretypes.Transaction, from common.Address) {
	// Reset our call frame states
	t.callDepth = 0
	t.coverageMaps = NewCoverageMaps()
	t.callFrameStates = t.callFrameStates[:0]
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
	t.callFrameStates = append(t.callFrameStates, coverageTracerCallFrameState{
		create: typ == byte(vm.CREATE) || typ == byte(vm.CREATE2),
	})
}

// OnExit is called after a call to finalize tracing completes for the top of a call frame, as defined by tracers.Tracer.
func (t *CoverageTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	currentCallFrameState := &t.callFrameStates[t.callDepth]

	// Record the exit in our coverage map
	// We should always be initialized here, but if we aren't then fields like address will be messed up, so we check to be sure
	if currentCallFrameState.initialized {
		var markerXor uint64
		if reverted {
			markerXor = REVERT_MARKER_XOR
		} else {
			markerXor = RETURN_MARKER_XOR
		}
		marker := bits.RotateLeft64(currentCallFrameState.lastPC, 32) ^ markerXor
		address := t.addressForCoverage(currentCallFrameState.address)
		_, coverageUpdateErr := t.coverageMaps.UpdateAt(address, currentCallFrameState.lookupHash, marker)
		if coverageUpdateErr != nil {
			logging.GlobalLogger.Panic("Coverage tracer failed to update coverage map while tracing state", coverageUpdateErr)
		}
	}

	if depth != 0 {
		t.callFrameStates = t.callFrameStates[:t.callDepth]
		t.callDepth--
	}
}

// OnOpcode records data from an EVM state update, as defined by tracers.Tracer.
func (t *CoverageTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	// Obtain our call frame state tracking struct
	callFrameState := &t.callFrameStates[t.callDepth]

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
		callFrameState.lookupHash = t.lookupCodeHash(scope.(*vm.ScopeContext), callFrameState.create)
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

	// Record coverage for this location in our map.
	address := t.addressForCoverage(callFrameState.address)
	_, coverageUpdateErr := t.coverageMaps.UpdateAt(address, callFrameState.lookupHash, marker)
	if coverageUpdateErr != nil {
		logging.GlobalLogger.Panic("Coverage tracer failed to update coverage map while tracing state", coverageUpdateErr)
	}
}

func (t *CoverageTracer) lookupCodeHash(scope *vm.ScopeContext, create bool) common.Hash {
	if create {
		return getContractCoverageMapHash(scope.Contract.Code, true)
	}
	codeHash := scope.Contract.CodeHash
	lookupHash, ok := t.codeHashCache[codeHash]
	if !ok {
		lookupHash = getContractCoverageMapHash(scope.Contract.Code, false)
		t.codeHashCache[codeHash] = lookupHash
	}
	return lookupHash
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *CoverageTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Store our tracer results.
	results.AdditionalResults[coverageTracerResultsKey] = t.coverageMaps
}
