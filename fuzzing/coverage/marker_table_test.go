package coverage

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMarkerTableMatchesMap compares exact counts and change flags to a Go map
// across growth, arbitrary uint64 keys, and wrapping increments.
func TestMarkerTableMatchesMap(t *testing.T) {
	t.Parallel()
	var got markerTable
	want := map[uint64]uint64{}
	rng := rand.New(rand.NewSource(27))
	keys := []uint64{0, math.MaxUint64, 1, 1 << 32, 1 << 63}
	for range 3000 {
		keys = append(keys, rng.Uint64())
	}
	for i := 0; i < 40000; i++ {
		key := keys[rng.Intn(len(keys))]
		delta := uint64(1)
		if i%17 == 0 {
			delta = math.MaxUint64
		}
		changed := got.add(key, delta)
		if changed != (want[key] == 0) {
			t.Fatal("changed", i, key)
		}
		want[key] += delta
		if got.get(key) != want[key] {
			t.Fatal("count", i, key)
		}
	}
	if got.count() != len(want) {
		t.Fatal("size", got.count(), len(want))
	}
	seen := 0
	for k, v := range got.all() {
		seen++
		if want[k] != v {
			t.Fatal("iteration", k, v)
		}
	}
	if seen != len(want) {
		t.Fatal("missing entries")
	}
	for _, k := range keys {
		if got.get(k) != want[k] {
			t.Fatal("final count")
		}
	}
}

// TestMarkerTableZeroCountAndMerge checks zero-valued markers, equality, and
// merging, including a map merged with itself.
func TestMarkerTableZeroCountAndMerge(t *testing.T) {
	t.Parallel()
	a, b := newContractCoverageMap(), newContractCoverageMap()
	for _, key := range []uint64{0, 42, math.MaxUint64} {
		a.executedMarkers.add(key, math.MaxUint64)
		a.executedMarkers.add(key, 1)
		if a.HitCount(key) != 0 {
			t.Fatal("wrap")
		}
	}
	if !a.Equal(b) || a.executedMarkers.count() != 3 {
		t.Fatal("zero-count semantics")
	}
	for _, key := range []uint64{0, 42, math.MaxUint64} {
		b.executedMarkers.add(key, 2)
	}
	changed, err := a.update(b)
	if !changed || err != nil || !a.Equal(b) {
		t.Fatal("merge")
	}
	_, err = a.update(a)
	if err != nil || a.HitCount(42) != 4 {
		t.Fatal("self merge")
	}
}

// TestMarkerTableCollisions checks probing across the end of the table and growth
// with deliberately colliding keys, plus lookup of an absent colliding marker.
func TestMarkerTableCollisions(t *testing.T) {
	t.Parallel()
	var table markerTable
	var keys []uint64
	for key := uint64(1); len(keys) < 97; key++ {
		if markerHash(key)&255 == 255 {
			keys = append(keys, key)
		}
	}
	for i, key := range keys[:96] {
		require.True(t, table.add(key, uint64(i+1)))
	}
	for i, key := range keys[:96] {
		require.EqualValues(t, i+1, table.get(key))
		require.False(t, table.add(key, 1))
		require.EqualValues(t, i+2, table.get(key))
	}
	require.Zero(t, table.get(keys[96]))
	seen := make(map[uint64]uint64)
	for key, count := range table.all() {
		_, duplicate := seen[key]
		require.False(t, duplicate)
		seen[key] = count
	}
	require.Len(t, seen, 96)
	// Iterators must honor early termination, including when marker zero exists.
	for _, withZero := range []bool{false, true} {
		if withZero {
			table.add(0, 1)
		}
		visits := 0
		for range table.all() {
			visits++
			break
		}
		require.Equal(t, 1, visits)
	}
}

// TestMarkerTableEmptyAndZeroMerge preserves the distinction between absent
// storage, allocated storage, and a marker whose uint64 hit count wrapped to zero.
func TestMarkerTableEmptyAndZeroMerge(t *testing.T) {
	t.Parallel()
	empty, wrapped := newContractCoverageMap(), newContractCoverageMap()
	changed, err := empty.update(wrapped)
	require.NoError(t, err)
	require.False(t, changed)
	require.Nil(t, empty.executedMarkers.entries)
	wrapped.executedMarkers.add(0, math.MaxUint64)
	wrapped.executedMarkers.add(0, 1)
	changed, err = empty.update(wrapped)
	require.NoError(t, err)
	require.False(t, changed)
	require.NotNil(t, empty.executedMarkers.entries)
	require.Zero(t, empty.executedMarkers.count())
	require.Equal(t, 1, wrapped.executedMarkers.count())
	require.True(t, empty.Equal(wrapped))
	require.True(t, wrapped.Equal(empty))
	require.True(t, wrapped.executedMarkers.add(0, 1))
	require.EqualValues(t, 1, wrapped.HitCount(0))
	var missing *ContractCoverageMap
	require.Zero(t, missing.HitCount(0))
}
