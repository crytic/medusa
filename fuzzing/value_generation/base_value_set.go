package value_generation

import (
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"hash"
	"math/big"
)

// BaseValueSet represents potential values of significance within the source code to be used in fuzz tests.
type BaseValueSet struct {
	// integers represents a set of integers to use in fuzz tests. A mapping is used to avoid duplicates.
	integers map[string]big.Int
	// strings represents a set of strings to use in fuzz tests. A mapping is used to avoid duplicates.
	strings map[string]interface{}
	// bytes represents a set of bytes to use in fuzz tests. A mapping is used to avoid duplicates.
	bytes map[string][]byte
	// hashProvider represents a hash provider used to create keys for some data.
	hashProvider hash.Hash
}

// NewBaseValueSet initializes a new BaseValueSet object for use with a Fuzzer.
func NewBaseValueSet() *BaseValueSet {
	baseValueSet := &BaseValueSet{
		integers:     make(map[string]big.Int, 0),
		strings:      make(map[string]interface{}, 0),
		bytes:        make(map[string][]byte, 0),
		hashProvider: sha3.NewLegacyKeccak256(),
	}
	return baseValueSet
}

// Integers converts the internal integers set into a standard list of integers. This may be inefficient to call often,
// so values obtained from this method should be cached in such cases.
func (bvs *BaseValueSet) Integers() []big.Int {
	res := make([]big.Int, len(bvs.integers))
	count := 0
	for _, v := range bvs.integers {
		res[count] = v
		count++
	}
	return res
}

// AddInteger adds an integer item to the BaseValueSet.
func (bvs *BaseValueSet) AddInteger(b big.Int) {
	bvs.integers[b.String()] = b
}

// Strings converts the internal string set into a standard list of strings. This may be inefficient to call often, so
// values obtained from this method should be cached in such cases.
func (bvs *BaseValueSet) Strings() []string {
	res := make([]string, len(bvs.strings))
	count := 0
	for k, _ := range bvs.strings {
		res[count] = k
		count++
	}
	return res
}

// AddString adds a string item to the BaseValueSet.
func (bvs *BaseValueSet) AddString(s string) {
	bvs.strings[s] = nil
}

// Bytes converts the internal bytes set into a standard list of byte arrays. This may be inefficient to call often, so
// values obtained from this method should be cached in such cases.
func (bvs *BaseValueSet) Bytes() [][]byte {
	res := make([][]byte, len(bvs.bytes))
	count := 0
	for _, v := range bvs.bytes {
		res[count] = v
		count++
	}
	return res
}

// AddBytes adds a byte sequence to the BaseValueSet.
func (bvs *BaseValueSet) AddBytes(b []byte) {
	// Calculate hash and reset our hash provider
	bvs.hashProvider.Write(b)
	hashStr := hex.EncodeToString(bvs.hashProvider.Sum(nil))
	bvs.hashProvider.Reset()

	// Add our hash to our "set" (map)
	bvs.bytes[hashStr] = b
}
