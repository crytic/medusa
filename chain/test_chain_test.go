package chain

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa/compilation/platforms"
	"github.com/crytic/medusa/utils"
	"github.com/crytic/medusa/utils/testutils"
	"github.com/stretchr/testify/assert"
)

// verifyChain verifies various state properties in a TestChain, such as if previous block hashes are correct,
// timestamps are in order, etc.
func verifyChain(t *testing.T, chain *TestChain) {
	// Assert there are blocks
	assert.Greater(t, len(chain.blocks), 0)

	// Assert that the head is the last block
	assert.EqualValues(t, chain.blocks[len(chain.blocks)-1], chain.Head())

	// Loop through all blocks
	for _, currentBlock := range chain.blocks {
		// Verify our count of messages, message results, and receipts match.
		assert.EqualValues(t, len(currentBlock.Messages), len(currentBlock.MessageResults))

		// Try to obtain the state for this block
		_, err := chain.StateAfterBlockNumber(uint64(currentBlock.Header.Number.Uint64()))
		assert.NoError(t, err)
	}
}

// createChain creates a TestChain used for unit testing purposes and returns the chain along with its initially
// funded accounts at genesis.
func createChain(t *testing.T) (*TestChain, []common.Address) {
	// Create our list of senders
	senders, err := utils.HexStringsToAddresses([]string{
		"0x0707",
		"0x0708",
		"0x0709",
	})
	assert.NoError(t, err)

	// NOTE: Sharing GenesisAlloc between nodes will result in some accounts not being funded for some reason.
	genesisAlloc := make(types.GenesisAlloc)

	// Fund all of our sender addresses in the genesis block
	initBalance := new(big.Int).Div(abi.MaxInt256, big.NewInt(2))
	for _, sender := range senders {
		genesisAlloc[sender] = types.Account{
			Balance: initBalance,
		}
	}

	// Create a test chain with a default test chain configuration
	chain, err := NewTestChain(context.Background(), genesisAlloc, nil)

	assert.NoError(t, err)

	return chain, senders
}

// TestChainReverting creates a TestChain and creates blocks and later reverts backward through all possible steps
// to ensure no error occurs and the chain state is restored.
func TestChainReverting(t *testing.T) {
	// Define our probability of jumping
	const blockNumberJumpProbability = 0.20
	const blockNumberJumpMin = 1
	const blockNumberJumpMax = 100
	const blocksToProduce = 50

	// Obtain our chain and senders
	chain, _ := createChain(t)
	chainBackups := make([]*TestChain, 0)

	// Create some empty blocks and ensure we can get our state for this block number.
	for x := 0; x < blocksToProduce; x++ {
		// Determine if this will jump a certain number of blocks.
		if rand.Float32() >= blockNumberJumpProbability {
			// We decided not to jump, so we commit a block with a normal consecutive block number.
			_, err := chain.PendingBlockCreate()
			assert.NoError(t, err)
			err = chain.PendingBlockCommit()
			assert.NoError(t, err)
		} else {
			// We decided to jump, so we commit a block with a random jump amount.
			newBlockNumber := chain.HeadBlockNumber()
			jumpDistance := (rand.Uint64() % (blockNumberJumpMax - blockNumberJumpMin)) + blockNumberJumpMin
			newBlockNumber += jumpDistance

			// Determine the jump amount, each block must have a unique timestamp, so we ensure it advanced by at least
			// the diff.

			// Create a block with our parameters
			_, err := chain.PendingBlockCreateWithParameters(newBlockNumber, chain.Head().Header.Time+jumpDistance, nil)
			assert.NoError(t, err)
			err = chain.PendingBlockCommit()
			assert.NoError(t, err)
		}

		// Clone our chain
		backup, err := chain.Clone(nil)
		assert.NoError(t, err)
		chainBackups = append(chainBackups, backup)
	}

	// Our chain backups should be in chronological order, so we loop backwards through them and test reverts.
	for i := len(chainBackups) - 1; i >= 0; i-- {
		// Alias our chain backup
		chainBackup := chainBackups[i]

		// Revert our main chain to this block height.
		err := chain.RevertToBlockIndex(uint64(len(chainBackup.CommittedBlocks())))
		assert.NoError(t, err)

		// Verify state matches
		// Verify both chains
		verifyChain(t, chain)
		verifyChain(t, chainBackup)

		// Verify our final block hashes equal in both chains.
		assert.EqualValues(t, chainBackup.Head().Hash, chain.Head().Hash)
		assert.EqualValues(t, chainBackup.Head().Header.Hash(), chain.Head().Header.Hash())
		assert.EqualValues(t, chainBackup.Head().Header.Root, chain.Head().Header.Root)
	}
}

// TestChainBlockNumberJumping creates a TestChain and creates blocks with block numbers which jumped (are
// non-consecutive) to ensure the chain appropriately spoofs intermediate blocks.
func TestChainBlockNumberJumping(t *testing.T) {
	// Define our probability of jumping
	const blockNumberJumpProbability = 0.20
	const blockNumberJumpMin = 1
	const blockNumberJumpMax = 100
	const blocksToProduce = 200

	// Obtain our chain and senders
	chain, _ := createChain(t)

	// Create some empty blocks and ensure we can get our state for this block number.
	for x := 0; x < blocksToProduce; x++ {
		// Determine if this will jump a certain number of blocks.
		if rand.Float32() >= blockNumberJumpProbability {
			// We decided not to jump, so we commit a block with a normal consecutive block number.
			_, err := chain.PendingBlockCreate()
			assert.NoError(t, err)
			err = chain.PendingBlockCommit()
			assert.NoError(t, err)
		} else {
			// We decided to jump, so we commit a block with a random jump amount.
			newBlockNumber := chain.HeadBlockNumber()
			jumpDistance := (rand.Uint64() % (blockNumberJumpMax - blockNumberJumpMin)) + blockNumberJumpMin
			newBlockNumber += jumpDistance

			// Determine the jump amount, each block must have a unique timestamp, so we ensure it advanced by at least
			// the diff.

			// Create a block with our parameters
			_, err := chain.PendingBlockCreateWithParameters(newBlockNumber, chain.Head().Header.Time+jumpDistance, nil)
			assert.NoError(t, err)
			err = chain.PendingBlockCommit()
			assert.NoError(t, err)
		}
	}

	// Clone our chain
	recreatedChain, err := chain.Clone(nil)
	assert.NoError(t, err)

	// Verify both chains
	verifyChain(t, chain)
	verifyChain(t, recreatedChain)

	// Verify our final block hashes equal in both chains.
	assert.EqualValues(t, chain.Head().Hash, recreatedChain.Head().Hash)
	assert.EqualValues(t, chain.Head().Header.Hash(), recreatedChain.Head().Header.Hash())
	assert.EqualValues(t, chain.Head().Header.Root, recreatedChain.Head().Header.Root)
}

// TestChainDynamicDeployments creates a TestChain, deploys a contract which dynamically deploys another contract,
// and ensures that both contract deployments were detected by the TestChain. It also creates empty blocks it
// verifies have no registered contract deployments.
func TestChainDynamicDeployments(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/contracts/deployment_with_inner.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create a crytic compile provider
		cryticCompile := platforms.NewCryticCompilationConfig(contractPath)

		// Obtain our compilations and ensure we didn't encounter an error
		compilations, _, err := cryticCompile.Compile()
		assert.NoError(t, err)
		assert.EqualValues(t, 1, len(compilations))
		assert.EqualValues(t, 1, len(compilations[0].SourcePathToArtifact))

		// Obtain our chain and senders
		chain, senders := createChain(t)

		// Deploy each contract that has no construct arguments.
		deployCount := 0
		for _, compilation := range compilations {
			for _, source := range compilation.SourcePathToArtifact {
				for _, contract := range source.Contracts {
					if len(contract.Abi.Constructor.Inputs) == 0 {
						// Listen for contract changes
						deployedContracts := 0
						chain.Events.ContractDeploymentAddedEventEmitter.Subscribe(func(event ContractDeploymentsAddedEvent) error {
							deployedContracts++
							return nil
						})
						chain.Events.ContractDeploymentRemovedEventEmitter.Subscribe(func(event ContractDeploymentsRemovedEvent) error {
							deployedContracts--
							return nil
						})

						// Deploy the currently indexed contract next
						// Create a message to represent our contract deployment.
						msg := core.Message{
							To:               nil,
							From:             senders[0],
							Nonce:            chain.State().GetNonce(senders[0]),
							Value:            big.NewInt(0),
							GasLimit:         chain.BlockGasLimit,
							GasPrice:         big.NewInt(1),
							GasFeeCap:        big.NewInt(0),
							GasTipCap:        big.NewInt(0),
							Data:             contract.InitBytecode,
							AccessList:       nil,
							SkipNonceChecks:  false,
							SkipFromEOACheck: false,
						}

						// Create a new pending block we'll commit to chain
						block, err := chain.PendingBlockCreate()
						assert.NoError(t, err)

						// Add our transaction to the block
						err = chain.PendingBlockAddTx(&msg)
						assert.NoError(t, err)

						// Commit the pending block to the chain, so it becomes the new head.
						err = chain.PendingBlockCommit()
						assert.NoError(t, err)

						// Ensure our transaction succeeded
						assert.EqualValues(t, types.ReceiptStatusSuccessful, block.MessageResults[0].Receipt.Status, "contract deployment tx returned a failed status: %v", block.MessageResults[0].ExecutionResult.Err)
						deployCount++

						// There should've been two address deployments, an outer and inner deployment.
						// (tx deployment and dynamic deployment).
						assert.EqualValues(t, 1, len(block.MessageResults))
						assert.EqualValues(t, 2, deployedContracts)

						// Ensure we could get our state
						_, err = chain.StateAfterBlockNumber(chain.HeadBlockNumber())
						assert.NoError(t, err)

						// Create some empty blocks and ensure we can get our state for this block number.
						for x := 0; x < 5; x++ {
							block, err = chain.PendingBlockCreate()
							assert.NoError(t, err)
							err = chain.PendingBlockCommit()
							assert.NoError(t, err)

							// Empty blocks should not record message results or dynamic deployments.
							assert.EqualValues(t, 0, len(block.MessageResults))

							_, err = chain.StateAfterBlockNumber(chain.HeadBlockNumber())
							assert.NoError(t, err)
						}
					}
				}
			}
		}

		// Clone our chain
		recreatedChain, err := chain.Clone(nil)
		assert.NoError(t, err)

		// Verify both chains
		verifyChain(t, chain)
		verifyChain(t, recreatedChain)

		// Verify our final block hashes equal in both chains.
		assert.EqualValues(t, chain.Head().Hash, recreatedChain.Head().Hash)
		assert.EqualValues(t, chain.Head().Header.Hash(), recreatedChain.Head().Header.Hash())
		assert.EqualValues(t, chain.Head().Header.Root, recreatedChain.Head().Header.Root)
	})
}

// TestChainDeploymentWithArgs creates a TestChain, deploys a contract which accepts constructor arguments,
// and ensures that constructor arguments were set successfully. It also creates empty blocks it verifies
// have no registered contract deployments.
func TestChainDeploymentWithArgs(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/contracts/deployment_with_args.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create a crytic compile provider
		cryticCompile := platforms.NewCryticCompilationConfig(contractPath)

		// Obtain our compilations and ensure we didn't encounter an error
		compilations, _, err := cryticCompile.Compile()
		assert.NoError(t, err)
		assert.EqualValues(t, 1, len(compilations))
		assert.EqualValues(t, 1, len(compilations[0].SourcePathToArtifact))

		// Obtain our chain and senders
		chain, senders := createChain(t)

		// Don't change the argument y, if length of bytes array changes then we will need to
		// read its value from different storage slots and this test will fail because it
		// reads the value from only a single precalculated slot assuming that length of the
		// bytes array is fixed at 32 bytes
		args := make(map[string][]any)
		x := big.NewInt(1234567890)
		y := []byte("Test deployment with arguments!!")
		args["DeploymentWithArgs"] = []any{x, y}

		// Deploy each contract
		deployCount := 0
		for _, compilation := range compilations {
			for _, source := range compilation.SourcePathToArtifact {
				for contractName, contract := range source.Contracts {

					// Listen for contract changes
					deployedContracts := 0
					chain.Events.ContractDeploymentAddedEventEmitter.Subscribe(func(event ContractDeploymentsAddedEvent) error {
						deployedContracts++
						return nil
					})
					chain.Events.ContractDeploymentRemovedEventEmitter.Subscribe(func(event ContractDeploymentsRemovedEvent) error {
						deployedContracts--
						return nil
					})

					// Obtain our message data to represent the deployment with the provided constructor args.
					msgData, err := contract.GetDeploymentMessageData(args[contractName])
					assert.NoError(t, err)

					// Create a message to represent our contract deployment.
					msg := core.Message{
						To:               nil,
						From:             senders[0],
						Nonce:            chain.State().GetNonce(senders[0]),
						Value:            big.NewInt(0),
						GasLimit:         chain.BlockGasLimit,
						GasPrice:         big.NewInt(1),
						GasFeeCap:        big.NewInt(0),
						GasTipCap:        big.NewInt(0),
						Data:             msgData,
						AccessList:       nil,
						SkipNonceChecks:  false,
						SkipFromEOACheck: false,
					}

					// Create a new pending block we'll commit to chain
					block, err := chain.PendingBlockCreate()
					assert.NoError(t, err)

					// Add our transaction to the block
					err = chain.PendingBlockAddTx(&msg)
					assert.NoError(t, err)

					// Commit the pending block to the chain, so it becomes the new head.
					err = chain.PendingBlockCommit()
					assert.NoError(t, err)

					// Ensure our transaction succeeded
					assert.EqualValues(t, types.ReceiptStatusSuccessful, block.MessageResults[0].Receipt.Status, "contract deployment tx returned a failed status: %v", block.MessageResults[0].ExecutionResult.Err)
					deployCount++

					assert.EqualValues(t, 1, len(block.MessageResults))
					assert.EqualValues(t, 1, deployedContracts)

					// Ensure we could get our state
					contractAddress := block.MessageResults[0].Receipt.ContractAddress
					stateDB, err := chain.StateAfterBlockNumber(chain.HeadBlockNumber())
					assert.NoError(t, err)

					// Verify contract state variables x and y
					slotX := "0x0000000000000000000000000000000000000000000000000000000000000000"
					contractX := stateDB.GetState(contractAddress, common.HexToHash(slotX)).Big()
					assert.EqualValues(t, x, contractX)

					// first element of bytes array is stored at slot number keccak256(uint256(1))
					slotY := "0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6"
					contractY := stateDB.GetState(contractAddress, common.HexToHash(slotY)).Bytes()
					assert.EqualValues(t, y, contractY)

					// Create some empty blocks and ensure we can get our state for this block number.
					for x := 0; x < 5; x++ {
						block, err = chain.PendingBlockCreate()
						assert.NoError(t, err)
						err = chain.PendingBlockCommit()
						assert.NoError(t, err)

						// Empty blocks should not record message results or dynamic deployments.
						assert.EqualValues(t, 0, len(block.MessageResults))

						_, err = chain.StateAfterBlockNumber(chain.HeadBlockNumber())
						assert.NoError(t, err)
					}
				}
			}
		}

		// Clone our chain
		recreatedChain, err := chain.Clone(nil)
		assert.NoError(t, err)

		// Verify both chains
		verifyChain(t, chain)
		verifyChain(t, recreatedChain)

		// Verify our final block hashes equal in both chains.
		assert.EqualValues(t, chain.Head().Hash, recreatedChain.Head().Hash)
		assert.EqualValues(t, chain.Head().Header.Hash(), recreatedChain.Head().Header.Hash())
		assert.EqualValues(t, chain.Head().Header.Root, recreatedChain.Head().Header.Root)
	})
}

// TestChainCloning creates a TestChain, sends some messages to it, then clones it into a new instance and ensures
// that the ending state is the same.
func TestChainCloning(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/contracts/deployment_single.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create a crytic-compile provider
		cryticCompile := platforms.NewCryticCompilationConfig(contractPath)

		// Obtain our compilations and ensure we didn't encounter an error
		compilations, _, err := cryticCompile.Compile()
		assert.NoError(t, err)
		assert.True(t, len(compilations) > 0)

		// Obtain our chain and senders
		chain, senders := createChain(t)

		// Deploy each contract that has no construct arguments 10 times.
		for _, compilation := range compilations {
			for _, source := range compilation.SourcePathToArtifact {
				for _, contract := range source.Contracts {
					if len(contract.Abi.Constructor.Inputs) == 0 {
						for i := 0; i < 10; i++ {
							// Deploy the currently indexed contract next

							// Create a message to represent our contract deployment.
							msg := core.Message{
								To:               nil,
								From:             senders[0],
								Nonce:            chain.State().GetNonce(senders[0]),
								Value:            big.NewInt(0),
								GasLimit:         chain.BlockGasLimit,
								GasPrice:         big.NewInt(1),
								GasFeeCap:        big.NewInt(0),
								GasTipCap:        big.NewInt(0),
								Data:             contract.InitBytecode,
								AccessList:       nil,
								SkipNonceChecks:  false,
								SkipFromEOACheck: false,
							}

							// Create a new pending block we'll commit to chain
							block, err := chain.PendingBlockCreate()
							assert.NoError(t, err)

							// Add our transaction to the block
							err = chain.PendingBlockAddTx(&msg)
							assert.NoError(t, err)

							// Commit the pending block to the chain, so it becomes the new head.
							err = chain.PendingBlockCommit()
							assert.NoError(t, err)

							// Ensure our transaction succeeded
							assert.EqualValues(t, types.ReceiptStatusSuccessful, block.MessageResults[0].Receipt.Status, "contract deployment tx returned a failed status: %v", block.MessageResults[0].ExecutionResult.Err)

							// Ensure we could get our state
							_, err = chain.StateAfterBlockNumber(chain.HeadBlockNumber())
							assert.NoError(t, err)

							// Create some empty blocks and ensure we can get our state for this block number.
							for x := 0; x < i; x++ {
								_, err = chain.PendingBlockCreate()
								assert.NoError(t, err)
								err = chain.PendingBlockCommit()
								assert.NoError(t, err)

								_, err = chain.StateAfterBlockNumber(chain.HeadBlockNumber())
								assert.NoError(t, err)
							}
						}
					}
				}
			}
		}

		// Clone our chain
		recreatedChain, err := chain.Clone(nil)
		assert.NoError(t, err)

		// Verify both chains
		verifyChain(t, chain)
		verifyChain(t, recreatedChain)

		// Verify our final block hashes equal in both chains.
		assert.EqualValues(t, chain.Head().Hash, recreatedChain.Head().Hash)
		assert.EqualValues(t, chain.Head().Header.Hash(), recreatedChain.Head().Header.Hash())
		assert.EqualValues(t, chain.Head().Header.Root, recreatedChain.Head().Header.Root)
	})
}

// TestChainCallSequenceReplayMatchSimple creates a TestChain, sends some messages to it, then creates another chain which
// it replays the same sequence on. It ensures that the ending state is the same.
// Note: this does not set block timestamps or other data that might be non-deterministic.
// This does not test replaying with a previous call sequence with different timestamps, etc. It expects the TestChain
// semantics to be the same whenever run with the same messages being sent for all the same blocks.
func TestChainCallSequenceReplayMatchSimple(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/contracts/deployment_single.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create a crytic-compile provider
		cryticCompile := platforms.NewCryticCompilationConfig(contractPath)

		// Obtain our compilations and ensure we didn't encounter an error
		compilations, _, err := cryticCompile.Compile()
		assert.NoError(t, err)
		assert.True(t, len(compilations) > 0)

		// Obtain our chain and senders
		chain, senders := createChain(t)

		// Deploy each contract that has no construct arguments 10 times.
		for _, compilation := range compilations {
			for _, source := range compilation.SourcePathToArtifact {
				for _, contract := range source.Contracts {
					if len(contract.Abi.Constructor.Inputs) == 0 {
						for i := 0; i < 10; i++ {
							// Create a message to represent our contract deployment.
							msg := core.Message{
								To:               nil,
								From:             senders[0],
								Nonce:            chain.State().GetNonce(senders[0]),
								Value:            big.NewInt(0),
								GasLimit:         chain.BlockGasLimit,
								GasPrice:         big.NewInt(1),
								GasFeeCap:        big.NewInt(0),
								GasTipCap:        big.NewInt(0),
								Data:             contract.InitBytecode,
								AccessList:       nil,
								SkipNonceChecks:  false,
								SkipFromEOACheck: false,
							}

							// Create a new pending block we'll commit to chain
							block, err := chain.PendingBlockCreate()
							assert.NoError(t, err)

							// Add our transaction to the block
							err = chain.PendingBlockAddTx(&msg)
							assert.NoError(t, err)

							// Commit the pending block to the chain, so it becomes the new head.
							err = chain.PendingBlockCommit()
							assert.NoError(t, err)

							// Ensure our transaction succeeded
							assert.EqualValues(t, types.ReceiptStatusSuccessful, block.MessageResults[0].Receipt.Status, "contract deployment tx returned a failed status: %v", block.MessageResults[0].ExecutionResult.Err)

							// Ensure we could get our state
							_, err = chain.StateAfterBlockNumber(chain.HeadBlockNumber())
							assert.NoError(t, err)

							// Create some empty blocks and ensure we can get our state for this block number.
							for x := 0; x < i; x++ {
								_, err = chain.PendingBlockCreate()
								assert.NoError(t, err)
								err = chain.PendingBlockCommit()
								assert.NoError(t, err)

								_, err = chain.StateAfterBlockNumber(chain.HeadBlockNumber())
								assert.NoError(t, err)
							}
						}
					}
				}
			}
		}

		// Create another test chain which we will recreate our state from.
		recreatedChain, err := NewTestChain(context.Background(), chain.genesisDefinition.Alloc, nil)
		assert.NoError(t, err)

		// Replay all messages after genesis
		for i := 1; i < len(chain.blocks); i++ {
			_, err := recreatedChain.PendingBlockCreate()
			assert.NoError(t, err)
			for _, message := range chain.blocks[i].Messages {
				err = recreatedChain.PendingBlockAddTx(message)
				assert.NoError(t, err)
			}
			err = recreatedChain.PendingBlockCommit()
			assert.NoError(t, err)
		}

		// Verify both chains
		verifyChain(t, chain)
		verifyChain(t, recreatedChain)

		// Verify our final block hashes equal in both chains.
		assert.EqualValues(t, chain.Head().Hash, recreatedChain.Head().Hash)
		assert.EqualValues(t, chain.Head().Header.Hash(), recreatedChain.Head().Header.Hash())
		assert.EqualValues(t, chain.Head().Header.Root, recreatedChain.Head().Header.Root)
	})
}
