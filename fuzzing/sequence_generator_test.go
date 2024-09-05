package fuzzing

import (
	"fmt"
	"math/big"
	"math/rand"

	"testing"

	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// getMockSimpleCorpusEntry creates a mock CorpusCallSequence with numBlocks blocks for testing
func getMockCallSequence(size, data int) calls.CallSequence {
	cs := make(calls.CallSequence, size)
	for i := 0; i < size; i++ {
		cs[i] = getMockCallSequenceElement(data)
	}
	return cs
}

// getMockSimpleBlockBlock creates a mock CorpusBlock with numTransactions transactions and receipts for testing
func getMockCallSequenceElement(data int) *calls.CallSequenceElement {

	return &calls.CallSequenceElement{
		Contract:            nil,
		Call:                getMockCallSequenceElementCall(data),
		BlockNumberDelay:    rand.Uint64(),
		BlockTimestampDelay: rand.Uint64(),
		ChainReference:      nil,
	}
}

// getMockCallSequenceElementCall creates a mock CallMessage for testing
func getMockCallSequenceElementCall(data int) *calls.CallMessage {
	to := common.BigToAddress(big.NewInt(rand.Int63()))
	txn := calls.CallMessage{
		From:      common.BigToAddress(big.NewInt(rand.Int63())),
		To:        &to,
		Nonce:     rand.Uint64(),
		Value:     big.NewInt(int64(rand.Int())),
		GasLimit:  rand.Uint64(),
		GasPrice:  big.NewInt(int64(rand.Int())),
		GasFeeCap: big.NewInt(int64(rand.Int())),
		GasTipCap: big.NewInt(int64(rand.Int())),
		Data:      []byte{uint8(data), uint8(data), uint8(data), uint8(data)},
	}
	return &txn
}

func TestSplice(t *testing.T) {
	strategies := map[string]func(rand *rand.Rand, sequenceGenerator func() (calls.CallSequence, error), sequence calls.CallSequence) error{
		// "interleave": callSeqGenFuncInterleaveAtRandom,
		// "splice":     callSeqGenFuncSpliceAtRandom,
		// "expansion":  callSeqGenFuncExpansion,
		"prepend": callSeqGenFuncCorpusHead,
		"apppend": callSeqGenFuncCorpusTail,
		// "delete":     callSeqDeleteRandomElement,
		// "swap": callSeqSwapRandomElement,
	}
	// Seed the PRNG to make the randomness deterministic

	// Prepare a destination sequence (with the expected size)
	// expectedSize := 8 // Adjust based on the expected interleave size in your scenario

	// Prepare a source sequence (with the expected size)

	for name, strategyFn := range strategies {
		// Call the function under test
		sourceSequence := []calls.CallSequence{getMockCallSequence(1, 1), getMockCallSequence(1, 5)}
		i := 0
		sequence := getMockCallSequence(4, 7)
		sequence = append(sequence, getMockCallSequence(4, 9)...)
		mockSequenceGenerator := func() (calls.CallSequence, error) {

			// Return the source sequence
			s := sourceSequence[i]
			i++
			return s, nil
		}
		fmt.Println("Before:")
		for _, s := range sequence {
			if s == nil {
				fmt.Println("nil")
				continue
			}
			fmt.Println(s.Call.Data)
		}
		// fmt.Println("Source sequence 1:")
		// fmt.Println(sourceSequence[0][0].Call.Data)
		// fmt.Println(sourceSequence[0][1].Call.Data)
		// fmt.Println(sourceSequence[0][2].Call.Data)
		// fmt.Println(sourceSequence[0][3].Call.Data)
		// fmt.Println("Source sequence 2:")
		// fmt.Println(sourceSequence[1][0].Call.Data)
		// fmt.Println(sourceSequence[1][1].Call.Data)
		// fmt.Println(sourceSequence[1][2].Call.Data)
		// fmt.Println(sourceSequence[1][3].Call.Data)

		err := strategyFn(rand.New(rand.NewSource(0)), mockSequenceGenerator, sequence)

		// Ensure no error
		assert.NoError(t, err)

		// Check if the interleaved sequence has the correct size and contents
		assert.NotNil(t, sequence)
		// assert.Len(t, sequence, expectedSize)

		// Check that the elements in the sequence are correctly interleaved
		// You can further check for specific fields if necessary
		fmt.Printf("Interleaved sequence using %s:\n", name)
		for _, s := range sequence {
			if s == nil {
				fmt.Println("nil")
				continue
			}
			fmt.Println(s.Call.Data)
		}
	}
}
