package chain

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/utils"
	"github.com/trailofbits/medusa/utils/test_utils"
	"math/big"
	"testing"
)

// verifyChain verifies various state properties in a TestChain, such as if previous block hashes are correct,
// timestamps are in order, etc.
func verifyChain(t *testing.T, chain *TestChain) {
	// Assert there are blocks
	assert.Greater(t, len(chain.blocks), 0)

	// Assert that the head is the last block
	assert.EqualValues(t, chain.blocks[len(chain.blocks)-1], chain.Head())

	// Loop through all blocks
	for i := len(chain.blocks) - 1; i >= 0; i-- {
		// Verify our method to fetch block hashes works appropriately.
		blockHash, err := chain.BlockHashFromNumber(uint64(i))
		assert.NoError(t, err)
		assert.EqualValues(t, chain.blocks[i].Hash(), blockHash)

		// Try to obtain the state for this block
		_, err = chain.StateAfterBlockNumber(uint64(i))
		assert.NoError(t, err)

		// If we're not on the last item, verify our previous block hash matches, and our timestamp is greater.
		if i > 0 {
			assert.EqualValues(t, chain.blocks[i].Header().ParentHash, chain.blocks[i-1].Hash())
			assert.Greater(t, chain.blocks[i].Header().Time, chain.blocks[i-1].Header().Time)
		}
	}
}

// TestCallSequenceReplayMatchSimple creates a TestChain, sends some messages to it, then creates another chain which
// it replays the same sequence on. It ensures that the ending state is the same.
// Note: this does not set block timestamps or other data that might be non-deterministic.
// This does not test replaying with a previous call sequence with different timestamps, etc. It expects the TestChain
// semantics to be the same whenever run with the same messages being sent for all the same blocks.
func TestCallSequenceReplayMatchSimple(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/contracts/deployment_test.sol")

	// Execute our tests in the given test path
	test_utils.ExecuteInDirectory(t, contractPath, func() {
		// Create a solc provider
		solc := platforms.NewSolcCompilationConfig(contractPath)

		// Obtain our compilations and ensure we didn't encounter an error
		compilations, _, err := solc.Compile()
		assert.NoError(t, err)
		assert.True(t, len(compilations) > 0)

		// Create our list of senders
		senders, err := utils.HexStringsToAddresses([]string{
			"0x0707",
			"0x0708",
			"0x0709",
		})
		assert.NoError(t, err)

		// NOTE: Sharing GenesisAlloc between nodes will result in some accounts not being funded for some reason.
		genesisAlloc := make(core.GenesisAlloc)

		// Fund all of our sender addresses in the genesis block
		initBalance := new(big.Int).Div(abi.MaxInt256, big.NewInt(2))
		for _, sender := range senders {
			genesisAlloc[sender] = core.GenesisAccount{
				Balance: initBalance,
			}
		}

		// Create a test chain
		chain, err := NewTestChain(genesisAlloc)
		assert.NoError(t, err)

		// Deploy each contract 10 times.
		for _, compilation := range compilations {
			for _, source := range compilation.Sources {
				for _, contract := range source.Contracts {
					contract := contract
					for i := 0; i < 10; i++ {
						// Deploy the currently indexed contract
						_, err = chain.DeployContract(&contract, senders[0])
						assert.NoError(t, err)

						// Ensure we could get our state
						_, err = chain.StateAfterBlockNumber(chain.BlockNumber())
						assert.NoError(t, err)

						// Create some empty blocks and ensure we can get our state for this block number.
						for x := 0; x < i; x++ {
							_, err = chain.CreateNewBlock()
							assert.NoError(t, err)

							_, err = chain.StateAfterBlockNumber(chain.BlockNumber())
							assert.NoError(t, err)
						}
					}
				}
			}
		}

		// Create another test chain which we will recreate our state from.
		recreatedChain, err := NewTestChain(genesisAlloc)
		assert.NoError(t, err)

		// Replay all messages after genesis
		for i := uint64(1); i < chain.Length(); i++ {
			_, err := recreatedChain.CreateNewBlock(chain.blocks[i].Messages()...)
			assert.NoError(t, err)
		}

		// Verify both chains
		verifyChain(t, chain)
		verifyChain(t, recreatedChain)

		// Verify our final block hashes equal in both chains.
		assert.EqualValues(t, chain.Head().Hash(), recreatedChain.Head().Hash())
		assert.EqualValues(t, chain.Head().Header().Hash(), recreatedChain.Head().Header().Hash())
	})
}
