package test_utils

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/fuzzing/corpus/simple_corpus"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus/types"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"testing"
)

// TODO is there a interface that can be extended for comparisons of CorpusBlockSequences?
// SequencesAreEqual tests whether two types.CorpusBlockSequence objects are equal to each other
func SequencesAreEqual(t *testing.T, seqOne *simple_corpus.SimpleCorpusEntry, seqTwo *simple_corpus.SimpleCorpusEntry) bool {
	// Ensure the lengths of both sequences are the same
	assert.True(t, len(seqOne.Blocks()) == len(seqTwo.Blocks()), "Different number of blocks in sequences")
	// Iterate through seqOne
	for idx, corpusBlock := range seqOne.Blocks() {
		// Make sure the headers are equal
		HeadersAreEqual(t, corpusBlock.(*simple_corpus.SimpleCorpusBlock).Header(), seqTwo.Blocks()[idx].(*simple_corpus.SimpleCorpusBlock).Header())
		// Make sure transactions are equal
		TransactionsAreEqual(t, corpusBlock.(*simple_corpus.SimpleCorpusBlock).Transactions(), seqTwo.Blocks()[idx].(*simple_corpus.SimpleCorpusBlock).Transactions())
	}
	return true
}

// HeadersAreEqual tests whether the headers of the two types.CorpusBlockSequence objects are equal to each other
func HeadersAreEqual(t *testing.T, headerOne corpusTypes.CorpusBlockHeader, headerTwo corpusTypes.CorpusBlockHeader) {
	// Make sure that the block number, block hash, and block timestamp are the same
	assert.True(t, headerOne.(*simple_corpus.SimpleCorpusBlockHeader).BlockNumber().Cmp(headerTwo.(*simple_corpus.SimpleCorpusBlockHeader).BlockNumber()) == 0, "Block numbers are not equal")
	assert.True(t, headerOne.(*simple_corpus.SimpleCorpusBlockHeader).BlockHash() == headerTwo.(*simple_corpus.SimpleCorpusBlockHeader).BlockHash(), "Block hashes are not equal")
	assert.True(t, headerOne.(*simple_corpus.SimpleCorpusBlockHeader).BlockTimestamp() == headerTwo.(*simple_corpus.SimpleCorpusBlockHeader).BlockTimestamp(), "Block timestamps are not equal")
}

// TransactionsAreEqual tests whether each transaction in both types.CorpusBlockSequence objects are equal
func TransactionsAreEqual(t *testing.T, txSeqOne []core.Message, txSeqTwo []core.Message) {
	// Ensure the lengths of both transaction sequences are the same
	assert.True(t, len(txSeqOne) == len(txSeqTwo), "Different number of transactions in blocks")
	// Iterate across each transaction
	for idx, txOneInterface := range txSeqOne {
		// De-reference
		txOne := txOneInterface.(*fuzzerTypes.CallMessage)
		txTwo := txSeqTwo[idx].(*fuzzerTypes.CallMessage)
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
