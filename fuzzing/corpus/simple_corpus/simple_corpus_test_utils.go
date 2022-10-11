package simple_corpus

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus"
	"math/big"
	"math/rand"
	"testing"
)

// getMockSimpleCorpus creates a mock corpus with numEntries entries for testing
func getMockSimpleCorpus(numEntries int) *SimpleCorpus {
	corpus := NewSimpleCorpus()
	for i := 0; i < numEntries; i++ {
		corpus.corpusEntries = append(corpus.corpusEntries, getMockSimpleCorpusEntry(numEntries))
	}
	return corpus
}

// getMockSimpleCorpusEntry creates a mock SimpleCorpusEntry with numBlocks blocks for testing
func getMockSimpleCorpusEntry(numBlocks int) *SimpleCorpusEntry {
	entry := NewSimpleCorpusEntry()
	for i := 0; i < numBlocks; i++ {
		entry.corpusBlocks = append(entry.corpusBlocks, getMockSimpleBlockBlock(numBlocks))
	}
	return entry
}

// getMockSimpleBlockBlock creates a mock SimpleCorpusBlock with numTransactions transactions and receipts for testing
func getMockSimpleBlockBlock(numTransactions int) *SimpleCorpusBlock {
	block := NewSimpleCorpusBlock()
	block.blockHeader = getMockSimpleCorpusBlockHeader()
	for i := 0; i < numTransactions; i++ {
		block.blockTransactions = append(block.blockTransactions, getMockTransaction())
		block.blockReceipts = append(block.blockReceipts, getMockReceipt())
	}
	return block
}

// getMockSimpleCorpusBlockHeader creates a mock SimpleCorpusBlockHeader for testing
func getMockSimpleCorpusBlockHeader() *SimpleCorpusBlockHeader {
	return NewSimpleCorpusBlockHeader(common.HexToHash("BlockHash"), rand.Uint64(), big.NewInt(int64(rand.Int())))
}

// getMockTransaction creates a mock CallMessage for testing
func getMockTransaction() *chainTypes.CallMessage {
	to := common.HexToAddress("ToAddress")
	txn := chainTypes.CallMessage{
		MsgFrom:      common.HexToAddress("FromAddress"),
		MsgTo:        &to,
		MsgNonce:     rand.Uint64(),
		MsgValue:     big.NewInt(int64(rand.Int())),
		MsgGas:       rand.Uint64(),
		MsgGasPrice:  big.NewInt(int64(rand.Int())),
		MsgGasFeeCap: big.NewInt(int64(rand.Int())),
		MsgGasTipCap: big.NewInt(int64(rand.Int())),
		MsgData:      []byte{uint8(rand.Uint64()), uint8(rand.Uint64()), uint8(rand.Uint64()), uint8(rand.Uint64())},
	}
	return &txn
}

// getMockReceipt creates a mock coreTypes.Receipt for testing
func getMockReceipt() *coreTypes.Receipt {
	receipt := coreTypes.Receipt{
		Type:              0,
		PostState:         []byte{uint8(rand.Uint64()), uint8(rand.Uint64()), uint8(rand.Uint64()), uint8(rand.Uint64())},
		Status:            rand.Uint64(),
		CumulativeGasUsed: rand.Uint64(),
		Bloom:             coreTypes.Bloom{},
		Logs:              []*coreTypes.Log{},
		TxHash:            common.Hash{},
		ContractAddress:   common.HexToAddress("ContractAddress"),
		GasUsed:           rand.Uint64(),
		BlockHash:         common.HexToHash("BlockHash"),
		BlockNumber:       big.NewInt(int64(rand.Int())),
		TransactionIndex:  uint(rand.Uint64()),
	}
	return &receipt
}

// The functions below were used to test whether two corpus entries are equal to each other. However, realized that it is
// way easier to just hash the entries and test like this. Keeping these functions in case they are useful for some other
// case.

// EntriesAreEqual tests whether two SimpleCorpusEntry objects are equal to each other
func EntriesAreEqual(t *testing.T, seqOne *SimpleCorpusEntry, seqTwo *SimpleCorpusEntry) {
	// Ensure the lengths of both sequences are the same
	assert.True(t, len(seqOne.Blocks()) == len(seqTwo.Blocks()), "Different number of blocks in sequences")
	// Iterate through seqOne
	for idx, corpusBlock := range seqOne.Blocks() {
		// Make sure the headers are equal
		testHeadersAreEqual(t, corpusBlock.(*SimpleCorpusBlock).Header(), seqTwo.Blocks()[idx].(*SimpleCorpusBlock).Header())
		// Make sure transactions are equal
		testTransactionsAreEqual(t, corpusBlock.(*SimpleCorpusBlock).Transactions(), seqTwo.Blocks()[idx].(*SimpleCorpusBlock).Transactions())
	}
}

// testHeadersAreEqual tests whether two SimpleCorpusBlockHeader objects are equal to each other
func testHeadersAreEqual(t *testing.T, headerOne corpusTypes.CorpusBlockHeader, headerTwo corpusTypes.CorpusBlockHeader) {
	// Make sure that the block number, block hash, and block timestamp are the same
	assert.True(t, headerOne.(*SimpleCorpusBlockHeader).BlockNumber().Cmp(headerTwo.(*SimpleCorpusBlockHeader).BlockNumber()) == 0, "Block numbers are not equal")
	assert.True(t, headerOne.(*SimpleCorpusBlockHeader).BlockHash() == headerTwo.(*SimpleCorpusBlockHeader).BlockHash(), "Block hashes are not equal")
	assert.True(t, headerOne.(*SimpleCorpusBlockHeader).BlockTimestamp() == headerTwo.(*SimpleCorpusBlockHeader).BlockTimestamp(), "Block timestamps are not equal")
}

// testTransactionsAreEqual tests whether each transaction in two SimpleCorpusBlock objects are equal
func testTransactionsAreEqual(t *testing.T, txSeqOne []core.Message, txSeqTwo []core.Message) {
	// Ensure the lengths of both transaction sequences are the same
	assert.True(t, len(txSeqOne) == len(txSeqTwo), "Different number of transactions in blocks")
	// Iterate across each transaction
	for idx, txOneInterface := range txSeqOne {
		// De-reference
		txOne := txOneInterface.(*chainTypes.CallMessage)
		txTwo := txSeqTwo[idx].(*chainTypes.CallMessage)
		// Check all fields of a types.CallMessage
		assert.True(t, txOne.MsgGasPrice.Cmp(txTwo.MsgGasPrice) == 0, "MsgGasPrices are not equal")
		assert.True(t, txOne.MsgGasTipCap.Cmp(txTwo.MsgGasTipCap) == 0, "MsgGasTips are not equal")
		assert.True(t, txOne.MsgGasFeeCap.Cmp(txTwo.MsgGasFeeCap) == 0, "MsgGasFeeCap are not equal")
		assert.True(t, txOne.MsgNonce == txTwo.MsgNonce, "Nonces are not equal")
		assert.True(t, string(txOne.MsgData) == string(txTwo.MsgData), "Data are not equal")
		assert.True(t, txOne.MsgGas == txTwo.MsgGas, "Gas amounts are not equal")
		assert.True(t, txOne.MsgValue.Cmp(txTwo.MsgValue) == 0, "Values are not equal")
		assert.True(t, txOne.MsgTo.String() == txTwo.MsgTo.String(), "TO addresses are not equal")
		assert.True(t, txOne.MsgFrom.String() == txTwo.MsgFrom.String(), "FROM addresses are not equal")
	}
}
