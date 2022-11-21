package value_generation

import (
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"golang.org/x/exp/maps"
	"hash"
	"math/big"
)

// ValueSet represents potential values of significance within the source code to be used in fuzz tests.
type ValueSet struct {
	// integers represents a set of integers to use in fuzz tests. A mapping is used to avoid duplicates.
	integers map[string]*big.Int
	// strings represents a set of strings to use in fuzz tests. A mapping is used to avoid duplicates.
	strings map[string]interface{}
	// bytes represents a set of bytes to use in fuzz tests. A mapping is used to avoid duplicates.
	bytes map[string][]byte
	// hashProvider represents a hash provider used to create keys for some data.
	hashProvider hash.Hash
}

// NewValueSet initializes a new ValueSet object for use with a Fuzzer.
func NewValueSet() *ValueSet {
	baseValueSet := &ValueSet{
		integers:     make(map[string]*big.Int, 0),
		strings:      make(map[string]interface{}, 0),
		bytes:        make(map[string][]byte, 0),
		hashProvider: sha3.NewLegacyKeccak256(),
	}
	return baseValueSet
}

// Clone creates a copy of the current ValueSet.
func (vs *ValueSet) Clone() *ValueSet {
	baseValueSet := &ValueSet{
		integers:     maps.Clone(vs.integers),
		strings:      maps.Clone(vs.strings),
		bytes:        maps.Clone(vs.bytes),
		hashProvider: sha3.NewLegacyKeccak256(),
	}
	return baseValueSet
}

// Integers converts the internal integers set into a standard list of integers. This may be inefficient to call often,
// so values obtained from this method should be cached in such cases.
func (vs *ValueSet) Integers() []*big.Int {
	res := make([]*big.Int, len(vs.integers))
	count := 0
	for _, v := range vs.integers {
		res[count] = v
		count++
	}
	return res
}

// AddInteger adds an integer item to the ValueSet.
func (vs *ValueSet) AddInteger(b *big.Int) {
	vs.integers[b.String()] = b
}

// Strings converts the internal string set into a standard list of strings. This may be inefficient to call often, so
// values obtained from this method should be cached in such cases.
func (vs *ValueSet) Strings() []string {
	res := make([]string, len(vs.strings))
	count := 0
	for k, _ := range vs.strings {
		res[count] = k
		count++
	}
	return res
}

// AddString adds a string item to the ValueSet.
func (vs *ValueSet) AddString(s string) {
	vs.strings[s] = nil
}

// Bytes converts the internal bytes set into a standard list of byte arrays. This may be inefficient to call often, so
// values obtained from this method should be cached in such cases.
func (vs *ValueSet) Bytes() [][]byte {
	res := make([][]byte, len(vs.bytes))
	count := 0
	for _, v := range vs.bytes {
		res[count] = v
		count++
	}
	return res
}

// AddBytes adds a byte sequence to the ValueSet.
func (vs *ValueSet) AddBytes(b []byte) {
	// Calculate hash and reset our hash provider
	vs.hashProvider.Write(b)
	hashStr := hex.EncodeToString(vs.hashProvider.Sum(nil))
	vs.hashProvider.Reset()

	// Add our hash to our "set" (map)
	vs.bytes[hashStr] = b
}
