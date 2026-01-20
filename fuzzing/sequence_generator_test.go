package fuzzing

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/stretchr/testify/assert"
)

// getMockCallSequence creates a mock CallSequence with the given size and data value for testing.
func getMockCallSequence(size, data int) calls.CallSequence {
	cs := make(calls.CallSequence, size)
	for i := 0; i < size; i++ {
		cs[i] = getMockCallSequenceElement(data)
	}
	return cs
}

// getMockCallSequenceElement creates a mock CallSequenceElement for testing.
func getMockCallSequenceElement(data int) *calls.CallSequenceElement {
	return &calls.CallSequenceElement{
		Contract:            nil,
		Call:                getMockCallSequenceElementCall(data),
		BlockNumberDelay:    rand.Uint64(),
		BlockTimestampDelay: rand.Uint64(),
		ChainReference:      nil,
	}
}

// getMockCallSequenceElementCall creates a mock CallMessage for testing.
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

func TestSequenceGenerationStrategies(t *testing.T) {
	strategies := map[string]func(rand *rand.Rand, sequenceGenerator func() (calls.CallSequence, error), sequence calls.CallSequence) error{
		"interleave": interleaveCorpus,
		"splice":     spliceCorpus,
		"prepend":    prependFromCorpus,
		"append":     appendFromCorpus,
	}

	for name, strategyFn := range strategies {
		t.Run(name, func(t *testing.T) {
			sourceSequence := []calls.CallSequence{getMockCallSequence(4, 1), getMockCallSequence(4, 5)}
			i := 0
			sequence := getMockCallSequence(4, 7)
			sequence = append(sequence, getMockCallSequence(4, 9)...)
			mockSequenceGenerator := func() (calls.CallSequence, error) {
				s := sourceSequence[i]
				i++
				return s, nil
			}

			err := strategyFn(rand.New(rand.NewSource(0)), mockSequenceGenerator, sequence)

			assert.NoError(t, err)
			assert.NotNil(t, sequence)
		})
	}
}

func TestSpliceAtRandomEmptySlices(t *testing.T) {
	provider := rand.New(rand.NewSource(0))

	// Test with empty first slice
	result := spliceAtRandom(provider, []*int{}, []*int{new(int)})
	assert.Len(t, result, 1)

	// Test with empty second slice
	val := 42
	result = spliceAtRandom(provider, []*int{&val}, []*int{})
	assert.Len(t, result, 1)
	assert.Equal(t, 42, *result[0])

	// Test with both empty
	result = spliceAtRandom(provider, []*int{}, []*int{})
	assert.Len(t, result, 0)
}

func TestInterleaveAtRandomEmptySlices(t *testing.T) {
	provider := rand.New(rand.NewSource(0))

	// Test with empty first slice
	result := interleaveAtRandom(provider, []*int{}, []*int{new(int)})
	assert.Len(t, result, 1)

	// Test with empty second slice
	val := 42
	result = interleaveAtRandom(provider, []*int{&val}, []*int{})
	assert.Len(t, result, 1)
	assert.Equal(t, 42, *result[0])

	// Test with both empty
	result = interleaveAtRandom(provider, []*int{}, []*int{})
	assert.Len(t, result, 0)
}
