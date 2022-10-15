package simple_corpus

import (
	"encoding/json"
	"math/big"
)

// SimpleCorpusBlockHeader implements the CorpusBlockHeader interface and holds a few core components of a
// fuzzing.TestNodeBlock header such as block hash, timestamp, and number
type SimpleCorpusBlockHeader struct {
	// timestamp is the block timestamp of a block
	timestamp uint64
	// blockNumber is the block number of a block
	blockNumber *big.Int
}

// NewSimpleCorpusBlockHeader instantiates a new instance of SimpleCorpusBlockHeader
func NewSimpleCorpusBlockHeader(blockTimestamp uint64, blockNumber *big.Int) *SimpleCorpusBlockHeader {
	return &SimpleCorpusBlockHeader{
		timestamp:   blockTimestamp,
		blockNumber: blockNumber,
	}
}

// BlockTimestamp returns the block timestamp
func (m *SimpleCorpusBlockHeader) BlockTimestamp() uint64 {
	return m.timestamp
}

// BlockNumber returns the block number
func (m *SimpleCorpusBlockHeader) BlockNumber() *big.Int {
	return m.blockNumber
}

// MarshalJSON marshals the SimpleCorpusBlockHeader object into JSON
func (m SimpleCorpusBlockHeader) MarshalJSON() ([]byte, error) {
	type SimpleCorpusBlockHeader struct {
		BlockHeaderNumber    *big.Int `json:"block_number"`
		BlockHeaderTimestamp uint64   `json:"block_timestamp"`
	}
	var enc SimpleCorpusBlockHeader
	enc.BlockHeaderTimestamp = m.BlockTimestamp()
	enc.BlockHeaderNumber = m.BlockNumber()
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals a byte string into a SimpleCorpusBlockHeader object.
func (m *SimpleCorpusBlockHeader) UnmarshalJSON(input []byte) error {
	type SimpleCorpusBlockHeader struct {
		BlockHeaderNumber    *big.Int `json:"block_number"`
		BlockHeaderTimestamp *uint64  `json:"block_timestamp"`
	}
	var dec SimpleCorpusBlockHeader
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.BlockHeaderTimestamp != nil {
		m.timestamp = *dec.BlockHeaderTimestamp
	}
	if dec.BlockHeaderTimestamp != nil {
		m.blockNumber = dec.BlockHeaderNumber
	}
	return nil
}
