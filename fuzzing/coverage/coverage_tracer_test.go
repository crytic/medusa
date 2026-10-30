package coverage

import (
	"maps"
	"math/bits"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/crypto"
	"github.com/crytic/medusa/chain/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func coverageScope(address common.Address, code []byte) *vm.ScopeContext {
	contract := vm.NewContract(common.Address{}, address, new(uint256.Int), 100000, nil)
	contract.SetCallCode(crypto.Keccak256Hash(code), code)
	return &vm.ScopeContext{Contract: contract}
}

func traceCoverageFrame(tracer *CoverageTracer, scope *vm.ScopeContext, depth int) {
	tracer.OnEnter(depth, byte(vm.CALL), common.Address{}, scope.Address(), nil, 100000, nil)
	tracer.OnOpcode(0, byte(vm.JUMPDEST), 0, 0, scope, nil, depth, nil)
	tracer.OnOpcode(1, byte(vm.JUMPI), 0, 0, scope, nil, depth, nil)
	tracer.OnOpcode(8, byte(vm.JUMPDEST), 0, 0, scope, nil, depth, nil)
}

func TestCoverageNestedFrames(t *testing.T) {
	t.Parallel()
	for _, reverted := range []bool{false, true} {
		t.Run(map[bool]string{false: "return", true: "revert"}[reverted], func(t *testing.T) {
			tracer := NewCoverageTracer()
			address := common.Address{1}
			code := []byte{byte(vm.JUMPDEST), byte(vm.JUMPI)}
			scope := coverageScope(address, code)
			tracer.OnTxStart(nil, nil, common.Address{})
			traceCoverageFrame(tracer, scope, 0)
			for i := 0; i < 3; i++ {
				traceCoverageFrame(tracer, scope, 1)
				tracer.OnExit(1, nil, 0, nil, reverted)
			}
			tracer.OnExit(0, nil, 0, nil, false)
			results := &types.MessageResults{AdditionalResults: make(map[string]any)}
			tracer.CaptureTxEndSetAdditionalResults(results)
			coverage := GetCoverageTracerResults(results)
			markers, err := coverage.GetContractCoverageMap(code, false)
			require.NoError(t, err)
			expected := map[uint64]uint64{
				bits.RotateLeft64(ENTER_MARKER_XOR, 32):      4,
				bits.RotateLeft64(1, 32) ^ 8:                 4,
				bits.RotateLeft64(8, 32) ^ RETURN_MARKER_XOR: 1,
			}
			exit := uint64(RETURN_MARKER_XOR)
			if reverted {
				exit = REVERT_MARKER_XOR
			}
			expected[bits.RotateLeft64(8, 32)^exit] += 3
			require.Equal(t, expected, maps.Collect(markers.executedMarkers.all()))
			// Subsequent transactions must not mutate already published coverage results.
			tracer.OnTxStart(nil, nil, common.Address{})
			traceCoverageFrame(tracer, scope, 0)
			tracer.OnExit(0, nil, 0, nil, true)
			markers, err = coverage.GetContractCoverageMap(code, false)
			require.NoError(t, err)
			require.Equal(t, expected, maps.Collect(markers.executedMarkers.all()))
		})
	}
}

func TestCoverageAddressesAndCode(t *testing.T) {
	t.Parallel()
	tracer := NewCoverageTracer()
	initial := map[common.Address]struct{}{{1}: {}}
	tracer.SetInitialContractsSet(&initial)
	tracer.OnTxStart(nil, nil, common.Address{})
	code := []byte{byte(vm.JUMPDEST)}
	traceCoverageFrame(tracer, coverageScope(common.Address{1}, code), 0)
	for _, address := range []common.Address{{2}, {3}} {
		traceCoverageFrame(tracer, coverageScope(address, code), 1)
		tracer.OnExit(1, nil, 0, nil, true)
	}
	// Empty-code calls create frames but have no opcodes or coverage markers.
	tracer.OnEnter(1, byte(vm.CALL), common.Address{}, common.Address{4}, nil, 0, nil)
	tracer.OnExit(1, nil, 0, nil, false)
	tracer.OnExit(0, nil, 0, nil, false)
	hash := getContractCoverageMapHash(code, false)
	require.Len(t, tracer.coverageMaps.maps, 1)
	require.Len(t, tracer.coverageMaps.maps[hash], 2)
	enter := bits.RotateLeft64(ENTER_MARKER_XOR, 32)
	require.Equal(t, uint64(1), tracer.coverageMaps.maps[hash][common.Address{1}].HitCount(enter))
	require.Equal(t, uint64(2), tracer.coverageMaps.maps[hash][common.Address{}].HitCount(enter))
}

// BenchmarkCoverageNestedFrames measures a transaction with four reverted child calls.
func BenchmarkCoverageNestedFrames(b *testing.B) {
	tracer := NewCoverageTracer()
	scope := coverageScope(common.Address{1}, []byte{byte(vm.JUMPDEST)})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracer.OnTxStart(nil, nil, common.Address{})
		traceCoverageFrame(tracer, scope, 0)
		for j := 0; j < 4; j++ {
			traceCoverageFrame(tracer, scope, 1)
			tracer.OnExit(1, nil, 0, nil, true)
		}
		tracer.OnExit(0, nil, 0, nil, false)
	}
}

func TestCoverageCreationAndReuse(t *testing.T) {
	t.Parallel()
	tracer := NewCoverageTracer()
	address := common.Address{1}
	enter := bits.RotateLeft64(ENTER_MARKER_XOR, 32)
	for _, create := range []bool{true, false} {
		tracer.OnTxStart(nil, nil, common.Address{})
		tracer.OnEnter(0, byte(vm.CALL), common.Address{}, common.Address{2}, nil, 100000, nil)
		for i := byte(0); i < 2; i++ {
			code := []byte{byte(vm.PUSH1), i, byte(vm.STOP)}
			scope := coverageScope(address, code)
			op := byte(vm.CALL)
			if create {
				op = byte(vm.CREATE)
				scope.Contract.CodeHash = common.Hash{}
			}
			tracer.OnEnter(1, op, common.Address{}, address, nil, 100000, nil)
			tracer.OnOpcode(0, byte(vm.PUSH1), 0, 0, scope, nil, 1, nil)
			tracer.OnExit(1, nil, 0, nil, false)
		}
		tracer.OnExit(0, nil, 0, nil, false)
		for i := byte(0); i < 2; i++ {
			code := []byte{byte(vm.PUSH1), i, byte(vm.STOP)}
			coverage, err := tracer.coverageMaps.GetContractCoverageMap(code, create)
			require.NoError(t, err)
			require.Equal(t, uint64(1), coverage.HitCount(enter))
		}
		require.Len(t, tracer.coverageMaps.maps, 2)
	}
}

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

// TestCoverageInitialSetChanges preserves address selection when a caller
// replaces the initial contract set between tracing callbacks, and applies an
// in-place edit to frames that start afterwards.
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
	empty := map[common.Address]struct{}{}
	tracer.SetInitialContractsSet(&empty)
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

	// An in-place edit is read when the next frame starts.
	tracer.SetInitialContractsSet(&initial)
	delete(initial, address)
	tracer.OnTxStart(nil, nil, address)
	tracer.OnEnter(0, byte(vm.CALL), address, address, nil, 1000, nil)
	tracer.OnOpcode(0, byte(vm.JUMP), 0, 0, scope, nil, 1, nil)
	tracer.OnExit(0, nil, 0, nil, false)
	result = &types.MessageResults{AdditionalResults: make(map[string]any)}
	tracer.CaptureTxEndSetAdditionalResults(result)
	maps = GetCoverageTracerResults(result).maps[getContractCoverageMapHash(code, false)]
	require.Len(t, maps, 1)
	require.EqualValues(t, 1, maps[common.Address{}].HitCount(bits.RotateLeft64(ENTER_MARKER_XOR, 32)))
}
