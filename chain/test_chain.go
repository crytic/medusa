package chain

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/params"
	chainTypes "github.com/trailofbits/medusa/chain/types"
	"github.com/trailofbits/medusa/chain/vendored"
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"math/big"
	"strings"
)

// TestNode represents a simulated Ethereum backend used for testing.
type TestNode struct {
	// chain represents the blocks post-genesis that were generated after sending messages.
	chain []*chainTypes.Block

	// dummyChain is core.BlockChain instance which is used to construct the genesis state. After such point it is
	// only used for reference to fulfill the needs of some methods.
	dummyChain *core.BlockChain

	// kvstore represents the underlying key-value store used to construct the db.
	kvstore *memorydb.Database

	// db represents the in-memory database used by the TestNode and its underlying chain and dummyChain objects to
	// store state changes.
	db ethdb.Database

	// state represents the current Ethereum world state.StateDB. It tracks all state across the chain and dummyChain
	// and is the subject of state changes when executing new transactions. This does not track the current block
	// head or anything of that nature and simply tracks accounts, balances, code, storage, etc.
	state *state.StateDB

	// snapshot is an identifier which is used by the Snapshot and RevertToSnapshot method. It represents a block number
	// on the chain which we should return to when RevertToSnapshot is used.
	snapshot int

	// tracerForwarder represents an execution trace provider for the VM which observes VM execution for any notable
	// events and forwards them to underlying vm.EVMLogger tracers.
	tracerForwarder *chainTypes.TracerForwarder

	// vmConfig represents a configuration given to the EVM when executing a transaction that specifies parameters
	// such as whether certain fees should be charged and which execution tracer should be used (if any).
	vmConfig *vm.Config
}

// NewTestNode creates a simulated Ethereum backend used for testing, or returns an error if one occurred.
func NewTestNode(genesisAlloc core.GenesisAlloc) (*TestNode, error) {
	// Define our chain configuration
	chainConfig := params.TestChainConfig

	// Create an in-memory database
	kvstore := memorydb.New()
	db := rawdb.NewDatabase(kvstore)

	// Create our genesis block
	genesisDefinition := &core.Genesis{
		Config: chainConfig,
		Alloc:  genesisAlloc,
		ExtraData: []byte{
			0x6D, 0x65, 0x64, 0x75, 0x24, 0x61,
		},
	}

	// Commit our genesis definition to get a block.
	genesisDefinition.MustCommit(db)

	// Create a multi-tracer to support the addition of multiple underlying tracers.
	tracerForwarder := chainTypes.NewTracerForwarder()
	vmConfig := &vm.Config{
		Debug:     true,
		Tracer:    tracerForwarder,
		NoBaseFee: true,
	}

	// Create a new blockchain provider
	dummyChain, err := core.NewBlockChain(db, nil, chainConfig, ethash.NewFullFaker(), *vmConfig, nil, nil)
	if err != nil {
		return nil, err
	}

	// Obtain our current state
	stateDb, err := dummyChain.State()
	if err != nil {
		return nil, err
	}

	// Create our instance
	g := &TestNode{
		chain:           make([]*chainTypes.Block, 0),
		dummyChain:      dummyChain,
		kvstore:         kvstore,
		db:              db,
		state:           stateDb,
		tracerForwarder: tracerForwarder,
		vmConfig:        vmConfig,
	}

	return g, nil
}

// TracerForwarder returns the tracer forwarder used to forward tracing calls to multiple underlying tracers.
func (t *TestNode) TracerForwarder() *chainTypes.TracerForwarder {
	return t.tracerForwarder
}

// BlockNumber returns the test chain head's block number, where zero is the genesis block.
func (t *TestNode) BlockNumber() uint64 {
	// Our chain length is genesis block + test node blocks
	return uint64(len(t.chain))
}

// BlockHeader returns the block header of the current test chain head.
func (t *TestNode) BlockHeader() *types.Header {
	// If we have any blocks on the test chain, return the latest
	if len(t.chain) > 0 {
		return t.chain[len(t.chain)-1].Header()
	}

	// Otherwise return the genesis header
	return t.dummyChain.CurrentHeader()
}

// BlockHashFromBlockNumber returns a block hash for a given block number. If the index is out of bounds, it returns
// an error.
func (t *TestNode) BlockHashFromBlockNumber(blockNumber uint64) (common.Hash, error) {
	// If our block number references something too new, return an error
	if blockNumber > uint64(len(t.chain)) {
		return common.Hash{}, fmt.Errorf("could not obtain block hash for block number %d because it exceeds the current chain length of %d", blockNumber, len(t.chain)+1)
	}

	// Fetch either the genesis block hash or one of our later simulated block hashes.
	if blockNumber == 0 {
		genesisHash := t.dummyChain.CurrentBlock().Hash()
		return genesisHash, nil
	} else {
		blockHash := t.chain[blockNumber-1].Hash()
		return blockHash, nil
	}
}

// MemoryDatabaseEntryCount returns the count of entries in the key-value store which backs the chain.
func (t *TestNode) MemoryDatabaseEntryCount() int {
	return t.kvstore.Len()
}

// Stop is a TestNode method used to tear down the node.
func (t *TestNode) Stop() {
	// Stop the underlying chain's update loop
	t.dummyChain.Stop()
}

// Snapshot saves the given chain state, which can later be reverted to by calling the RevertToSnapshot
// method.
func (t *TestNode) Snapshot() {
	// Save our snapshot (block height)
	t.snapshot = len(t.chain)
}

// RevertToSnapshot uses a snapshot set by the Snapshot method to revert the test chain state to its previous state.
// Returns an error if one occurs.
func (t *TestNode) RevertToSnapshot() error {
	var err error

	// Adjust our chain length to match our snapshot
	t.chain = t.chain[:t.snapshot]

	// Reload our state from our database
	t.state, err = state.New(t.BlockHeader().Root, t.state.Database(), nil)
	if err != nil {
		return err
	}
	return nil
}

// CreateMessage creates an object which satisfies the types.Message interface. It populates gas limit, price, nonce,
// and other fields automatically, and sets fee/tip caps such that no base fee is charged (for testing).
func (t *TestNode) CreateMessage(from common.Address, to *common.Address, value *big.Int, data []byte) *chainTypes.CallMessage {
	// Obtain our message parameters
	nonce := t.state.GetNonce(from)
	gasLimit := t.dummyChain.GasLimit()
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

// evmOpBlockHash represents a function used for a vm.BlockContext to facilitate BLOCKHASH instruction operations in the
// EVM. This method is used as a hook to fetch block hashes from our TestNode rather than the dummy core.BlockChain we
// use to create vm.BlockContext objects. A user supplies a block number for which they wish to obtain a hash. If the
// number refers to the currently executing block number or a future one, a zero hash is returned. If a block number
// requested is over 256 blocks in the past from the current executing block number, a zero hash is returned. Otherwise,
// the block hash is returned.
func (t *TestNode) evmOpBlockHash(n uint64) common.Hash {
	// If we're asking for our current block or newer, we can't provide that information at that time, so we return
	// a zero hash, per the Ethereum spec.
	// Note: We add 1 to block number here because if we're executing this, it means we're constructing a new block
	// or doing a new call on top of the last recorded block number.
	currentBlockNumber := t.BlockNumber() + 1
	if currentBlockNumber <= n {
		return common.Hash{}
	}

	// Calculate our distance from our last block and ensure it does not index more than 256 items (255 indexes away
	// from the last block), as the BLOCKHASH opcode cannot obtain history further than this.
	distanceFromLastBlock := currentBlockNumber - n - 1
	if distanceFromLastBlock > 255 {
		return common.Hash{}
	}

	// Obtain our requested block hash and return it.
	requestedBlockHash, err := t.BlockHashFromBlockNumber(n)
	if err != nil {
		return common.Hash{}
	} else {
		return requestedBlockHash
	}
}

// CallContract performs a message call over the current test chain  state and obtains a core.ExecutionResult.
// This is similar to the CallContract method provided by Ethereum for use in calling pure/view functions.
func (t *TestNode) CallContract(msg *chainTypes.CallMessage) (*core.ExecutionResult, error) {
	// Obtain our state snapshot (note: this is different from the test node snapshot)
	snapshot := t.state.Snapshot()

	// Set infinite balance to the fake caller account
	from := t.state.GetOrNewStateObject(msg.From())
	from.SetBalance(math.MaxBig256)

	// Create our transaction and block contexts for the vm
	txContext := core.NewEVMTxContext(msg)
	blockContext := core.NewEVMBlockContext(t.BlockHeader(), t.dummyChain, nil)

	// Hook the method used for the BLOCKHASH opcode to get previous block hashes so the dummyChain is not used in the
	// original implementation, as it is not maintained for the true chain state.
	blockContext.GetHash = t.evmOpBlockHash

	// Create our EVM instance
	evm := vm.NewEVM(blockContext, txContext, t.state, t.dummyChain.Config(), vm.Config{NoBaseFee: true})

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
func (t *TestNode) SendMessages(messages ...*chainTypes.CallMessage) (*chainTypes.Block, error) {
	// Set up some parameters used to construct our test block
	config := t.dummyChain.Config()

	//parentBlockNumber := big.NewInt(0).Sub(blockNumber, big.NewInt(1))
	parentBlockHash := t.BlockHeader().Hash()

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
		Coinbase:    t.BlockHeader().Coinbase, // reusing same coinbase throughout the chain
		Root:        types.EmptyRootHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      big.NewInt(int64(t.BlockNumber()) + 1),
		GasLimit:    t.dummyChain.GasLimit(), // reusing same gas limit throughout the chain
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
		blockContext := core.NewEVMBlockContext(header, t.dummyChain, &header.Coinbase)

		// Hook the method used for the BLOCKHASH opcode to get previous block hashes so the dummyChain is not used in the
		// original implementation, as it is not maintained for the true chain state.
		blockContext.GetHash = t.evmOpBlockHash

		// Create our EVM instance.
		evm := vm.NewEVM(blockContext, vm.TxContext{}, t.state, config, *t.vmConfig)

		// Apply our transaction
		var usedGas uint64
		gasPool := new(core.GasPool).AddGas(math.MaxUint64) // TODO: Verify this is sensible
		receipt, err := vendored.EVMApplyTransaction(messages[i].ToEVMMessage(), config, &header.Coinbase, gasPool, t.state, header.Number, blockHash, tx, &usedGas, evm)
		if err != nil {
			return nil, fmt.Errorf("test chain state write error: %v", err)
		}

		// Add our receipt to our list
		receipts = append(receipts, receipt)
	}

	// Write state changes to db
	root, err := t.state.Commit(config.IsEIP158(header.Number))
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
	t.chain = append(t.chain, block)
	return block, nil
}

// DeployContract is a helper method used to deploy a given types.CompiledContract to the current instance of the
// test node, using the address provided as the deployer. Returns the address of the deployed contract if successful,
// otherwise returns an error.
func (t *TestNode) DeployContract(contract *compilationTypes.CompiledContract, deployer common.Address) (common.Address, error) {
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
