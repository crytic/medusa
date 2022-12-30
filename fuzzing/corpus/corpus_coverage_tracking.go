package corpus

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/fuzzing/coverage"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
)

// MeasureCorpusCoverage takes a test chain in its post-setup (deployment), pre-fuzzing campaign state, and a corpus,
// and returns coverage maps which represent the coverage achieved when replaying corpus call sequences over the
// provided chain.
func MeasureCorpusCoverage(corpus *Corpus, baseTestChain *chain.TestChain, contracts fuzzerTypes.Contracts) (*coverage.CoverageMaps, error) {
	// Create our coverage maps and a coverage tracer
	coverageMaps := coverage.NewCoverageMaps()
	coverageTracer := coverage.NewCoverageTracer()

	// Create our structure and event listeners to track deployed contracts
	deployedContracts := make(map[common.Address]*fuzzerTypes.Contract, 0)

	// Clone our test chain.
	// and listen for contract deployment events.
	testChain, err := baseTestChain.Clone(func(newChain *chain.TestChain) error {
		// After genesis, prior to adding other blocks, we attach our coverage tracer
		newChain.AddTracer(coverageTracer, true, false)

		// We also track any contract deployments, so we can resolve contract/method definitions for corpus call
		// sequences.
		newChain.Events.ContractDeploymentAddedEventEmitter.Subscribe(func(event chain.ContractDeploymentsAddedEvent) error {
			matchedContract := contracts.MatchBytecode(event.Contract.InitBytecode, event.Contract.RuntimeBytecode)
			if matchedContract != nil {
				deployedContracts[event.Contract.Address] = matchedContract
			}
			return nil
		})
		newChain.Events.ContractDeploymentRemovedEventEmitter.Subscribe(func(event chain.ContractDeploymentsRemovedEvent) error {
			delete(deployedContracts, event.Contract.Address)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize coverage maps, base test chain cloning encountered error: %v", err)
	}

	// Next we measure coverage for every corpus call sequence.
	corpus.callSequencesLock.Lock()
	defer corpus.callSequencesLock.Unlock()

	// Cache current HeadBlockNumber so that you can reset back to it after every sequence
	baseBlockNumber := testChain.HeadBlockNumber()

	// Loop for each sequence
	for _, sequenceFileData := range corpus.callSequences {
		// Unwrap the underlying sequence.
		sequence := sequenceFileData.data

		// Define a variable to track whether we should disable this sequence (if it is no longer applicable in some
		// way).
		sequenceInvalid := false

		// Define actions to perform prior to executing each call in the sequence.
		preStepFunc := func(index int) (bool, error) {
			// If we are deploying a contract and not targeting one with this call, there should be no work to do.
			currentSequenceElement := sequence[index]
			if currentSequenceElement.Call.MsgTo == nil {
				return false, nil
			}

			// We are calling a contract with this call, ensure we can resolve the contract call is targeting.
			resolvedContract, resolvedContractExists := deployedContracts[*currentSequenceElement.Call.MsgTo]
			if !resolvedContractExists {
				sequenceInvalid = true
				return true, nil
			}
			currentSequenceElement.Contract = resolvedContract

			// Next, if our sequence element uses abi values for call data, ensure the method is resolved.
			callAbiValues := currentSequenceElement.Call.MsgDataAbiValues
			if callAbiValues != nil {
				callAbiValues.Method, err = currentSequenceElement.Contract.CompiledContract().Abi.MethodById(callAbiValues.MethodID)
				if err != nil || callAbiValues.Method == nil {
					sequenceInvalid = true
					return true, nil
				}
			}
			return false, nil
		}

		// Define actions to perform after executing each call in the sequence.
		postStepFunc := func(index int) (bool, error) {
			// Update our coverage maps for each call executed in our sequence.
			covMaps := coverage.GetCoverageTracerResults(sequence[index].ChainReference.MessageResults())
			_, covErr := coverageMaps.Update(covMaps)
			if covErr != nil {
				return true, err
			}
			return false, nil
		}

		// Execute each call sequence, populating runtime data and collecting coverage data along the way.
		_, err = sequence.ExecuteOnChain(testChain, true, preStepFunc, postStepFunc)

		// If we failed to replay a sequence and measure coverage, report it.
		if err != nil {
			return nil, fmt.Errorf("failed to initialize coverage maps from corpus, encountered an error while executing call sequence: %v\n", err)
		}

		// TODO: If this is an invalid sequence, disable it.
		if sequenceInvalid {
			fmt.Printf("corpus item disabled because it references an unresolved contract/method: %v\n", sequenceFileData.filePath)
		}

		// Revert chain state to our starting point to test the next sequence.
		err = testChain.RevertToBlockNumber(baseBlockNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to reset the chain while seeding coverage: %v\n", err)
		}
	}
	return coverageMaps, nil
}

// UpdateCorpusAndCoverageMaps takes the provided coverage maps and corpus, and checks if the most recent call
// executed in the provided call sequence achieved new coverage. If it did, the call sequence is added to the corpus
// and the coverage maps are updated accordingly.
// Returns an error if one occurs.
func UpdateCorpusAndCoverageMaps(coverageMaps *coverage.CoverageMaps, corpus *Corpus, callSequence fuzzerTypes.CallSequence) error {
	// If we have coverage-guided fuzzing disabled or no calls in our sequence, there is nothing to do.
	if len(callSequence) == 0 {
		return nil
	}

	// Obtain our coverage maps for our last call.
	lastCallChainReference := callSequence[len(callSequence)-1].ChainReference
	lastMessageResult := lastCallChainReference.Block.MessageResults[lastCallChainReference.TransactionIndex]
	lastMessageCoverageMaps := coverage.GetCoverageTracerResults(lastMessageResult)

	// If we have none, because a coverage tracer wasn't attached when processing this call, we can stop.
	if lastMessageCoverageMaps == nil {
		return nil
	}

	// Memory optimization: Remove them from the results now that we obtained them, to free memory later.
	coverage.RemoveCoverageTracerResults(lastMessageResult)

	// Merge the coverage maps into our total coverage maps and check if we had an update.
	coverageUpdated, err := coverageMaps.Update(lastMessageCoverageMaps)
	if err != nil {
		return err
	}
	if coverageUpdated {
		// New coverage has been found with this call sequence, so we add it to the corpus.
		err = corpus.AddCallSequence(callSequence)
		if err != nil {
			return err
		}

		err = corpus.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}
