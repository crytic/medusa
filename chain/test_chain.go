package chain

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	"github.com/trailofbits/medusa/chain/vendored"
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/events"
	"github.com/trailofbits/medusa/utils"
	"golang.org/x/exp/slices"
	"math/big"
	"sort"
)

// TestChain represents a simulated Ethereum chain used for testing. It maintains blocks in-memory and strips away
// typical consensus/chain objects to allow for more specialized testing closer to the EVM.
type TestChain struct {
	// genesisDefinition represents the Genesis information used to generate the chain's initial state.
	genesisDefinition *core.Genesis

	// blocks represents the blocks created on the current chain. If blocks are sent to the chain which skip some
	// block numbers, any block in that gap will not be committed here and its block hash and other parameters
	// will be spoofed when requested through the API, for efficiency.
	blocks []*chainTypes.Block

	// state represents the current Ethereum world state.StateDB. It tracks all state across the chain and dummyChain
	// and is the subject of state changes when executing new transactions. This does not track the current block
	// head or anything of that nature and simply tracks accounts, balances, code, storage, etc.
	state *state.StateDB

	// stateDatabase refers to the database object which state uses to store data. It is constructed over db.
	stateDatabase state.Database

	// db represents the in-memory database used by the TestChain and its underlying chain to store state changes.
	// This is constructed over the kvstore.
	db ethdb.Database

	// keyValueStore represents the underlying key-value store used to construct the db.
	keyValueStore *memorydb.Database

	// tracerForwarder represents an execution trace provider for the VM which observes VM execution for any notable
	// events and forwards them to underlying vm.EVMLogger tracers.
	tracerForwarder *chainTypes.TracerForwarder

	// internalTracer is a testChainTracer used to serve portions of the TestChain API.
	internalTracer *testChainTracer

	// chainConfig represents the configuration used to instantiate and manage this chain.
	chainConfig *params.ChainConfig

	// vmConfig represents a configuration given to the EVM when executing a transaction that specifies parameters
	// such as whether certain fees should be charged and which execution tracer should be used (if any).
	vmConfig *vm.Config

	// BlockMiningEventEmitter emits events indicating a block was added to the chain.
	BlockMiningEventEmitter events.EventEmitter[BlockMiningEvent]
	// BlockMinedEventEmitter emits events indicating a block was added to the chain.
	BlockMinedEventEmitter events.EventEmitter[BlockMinedEvent]
	// BlockRemovedEventEmitter emits events indicating a block was removed from the chain.
	BlockRemovedEventEmitter events.EventEmitter[BlockRemovedEvent]

	// ContractDeploymentAddedEventEmitter emits events indicating a new contract was deployed to the chain.
	ContractDeploymentAddedEventEmitter events.EventEmitter[ContractDeploymentsAddedEvent]
	// ContractDeploymentAddedEventEmitter emits events indicating a previously deployed contract was removed
	// from the chain.
	ContractDeploymentRemovedEventEmitter events.EventEmitter[ContractDeploymentsRemovedEvent]
}

// NewTestChain creates a simulated Ethereum backend used for testing, or returns an error if one occurred.
// This creates a test chain with a default test chain configuration and the provided genesis allocation.
func NewTestChain(genesisAlloc core.GenesisAlloc) (*TestChain, error) {
	// Create our genesis definition with our default chain config.
	genesisDefinition := &core.Genesis{
		Config:    params.TestChainConfig,
		Nonce:     0,
		Timestamp: 0,
		ExtraData: []byte{
			0x6D, 0x65, 0x64, 0x75, 0x24, 0x61,
		},
		GasLimit:   0,
		Difficulty: common.Big0,
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      genesisAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(0),
	}

	// Create the test chain with this genesis definition.
	return NewTestChainWithGenesis(genesisDefinition)
}

// NewTestChainWithGenesis creates a simulated Ethereum backend used for testing, or returns an error if one occurred.
// The genesis definition provided is used to construct the genesis block and specify the chain configuration.
func NewTestChainWithGenesis(genesisDefinition *core.Genesis) (*TestChain, error) {
	// Create an in-memory database
	keyValueStore := memorydb.New()
	db := rawdb.NewDatabase(keyValueStore)

	// Commit our genesis definition to get a block.
	genesisBlock := genesisDefinition.MustCommit(db)

	// Convert our genesis block (go-ethereum type) to a test chain block.
	callMessages := make([]*chainTypes.CallMessage, 0)
	callMessageResults := make([]*chainTypes.CallMessageResults, 0)
	receipts := make(types.Receipts, 0)
	testChainGenesisBlock, err := chainTypes.NewBlock(genesisBlock.Header().Hash(), genesisBlock.Header(), callMessages, receipts, callMessageResults)
	if err != nil {
		return nil, err
	}

	// Create our state database over-top our database.
	stateDatabase := state.NewDatabaseWithConfig(db, &trie.Config{
		Cache: 256,
	})

	// Create a tracer forwarder to support the addition of multiple underlying tracers.
	tracerForwarder := chainTypes.NewTracerForwarder()

	// Add our test chain tracer.
	internalTracer := newTestChainTracer()
	tracerForwarder.AddTracer(internalTracer)

	// Create our instance
	g := &TestChain{
		genesisDefinition: genesisDefinition,
		blocks:            []*chainTypes.Block{testChainGenesisBlock},
		keyValueStore:     keyValueStore,
		db:                db,
		state:             nil,
		stateDatabase:     stateDatabase,
		tracerForwarder:   tracerForwarder,
		internalTracer:    internalTracer,
		chainConfig:       genesisDefinition.Config,
		vmConfig: &vm.Config{
			Debug:     true,
			Tracer:    tracerForwarder,
			NoBaseFee: true,
		},
	}

	// Obtain the state for the genesis block and set it as the chain's current state.
	stateDB, err := g.StateAfterBlockNumber(0)
	if err != nil {
		return nil, err
	}
	g.state = stateDB
	return g, nil
}

// Clone recreates the current TestChain state into a new instance. This simply reconstructs the block/chain state
// but does not perform any other API-related changes such as adding additional tracers the original had, unless
// otherwise specified in function input parameters.
// Returns the new chain, or an error if one occurred.
func (t *TestChain) Clone(tracers ...vm.EVMLogger) (*TestChain, error) {
	// Create a new chain with the same genesis definition
	chain, err := NewTestChainWithGenesis(t.genesisDefinition)
	if err != nil {
		return nil, err
	}

	// Add our tracers to the new chain.
	chain.TracerForwarder().AddTracers(tracers...)

	// Copy our current chain state onto the new chain.
	err = t.CopyTo(chain)
	if err != nil {
		return nil, err
	}

	// Return our new chain
	return chain, nil
}

// CopyTo recreates the current TestChain state onto the provided one. This simply reconstructs the block/chain state
// by sending the same call messages with the same block creation properties.
// Returns an error if one occurred.
func (t *TestChain) CopyTo(targetChain *TestChain) error {
	if targetChain.blocks[0].Hash() != t.blocks[0].Hash() {
		return errors.New("could not copy chain state onto a new chain because the genesis block hashes did not match")
	}

	// If the head block number is not genesis, revert
	if targetChain.HeadBlockNumber() > 0 {
		err := targetChain.RevertToBlockNumber(0)
		if err != nil {
			return err
		}
	}

	// Replay all messages after genesis onto it.
	for i := 1; i < len(t.blocks); i++ {
		blockHeader := t.blocks[i].Header()
		blockNumber := blockHeader.Number.Uint64()
		_, err := targetChain.MineBlockWithParameters(blockNumber, blockHeader.Time, t.blocks[i].Messages()...)
		if err != nil {
			return err
		}
	}
	return nil
}

// TracerForwarder returns the tracer forwarder used to forward tracing calls to multiple underlying tracers.
func (t *TestChain) TracerForwarder() *chainTypes.TracerForwarder {
	return t.tracerForwarder
}

// GenesisDefinition returns the core.Genesis definition used to initialize the chain.
func (t *TestChain) GenesisDefinition() *core.Genesis {
	return t.genesisDefinition
}

// MemoryDatabaseEntryCount returns the count of entries in the key-value store which backs the chain.
func (t *TestChain) MemoryDatabaseEntryCount() int {
	return t.keyValueStore.Len()
}

// CommittedBlocks returns the real blocks which were committed to the chain, where methods such as BlockFromNumber
// return the simulated chain state with intermediate blocks injected for block number jumps, etc.
func (t *TestChain) CommittedBlocks() []*chainTypes.Block {
	return t.blocks
}

// Head returns the head of the chain (the latest block).
func (t *TestChain) Head() *chainTypes.Block {
	return t.blocks[len(t.blocks)-1]
}

// HeadBlockNumber returns the test chain head's block number, where zero is the genesis block.
func (t *TestChain) HeadBlockNumber() uint64 {
	return t.Head().Header().Number.Uint64()
}

// GasLimit returns the current gas limit for the chain.
func (t *TestChain) GasLimit() uint64 {
	// TODO: Determine if/how we'd like to support GasLimit changes. We could leave it static and create a setter
	//  method for use only with the API, in the least.

	// For now the gas limit remains consistent from genesis.
	return t.Head().Header().GasLimit
}

// fetchClosestInternalBlock obtains the closest preceding block that is internally committed to the TestChain.
// When the TestChain creates a new block that jumps the block number forward, the existence of any intermediate
// block will be spoofed based off of the closest preceding internally committed block.
// Returns the index of the closest preceding block in blocks and the Block itself.
func (t *TestChain) fetchClosestInternalBlock(blockNumber uint64) (int, *chainTypes.Block) {
	// Perform a binary search for this exact block number, or the closest preceding block we committed.
	k := sort.Search(len(t.blocks), func(n int) bool {
		return t.blocks[n].Header().Number.Uint64() >= blockNumber
	})

	// Determine our closest block index
	var blockIndex int
	if k >= len(t.blocks) {
		// If our result is out of bounds, it means we supplied a block number too high, so we return our head
		blockIndex = len(t.blocks) - 1
	} else if t.blocks[k].Header().Number.Uint64() == blockNumber {
		// If our result is an exact match, k is our block index
		blockIndex = k
	} else {
		// If the result is not an exact match, binary search will return the index where the block number should've
		// existed. This means the closest preceding block is just the index before
		blockIndex = k - 1
	}

	// Return the resulting block index and block.
	return blockIndex, t.blocks[blockIndex]
}

// BlockFromNumber obtains the block with the provided block number from the current chain. If blocks committed to
// the TestChain skip block numbers, this method will simulate the existence of well-formed intermediate blocks to
// ensure chain validity throughout. Thus, this is a "simulated" chain API method.
// Returns the block, or an error if one occurs.
func (t *TestChain) BlockFromNumber(blockNumber uint64) (*chainTypes.Block, error) {
	// If the block number is past our current head, return an error.
	if blockNumber > t.HeadBlockNumber() {
		return nil, fmt.Errorf("could not obtain block for block number %d because it exceeds the current head block number %d", blockNumber, t.HeadBlockNumber())
	}

	// We only commit blocks that were created by this chain. If block numbers are skipped, we simulate their existence
	// by returning deterministic values for them (block hash, timestamp). This helps us avoid actually creating
	// thousands of blocks to jump forward in time, while maintaining chain integrity (so BLOCKHASH instructions
	// continue to work).

	// First, search for this exact block number, or the closest preceding block we committed to derive information
	// from. There will always be one, given the genesis block always exists.
	_, closestBlock := t.fetchClosestInternalBlock(blockNumber)
	closestBlockNumber := closestBlock.Header().Number.Uint64()

	// If we have an exact match, return it.
	if closestBlockNumber == blockNumber {
		return closestBlock, nil
	}

	// If we didn't have an exact match, it means we skipped block numbers, so we simulate these blocks existing.
	// The block hash for the block we construct will simply be the block number
	blockHash := getSpoofedBlockHashFromNumber(blockNumber)

	// If the block preceding this is the closest internally committed block, we reference that for the previous block
	// hash. Otherwise, we know it's another spoofed block in between.
	previousBlockHash := closestBlock.Hash()
	if closestBlockNumber != blockNumber-1 {
		previousBlockHash = getSpoofedBlockHashFromNumber(blockNumber - 1)
	}

	// Create our new block header
	// - Unchanged state from last block
	// - No transactions or receipts
	// - Reuses gas limit from last committed block.
	// - We reuse the previous timestamp and add 1 for every block generated (blocks must have different timestamps)
	//   - Note: This means that we must check that our timestamp jump >= block number jump when committing a new block.
	blockHeader := &types.Header{
		ParentHash:  previousBlockHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        closestBlock.Header().Root,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      big.NewInt(int64(blockNumber)),
		GasLimit:    closestBlock.Header().GasLimit,
		GasUsed:     0,
		Time:        closestBlock.Header().Time + (blockNumber - closestBlockNumber),
		Extra:       []byte{},
		MixDigest:   previousBlockHash,
		Nonce:       types.BlockNonce{},
		BaseFee:     closestBlock.Header().BaseFee,
	}

	// Create our transaction-related fields (empty, non-state changing).
	messages := make([]*chainTypes.CallMessage, 0)
	messageResults := make([]*chainTypes.CallMessageResults, 0)
	receipts := make(types.Receipts, 0)

	// Create our new block and return it.
	block, err := chainTypes.NewBlock(blockHash, blockHeader, messages, receipts, messageResults)
	return block, err
}

// getSpoofedBlockHashFromNumber is a helper method used to create a deterministic hash for a spoofed block at a given
// block number. This avoids costly calculation of potentially thousands of simulated blocks and allows us to generate
// simulated blocks on demand, rather than storing them.
func getSpoofedBlockHashFromNumber(blockNumber uint64) common.Hash {
	// For blocks which were not internally committed, which we fake the existence of (for block number jumping), we use
	// the block number as the block hash.
	return common.BigToHash(new(big.Int).SetUint64(blockNumber))
}

// BlockHashFromNumber returns a block hash for a given block number. If the index is out of bounds, it returns
// an error.
func (t *TestChain) BlockHashFromNumber(blockNumber uint64) (common.Hash, error) {
	// If our block number references something too new, return an error
	if blockNumber > t.HeadBlockNumber() {
		return common.Hash{}, fmt.Errorf("could not obtain block hash for block number %d because it exceeds the current head block number %d", blockNumber, t.HeadBlockNumber())
	}

	// Obtain our closest internally committed block
	_, closestBlock := t.fetchClosestInternalBlock(blockNumber)

	// If the block is an exact match, return its hash.
	if closestBlock.Header().Number.Uint64() == blockNumber {
		return closestBlock.Hash(), nil
	} else {
		// Otherwise, the block hash comes from a spoofed block we pretend exists, as blocks which jumped block numbers
		// must've been committed. For these blocks we pretend exist in between, their block hash is their block number.
		return getSpoofedBlockHashFromNumber(blockNumber), nil
	}
}

// StateAfterBlockNumber obtains the Ethereum world state after processing all transactions in the provided block
// number. Returns the state, or an error if one occurs.
func (t *TestChain) StateAfterBlockNumber(blockNumber uint64) (*state.StateDB, error) {
	// If our block number references something too new, return an error
	if blockNumber > t.HeadBlockNumber() {
		return nil, fmt.Errorf("could not obtain post-state for block number %d because it exceeds the current head block number %d", blockNumber, t.HeadBlockNumber())
	}

	// Obtain our closest internally committed block
	_, closestBlock := t.fetchClosestInternalBlock(blockNumber)

	// Load our state from the database
	stateDB, err := state.New(closestBlock.Header().Root, t.stateDatabase, nil)
	if err != nil {
		return nil, err
	}
	return stateDB, nil
}

// RevertToBlockNumber sets the head of the chain to the block specified by the provided block number and reloads
// the state from the underlying database.
func (t *TestChain) RevertToBlockNumber(blockNumber uint64) error {
	// If our block number references something too new, return an error
	if blockNumber > t.HeadBlockNumber() {
		return fmt.Errorf("could not revert to block number %d because it exceeds the current head block number %d", blockNumber, t.HeadBlockNumber())
	}

	// Obtain our closest internally committed block, if it's not an exact match, it means we're trying to revert
	// to a spoofed block, which we disallow for now.
	closestBlockIndex, closestBlock := t.fetchClosestInternalBlock(blockNumber)
	if closestBlock.Header().Number.Uint64() != blockNumber {
		return fmt.Errorf("could not revert to block number %d because it does not refer to an internally committed block", blockNumber)
	}

	// Slice off our blocks to be removed (to produce relevant events)
	removedBlocks := t.blocks[closestBlockIndex+1:]

	// Remove the relevant blocks from the chain
	t.blocks = t.blocks[:closestBlockIndex+1]

	// Emit our events for newly deployed contracts
	for i := len(removedBlocks) - 1; i >= 0; i-- {
		// Emit our event for removing a block
		err := t.BlockRemovedEventEmitter.Publish(BlockRemovedEvent{
			Chain: t,
			Block: removedBlocks[i],
		})
		if err != nil {
			return err
		}

		// For each call message in our block, if we had any resulting deployed smart contracts, signal that they have
		// now been removed.
		for _, messageResult := range removedBlocks[i].MessageResults() {
			if len(messageResult.DeployedContractBytecodes) > 0 {
				err = t.ContractDeploymentRemovedEventEmitter.Publish(ContractDeploymentsRemovedEvent{
					Chain:                     t,
					DeployedContractBytecodes: messageResult.DeployedContractBytecodes,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	// Reload our state from our database
	var err error
	t.state, err = t.StateAfterBlockNumber(blockNumber)
	return err
}

// CreateMessage creates an object which satisfies the types.Message interface. It populates gas limit, price, nonce,
// and other fields automatically, and sets fee/tip caps such that no base fee is charged (for testing).
func (t *TestChain) CreateMessage(from common.Address, to *common.Address, value *big.Int, data []byte) *chainTypes.CallMessage {
	// Obtain our message parameters
	nonce := t.state.GetNonce(from)
	gasLimit := t.GasLimit()
	gasPrice := big.NewInt(1)

	// Setting fee and tip cap to zero alongside the NoBaseFee for the vm.Config will bypass base fee validation.
	// TODO: Set this appropriately for newer transaction types.
	gasFeeCap := big.NewInt(0)
	gasTipCap := big.NewInt(0)

	// Construct and return a new message from our given parameters.
	return chainTypes.NewCallMessage(from, to, nonce, value, gasLimit, gasPrice, gasFeeCap, gasTipCap, data)
}

// CallContract performs a message call over the current test chain state and obtains a core.ExecutionResult.
// This is similar to the CallContract method provided by Ethereum for use in calling pure/view functions.
func (t *TestChain) CallContract(msg *chainTypes.CallMessage) (*core.ExecutionResult, error) {
	// Obtain our state snapshot (note: this is different from the test chain snapshot)
	snapshot := t.state.Snapshot()

	// Set infinite balance to the fake caller account
	from := t.state.GetOrNewStateObject(msg.From())
	from.SetBalance(math.MaxBig256)

	// Create our transaction and block contexts for the vm
	txContext := core.NewEVMTxContext(msg)
	blockContext := newTestChainBlockContext(t, t.Head().Header())

	// Create our EVM instance.
	evm := vm.NewEVM(blockContext, txContext, t.state, t.chainConfig, vm.Config{NoBaseFee: true})

	// Fund the gas pool, so it can execute endlessly (no block gas limit).
	gasPool := new(core.GasPool).AddGas(math.MaxUint64)

	// Perform our state transition to obtain the result.
	res, err := core.NewStateTransition(evm, msg, gasPool).TransitionDb()

	// Revert to our state snapshot to undo any changes.
	t.state.RevertToSnapshot(snapshot)

	return res, err
}

// MineBlock takes messages (internal txs), constructs a block with them, and updates the chain head, similar to a
// real chain using  transactions to construct a block.
// Returns the constructed block, or an error if one occurred.
func (t *TestChain) MineBlock(messages ...*chainTypes.CallMessage) (*chainTypes.Block, error) {
	// Create a block with default parameters
	blockNumber := t.HeadBlockNumber() + 1
	timestamp := t.Head().Header().Time + 1 // TODO: Find a sensible default step for timestamp.
	return t.MineBlockWithParameters(blockNumber, timestamp, messages...)
}

// MineBlockWithParameters takes messages (internal txs), constructs a block with them, and updates the chain head,
// similar to a real chain using transactions to construct a block. It accepts block header parameters used to produce
// the block. Values should be sensibly chosen (e.g., block number and timestamps should be greater than the previous
// block). Providing a block number that is greater than the previous block number plus one will simulate empty blocks
// between.
// Returns the constructed block, or an error if one occurred.
func (t *TestChain) MineBlockWithParameters(blockNumber uint64, blockTime uint64, messages ...*chainTypes.CallMessage) (*chainTypes.Block, error) {
	// Emit our event for mining a new block
	err := t.BlockMiningEventEmitter.Publish(BlockMiningEvent{
		Chain: t,
	})
	if err != nil {
		return nil, err
	}

	// Validate our block number exceeds our previous head
	currentHeadBlockNumber := t.Head().Header().Number.Uint64()
	if blockNumber <= currentHeadBlockNumber {
		return nil, fmt.Errorf("failed to create block with a block number of %d as does precedes the chain head block number of %d", blockNumber, currentHeadBlockNumber)
	}

	// Obtain our parent block hash to reference in our new block.
	parentBlockHash := t.Head().Hash()

	// If the head's block number is not immediately preceding the one we're trying to add, the chain will fake
	// the existence of intermediate blocks, where the block hash is deterministically spoofed based off the block
	// number. Check this condition and substitute the parent block hash if we jumped.
	blockNumberDifference := blockNumber - currentHeadBlockNumber
	if blockNumberDifference > 1 {
		parentBlockHash = getSpoofedBlockHashFromNumber(blockNumber - 1)
	}

	// Timestamps must be unique per block, that means our timestamp must've advanced at least as many steps as the
	// block number for us to spoof the existence of those intermediate blocks, each with their own unique timestamp.
	currentHeadTimeStamp := t.Head().Header().Time
	if currentHeadTimeStamp >= blockTime {
		return nil, fmt.Errorf("failed to create block with a timestamp of %d as it precedes the chain head timestamp of %d", blockTime, currentHeadTimeStamp)
	}
	if currentHeadTimeStamp >= blockTime || blockNumberDifference > blockTime-currentHeadTimeStamp {
		return nil, fmt.Errorf("failed to create block as block number was advanced by %d while block timestamp was advanced by %d. timestamps must be unique per block", blockNumberDifference, blockTime-currentHeadTimeStamp)
	}

	// Create a block header for this block:
	// - Root hashes are not populated on first run.
	// - State root hash is populated later in this method.
	// - Bloom is not populated on first run.
	// - TODO: Difficulty is not proven to be safe
	// - GasUsed is not populated on first run.
	// - Mix digest is only useful for randomness, so we just use previous block hash.
	// - TODO: Figure out appropriate params for BaseFee
	header := &types.Header{
		ParentHash:  parentBlockHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    t.Head().Header().Coinbase,
		Root:        t.Head().Header().Root,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      big.NewInt(int64(blockNumber)),
		GasLimit:    t.GasLimit(),
		GasUsed:     0,
		Time:        blockTime,
		Extra:       []byte{},
		MixDigest:   parentBlockHash,
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}

	// Calculate our block hash for this block
	blockHash := header.Hash()

	// Create our gas pool that lets us execute up to the full gas limit.
	gasPool := new(core.GasPool).AddGas(header.GasLimit)

	// Process each message and collect transaction receipts and results
	messageResults := make([]*chainTypes.CallMessageResults, 0)
	receipts := make(types.Receipts, 0)
	for i := 0; i < len(messages); i++ {
		// Create a tx from our msg, for hashing/receipt purposes
		tx := utils.MessageToTransaction(messages[i])

		// Create a new context to be used in the EVM environment
		blockContext := newTestChainBlockContext(t, header)

		// Create our EVM instance.
		evm := vm.NewEVM(blockContext, core.NewEVMTxContext(messages[i]), t.state, t.chainConfig, *t.vmConfig)

		// Apply our transaction
		var usedGas uint64
		receipt, err := vendored.EVMApplyTransaction(messages[i].ToEVMMessage(), t.chainConfig, &header.Coinbase, gasPool, t.state, header.Number, blockHash, tx, &usedGas, evm)
		if err != nil {
			return nil, fmt.Errorf("test chain state write error: %v", err)
		}

		// Update our gas used in the block header
		header.GasUsed += receipt.GasUsed

		// Update our block's bloom filter
		header.Bloom.Add(receipt.Bloom.Bytes())

		// Add our receipt to our list
		receipts = append(receipts, receipt)

		// Create our execution result and append it to the list.
		// - We take the deployed contract addresses detected by the tracer and copy them into our results.
		messageResults = append(messageResults, &chainTypes.CallMessageResults{
			DeployedContractBytecodes: slices.Clone(t.internalTracer.deployedContractBytecode),
		})
	}

	// Write state changes to db
	root, err := t.state.Commit(t.chainConfig.IsEIP158(header.Number))
	if err != nil {
		return nil, fmt.Errorf("test chain state write error: %v", err)
	}
	if err := t.state.Database().TrieDB().Commit(root, false, nil); err != nil {
		return nil, fmt.Errorf("test chain trie write error: %v", err)
	}

	// Update the header's state root hash
	// Note: You could also retrieve the root without committing by using
	// state.IntermediateRoot(config.IsEIP158(parentBlockNumber)).
	header.Root = root

	// Create a new block for our test node
	block, err := chainTypes.NewBlock(blockHash, header, messages, receipts, messageResults)

	// Append our new block to our chain.
	t.blocks = append(t.blocks, block)

	// Emit our event for mining a new block
	err = t.BlockMinedEventEmitter.Publish(BlockMinedEvent{
		Chain: t,
		Block: block,
	})
	if err != nil {
		return nil, err
	}

	// Emit our events for newly deployed contracts
	for _, messageResult := range messageResults {
		if len(messageResult.DeployedContractBytecodes) > 0 {
			err = t.ContractDeploymentAddedEventEmitter.Publish(ContractDeploymentsAddedEvent{
				Chain:                     t,
				DeployedContractBytecodes: messageResult.DeployedContractBytecodes,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	// Return our created block.
	return block, nil
}

// DeployContract is a helper method used to deploy a given types.CompiledContract to the current instance of the
// test node, using the address provided as the deployer. Returns the address of the deployed contract if successful,
// the resulting block the deployment transaction was processed in, and an error if one occurred.
func (t *TestChain) DeployContract(contract *compilationTypes.CompiledContract, deployer common.Address) (common.Address, *chainTypes.Block, error) {
	// Obtain the byte code as a byte array
	b, err := contract.InitBytecodeBytes()
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("could not convert compiled contract bytecode from hex string to byte code")
	}

	// Constructor args don't need ABI encoding and appending to the end of the bytecode since there are none for these
	// contracts.

	// Create a message to represent our contract deployment.
	value := big.NewInt(0)
	msg := t.CreateMessage(deployer, nil, value, b)

	// Create a new block with our deployment message/tx.
	block, err := t.MineBlock(msg)
	if err != nil {
		return common.Address{}, nil, err
	}

	// Ensure our transaction succeeded
	if block.Receipts()[0].Status != types.ReceiptStatusSuccessful {
		return common.Address{}, block, fmt.Errorf("contract deployment tx returned a failed status")
	}

	// Return the address for the deployed contract.
	return block.Receipts()[0].ContractAddress, block, nil
}
