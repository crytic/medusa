package compilation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/logging/colors"
)

// ArtifactHashCacheFileName is the name of the file used to store the artifact hash.
const ArtifactHashCacheFileName = ".medusa-artifact-hash"

// ArtifactHashCache stores the hash of compilation artifacts along with metadata.
type ArtifactHashCache struct {
	// Hash is the SHA-256 hash of the compiled bytecode.
	Hash string `json:"hash"`
	// Timestamp is when the hash was computed.
	Timestamp time.Time `json:"timestamp"`
}

// ComputeArtifactHash computes a SHA-256 hash of all compiled contract bytecode
// from the provided compilations. The hash is computed deterministically by sorting
// contract names before hashing.
func ComputeArtifactHash(compilations []types.Compilation) string {
	hasher := sha256.New()

	// Collect all contract bytecodes with their names for deterministic ordering
	type contractBytecode struct {
		name            string
		initBytecode    []byte
		runtimeBytecode []byte
	}
	var contracts []contractBytecode

	for _, compilation := range compilations {
		for _, source := range compilation.SourcePathToArtifact {
			for name, contract := range source.Contracts {
				contracts = append(contracts, contractBytecode{
					name:            name,
					initBytecode:    contract.InitBytecode,
					runtimeBytecode: contract.RuntimeBytecode,
				})
			}
		}
	}

	// Sort by contract name for deterministic hashing
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].name < contracts[j].name
	})

	// Hash each contract's bytecode
	for _, c := range contracts {
		hasher.Write([]byte(c.name))
		hasher.Write(c.initBytecode)
		hasher.Write(c.runtimeBytecode)
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// LoadArtifactHashCache loads the artifact hash cache from the specified directory.
// Returns nil if the cache file does not exist or cannot be parsed.
func LoadArtifactHashCache(directory string) *ArtifactHashCache {
	cachePath := filepath.Join(directory, ArtifactHashCacheFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil
	}

	var cache ArtifactHashCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}

	return &cache
}

// SaveArtifactHashCache saves the artifact hash cache to the specified directory.
// Returns an error if the cache cannot be written.
func SaveArtifactHashCache(directory string, cache *ArtifactHashCache) error {
	// Ensure the directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath := filepath.Join(directory, ArtifactHashCacheFileName)
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// NotifyArtifactHashStatus compares the current artifact hash with a cached hash
// and logs an appropriate message. It also updates the cache with the new hash.
// The cacheDirectory parameter specifies where the cache file is stored.
func NotifyArtifactHashStatus(
	compilations []types.Compilation,
	cacheDirectory string,
	logger *logging.Logger,
) {
	if len(compilations) == 0 {
		return
	}

	// Compute the current hash
	currentHash := ComputeArtifactHash(compilations)

	// Load the cached hash
	cachedHash := LoadArtifactHashCache(cacheDirectory)

	// Compare and log the appropriate message
	if cachedHash == nil || cachedHash.Hash != currentHash {
		// New or changed artifacts
		logger.Info(
			colors.Bold, "artifacts: ", colors.Reset,
			"Medusa is running against a ", colors.GreenBold, "new", colors.Reset, " set of build artifacts",
		)
	} else {
		// Same artifacts as before
		timeSince := time.Since(cachedHash.Timestamp)
		logger.Warn(
			colors.Bold, "artifacts: ", colors.Reset,
			"Medusa is running against the ", colors.YellowBold, "same", colors.Reset,
			" build artifacts as previously (last run: ", formatDuration(timeSince), " ago)",
		)
	}

	// Update the cache with the current hash
	newCache := &ArtifactHashCache{
		Hash:      currentHash,
		Timestamp: time.Now(),
	}
	if err := SaveArtifactHashCache(cacheDirectory, newCache); err != nil {
		logger.Warn("Failed to save artifact hash cache", err)
	}
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
