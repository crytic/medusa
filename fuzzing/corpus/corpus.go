package corpus

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

// Corpus is the generic interface that a corpus type should implement for coverage-based testing
type Corpus interface {
	// Entries returns the list of CorpusEntry objects that are stored in the Corpus
	Entries() []CorpusEntry
	// AddEntry adds a CorpusEntry to the corpus and returns an error in case of an issue
	AddEntry(entry CorpusEntry) error
	// WriteCorpusToDisk writes the Corpus to disk at writeDirectory and throws an error in case of an issue
	WriteCorpusToDisk(writeDirectory string) error
	// ReadCorpusFromDisk reads the Corpus from disk at readDirectory and throws an error in case of an issue
	ReadCorpusFromDisk(readDirectory string) error
}

// CorpusEntry is the generic interface for a single entry in the Corpus. It represents a sequence of blocks processed
// over some initial state that produced an interesting result (e.g., increased coverage).
type CorpusEntry interface {
	// StateRoot represents the ethereum state root hash of the world state prior to the processing of the first block
	// in the Blocks sequence. This may be nil if it was not recorded, or validation of the pre-processing state root
	// hash should not be enforced for this entry.
	StateRoot() *common.Hash
	// Blocks returns the list of CorpusBlock objects that are stored in the CorpusEntry
	Blocks() []CorpusBlock
	// MarshalJSON marshals the CorpusEntry object into a JSON object
	MarshalJSON() ([]byte, error)
	// UnmarshalJSON unmarshals a JSON object into a CorpusEntry object
	UnmarshalJSON(input []byte) error
}

// CorpusBlock is the generic interface for a single block in a CorpusEntry. A CorpusEntry is simply a list of corpus blocks
type CorpusBlock interface {
	// Header returns the CorpusBlockHeader of the CorpusBlock
	Header() CorpusBlockHeader
	// Transactions returns the transactions of the CorpusBlock
	Transactions() []core.Message
	// Receipts returns the receipts of the CorpusBlock
	Receipts() []*coreTypes.Receipt
}

// CorpusBlockHeader is the generic interface for the block header of a CorpusBlock.
type CorpusBlockHeader interface {
	// BlockTimestamp returns the block timestamp
	BlockTimestamp() uint64
	// BlockNumber returns the block number
	BlockNumber() *big.Int
}
