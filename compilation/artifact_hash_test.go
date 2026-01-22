package compilation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crytic/medusa/compilation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeArtifactHash_EmptyCompilations(t *testing.T) {
	t.Parallel()

	hash := ComputeArtifactHash(nil)
	assert.NotEmpty(t, hash, "hash should not be empty even for nil compilations")

	hash2 := ComputeArtifactHash([]types.Compilation{})
	assert.Equal(t, hash, hash2, "hash should be the same for nil and empty slice")
}

func TestComputeArtifactHash_Deterministic(t *testing.T) {
	t.Parallel()

	compilations := createTestCompilations()

	hash1 := ComputeArtifactHash(compilations)
	hash2 := ComputeArtifactHash(compilations)

	assert.Equal(t, hash1, hash2, "hash should be deterministic")
}

func TestComputeArtifactHash_DifferentBytecode(t *testing.T) {
	t.Parallel()

	compilations1 := createTestCompilations()
	compilations2 := createTestCompilationsWithDifferentBytecode()

	hash1 := ComputeArtifactHash(compilations1)
	hash2 := ComputeArtifactHash(compilations2)

	assert.NotEqual(t, hash1, hash2, "different bytecode should produce different hash")
}

func TestComputeArtifactHash_OrderIndependent(t *testing.T) {
	t.Parallel()

	// Create compilations with contracts in different orders
	compilation1 := types.Compilation{
		SourcePathToArtifact: map[string]types.SourceArtifact{
			"Contract.sol": {
				Contracts: map[string]types.CompiledContract{
					"Alpha": {
						InitBytecode:    []byte{0x01, 0x02},
						RuntimeBytecode: []byte{0x03, 0x04},
					},
					"Beta": {
						InitBytecode:    []byte{0x05, 0x06},
						RuntimeBytecode: []byte{0x07, 0x08},
					},
				},
			},
		},
	}

	compilation2 := types.Compilation{
		SourcePathToArtifact: map[string]types.SourceArtifact{
			"Contract.sol": {
				Contracts: map[string]types.CompiledContract{
					"Beta": {
						InitBytecode:    []byte{0x05, 0x06},
						RuntimeBytecode: []byte{0x07, 0x08},
					},
					"Alpha": {
						InitBytecode:    []byte{0x01, 0x02},
						RuntimeBytecode: []byte{0x03, 0x04},
					},
				},
			},
		},
	}

	hash1 := ComputeArtifactHash([]types.Compilation{compilation1})
	hash2 := ComputeArtifactHash([]types.Compilation{compilation2})

	assert.Equal(t, hash1, hash2, "hash should be independent of contract order in map")
}

func TestLoadArtifactHashCache_NonExistent(t *testing.T) {
	t.Parallel()

	cache := LoadArtifactHashCache("/nonexistent/path")
	assert.Nil(t, cache, "should return nil for non-existent cache")
}

func TestSaveAndLoadArtifactHashCache(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	originalCache := &ArtifactHashCache{
		Hash:      "abc123def456",
		Timestamp: time.Now().Truncate(time.Second), // Truncate for JSON round-trip
	}

	err := SaveArtifactHashCache(tempDir, originalCache)
	require.NoError(t, err, "should save cache without error")

	loadedCache := LoadArtifactHashCache(tempDir)
	require.NotNil(t, loadedCache, "should load cache successfully")

	assert.Equal(t, originalCache.Hash, loadedCache.Hash, "hash should match")
	assert.WithinDuration(t, originalCache.Timestamp, loadedCache.Timestamp, time.Second, "timestamp should match")
}

func TestSaveArtifactHashCache_CreatesDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nested", "dir")

	cache := &ArtifactHashCache{
		Hash:      "test123",
		Timestamp: time.Now(),
	}

	err := SaveArtifactHashCache(nestedDir, cache)
	require.NoError(t, err, "should create nested directories")

	// Verify file exists
	_, err = os.Stat(filepath.Join(nestedDir, ArtifactHashCacheFileName))
	assert.NoError(t, err, "cache file should exist")
}

func TestLoadArtifactHashCache_InvalidJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, ArtifactHashCacheFileName)

	err := os.WriteFile(cachePath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	cache := LoadArtifactHashCache(tempDir)
	assert.Nil(t, cache, "should return nil for invalid JSON")
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30 seconds"},
		{1 * time.Minute, "1 minute"},
		{5 * time.Minute, "5 minutes"},
		{1 * time.Hour, "1 hour"},
		{3 * time.Hour, "3 hours"},
		{24 * time.Hour, "1 day"},
		{72 * time.Hour, "3 days"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

func createTestCompilations() []types.Compilation {
	return []types.Compilation{
		{
			SourcePathToArtifact: map[string]types.SourceArtifact{
				"TestContract.sol": {
					Contracts: map[string]types.CompiledContract{
						"TestContract": {
							InitBytecode:    []byte{0x60, 0x80, 0x60, 0x40},
							RuntimeBytecode: []byte{0x60, 0x80, 0x60, 0x40, 0x52},
						},
					},
				},
			},
		},
	}
}

func createTestCompilationsWithDifferentBytecode() []types.Compilation {
	return []types.Compilation{
		{
			SourcePathToArtifact: map[string]types.SourceArtifact{
				"TestContract.sol": {
					Contracts: map[string]types.CompiledContract{
						"TestContract": {
							InitBytecode:    []byte{0x60, 0x80, 0x60, 0x41}, // Different bytecode
							RuntimeBytecode: []byte{0x60, 0x80, 0x60, 0x40, 0x53},
						},
					},
				},
			},
		},
	}
}
