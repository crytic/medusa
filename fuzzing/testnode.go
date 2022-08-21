package fuzzing

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
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/fuzzing/tracing"
	fuzzingTypes "github.com/trailofbits/medusa/fuzzing/types"
	"github.com/trailofbits/medusa/fuzzing/vendored"
	"math/big"
	"strings"
)

// TestNode represents a simulated Ethereum backend used for testing
type TestNode struct {
	// chain represents the blocks post-genesis that were generated after sending messages.
	chain []TestNodeBlock

	// dummyChain is core.BlockChain instance which is used to construct the genesis state. After such point it is
	// only used for reference to fulfill the needs of some methods.
	dummyChain *core.BlockChain
	kvstore    *memorydb.Database
	db         ethdb.Database
	signer     *types.HomesteadSigner

	state    *state.StateDB
	snapshot int

	tracer   *tracing.FuzzerTracer
	vmConfig *vm.Config
}

// TestNodeBlock represents a block or update within a TestNode
type TestNodeBlock struct {
	header    *types.Header
	message   core.Message
	receipt   *types.Receipt
	blockHash common.Hash
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

	// Create a VM config that traces execution, so we can establish a coverage map
	tracer := tracing.NewFuzzerTracer(true)
	vmConfig := &vm.Config{
		Debug:     true,
		Tracer:    tracer,
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
		chain:      make([]TestNodeBlock, 0),
		dummyChain: dummyChain,
		kvstore:    kvstore,
		db:         db,
		signer:     new(types.HomesteadSigner),
		state:      stateDb,
		tracer:     tracer,
		vmConfig:   vmConfig,
	}

	return g, nil
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

// CallContract performs a message call over the current test chain  state and obtains a core.ExecutionResult.
// This is similar to the CallContract method provided by Ethereum for use in calling pure/view functions.
func (t *TestNode) CallContract(msg *fuzzingTypes.CallMessage) (*core.ExecutionResult, error) {
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

// CreateMessage creates an object which satisfies the types.Message interface. It populates gas limit, price, nonce,
// and other fields automatically, and sets fee/tip caps such that no base fee is charged (for testing).
func (t *TestNode) CreateMessage(from common.Address, to *common.Address, value *big.Int, data []byte) *fuzzingTypes.CallMessage {
	// Obtain our message parameters
	nonce := t.state.GetNonce(from)
	gasLimit := t.dummyChain.GasLimit()
	gasPrice := big.NewInt(1)

	// Setting fee and tip cap to zero alongside the NoBaseFee for the vm.Config will bypass base fee validation.
	gasFeeCap := big.NewInt(0)
	gasTipCap := big.NewInt(0)

	// Construct and return a new message from our given parameters.
	return fuzzingTypes.NewCallMessage(from, to, nonce, value, gasLimit, gasPrice, gasFeeCap, gasTipCap, data)
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
		return t.chain[len(t.chain)-1].header
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
		blockHash := t.chain[blockNumber-1].blockHash
		return blockHash, nil
	}
}

// evmOpBlockHash represents a function used for a vm.BlockContext to facilitate BLOCKHASH instruction operations in the
// EVM. This method is used as a hook to fetch block hashes from our TestNode rather than the dummy chain we use to
// create vm.BlockContext objects. A user supplies a block number for which they wish to obtain a hash. If the number
// refers to the currently executing block number or a future one, a zero hash is returned. If a block number requested
// is over 256 blocks in the past from the current executing block number, a zero hash is returned. Otherwise the block
// hash is returned.
func (t *TestNode) evmOpBlockHash(n uint64) common.Hash {
	// If we're asking for our current block or newer, we can't provide that information at that time so we return
	// a zero hash, per the Ethereum spec.
	// Note: We add 1 to block number here because if we're executing this, it means we're constructing a new block
	// or doing a new call on top of the last recorded block number.
	currentBlockNumber := t.BlockNumber() + 1
	if currentBlockNumber <= n {
		return common.Hash{}
	}

	// Calculate our distance from our last block and ensure it does not index more than 256 items (255 indexes away
	// from zero), as the BLOCKHASH opcode cannot obtain history further than this.
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

// SendMessage is similar to Ethereum's SendTransaction, it takes a message (internal tx) and applies a state update
// with it, as if a transaction were just received. Returns the TestNodeBlock representing the result of the state
// transition.
func (t *TestNode) SendMessage(msg *fuzzingTypes.CallMessage) *TestNodeBlock {
	// Set up some parameters used to construct our test block
	blockNumber := big.NewInt(int64(t.BlockNumber()) + 1)
	blockTimestamp := uint64(t.BlockNumber() + 1) // TODO: Determine proper timestamp advance logic
	coinbase := t.BlockHeader().Coinbase
	config := t.dummyChain.Config()
	gasPool := new(core.GasPool).AddGas(math.MaxUint64) // TODO: Verify this is sensible
	var usedGas uint64

	// Use the default set gas limit from the dummy chain
	gasLimit := t.dummyChain.GasLimit()

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
		Coinbase:    coinbase,
		Root:        types.EmptyRootHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      blockNumber,
		GasLimit:    gasLimit,
		GasUsed:     0,
		Time:        blockTimestamp,
		Extra:       []byte{},
		MixDigest:   parentBlockHash,
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}

	// Calculate our block hash for this block
	blockHash := header.Hash()

	// Create a tx from our msg, for hashing/receipt purposes
	tx := messageToTransaction(msg)

	// Create a new context to be used in the EVM environment
	blockContext := core.NewEVMBlockContext(header, t.dummyChain, &coinbase)

	// Hook the method used for the BLOCKHASH opcode to get previous block hashes so the dummyChain is not used in the
	// original implementation, as it is not maintained for the true chain state.
	blockContext.GetHash = t.evmOpBlockHash

	// Create our EVM instance.
	evm := vm.NewEVM(blockContext, vm.TxContext{}, t.state, config, *t.vmConfig)

	// Apply our transaction
	receipt, err := vendored.EVMApplyTransaction(msg.ToEVMMessage(), config, &coinbase, gasPool, t.state, blockNumber, blockHash, tx, &usedGas, evm)
	if err != nil {
		panic(fmt.Sprintf("state write error: %v", err))
	}

	// Write state changes to db
	root, err := t.state.Commit(config.IsEIP158(header.Number))
	if err != nil {
		panic(fmt.Sprintf("state write error: %v", err))
	}
	if err := t.state.Database().TrieDB().Commit(root, false, nil); err != nil {
		panic(fmt.Sprintf("trie write error: %v", err))
	}

	// Update the header's state root hash
	// Note: You could also retrieve the root with state.IntermediateRoot(config.IsEIP158(parentBlockNumber))
	header.Root = root

	// Create a new block for our test node
	block := TestNodeBlock{
		header:    header,
		message:   msg,
		receipt:   receipt,
		blockHash: blockHash,
	}

	// Append it to our chain
	t.chain = append(t.chain, block)
	return &block
}

// DeployContract is a helper method used to deploy a given types.CompiledContract to the current instance of the
// test node, using the address provided as the deployer. Returns the address of the deployed contract if successful,
// otherwise returns an error.
func (t *TestNode) DeployContract(contract compilationTypes.CompiledContract, deployer common.Address) (common.Address, error) {
	// Obtain the byte code as a byte array
	b, err := hex.DecodeString(strings.TrimPrefix(contract.InitBytecode, "0x"))
	if err != nil {
		panic("could not convert compiled contract bytecode from hex string to byte code")
	}

	// Constructor args don't need ABI encoding and appending to the end of the bytecode since there are none for these
	// contracts.

	// Create a message to represent our contract deployment.
	value := big.NewInt(0)
	msg := t.CreateMessage(deployer, nil, value, b)

	// Send our deployment transaction
	block := t.SendMessage(msg)

	// Ensure our transaction succeeded
	if block.receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, fmt.Errorf("contract deployment tx returned a failed status")
	}

	// Return the address for the deployed contract.
	return block.receipt.ContractAddress, nil
}
