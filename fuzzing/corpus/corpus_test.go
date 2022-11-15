package corpus

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	"github.com/trailofbits/medusa/utils/testutils"
	"math/big"
	"math/rand"
	"path/filepath"
	"testing"
)

// getMockSimpleCorpus creates a mock corpus with numEntries callSequencesByFilePath for testing
func getMockSimpleCorpus(minSequences int, maxSequences, minBlocks int, maxBlocks int) (*Corpus, error) {
	// Create a new corpus
	corpus, err := NewCorpus("corpus")
	if err != nil {
		return nil, err
	}

	// Add the requested number of entries.
	numSequences := minSequences + (rand.Int() % (maxSequences - minSequences))
	for i := 0; i < numSequences; i++ {
		err := corpus.AddCallSequence(*getMockSimpleCorpusEntry(minBlocks + (rand.Int() % (maxBlocks - minBlocks))))
		if err != nil {
			return nil, err
		}
	}
	return corpus, nil
}

// getMockSimpleCorpusEntry creates a mock CorpusCallSequence with numBlocks blocks for testing
func getMockSimpleCorpusEntry(numBlocks int) *CorpusCallSequence {
	entry := &CorpusCallSequence{
		Blocks: nil,
	}
	for i := 0; i < numBlocks; i++ {
		entry.Blocks = append(entry.Blocks, *getMockSimpleBlockBlock(numBlocks))
	}
	return entry
}

// getMockSimpleBlockBlock creates a mock CorpusBlock with numTransactions transactions and receipts for testing
func getMockSimpleBlockBlock(numTransactions int) *CorpusBlock {
	block := &CorpusBlock{
		Header: CorpusBlockHeader{
			Number:    big.NewInt(int64(rand.Int())),
			Timestamp: rand.Uint64(),
		},
		Transactions: nil,
	}
	for i := 0; i < numTransactions; i++ {
		block.Transactions = append(block.Transactions, *getMockTransaction())
	}
	return block
}

// getMockTransaction creates a mock CallMessage for testing
func getMockTransaction() *chainTypes.CallMessage {
	to := common.BigToAddress(big.NewInt(rand.Int63()))
	txn := chainTypes.CallMessage{
		MsgFrom:      common.BigToAddress(big.NewInt(rand.Int63())),
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

// testCorpusBlockSequencesAreEqual tests whether two CorpusCallSequence objects are equal to each other
func testCorpusBlockSequencesAreEqual(t *testing.T, seqOne *CorpusCallSequence, seqTwo *CorpusCallSequence) {
	// Ensure the lengths of both sequences are the same
	assert.EqualValues(t, len(seqOne.Blocks), len(seqTwo.Blocks), "Different number of blocks in sequences")

	// Iterate through seqOne
	for idx, corpusBlock := range seqOne.Blocks {
		// Make sure the headers are equal
		testCorpusBlockHeadersAreEqual(t, corpusBlock.Header, seqTwo.Blocks[idx].Header)
		// Make sure transactions are equal
		testCorpusBlockTransactionsAreEqual(t, corpusBlock.Transactions, seqTwo.Blocks[idx].Transactions)
	}
}

// testCorpusBlockHeadersAreEqual tests whether two CorpusBlockHeader objects are equal to each other
func testCorpusBlockHeadersAreEqual(t *testing.T, headerOne CorpusBlockHeader, headerTwo CorpusBlockHeader) {
	// Make sure that the block number, and block timestamp are the same
	assert.True(t, headerOne.Number.Cmp(headerTwo.Number) == 0, "Block numbers are not equal")
	assert.EqualValues(t, headerOne.Timestamp, headerTwo.Timestamp, "Block timestamps are not equal")
}

// testCorpusBlockTransactionsAreEqual tests whether each transaction in two CorpusBlock objects are equal
func testCorpusBlockTransactionsAreEqual(t *testing.T, txSeqOne []chainTypes.CallMessage, txSeqTwo []chainTypes.CallMessage) {
	// Ensure the lengths of both transaction sequences are the same
	assert.True(t, len(txSeqOne) == len(txSeqTwo), "Different number of transactions in blocks")
	// Iterate across each transaction
	for idx, txOne := range txSeqOne {
		// De-reference our supposed parallel equivalent transaction.
		txTwo := txSeqTwo[idx]

		// Check all fields of a types.CallMessage
		assert.True(t, txOne.MsgGasPrice.Cmp(txTwo.MsgGasPrice) == 0, "Gas prices field is not equal")
		assert.True(t, txOne.MsgGasTipCap.Cmp(txTwo.MsgGasTipCap) == 0, "GasTipCaps field is not equal")
		assert.True(t, txOne.MsgGasFeeCap.Cmp(txTwo.MsgGasFeeCap) == 0, "GasFeeCaps field is not equal")
		assert.EqualValues(t, txOne.MsgNonce, txTwo.MsgNonce, "Nonces field is not equal")
		assert.EqualValues(t, txOne.MsgData, txTwo.MsgData, "Data field is not equal")
		assert.EqualValues(t, txOne.MsgGas, txTwo.MsgGas, "Gas field is not equal")
		assert.True(t, txOne.MsgValue.Cmp(txTwo.MsgValue) == 0, "Value field is not equal")
		assert.EqualValues(t, txOne.MsgTo.String(), txTwo.MsgTo.String(), "To field is not equal")
		assert.EqualValues(t, txOne.MsgFrom.String(), txTwo.MsgFrom.String(), "From field is not equal")
	}
}

// TestCorpusReadWrite first writes the corpus to disk and then reads it back from the disk and ensures integrity.
func TestCorpusReadWrite(t *testing.T) {
	// Create a mock corpus
	corpus, err := getMockSimpleCorpus(10, 20, 1, 7)
	assert.NoError(t, err)
	testutils.ExecuteInDirectory(t, t.TempDir(), func() {
		// Write to disk
		err := corpus.Flush()
		assert.NoError(t, err)

		// Ensure that there are the correct number of call sequence files
		matches, err := filepath.Glob(filepath.Join(corpus.CallSequencesDirectory(), "*.json"))
		assert.NoError(t, err)
		assert.EqualValues(t, corpus.CallSequenceCount(), len(matches), "Did not find numEntries matches")

		// Wipe corpus clean so that you can now read it in from disk
		corpus, err = NewCorpus("corpus")
		assert.NoError(t, err)

		// Create a new corpus object and read our previously read artifacts.
		corpus, err = NewCorpus(corpus.storageDirectory)
		assert.NoError(t, err)
	})
}

// TestCorpusCallSequenceMarshaling ensures that a corpus entry that is round trip serialized retains its original
// values.
func TestCorpusCallSequenceMarshaling(t *testing.T) {
	// Create a mock corpus
	corpus, err := getMockSimpleCorpus(10, 20, 1, 7)
	assert.NoError(t, err)

	// Run the test in our temporary test directory to avoid artifact pollution.
	testutils.ExecuteInDirectory(t, t.TempDir(), func() {
		// For each entry, marshal it and then unmarshal the byte array
		for _, entryFile := range corpus.callSequences {
			// Marshal the entry
			b, err := json.Marshal(entryFile.data)
			assert.NoError(t, err)

			// Unmarshal byte array
			var sameEntry CorpusCallSequence
			err = json.Unmarshal(b, &sameEntry)
			assert.NoError(t, err)

			// Check equality
			testCorpusBlockSequencesAreEqual(t, entryFile.data, &sameEntry)
		}
	})
}
