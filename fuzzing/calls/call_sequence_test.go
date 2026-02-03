package calls

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa/chain"
	"github.com/stretchr/testify/assert"
)

// createTestChain creates a TestChain with funded accounts for testing purposes.
func createTestChain(t *testing.T) (*chain.TestChain, []common.Address) {
	// Create funded accounts
	senders := []common.Address{
		common.HexToAddress("0x1000"),
		common.HexToAddress("0x2000"),
	}

	genesisAlloc := make(types.GenesisAlloc)
	initBalance := new(big.Int).Div(abi.MaxInt256, big.NewInt(2))
	for _, sender := range senders {
		genesisAlloc[sender] = types.Account{
			Balance: initBalance,
		}
	}

	// Create test chain
	testChain, err := chain.NewTestChain(context.Background(), genesisAlloc, nil)
	assert.NoError(t, err)

	return testChain, senders
}

// TestCallSequenceBlockTimestampDisplay verifies that block number and timestamp
// are correctly captured and displayed for each call in a sequence, even when
// the underlying block header is modified during execution.
func TestCallSequenceBlockTimestampDisplay(t *testing.T) {
	// Create test chain with funded accounts
	testChain, senders := createTestChain(t)
	sender := senders[0]
	recipient := senders[1]

	// Define block delays for each call in the sequence
	// Call 0: no delay (should use current block)
	// Call 1: delay by 5 blocks and 5 seconds
	// Call 2: delay by 10 blocks and 10 seconds
	delays := []struct {
		blockNumberDelay    uint64
		blockTimestampDelay uint64
	}{
		{0, 0},
		{5, 5},
		{10, 10},
	}

	// Create call sequence with varying delays
	var callSequence CallSequence
	for i := 0; i < len(delays); i++ {
		// Create a simple value transfer call
		call := NewCallMessage(
			sender,
			&recipient,
			uint64(i),
			big.NewInt(1),
			100000,
			big.NewInt(1),
			big.NewInt(0),
			big.NewInt(0),
			[]byte{}, // empty data for simple transfer
		)

		// Create call sequence element with delays
		element := NewCallSequenceElement(
			nil, // no contract for simple transfer
			call,
			delays[i].blockNumberDelay,
			delays[i].blockTimestampDelay,
		)

		callSequence = append(callSequence, element)
	}

	// Execute the call sequence
	executedSequence, err := ExecuteCallSequence(testChain, callSequence)
	assert.NoError(t, err)
	assert.Equal(t, len(delays), len(executedSequence), "all calls should have been executed")

	// Verify each element has correct block number and timestamp captured
	var expectedBlockNumber uint64 = 1 // Genesis is block 0, first pending block is 1
	var expectedTimestamp uint64 = 1   // Genesis timestamp is 0, first block is at least 1

	for i, element := range executedSequence {
		// Verify ChainReference was set
		assert.NotNil(t, element.ChainReference, "ChainReference should be set for call %d", i)

		// For first call (i=0), we use the initial block
		// For subsequent calls, we apply the delays
		if i > 0 {
			// Apply the delay from the current call
			// The delay is applied relative to the previous call's block
			expectedBlockNumber += delays[i].blockNumberDelay
			expectedTimestamp += delays[i].blockTimestampDelay
		}

		// Verify the snapshots match expected values
		assert.Equal(t, expectedBlockNumber, element.ChainReference.BlockNumber,
			"Call %d: BlockNumber snapshot mismatch", i)
		assert.Equal(t, expectedTimestamp, element.ChainReference.BlockTimestamp,
			"Call %d: BlockTimestamp snapshot mismatch", i)

		// Verify String() output contains the correct values
		strOutput := element.String()
		assert.Contains(t, strOutput, "block="+big.NewInt(int64(expectedBlockNumber)).String(),
			"Call %d: String() should contain correct block number", i)
		assert.Contains(t, strOutput, "time="+big.NewInt(int64(expectedTimestamp)).String(),
			"Call %d: String() should contain correct timestamp", i)
	}

	// Verify that each call shows DISTINCT block numbers and timestamps
	blockNumbers := make(map[uint64]bool)
	timestamps := make(map[uint64]bool)
	for i, element := range executedSequence {
		blockNum := element.ChainReference.BlockNumber
		timestamp := element.ChainReference.BlockTimestamp

		// For calls with non-zero delays, verify uniqueness
		if i > 0 {
			assert.False(t, blockNumbers[blockNum],
				"Call %d: Block number %d should be unique", i, blockNum)
			assert.False(t, timestamps[timestamp],
				"Call %d: Timestamp %d should be unique", i, timestamp)
		}

		blockNumbers[blockNum] = true
		timestamps[timestamp] = true
	}
}

// TestCallSequenceDisplayWithoutChainReference verifies that String() handles
// missing ChainReference gracefully (shows "n/a").
func TestCallSequenceDisplayWithoutChainReference(t *testing.T) {
	sender := common.HexToAddress("0x1000")
	recipient := common.HexToAddress("0x2000")

	// Create a call message
	call := NewCallMessage(
		sender,
		&recipient,
		0,
		big.NewInt(1),
		100000,
		big.NewInt(1),
		big.NewInt(0),
		big.NewInt(0),
		[]byte{},
	)

	// Create element without executing (no ChainReference)
	element := NewCallSequenceElement(nil, call, 0, 0)

	// Verify ChainReference is nil
	assert.Nil(t, element.ChainReference)

	// Verify String() output contains "n/a" for block and time
	strOutput := element.String()
	assert.Contains(t, strOutput, "block=n/a")
	assert.Contains(t, strOutput, "time=n/a")
}

// TestCallSequenceMultipleCallsSameBlock verifies that when multiple calls
// are added to the same block with zero delay, they all show the same block
// number and timestamp (as expected).
func TestCallSequenceMultipleCallsSameBlock(t *testing.T) {
	// Create test chain
	testChain, senders := createTestChain(t)
	sender := senders[0]
	recipient := senders[1]

	// Create sequence with three calls, all with zero delay
	// These should all be in the same block with same timestamp
	var callSequence CallSequence
	for i := 0; i < 3; i++ {
		call := NewCallMessage(
			sender,
			&recipient,
			uint64(i),
			big.NewInt(1),
			100000,
			big.NewInt(1),
			big.NewInt(0),
			big.NewInt(0),
			[]byte{},
		)

		element := NewCallSequenceElement(nil, call, 0, 0)
		callSequence = append(callSequence, element)
	}

	// Execute the sequence
	executedSequence, err := ExecuteCallSequence(testChain, callSequence)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(executedSequence))

	// All calls should have the same block number and timestamp
	firstBlockNum := executedSequence[0].ChainReference.BlockNumber
	firstTimestamp := executedSequence[0].ChainReference.BlockTimestamp

	for i := 1; i < len(executedSequence); i++ {
		assert.Equal(t, firstBlockNum, executedSequence[i].ChainReference.BlockNumber,
			"Call %d should have same block number as first call", i)
		assert.Equal(t, firstTimestamp, executedSequence[i].ChainReference.BlockTimestamp,
			"Call %d should have same timestamp as first call", i)
	}

	// Verify all String() outputs show the same block and time
	firstStr := executedSequence[0].String()
	for i := 1; i < len(executedSequence); i++ {
		str := executedSequence[i].String()

		// Extract block= and time= portions
		firstBlock := strings.Split(strings.Split(firstStr, "block=")[1], ",")[0]
		currentBlock := strings.Split(strings.Split(str, "block=")[1], ",")[0]
		assert.Equal(t, firstBlock, currentBlock, "Call %d should show same block number", i)

		firstTime := strings.Split(strings.Split(firstStr, "time=")[1], ",")[0]
		currentTime := strings.Split(strings.Split(str, "time=")[1], ",")[0]
		assert.Equal(t, firstTime, currentTime, "Call %d should show same timestamp", i)
	}
}
