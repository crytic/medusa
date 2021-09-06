package fuzzer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	core "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"medusa/compilation/types"
	"strings"
)

type TestNode struct {
	chain *core.BlockChain
	kvstore *memorydb.Database
	db ethdb.Database
	signer *coreTypes.HomesteadSigner

	pendingBlock *coreTypes.Block
	pendingState *state.StateDB
}

func NewTestNode(genesisAlloc core.GenesisAlloc) (*TestNode, error) {
	// Define our chain configuration
	chainConfig := params.TestChainConfig

	// Create an in-memory database
	kvstore := memorydb.New()
	db := rawdb.NewDatabase(kvstore)

	// Create our genesis block
	genesisDefinition := &core.Genesis{
		Config: chainConfig,
		Alloc: genesisAlloc,
		ExtraData: []byte {
			0x74, 0x72, 0x61, 0x73, 0x68, 0x20, 0x70, 0x61, 0x6E, 0x64, 0x61, 0x73, 0x20, 0x6E, 0x65, 0x65,
			0x64, 0x20, 0x6C, 0x6F, 0x76, 0x65, 0x20, 0x74, 0x6F, 0x6F, 0x2E, 0x20, 0x2D, 0x58, 0x39,
		},
	}

	// Commit our genesis definition to get a block.
	genesisDefinition.MustCommit(db)

	// Create a new blockchain
	// TODO: Determine if we should use a cache configs
	chain, err := core.NewBlockChain(db, nil, chainConfig, ethash.NewFullFaker(), vm.Config{}, nil, nil)
	if err != nil {
		return nil, err
	}

	// Obtain our current state
	pendingState, err := chain.State()
	if err != nil {
		return nil, err
	}

	// Create our instance
	g := &TestNode{
		chain:        chain,
		kvstore:      kvstore,
		db:           db,
		signer:       new(coreTypes.HomesteadSigner),
		pendingBlock: chain.CurrentBlock(),
		pendingState: pendingState,
	}

	return g, nil
}

func (t *TestNode) GetMemoryUsage() int {
	return t.kvstore.Len()
}

func (t *TestNode) Stop() {
	// Stop the underlying chain's update loop
	t.chain.Stop()
}

func (t *TestNode) SendTransaction(tx *coreTypes.Transaction) (*coreTypes.Block, *coreTypes.Receipts, error) {
	// Create our blocks.
	blocks, receipts := core.GenerateChain(t.chain.Config(), t.pendingBlock, t.chain.Engine(), t.db, 1, func(i int, b *core.BlockGen) {
		// Set the coinbase and difficulty
		b.SetCoinbase(common.Address{1})
		b.SetDifficulty(big.NewInt(1))

		// Add the transaction.
		b.AddTx(tx)
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

func (t *TestNode) Commit() {
	// Insert our pending block into the chain.
	_, err := t.chain.InsertChain([]*coreTypes.Block{t.pendingBlock})
	if err != nil {
		panic("failed to insert pending block into chain.")
	}
}

func (t *TestNode) CallContract(call ethereum.CallMsg) (*core.ExecutionResult, error) {
	// Obtain our snapshot
	snapshot := t.pendingState.Snapshot()

	// Call our contract
	res, err := t.callContract(call, t.pendingBlock, t.pendingState)

	// Revert to our snapshot to undo any changes.
	t.pendingState.RevertToSnapshot(snapshot)

	return res, err
}

// Copied from go-ethereum/accounts/abi/bind/backends/simulated.go
func (t *TestNode) callContract(call ethereum.CallMsg, block *coreTypes.Block, stateDB *state.StateDB) (*core.ExecutionResult, error) {
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

func (t *TestNode) deployContract(contract types.CompiledContract, deployer fuzzerAccount) (common.Address, error) {
	// Obtain the byte code as a byte array
	b, err := hex.DecodeString(strings.TrimPrefix(contract.InitBytecode, "0x"))
	if err != nil {
		panic("could not convert compiled contract bytecode from hex string to byte code")
	}

	// Constructor args don't need ABI encoding and appending to the end of the bytecode since there are none for these
	// contracts.

	// Create a transaction to represent our contract deployment.
	tx := &coreTypes.LegacyTx{
		Nonce: t.pendingState.GetNonce(deployer.address),
		GasPrice: big.NewInt(params.InitialBaseFee),
		Gas: t.pendingBlock.GasLimit(),
		To: nil,
		Value: big.NewInt(0),
		Data: b,
	}

	// Sign the transaction
	signedTx, err := coreTypes.SignNewTx(deployer.key, t.signer, tx)
	if err != nil {
		return common.Address{0}, fmt.Errorf("could not sign tx to deploy contract due to an error when signing: %s", err.Error())
	}

	// Send our deployment transaction
	_, receipts, err := t.SendTransaction(signedTx)
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

func (m callMsg) From() common.Address         { return m.CallMsg.From }
func (m callMsg) Nonce() uint64                { return 0 }
func (m callMsg) IsFake() bool                 { return true }
func (m callMsg) To() *common.Address          { return m.CallMsg.To }
func (m callMsg) GasPrice() *big.Int           { return m.CallMsg.GasPrice }
func (m callMsg) GasFeeCap() *big.Int          { return m.CallMsg.GasFeeCap }
func (m callMsg) GasTipCap() *big.Int          { return m.CallMsg.GasTipCap }
func (m callMsg) Gas() uint64                  { return m.CallMsg.Gas }
func (m callMsg) Value() *big.Int              { return m.CallMsg.Value }
func (m callMsg) Data() []byte                 { return m.CallMsg.Data }
func (m callMsg) AccessList() coreTypes.AccessList { return m.CallMsg.AccessList }
