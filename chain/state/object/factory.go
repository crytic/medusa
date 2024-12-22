package object

import (
	"context"
	"errors"
	"os"
)

var _ StateCache = (*nonPersistentStateCache)(nil)
var _ StateCache = (*persistentCache)(nil)

var ErrCacheMiss = errors.New("not found in cache")

// NewPersistentCache creates a new set of persistent caches that will persist cache content to disk.
// Each cache is indexed by the RPC address (to separate network caches) and blockNum
func NewPersistentCache(ctx context.Context, rpcAddr string, height uint64) (StateCache, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return newPersistentCache(ctx, workingDir, rpcAddr, height)
}

func NewNonPersistentCache() (StateCache, error) {
	return newNonPersistentStateCache(), nil
}
