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

//Corpus holds the list of block sequences that increased fuzz coverage
type Corpus struct {
	//MetaBlockSequences is a mapping between a hashed string and a MetaBlockSequence
	MetaBlockSequences map[string]*MetaBlockSequence
	//Mutex is used to prevent races to write and read from corpus
	Mutex sync.Mutex
	//WriteIndex is an index in the TransactionSequences list that points to the next object to be written to disk
	//TODO: Still need to figure out if we need this
	WriteIndex int
}

//NewCorpus initializes a new Corpus object for the Fuzzer
func NewCorpus() *Corpus {
	return &Corpus{
		MetaBlockSequences: map[string]*MetaBlockSequence{},
		WriteIndex:         0,
	}
}

//MetaBlockSequence is a list of MetaBlock
type MetaBlockSequence []*MetaBlock

//Hash hashes the list of MetaBlock in the MetaBlockSequence
func (m MetaBlockSequence) Hash() string {
	// Concatenate the hashes of each block
	var metaBlockSequenceHashString string
	for _, block := range m {
		metaBlockSequenceHashString = metaBlockSequenceHashString + block.Hash()
	}
	hash := sha3.NewLegacyKeccak256()
	// TODO: not checking returned error from hash.Write. can add this later
	// Hash the entire sequence of hashes
	hash.Write([]byte(metaBlockSequenceHashString))
	return hex.EncodeToString(hash.Sum(nil))
}

//MarshalJSON marshals the MetaBlockSequence object into JSON
func (m MetaBlockSequence) MarshalJSON() ([]byte, error) {
	type MetaBlockSequence struct {
		MetaBlockSequence []*MetaBlock `json:"sequence"`
	}
	var enc MetaBlockSequence
	enc.MetaBlockSequence = m
	return json.Marshal(&enc)
}

//MetaBlock is an abstraction of a fuzzing.TestNodeBlock. It includes some core header components in Header, transactions
//in Transactions, and receipts in Receipts
type MetaBlock struct {
	//Header is of type MetaHeader and holds the block hash, timestamp, and number
	Header MetaHeader
	//Transactions is a list of CallMessage. This is the sequence of transactions that increased coverage.
	Transactions []*CallMessage
	//Receipts is a list of types.Receipt. Each receipt is associated with a transaction in MetaBlock.Transactions
	Receipts []*types.Receipt
}

//Hash hashes the fields of a MetaBlock
func (m *MetaBlock) Hash() string {
	// Concatenate the hashes of each CallMessage
	var txnSequenceHashString string
	for _, txn := range m.Transactions {
		txnSequenceHashString = txnSequenceHashString + txn.Hash()
	}
	// Concatenate the hash of the header and txns
	blockSequenceString := strings.Join([]string{m.Header.Hash(), txnSequenceHashString}, ",")
	hash := sha3.NewLegacyKeccak256()
	// TODO: not checking returned error from hash.Write. can add this later
	// Hash the entire sequence
	hash.Write([]byte(blockSequenceString))
	return hex.EncodeToString(hash.Sum(nil))
}

//MarshalJSON marshals the MetaBlock object into JSON
func (m MetaBlock) MarshalJSON() ([]byte, error) {
	type MetaBlock struct {
		MetaHeader   MetaHeader       `json:"header"`
		Transactions []*CallMessage   `json:"transactions"`
		Receipts     []*types.Receipt `json:"receipts"`
	}
	var enc MetaBlock
	enc.MetaHeader = m.Header
	enc.Transactions = m.Transactions
	enc.Receipts = m.Receipts
	return json.Marshal(&enc)
}

//NewMetaBlock instantiates a new instance of MetaBlock
func NewMetaBlock() *MetaBlock {
	return &MetaBlock{
		Header:       MetaHeader{},
		Transactions: []*CallMessage{},
		Receipts:     []*types.Receipt{},
	}
}

//MetaHeader holds a few core components of a fuzzing.TestNodeBlock header such as block hash, timestamp, and number
type MetaHeader struct {
	//BlockHash is the block hash of a block
	BlockHash common.Hash
	//BlockTimestamp is the block timestamp of a block
	BlockTimestamp uint64
	//BlockNumber is the block number of a block
	BlockNumber *big.Int
}

//Hash hashes the fields of a MetaHeader
func (h *MetaHeader) Hash() string {
	// Stringify the header
	metaHeaderHashString := strings.Join([]string{h.BlockHash.String(), strconv.FormatUint(h.BlockTimestamp, 10), h.BlockNumber.String()}, ",")
	hash := sha3.NewLegacyKeccak256()
	// TODO: not checking returned error from hash.Write. can add this later
	// Hash the header string
	hash.Write([]byte(metaHeaderHashString))
	return hex.EncodeToString(hash.Sum(nil))
}

//MarshalJSON marshals the MetaHeader object into JSON
func (m MetaHeader) MarshalJSON() ([]byte, error) {
	type MetaHeader struct {
		BlockHash      common.Hash `json:"block_hash"`
		BlockTimestamp uint64      `json:"block_timestamp"`
		BlockNumber    *big.Int    `json:"block_number"`
	}
	var enc MetaHeader
	enc.BlockHash = m.BlockHash
	enc.BlockTimestamp = m.BlockTimestamp
	enc.BlockNumber = m.BlockNumber
	return json.Marshal(&enc)
}
