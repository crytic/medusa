package valuegeneration

import (
	"encoding/hex"
	"hash"
	"maps"
	"math/big"
	"reflect"

	"github.com/crytic/medusa/utils/reflectionutils"

	"github.com/crytic/medusa-geth/common"
	"golang.org/x/crypto/sha3"
)

// ValueSet represents potential values of significance within the source code to be used in fuzz tests.
type ValueSet struct {
	// addresses represents a set of common.Address to use in fuzz tests. A mapping is used to avoid duplicates.
	addresses map[common.Address]any
	// integers represents a set of integers to use in fuzz tests. A mapping is used to avoid duplicates.
	integers map[string]*big.Int
	// strings represents a set of strings to use in fuzz tests. A mapping is used to avoid duplicates.
	strings map[string]any
	// bytes represents a set of bytes to use in fuzz tests. A mapping is used to avoid duplicates.
	bytes map[string][]byte
	// hashProvider represents a hash provider used to create keys for some data.
	hashProvider hash.Hash
}

// NewValueSet initializes a new ValueSet object for use with a Fuzzer.
func NewValueSet() *ValueSet {
	baseValueSet := &ValueSet{
		addresses:    make(map[common.Address]any, 0),
		integers:     make(map[string]*big.Int, 0),
		strings:      make(map[string]any, 0),
		bytes:        make(map[string][]byte, 0),
		hashProvider: sha3.NewLegacyKeccak256(),
	}
	return baseValueSet
}

// Clone creates a copy of the current ValueSet.
func (vs *ValueSet) Clone() *ValueSet {
	baseValueSet := &ValueSet{
		addresses:    maps.Clone(vs.addresses),
		integers:     maps.Clone(vs.integers),
		strings:      maps.Clone(vs.strings),
		bytes:        maps.Clone(vs.bytes),
		hashProvider: sha3.NewLegacyKeccak256(),
	}
	return baseValueSet
}

// Addresses returns a list of addresses contained within the set.
func (vs *ValueSet) Addresses() []common.Address {
	res := make([]common.Address, len(vs.addresses))
	count := 0
	for k := range vs.addresses {
		res[count] = k
		count++
	}
	return res
}

// AddAddress adds an address item to the ValueSet.
func (vs *ValueSet) AddAddress(a common.Address) {
	vs.addresses[a] = nil
}

// ContainsAddress checks if an address is contained in the ValueSet.
func (vs *ValueSet) ContainsAddress(a common.Address) bool {
	_, contains := vs.addresses[a]
	return contains
}

// RemoveAddress removes an address item from the ValueSet.
func (vs *ValueSet) RemoveAddress(a common.Address) {
	delete(vs.addresses, a)
}

// Integers returns a list of integers contained within the set.
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

// ContainsInteger checks if an integer is contained in the ValueSet.
func (vs *ValueSet) ContainsInteger(b *big.Int) bool {
	_, contains := vs.integers[b.String()]
	return contains
}

// RemoveInteger removes an integer item from the ValueSet.
func (vs *ValueSet) RemoveInteger(b *big.Int) {
	delete(vs.integers, b.String())
}

// Strings returns a list of strings contained within the set.
func (vs *ValueSet) Strings() []string {
	res := make([]string, len(vs.strings))
	count := 0
	for k := range vs.strings {
		res[count] = k
		count++
	}
	return res
}

// AddString adds a string item to the ValueSet.
func (vs *ValueSet) AddString(s string) {
	vs.strings[s] = nil
}

// ContainsString checks if a string is contained in the ValueSet.
func (vs *ValueSet) ContainsString(s string) bool {
	_, contains := vs.strings[s]
	return contains
}

// RemoveString removes a string item from the ValueSet.
func (vs *ValueSet) RemoveString(s string) {
	delete(vs.strings, s)
}

// Bytes returns a list of bytes contained within the set.
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

// ContainsBytes checks if a byte sequence is contained in the ValueSet.
func (vs *ValueSet) ContainsBytes(b []byte) bool {
	// Calculate hash and reset our hash provider
	vs.hashProvider.Write(b)
	hashStr := hex.EncodeToString(vs.hashProvider.Sum(nil))
	vs.hashProvider.Reset()

	// Check if the key exists in our lookup
	_, contains := vs.bytes[hashStr]
	return contains
}

// RemoveBytes removes a byte sequence item from the ValueSet.
func (vs *ValueSet) RemoveBytes(b []byte) {
	// Calculate hash and reset our hash provider
	vs.hashProvider.Write(b)
	hashStr := hex.EncodeToString(vs.hashProvider.Sum(nil))
	vs.hashProvider.Reset()

	delete(vs.bytes, hashStr)
}

// Add adds one or more values. Note the values must be a primitive type (signed/unsigned integer, address, string,
// bytes, fixed bytes)
func (vs *ValueSet) Add(values []any) {
	// Iterate across each value and assert on its type
	for _, value := range values {
		switch v := value.(type) {
		case uint8:
			vs.AddInteger(new(big.Int).SetUint64(uint64(v)))
		case uint16:
			vs.AddInteger(new(big.Int).SetUint64(uint64(v)))
		case uint32:
			vs.AddInteger(new(big.Int).SetUint64(uint64(v)))
		case uint64:
			vs.AddInteger(new(big.Int).SetUint64(v))
		case int8:
			vs.AddInteger(new(big.Int).SetInt64(int64(v)))
		case int16:
			vs.AddInteger(new(big.Int).SetInt64(int64(v)))
		case int32:
			vs.AddInteger(new(big.Int).SetInt64(int64(v)))
		case int64:
			vs.AddInteger(new(big.Int).SetInt64(v))
		case *big.Int:
			vs.AddInteger(v)
		case common.Address:
			vs.AddAddress(v)
		case bool:
			if value == true {
				vs.AddInteger(new(big.Int).SetUint64(1))
			} else {
				vs.AddInteger(new(big.Int).SetUint64(0))
			}
		case string:
			vs.AddString(v)
		case []byte:
			vs.AddBytes(v)
		default:
			// We need to be able to capture fixed bytes. Unfortunately, the only way to do so is using reflection
			r := reflect.TypeOf(value)
			// If we have a fixed array of uint8 (aka byte), then we will convert it into a slice and add to value set
			if r.Kind() == reflect.Array && r.Elem().Kind() == reflect.Uint8 {
				b := reflectionutils.ArrayToSlice(reflect.ValueOf(value)).([]byte)
				vs.AddBytes(b)
			}
			continue
		}
	}
}
