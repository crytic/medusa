package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/crytic/medusa-geth/common"
	"go.etcd.io/bbolt"
)

// persistentCache provides a thread-safe cache for storing objects/slots that persists the cache to disk.
type persistentCache struct {
	memCache *nonPersistentStateCache
	db       *bbolt.DB

	pendingWriteMutex sync.Mutex
	pendingWrites     []pendingWrite
	flushThreshold    int
}

type pendingWrite struct {
	key   []byte
	value []byte
}

func newPersistentCache(ctx context.Context, workingDir string, rpcAddr string, height uint64) (*persistentCache, error) {
	cacheDir, err := createCacheDirectory(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	cacheFile := filepath.Join(cacheDir, getCacheFilename(rpcAddr, height))
	db, err := bbolt.Open(cacheFile, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("could not open db: %v", err)
	}

	// create default bucket if it doesn't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("cache"))
		return err
	})
	if err != nil {
		return nil, err
	}

	memCache := newNonPersistentStateCache()
	p := &persistentCache{
		memCache:          memCache,
		db:                db,
		flushThreshold:    25,
		pendingWrites:     []pendingWrite{},
		pendingWriteMutex: sync.Mutex{},
	}

	// close db if context cancelled
	go func() {
		<-ctx.Done()
		err := p.Close()
		if err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	return p, nil
}

func (p *persistentCache) getFromPersist(key []byte, value interface{}) (bool, error) {
	found := false
	err := p.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("cache"))
		data := bucket.Get(key)
		if data == nil {
			return nil
		}
		found = true
		return json.Unmarshal(data, &value)
	})
	if err != nil {
		return false, fmt.Errorf("could not get value: %v", err)
	}

	if !found {
		return false, nil
	} else {
		return true, nil
	}
}

func (p *persistentCache) writeToPersist(key []byte, value []byte) error {
	item := pendingWrite{
		key:   key,
		value: value,
	}
	p.pendingWriteMutex.Lock()
	defer p.pendingWriteMutex.Unlock()

	p.pendingWrites = append(p.pendingWrites, item)
	if len(p.pendingWrites) >= p.flushThreshold {
		return p.flushWrites()
	} else {
		return nil
	}
}

func (p *persistentCache) flushWrites() error {
	err := p.db.Update(func(tx *bbolt.Tx) error {
		for _, pw := range p.pendingWrites {
			bucket := tx.Bucket([]byte("cache"))
			err := bucket.Put(pw.key, pw.value)
			if err != nil {
				return err
			}
		}
		p.pendingWrites = p.pendingWrites[:0]
		return nil
	})
	return err
}

func (p *persistentCache) GetStateObject(addr common.Address) (*StateObject, error) {
	so, err := p.memCache.GetStateObject(addr)
	if err == nil {
		return so, err
	}

	if errors.Is(err, ErrCacheMiss) {
		// check persistent cache
		s := StateObject{}
		exists, err := p.getFromPersist(addr[:], &s)
		if err != nil {
			return nil, err
		}
		if exists {
			err = p.memCache.WriteStateObject(addr, s)
			return &s, err
		} else {
			return nil, ErrCacheMiss
		}
	} else {
		return nil, err
	}
}

func (p *persistentCache) GetSlotData(addr common.Address, slot common.Hash) (common.Hash, error) {
	data, err := p.memCache.GetSlotData(addr, slot)
	if err == nil {
		return data, err
	}

	if errors.Is(err, ErrCacheMiss) {
		// check persistent cache
		data := common.Hash{}

		key := append(addr[:], slot[:]...)
		exists, err := p.getFromPersist(key, &data)
		if err != nil {
			return common.Hash{}, err
		}
		if exists {
			err = p.memCache.WriteSlotData(addr, slot, data)
			return data, err
		} else {
			return common.Hash{}, ErrCacheMiss
		}
	} else {
		return common.Hash{}, err
	}
}

func (p *persistentCache) WriteStateObject(addr common.Address, data StateObject) error {
	err := p.memCache.WriteStateObject(addr, data)
	if err != nil {
		return err
	}

	serialized, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = p.writeToPersist(addr[:], serialized)
	return err
}

func (p *persistentCache) WriteSlotData(addr common.Address, slot common.Hash, data common.Hash) error {
	err := p.memCache.WriteSlotData(addr, slot, data)
	if err != nil {
		return err
	}

	serialized, err := json.Marshal(data)
	if err != nil {
		return err
	}

	key := append(addr[:], slot[:]...)
	err = p.writeToPersist(key, serialized)
	return err
}

func (p *persistentCache) Close() error {
	err := p.flushWrites()
	if err != nil {
		return err
	}
	err = p.db.Close()
	return err
}

func createCacheDirectory(workingDir string) (string, error) {
	cachePath := filepath.Join(workingDir, ".medusacache")
	_, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		// Create directory with 0755 permissions if it doesn't exist
		err = os.Mkdir(cachePath, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create cache directory: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to check cache directory: %w", err)
	}
	return cachePath, nil
}

func getCacheFilename(rpcAddr string, height uint64) string {
	h := sha256.New()
	h.Write([]byte(rpcAddr))
	bs := h.Sum(nil)

	return fmt.Sprintf("%d-%x.dat", height, bs[0:10])
}
