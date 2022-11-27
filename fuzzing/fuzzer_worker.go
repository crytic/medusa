package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/chain/types"
	"github.com/trailofbits/medusa/fuzzing/corpus"
	"github.com/trailofbits/medusa/fuzzing/coverage"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"github.com/trailofbits/medusa/utils"
	"math/big"
	"math/rand"
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
	// coverageTracer describes the tracer used to collect coverage maps during fuzzing campaigns.
	coverageTracer *coverage.CoverageTracer

	// testingBaseBlockNumber refers to the block number at which all contracts for testing have been deployed, prior
	// to any fuzzing activity. This block number is reverted to after testing each call sequence to reset state.
	testingBaseBlockNumber uint64

	// deployedContracts describes a mapping of deployed contractDefinitions and the addresses they were deployed to.
	deployedContracts map[common.Address]*fuzzerTypes.Contract
	// stateChangingMethods is a list of contract functions which are suspected of changing contract state
	// (non-read-only). A sequence of calls is generated by the FuzzerWorker, targeting stateChangingMethods
	// before executing tests.
	stateChangingMethods []fuzzerTypes.DeployedContractMethod

	// valueSet defines a set derived from Fuzzer.BaseValueSet which is further populated with runtime values by the
	// FuzzerWorker. It is the value set shared with the underlying valueGenerator.
	valueSet *valuegeneration.ValueSet
	// valueGenerator generates values for use in the fuzzing campaign (e.g. when populating abi function call
	// arguments)
	valueGenerator valuegeneration.ValueGenerator

	// Events describes the event system for the FuzzerWorker.
	Events FuzzerWorkerEvents
}

// CallSequenceTestFunc defines a method called after a fuzzing.FuzzerWorker sends another call in a types.CallSequence
// during a fuzzing campaign. It returns a ShrinkCallSequenceRequest set, which represents a set of requests for
// shrunken call sequences alongside verifiers to guide the shrinking process. This signals to the FuzzerWorker
// that this current call sequence was interesting, and it should stop building on it and find a shrunken
// sequence that satisfies the conditions specified by the ShrinkCallSequenceRequest, before generating
// entirely new call sequences. Shrink requests should not be unnecessarily requested, as this will cancel the
// current call sequence from being further generated and tested.
type CallSequenceTestFunc func(worker *FuzzerWorker, callSequence fuzzerTypes.CallSequence) ([]ShrinkCallSequenceRequest, error)

// ShrinkCallSequenceRequest is a structure signifying a request for a shrunken call sequence from the FuzzerWorker.
type ShrinkCallSequenceRequest struct {
	// VerifierFunction is a method is called upon by a FuzzerWorker to check if a shrunken call sequence satisfies
	// the needs of an original method.
	VerifierFunction func(worker *FuzzerWorker, callSequence fuzzerTypes.CallSequence) (bool, error)
	// FinishedCallback is a method called upon when the shrink request has concluded. It provides the finalized
	// shrunken call sequence.
	FinishedCallback func(worker *FuzzerWorker, shrunkenCallSequence fuzzerTypes.CallSequence) error
}

// newFuzzerWorker creates a new FuzzerWorker, assigning it the provided worker index/id and associating it to the
// Fuzzer instance supplied.
// Returns the new FuzzerWorker
func newFuzzerWorker(fuzzer *Fuzzer, workerIndex int) (*FuzzerWorker, error) {
	// Clone the fuzzer's base value set, so we can build on it with runtime values.
	valueSet := fuzzer.baseValueSet.Clone()

	// Create a value generator for the worker
	valueGenerator, err := fuzzer.NewValueGeneratorFunc(fuzzer, valueSet)
	if err != nil {
		return nil, err
	}

	// Create a fuzzing worker struct, referencing our parent fuzzing.
	worker := FuzzerWorker{
		workerIndex:          workerIndex,
		fuzzer:               fuzzer,
		deployedContracts:    make(map[common.Address]*fuzzerTypes.Contract),
		stateChangingMethods: make([]fuzzerTypes.DeployedContractMethod, 0),
		coverageTracer:       nil,
		valueSet:             valueSet,
		valueGenerator:       valueGenerator,
	}

	return &worker, nil
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

// workerMetrics returns the fuzzerWorkerMetrics for this specific worker.
func (fw *FuzzerWorker) workerMetrics() *fuzzerWorkerMetrics {
	return &fw.fuzzer.metrics.workerMetrics[fw.workerIndex]
}

// onChainContractDeploymentAddedEvent is the event callback used when the chain detects a new contract deployment.
// It attempts bytecode matching and updates the list of deployed contracts the worker should use for fuzz testing.
func (fw *FuzzerWorker) onChainContractDeploymentAddedEvent(event chain.ContractDeploymentsAddedEvent) error {
	// Add the contract address to our value set so our generator can use it in calls.
	fw.valueSet.AddAddress(event.Contract.Address)

	// Loop through all our known contract definitions
	matchedDeployment := false
	for i := 0; i < len(fw.fuzzer.contractDefinitions); i++ {
		contractDefinition := &fw.fuzzer.contractDefinitions[i]

		// If we have a match, register the deployed contract.
		if event.Contract.IsMatch(contractDefinition.CompiledContract()) {
			// Set our deployed contract address in our deployed contract lookup, so we can reference it later.
			fw.deployedContracts[event.Contract.Address] = contractDefinition

			// Update our state changing methods
			fw.updateStateChangingMethods()

			// Emit an event indicating the worker detected a new contract deployment on its chain.
			err := fw.Events.ContractAdded.Publish(FuzzerWorkerContractAddedEvent{
				Worker:             fw,
				ContractAddress:    event.Contract.Address,
				ContractDefinition: contractDefinition,
			})
			if err != nil {
				return fmt.Errorf("error returned by an event handler when a worker emitted a deployed contract added event: %v", err)
			}

			// Skip to the next deployed contract to evaluate
			matchedDeployment = true
			break
		}
	}

	// If we didn't match any deployment, report it.
	if !matchedDeployment {
		// TODO: More elegant error handling/messaging
		return fmt.Errorf("could not match bytecode of a deployed contract to any contract definition known to the fuzzer")
	}
	return nil
}

// onChainContractDeploymentRemovedEvent is the event callback used when the chain detects removal of a previously
// deployed contract. It updates the list of deployed contracts the worker should use for fuzz testing.
func (fw *FuzzerWorker) onChainContractDeploymentRemovedEvent(event chain.ContractDeploymentsRemovedEvent) error {
	// Remove the contract address from our value set so our generator doesn't use it any longer
	fw.valueSet.RemoveAddress(event.Contract.Address)

	// Obtain our contract definition for this address. If we didn't record this contract deployment in the first place,
	// there is nothing to remove, so we exit early.
	contractDefinition, previouslyRegistered := fw.deployedContracts[event.Contract.Address]
	if !previouslyRegistered {
		return nil
	}

	// Remove the contract from our deployed contracts mapping the worker maintains.
	delete(fw.deployedContracts, event.Contract.Address)

	// Update our state changing methods
	fw.updateStateChangingMethods()

	// Emit an event indicating the worker detected the removal of a previously deployed contract on its chain.
	err := fw.Events.ContractDeleted.Publish(FuzzerWorkerContractDeletedEvent{
		Worker:             fw,
		ContractAddress:    event.Contract.Address,
		ContractDefinition: contractDefinition,
	})
	if err != nil {
		return fmt.Errorf("error returned by an event handler when a worker emitted a deployed contract deleted event: %v", err)
	}
	return nil
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

// updateCoverageAndCorpus updates the corpus with the provided corpus input variables if new coverage was achieved
// when executing the last call. Coverage is measured on the transactions in the last executed block, thus the last
// block provided in the sequence is expected to be the last block constructed by the worker.
func (fw *FuzzerWorker) updateCoverageAndCorpus(callSequenceBlocks []*types.Block) error {
	// If we have coverage-guided fuzzing disabled or no calls in our sequence, there is nothing to do.
	if fw.coverageTracer == nil || len(callSequenceBlocks) == 0 {
		return nil
	}

	// Obtain our coverage maps for our last call.
	lastBlockResults := callSequenceBlocks[len(callSequenceBlocks)-1].MessageResults
	lastMessageResult := lastBlockResults[len(lastBlockResults)-1]
	lastMessageCoverageMaps := coverage.GetCoverageTracerResults(lastMessageResult)

	// Memory optimization: Remove them from the results now that we obtained them, to free memory later.
	coverage.RemoveCoverageTracerResults(lastMessageResult)

	// Merge the coverage maps into our total coverage maps and check if we had an update.
	coverageUpdated, err := fw.fuzzer.coverageMaps.Update(lastMessageCoverageMaps)
	if err != nil {
		return err
	}
	if coverageUpdated {
		// New coverage has been found with this call sequence, so we add it to the corpus.
		entry := corpus.NewCorpusEntry(callSequenceBlocks)
		err = fw.fuzzer.corpus.AddCallSequence(*entry)
		if err != nil {
			return err
		}

		// TODO: For now we flush immediately but later we'll want to move this to another routine that flushes
		//  periodically so fuzzer workers don't collide with mutex locks.
		err = fw.fuzzer.corpus.Flush()
		if err != nil {
			return err
		}
	}

	return nil
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
	// TODO: Don't use rand.Intn here as we'll want to use a seeded rng for reproducibility.
	selectedMethod := &fw.stateChangingMethods[rand.Intn(len(fw.stateChangingMethods))]
	selectedSender := fw.fuzzer.senders[rand.Intn(len(fw.fuzzer.senders))]

	// Generate fuzzed parameters for the function call
	args := make([]any, len(selectedMethod.Method.Inputs))
	for i := 0; i < len(args); i++ {
		// Create our fuzzed parameters.
		input := selectedMethod.Method.Inputs[i]
		args[i] = valuegeneration.GenerateAbiValue(fw.valueGenerator, &input.Type)
	}

	// Encode our parameters.
	data, err := selectedMethod.Contract.CompiledContract().Abi.Pack(selectedMethod.Method.Name, args...)
	if err != nil {
		return nil, fmt.Errorf("could not generate tx due to error: %v", err)
	}

	// Create a new call and return it
	// If this is a payable function, generate value to send
	var value *big.Int
	value = big.NewInt(0)
	if selectedMethod.Method.StateMutability == "payable" {
		value = fw.valueGenerator.GenerateInteger(false, 64)
	}
	msg := fw.chain.CreateMessage(selectedSender, &selectedMethod.Address, value, nil, nil, data)

	// Return our call sequence element.
	return fuzzerTypes.NewCallSequenceElement(selectedMethod.Contract, msg), nil
}

// testCallSequence tests a call message sequence against the underlying FuzzerWorker's Chain and calls every
// CallSequenceTestFunc registered with the parent Fuzzer to update any test results. If any call message in the
// sequence is nil, a call message will be created in its place, targeting a state changing method of a contract
// deployed in the Chain.
// Returns the length of the call sequence tested, any requests for call sequence shrinking, or an error if one occurs.
func (fw *FuzzerWorker) testCallSequence(callSequence fuzzerTypes.CallSequence) (int, []ShrinkCallSequenceRequest, error) {
	// After testing the sequence, we'll want to rollback changes to reset our testing state.
	defer func() {
		if err := fw.chain.RevertToBlockNumber(fw.testingBaseBlockNumber); err != nil {
			panic(err.Error())
		}
	}()

	// Define the list of resulting blocks produced by our call sequence, used to record corpus entries.
	callSequenceBlocks := make([]*types.Block, 0)

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

		// Create a new pending block
		// TODO: Select smart params for jumping this
		block, err := fw.chain.PendingBlockCreate()
		if err != nil {
			return i, nil, err
		}

		// Add our transaction to it.
		err = fw.chain.PendingBlockAddTx(callSequence[i].Call)
		if err != nil {
			return i, nil, err
		}

		// Set our chain reference data for this call sequence element. It should now be the latest item in our block.
		callSequence[i].ChainReference = &fuzzerTypes.CallSequenceElementChainReference{
			Block:            block,
			TransactionIndex: len(block.Messages) - 1,
		}

		// Commit the block
		err = fw.chain.PendingBlockCommit()
		if err != nil {
			return i, nil, err
		}

		// Record the resulting block and check for updates to coverage and corpus.
		callSequenceBlocks = append(callSequenceBlocks, block)
		err = fw.updateCoverageAndCorpus(callSequenceBlocks)
		if err != nil {
			return i, nil, err
		}

		// Loop through each test provider, signal our worker tested a call, and collect any requests to shrink
		// this call sequence.
		shrinkCallSequenceRequests := make([]ShrinkCallSequenceRequest, 0)
		for _, callSequenceTestFunc := range fw.fuzzer.CallSequenceTestFunctions {
			newShrinkRequests, err := callSequenceTestFunc(fw, callSequence[:i+1])
			if err != nil {
				return i, nil, err
			}
			shrinkCallSequenceRequests = append(shrinkCallSequenceRequests, newShrinkRequests...)
		}

		// If we have shrink requests, it means we violated a test, so we quit at this point
		if len(shrinkCallSequenceRequests) > 0 {
			return i + 1, shrinkCallSequenceRequests, nil
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
	// In case of any error, we defer an operation to revert our chain state. We purposefully ignore errors from it to
	// prioritize any others which occurred.
	defer fw.chain.RevertToBlockNumber(fw.testingBaseBlockNumber)

	// Define another slice to store our tx sequence
	optimizedSequence := callSequence
	for i := 0; i < len(optimizedSequence); {
		// Recreate our sequence without the item at this index
		testSeq := make(fuzzerTypes.CallSequence, 0)
		testSeq = append(testSeq, optimizedSequence[:i]...)
		testSeq = append(testSeq, optimizedSequence[i+1:]...)

		// Define the list of resulting blocks produced by our call sequence, used to record corpus entries.
		callSequenceBlocks := make([]*types.Block, 0)

		// Replay the call sequence
		for _, callSequenceElement := range testSeq {
			// Create a new block with our call
			block, err := fw.chain.PendingBlockCreate()
			if err != nil {
				return nil, err
			}

			// Add our transaction to it.
			err = fw.chain.PendingBlockAddTx(callSequenceElement.Call)
			if err != nil {
				return nil, err
			}

			// Set our chain reference data for this call sequence element. It should now be the latest item in our block.
			callSequenceElement.ChainReference = &fuzzerTypes.CallSequenceElementChainReference{
				Block:            block,
				TransactionIndex: len(block.Messages) - 1,
			}

			// Commit the block
			err = fw.chain.PendingBlockCommit()
			if err != nil {
				return nil, err
			}

			// Record the resulting block and check for updates to coverage and corpus.
			callSequenceBlocks = append(callSequenceBlocks, block)
			err = fw.updateCoverageAndCorpus(callSequenceBlocks)
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

	// Subscribe our chain event handlers
	fw.chain.Events.ContractDeploymentAddedEventEmitter.Subscribe(fw.onChainContractDeploymentAddedEvent)
	fw.chain.Events.ContractDeploymentRemovedEventEmitter.Subscribe(fw.onChainContractDeploymentRemovedEvent)

	// If we have coverage-guided fuzzing enabled, create a tracer to collect coverage and connect it to the chain.
	if fw.fuzzer.config.Fuzzing.CoverageEnabled {
		fw.coverageTracer = coverage.NewCoverageTracer()
		fw.chain.AddTracer(fw.coverageTracer, true, false)
	}

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
	sequencesTested := 0
	for sequencesTested <= fw.fuzzer.config.Fuzzing.WorkerResetLimit {
		// If our context signalled to close the operation, exit our testing loop accordingly, otherwise continue.
		if utils.CheckContextDone(fw.fuzzer.ctx) {
			return true, nil
		}

		// Emit an event indicating the worker is about to test a new call sequence.
		err := fw.Events.CallSequenceTesting.Publish(FuzzerWorkerCallSequenceTestingEvent{
			Worker: fw,
		})
		if err != nil {
			return false, fmt.Errorf("error returned by an event handler when a worker emitted an event indicating testing of a new call sequence is starting: %v", err)
		}

		// Define our call sequence slice to populate.
		callSequence := make(fuzzerTypes.CallSequence, fw.fuzzer.config.Fuzzing.CallSequenceLength)

		// Test a newly generated call sequence (nil entries are filled by the method during testing)
		txsTested, shrinkVerifiers, err := fw.testCallSequence(callSequence)
		if err != nil {
			return false, err
		}

		// If we have any requests to shrink call sequences, do so now.
		for _, shrinkVerifier := range shrinkVerifiers {
			_, err = fw.shrinkCallSequence(callSequence[:txsTested], shrinkVerifier)
			if err != nil {
				return false, err
			}
		}

		// Emit an event indicating the worker is about to test a new call sequence.
		err = fw.Events.CallSequenceTested.Publish(FuzzerWorkerCallSequenceTestedEvent{
			Worker: fw,
		})
		if err != nil {
			return false, fmt.Errorf("error returned by an event handler when a worker emitted an event indicating testing of a new call sequence has concluded: %v", err)
		}

		// Update our metrics
		fw.workerMetrics().callsTested += uint64(txsTested)
		fw.workerMetrics().sequencesTested++
		sequencesTested++
	}

	// We have not cancelled fuzzing operations, but this worker exited, signalling for it to be regenerated.
	return false, nil
}
