package chain

import (
	"errors"
	"fmt"
	"github.com/crytic/medusa/chain/config"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"golang.org/x/exp/maps"
	"math/big"
	"sort"

	chainTypes "github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/chain/vendored"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

// TestChain represents a simulated Ethereum chain used for testing. It maintains blocks in-memory and strips away
// typical consensus/chain objects to allow for more specialized testing closer to the EVM.
type TestChain struct {
	// blocks represents the blocks created on the current chain. If blocks are sent to the chain which skip some
	// block numbers, any block in that gap will not be committed here and its block hash and other parameters
	// will be spoofed when requested through the API, for efficiency.
	blocks []*chainTypes.Block

	// pendingBlock is a block currently under construction by the chain which has not yet been committed.
	pendingBlock *chainTypes.Block

	// BlockGasLimit defines the maximum amount of gas that can be consumed by transactions in a block.
	// Transactions which push the block gas usage beyond this limit will not be added to a block without error.
	BlockGasLimit uint64

	// testChainConfig represents the configuration used by this TestChain.
	testChainConfig *config.TestChainConfig

	// chainConfig represents the configuration used to instantiate and manage this chain's underlying go-ethereum
	// components.
	chainConfig *params.ChainConfig

	// vmConfigExtensions defines EVM extensions to use with each chain call or transaction.
	vmConfigExtensions *vm.ConfigExtensions

	// genesisDefinition represents the Genesis information used to generate the chain's initial state.
	genesisDefinition *core.Genesis

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

	// callTracerRouter forwards vm.EVMLogger and TestChainTracer calls to any instances added to it. This
	// router is used for non-state changing calls.
	callTracerRouter *TestChainTracerRouter

	// transactionTracerRouter forwards vm.EVMLogger and TestChainTracer calls to any instances added to it. This
	// router is used for transaction execution when constructing blocks.
	transactionTracerRouter *TestChainTracerRouter

	// Events defines the event system for the TestChain.
	Events TestChainEvents
}

// NewTestChain creates a simulated Ethereum backend used for testing, or returns an error if one occurred.
// This creates a test chain with a test chain configuration and the provided genesis allocation and config.
// If a nil config is provided, a default one is used.
func NewTestChain(genesisAlloc core.GenesisAlloc, testChainConfig *config.TestChainConfig) (*TestChain, error) {
	// Copy our chain config, so it is not shared across chains.
	chainConfig, err := utils.CopyChainConfig(params.TestChainConfig)
	if err != nil {
		return nil, err
	}

	// TODO: go-ethereum doesn't set shanghai start time for THEIR test `ChainConfig` struct.
	//   Note: We have our own `TestChainConfig` definition that is different (second argument in this function).
	//  We should allow the user to provide a go-ethereum `ChainConfig` to do custom fork selection, inside of our
	//  `TestChainConfig` definition. Or we should wrap it in our own struct to simplify the options and not pollute
	//  our overall medusa project config.
	shanghaiTime := uint64(0)
	chainConfig.ShanghaiTime = &shanghaiTime

	// Create our genesis definition with our default chain config.
	genesisDefinition := &core.Genesis{
		Config:    chainConfig,
		Nonce:     0,
		Timestamp: 0,
		ExtraData: []byte{
			0x6D, 0x65, 0x64, 0x75, 0x24, 0x61,
		},
		GasLimit:   0,
		Difficulty: common.Big0,
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      maps.Clone(genesisAlloc), // cloned to avoid concurrent access issues across cloned chains
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(0),
	}

	// Use a default config if we were not provided one
	if testChainConfig == nil {
		testChainConfig, err = config.DefaultTestChainConfig()
		if err != nil {
			return nil, err
		}
	}

	// Obtain our VM extensions from our config
	vmConfigExtensions := testChainConfig.GetVMConfigExtensions()

	// Obtain our cheatcode providers
	cheatTracer, cheatContracts, err := getCheatCodeProviders()
	if err != nil {
		return nil, err
	}

	// Add all cheat code contract addresses to the genesis config. This is done because cheat codes are implemented
	// as pre-compiles, but we still want code to exist at these addresses, because smart contracts compiled with
	// newer solidity versions perform code size checks prior to external calls.
	// Additionally, add the pre-compiled cheat code contract to our vm extensions.
	if testChainConfig.CheatCodeConfig.CheatCodesEnabled {
		for _, cheatContract := range cheatContracts {
			genesisDefinition.Alloc[cheatContract.address] = core.GenesisAccount{
				Balance: big.NewInt(0),
				Code:    []byte{0xFF},
			}
			vmConfigExtensions.AdditionalPrecompiles[cheatContract.address] = cheatContract
		}
	}

	// Create an in-memory database
	keyValueStore := memorydb.New()
	db := rawdb.NewDatabase(keyValueStore)

	// Commit our genesis definition to get a genesis block.
	genesisBlock := genesisDefinition.MustCommit(db)

	// Convert our genesis block (go-ethereum type) to a test chain block.
	testChainGenesisBlock := chainTypes.NewBlock(genesisBlock.Header())

	// Create our state database over-top our database.
	stateDatabase := state.NewDatabaseWithConfig(db, &trie.Config{
		Cache: 256, // this is important in keeping the database performant, so it does not need to fetch repetitively.
	})

	// Create a tracer forwarder to support the addition of multiple tracers for transaction and call execution.
	transactionTracerRouter := NewTestChainTracerRouter()
	callTracerRouter := NewTestChainTracerRouter()

	// Create our instance
	chain := &TestChain{
		genesisDefinition:       genesisDefinition,
		BlockGasLimit:           genesisBlock.Header().GasLimit,
		blocks:                  []*chainTypes.Block{testChainGenesisBlock},
		pendingBlock:            nil,
		keyValueStore:           keyValueStore,
		db:                      db,
		state:                   nil,
		stateDatabase:           stateDatabase,
		transactionTracerRouter: transactionTracerRouter,
		callTracerRouter:        callTracerRouter,
		testChainConfig:         testChainConfig,
		chainConfig:             genesisDefinition.Config,
		vmConfigExtensions:      vmConfigExtensions,
	}

	// Add our internal tracers to this chain.
	chain.AddTracer(newTestChainDeploymentsTracer(), true, false)
	if testChainConfig.CheatCodeConfig.CheatCodesEnabled {
		chain.AddTracer(cheatTracer, true, true)
		cheatTracer.bindToChain(chain)
	}

	// Obtain the state for the genesis block and set it as the chain's current state.
	stateDB, err := chain.StateAfterBlockNumber(0)
	if err != nil {
		return nil, err
	}
	chain.state = stateDB
	return chain, nil
}

// Close will release any objects from the TestChain that must be _explicitly_ released. Currently, the one object that
// must be explicitly released is the stateDB trie's underlying cache. This cache, if not released, prevents the TestChain
// object from being freed by the garbage collector and causes a severe memory leak.
func (t *TestChain) Close() {
	// Reset the state DB's cache
	t.stateDatabase.TrieDB().ResetCache()
}

// Clone recreates the current TestChain state into a new instance. This simply reconstructs the block/chain state
// but does not perform any other API-related changes such as adding additional tracers the original had. Additionally,
// this does not clone pending blocks. The provided method, if non-nil, is used as callback to provide an intermediate
// step between chain creation, and copying of all blocks, allowing for tracers to be added.
// Returns the new chain, or an error if one occurred.
func (t *TestChain) Clone(onCreateFunc func(chain *TestChain) error) (*TestChain, error) {
	// Create a new chain with the same genesis definition and config
	targetChain, err := NewTestChain(t.genesisDefinition.Alloc, t.testChainConfig)
	if err != nil {
		return nil, err
	}

	// If we have a provided function for our creation event, execute it now
	if onCreateFunc != nil {
		err = onCreateFunc(targetChain)
		if err != nil {
			return nil, fmt.Errorf("could not clone chain due to error: %v", err)
		}
	}

	// Replay all messages after genesis onto it. We set the block gas limit each time we mine so the chain acts as it
	// did originally.
	for i := 1; i < len(t.blocks); i++ {
		// First create a new pending block to commit
		blockHeader := t.blocks[i].Header
		_, err = targetChain.PendingBlockCreateWithParameters(blockHeader.Number.Uint64(), blockHeader.Time, &blockHeader.GasLimit)
		if err != nil {
			return nil, err
		}

		// Now add each transaction/message to it.
		messages := t.blocks[i].Messages
		for j := 0; j < len(messages); j++ {
			err = targetChain.PendingBlockAddTx(messages[j])
			if err != nil {
				return nil, err
			}
		}

		// Commit the block finally
		err = targetChain.PendingBlockCommit()
		if err != nil {
			return nil, err
		}
	}

	// Set our final block gas limit
	targetChain.BlockGasLimit = t.BlockGasLimit

	// Verify our state
	if targetChain.Head().Hash != t.Head().Hash {
		return nil, errors.New("could not copy chain state onto a new chain, resulting chain head hashes did not match")
	}

	// Return our new chain
	return targetChain, nil
}

// AddTracer adds a given vm.EVMLogger or TestChainTracer to the TestChain. If directed, the tracer will be attached
// for transactions and/or non-state changing calls made via CallContract.
func (t *TestChain) AddTracer(tracer vm.EVMLogger, txs bool, calls bool) {
	if txs {
		t.transactionTracerRouter.AddTracer(tracer)
	}
	if calls {
		t.callTracerRouter.AddTracer(tracer)
	}
}

// GenesisDefinition returns the core.Genesis definition used to initialize the chain.
func (t *TestChain) GenesisDefinition() *core.Genesis {
	return t.genesisDefinition
}

// State returns the current state.StateDB of the chain.
func (t *TestChain) State() *state.StateDB {
	return t.state
}

// CheatCodeContracts returns all cheat code contracts which are installed in the chain.
func (t *TestChain) CheatCodeContracts() map[common.Address]*CheatCodeContract {
	// Create a map of cheat code contracts to store our results
	contracts := make(map[common.Address]*CheatCodeContract, 0)

	// Loop for each precompile, and try to see any which are of the "cheat code contract" type.
	for address, precompile := range t.vmConfigExtensions.AdditionalPrecompiles {
		if cheatCodeContract, ok := precompile.(*CheatCodeContract); ok {
			contracts[address] = cheatCodeContract
		}
	}

	// Return the results
	return contracts
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
	return t.Head().Header.Number.Uint64()
}

// fetchClosestInternalBlock obtains the closest preceding block that is internally committed to the TestChain.
// When the TestChain creates a new block that jumps the block number forward, the existence of any intermediate
// block will be spoofed based off of the closest preceding internally committed block.
// Returns the index of the closest preceding block in blocks and the Block itself.
func (t *TestChain) fetchClosestInternalBlock(blockNumber uint64) (int, *chainTypes.Block) {
	// Perform a binary search for this exact block number, or the closest preceding block we committed.
	k := sort.Search(len(t.blocks), func(n int) bool {
		return t.blocks[n].Header.Number.Uint64() >= blockNumber
	})

	// Determine our closest block index
	var blockIndex int
	if k >= len(t.blocks) {
		// If our result is out of bounds, it means we supplied a block number too high, so we return our head
		blockIndex = len(t.blocks) - 1
	} else if t.blocks[k].Header.Number.Uint64() == blockNumber {
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
	closestBlockNumber := closestBlock.Header.Number.Uint64()

	// If we have an exact match, return it.
	if closestBlockNumber == blockNumber {
		return closestBlock, nil
	}

	// If we didn't have an exact match, it means we skipped block numbers, so we simulate these blocks existing.
	// The block hash for the block we construct will simply be the block number
	blockHash := getSpoofedBlockHashFromNumber(blockNumber)

	// If the block preceding this is the closest internally committed block, we reference that for the previous block
	// hash. Otherwise, we know it's another spoofed block in between.
	previousBlockHash := closestBlock.Hash
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
		Root:        closestBlock.Header.Root,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      big.NewInt(int64(blockNumber)),
		GasLimit:    closestBlock.Header.GasLimit,
		GasUsed:     0,
		Time:        closestBlock.Header.Time + (blockNumber - closestBlockNumber),
		Extra:       []byte{},
		MixDigest:   previousBlockHash,
		Nonce:       types.BlockNonce{},
		BaseFee:     closestBlock.Header.BaseFee,
	}

	// Create our new empty block with our provided header and return it.
	block := chainTypes.NewBlock(blockHeader)
	block.Hash = blockHash // we patch our block hash with our spoofed one immediately
	return block, nil
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
	if closestBlock.Header.Number.Uint64() == blockNumber {
		return closestBlock.Hash, nil
	} else {
		// Otherwise, the block hash comes from a spoofed block we pretend exists, as blocks which jumped block numbers
		// must've been committed. For these blocks we pretend exist in between, their block hash is their block number.
		return getSpoofedBlockHashFromNumber(blockNumber), nil
	}
}

// StateFromRoot obtains a state from a given state root hash.
// Returns the state, or an error if one occurred.
func (t *TestChain) StateFromRoot(root common.Hash) (*state.StateDB, error) {
	// Load our state from the database
	stateDB, err := state.New(root, t.stateDatabase, nil)
	if err != nil {
		return nil, err
	}
	return stateDB, nil
}

// StateRootAfterBlockNumber obtains the Ethereum world state root hash after processing all transactions in the
// provided block number. Returns the state, or an error if one occurs.
func (t *TestChain) StateRootAfterBlockNumber(blockNumber uint64) (common.Hash, error) {
	// If our block number references something too new, return an error
	if blockNumber > t.HeadBlockNumber() {
		return common.Hash{}, fmt.Errorf("could not obtain post-state for block number %d because it exceeds the current head block number %d", blockNumber, t.HeadBlockNumber())
	}

	// Obtain our closest internally committed block
	_, closestBlock := t.fetchClosestInternalBlock(blockNumber)

	// Return our state root hash
	return closestBlock.Header.Root, nil
}

// StateAfterBlockNumber obtains the Ethereum world state after processing all transactions in the provided block
// number. Returns the state, or an error if one occurs.
func (t *TestChain) StateAfterBlockNumber(blockNumber uint64) (*state.StateDB, error) {
	// Obtain our block's post-execution state root hash
	root, err := t.StateRootAfterBlockNumber(blockNumber)
	if err != nil {
		return nil, err
	}

	// Load our state from the database
	return t.StateFromRoot(root)
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
	if closestBlock.Header.Number.Uint64() != blockNumber {
		return fmt.Errorf("could not revert to block number %d because it does not refer to an internally committed block", blockNumber)
	}

	// Slice off our blocks to be removed (to produce relevant events)
	removedBlocks := t.blocks[closestBlockIndex+1:]

	// Remove the relevant blocks from the chain
	t.blocks = t.blocks[:closestBlockIndex+1]

	// Discard our pending block
	err := t.PendingBlockDiscard()
	if err != nil {
		return err
	}

	// Loop backwards through removed blocks to emit reverted contract deployment change events.
	for i := len(removedBlocks) - 1; i >= 0; i-- {
		removedBlock := removedBlocks[i]
		err = t.emitContractChangeEvents(true, removedBlock.MessageResults...)
		if err != nil {
			return err
		}

		// Execute our revert hooks for each block in reverse order.
		for x := len(removedBlock.MessageResults) - 1; x >= 0; x-- {
			removedBlock.MessageResults[x].OnRevertHookFuncs.Execute(false, true)
		}
	}

	// Reload our state from our database
	t.state, err = t.StateAfterBlockNumber(blockNumber)
	if err != nil {
		return err
	}

	// Emit our event for the removed blocks.
	err = t.Events.BlocksRemoved.Publish(BlocksRemovedEvent{
		Chain:  t,
		Blocks: removedBlocks,
	})
	return err
}

// CallContract performs a message call over the current test chain state and obtains a core.ExecutionResult.
// This is similar to the CallContract method provided by Ethereum for use in calling pure/view functions, as it
// executed a transaction without committing any changes, instead discarding them.
// It takes an optional state argument, which is the state to execute the message over. If not provided, the
// current pending state (or committed state if none is pending) will be used instead.
// The state executed over may be a pending block state.
func (t *TestChain) CallContract(msg *core.Message, state *state.StateDB, additionalTracers ...vm.EVMLogger) (*core.ExecutionResult, error) {
	// If our provided state is nil, use our current chain state.
	if state == nil {
		state = t.state
	}

	// Obtain our state snapshot to revert any changes after our call
	snapshot := state.Snapshot()

	// Set infinite balance to the fake caller account
	from := state.GetOrNewStateObject(msg.From)
	from.SetBalance(math.MaxBig256)

	// Create our transaction and block contexts for the vm
	txContext := core.NewEVMTxContext(msg)
	blockContext := newTestChainBlockContext(t, t.Head().Header)

	// Create a new call tracer router that incorporates any additional tracers provided just for this call, while
	// still calling our internal tracers.
	extendedTracerRouter := NewTestChainTracerRouter()
	extendedTracerRouter.AddTracer(t.callTracerRouter)
	extendedTracerRouter.AddTracers(additionalTracers...)

	// Create our EVM instance.
	evm := vm.NewEVM(blockContext, txContext, state, t.chainConfig, vm.Config{
		//Debug:            true,
		Tracer:           extendedTracerRouter,
		NoBaseFee:        true,
		ConfigExtensions: t.vmConfigExtensions,
	})

	// Fund the gas pool, so it can execute endlessly (no block gas limit).
	gasPool := new(core.GasPool).AddGas(math.MaxUint64)

	// Perform our state transition to obtain the result.
	res, err := core.NewStateTransition(evm, msg, gasPool).TransitionDb()

	// Revert to our state snapshot to undo any changes.
	state.RevertToSnapshot(snapshot)

	return res, err
}

// PendingBlock describes the current pending block which is being constructed and awaiting commitment to the chain.
// This may be nil if no pending block was created.
func (t *TestChain) PendingBlock() *chainTypes.Block {
	return t.pendingBlock
}

// PendingBlockCreate constructs an empty block which is pending addition to the chain. The block produces by this
// method will have a block number and timestamp that is greater by the current chain head by 1.
// Returns the constructed block, or an error if one occurred.
func (t *TestChain) PendingBlockCreate() (*chainTypes.Block, error) {
	// Create a block with default parameters
	blockNumber := t.HeadBlockNumber() + 1
	timestamp := t.Head().Header.Time + 1
	return t.PendingBlockCreateWithParameters(blockNumber, timestamp, nil)
}

// PendingBlockCreateWithParameters constructs an empty block which is pending addition to the chain, using the block
// properties provided. Values should be sensibly chosen (e.g., block number and timestamps should be greater than the
// previous block). Providing a block number that is greater than the previous block number plus one will simulate empty
// blocks between.
// Returns the constructed block, or an error if one occurred.
func (t *TestChain) PendingBlockCreateWithParameters(blockNumber uint64, blockTime uint64, blockGasLimit *uint64) (*chainTypes.Block, error) {
	// If we already have a pending block, return an error.
	if t.pendingBlock != nil {
		return nil, fmt.Errorf("could not create a new pending block for chain, as a block is already pending")
	}

	// If our block gas limit is not specified, use the default defined by this chain.
	if blockGasLimit == nil {
		blockGasLimit = &t.BlockGasLimit
	}

	// Validate our block number exceeds our previous head
	currentHeadBlockNumber := t.Head().Header.Number.Uint64()
	if blockNumber <= currentHeadBlockNumber {
		return nil, fmt.Errorf("failed to create block with a block number of %d as does precedes the chain head block number of %d", blockNumber, currentHeadBlockNumber)
	}

	// Obtain our parent block hash to reference in our new block.
	parentBlockHash := t.Head().Hash

	// If the head's block number is not immediately preceding the one we're trying to add, the chain will fake
	// the existence of intermediate blocks, where the block hash is deterministically spoofed based off the block
	// number. Check this condition and substitute the parent block hash if we jumped.
	blockNumberDifference := blockNumber - currentHeadBlockNumber
	if blockNumberDifference > 1 {
		parentBlockHash = getSpoofedBlockHashFromNumber(blockNumber - 1)
	}

	// Timestamps must be unique per block, that means our timestamp must've advanced at least as many steps as the
	// block number for us to spoof the existence of those intermediate blocks, each with their own unique timestamp.
	currentHeadTimeStamp := t.Head().Header.Time
	if currentHeadTimeStamp >= blockTime || blockNumberDifference > blockTime-currentHeadTimeStamp {
		return nil, fmt.Errorf("failed to create block as block number was advanced by %d while block timestamp was advanced by %d. timestamps must be unique per block", blockNumberDifference, blockTime-currentHeadTimeStamp)
	}

	// Create a block header for this block:
	// - State root hash reflects the state after applying block updates (no transactions, so unchanged from last block)
	// - Bloom is aggregated for each transaction in the block (for now empty).
	// - TODO: Difficulty should be revisited/checked.
	// - GasUsed is aggregated for each transaction in the block (for now zero).
	// - Mix digest is only useful for randomness, so we just fake randomness by using the previous block hash.
	// - TODO: BaseFee should be revisited/checked.
	header := &types.Header{
		ParentHash:  parentBlockHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    t.Head().Header.Coinbase,
		Root:        t.Head().Header.Root,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      big.NewInt(int64(blockNumber)),
		GasLimit:    *blockGasLimit,
		GasUsed:     0,
		Time:        blockTime,
		Extra:       []byte{},
		MixDigest:   parentBlockHash,
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}

	// Create a new block for our test node
	t.pendingBlock = chainTypes.NewBlock(header)
	t.pendingBlock.Hash = t.pendingBlock.Header.Hash()

	// Emit our event for the pending block being created
	err := t.Events.PendingBlockCreated.Publish(PendingBlockCreatedEvent{
		Chain: t,
		Block: t.pendingBlock,
	})
	if err != nil {
		return nil, err
	}

	// Return our created block.
	return t.pendingBlock, nil
}

// PendingBlockAddTx takes a message (internal txs) and adds it to the current pending block, updating the header
// with relevant execution information. If a pending block was not created, an error is returned.
// Returns the constructed block, or an error if one occurred.
func (t *TestChain) PendingBlockAddTx(message *core.Message) error {
	// If we don't have a pending block, return an error
	if t.pendingBlock == nil {
		return errors.New("could not add tx to the chain's pending block because no pending block was created")
	}

	// Obtain our state root hash prior to execution.
	previousStateRoot := t.pendingBlock.Header.Root

	// Create a gas pool indicating how much gas can be spent executing the transaction.
	gasPool := new(core.GasPool).AddGas(t.pendingBlock.Header.GasLimit - t.pendingBlock.Header.GasUsed)

	// Create a tx from our msg, for hashing/receipt purposes
	tx := utils.MessageToTransaction(message)

	// Create a new context to be used in the EVM environment
	blockContext := newTestChainBlockContext(t, t.pendingBlock.Header)

	// Create our EVM instance.
	evm := vm.NewEVM(blockContext, core.NewEVMTxContext(message), t.state, t.chainConfig, vm.Config{
		//Debug:            true,
		Tracer:           t.transactionTracerRouter,
		NoBaseFee:        true,
		ConfigExtensions: t.vmConfigExtensions,
	})

	// Apply our transaction
	var usedGas uint64
	t.state.SetTxContext(tx.Hash(), len(t.pendingBlock.Messages))
	receipt, executionResult, err := vendored.EVMApplyTransaction(message, t.chainConfig, &t.pendingBlock.Header.Coinbase, gasPool, t.state, t.pendingBlock.Header.Number, t.pendingBlock.Hash, tx, &usedGas, evm)
	if err != nil {
		// If we encountered an error, reset our state, as we couldn't add the tx.
		t.state, _ = state.New(t.pendingBlock.Header.Root, t.stateDatabase, nil)
		return fmt.Errorf("test chain state write error when adding tx to pending block: %v", err)
	}

	// Create our message result
	messageResult := &chainTypes.MessageResults{
		PreStateRoot:      previousStateRoot,
		PostStateRoot:     common.Hash{},
		ExecutionResult:   executionResult,
		Receipt:           receipt,
		AdditionalResults: make(map[string]any, 0),
	}

	// For every tracer we have, we call upon them to set their results for this transaction now.
	t.transactionTracerRouter.CaptureTxEndSetAdditionalResults(messageResult)

	// Write state changes to database.
	// NOTE: If this completes without an error, we know we didn't hit the block gas limit or other errors, so we are
	// safe to update the block header afterwards.
	root, err := t.state.Commit(t.chainConfig.IsEIP158(t.pendingBlock.Header.Number))
	if err != nil {
		return fmt.Errorf("test chain state write error: %v", err)
	}
	if err := t.state.Database().TrieDB().Commit(root, false); err != nil {
		// If we encountered an error, reset our state, as we couldn't add the tx.
		t.state, _ = state.New(t.pendingBlock.Header.Root, t.stateDatabase, nil)
		return fmt.Errorf("test chain trie write error: %v", err)
	}

	// Update our gas used in the block header
	t.pendingBlock.Header.GasUsed += receipt.GasUsed

	// Update our block's bloom filter
	t.pendingBlock.Header.Bloom.Add(receipt.Bloom.Bytes())

	// Update the header's state root hash, as well as our message result's
	// Note: You could also retrieve the root without committing by using
	// state.IntermediateRoot(config.IsEIP158(parentBlockNumber)).
	t.pendingBlock.Header.Root = root
	messageResult.PostStateRoot = root

	// Update our block's transactions and results.
	t.pendingBlock.Messages = append(t.pendingBlock.Messages, message)
	t.pendingBlock.MessageResults = append(t.pendingBlock.MessageResults, messageResult)

	// Emit our contract change events for this message
	err = t.emitContractChangeEvents(false, messageResult)
	if err != nil {
		return err
	}

	// Emit our event for having added a new transaction to the pending block.
	err = t.Events.PendingBlockAddedTx.Publish(PendingBlockAddedTxEvent{
		Chain:            t,
		Block:            t.pendingBlock,
		TransactionIndex: len(t.pendingBlock.Messages),
	})
	if err != nil {
		return err
	}

	return nil
}

// PendingBlockCommit commits a pending block to the chain, so it is set as the new head. The pending block is set
// to nil after doing so. If there is no pending block when calling this function, an error is returned.
func (t *TestChain) PendingBlockCommit() error {
	// If we have no pending block, we cannot commit it.
	if t.pendingBlock == nil {
		return fmt.Errorf("could not commit chain's pending block, as no pending block was created")
	}

	// Append our new block to our chain.
	t.blocks = append(t.blocks, t.pendingBlock)

	// Clear our pending block, but keep a copy of it to emit our event
	pendingBlock := t.pendingBlock
	t.pendingBlock = nil

	// Emit our event for committing a new block as the chain head
	err := t.Events.PendingBlockCommitted.Publish(PendingBlockCommittedEvent{
		Chain: t,
		Block: pendingBlock,
	})
	if err != nil {
		return err
	}

	return nil
}

// PendingBlockDiscard discards a pending block, allowing a different one to be created.
func (t *TestChain) PendingBlockDiscard() error {
	// If we have no pending block, there is nothing to do.
	if t.pendingBlock == nil {
		return nil
	}

	// Clear our pending block, but keep a copy of it to emit our event
	pendingBlock := t.pendingBlock
	t.pendingBlock = nil

	// Emit our contract change events for the messages reverted
	err := t.emitContractChangeEvents(true, pendingBlock.MessageResults...)
	if err != nil {
		return err
	}

	// Execute our revert hooks for each block in reverse order.
	for i := len(pendingBlock.MessageResults) - 1; i >= 0; i-- {
		pendingBlock.MessageResults[i].OnRevertHookFuncs.Execute(false, true)
	}

	// Reload our state from our database
	t.state, err = t.StateAfterBlockNumber(t.HeadBlockNumber())
	if err != nil {
		return err
	}

	// Emit our pending block discarded event
	err = t.Events.PendingBlockDiscarded.Publish(PendingBlockDiscardedEvent{
		Chain: t,
		Block: pendingBlock,
	})
	if err != nil {
		return err
	}
	return nil
}

// emitContractChangeEvents emits events for contract deployments being added or removed by playing through a list
// of provided message results. If reverting, the inverse events are emitted.
func (t *TestChain) emitContractChangeEvents(reverting bool, messageResults ...*chainTypes.MessageResults) error {
	// If we're not reverting, we simply play events for our contract deployment changes in order. If we are, inverse
	// all the events.
	var err error
	if !reverting {
		// Loop through all deployment changes stored in our call results and emit relevant events.
		for i := 0; i < len(messageResults); i++ {
			for _, deploymentChange := range messageResults[i].ContractDeploymentChanges {
				// We emit the relevant event depending on the contract deployment change, as a block with
				// this execution result is being committed to chain.
				if deploymentChange.Creation {
					err = t.Events.ContractDeploymentAddedEventEmitter.Publish(ContractDeploymentsAddedEvent{
						Chain:             t,
						Contract:          deploymentChange.Contract,
						DynamicDeployment: deploymentChange.DynamicCreation,
					})
				} else if deploymentChange.Destroyed {
					err = t.Events.ContractDeploymentRemovedEventEmitter.Publish(ContractDeploymentsRemovedEvent{
						Chain:    t,
						Contract: deploymentChange.Contract,
					})
				}
				if err != nil {
					return err
				}
			}
		}
	} else {
		// Loop through all deployment changes stored in our call results in reverse order, as we are reverting and wish
		// to invert the events we emitted when these blocks were mined.
		for j := len(messageResults) - 1; j >= 0; j-- {
			result := messageResults[j]
			for k := len(result.ContractDeploymentChanges) - 1; k >= 0; k-- {
				deploymentChange := result.ContractDeploymentChanges[k]

				// We emit the *opposite* event depending on the contract deployment change, as a block with
				// this execution result is being reverted/removed from the chain.
				if deploymentChange.Creation {
					err = t.Events.ContractDeploymentRemovedEventEmitter.Publish(ContractDeploymentsRemovedEvent{
						Chain:    t,
						Contract: deploymentChange.Contract,
					})
				} else if deploymentChange.Destroyed {
					err = t.Events.ContractDeploymentAddedEventEmitter.Publish(ContractDeploymentsAddedEvent{
						Chain:             t,
						Contract:          deploymentChange.Contract,
						DynamicDeployment: deploymentChange.DynamicCreation,
					})
				}
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
