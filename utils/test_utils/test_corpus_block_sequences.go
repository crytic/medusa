package test_utils

import (
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/fuzzing/types"
	"testing"
)

// TODO is there a interface that can be extended for comparisons of CorpusBlockSequences?
// SequencesAreEqual tests whether two types.CorpusBlockSequence objects are equal to each other
func SequencesAreEqual(t *testing.T, seqOne types.CorpusBlockSequence, seqTwo types.CorpusBlockSequence) bool {
	// Ensure the lengths of both sequences are the same
	assert.True(t, len(seqOne) == len(seqTwo), "Different number of blocks in sequences")
	// Iterate through seqOne
	for idx, corpusBlock := range seqOne {
		// Make sure the headers are equal
		HeadersAreEqual(t, corpusBlock.Header, seqTwo[idx].Header)
		// Make sure transactions are equal
		TransactionsAreEqual(t, corpusBlock.Transactions, corpusBlock.Transactions)
	}
	return true
}

// HeadersAreEqual tests whether the headers of the two types.CorpusBlockSequence objects are equal to each other
func HeadersAreEqual(t *testing.T, headerOne types.CorpusBlockHeader, headerTwo types.CorpusBlockHeader) {
	// Make sure that the block number, block hash, and block timestamp are the same
	assert.True(t, headerOne.BlockNumber.Cmp(headerTwo.BlockNumber) == 0, "Block numbers are not equal")
	assert.True(t, headerOne.BlockHash == headerTwo.BlockHash, "Block hashes are not equal")
	assert.True(t, headerOne.BlockTimestamp == headerTwo.BlockTimestamp, "Block timestamps are not equal")
}

// TransactionsAreEqual tests whether each transaction in both types.CorpusBlockSequence objects are equal
func TransactionsAreEqual(t *testing.T, txSeqOne []*types.CallMessage, txSeqTwo []*types.CallMessage) {
	// Ensure the lengths of both transaction sequences are the same
	assert.True(t, len(txSeqOne) == len(txSeqTwo), "Different number of transactions in blocks")
	// Iterate across each transaction
	for idx, txPointer := range txSeqOne {
		// De-reference
		txOne := *txPointer
		txTwo := *txSeqTwo[idx]
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
