package fuzzing

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/fuzzing/tracing"
	"github.com/trailofbits/medusa/fuzzing/vendored"
	"math/big"
	"strings"
)

type testNode struct {
	// chain represents the blocks post-genesis that were generated after sending messages.
	chain []TestNodeBlock

	// dummyChain is core.BlockChain instance which is used to construct the genesis state. After such point it is
	// only used for reference to fulfill the needs of some methods.
	dummyChain *core.BlockChain
	kvstore    *memorydb.Database
	db         ethdb.Database
	signer     *coreTypes.HomesteadSigner

	state    *state.StateDB
	snapshot int

	tracer   *tracing.FuzzerTracer
	vmConfig *vm.Config
}

type TestNodeBlock struct {
	header  *coreTypes.Header
	message core.Message
	receipt *coreTypes.Receipt
}

func newTestNode(genesisAlloc core.GenesisAlloc) (*testNode, error) {
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
	state, err := dummyChain.State()
	if err != nil {
		return nil, err
	}

	// Create our instance
	g := &testNode{
		chain:      make([]TestNodeBlock, 0),
		dummyChain: dummyChain,
		kvstore:    kvstore,
		db:         db,
		signer:     new(coreTypes.HomesteadSigner),
		state:      state,
		tracer:     tracer,
		vmConfig:   vmConfig,
	}

	return g, nil
}

func (t *testNode) MemoryDatabaseEntryCount() int {
	return t.kvstore.Len()
}

func (t *testNode) Stop() {
	// Stop the underlying chain's update loop
	t.dummyChain.Stop()
}

func (t *testNode) Snapshot() {
	// Save our snapshot (block height)
	t.snapshot = len(t.chain)
}

func (t *testNode) Revert() error {
	var err error

	// Adjust our chain length to match our snapshot
	t.chain = t.chain[:t.snapshot]

	// Reload our state from our database
	t.state, err = state.New(t.GetBlockHeader().Root, t.state.Database(), nil)
	if err != nil {
		return err
	}
	return nil
}

func (t *testNode) CallContract(call ethereum.CallMsg) (*core.ExecutionResult, error) {
	// Obtain our snapshot
	snapshot := t.state.Snapshot()

	// Call our contract
	res, err := t.callContract(call, t.GetBlockHeader(), t.state)

	// Revert to our snapshot to undo any changes.
	t.state.RevertToSnapshot(snapshot)

	return res, err
}

// Copied from go-ethereum/accounts/abi/bind/backends/simulated.go
func (t *testNode) callContract(call ethereum.CallMsg, header *coreTypes.Header, stateDB *state.StateDB) (*core.ExecutionResult, error) {
	// Gas prices post 1559 need to be initialized
	if call.GasPrice != nil && (call.GasFeeCap != nil || call.GasTipCap != nil) {
		return nil, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	head := t.dummyChain.CurrentHeader()
	if !t.dummyChain.Config().IsLondon(head.Number) {
		// If there's no basefee, then it must be a non-1559 execution
		if call.GasPrice == nil {
			call.GasPrice = new(big.Int)
		}
		call.GasFeeCap, call.GasTipCap = call.GasPrice, call.GasPrice
	} else {
		// A basefee is provided, necessitating 1559-type execution
		if call.GasPrice != nil {
			// User specified the legacy gas field, convert to 1559 gas typing
			call.GasFeeCap, call.GasTipCap = call.GasPrice, call.GasPrice
		} else {
			// User specified 1559 gas feilds (or none), use those
			if call.GasFeeCap == nil {
				call.GasFeeCap = new(big.Int)
			}
			if call.GasTipCap == nil {
				call.GasTipCap = new(big.Int)
			}
			// Backfill the legacy gasPrice for EVM execution, unless we're all zeroes
			call.GasPrice = new(big.Int)
			if call.GasFeeCap.BitLen() > 0 || call.GasTipCap.BitLen() > 0 {
				call.GasPrice = math.BigMin(new(big.Int).Add(call.GasTipCap, head.BaseFee), call.GasFeeCap)
			}
		}
	}
	// Ensure message is initialized properly.
	if call.Gas == 0 {
		call.Gas = 50000000
	}
	if call.Value == nil {
		call.Value = new(big.Int)
	}
	// Set infinite balance to the fake caller account.
	from := stateDB.GetOrNewStateObject(call.From)
	from.SetBalance(math.MaxBig256)

	// Execute the call.
	msg := t.createMessage(call.From, call.To, call.Value, call.Data)

	txContext := core.NewEVMTxContext(msg)
	evmContext := core.NewEVMBlockContext(header, t.dummyChain, nil)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmEnv := vm.NewEVM(evmContext, txContext, stateDB, t.dummyChain.Config(), vm.Config{NoBaseFee: true})
	gasPool := new(core.GasPool).AddGas(math.MaxUint64)

	return core.NewStateTransition(vmEnv, msg, gasPool).TransitionDb()
}

func messageToTransaction(msg coreTypes.Message) *coreTypes.Transaction {
	// TODO: This probably might not hash due to invalid signatures.
	return coreTypes.NewTx(&coreTypes.LegacyTx{
		Nonce:    msg.Nonce(),
		GasPrice: msg.GasPrice(),
		Gas:      msg.Gas(),
		To:       msg.To(),
		Value:    msg.Value(),
		Data:     msg.Data(),
	})
}

func (t *testNode) createMessage(from common.Address, to *common.Address, value *big.Int, data []byte) coreTypes.Message {
	nonce := t.state.GetNonce(from)
	msg := coreTypes.NewMessage(from, to, nonce, value, t.dummyChain.GasLimit(), big.NewInt(1), big.NewInt(0), big.NewInt(0), data, nil, true)
	return msg
}

func (t *testNode) GetBlockNumber() int64 {
	// Our chain length is genesis block + test node blocks
	return int64(len(t.chain))
}

func (t *testNode) GetBlockHeader() *coreTypes.Header {
	// If we have any blocks on the test chain, return the latest
	if len(t.chain) > 0 {
		return t.chain[len(t.chain)-1].header
	}

	// Otherwise return the genesis header
	return t.dummyChain.CurrentHeader()
}

func (t *testNode) GetBlockHashFromBlockNumber(blockNumber uint64) common.Hash {
	// If this is the genesis block, return that hash from the chain.
	if len(t.chain) == 0 {
		return t.dummyChain.CurrentBlock().Hash()
	} else {
		// Otherwise any block but the genesis will be given an empty hash with the block number set at the end
		// for simplicity/uniqueness/computational speed..
		return t.chain[len(t.chain)-1].header.Hash()
	}
}

func (t *testNode) SendMessage(msg coreTypes.Message) *TestNodeBlock {
	blockNumber := big.NewInt(t.GetBlockNumber() + 1)
	blockHash := t.GetBlockHashFromBlockNumber(blockNumber.Uint64())
	blockTimestamp := uint64(t.GetBlockNumber() + 1) // TODO:
	coinbase := common.Address{}                     // TODO: Reference the previous coinbase.
	config := t.dummyChain.Config()
	gasPool := new(core.GasPool).AddGas(math.MaxUint64) // TODO: Verify this is safe, this is a lot of gas!
	var usedGas uint64

	parentBlockNumber := big.NewInt(0).Sub(blockNumber, big.NewInt(1))
	parentBlockHash := t.GetBlockHashFromBlockNumber(parentBlockNumber.Uint64()) // TODO: Figure this out

	// Use the default set gas limit from the dummy chain
	gasLimit := t.dummyChain.GasLimit()

	// Create a block header for this block:
	// - Root hashes are not populated on first run.
	// - State root hash is populated later in this method.
	// - Bloom is not populated on first run.
	// - TODO: Difficulty is not proven to be safe
	// - GasUsed is not populated on first run.
	// - Mix digest is only useful for randomness, so we just use block hash.
	// - TODO: Figure out appropriate params for BaseFee
	header := &coreTypes.Header{
		ParentHash:  parentBlockHash,
		UncleHash:   coreTypes.EmptyUncleHash,
		Coinbase:    coinbase,
		Root:        coreTypes.EmptyRootHash,
		TxHash:      coreTypes.EmptyRootHash,
		ReceiptHash: coreTypes.EmptyRootHash,
		Bloom:       coreTypes.Bloom{},
		Difficulty:  common.Big0,
		Number:      blockNumber,
		GasLimit:    gasLimit,
		GasUsed:     0,
		Time:        blockTimestamp,
		Extra:       []byte{},
		MixDigest:   blockHash,
		Nonce:       coreTypes.BlockNonce{},
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}

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

func (t *testNode) DeployContract(contract types.CompiledContract, deployer common.Address) (common.Address, error) {
	// Obtain the byte code as a byte array
	b, err := hex.DecodeString(strings.TrimPrefix(contract.InitBytecode, "0x"))
	if err != nil {
		panic("could not convert compiled contract bytecode from hex string to byte code")
	}

	// Constructor args don't need ABI encoding and appending to the end of the bytecode since there are none for these
	// contracts.

	// Create a transaction to represent our contract deployment.
	// NOTE: We don't fill out nonce/gas as SignAndSendLegacyTransaction will apply fixups below.
	amount := big.NewInt(0)
	msg := t.createMessage(deployer, nil, amount, b)

	// Send our deployment transaction
	block := t.SendMessage(msg)

	// Ensure our transaction succeeded
	if block.receipt.Status != coreTypes.ReceiptStatusSuccessful {
		return common.Address{0}, fmt.Errorf("contract deployment tx returned a failed status")
	}

	// Return the address for the deployed contract.
	return block.receipt.ContractAddress, nil
}
