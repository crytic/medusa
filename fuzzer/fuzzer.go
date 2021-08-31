package fuzzer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"medusa/compilation"
	"medusa/compilation/types"
	"medusa/configs"
	"strings"
)

type FuzzerAccount struct {
	key *ecdsa.PrivateKey
	address common.Address
}

type Fuzzer struct {
	config   configs.ProjectConfig
	accounts []FuzzerAccount
}

func NewFuzzer(config configs.ProjectConfig) (*Fuzzer, error) {
	// Create our accounts based on our configs
	accounts := make([]FuzzerAccount, 0)

	// Generate new accounts as requested.
	for i := 0; i < config.Accounts.Generate; i++ {
		// Generate a new key
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}

		// Add it to our account list
		acc := FuzzerAccount{
			key: key,
			address: crypto.PubkeyToAddress(key.PublicKey),
		}
		accounts = append(accounts, acc)
	}

	// Set all existing accounts as requested
	for i := 0; i < len(config.Accounts.Keys); i++ {
		// Parse our provided key string
		keyStr := config.Accounts.Keys[i]
		key, err := crypto.HexToECDSA(keyStr)
		if err != nil {
			return nil, err
		}

		// Add it to our account list
		acc := FuzzerAccount{
			key: key,
			address: crypto.PubkeyToAddress(key.PublicKey),
		}
		accounts = append(accounts, acc)
	}

	// Print some output
	fmt.Printf("Account keys loaded (%d generated, %d pre-defined) ...\n", config.Accounts.Generate, len(config.Accounts.Keys))

	// Create and return our fuzzer instance.
	fuzzer := &Fuzzer{
		config: config,
		accounts: accounts,
	}
	return fuzzer, nil
}

func (t *TestNode) deployContract(contract types.CompiledContract, deployer FuzzerAccount) (common.Address, error) {
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
		Gas: 300000,
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

func (f *Fuzzer) Start() error {
	// Compile our targets
	fmt.Printf("Compiling targets (platform '%s') ...\n", f.config.Compilation.Platform)
	compilations, err := compilation.Compile(f.config.Compilation)
	if err != nil {
		return err
	}

	// Create our genesis allocations
	genesisAlloc := make(core.GenesisAlloc)

	// Fund all of our users in the genesis block
	initBalance := new(big.Int).Div(abi.MaxInt256, big.NewInt(2))
	for i := 0; i < len(f.accounts); i++ {
		genesisAlloc[f.accounts[i].address] = core.GenesisAccount{
			Balance: initBalance, // we'll avoid uint256 in case we trigger some arithmetic error
		}
	}

	// Create a test node for each thread we intend to create.
	fmt.Printf("Creating %d test node threads ...\n", f.config.ThreadCount)
	testNodes := make([]*TestNode, 0)
	for i := 0; i < f.config.ThreadCount; i++ {
		// Create a test node
		t, err := NewTestNode(genesisAlloc)
		if err != nil {
			return err
		}

		// Add our test node to the list
		testNodes = append(testNodes, t)
	}

	// For each test node, deploy every compatible contract.
	deployedContracts := make(map[common.Address]types.CompiledContract)
	fmt.Printf("Deploying compiled contracts ...\n")
	for _, testNode := range testNodes {
		// Loop for each contract in each compilation
		for _, compilation :=  range compilations {
			for _, source := range compilation.Sources {
				for _, contract := range source.Contracts {
					// If the contract has no constructor args, deploy it. Only these contracts are supported for now.
					if len(contract.Abi.Constructor.Inputs) == 0 {
						deployedAddress, err := testNode.deployContract(contract, f.accounts[0])
						if err != nil {
							return err
						}

						// Set our deployed contract address in our lookup so we can reference it.
						deployedContracts[deployedAddress] = contract
					}
				}
			}
		}
	}

	// TODO:
	fmt.Printf("Fuzzing %d contract(s) across %d nodes ...\n", len(deployedContracts), len(testNodes))

	return nil
}