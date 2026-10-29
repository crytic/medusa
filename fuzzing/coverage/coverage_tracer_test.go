package coverage

import (
	"math/bits"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa/chain/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCoverageNestedFramesAndReverts checks exact counts across recursion, reverts,
// empty calls, and reuse of the tracer with a retained transaction result.
func TestCoverageNestedFramesAndReverts(t *testing.T) {
	t.Parallel()
	tracer := NewCoverageTracer()
	address := common.Address{1}
	code := []byte{byte(vm.JUMPDEST), byte(vm.JUMP), byte(vm.STOP)}
	contract := vm.NewContract(common.Address{}, address, new(uint256.Int), 1000, nil)
	contract.SetCallCode(common.Hash{1}, code)
	scope := &vm.ScopeContext{Contract: contract}
	hash := getContractCoverageMapHash(code, false)
	enter := func(depth int) {
		tracer.OnEnter(depth, byte(vm.CALL), address, address, nil, 1000, nil)
		tracer.OnOpcode(0, byte(vm.JUMPDEST), 0, 0, scope, nil, depth+1, nil)
		tracer.OnOpcode(1, byte(vm.JUMP), 0, 0, scope, nil, depth+1, nil)
		tracer.OnOpcode(2, byte(vm.STOP), 0, 0, scope, nil, depth+1, nil)
	}
	tracer.OnTxStart(nil, nil, address)
	enter(0)
	enter(1) // Recursion shares code and address, but has distinct hit counts.
	tracer.OnExit(1, nil, 0, vm.ErrExecutionReverted, true)
	tracer.OnEnter(1, byte(vm.CALL), address, common.Address{2}, nil, 1000, nil)
	tracer.OnExit(1, nil, 0, nil, false) // An empty/precompile frame has no coverage.
	enter(1)
	tracer.OnExit(1, nil, 0, nil, false)
	tracer.OnExit(0, nil, 0, vm.ErrExecutionReverted, true)
	results := &types.MessageResults{AdditionalResults: make(map[string]any)}
	tracer.CaptureTxEndSetAdditionalResults(results)
	got := GetCoverageTracerResults(results)
	markers := got.maps[hash][address]
	require.Equal(t, 4, markers.executedMarkers.count())
	require.EqualValues(t, 3, markers.HitCount(bits.RotateLeft64(ENTER_MARKER_XOR, 32)))
	require.EqualValues(t, 3, markers.HitCount(uint64(1)<<32|2))
	require.EqualValues(t, 2, markers.HitCount(uint64(2)<<32|REVERT_MARKER_XOR))
	require.EqualValues(t, 1, markers.HitCount(uint64(2)<<32|RETURN_MARKER_XOR))
	// Reusing tracer storage must not change retained transaction results.
	tracer.OnTxStart(nil, nil, address)
	enter(0)
	tracer.OnExit(0, nil, 0, nil, false)
	require.EqualValues(t, 3, markers.HitCount(uint64(1)<<32|2))
	require.EqualValues(t, 1, tracer.coverageMaps.maps[hash][address].HitCount(uint64(1)<<32|2))
}

// TestCoverageFrameTargets verifies that interleaved frames keep their own code
// and address, including creation and addresses collapsed into the zero address.
func TestCoverageFrameTargets(t *testing.T) {
	t.Parallel()
	tracer := NewCoverageTracer()
	initial := map[common.Address]struct{}{{1}: {}}
	tracer.SetInitialContractsSet(&initial)
	runtimeCode := []byte{byte(vm.JUMPDEST), byte(vm.STOP)}
	initCode := []byte{byte(vm.JUMPDEST), byte(vm.JUMPDEST), byte(vm.STOP)}
	tracer.OnTxStart(nil, nil, common.Address{})
	for depth := 0; depth < 20; depth++ {
		address := common.Address{byte(depth + 1)}
		code, op := runtimeCode, vm.CALL
		if depth%2 == 1 {
			code, op = initCode, vm.CREATE2
		}
		contract := vm.NewContract(common.Address{}, address, new(uint256.Int), 1000, nil)
		contract.SetCallCode(common.Hash{byte(len(code))}, code)
		scope := &vm.ScopeContext{Contract: contract}
		tracer.OnEnter(depth, byte(op), common.Address{}, address, nil, 1000, nil)
		tracer.OnOpcode(0, byte(vm.JUMPDEST), 0, 0, scope, nil, depth+1, nil)
	}
	for depth := 19; depth >= 0; depth-- {
		tracer.OnExit(depth, nil, 0, nil, depth%2 == 1)
	}
	result := &types.MessageResults{AdditionalResults: make(map[string]any)}
	tracer.CaptureTxEndSetAdditionalResults(result)
	got := GetCoverageTracerResults(result)
	enter := bits.RotateLeft64(ENTER_MARKER_XOR, 32)
	runtimeMaps := got.maps[getContractCoverageMapHash(runtimeCode, false)]
	initMaps := got.maps[getContractCoverageMapHash(initCode, true)]
	require.Len(t, runtimeMaps, 2)
	require.Len(t, initMaps, 1)
	require.EqualValues(t, 1, runtimeMaps[common.Address{1}].HitCount(enter))
	require.EqualValues(t, 9, runtimeMaps[common.Address{}].HitCount(RETURN_MARKER_XOR))
	require.EqualValues(t, 10, initMaps[common.Address{}].HitCount(REVERT_MARKER_XOR))
}

// TestCoverageRetainedResults verifies that reading a published result remains
// safe while the same tracer executes later transactions with fresh storage.
func TestCoverageRetainedResults(t *testing.T) {
	t.Parallel()
	tracer := NewCoverageTracer()
	address := common.Address{1}
	code := []byte{byte(vm.STOP)}
	contract := vm.NewContract(common.Address{}, address, new(uint256.Int), 1000, nil)
	contract.SetCallCode(common.Hash{1}, code)
	scope := &vm.ScopeContext{Contract: contract}
	execute := func() *CoverageMaps {
		tracer.OnTxStart(nil, nil, address)
		tracer.OnEnter(0, byte(vm.CALL), address, address, nil, 1000, nil)
		tracer.OnOpcode(0, byte(vm.STOP), 0, 0, scope, nil, 1, nil)
		tracer.OnExit(0, nil, 0, nil, false)
		result := &types.MessageResults{AdditionalResults: make(map[string]any)}
		tracer.CaptureTxEndSetAdditionalResults(result)
		retained := GetCoverageTracerResults(result)
		RemoveCoverageTracerResults(result)
		return retained
	}
	retained := execute()
	readDone := make(chan bool, 1)
	go func() {
		valid := true
		for range 1000 {
			valid = valid && retained.BranchesHit() == 2 && retained.maps[getContractCoverageMapHash(code, false)][address].HitCount(RETURN_MARKER_XOR) == 1
		}
		readDone <- valid
	}()
	for range 100 {
		next := execute()
		require.NotSame(t, retained, next)
		require.True(t, retained.Equal(next))
	}
	require.True(t, <-readDone, "published coverage changed during a later transaction")
}

// TestCoverageInitialSetChanges preserves address selection when a caller edits
// the externally supplied initial contract set between tracing callbacks.
func TestCoverageInitialSetChanges(t *testing.T) {
	t.Parallel()
	tracer := NewCoverageTracer()
	address := common.Address{1}
	initial := map[common.Address]struct{}{address: {}}
	tracer.SetInitialContractsSet(&initial)
	code := []byte{byte(vm.JUMP), byte(vm.STOP)}
	contract := vm.NewContract(common.Address{}, address, new(uint256.Int), 1000, nil)
	contract.SetCallCode(common.Hash{1}, code)
	scope := &vm.ScopeContext{Contract: contract}
	tracer.OnTxStart(nil, nil, address)
	tracer.OnEnter(0, byte(vm.CALL), address, address, nil, 1000, nil)
	tracer.OnOpcode(0, byte(vm.JUMP), 0, 0, scope, nil, 1, nil)
	delete(initial, address)
	tracer.OnOpcode(1, byte(vm.STOP), 0, 0, scope, nil, 1, nil)
	tracer.SetInitialContractsSet(nil)
	tracer.OnExit(0, nil, 0, nil, false)
	result := &types.MessageResults{AdditionalResults: make(map[string]any)}
	tracer.CaptureTxEndSetAdditionalResults(result)
	maps := GetCoverageTracerResults(result).maps[getContractCoverageMapHash(code, false)]
	require.Len(t, maps, 2)
	require.EqualValues(t, 1, maps[address].HitCount(bits.RotateLeft64(ENTER_MARKER_XOR, 32)))
	require.EqualValues(t, 1, maps[common.Address{}].HitCount(1))
	require.EqualValues(t, 1, maps[address].HitCount(uint64(1)<<32|RETURN_MARKER_XOR))
}
