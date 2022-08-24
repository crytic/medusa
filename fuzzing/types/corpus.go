package types

import (
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
	"math/big"
	"strconv"
	"strings"
	"sync"
)

// Corpus holds the list of block sequences that increased fuzz coverage
type Corpus struct {
	// CorpusBlockSequences is a mapping between a hashed string and a CorpusBlockSequence
	CorpusBlockSequences map[string]*CorpusBlockSequence
	// Mutex is used to prevent races to write and read from corpus
	Mutex sync.Mutex
	// WriteIndex is an index in the CorpusBlockSequences list that points to the next object to be written to disk
	// TODO: Still need to figure out if we need this
	WriteIndex int
}

// NewCorpus initializes a new Corpus object for the Fuzzer
func NewCorpus() *Corpus {
	return &Corpus{
		CorpusBlockSequences: map[string]*CorpusBlockSequence{},
		WriteIndex:           0,
	}
}

// CorpusBlockSequence is a list of CorpusBlock
type CorpusBlockSequence []*CorpusBlock

// Hash hashes the list of CorpusBlock in the CorpusBlockSequence
func (m CorpusBlockSequence) Hash() (string, error) {
	// Concatenate the hashes of each block
	var corpusBlockSequenceHashString string
	for _, block := range m {
		blockHash, err := block.Hash()
		if err != nil {
			return "", err
		}
		corpusBlockSequenceHashString = corpusBlockSequenceHashString + blockHash
	}
	hash := sha3.NewLegacyKeccak256()
	// Hash the entire sequence of hashes
	_, err := hash.Write([]byte(corpusBlockSequenceHashString))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// MarshalJSON marshals the CorpusBlockSequence object into JSON
func (m CorpusBlockSequence) MarshalJSON() ([]byte, error) {
	type CorpusBlockSequence struct {
		CorpusBlockSequence []*CorpusBlock `json:"sequence"`
	}
	var enc CorpusBlockSequence
	enc.CorpusBlockSequence = m
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals a byte string into a CorpusBlockSequence object.
func (m *CorpusBlockSequence) UnmarshalJSON(input []byte) error {
	type CorpusBlockSequence struct {
		CorpusBlockSequence []*CorpusBlock `json:"sequence"`
	}
	var dec CorpusBlockSequence
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.CorpusBlockSequence != nil {
		*m = dec.CorpusBlockSequence
	}
	return nil
}

// CorpusBlock is an abstraction of a fuzzing.TestNodeBlock. It includes some core header components in Header, transactions
// in Transactions, and receipts in Receipts
type CorpusBlock struct {
	// Header is of type CorpusBlockHeader and holds the block hash, timestamp, and number
	Header CorpusBlockHeader
	// Transactions is a list of CallMessage. This is the sequence of transactions that increased coverage.
	Transactions []*CallMessage
	// Receipts is a list of types.Receipt. Each receipt is associated with a transaction in CorpusBlock.Transactions
	Receipts []*types.Receipt
}

// Hash hashes the fields of a CorpusBlock
func (m *CorpusBlock) Hash() (string, error) {
	// Concatenate the hashes of each CallMessage
	var txnSequenceHashString string
	for _, txn := range m.Transactions {
		txnHash, err := txn.Hash()
		if err != nil {
			return "", err
		}
		txnSequenceHashString = txnSequenceHashString + txnHash
	}
	// Concatenate the hash of the header and txns
	headerHash, err := m.Header.Hash()
	if err != nil {
		return "", err
	}
	blockSequenceString := strings.Join([]string{headerHash, txnSequenceHashString}, ",")
	hash := sha3.NewLegacyKeccak256()
	// TODO: not checking returned error from hash.Write. can add this later
	// Hash the entire sequence
	_, err = hash.Write([]byte(blockSequenceString))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// MarshalJSON marshals the CorpusBlock object into JSON
func (m CorpusBlock) MarshalJSON() ([]byte, error) {
	type CorpusBlock struct {
		MetaHeader   CorpusBlockHeader `json:"header"`
		Transactions []*CallMessage    `json:"transactions"`
		Receipts     []*types.Receipt  `json:"receipts"`
	}
	var enc CorpusBlock
	enc.MetaHeader = m.Header
	enc.Transactions = m.Transactions
	enc.Receipts = m.Receipts
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals a byte string into a CorpusBlock object.
func (m *CorpusBlock) UnmarshalJSON(input []byte) error {
	type MetaBlock struct {
		MetaHeader   *CorpusBlockHeader `json:"header"`
		Transactions []*CallMessage     `json:"transactions"`
		Receipts     []*types.Receipt   `json:"receipts"`
	}
	var dec MetaBlock
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.MetaHeader != nil {
		m.Header = *dec.MetaHeader
	}
	if dec.Transactions != nil {
		m.Transactions = dec.Transactions
	}
	if dec.Receipts != nil {
		m.Receipts = dec.Receipts
	}
	return nil
}

// NewCorpusBlock instantiates a new instance of CorpusBlock
func NewCorpusBlock() *CorpusBlock {
	return &CorpusBlock{
		Header:       CorpusBlockHeader{},
		Transactions: []*CallMessage{},
		Receipts:     []*types.Receipt{},
	}
}

// CorpusBlockHeader holds a few core components of a fuzzing.TestNodeBlock header such as block hash, timestamp, and number
type CorpusBlockHeader struct {
	// BlockHash is the block hash of a block
	BlockHash common.Hash
	// BlockTimestamp is the block timestamp of a block
	BlockTimestamp uint64
	// BlockNumber is the block number of a block
	BlockNumber *big.Int
}

// Hash hashes the fields of a CorpusBlockHeader
func (h *CorpusBlockHeader) Hash() (string, error) {
	// Stringify the header
	corpusBlockHeaderHashString := strings.Join([]string{h.BlockHash.String(), strconv.FormatUint(h.BlockTimestamp, 10), h.BlockNumber.String()}, ",")
	hash := sha3.NewLegacyKeccak256()
	// Hash the header string
	_, err := hash.Write([]byte(corpusBlockHeaderHashString))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// MarshalJSON marshals the CorpusBlockHeader object into JSON
func (m CorpusBlockHeader) MarshalJSON() ([]byte, error) {
	type CorpusBlockHeader struct {
		BlockHash      common.Hash `json:"block_hash"`
		BlockTimestamp uint64      `json:"block_timestamp"`
		BlockNumber    *big.Int    `json:"block_number"`
	}
	var enc CorpusBlockHeader
	enc.BlockHash = m.BlockHash
	enc.BlockTimestamp = m.BlockTimestamp
	enc.BlockNumber = m.BlockNumber
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals a byte string into a CorpusBlockHeader object.
func (m *CorpusBlockHeader) UnmarshalJSON(input []byte) error {
	type CorpusBlockHeader struct {
		BlockHash      *common.Hash `json:"block_hash"`
		BlockTimestamp *uint64      `json:"block_timestamp"`
		BlockNumber    *big.Int     `json:"block_number"`
	}
	var dec CorpusBlockHeader
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.BlockHash != nil {
		m.BlockHash = *dec.BlockHash
	}
	if dec.BlockTimestamp != nil {
		m.BlockTimestamp = *dec.BlockTimestamp
	}
	if dec.BlockNumber != nil {
		m.BlockNumber = dec.BlockNumber
	}
	return nil
}
