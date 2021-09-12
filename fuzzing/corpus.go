package fuzzing

import (
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"hash"
	"math/big"
)

// Corpus represents a fuzzing corpus, a set of inputs for a given target. This represents potential values of
// significance within the source code to be used in fuzz tests.
type Corpus struct {
	// integers represents a set of integers to use in fuzz tests. A mapping is used to avoid duplicates.
	integers map[string]big.Int
	// strings represents a set of strings to use in fuzz tests. A mapping is used to avoid duplicates.
	strings map[string]interface{}
	// bytes represents a set of bytes to use in fuzz tests. A mapping is used to avoid duplicates.
	bytes map[string][]byte
	// hashProvider represents a hash provider used to discern duplicate corpus entries.
	hashProvider hash.Hash
}

// NewCorpus initializes a new Corpus object for use with a Fuzzer.
func NewCorpus() *Corpus {
	corpus := &Corpus {
		integers: make(map[string]big.Int, 0),
		strings: make(map[string]interface{}, 0),
		bytes: make(map[string][]byte, 0),
		hashProvider: sha3.New256(),
	}
	return corpus
}

// Integers converts internal corpus strings into a standard list of integers. This may be inefficient to call often, so
// values obtained from this method should be cached in such cases.
func (c *Corpus) Integers() []big.Int {
	res := make([]big.Int, len(c.integers))
	count := 0
	for _, v := range c.integers {
		res[count] = v
		count++
	}
	return res
}

// AddInteger adds an integer item to the corpus.
func (c *Corpus) AddInteger(b big.Int) {
	c.integers[b.String()] = b
}

// Strings converts internal corpus strings into a standard list of strings. This may be inefficient to call often, so
// values obtained from this method should be cached in such cases.
func (c *Corpus) Strings() []string {
	res := make([]string, len(c.strings))
	count := 0
	for k, _ := range c.strings {
		res[count] = k
		count++
	}
	return res
}

// AddString adds a string item to the corpus.
func (c *Corpus) AddString(s string) {
	c.strings[s] = nil
}

// Bytes converts internal corpus bytes into a standard list of byte arrays. This may be inefficient to call often, so
// values obtained from this method should be cached in such cases.
func (c *Corpus) Bytes() [][]byte {
	res := make([][]byte, len(c.bytes))
	count := 0
	for _, v := range c.bytes {
		res[count] = v
		count++
	}
	return res
}

// AddBytes adds a byte sequence to the corpus.
func (c *Corpus) AddBytes(b []byte) {
	// Calculate hash and reset our hash provider
	c.hashProvider.Write(b)
	hashStr := hex.EncodeToString(c.hashProvider.Sum(nil))
	c.hashProvider.Reset()

	// Add our hash to our "set" (map)
	c.bytes[hashStr] = b
}