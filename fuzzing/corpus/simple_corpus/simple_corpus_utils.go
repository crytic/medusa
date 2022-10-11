package simple_corpus

import (
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus"
)

// testNodeBlockToCorpusBlock converts a testnode.TestNodeBlock to a SimpleCorpusBlock. Components such as the
// block timestamp, block number, block hash, transaction receipts, and the transactions are maintained during conversion.
func testNodeBlockToCorpusBlock(testNodeBlock *chainTypes.Block) corpusTypes.CorpusBlock {
	// Create corpusBlock object
	simpleBlock := NewSimpleCorpusBlock()
	// Set header fields
	simpleBlock.blockHeader = NewSimpleCorpusBlockHeader(
		testNodeBlock.Hash(),
		testNodeBlock.Header().Time,
		testNodeBlock.Header().Number)

	// Update our transactions and receipts
	simpleBlock.blockTransactions = testNodeBlock.Messages()
	simpleBlock.blockReceipts = testNodeBlock.Receipts()

	// TODO: Can we check receipts upstream? maybe during block creation?
	checkAndUpdateReceipts(simpleBlock.blockReceipts)

	return corpusTypes.CorpusBlock(simpleBlock)
}

// checkAndUpdateReceipts ensures that each receipt has a Log object that is not nil. This is performed so that unmarshaling
// works as expected when the corpus is read from disk. So you do this while marshaling so that unmarshaling works as expected
// TODO: Is there a way to avoid this? It seems like receipts unmarshaling requires that Logs is not nil.
func checkAndUpdateReceipts(receipts []*coreTypes.Receipt) {
	// Iterate through receipts and if there is a nil receipt.Logs, update it to an empty list
	for _, receipt := range receipts {
		if receipt.Logs == nil {
			receipt.Logs = []*coreTypes.Log{}
		}
	}
}
