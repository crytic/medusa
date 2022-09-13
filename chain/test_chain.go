package chain

import (
	"encoding/hex"
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
	"math/big"
	"strings"
)

// TestChain represents a simulated Ethereum chain used for testing. It maintains blocks in-memory and strips away
// typical consensus/chain objects to allow for more specialized testing closer to the EVM.
type TestChain struct {
	// blocks represents the blocks that represent the overarching chain.
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

	// chainConfig represents the configuration used to instantiate and manage this chain.
	chainConfig *params.ChainConfig

	// vmConfig represents a configuration given to the EVM when executing a transaction that specifies parameters
	// such as whether certain fees should be charged and which execution tracer should be used (if any).
	vmConfig *vm.Config
}

// NewTestChain creates a simulated Ethereum backend used for testing, or returns an error if one occurred.
func NewTestChain(genesisAlloc core.GenesisAlloc) (*TestChain, error) {
	// Create an in-memory database
	keyValueStore := memorydb.New()
	db := rawdb.NewDatabase(keyValueStore)

	// Create our genesis block
	genesisDefinition := &core.Genesis{
		Config:    params.TestChainConfig,
		Nonce:     0,
		Timestamp: 0,
		ExtraData: []byte{
			0x6D, 0x65, 0x64, 0x75, 0x24, 0x61,
		},
		GasLimit:   0, // TODO: Set this properly
		Difficulty: common.Big0,
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      genesisAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(0),
	}

	// Commit our genesis definition to get a block.
	genesisBlock := genesisDefinition.MustCommit(db)

	// Convert our genesis block (go-ethereum type) to a test chain block.
	callMessages := make([]*chainTypes.CallMessage, 0)
	for _, tx := range genesisBlock.Transactions() {
		msg := chainTypes.NewCallMessage(common.Address{}, tx.To(), tx.Nonce(), tx.Value(), tx.Gas(), tx.GasPrice(), tx.GasFeeCap(), tx.GasTipCap(), tx.Data())
		callMessages = append(callMessages, msg)
	}
	receipts := make(types.Receipts, 0)
	testChainGenesisBlock, err := chainTypes.NewBlock(genesisBlock.Header().Hash(), genesisBlock.Header(), callMessages, receipts)
	if err != nil {
		return nil, err
	}

	// Create a tracer forwarder to support the addition of multiple underlying tracers.
	tracerForwarder := chainTypes.NewTracerForwarder()

	// Create our instance
	g := &TestChain{
		blocks:        []*chainTypes.Block{testChainGenesisBlock},
		keyValueStore: keyValueStore,
		db:            db,
		state:         nil,
		stateDatabase: state.NewDatabaseWithConfig(db, &trie.Config{
			Cache: 256,
		}),
		tracerForwarder: tracerForwarder,
		chainConfig:     genesisDefinition.Config,
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

// TracerForwarder returns the tracer forwarder used to forward tracing calls to multiple underlying tracers.
func (t *TestChain) TracerForwarder() *chainTypes.TracerForwarder {
	return t.tracerForwarder
}

// MemoryDatabaseEntryCount returns the count of entries in the key-value store which backs the chain.
func (t *TestChain) MemoryDatabaseEntryCount() int {
	return t.keyValueStore.Len()
}

// Head returns the head of the chain (the latest block).
func (t *TestChain) Head() *chainTypes.Block {
	// Return the latest block header
	return t.blocks[len(t.blocks)-1]
}

// Length returns the test chain in blocks.
func (t *TestChain) Length() uint64 {
	return uint64(len(t.blocks))
}

// GasLimit returns the current gas limit for the chain.
func (t *TestChain) GasLimit() uint64 {
	// For now the gas limit remains consistent from genesis.
	return t.Head().Header().GasLimit
}

// BlockNumber returns the test chain head's block number, where zero is the genesis block.
func (t *TestChain) BlockNumber() uint64 {
	return t.Length() - 1
}

// BlockHashFromNumber returns a block hash for a given block number. If the index is out of bounds, it returns
// an error.
func (t *TestChain) BlockHashFromNumber(blockNumber uint64) (common.Hash, error) {
	// If our block number references something too new, return an error
	if blockNumber > t.BlockNumber() {
		return common.Hash{}, fmt.Errorf("could not obtain block hash for block number %d because it exceeds the current chain length of %d", blockNumber, t.Length())
	}

	// Fetch the block hash of a previously recorded block.
	blockHash := t.blocks[blockNumber].Hash()
	return blockHash, nil
}

// StateAfterBlockNumber obtains the Ethereum world state after processing all transactions in the provided block
// number. Returns the state, or an error if one occurs.
func (t *TestChain) StateAfterBlockNumber(blockNumber uint64) (*state.StateDB, error) {
	// If our block number references something too new, return an error
	if blockNumber > t.BlockNumber() {
		return nil, fmt.Errorf("could not obtain post-state for block number %d because it exceeds the current chain length of %d", blockNumber, t.Length())
	}

	// Load our state from the database
	stateDB, err := state.New(t.blocks[blockNumber].Header().Root, t.stateDatabase, nil)
	if err != nil {
		return nil, err
	}
	return stateDB, nil
}

// RevertToBlockNumber sets the head of the chain to the block specified by the provided block number and reloads
// the state from the underlying database.
func (t *TestChain) RevertToBlockNumber(blockNumber uint64) error {
	// Adjust our chain length to match our snapshot
	t.blocks = t.blocks[:blockNumber+1]

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
	gasFeeCap := big.NewInt(0)
	gasTipCap := big.NewInt(0)

	// Construct and return a new message from our given parameters.
	return chainTypes.NewCallMessage(from, to, nonce, value, gasLimit, gasPrice, gasFeeCap, gasTipCap, data)
}

// messageToTransaction derived a types.Transaction from a types.Message.
func messageToTransaction(msg core.Message) *types.Transaction {
	// TODO: This might have issues in the future due to not being given a valid signatures.
	//  This should probably be verified at some point.
	return types.NewTx(&types.LegacyTx{
		Nonce:    msg.Nonce(),
		GasPrice: msg.GasPrice(),
		Gas:      msg.Gas(),
		To:       msg.To(),
		Value:    msg.Value(),
		Data:     msg.Data(),
	})
}

// CallContract performs a message call over the current test chain  state and obtains a core.ExecutionResult.
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

	// Create our EVM instance
	evm := vm.NewEVM(blockContext, txContext, t.state, t.chainConfig, vm.Config{NoBaseFee: true})

	// Fund the gas pool for execution appropriately
	gasPool := new(core.GasPool).AddGas(math.MaxUint64)

	// Perform our state transition to obtain the result.
	res, err := core.NewStateTransition(evm, msg, gasPool).TransitionDb()

	// Revert to our state snapshot to undo any changes.
	t.state.RevertToSnapshot(snapshot)

	return res, err
}

// SendMessages is similar to Ethereum's SendTransaction, it takes messages (internal txs) and applies state updates
// with them, as if transactions were just received in a block. Returns the block representing the result of the state
// transitions.
func (t *TestChain) SendMessages(messages ...*chainTypes.CallMessage) (*chainTypes.Block, error) {
	//parentBlockNumber := big.NewInt(0).Sub(blockNumber, big.NewInt(1))
	parentBlockHash := t.Head().Hash()

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
		Coinbase:    t.Head().Header().Coinbase, // reusing same coinbase throughout the chain
		Root:        types.EmptyRootHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      big.NewInt(int64(t.BlockNumber()) + 1),
		GasLimit:    t.GasLimit(), // reusing same gas limit throughout the chain
		GasUsed:     0,
		Time:        t.BlockNumber() + 1, // TODO: Determine proper timestamp advance logic
		Extra:       []byte{},
		MixDigest:   parentBlockHash,
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}

	// Calculate our block hash for this block
	blockHash := header.Hash()

	// Process each message and collect transaction receipts
	receipts := make(types.Receipts, 0)
	for i := 0; i < len(messages); i++ {
		// Create a tx from our msg, for hashing/receipt purposes
		tx := messageToTransaction(messages[i])

		// Create a new context to be used in the EVM environment
		blockContext := newTestChainBlockContext(t, header)

		// Create our EVM instance.
		evm := vm.NewEVM(blockContext, vm.TxContext{}, t.state, t.chainConfig, *t.vmConfig)

		// Apply our transaction
		var usedGas uint64
		gasPool := new(core.GasPool).AddGas(math.MaxUint64) // TODO: Verify this is sensible
		receipt, err := vendored.EVMApplyTransaction(messages[i].ToEVMMessage(), t.chainConfig, &header.Coinbase, gasPool, t.state, header.Number, blockHash, tx, &usedGas, evm)
		if err != nil {
			return nil, fmt.Errorf("test chain state write error: %v", err)
		}

		// Add our receipt to our list
		receipts = append(receipts, receipt)
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
	// (state.IntermediateRoot(config.IsEIP158(parentBlockNumber)))
	header.Root = root

	// Create a new block for our test node
	block, err := chainTypes.NewBlock(blockHash, header, messages, receipts)

	// Append our new block to our chain and return it.
	t.blocks = append(t.blocks, block)
	return block, nil
}

// DeployContract is a helper method used to deploy a given types.CompiledContract to the current instance of the
// test node, using the address provided as the deployer. Returns the address of the deployed contract if successful,
// otherwise returns an error.
func (t *TestChain) DeployContract(contract *compilationTypes.CompiledContract, deployer common.Address) (common.Address, error) {
	// Obtain the byte code as a byte array
	b, err := hex.DecodeString(strings.TrimPrefix(contract.InitBytecode, "0x"))
	if err != nil {
		return common.Address{}, fmt.Errorf("could not convert compiled contract bytecode from hex string to byte code")
	}

	// Constructor args don't need ABI encoding and appending to the end of the bytecode since there are none for these
	// contracts.

	// Create a message to represent our contract deployment.
	value := big.NewInt(0)
	msg := t.CreateMessage(deployer, nil, value, b)

	// Send our deployment transaction
	block, err := t.SendMessages(msg)
	if err != nil {
		return common.Address{}, err
	}

	// Ensure our transaction succeeded
	if block.Receipts()[0].Status != types.ReceiptStatusSuccessful {
		return common.Address{}, fmt.Errorf("contract deployment tx returned a failed status")
	}

	// Return the address for the deployed contract.
	return block.Receipts()[0].ContractAddress, nil
}
