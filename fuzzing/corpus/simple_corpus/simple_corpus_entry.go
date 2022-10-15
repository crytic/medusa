package simple_corpus

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus"
)

// SimpleCorpusEntry implements the CorpusEntry interface and represents a block sequence that increased coverage.
type SimpleCorpusEntry struct {
	// stateRoot represents the ethereum state root hash of the world state prior to the processing of the first block
	// in the Blocks sequence. This may be nil if it was not recorded, or validation of the pre-processing state root
	// hash should not be enforced for this entry.
	stateRoot *common.Hash

	// blocks represents the corpus representation of a block sequence provided by this entry.
	blocks []*SimpleCorpusBlock
}

// NewSimpleCorpusEntry instantiates a new instance of SimpleCorpusEntry
func NewSimpleCorpusEntry(stateRoot *common.Hash, blocks []*chainTypes.Block) *SimpleCorpusEntry {
	// Convert the provided chain blocks to corpus block structures.
	corpusBlocks := make([]*SimpleCorpusBlock, len(blocks))
	for i := 0; i < len(corpusBlocks); i++ {
		corpusBlocks[i] = NewSimpleCorpusBlockFromTestChainBlock(blocks[i])
	}

	// Create our corpus entry with the provided state root hash and the corpus blocks.
	return &SimpleCorpusEntry{
		stateRoot: stateRoot,
		blocks:    corpusBlocks,
	}
}

// StateRoot represents the ethereum state root hash of the world state prior to the processing of the first block
// in the Blocks sequence. This may be nil if it was not recorded, or validation of the pre-processing state root
// hash should not be enforced for this entry.
func (m *SimpleCorpusEntry) StateRoot() *common.Hash {
	return m.stateRoot
}

// Blocks returns the list of SimpleCorpusBlock objects that are stored in the SimpleCorpusEntry
func (m *SimpleCorpusEntry) Blocks() []corpusTypes.CorpusBlock {
	var blocks []corpusTypes.CorpusBlock
	for _, simpleBlock := range m.blocks {
		block := corpusTypes.CorpusBlock(simpleBlock)
		blocks = append(blocks, block)
	}
	return blocks
}

// MarshalJSON marshals the SimpleCorpusEntry object into JSON
func (m SimpleCorpusEntry) MarshalJSON() ([]byte, error) {
	type SimpleCorpusEntry struct {
		CorpusBlocks []*SimpleCorpusBlock `json:"sequence"`
	}
	var enc SimpleCorpusEntry
	enc.CorpusBlocks = m.blocks
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals a byte string into a SimpleCorpusEntry object.
func (m *SimpleCorpusEntry) UnmarshalJSON(input []byte) error {
	type SimpleCorpusEntry struct {
		CorpusBlocks []*SimpleCorpusBlock `json:"sequence"`
	}
	var dec SimpleCorpusEntry
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.CorpusBlocks != nil {
		m.blocks = dec.CorpusBlocks
	}
	return nil
}
