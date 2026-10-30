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
	// markerCapacityHints remembers sizes, never published result storage. Hints
	// are capped and shrink after a small transaction, limiting over-allocation.
	markerCapacityHints map[common.Hash]int
	// coverageMaps records all executed branches, including reverted frames.
	coverageMaps *CoverageMaps

	// callFrameStates describes the state tracked by the tracer per call frame.
	callFrameStates []coverageTracerCallFrameState

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

	// initialContractsSet records the set of contract addresses present in the base chain,
	// before any contracts are added by test sequences. Only these addresses will be recorded
	// in coverage; others will be replaced with the zero address to prevent infinitely growing corpus.
	initialContractsSet *map[common.Address]struct{}
}

// coverageTracerCallFrameState tracks state across call frames in the tracer.
type coverageTracerCallFrameState struct {
	// Coverage is initialized on the first opcode, when code and scope are available.
	// initialized tracks whether or not this has happened yet.
	initialized bool

	// create indicates whether the current call frame is executing on init bytecode (deploying a contract).
	create bool

	// coverageMap points into the transaction coverage, normally resolved on the first opcode.
	coverageMap *ContractCoverageMap

	// address and lookupHash identify this frame's code. coverageAddress is the
	// address used for coverageMap; it can change if the initial contract set changes.
	address         common.Address
	lookupHash      common.Hash
	coverageAddress common.Address

	// lastPC is the most recent PC that has been executed. Used for coverage tracking.
	lastPC uint64

	// justJumped indicates whether or not the most recent instruction (the one indicated by lastPC) was JUMP/JUMPI.
	justJumped bool
}

// NewCoverageTracer returns a new CoverageTracer.
func NewCoverageTracer() *CoverageTracer {
	tracer := &CoverageTracer{
		markerCapacityHints: make(map[common.Hash]int),
		coverageMaps:        NewCoverageMaps(),
		callFrameStates:     make([]coverageTracerCallFrameState, 0),
		codeHashCache:       [2]map[common.Hash]common.Hash{make(map[common.Hash]common.Hash), make(map[common.Hash]common.Hash)},
	}
	nativeTracer := &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart:        tracer.OnTxStart,
			OnEnter:          tracer.OnEnter,
			OnExit:           tracer.OnExit,
			OnOpcode:         tracer.OnOpcode,
			OnOpcodeForScope: tracer.OnOpcodeForScope,
		},
	}
	tracer.nativeTracer = &chain.TestChainTracer{Tracer: nativeTracer, CaptureTxEndSetAdditionalResults: tracer.CaptureTxEndSetAdditionalResults}

	return tracer
}

// NativeTracer returns the underlying TestChainTracer.
func (t *CoverageTracer) NativeTracer() *chain.TestChainTracer {
	return t.nativeTracer
}

// OnOpcodeForScope selects the current OnOpcode hook for every frame. Coverage
// needs every frame; selecting per frame lets a router call it directly.
func (t *CoverageTracer) OnOpcodeForScope(tracing.OpContext) tracing.OpcodeHook {
	return t.nativeTracer.OnOpcode
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
	// Allocate a fresh result for this transaction. Observers and the corpus may
	// still hold maps from earlier transactions, so their storage must not be reused.
	// Only the private frame bookkeeping is cleared and reused.
	t.callDepth = 0
	t.coverageMaps = NewCoverageMaps()
	clear(t.callFrameStates)
	t.callFrameStates = t.callFrameStates[:0]
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
	t.callFrameStates = append(t.callFrameStates, coverageTracerCallFrameState{
		create: typ == byte(vm.CREATE) || typ == byte(vm.CREATE2),
	})
}

// OnExit is called after a call to finalize tracing completes for the top of a call frame, as defined by tracers.Tracer.
func (t *CoverageTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	frame := &t.callFrameStates[t.callDepth]
	if frame.initialized {
		markerXor := uint64(RETURN_MARKER_XOR)
		if reverted {
			markerXor = REVERT_MARKER_XOR
		}
		marker := bits.RotateLeft64(frame.lastPC, 32) ^ markerXor
		t.updateFrameCoverage(frame, marker)
	}
	// Coverage includes reverted frames, so no per-frame merge is necessary.
	// Clear references before reusing the frame storage in the next transaction.
	*frame = coverageTracerCallFrameState{}
	t.callFrameStates = t.callFrameStates[:t.callDepth]
	if depth != 0 {
		t.callDepth--
	}
}

// OnOpcode records data from an EVM state update, as defined by tracers.Tracer.
func (t *CoverageTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	frame := &t.callFrameStates[t.callDepth]
	initialized, justJumped, lastPC := frame.initialized, frame.justJumped, frame.lastPC
	frame.lastPC = pc
	frame.justJumped = vm.OpCode(op) == vm.JUMP || vm.OpCode(op) == vm.JUMPI
	if !initialized {
		frame.initialized = true
		frame.address = scope.Address()
		contract := scope.(*vm.ScopeContext).Contract
		var lookupHash common.Hash
		if frame.create {
			lookupHash = getContractCoverageMapHash(contract.Code, true)
		} else {
			var found bool
			lookupHash, found = t.codeHashCache[1][contract.CodeHash]
			if !found {
				lookupHash = getContractCoverageMapHash(contract.Code, false)
				t.codeHashCache[1][contract.CodeHash] = lookupHash
			}
		}
		frame.lookupHash = lookupHash
		marker := bits.RotateLeft64(ENTER_MARKER_XOR, 32) ^ pc
		frame.coverageAddress = t.addressForCoverage(frame.address)
		_, _ = t.coverageMaps.updateAtWithCapacity(frame.coverageAddress, lookupHash, marker, t.markerCapacityHints[lookupHash])
		frame.coverageMap = t.coverageMaps.cachedMap
	} else if justJumped {
		marker := bits.RotateLeft64(lastPC, 32) ^ pc
		t.updateFrameCoverage(frame, marker)
	}
}

// updateFrameCoverage uses the frame's resolved map while preserving changes to
// the externally supplied initial contract set between tracing callbacks.
func (t *CoverageTracer) updateFrameCoverage(frame *coverageTracerCallFrameState, marker uint64) {
	address := t.addressForCoverage(frame.address)
	if address != frame.coverageAddress {
		_, _ = t.coverageMaps.updateAtWithCapacity(address, frame.lookupHash, marker, t.markerCapacityHints[frame.lookupHash])
		frame.coverageMap = t.coverageMaps.cachedMap
		frame.coverageAddress = address
		return
	}
	_, _ = frame.coverageMap.updateCoveredAt(marker)
}

// CaptureTxEndSetAdditionalResults can be used to set additional results captured from execution tracing. If this
// tracer is used during transaction execution (block creation), the results can later be queried from the block.
// This method will only be called on the added tracer if it implements the extended TestChainTracer interface.
func (t *CoverageTracer) CaptureTxEndSetAdditionalResults(results *types.MessageResults) {
	// Learn from occupancy rather than allocated capacity so a large transaction
	// cannot keep subsequent small results over-allocated indefinitely. Use the
	// largest deployment for each code hash, capped at 512 initial slots. Tables
	// still grow normally if more markers are recorded.
	for hash, byAddress := range t.coverageMaps.maps {
		capacityHint := 16
		for _, cov := range byAddress {
			for capacityHint < 512 && cov.executedMarkers.used > capacityHint/4*3 {
				capacityHint *= 2
			}
		}
		t.markerCapacityHints[hash] = capacityHint
	}
	// Publish this transaction's maps without transferring exclusive ownership.
	// OnTxStart allocates fresh maps even if these results have been removed from
	// MessageResults: observers may still retain references to them.
	results.AdditionalResults[coverageTracerResultsKey] = t.coverageMaps
}
