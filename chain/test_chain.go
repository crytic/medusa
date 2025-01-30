package chain

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/crytic/medusa/chain/config"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/holiman/uint256"
	"golang.org/x/exp/maps"

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
	"github.com/ethereum/go-ethereum/params"
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

	// pendingBlockContext is the vm.BlockContext for the current pending block. This is used by cheatcodes to override the EVM
	// interpreter's behavior. This should be set when a new EVM is created by the test chain e.g. using vm.NewEVM.
	pendingBlockContext *vm.BlockContext

	// pendingBlockChainConfig is params.ChainConfig for the current pending block. This is used by cheatcodes to override
	// the chain ID. This should be set when a new EVM is created by the test chain e.g. using vm.NewEVM.
	pendingBlockChainConfig *params.ChainConfig

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

	// callTracerRouter forwards tracers.Tracer and TestChainTracer calls to any instances added to it. This
	// router is used for non-state changing calls.
	callTracerRouter *TestChainTracerRouter

	// transactionTracerRouter forwards tracers.Tracer and TestChainTracer calls to any instances added to it. This
	// router is used for transaction execution when constructing blocks.
	transactionTracerRouter *TestChainTracerRouter

	// Events defines the event system for the TestChain.
	Events TestChainEvents
}

// NewTestChain creates a simulated Ethereum backend used for testing, or returns an error if one occurred.
// This creates a test chain with a test chain configuration and the provided genesis allocation and config.
// If a nil config is provided, a default one is used.
func NewTestChain(genesisAlloc types.GenesisAlloc, testChainConfig *config.TestChainConfig) (*TestChain, error) {
	// Copy our chain config, so it is not shared across chains.
	chainConfig, err := utils.CopyChainConfig(params.TestChainConfig)
	if err != nil {
		return nil, err
	}

	// TODO: go-ethereum doesn't set cancun start time for THEIR test `ChainConfig` struct.
	//   Note: We have our own `TestChainConfig` definition that is different (second argument in this function).
	//  We should allow the user to provide a go-ethereum `ChainConfig` to do custom fork selection, inside of our
	//  `TestChainConfig` definition. Or we should wrap it in our own struct to simplify the options and not pollute
	//  our overall medusa project config.
	cancunTime := uint64(0)
	chainConfig.ShanghaiTime = &cancunTime
	chainConfig.CancunTime = &cancunTime

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

	// Add all cheat code contract addresses to the genesis config. This is done because cheat codes are implemented
	// as pre-compiles, but we still want code to exist at these addresses, because smart contracts compiled with
	// newer solidity versions perform code size checks prior to external calls.
	// Additionally, add the pre-compiled cheat code contract to our vm extensions.
	var cheatTracer *cheatCodeTracer
	if testChainConfig.CheatCodeConfig.CheatCodesEnabled {
		// Obtain our cheatcode providers
		var cheatContracts []*CheatCodeContract
		cheatTracer, cheatContracts, err = getCheatCodeProviders()
		if err != nil {
			return nil, err
		}
		for _, cheatContract := range cheatContracts {
			genesisDefinition.Alloc[cheatContract.address] = types.Account{
				Balance: big.NewInt(0),
				Code:    []byte{0xFF},
			}
			vmConfigExtensions.AdditionalPrecompiles[cheatContract.address] = cheatContract
		}
	}

	// Create an in-memory database
	db := rawdb.NewMemoryDatabase()
	dbConfig := &triedb.Config{
		HashDB: hashdb.Defaults,
		// TODO	Add cleanCacheSize of 256 depending on the resolution of this issue https://github.com/ethereum/go-ethereum/issues/30099
		// PathDB: pathdb.Defaults,
	}
	trieDB := triedb.NewDatabase(db, dbConfig)

	// Commit our genesis definition to get a genesis block.
	genesisBlock := genesisDefinition.MustCommit(db, trieDB)

	// Convert our genesis block (go-ethereum type) to a test chain block.
	testChainGenesisBlock := chainTypes.NewBlock(genesisBlock.Header())

	// Create our state database over-top our database.
	stateDatabase := state.NewDatabaseWithConfig(db, dbConfig)

	// Create a tracer forwarder to support the addition of multiple tracers for transaction and call execution.
	transactionTracerRouter := NewTestChainTracerRouter()
	callTracerRouter := NewTestChainTracerRouter()

	// Create our instance
	chain := &TestChain{
		genesisDefinition:       genesisDefinition,
		BlockGasLimit:           genesisBlock.Header().GasLimit,
		blocks:                  []*chainTypes.Block{testChainGenesisBlock},
		pendingBlock:            nil,
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
	chain.AddTracer(newTestChainDeploymentsTracer().NativeTracer(), true, false)
	if testChainConfig.CheatCodeConfig.CheatCodesEnabled {
		chain.AddTracer(cheatTracer.NativeTracer(), true, true)
		cheatTracer.bindToChain(chain)
	}

	// Obtain the state for the genesis block and set it as the chain's current state.
	stateDB, err := chain.StateAfterBlockNumber(0)
	if err != nil {
		return nil, err
	}

	// Set our state database logger e.g. to monitor OnCodeChange events.
	stateDB.SetLogger(transactionTracerRouter.NativeTracer().Tracer.Hooks)
	chain.state = stateDB
	return chain, nil
}

// Close will release any objects from the TestChain that must be _explicitly_ released. Currently, the one object that
// must be explicitly released is the stateDB trie's underlying cache. This cache, if not released, prevents the TestChain
// object from being freed by the garbage collector and causes a severe memory leak.
func (t *TestChain) Close() {
	// Reset the state DB's cache
	t.stateDatabase.TrieDB().Close()
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

// AddTracer adds a given tracers.Tracer or TestChainTracer to the TestChain. If directed, the tracer will be attached
// for transactions and/or non-state changing calls made via CallContract.
func (t *TestChain) AddTracer(tracer *TestChainTracer, txs bool, calls bool) {
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

// BlockFromNumber obtains the block with the provided block number from the current chain. If the block is not found,
// we return an error with an empty block. Thus, the block must be committed to the chain to be retrieved.
func (t *TestChain) BlockFromNumber(blockNumber uint64) (*chainTypes.Block, error) {
	// Check to see if we have the block in our committed blocks.
	for _, block := range t.blocks {
		if block.Header.Number.Uint64() == blockNumber {
			return block, nil
		}
	}

	// TODO: In the future, we can reintroduce spoofing a block instead of throwing an error.

	// We cannot find the block, so return an error with an empty block.
	return nil, fmt.Errorf("could not find block with block number %v", blockNumber)
}

// BlockHashFromNumber returns a block hash for a given block number. If the block doesn't exist, because it wasn't committed,
// we return an error with an empty hash. Thus, the block must be committed to the chain to be retrieved.
func (t *TestChain) BlockHashFromNumber(blockNumber uint64) (common.Hash, error) {
	// Obtain the block from the chain if it exists
	block, err := t.BlockFromNumber(blockNumber)
	if err != nil {
		return common.Hash{}, err
	}

	// Return the block hash
	return block.Hash, nil
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
// provided block number. If the block doesn't exist, because it wasn't committed,
// we return an error with an empty state root hash. Thus, the block must be committed to the chain.
func (t *TestChain) StateRootAfterBlockNumber(blockNumber uint64) (common.Hash, error) {
	// Obtain the block from the chain if it exists
	block, err := t.BlockFromNumber(blockNumber)
	if err != nil {
		return common.Hash{}, err
	}

	// Return the state root hash
	return block.Header.Root, nil
}

// StateAfterBlockNumber obtains the Ethereum world state after processing all transactions in the provided block
// number. If the block doesn't exist, because it wasn't committed,
// we return an error. Thus, the block must be committed to the chain.
func (t *TestChain) StateAfterBlockNumber(blockNumber uint64) (*state.StateDB, error) {
	// Obtain our block's post-execution state root hash
	root, err := t.StateRootAfterBlockNumber(blockNumber)
	if err != nil {
		return nil, err
	}

	// Load our state from the database
	return t.StateFromRoot(root)
}

// RevertToBlockIndex reverts all blocks after the provided block index and reloads the state from the underlying database.
func (t *TestChain) RevertToBlockIndex(index uint64) error {
	if index > uint64(len(t.blocks)) {
		return fmt.Errorf("could not revert to block index %d because it exceeds the current chain length of %d", index, len(t.blocks))
	}

	// Slice off our blocks to be removed (to produce relevant events)
	removedBlocks := t.blocks[index:]

	// Keep the relevant blocks up till index
	t.blocks = t.blocks[:index]

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

	// Reload our state from our database using the block number at the index we're reverting to.
	t.state, err = t.StateAfterBlockNumber(t.blocks[index-1].Header.Number.Uint64())
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
func (t *TestChain) CallContract(msg *core.Message, state *state.StateDB, additionalTracers ...*TestChainTracer) (*core.ExecutionResult, error) {
	// If our provided state is nil, use our current chain state.
	if state == nil {
		state = t.state
	}

	// Obtain our state snapshot to revert any changes after our call
	snapshot := state.Snapshot()

	// Set infinite balance to the fake caller account
	state.SetBalance(msg.From, uint256.MustFromBig(math.MaxBig256), tracing.BalanceChangeUnspecified)

	// Create our transaction and block contexts for the vm
	txContext := core.NewEVMTxContext(msg)
	blockContext := newTestChainBlockContext(t, t.Head().Header)

	// Create a new call tracer router that incorporates any additional tracers provided just for this call, while
	// still calling our internal tracers.
	extendedTracerRouter := NewTestChainTracerRouter()
	extendedTracerRouter.AddTracer(t.callTracerRouter.NativeTracer())
	extendedTracerRouter.AddTracers(additionalTracers...)

	// Create our EVM instance.
	evm := vm.NewEVM(blockContext, txContext, state, t.chainConfig, vm.Config{
		Tracer:           extendedTracerRouter.NativeTracer().Tracer.Hooks,
		NoBaseFee:        true,
		ConfigExtensions: t.vmConfigExtensions,
	})
	// Set our block context and chain config in order for cheatcodes to override what EVM interpreter sees.
	t.pendingBlockContext = &evm.Context
	t.pendingBlockChainConfig = evm.ChainConfig()

	// Create a tx from our msg, for hashing/receipt purposes
	tx := utils.MessageToTransaction(msg)

	// Need to explicitly call OnTxStart hook
	if evm.Config.Tracer != nil && evm.Config.Tracer.OnTxStart != nil {
		evm.Config.Tracer.OnTxStart(evm.GetVMContext(), tx, msg.From)
	}
	// Fund the gas pool, so it can execute endlessly (no block gas limit).
	gasPool := new(core.GasPool).AddGas(math.MaxUint64)

	// Perform our state transition to obtain the result.
	msgResult, err := core.ApplyMessage(evm, msg, gasPool)

	// Revert to our state snapshot to undo any changes.
	state.RevertToSnapshot(snapshot)

	// Gather receipt for OnTxEnd
	receipt := &types.Receipt{Type: tx.Type()}
	if msgResult.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = msgResult.UsedGas

	// HACK: use OnTxEnd to store the execution trace.
	// Need to explicitly call OnTxEnd
	if evm.Config.Tracer != nil && evm.Config.Tracer.OnTxEnd != nil {
		evm.Config.Tracer.OnTxEnd(receipt, err)
	}

	return msgResult, err
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
// properties provided. Note that there are no constraints on the next block number or timestamp. Because of cheatcode
// usage, the next block can go back in time.
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

	// Obtain our parent block hash to reference in our new block.
	parentBlockHash := t.Head().Hash

	// Note we do not perform any block number or timestamp validation since cheatcodes can permanently update the
	// block number or timestamp which could violate the invariants of a blockchain (e.g. block.number is strictly
	// increasing)

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
		BaseFee:     new(big.Int).Set(t.Head().Header.BaseFee),
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
// Returns an error if one occurred.
func (t *TestChain) PendingBlockAddTx(message *core.Message, additionalTracers ...*TestChainTracer) error {
	// If we don't have a pending block, return an error
	if t.pendingBlock == nil {
		return errors.New("could not add tx to the chain's pending block because no pending block was created")
	}

	// Create a gas pool indicating how much gas can be spent executing the transaction.
	gasPool := new(core.GasPool).AddGas(t.pendingBlock.Header.GasLimit - t.pendingBlock.Header.GasUsed)

	// Create a tx from our msg, for hashing/receipt purposes
	tx := utils.MessageToTransaction(message)

	// Create a new context to be used in the EVM environment
	blockContext := newTestChainBlockContext(t, t.pendingBlock.Header)

	// Create our VM config
	vmConfig := vm.Config{
		NoBaseFee:        true,
		ConfigExtensions: t.vmConfigExtensions,
	}

	// Figure out whether we need to attach any more tracers
	var extendedTracerRouter *TestChainTracerRouter
	if len(additionalTracers) > 0 {
		// If we have more tracers, extend the transaction tracer router's tracers with additional ones
		extendedTracerRouter = NewTestChainTracerRouter()
		extendedTracerRouter.AddTracer(t.transactionTracerRouter.NativeTracer())
		extendedTracerRouter.AddTracers(additionalTracers...)
	} else {
		extendedTracerRouter = t.transactionTracerRouter
	}

	// Update the VM's tracer
	vmConfig.Tracer = extendedTracerRouter.NativeTracer().Tracer.Hooks

	// Set tx context
	t.state.SetTxContext(tx.Hash(), len(t.pendingBlock.Messages))

	// Create our EVM instance.
	evm := vm.NewEVM(blockContext, core.NewEVMTxContext(message), t.state, t.chainConfig, vmConfig)

	// Set our block context and chain config in order for cheatcodes to override what EVM interpreter sees.
	t.pendingBlockContext = &evm.Context
	t.pendingBlockChainConfig = evm.ChainConfig()

	// Apply our transaction
	var usedGas uint64
	receipt, executionResult, err := vendored.EVMApplyTransaction(message, t.chainConfig, t.testChainConfig, &t.pendingBlock.Header.Coinbase, gasPool, t.state, t.pendingBlock.Header.Number, t.pendingBlock.Hash, tx, &usedGas, evm)
	if err != nil {
		return fmt.Errorf("test chain state write error when adding tx to pending block: %v", err)
	}

	// Create our message result
	messageResult := &chainTypes.MessageResults{
		PostStateRoot:     common.BytesToHash(receipt.PostState),
		ExecutionResult:   executionResult,
		Receipt:           receipt,
		AdditionalResults: make(map[string]any, 0),
	}

	// For every tracer we have, we call upon them to set their results for this transaction now.
	t.transactionTracerRouter.CaptureTxEndSetAdditionalResults(messageResult)

	// Update our gas used in the block header
	t.pendingBlock.Header.GasUsed += receipt.GasUsed
	// Update our block's bloom filter
	t.pendingBlock.Header.Bloom.Add(receipt.Bloom.Bytes())
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

	// Perform a state commit to obtain the root hash for our block.
	root, err := t.state.Commit(t.pendingBlock.Header.Number.Uint64(), true)
	t.pendingBlock.Header.Root = root

	if err != nil {
		return err
	}

	// Committing the state invalidates the cached tries and we need to reload the state.
	// Otherwise, methods such as FillFromTestChainProperties will not work correctly.
	t.state, err = state.New(root, t.stateDatabase, nil)
	if err != nil {
		return err
	}

	// Discard the test chain's reference to the EVM interpreter's block context and chain config.
	t.pendingBlockContext = nil
	t.pendingBlockChainConfig = nil

	// Append our new block to our chain.
	// Update the block hash since cheatcodes may have changed aspects of the header (e.g. time or number)
	t.pendingBlock.Hash = t.pendingBlock.Header.Hash()
	t.blocks = append(t.blocks, t.pendingBlock)

	// Clear our pending block, but keep a copy of it to emit our event
	pendingBlock := t.pendingBlock
	t.pendingBlock = nil

	// Emit our event for committing a new block as the chain head
	err = t.Events.PendingBlockCommitted.Publish(PendingBlockCommittedEvent{
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
	t.pendingBlockContext = nil
	t.pendingBlockChainConfig = nil

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
