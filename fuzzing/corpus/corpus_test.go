package corpus

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/fuzzing/calls"
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
		err := corpus.AddCallSequence(getMockCallSequence(minBlocks+(rand.Int()%(maxBlocks-minBlocks))), nil)
		if err != nil {
			return nil, err
		}
	}
	return corpus, nil
}

// getMockSimpleCorpusEntry creates a mock CorpusCallSequence with numBlocks blocks for testing
func getMockCallSequence(size int) calls.CallSequence {
	cs := make(calls.CallSequence, size)
	for i := 0; i < size; i++ {
		cs[i] = getMockCallSequenceElement()
	}
	return cs
}

// getMockSimpleBlockBlock creates a mock CorpusBlock with numTransactions transactions and receipts for testing
func getMockCallSequenceElement() *calls.CallSequenceElement {
	return &calls.CallSequenceElement{
		Contract:            nil,
		Call:                getMockCallSequenceElementCall(),
		BlockNumberDelay:    rand.Uint64(),
		BlockTimestampDelay: rand.Uint64(),
		ChainReference:      nil,
	}
}

// getMockCallSequenceElementCall creates a mock CallMessage for testing
func getMockCallSequenceElementCall() *calls.CallMessage {
	to := common.BigToAddress(big.NewInt(rand.Int63()))
	txn := calls.CallMessage{
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

// testCorpusCallSequencesAreEqual tests whether two CorpusCallSequence objects are equal to each other
func testCorpusCallSequencesEqual(t *testing.T, expected calls.CallSequence, actual calls.CallSequence) {
	// Ensure the lengths of both sequences are the same
	assert.EqualValues(t, len(expected), len(actual), "Different number of calls in sequences")

	// Iterate through seqOne
	for i := 0; i < len(expected); i++ {
		testCorpusCallSequenceElementsEqual(t, *expected[i], *actual[i])
	}
}

// testCorpusBlockHeadersAreEqual tests whether two CorpusBlockHeader objects are equal to each other
func testCorpusCallSequenceElementsEqual(t *testing.T, expected calls.CallSequenceElement, actual calls.CallSequenceElement) {
	// Make sure the call is equal
	assert.EqualValues(t, *expected.Call, *actual.Call)

	// Make sure delays are equal
	assert.EqualValues(t, expected.BlockNumberDelay, actual.BlockNumberDelay)
	assert.EqualValues(t, expected.BlockTimestampDelay, actual.BlockTimestampDelay)
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
			var sameEntry calls.CallSequence
			err = json.Unmarshal(b, &sameEntry)
			assert.NoError(t, err)

			// Check equality
			testCorpusCallSequencesEqual(t, entryFile.data, sameEntry)
		}
	})
}
