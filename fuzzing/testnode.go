package fuzzing

import (
	"crypto/ecdsa"
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
	"github.com/ethereum/go-ethereum/crypto"
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
	chain   *core.BlockChain
	kvstore *memorydb.Database
	db      ethdb.Database
	signer  *coreTypes.HomesteadSigner

	pendingBlock *coreTypes.Block
	pendingState *state.StateDB

	tracer   *tracing.FuzzerTracer
	vmConfig *vm.Config
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
		Debug:  true,
		Tracer: tracer,
	}

	// Create a new blockchain
	chain, err := core.NewBlockChain(db, nil, chainConfig, ethash.NewFullFaker(), *vmConfig, nil, nil)
	if err != nil {
		return nil, err
	}

	// Obtain our current state
	pendingState, err := chain.State()
	if err != nil {
		return nil, err
	}

	// Create our instance
	g := &testNode{
		chain:        chain,
		kvstore:      kvstore,
		db:           db,
		signer:       new(coreTypes.HomesteadSigner),
		pendingBlock: chain.CurrentBlock(),
		pendingState: pendingState,
		tracer:       tracer,
		vmConfig:     vmConfig,
	}

	return g, nil
}

func (t *testNode) MemoryDatabaseEntryCount() int {
	return t.kvstore.Len()
}

func (t *testNode) Stop() {
	// Stop the underlying chain's update loop
	t.chain.Stop()
}

func (t *testNode) SendTransaction(tx *coreTypes.Transaction) (*coreTypes.Block, *coreTypes.Receipts, error) {
	// Create our blocks.
	blocks, receipts := vendored.GenerateChain(t.chain.Config(), t.pendingBlock, t.chain.Engine(), t.db, 1, func(i int, b *vendored.BlockGen) {
		// Set the coinbase and difficulty
		b.SetCoinbase(common.Address{1})
		b.SetDifficulty(big.NewInt(1))

		// Add the transaction.
		b.AddTx(tx, *t.vmConfig)
	})

	// Obtain our current chain's state, so that we can use its database to obtain the pending state.
	stateDB, err := t.chain.State()
	if err != nil {
		return nil, nil, err
	}

	// Set our pending block and state.
	t.pendingBlock = blocks[0]
	t.pendingState, err = state.New(t.pendingBlock.Root(), stateDB.Database(), nil)
	if err != nil {
		return nil, nil, err
	}
	return blocks[0], &receipts[0], nil
}

func (t *testNode) Commit() {
	// Insert our pending block into the chain.
	_, err := t.chain.InsertChain([]*coreTypes.Block{t.pendingBlock})
	if err != nil {
		panic("failed to insert pending block into chain.")
	}
}

func (t *testNode) RevertUncommittedChanges() error {
	// Reset our pending block to our chain's current block
	t.pendingBlock = t.chain.CurrentBlock()

	// Reset our pending state to our chain's current state.
	var err error
	t.pendingState, err = t.chain.State()
	if err != nil {
		return err
	}
	return nil
}

func (t *testNode) CallContract(call ethereum.CallMsg) (*core.ExecutionResult, error) {
	// Obtain our snapshot
	snapshot := t.pendingState.Snapshot()

	// Call our contract
	res, err := t.callContract(call, t.pendingBlock, t.pendingState)

	// Revert to our snapshot to undo any changes.
	t.pendingState.RevertToSnapshot(snapshot)

	return res, err
}

// Copied from go-ethereum/accounts/abi/bind/backends/simulated.go
func (t *testNode) callContract(call ethereum.CallMsg, block *coreTypes.Block, stateDB *state.StateDB) (*core.ExecutionResult, error) {
	// Gas prices post 1559 need to be initialized
	if call.GasPrice != nil && (call.GasFeeCap != nil || call.GasTipCap != nil) {
		return nil, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	head := t.chain.CurrentHeader()
	if !t.chain.Config().IsLondon(head.Number) {
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
	msg := callMsg{call}

	txContext := core.NewEVMTxContext(msg)
	evmContext := core.NewEVMBlockContext(block.Header(), t.chain, nil)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmEnv := vm.NewEVM(evmContext, txContext, stateDB, t.chain.Config(), vm.Config{NoBaseFee: true})
	gasPool := new(core.GasPool).AddGas(math.MaxUint64)

	return core.NewStateTransition(vmEnv, msg, gasPool).TransitionDb()
}

func (t *testNode) SignAndSendLegacyTransaction(tx *coreTypes.LegacyTx, signerKey *ecdsa.PrivateKey, applyFixups bool) (*coreTypes.Block, *coreTypes.Receipts, error) {
	// Apply fixups related to gas/nonce
	if applyFixups {
		accountAddress := crypto.PubkeyToAddress(signerKey.PublicKey)
		tx.Nonce = t.pendingState.GetNonce(accountAddress)
		tx.GasPrice = big.NewInt(params.InitialBaseFee)
		tx.Gas = t.pendingBlock.GasLimit()
	}

	// Sign the transaction
	signedTx, err := coreTypes.SignNewTx(signerKey, t.signer, tx)
	if err != nil {
		return nil, nil, fmt.Errorf("could not sign tx due to an error when signing: %s", err.Error())
	}

	// Send our deployment transaction
	return t.SendTransaction(signedTx)
}

func (t *testNode) DeployContract(contract types.CompiledContract, deployerKey *ecdsa.PrivateKey) (common.Address, error) {
	// Obtain the byte code as a byte array
	b, err := hex.DecodeString(strings.TrimPrefix(contract.InitBytecode, "0x"))
	if err != nil {
		panic("could not convert compiled contract bytecode from hex string to byte code")
	}

	// Constructor args don't need ABI encoding and appending to the end of the bytecode since there are none for these
	// contracts.

	// Create a transaction to represent our contract deployment.
	// NOTE: We don't fill out nonce/gas as SignAndSendLegacyTransaction will apply fixups below.
	tx := &coreTypes.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(0),
		Gas:      0,
		To:       nil,
		Value:    big.NewInt(0),
		Data:     b,
	}

	// Send our deployment transaction
	_, receipts, err := t.SignAndSendLegacyTransaction(tx, deployerKey, true)
	if err != nil {
		return common.Address{0}, err
	}

	// Ensure our transaction succeeded
	if (*receipts)[0].Status != coreTypes.ReceiptStatusSuccessful {
		return common.Address{0}, fmt.Errorf("contract deployment tx returned a failed status")
	}

	// Commit our state immediately so our pending state can access
	t.Commit()

	// Return the address for the deployed contract.
	return (*receipts)[0].ContractAddress, nil
}

// callMsg implements core.Message to allow passing it as a transaction simulator.
type callMsg struct {
	ethereum.CallMsg
}

func (m callMsg) From() common.Address             { return m.CallMsg.From }
func (m callMsg) Nonce() uint64                    { return 0 }
func (m callMsg) IsFake() bool                     { return true }
func (m callMsg) To() *common.Address              { return m.CallMsg.To }
func (m callMsg) GasPrice() *big.Int               { return m.CallMsg.GasPrice }
func (m callMsg) GasFeeCap() *big.Int              { return m.CallMsg.GasFeeCap }
func (m callMsg) GasTipCap() *big.Int              { return m.CallMsg.GasTipCap }
func (m callMsg) Gas() uint64                      { return m.CallMsg.Gas }
func (m callMsg) Value() *big.Int                  { return m.CallMsg.Value }
func (m callMsg) Data() []byte                     { return m.CallMsg.Data }
func (m callMsg) AccessList() coreTypes.AccessList { return m.CallMsg.AccessList }
