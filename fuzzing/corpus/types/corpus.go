package types

import (
	"github.com/ethereum/go-ethereum/common"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/fuzzing/testnode"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
)

// Corpus is the generic interface that a corpus type should implement for coverage-based testing
type Corpus interface {
	// Entries returns the list of CorpusEntry objects that are stored in the Corpus
	Entries() []CorpusEntry
	// AddEntry adds a CorpusEntry to the corpus and returns an error in case of an issue
	AddEntry(entry CorpusEntry) error
	// RemoveEntry removes a CorpusEntry from the corpus and returns an error in case of an issue
	RemoveEntry(entry CorpusEntry) error
	// RemoveEntryAt removes the CorpusEntry at index from the corpus and returns an error in case of an issue
	RemoveEntryAt(index uint64) error
	// GetRandomEntry returns a random CorpusEntry from the Corpus and throws an error in case of an issue
	GetRandomEntry() (CorpusEntry, error)
	// GetEntry returns the CorpusEntry at index and returns an error in case of an issue
	GetEntry(index uint64) (CorpusEntry, error)
	// WriteCorpusToDisk writes the Corpus to disk at writeDirectory and throws an error in case of an issue
	WriteCorpusToDisk(writeDirectory string) error
	// ReadCorpusFromDisk reads the Corpus from disk at readDirectory and throws an error in case of an issue
	ReadCorpusFromDisk(readDirectory string) error
	// TODO: Note for David - should this function be here? I added it here so that the corpus is responsible for this task
	// but it is not a very generic function.
	// TestSequenceToCorpusEntry takes a testNodeBlockSequence and a txSequence and converts it into a corpus entry
	TestSequenceToCorpusEntry(testNodeBlockSequence []*testnode.TestNodeBlock, txSequence []*fuzzerTypes.CallMessage) (CorpusEntry, error)
}

// CorpusEntry is the generic interface for a single entry in the Corpus. The Corpus is simply a list of corpus entries.
type CorpusEntry interface {
	// Blocks returns the list of CorpusBlock objects that are stored in the CorpusEntry
	Blocks() []CorpusBlock
	// AddCorpusBlock adds a CorpusBlock to the list of blocks in a CorpusEntry
	AddCorpusBlock(block CorpusBlock) error
	// Hash hashes the contents of a CorpusEntry
	Hash() (string, error)
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
	Transactions() []*fuzzerTypes.CallMessage
	// Receipts returns the receipts of the CorpusBlock
	Receipts() []*coreTypes.Receipt
	// Hash hashes the contents of a CorpusBlock
	Hash() (string, error)
	// MarshalJSON marshals the CorpusBlock object into a JSON object
	MarshalJSON() ([]byte, error)
	// UnmarshalJSON unmarshals a JSON object into a CorpusBlock object
	UnmarshalJSON(input []byte) error
}

// CorpusBlockHeader is the generic interface for the block header of a CorpusBlock.
type CorpusBlockHeader interface {
	// BlockHash returns the block hash
	BlockHash() common.Hash
	// BlockTimestamp returns the block timestamp
	BlockTimestamp() uint64
	// BlockNumber returns the block number
	BlockNumber() *big.Int
	// Hash hashes the contents of a CorpusBlockHeader
	Hash() (string, error)
	// MarshalJSON marshals the CorpusBlockHeader object into a JSON object
	MarshalJSON() ([]byte, error)
	// UnmarshalJSON unmarshals a JSON object into a CorpusBlockHeader object
	UnmarshalJSON(input []byte) error
}
