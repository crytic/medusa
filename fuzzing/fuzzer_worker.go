package fuzzing

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/trailofbits/medusa/compilation/types"
)

// fuzzerWorker describes a single thread worker utilizing its own go-ethereum test node to run property tests against
// Fuzzer-generated transaction sequences.
type fuzzerWorker struct {
	// workerIndex describes the index of the worker spun up by the fuzzer.
	workerIndex int

	// fuzzer describes the Fuzzer instance which this worker belongs to.
	fuzzer *Fuzzer

	// testNode describes a testNode created by the fuzzerWorker to run tests against.
	testNode *testNode

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

// newFuzzerWorker creates a new fuzzerWorker, assigning it the provided worker index/id and associating it to the
// Fuzzer instance supplied.
// Returns the new fuzzerWorker
func newFuzzerWorker(fuzzer *Fuzzer, workerIndex int) *fuzzerWorker {
	// Create a fuzzing worker struct, referencing our parent fuzzing.
	worker := fuzzerWorker{
		workerIndex:          workerIndex,
		fuzzer:               fuzzer,
		deployedContracts:    make(map[common.Address]types.CompiledContract),
		propertyTests:        make([]deployedMethod, 0),
		stateChangingMethods: make([]deployedMethod, 0),
	}
	return &worker
}

// metrics returns the FuzzerMetrics for the fuzzing campaign.
func (fw *fuzzerWorker) metrics() *FuzzerMetrics {
	return fw.fuzzer.metrics
}

// workerMetrics returns the fuzzerWorkerMetrics for this specific worker.
func (fw *fuzzerWorker) workerMetrics() *fuzzerWorkerMetrics {
	return &fw.fuzzer.metrics.workerMetrics[fw.workerIndex]
}

// registerDeployedContract registers an address with a compiled contract descriptor for it to be tracked by the
// fuzzerWorker, both as methods of changing state and for properties to assert.
func (fw *fuzzerWorker) registerDeployedContract(deployedAddress common.Address, contract types.CompiledContract) {
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

// deployAndRegisterCompiledContracts deploys all contracts in the parent Fuzzer.compilations to a test node and
// registers their addresses to be tracked by the fuzzerWorker.
// Returns an error if one is encountered.
func (fw *fuzzerWorker) deployAndRegisterCompiledContracts() error {
	// Loop for each contract in each compilation and deploy it to the test node.
	for _, comp := range fw.fuzzer.compilations {
		for _, source := range comp.Sources {
			for _, contract := range source.Contracts {
				// If the contract has no constructor args, deploy it. Only these contracts are supported for now.
				if len(contract.Abi.Constructor.Inputs) == 0 {
					// TODO: Determine if we should use random accounts to deploy each contract, the same, or
					//  user-specified, instead of `accounts[0]`.
					deployedAddress, err := fw.testNode.DeployContract(contract, fw.fuzzer.accounts[0].key)
					if err != nil {
						return err
					}

					// Ensure our worker tracks the deployed contract and any property tests
					fw.registerDeployedContract(deployedAddress, contract)
				}
			}
		}
	}

	return nil
}

// checkViolatedPropertyTests executes all property tests in deployed contracts in this fuzzerWorker's testNode.
// Returns deployedMethod references for all failed property test results.
func (fw *fuzzerWorker) checkViolatedPropertyTests() []deployedMethod {
	// Create a list of violated properties
	violatedProperties := make([]deployedMethod, 0)

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
			From:      fw.fuzzer.accounts[0].address,
			To:        &propertyTest.address,
			Gas:       fw.testNode.pendingBlock.GasLimit(),
			GasFeeCap: big.NewInt(1e14), // maxgascost = 2.1ether
			GasTipCap: big.NewInt(1),
			Value:     big.NewInt(0), // the remaining balance for fee is 2.1ether
			Data:      data,
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
				panic(fmt.Sprintf("unexpected number of return values in property '%s'", propertyTest.method.Name))
			}

			// The one return value should be a bool
			bl, ok := retVals[0].(bool)
			if !ok {
				panic(fmt.Sprintf("could not obtain bool from first ABI output element in property '%s'", propertyTest.method.Name))
			}

			// If we returned true, our property test upheld, so we can continue to the next.
			if bl {
				continue
			}

			// Handle `false` property assertion result
			violatedProperties = append(violatedProperties, propertyTest)
			continue
		}

		// Handle revert/failed tx result
		violatedProperties = append(violatedProperties, propertyTest)
		continue
	}

	// Return our violated properties.
	return violatedProperties
}

// generateFuzzedAbiValue generates a value of the provided abi.Type.
// Returns the generated value.
func (fw *fuzzerWorker) generateFuzzedAbiValue(inputType *abi.Type) interface{} {
	//
	if inputType.T == abi.AddressTy {
		return fw.fuzzer.generator.GenerateAddress()
	} else if inputType.T == abi.UintTy {
		if inputType.Size == 64 {
			return fw.fuzzer.generator.GenerateInteger(false, inputType.Size).Uint64()
		} else if inputType.Size == 32 {
			return uint32(fw.fuzzer.generator.GenerateInteger(false, inputType.Size).Uint64())
		} else if inputType.Size == 16 {
			return uint16(fw.fuzzer.generator.GenerateInteger(false, inputType.Size).Uint64())
		} else if inputType.Size == 8 {
			return uint8(fw.fuzzer.generator.GenerateInteger(false, inputType.Size).Uint64())
		} else {
			return fw.fuzzer.generator.GenerateInteger(false, inputType.Size)
		}
	} else if inputType.T == abi.IntTy {
		if inputType.Size == 64 {
			return fw.fuzzer.generator.GenerateInteger(true, inputType.Size).Int64()
		} else if inputType.Size == 32 {
			return int32(fw.fuzzer.generator.GenerateInteger(true, inputType.Size).Int64())
		} else if inputType.Size == 16 {
			return int16(fw.fuzzer.generator.GenerateInteger(true, inputType.Size).Int64())
		} else if inputType.Size == 8 {
			return int8(fw.fuzzer.generator.GenerateInteger(true, inputType.Size).Int64())
		} else {
			return fw.fuzzer.generator.GenerateInteger(true, inputType.Size)
		}
	} else if inputType.T == abi.BoolTy {
		return fw.fuzzer.generator.GenerateBool()
	} else if inputType.T == abi.StringTy {
		return fw.fuzzer.generator.GenerateString()
	} else if inputType.T == abi.BytesTy {
		return fw.fuzzer.generator.GenerateBytes()
	} else if inputType.T == abi.FixedBytesTy {
		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		// We opt to keep our API for generators simple, creating the array here and copying elements from a slice.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		bytes := reflect.ValueOf(fw.fuzzer.generator.GenerateFixedBytes(inputType.Size))
		for i := 0; i < array.Len(); i++ {
			array.Index(i).Set(bytes.Index(i))
		}
		return array.Interface()
	} else if inputType.T == abi.ArrayTy {
		// Read notes for fixed bytes to understand the need to create this array through reflection.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < array.Len(); i++ {
			array.Index(i).Set(reflect.ValueOf(fw.generateFuzzedAbiValue(inputType.Elem)))
		}
		return array.Interface()
	} else if inputType.T == abi.SliceTy {
		// Dynamic sized arrays are represented as slices.
		sliceSize := fw.fuzzer.generator.GenerateArrayLength()
		slice := reflect.MakeSlice(inputType.GetType(), sliceSize, sliceSize)
		for i := 0; i < slice.Len(); i++ {
			slice.Index(i).Set(reflect.ValueOf(fw.generateFuzzedAbiValue(inputType.Elem)))
		}
		return slice.Interface()
	} else if inputType.T == abi.TupleTy {
		// Tuples are used to represent structs. For go-ethereum's ABI provider, we're intended to supply matching
		// struct implementations, so we create and populate them through reflection.
		st := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < len(inputType.TupleElems); i++ {
			st.Field(i).Set(reflect.ValueOf(fw.generateFuzzedAbiValue(inputType.TupleElems[i])))
		}
		return st.Interface()
	}

	// Unexpected types will result in a panic as we should support these values as soon as possible:
	// - Mappings cannot be used in public/external methods and must reference storage, so we shouldn't ever
	//	 see cases of it unless Solidity was updated in the future.
	// - FixedPoint types are currently unsupported.
	panic(fmt.Sprintf("attempt to generate function argument of unsupported type: '%s'", inputType.String()))
}

// generateFuzzedTx generates a new transaction and determines which fuzzerAccount should send it on this fuzzerWorker's
// testNode.
// Returns the transaction and a fuzzerAccount intended to be the sender, or an error if one was encountered.
func (fw *fuzzerWorker) generateFuzzedTx() (*txSequenceElement, error) {
	// Verify we have state changing methods to call
	if len(fw.stateChangingMethods) == 0 {
		return nil, fmt.Errorf("cannot generate fuzzed tx as there are no state changing methods to call")
	}

	// Select a random method and sender
	// TODO: Determine if we should bias towards certain senders
	selectedMethod := &fw.stateChangingMethods[rand.Intn(len(fw.stateChangingMethods))]
	selectedSender := &fw.fuzzer.accounts[rand.Intn(len(fw.fuzzer.accounts))]

	// Generate fuzzed parameters for the function call
	args := make([]interface{}, len(selectedMethod.method.Inputs))
	for i := 0; i < len(args); i++ {
		// Create our fuzzed parameters.
		input := selectedMethod.method.Inputs[i]
		args[i] = fw.generateFuzzedAbiValue(&input.Type)
	}

	// Encode our parameters.
	data, err := selectedMethod.contract.Abi.Pack(selectedMethod.method.Name, args...)
	if err != nil {
		panic("could not generate tx due to error: " + err.Error())
	}

	// Create a new transaction and return it
	// If this is a payable function, generate value to send
	var value *big.Int
	value = big.NewInt(0)
	if selectedMethod.method.StateMutability == "payable" {
		value = fw.fuzzer.generator.GenerateInteger(false, 64)
	}

	tx := &coreTypes.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(0),
		Gas:      0,
		To:       &selectedMethod.address,
		Value:    value,
		Data:     data,
	}

	// Return our transaction sequence element.
	return newTxSequenceElement(tx, selectedSender), nil
}

// testTxSequence tests a transaction sequence and checks if it violates any known property tests. If any element of
// the provided transaction sequence array is nil, a new transaction will be generated in its place. Thus, this method
// can be used to check a pre-defined sequence, or to generate and check one of a provided length.
// Returns the length of the transaction sequence tested, the violated property test methods, or any error if one
// occurs.
func (fw *fuzzerWorker) testTxSequence(txSequence []*txSequenceElement) (int, []deployedMethod, error) {
	// After testing the sequence, we'll want to rollback changes and panic if we encounter an error, as it might
	// mean our testing state is compromised.
	defer func() {
		if err := fw.testNode.RevertUncommittedChanges(); err != nil {
			panic(err.Error())
		}
	}()

	// Loop for each transaction to execute
	for i := 0; i < len(txSequence); i++ {
		// If the transaction sequence element is nil, generate a new fuzzed tx in its place.
		var err error
		if txSequence[i] == nil {
			txSequence[i], err = fw.generateFuzzedTx()
			if err != nil {
				return i, nil, err
			}
		}

		// Obtain our transaction information
		txInfo := txSequence[i]

		// Send our transaction
		_, _, err = fw.testNode.SignAndSendLegacyTransaction(txInfo.tx, txInfo.sender.key, true)
		if err != nil {
			return i, nil, err
		}

		// Record any violated property tests.
		violatedPropertyTests := fw.checkViolatedPropertyTests()

		// Check if we have any violated property tests.
		if len(violatedPropertyTests) > 0 {
			// We have violated properties, return the tx sequence length needed to cause the issue, as well as the
			// violated tests.
			return i + 1, violatedPropertyTests, nil
		}
	}

	// Return the amount of txs we tested and no violated properties or errors.
	return len(txSequence), nil, nil
}

// shrinkTxSequence takes a provided transaction sequence and attempts to shrink it by looking for redundant
// transactions in the sequence which can be removed while maintaining the same property test violations.
// Returns a transaction sequence that was optimized to include as little transactions as possible to trigger the
// expected number of property test violations, or returns an error if one occurs.
func (fw *fuzzerWorker) shrinkTxSequence(txSequence []*txSequenceElement, expectedFailures int) ([]*txSequenceElement, error) {
	// Define another slice to store our tx sequence
	optimizedSequence := txSequence
	for i := 0; i < len(optimizedSequence); {
		// Recreate our sequence without the item at this index
		testSeq := make([]*txSequenceElement, 0)
		testSeq = append(testSeq, optimizedSequence[:i]...)
		testSeq = append(testSeq, optimizedSequence[i+1:]...)

		// Test this shrunk sequence
		txsTested, violatedPropertyTests, err := fw.testTxSequence(testSeq)
		if err != nil {
			return nil, err
		}

		// If we violated the expected amount of properties, we can continue to the next iteration to try and remove
		// the element at this index again.
		if len(violatedPropertyTests) == expectedFailures {
			// Set the sequence to this one as it holds our violated properties, then continue to trying to remove
			// at this index again since the item at that index will now be new.
			optimizedSequence = testSeq[:txsTested]
			continue
		}

		// We didn't remove an item at this index, so we'll iterate to the next one.
		i++
	}
	return optimizedSequence, nil
}

// run sets up a testNode and begins executing fuzzed transaction calls and asserting properties are upheld.
// This runs until Fuzzer.ctx cancels the operation.
// Returns a boolean indicating whether Fuzzer.ctx has indicated we cancel the operation, and an error if one occurred.
func (fw *fuzzerWorker) run() (bool, error) {
	// Create our genesis allocations.
	// NOTE: Sharing GenesisAlloc between nodes will result in some accounts not being funded for some reason.
	genesisAlloc := make(core.GenesisAlloc)

	// Fund all of our users in the genesis block
	initBalance := new(big.Int).Div(abi.MaxInt256, big.NewInt(2)) // TODO: make this configurable
	for i := 0; i < len(fw.fuzzer.accounts); i++ {
		genesisAlloc[fw.fuzzer.accounts[i].address] = core.GenesisAccount{
			Balance: initBalance,
		}
	}

	// Create a test node
	var err error
	fw.testNode, err = newTestNode(genesisAlloc)
	if err != nil {
		return false, err
	}

	// When exiting this function, stop the test node
	defer fw.testNode.Stop()

	// Increase our generation metric as we successfully generated a test node
	fw.workerMetrics().workerStartupCount++

	// Deploy and track all compiled contracts
	err = fw.deployAndRegisterCompiledContracts()
	if err != nil {
		return false, err
	}

	// Enter the main fuzzing loop, restricting our memory database size based on our config variable.
	// When the limit is reached, we exit this method gracefully, which will cause the fuzzing to recreate
	// this worker with a fresh memory database.
	for fw.testNode.MemoryDatabaseEntryCount() <= fw.fuzzer.config.Fuzzing.WorkerDatabaseEntryLimit {
		// If our context signalled to close the operation, exit our testing loop accordingly, otherwise continue.
		select {
		case <-fw.fuzzer.ctx.Done():
			return true, nil
		default:
			break // no signal to exit, break out of select to continue processing
		}

		// Define our transaction sequence slice to populate.
		txSequence := make([]*txSequenceElement, fw.fuzzer.config.Fuzzing.MaxTxSequenceLength)

		// Test a newly generated transaction sequence (nil entries in the slice result in generated txs)
		txsTested, violatedPropertyTests, err := fw.testTxSequence(txSequence)
		if err != nil {
			return false, err
		}

		// Update our metrics
		fw.workerMetrics().transactionsTested += uint64(txsTested)
		fw.workerMetrics().sequencesTested++
		newCoverageMaps := fw.testNode.tracer.CoverageMaps()
		if newCoverageMaps != nil {
			coverageUpdated, err := fw.metrics().coverageMaps.Update(newCoverageMaps)
			if err != nil {
				return false, err
			}

			// TODO: New coverage was achieved
			_ = coverageUpdated
		}

		// Check if we have violated properties
		if len(violatedPropertyTests) > 0 {
			// We'll want to shrink our tx sequence to remove unneeded txs from our tx list.
			txSequence, err = fw.shrinkTxSequence(txSequence[:txsTested], len(violatedPropertyTests))
			if err != nil {
				return false, err
			}

			// Create our struct to track tx sequence information for our failed test.
			txInfoSeq := make([]FuzzerResultFailedTestTx, len(txSequence))
			for x := 0; x < len(txInfoSeq); x++ {
				contract := fw.deployedContracts[*txSequence[x].tx.To]
				txInfoSeq[x] = *NewFuzzerResultFailedTestTx(&contract, txSequence[x].tx)
			}
			fw.fuzzer.results.addFailedTest(NewFuzzerResultFailedTest(txInfoSeq, violatedPropertyTests))

			// TODO: For now we'll stop our fuzzer and print our results, but we should add a toggle to allow
			//  for continued execution to find more property violations.
			fmt.Printf("%s\n", fw.fuzzer.results.GetFailedTests()[0].String())
			fw.fuzzer.Stop()
		}
	}

	// We have not cancelled fuzzing operations, but this worker exited, signalling for it to be regenerated.
	return false, nil
}
