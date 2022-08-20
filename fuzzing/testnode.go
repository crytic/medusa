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
	header  *types.Header
	message core.Message
	receipt *types.Receipt
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
func (t *TestNode) CallContract(call types.Message) (*core.ExecutionResult, error) {
	// Obtain our state snapshot (note: this is different from the test node snapshot)
	snapshot := t.state.Snapshot()

	// Set infinite balance to the fake caller account
	from := t.state.GetOrNewStateObject(call.From())
	from.SetBalance(math.MaxBig256)

	// Execute the call.
	msg := t.CreateMessage(call.From(), call.To(), call.Value(), call.Data())

	// Create our transaction and block contexts for the vm
	txContext := core.NewEVMTxContext(msg)
	evmContext := core.NewEVMBlockContext(t.BlockHeader(), t.dummyChain, nil)

	// Create our EVM instance
	evm := vm.NewEVM(evmContext, txContext, t.state, t.dummyChain.Config(), vm.Config{NoBaseFee: true})

	// Fund the gas pool for execution appropriately
	gasPool := new(core.GasPool).AddGas(math.MaxUint64)

	// Perform our state transition to obtain the result.
	res, err := core.NewStateTransition(evm, msg, gasPool).TransitionDb()

	// Revert to our state snapshot to undo any changes.
	t.state.RevertToSnapshot(snapshot)

	return res, err
}

// messageToTransaction derived a types.Transaction from a types.Message.
func messageToTransaction(msg types.Message) *types.Transaction {
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
func (t *TestNode) CreateMessage(from common.Address, to *common.Address, value *big.Int, data []byte) types.Message {
	// Obtain our message parameters
	nonce := t.state.GetNonce(from)
	gasLimit := t.dummyChain.GasLimit()
	gasPrice := big.NewInt(1)

	// Setting fee and tip cap to zero alongside the NoBaseFee for the vm.Config will bypass base fee validation.
	gasFeeCap := big.NewInt(0)
	gasTipCap := big.NewInt(0)

	// Construct and return a new message from our given parameters.
	return types.NewMessage(from, to, nonce, value, gasLimit, gasPrice, gasFeeCap, gasTipCap, data, nil, true)
}

// BlockNumber returns the test chain head's block number, where zero is the genesis block.
func (t *TestNode) BlockNumber() int64 {
	// Our chain length is genesis block + test node blocks
	return int64(len(t.chain))
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

// SendMessage is similar to Ethereum's SendTransaction, it takes a message (internal tx) and applies a state update
// with it, as if a transaction were just received. Returns the TestNodeBlock representing the result of the state
// transition.
func (t *TestNode) SendMessage(msg types.Message) *TestNodeBlock {
	// Set up some parameters used to construct our test block
	blockNumber := big.NewInt(t.BlockNumber() + 1)
	blockTimestamp := uint64(t.BlockNumber() + 1) // TODO:
	coinbase := t.BlockHeader().Coinbase
	config := t.dummyChain.Config()
	gasPool := new(core.GasPool).AddGas(math.MaxUint64) // TODO: Verify this is safe, this is a lot of gas!
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
	blockHash := header.Hash()

	// Create a tx from our msg, for hashing/receipt purposes
	tx := messageToTransaction(msg)

	// Create a new context to be used in the EVM environment
	blockContext := core.NewEVMBlockContext(header, t.dummyChain, &coinbase)

	// Hook the method used for the BLOCKHASH opcode to get previous block hashes so the dummyChain is not used.
	blockContext.GetHash = func(num uint64) common.Hash {
		// TODO: Implement getting header hash.
		return common.Hash{}
	}

	// Create our EVM instance.
	evm := vm.NewEVM(blockContext, vm.TxContext{}, t.state, config, *t.vmConfig)

	// Apply our transaction
	receipt, err := vendored.EVMApplyTransaction(msg, config, &coinbase, gasPool, t.state, blockNumber, blockHash, tx, &usedGas, evm)
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
	header.Root = root // TODO: t.state.IntermediateRoot(config.IsEIP158(parentBlockNumber))

	// Create a new block for our test node
	block := TestNodeBlock{
		header:  header,
		message: msg,
		receipt: receipt,
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
