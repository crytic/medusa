package fuzzer

import (
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"math/big"
	"medusa/compilation/types"
	"strings"
)

// fuzzerWorker describes a single thread worker utilizing its own go-ethereum test node to run property tests against
// Fuzzer-generated transaction sequences.
type fuzzerWorker struct {
	// fuzzer describes the Fuzzer instance which this worker belongs to.
	fuzzer *Fuzzer

	// testNode describes a TestNode created by the fuzzerWorker to run tests against.
	testNode *TestNode

	// deployedContracts describes a mapping of deployed contracts and the addresses they were deployed to.
	deployedContracts map[common.Address]types.CompiledContract

	// propertyTests describes the contract functions which represent properties to be tested.
	// These should be read-only (pure/view) functions which take no input parameters and return a boolean variable.
	// The functions return true if the property/invariant is upheld.
	propertyTests []deployedMethod

	// stateChangingMethods is a list of contract functions which are suspected of changing contract state
	// (non-read-only). Each fuzzerWorker fuzzes a sequence of transactions targeting stateChangingMethods, while
	// calling all propertyTests intermittently to verify state.
	stateChangingMethods []deployedMethod
}

// deployedMethod describes a method which is accessible through contract deployed on the test node.
type deployedMethod struct {
	// address represents the Ethereum address where the deployed contract containing the method exists.
	address common.Address

	// contract describes the contract which was deployed and contains the target method.
	contract types.CompiledContract

	// method describes the method which is available through the deployed contract.
	method abi.Method
}

func newFuzzWorker(fuzzer *Fuzzer) *fuzzerWorker {
	// Create a fuzzer worker struct, referencing our parent fuzzer.
	worker := fuzzerWorker{
		fuzzer: fuzzer,
		deployedContracts: make(map[common.Address]types.CompiledContract),
		propertyTests: make([]deployedMethod, 0),
		stateChangingMethods: make([]deployedMethod, 0),
	}
	return &worker
}

func (fw *fuzzerWorker) trackDeployedContract(deployedAddress common.Address, contract types.CompiledContract) {
	// Set our deployed contract address in our deployed contract lookup, so we can reference it later.
	fw.deployedContracts[deployedAddress] = contract

	// If we deployed the contract, also enumerate property tests and state changing methods.
	for _, method := range contract.Abi.Methods {
		if method.IsConstant() {
			// Check if this is a property test and add it to our list if so.
			if len(method.Inputs) == 0 && len(method.Outputs) == 1 && method.Outputs[0].Type.T == abi.BoolTy &&
				(strings.HasPrefix(method.Name, "medusa_") || strings.HasPrefix(method.Name, "echidna_")) {
				fw.propertyTests = append(fw.propertyTests, deployedMethod{address: deployedAddress, contract: contract, method: method})
			}
			continue
		}
		// Any non-constant method should be tracked as a state changing method.
		fw.stateChangingMethods = append(fw.stateChangingMethods, deployedMethod{address: deployedAddress, contract: contract, method: method})
	}
}

func (fw *fuzzerWorker) deployAndTrackCompiledContracts() error {
	// Loop for each contract in each compilation and deploy it to the test node.
	for _, comp :=  range fw.fuzzer.compilations {
		for _, source := range comp.Sources {
			for _, contract := range source.Contracts {
				// If the contract has no constructor args, deploy it. Only these contracts are supported for now.
				if len(contract.Abi.Constructor.Inputs) == 0 {
					// TODO: Determine if we should use random accounts to deploy each contract, the same, or
					//  user-specified, instead of `accounts[0]`.
					deployedAddress, err := fw.testNode.deployContract(contract, fw.fuzzer.accounts[0])
					if err != nil {
						return err
					}

					// Ensure our worker tracks the deployed contract and any property tests
					fw.trackDeployedContract(deployedAddress, contract)
				}
			}
		}
	}

	return nil
}

func (fw *fuzzerWorker) checkViolatedPropertyTests() *deployedMethod {
	// Loop through all property tests methods
	for _, propertyTest := range fw.propertyTests {
		// Generate our ABI input data for the call (just the method ID, no args)
		data, err := propertyTest.contract.Abi.Pack(propertyTest.method.Name)
		if err != nil {
			panic(err)
		}

		// Call the underlying contract
		// TODO: Determine if we should use `accounts[0]` or have a separate funded account for the assertions.
		res, err := fw.testNode.CallContract(ethereum.CallMsg{
			From: fw.fuzzer.accounts[0].address,
			To: &propertyTest.address,
			Gas: fw.testNode.pendingBlock.GasLimit(),
			GasFeeCap: big.NewInt(1e14), // maxgascost = 2.1ether
			GasTipCap: big.NewInt(1),
			Value: big.NewInt(0), // the remaining balance for fee is 2.1ether
			Data: data,
		})

		// If we have an error calling an invariant method, we should panic as we never want this to fail.
		if err != nil {
			panic(err)
		}

		// Verify the execution did not revert
		if !res.Failed() {
			// Decode our ABI outputs
			retVals, err := propertyTest.method.Outputs.Unpack(res.Return())

			// We should not have an issue decoding ABI
			if err != nil {
				panic(err)
			}

			// We should have one return value.
			if len(retVals) != 1 {
				panic (fmt.Sprintf("unexpected number of return values in property '%s'", propertyTest.method.Name))
			}

			// The one return value should be a bool
			bl, ok := retVals[0].(bool)
			if !ok {
				panic (fmt.Sprintf("could not obtain bool from first ABI output element in property '%s'", propertyTest.method.Name))
			}

			// If we returned true, our property test upheld, so we can continue to the next.
			if bl {
				continue
			}

			// Handle `false` property assertion result
			return &propertyTest
		}

		// Handle revert/failed tx result
		return &propertyTest
	}

	// We did not fail any property tests.
	return nil
}

func (fw *fuzzerWorker) run() error {
	// Create our genesis allocations.
	// NOTE: Sharing GenesisAlloc between nodes will result in some accounts not being funded for some reason.
	genesisAlloc := make(core.GenesisAlloc)

	// Fund all of our users in the genesis block
	initBalance := new(big.Int).Div(abi.MaxInt256, big.NewInt(2))
	for i := 0; i < len(fw.fuzzer.accounts); i++ {
		genesisAlloc[fw.fuzzer.accounts[i].address] = core.GenesisAccount{
			Balance: initBalance, // we'll avoid uint256 in case we trigger some arithmetic error
		}
	}

	// Create a test node
	var err error
	fw.testNode, err = NewTestNode(genesisAlloc)
	if err != nil {
		return err
	}

	// When exiting this function, stop the test node
	defer fw.testNode.Stop()

	// Deploy and track all compiled contracts
	err = fw.deployAndTrackCompiledContracts()
	if err != nil {
		return err
	}

	// Enter the main fuzzing loop, restricting our memory
	// TODO: temporarily set at 1GB per thread, make this configurable
	for fw.testNode.GetMemoryUsage() <= 1024*1024*1024 {
		// TODO: Generate fuzzed tx sequence

		// Look for a violated property test
		failedPropertyTest := fw.checkViolatedPropertyTests()
		if failedPropertyTest != nil {
			// TODO: Handle violated property test
			panic("PROPERTY TEST FAILED: " + failedPropertyTest.method.Name)
		}

		// If our context signalled that we're done, quit, otherwise we'll continue.
		select {
		case <- fw.fuzzer.ctx.Done():
			return fw.fuzzer.ctx.Err()
		default:
			break
		}
	}
	return nil
}