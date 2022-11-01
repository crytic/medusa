package corpus

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	"github.com/trailofbits/medusa/utils"
	"math/big"
)

// The following directives will be picked up by the `go generate` command to generate JSON marshaling code from
// templates defined below. They should be preserved for re-use in case we change our structures.
//go:generate go get github.com/fjl/gencodec
//go:generate go run github.com/fjl/gencodec -type CorpusBlockHeader -field-override corpusBlockHeaderMarshaling -out gen_corpus_call_sequence.go

// CorpusCallSequence represents a Corpus structure that contains a coverage increasing block sequence.
type CorpusCallSequence struct {
	// Blocks represents the corpus representation of a block sequence provided by this entry.
	Blocks []CorpusBlock `json:"blocks"`
}

// CorpusBlock implements the CorpusBlock interface and is a simplified version of a fuzzing.TestNodeBlock.
type CorpusBlock struct {
	// Header is a pointer to a CorpusBlockHeader and holds the block hash, timestamp, and block number
	Header CorpusBlockHeader `json:"header"`
	// Transactions is a list of CallMessages. This is the sequence of transactions that increased coverage.
	Transactions []chainTypes.CallMessage `json:"transactions"`
}

// CorpusBlockHeader defines the Corpus representation of a block header.
type CorpusBlockHeader struct {
	// Number is the block number of the block this header belongs to.
	Number *big.Int `json:"number"`
	// Timestamp is the block timestamp of the block this header belongs to.
	Timestamp uint64 `json:"timestamp"`
}

// corpusBlockHeaderMarshaling overrides the JSON marshaling of CorpusBlockHeader using the `go generate` statement
// earlier in this file to provide encoding overrides for the underlying fields.
type corpusBlockHeaderMarshaling struct {
	Number *hexutil.Big
}

// NewCorpusEntry returns a new instance of CorpusCallSequence with the provided block sequence and state root hash prior to
// the inclusion of the blocks.
func NewCorpusEntry(blocks []*chainTypes.Block) *CorpusCallSequence {
	// Convert the provided chain blocks to corpus block structures.
	corpusBlocks := make([]CorpusBlock, len(blocks))
	for i := 0; i < len(blocks); i++ {
		corpusBlocks[i] = CorpusBlock{
			Header: CorpusBlockHeader{
				Number:    blocks[i].Header().Number,
				Timestamp: blocks[i].Header().Time,
			},
			Transactions: utils.SlicePointersToValues(blocks[i].Messages()),
		}
	}

	// Create our corpus entry with the provided state root hash and the corpus blocks.
	return &CorpusCallSequence{
		Blocks: corpusBlocks,
	}
}
