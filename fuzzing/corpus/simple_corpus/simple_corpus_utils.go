package simple_corpus

import (
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus/types"
	"github.com/trailofbits/medusa/fuzzing/testnode"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
)

// testNodeBlockToCorpusBlock converts a coreTypes.TestNodeBlock to a fuzzerTypes.MedusaCorpusBlock. Components such as the
// block timestamp, block number, block hash, transaction receipts, and the transactions are used during the
// conversion
func testNodeBlockToCorpusBlock(testNodeBlock *testnode.TestNodeBlock, tx *fuzzerTypes.CallMessage) corpusTypes.CorpusBlock {
	// Create corpusBlock object
	simpleBlock := NewSimpleCorpusBlock()
	// Set header fields
	simpleBlock.blockHeader = NewSimpleCorpusBlockHeader(
		testNodeBlock.BlockHash,
		testNodeBlock.Header.Time,
		testNodeBlock.Header.Number)
	// TODO: This will change when more than one receipt goes in a block
	simpleBlock.blockReceipts = append(simpleBlock.blockReceipts, testNodeBlock.Receipt)
	// TODO: Can we check receipts upstream? maybe during block creation?
	simpleBlock.blockReceipts = checkReceipts(simpleBlock.blockReceipts)
	// TODO: This will change when more than one transaction goes in a block
	simpleBlock.blockTransactions = append(simpleBlock.blockTransactions, tx)
	block := corpusTypes.CorpusBlock(simpleBlock)
	return block
}

// checkReceipts ensures that each receipt has a Log object that is not nil. This is performed so that unmarshaling
// works as expected
func checkReceipts(receipts []*coreTypes.Receipt) []*coreTypes.Receipt {
	// Iterate through receipts and if there is a nil receipt.Logs, update it to an empty list
	for _, receipt := range receipts {
		if receipt.Logs == nil {
			receipt.Logs = []*coreTypes.Log{}
		}
	}
	return receipts
}
