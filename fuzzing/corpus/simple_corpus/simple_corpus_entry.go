package simple_corpus

import (
	"encoding/hex"
	"encoding/json"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus"
	"golang.org/x/crypto/sha3"
)

// SimpleCorpusEntry implements the CorpusEntry interface and represents a block sequence that increased coverage.
type SimpleCorpusEntry struct {
	blocks []*SimpleCorpusBlock
}

// NewSimpleCorpusEntry instantiates a new instance of SimpleCorpusEntry
func NewSimpleCorpusEntry() *SimpleCorpusEntry {
	return &SimpleCorpusEntry{
		blocks: []*SimpleCorpusBlock{},
	}
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

// Hash hashes the list of SimpleCorpusBlock in the SimpleCorpusEntry
func (m *SimpleCorpusEntry) Hash() (string, error) {
	// Concatenate the hashes of each block
	var simpleEntryHashString string
	for _, simpleBlock := range m.blocks {
		simpleBlockHash, err := simpleBlock.Hash()
		if err != nil {
			return "", err
		}
		simpleEntryHashString = simpleEntryHashString + simpleBlockHash
	}
	hash := sha3.NewLegacyKeccak256()
	// Hash the entire sequence of hashes
	_, err := hash.Write([]byte(simpleEntryHashString))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
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
