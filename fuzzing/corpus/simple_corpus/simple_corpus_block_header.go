package simple_corpus

import (
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
	"math/big"
	"strconv"
	"strings"
)

// SimpleCorpusBlockHeader implements the CorpusBlockHeader interface and holds a few core components of a
// fuzzing.TestNodeBlock header such as block hash, timestamp, and number
type SimpleCorpusBlockHeader struct {
	// blockHeaderHash is the block hash of a block
	blockHeaderHash common.Hash
	// blockHeaderTimestamp is the block timestamp of a block
	blockHeaderTimestamp uint64
	// blockHeaderNumber is the block number of a block
	blockHeaderNumber *big.Int
}

// NewSimpleCorpusBlockHeader instantiates a new instance of SimpleCorpusBlockHeader
func NewSimpleCorpusBlockHeader(blockHash common.Hash, blockTimestamp uint64, blockNumber *big.Int) *SimpleCorpusBlockHeader {
	return &SimpleCorpusBlockHeader{
		blockHeaderHash:      blockHash,
		blockHeaderTimestamp: blockTimestamp,
		blockHeaderNumber:    blockNumber,
	}
}

// BlockHash returns the block hash
func (m *SimpleCorpusBlockHeader) BlockHash() common.Hash {
	return m.blockHeaderHash
}

// BlockTimestamp returns the block timestamp
func (m *SimpleCorpusBlockHeader) BlockTimestamp() uint64 {
	return m.blockHeaderTimestamp
}

// BlockNumber returns the block number
func (m *SimpleCorpusBlockHeader) BlockNumber() *big.Int {
	return m.blockHeaderNumber
}

// Hash hashes the contents of a SimpleCorpusBlockHeader
func (h *SimpleCorpusBlockHeader) Hash() (string, error) {
	// Stringify the header
	simpleHeaderHashString := strings.Join([]string{
		h.BlockHash().String(),
		strconv.FormatUint(h.BlockTimestamp(), 10),
		h.BlockNumber().String()}, ",")
	hash := sha3.NewLegacyKeccak256()
	// Hash the header string
	_, err := hash.Write([]byte(simpleHeaderHashString))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// MarshalJSON marshals the SimpleCorpusBlockHeader object into JSON
func (m SimpleCorpusBlockHeader) MarshalJSON() ([]byte, error) {
	type SimpleCorpusBlockHeader struct {
		BlockHeaderHash      common.Hash `json:"block_hash"`
		BlockHeaderTimestamp uint64      `json:"block_timestamp"`
		BlockHeaderNumber    *big.Int    `json:"block_number"`
	}
	var enc SimpleCorpusBlockHeader
	enc.BlockHeaderHash = m.BlockHash()
	enc.BlockHeaderTimestamp = m.BlockTimestamp()
	enc.BlockHeaderNumber = m.BlockNumber()
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals a byte string into a SimpleCorpusBlockHeader object.
func (m *SimpleCorpusBlockHeader) UnmarshalJSON(input []byte) error {
	type SimpleCorpusBlockHeader struct {
		BlockHeaderHash      *common.Hash `json:"block_hash"`
		BlockHeaderTimestamp *uint64      `json:"block_timestamp"`
		BlockHeaderNumber    *big.Int     `json:"block_number"`
	}
	var dec SimpleCorpusBlockHeader
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.BlockHeaderHash != nil {
		m.blockHeaderHash = *dec.BlockHeaderHash
	}
	if dec.BlockHeaderTimestamp != nil {
		m.blockHeaderTimestamp = *dec.BlockHeaderTimestamp
	}
	if dec.BlockHeaderTimestamp != nil {
		m.blockHeaderNumber = dec.BlockHeaderNumber
	}
	return nil
}
