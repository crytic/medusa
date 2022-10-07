package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/fuzzing/tracing"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
	"math/rand"
	"reflect"
)

// FuzzerWorker describes a single thread worker utilizing its own go-ethereum test node to run property tests against
// Fuzzer-generated transaction sequences.
type FuzzerWorker struct {
	// workerIndex describes the index of the worker spun up by the fuzzer.
	workerIndex int

	// fuzzer describes the Fuzzer instance which this worker belongs to.
	fuzzer *Fuzzer

	// chain describes a test chain created by the FuzzerWorker to deploy contracts and run tests against.
	chain *chain.TestChain

	// deployedContracts describes a mapping of deployed contracts and the addresses they were deployed to.
	deployedContracts map[common.Address]*fuzzerTypes.Contract

	// testingBaseBlockNumber refers to the block number at which all contracts for testing have been deployed.
	testingBaseBlockNumber uint64

	// stateChangingMethods is a list of contract functions which are suspected of changing contract state
	// (non-read-only). Each FuzzerWorker fuzzes a sequence of transactions targeting stateChangingMethods, while
	// calling all propertyTests intermittently to verify state.
	stateChangingMethods []fuzzerTypes.DeployedContractMethod

	// fuzzerTracer describes the built-in tracer used to power the FuzzerWorker's coverage and execution information.
	fuzzerTracer *tracing.FuzzerTracer
}

// newFuzzerWorker creates a new FuzzerWorker, assigning it the provided worker index/id and associating it to the
// Fuzzer instance supplied.
// Returns the new FuzzerWorker
func newFuzzerWorker(fuzzer *Fuzzer, workerIndex int) *FuzzerWorker {
	// Create a fuzzing worker struct, referencing our parent fuzzing.
	worker := FuzzerWorker{
		workerIndex:          workerIndex,
		fuzzer:               fuzzer,
		deployedContracts:    make(map[common.Address]*fuzzerTypes.Contract),
		stateChangingMethods: make([]fuzzerTypes.DeployedContractMethod, 0),
		fuzzerTracer:         nil,
	}
	return &worker
}

// WorkerIndex returns the index of this FuzzerWorker in relation to its parent Fuzzer.
func (fw *FuzzerWorker) WorkerIndex() int {
	return fw.workerIndex
}

// Fuzzer returns the parent Fuzzer which spawned this FuzzerWorker.
func (fw *FuzzerWorker) Fuzzer() *Fuzzer {
	return fw.fuzzer
}

// Chain returns the Chain used by this worker as the backend for tests.
func (fw *FuzzerWorker) Chain() *chain.TestChain {
	return fw.chain
}

// metrics returns the FuzzerMetrics for the fuzzing campaign.
func (fw *FuzzerWorker) metrics() *FuzzerMetrics {
	return fw.fuzzer.metrics
}

// workerMetrics returns the fuzzerWorkerMetrics for this specific worker.
func (fw *FuzzerWorker) workerMetrics() *fuzzerWorkerMetrics {
	return &fw.fuzzer.metrics.workerMetrics[fw.workerIndex]
}

// onChainContractDeploymentAddedEvent is the event callback used when the chain detects a new contract deployment.
// It attempts bytecode matching and updates the list of deployed contracts the worker should use for fuzz testing.
func (fw *FuzzerWorker) onChainContractDeploymentAddedEvent(event chain.ContractDeploymentsAddedEvent) {
	// TODO: Optimizations to call updateStateChangingMethods less.

	// Loop through all deployed bytecode
	for _, deployedBytecode := range event.DeployedContractBytecodes {
		// Loop through all our known contract definitions
		matchedDeployment := false
		for i := 0; i < len(fw.fuzzer.contracts); i++ {
			contractDefinition := &fw.fuzzer.contracts[i]

			// If we have a match, register the deployed contract.
			if deployedBytecode.IsMatch(contractDefinition.CompiledContract()) {
				// Set our deployed contract address in our deployed contract lookup, so we can reference it later.
				fw.deployedContracts[deployedBytecode.Address] = contractDefinition

				// Update our state changing methods
				fw.updateStateChangingMethods()

				// Report our deployed contract to any test providers
				for _, testProvider := range fw.fuzzer.testCaseProviders {
					err := testProvider.OnWorkerDeployedContractAdded(fw, deployedBytecode.Address, contractDefinition)
					if err != nil {
						panic(fmt.Sprintf("a test provider returned an error when alerted of a new contract deployment: %v", err))
					}
				}

				// Skip to the next deployed contract to evaluate
				matchedDeployment = true
				break
			}
		}

		// If we didn't match any deployment, report it.
		if !matchedDeployment {
			// TODO: Figure out what to actually do here, we don't want to print errors as it could flood the console
			//  in a tight execution loop if a contract that cannot be matched is repetitively deployed.
			panic("could not match bytecode!")
		}
	}
}

// onChainContractDeploymentRemovedEvent is the event callback used when the chain detects removal of a previously
// deployed contract. It updates the list of deployed contracts the worker should use for fuzz testing.
func (fw *FuzzerWorker) onChainContractDeploymentRemovedEvent(event chain.ContractDeploymentsRemovedEvent) {
	// TODO: Optimizations to call updateStateChangingMethods less.

	// Loop through all deployed bytecode
	for _, deployedBytecode := range event.DeployedContractBytecodes {
		// Obtain our contract definition for this address
		contractDefinition, previouslyRegistered := fw.deployedContracts[deployedBytecode.Address]
		if !previouslyRegistered {
			continue
		}

		// Remove the contract from our deployed contracts mapping the worker maintains.
		delete(fw.deployedContracts, deployedBytecode.Address)

		// Update our state changing methods
		fw.updateStateChangingMethods()

		// Report our deployed contract to any test providers
		for _, testProvider := range fw.fuzzer.testCaseProviders {
			err := testProvider.OnWorkerDeployedContractDeleted(fw, deployedBytecode.Address, contractDefinition)
			if err != nil {
				panic(fmt.Sprintf("a test provider returned an error when alerted of a removed contract deployment: %v", err))
			}
		}
	}
}

// updateStateChangingMethods updates the list of state changing methods used by the worker by re-evaluating them
// from the deployedContracts lookup.
func (fw *FuzzerWorker) updateStateChangingMethods() {
	// Clear our list of state changing methods
	fw.stateChangingMethods = make([]fuzzerTypes.DeployedContractMethod, 0)

	// Loop through each deployed contract
	for contractAddress, contractDefinition := range fw.deployedContracts {
		// If we deployed the contract, also enumerate property tests and state changing methods.
		for _, method := range contractDefinition.CompiledContract().Abi.Methods {
			if !method.IsConstant() {
				// Any non-constant method should be tracked as a state changing method.
				fw.stateChangingMethods = append(fw.stateChangingMethods, fuzzerTypes.DeployedContractMethod{Address: contractAddress, Contract: contractDefinition, Method: method})
			}
		}
	}
}

// generateFuzzedAbiValue generates a value of the provided abi.Type.
// Returns the generated value.
func (fw *FuzzerWorker) generateFuzzedAbiValue(inputType *abi.Type) interface{} {
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

// generateFuzzedCall generates a new call sequence element which targets a state changing method in a contract
// deployed to this FuzzerWorker's Chain with fuzzed call data.
// Returns the call sequence element, or an error if one was encountered.
func (fw *FuzzerWorker) generateFuzzedCall() (*fuzzerTypes.CallSequenceElement, error) {
	// Verify we have state changing methods to call
	if len(fw.stateChangingMethods) == 0 {
		return nil, fmt.Errorf("cannot generate fuzzed tx as there are no state changing methods to call")
	}

	// Select a random method and sender
	// TODO: Determine if we should bias towards certain senders
	// TODO: Don't use rand.Intn here as we'll want to use a seeded rng for reproducibility.
	selectedMethod := &fw.stateChangingMethods[rand.Intn(len(fw.stateChangingMethods))]
	selectedSender := fw.fuzzer.senders[rand.Intn(len(fw.fuzzer.senders))]

	// Generate fuzzed parameters for the function call
	args := make([]interface{}, len(selectedMethod.Method.Inputs))
	for i := 0; i < len(args); i++ {
		// Create our fuzzed parameters.
		input := selectedMethod.Method.Inputs[i]
		args[i] = fw.generateFuzzedAbiValue(&input.Type)
	}

	// Encode our parameters.
	data, err := selectedMethod.Contract.CompiledContract().Abi.Pack(selectedMethod.Method.Name, args...)
	if err != nil {
		panic("could not generate tx due to error: " + err.Error())
	}

	// Create a new call and return it
	// If this is a payable function, generate value to send
	var value *big.Int
	value = big.NewInt(0)
	if selectedMethod.Method.StateMutability == "payable" {
		value = fw.fuzzer.generator.GenerateInteger(false, 64)
	}
	msg := fw.chain.CreateMessage(selectedSender, &selectedMethod.Address, value, data)

	// Return our call sequence element.
	return fuzzerTypes.NewCallSequenceElement(selectedMethod.Contract, msg), nil
}

// testCallSequence tests a call message sequence against the underlying FuzzerWorker's Chain and calls every
// TestCaseProvider registered with the parent Fuzzer to update any test results. If any call message in the sequence
// is nil, a call message will be created in its place, targeting a state changing method of a contract deployed in the
// Chain.
// Returns the length of the call sequence tested, any requests for call sequence shrinking, or an error if one occurs.
func (fw *FuzzerWorker) testCallSequence(callSequence fuzzerTypes.CallSequence) (int, []ShrinkCallSequenceRequest, error) {
	// After testing the sequence, we'll want to rollback changes to reset our testing state.
	defer func() {
		if err := fw.chain.RevertToBlockNumber(fw.testingBaseBlockNumber); err != nil {
			panic(err.Error())
		}
	}()

	// Loop for each call to send
	for i := 0; i < len(callSequence); i++ {
		// If the call sequence element is nil, generate a new call in its place.
		var err error
		if callSequence[i] == nil {
			callSequence[i], err = fw.generateFuzzedCall()
			if err != nil {
				return i, nil, err
			}
		}

		// Create a new block with our call.
		_, err = fw.chain.CreateNewBlock(callSequence[i].Call())
		if err != nil {
			return i, nil, err
		}

		// Loop through each test provider, signal our worker tested a call, and collect any requests to shrink
		// this call sequence.
		shrinkCallSequenceRequests := make([]ShrinkCallSequenceRequest, 0)
		for _, testProvider := range fw.fuzzer.testCaseProviders {
			newShrinkRequests, err := testProvider.OnWorkerCallSequenceCallTested(fw, callSequence[:i+1])
			if err != nil {
				return i, nil, err
			}
			shrinkCallSequenceRequests = append(shrinkCallSequenceRequests, newShrinkRequests...)
		}

		// If we have shrink requests, it means we violated a test, so we quit at this point
		if len(shrinkCallSequenceRequests) > 0 {
			return i + 1, shrinkCallSequenceRequests, nil
		}

		// TODO: Move everything below elsewhere

		// If we encountered an invalid opcode error, it is indicative of an assertion failure
		if _, hitInvalidOpcode := fw.fuzzerTracer.VMError().(*vm.ErrInvalidOpCode); hitInvalidOpcode {
			// TODO: Report assertion failure
		}
	}

	// Return the amount of txs we tested and no violated properties or errors.
	return len(callSequence), nil, nil
}

// shrinkCallSequence takes a provided call sequence and attempts to shrink it by looking for redundant
// calls which can be removed that continue to satisfy the provided shrink verifier.
// Returns a call sequence that was optimized to include as little calls as possible to trigger the
// expected conditions, or an error if one occurred.
func (fw *FuzzerWorker) shrinkCallSequence(callSequence fuzzerTypes.CallSequence, shrinkRequest ShrinkCallSequenceRequest) (fuzzerTypes.CallSequence, error) {
	// Define another slice to store our tx sequence
	optimizedSequence := callSequence
	for i := 0; i < len(optimizedSequence); {
		// Recreate our sequence without the item at this index
		testSeq := make(fuzzerTypes.CallSequence, 0)
		testSeq = append(testSeq, optimizedSequence[:i]...)
		testSeq = append(testSeq, optimizedSequence[i+1:]...)

		// Replay the call sequence
		for _, callSequenceElement := range testSeq {
			// Create a new block with our call
			_, err := fw.chain.CreateNewBlock(callSequenceElement.Call())
			if err != nil {
				return nil, err
			}
		}

		// Check if our verifier signalled that we met our conditions
		validShrunkSequence, err := shrinkRequest.VerifierFunction(fw, testSeq)
		if err != nil {
			return nil, err
		}

		// After testing the sequence, we'll want to rollback changes to reset our testing state.
		if err = fw.chain.RevertToBlockNumber(fw.testingBaseBlockNumber); err != nil {
			return nil, err
		}

		// If this current sequence satisfied our conditions, set it as our optimized sequence.
		if validShrunkSequence {
			optimizedSequence = testSeq
		} else {
			// We didn't remove an item at this index, so we'll iterate to the next one.
			i++
		}
	}

	// After we finished shrinking, report our result and return it.
	err := shrinkRequest.FinishedCallback(fw, optimizedSequence)
	if err != nil {
		return nil, err
	}

	return optimizedSequence, nil
}

// run takes a base Chain in a setup state ready for testing, clones it, and begins executing fuzzed transaction calls
// and asserting properties are upheld. This runs until Fuzzer.ctx cancels the operation.
// Returns a boolean indicating whether Fuzzer.ctx has indicated we cancel the operation, and an error if one occurred.
func (fw *FuzzerWorker) run(baseTestChain *chain.TestChain) (bool, error) {
	// Clone our setup base chain.
	var err error
	fw.chain, err = chain.NewTestChainWithGenesis(baseTestChain.GenesisDefinition())
	if err != nil {
		return false, err
	}

	// Subscribe our event handlers to detect contract deployment changes
	fw.chain.ContractDeploymentAddedEventEmitter.Subscribe(fw.onChainContractDeploymentAddedEvent)
	fw.chain.ContractDeploymentRemovedEventEmitter.Subscribe(fw.onChainContractDeploymentRemovedEvent)

	// Create a new tracer to power our base-feature set. Register it with our test chain's tracer forwarder.
	fw.fuzzerTracer = tracing.NewFuzzerTracer(true)
	fw.chain.TracerForwarder().AddTracer(fw.fuzzerTracer)

	// Copy our chain data from our base chain to this one (triggering all relevant events along the way).
	err = baseTestChain.CopyTo(fw.chain)
	if err != nil {
		return false, err
	}

	// Increase our generation metric as we successfully generated a test node
	fw.workerMetrics().workerStartupCount++

	// Save the current block number as all contracts have been deployed at this point, and we'll want to revert
	// to this state between testing.
	fw.testingBaseBlockNumber = fw.chain.HeadBlockNumber()

	// Enter the main fuzzing loop, restricting our memory database size based on our config variable.
	// When the limit is reached, we exit this method gracefully, which will cause the fuzzing to recreate
	// this worker with a fresh memory database.
	for fw.chain.MemoryDatabaseEntryCount() <= fw.fuzzer.config.Fuzzing.WorkerDatabaseEntryLimit {
		// If our context signalled to close the operation, exit our testing loop accordingly, otherwise continue.
		select {
		case <-fw.fuzzer.ctx.Done():
			return true, nil
		default:
			break // no signal to exit, break out of select to continue processing
		}

		// Loop through each test provider, signal our worker is about to test a new call sequence.
		for _, testProvider := range fw.fuzzer.testCaseProviders {
			err = testProvider.OnWorkerCallSequenceTesting(fw)
			if err != nil {
				return false, err
			}
		}

		// Define our call sequence slice to populate.
		callSequence := make(fuzzerTypes.CallSequence, fw.fuzzer.config.Fuzzing.MaxTxSequenceLength)

		// Test a newly generated call sequence (nil entries are filled by the method during testing)
		txsTested, shrinkVerifiers, err := fw.testCallSequence(callSequence)
		if err != nil {
			return false, err
		}

		// Update our coverage maps
		newCoverageMaps := fw.fuzzerTracer.CoverageMaps()
		if newCoverageMaps != nil {
			coverageUpdated, err := fw.metrics().coverageMaps.Update(newCoverageMaps)
			if err != nil {
				return false, err
			}

			// TODO: New coverage was achieved
			_ = coverageUpdated
		}

		// If we have any requests to shrink call sequences, do so now.
		for _, shrinkVerifier := range shrinkVerifiers {
			_, err = fw.shrinkCallSequence(callSequence[:txsTested], shrinkVerifier)
			if err != nil {
				return false, err
			}
		}

		// Loop through each test provider, signal our worker completed testing of a call sequence.
		for _, testProvider := range fw.fuzzer.testCaseProviders {
			err = testProvider.OnWorkerCallSequenceTesting(fw)
			if err != nil {
				return false, err
			}
		}

		// Update our metrics
		fw.workerMetrics().transactionsTested += uint64(txsTested)
		fw.workerMetrics().sequencesTested++
	}

	// We have not cancelled fuzzing operations, but this worker exited, signalling for it to be regenerated.
	return false, nil
}
