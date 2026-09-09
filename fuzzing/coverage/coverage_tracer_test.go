package coverage

import (
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
			require.Equal(t, expected, markers.executedMarkers)
			// Subsequent transactions must not mutate already published coverage results.
			tracer.OnTxStart(nil, nil, common.Address{})
			traceCoverageFrame(tracer, scope, 0)
			tracer.OnExit(0, nil, 0, nil, true)
			markers, err = coverage.GetContractCoverageMap(code, false)
			require.NoError(t, err)
			require.Equal(t, expected, markers.executedMarkers)
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
