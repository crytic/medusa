package simple_corpus

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/core"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus"
)

// SimpleCorpusBlock implements the CorpusBlock interface and is a simplified version of a fuzzing.TestNodeBlock.
type SimpleCorpusBlock struct {
	// blockHeader is a pointer to a SimpleCorpusBlockHeader and holds the block hash, timestamp, and block number
	blockHeader *SimpleCorpusBlockHeader
	// blockTransactions is a list of CallMessages. This is the sequence of transactions that increased coverage.
	blockTransactions []*chainTypes.CallMessage
	// blockReceipts is a list of core.Receipt.
	blockReceipts []*coreTypes.Receipt
}

// NewSimpleCorpusBlock instantiates a new instance of SimpleCorpusBlock
func NewSimpleCorpusBlock() *SimpleCorpusBlock {
	return &SimpleCorpusBlock{
		blockHeader:       &SimpleCorpusBlockHeader{},
		blockTransactions: []*chainTypes.CallMessage{},
		blockReceipts:     []*coreTypes.Receipt{},
	}
}

// NewSimpleCorpusBlockFromTestChainBlock converts a test chain block to a SimpleCorpusBlock. Components such as the
// block timestamp, block number, block hash, transaction receipts, and the transactions are maintained during conversion.
func NewSimpleCorpusBlockFromTestChainBlock(block *chainTypes.Block) *SimpleCorpusBlock {
	// Create our corpus block
	corpusBlock := NewSimpleCorpusBlock()

	// Set header fields
	corpusBlock.blockHeader = NewSimpleCorpusBlockHeader(
		block.Header().Time,
		block.Header().Number)

	// Update our transactions and receipts
	corpusBlock.blockTransactions = block.Messages()
	corpusBlock.blockReceipts = block.Receipts()

	// TODO: Can we check receipts upstream? maybe during block creation?
	// Iterate through receipts and if there is a nil receipt.Logs, update it to an empty list
	for _, receipt := range corpusBlock.blockReceipts {
		if receipt.Logs == nil {
			receipt.Logs = []*coreTypes.Log{}
		}
	}

	return corpusBlock
}

// Header returns the SimpleCorpusBlockHeader of the SimpleCorpusBlock
func (m *SimpleCorpusBlock) Header() corpusTypes.CorpusBlockHeader {
	corpusBlockHeader := corpusTypes.CorpusBlockHeader(m.blockHeader)
	return corpusBlockHeader
}

// Transactions returns the transactions of the SimpleCorpusBlock
func (m *SimpleCorpusBlock) Transactions() []core.Message {
	var messages []core.Message
	for _, callMessage := range m.blockTransactions {
		message := core.Message(callMessage)
		messages = append(messages, message)
	}
	return messages
}

// Receipts returns the receipts of the SimpleCorpusBlock
func (m *SimpleCorpusBlock) Receipts() []*coreTypes.Receipt {
	return m.blockReceipts
}

// MarshalJSON marshals the SimpleCorpusBlock object into JSON
func (m SimpleCorpusBlock) MarshalJSON() ([]byte, error) {
	type SimpleCorpusBlock struct {
		BlockHeader       *SimpleCorpusBlockHeader  `json:"header"`
		BlockTransactions []*chainTypes.CallMessage `json:"transactions"`
		BlockReceipts     []*coreTypes.Receipt      `json:"receipts"`
	}
	var enc SimpleCorpusBlock
	enc.BlockHeader = m.blockHeader
	enc.BlockTransactions = m.blockTransactions
	enc.BlockReceipts = m.blockReceipts
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals a byte string into a SimpleCorpusBlock object.
func (m *SimpleCorpusBlock) UnmarshalJSON(input []byte) error {
	type SimpleCorpusBlock struct {
		BlockHeader       *SimpleCorpusBlockHeader  `json:"header"`
		BlockTransactions []*chainTypes.CallMessage `json:"transactions"`
		BlockReceipts     []*coreTypes.Receipt      `json:"receipts"`
	}
	var dec SimpleCorpusBlock
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.BlockHeader != nil {
		m.blockHeader = dec.BlockHeader
	}
	if dec.BlockTransactions != nil {
		m.blockTransactions = dec.BlockTransactions
	}
	if dec.BlockReceipts != nil {
		m.blockReceipts = dec.BlockReceipts
	}
	return nil
}
