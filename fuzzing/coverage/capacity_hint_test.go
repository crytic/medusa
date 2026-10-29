package coverage

import (
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa/chain/types"
	"github.com/stretchr/testify/require"
)

// TestCapacityHintsKeepResultsOwned checks growth beyond the initial hint cap,
// shrinking after an outlier, and the lifetime of a published result.
func TestCapacityHintsKeepResultsOwned(t *testing.T) {
	t.Parallel()
	tracer := NewCoverageTracer()
	address, hash := common.Address{1}, common.Hash{1}
	for marker := uint64(1); marker <= 5000; marker++ {
		_, err := tracer.coverageMaps.UpdateAt(address, hash, marker)
		require.NoError(t, err)
	}
	result := &types.MessageResults{AdditionalResults: make(map[string]any)}
	tracer.CaptureTxEndSetAdditionalResults(result)
	retained := GetCoverageTracerResults(result)
	require.Equal(t, 512, tracer.markerCapacityHints[hash])
	tracer.OnTxStart(nil, nil, common.Address{})
	_, err := tracer.coverageMaps.updateAtWithCapacity(address, hash, 7, tracer.markerCapacityHints[hash])
	require.NoError(t, err)
	require.Len(t, tracer.coverageMaps.cachedMap.executedMarkers.entries, 512)
	tracer.CaptureTxEndSetAdditionalResults(&types.MessageResults{AdditionalResults: make(map[string]any)})
	require.Equal(t, 16, tracer.markerCapacityHints[hash], "a large outlier must not permanently inflate future tables")
	require.EqualValues(t, 5000, retained.BranchesHit())
	require.EqualValues(t, 1, retained.cachedMap.HitCount(7))
	require.EqualValues(t, 1, tracer.coverageMaps.BranchesHit())
}

// TestCapacityHintsAcrossAddresses verifies that deployments sharing a code hash
// get an adequate hint without sharing tables or changing earlier results.
func TestCapacityHintsAcrossAddresses(t *testing.T) {
	t.Parallel()
	tracer := NewCoverageTracer()
	hash := common.Hash{1}
	for i, size := range []uint64{40, 100} {
		address := common.Address{byte(i + 1)}
		for marker := uint64(1); marker <= size; marker++ {
			_, err := tracer.coverageMaps.UpdateAt(address, hash, marker)
			require.NoError(t, err)
		}
	}
	result := &types.MessageResults{AdditionalResults: make(map[string]any)}
	tracer.CaptureTxEndSetAdditionalResults(result)
	retained := GetCoverageTracerResults(result)
	tracer.OnTxStart(nil, nil, common.Address{})
	for i := byte(1); i <= 2; i++ {
		_, err := tracer.coverageMaps.updateAtWithCapacity(common.Address{i}, hash, uint64(100+i), tracer.markerCapacityHints[hash])
		require.NoError(t, err)
	}
	first := tracer.coverageMaps.maps[hash][common.Address{1}]
	second := tracer.coverageMaps.maps[hash][common.Address{2}]
	require.NotSame(t, first, second)
	require.EqualValues(t, 1, first.HitCount(101))
	require.Zero(t, first.HitCount(102))
	require.EqualValues(t, 1, second.HitCount(102))
	require.Zero(t, second.HitCount(101))
	require.EqualValues(t, 100, retained.BranchesHit())
	require.EqualValues(t, 1, retained.maps[hash][common.Address{2}].HitCount(100))
	require.Zero(t, retained.maps[hash][common.Address{1}].HitCount(101))
	require.Zero(t, retained.maps[hash][common.Address{2}].HitCount(102))
}
