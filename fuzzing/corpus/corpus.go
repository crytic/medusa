package corpus

import (
	"bytes"
	"fmt"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/coverage"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/logging/colors"
	"github.com/crytic/medusa/utils/randomutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"math/big"
	"path/filepath"
	"sync"
	"time"

	"github.com/crytic/medusa/fuzzing/contracts"
)

// Corpus describes an archive of fuzzer-generated artifacts used to further fuzzing efforts. These artifacts are
// reusable across fuzzer runs. Changes to the fuzzer/chain configuration or definitions within smart contracts
// may create incompatibilities with corpus items.
type Corpus struct {
	// storageDirectory describes the directory to save corpus callSequencesByFilePath within.
	storageDirectory string

	// coverageMaps describes the total code coverage known to be achieved across all corpus call sequences.
	coverageMaps *coverage.CoverageMaps

	// mutableSequenceFiles represents a corpus directory with files which describe call sequences that should
	// be used for mutations.
	mutableSequenceFiles *corpusDirectory[calls.CallSequence]

	// immutableSequenceFiles represents a corpus directory with files which describe call sequences that should not be
	// used for mutations.
	immutableSequenceFiles *corpusDirectory[calls.CallSequence]

	// testResultSequenceFiles represents a corpus directory with files which describe call sequences that were flagged
	// to be saved by a test case provider. These are not used in mutations.
	testResultSequenceFiles *corpusDirectory[calls.CallSequence]

	// unexecutedCallSequences defines the callSequences which have not yet been executed by the fuzzer. As each item
	// is selected for execution by the fuzzer on startup, it is removed. This way, all call sequences loaded from disk
	// are executed to check for test failures.
	unexecutedCallSequences []calls.CallSequence

	// mutationTargetSequenceChooser is a provider that allows for weighted random selection of callSequences. If a
	// call sequence was not found to be compatible with this run, it is not added to the chooser.
	mutationTargetSequenceChooser *randomutils.WeightedRandomChooser[calls.CallSequence]

	// callSequencesLock provides thread synchronization to prevent concurrent access errors into
	// callSequences.
	callSequencesLock sync.Mutex

	// logger describes the Corpus's log object that can be used to log important events
	logger *logging.Logger
}

// NewCorpus initializes a new Corpus object, reading artifacts from the provided directory. If the directory refers
// to an empty path, artifacts will not be persistently stored.
func NewCorpus(corpusDirectory string) (*Corpus, error) {
	var err error
	corpus := &Corpus{
		storageDirectory:        corpusDirectory,
		coverageMaps:            coverage.NewCoverageMaps(),
		mutableSequenceFiles:    newCorpusDirectory[calls.CallSequence](""),
		immutableSequenceFiles:  newCorpusDirectory[calls.CallSequence](""),
		testResultSequenceFiles: newCorpusDirectory[calls.CallSequence](""),
		unexecutedCallSequences: make([]calls.CallSequence, 0),
		logger:                  logging.GlobalLogger.NewSubLogger("module", "corpus"),
	}

	// If we have a corpus directory set, parse our call sequences.
	if corpus.storageDirectory != "" {
		// Read mutable call sequences.
		corpus.mutableSequenceFiles.path = filepath.Join(corpus.storageDirectory, "call_sequences", "mutable")
		err = corpus.mutableSequenceFiles.readFiles("*.json")
		if err != nil {
			return nil, err
		}

		// Read immutable call sequences.
		corpus.immutableSequenceFiles.path = filepath.Join(corpus.storageDirectory, "call_sequences", "immutable")
		err = corpus.immutableSequenceFiles.readFiles("*.json")
		if err != nil {
			return nil, err
		}

		// Read test case provider related call sequences (test failures, etc).
		corpus.testResultSequenceFiles.path = filepath.Join(corpus.storageDirectory, "test_results")
		err = corpus.testResultSequenceFiles.readFiles("*.json")
		if err != nil {
			return nil, err
		}
	}

	return corpus, nil
}

// CoverageMaps exposes coverage details for all call sequences known to the corpus.
func (c *Corpus) CoverageMaps() *coverage.CoverageMaps {
	return c.coverageMaps
}

// CallSequenceEntryCount returns the total number of call sequences entries in the corpus, based on the provided filter
// flags. Some call sequences may not be valid for use if they fail validation when initializing the corpus.
// Returns the count of the requested call sequence entries.
func (c *Corpus) CallSequenceEntryCount(mutable bool, immutable bool, testResults bool) int {
	count := 0
	if mutable {
		count += len(c.mutableSequenceFiles.files)
	}
	if immutable {
		count += len(c.immutableSequenceFiles.files)
	}
	if testResults {
		count += len(c.testResultSequenceFiles.files)
	}
	return count
}

// ActiveMutableSequenceCount returns the count of call sequences recorded in the corpus which have been validated
// after Corpus initialization and are ready for use in mutations.
func (c *Corpus) ActiveMutableSequenceCount() int {
	if c.mutationTargetSequenceChooser == nil {
		return 0
	}
	return c.mutationTargetSequenceChooser.ChoiceCount()
}

// RandomMutationTargetSequence returns a weighted random call sequence from the Corpus, or an error if one occurs.
func (c *Corpus) RandomMutationTargetSequence() (calls.CallSequence, error) {
	// If we didn't initialize a chooser, return an error
	if c.mutationTargetSequenceChooser == nil {
		return nil, fmt.Errorf("corpus could not return a random call sequence because the corpus was not initialized")
	}

	// Pick a random call sequence, then clone it before returning it, so the original is untainted.
	seq, err := c.mutationTargetSequenceChooser.Choose()
	if seq == nil || err != nil {
		return nil, err
	}
	return seq.Clone()
}

// initializeSequences is a helper method for Initialize. It validates a list of call sequence files on a given
// chain, using the map of deployed contracts (e.g. to check for non-existent method called, due to code changes).
// Valid call sequences are added to the list of un-executed sequences the fuzzer should execute first.
// If this sequence list being initialized is for use with mutations, it is added to the mutationTargetSequenceChooser.
// Returns an error if one occurs.
func (c *Corpus) initializeSequences(sequenceFiles *corpusDirectory[calls.CallSequence], testChain *chain.TestChain, deployedContracts map[common.Address]*contracts.Contract, useInMutations bool) error {
	// Cache current HeadBlockNumber so that you can reset back to it after every sequence
	baseBlockNumber := testChain.HeadBlockNumber()

	// Loop for each sequence
	var err error
	for _, sequenceFileData := range sequenceFiles.files {
		// Unwrap the underlying sequence.
		sequence := sequenceFileData.data

		// Define a variable to track whether we should disable this sequence (if it is no longer applicable in some
		// way).
		sequenceInvalidError := error(nil)
		fetchElementFunc := func(currentIndex int) (*calls.CallSequenceElement, error) {
			// If we are at the end of our sequence, return nil indicating we should stop executing.
			if currentIndex >= len(sequence) {
				return nil, nil
			}

			// If we are deploying a contract and not targeting one with this call, there should be no work to do.
			currentSequenceElement := sequence[currentIndex]
			if currentSequenceElement.Call.To == nil {
				return currentSequenceElement, nil
			}

			// We are calling a contract with this call, ensure we can resolve the contract call is targeting.
			resolvedContract, resolvedContractExists := deployedContracts[*currentSequenceElement.Call.To]
			if !resolvedContractExists {
				sequenceInvalidError = fmt.Errorf("contract at address '%v' could not be resolved", currentSequenceElement.Call.To.String())
				return nil, nil
			}
			currentSequenceElement.Contract = resolvedContract

			// Next, if our sequence element uses ABI values to produce call data, our deserialized data is not yet
			// sufficient for runtime use, until we use it to resolve runtime references.
			callAbiValues := currentSequenceElement.Call.DataAbiValues
			if callAbiValues != nil {
				sequenceInvalidError = callAbiValues.Resolve(currentSequenceElement.Contract.CompiledContract().Abi)
				if sequenceInvalidError != nil {
					sequenceInvalidError = fmt.Errorf("error resolving method in contract '%v': %v", currentSequenceElement.Contract.Name(), sequenceInvalidError)
					return nil, nil
				}
			}
			return currentSequenceElement, nil
		}

		// Define actions to perform after executing each call in the sequence.
		executionCheckFunc := func(currentlyExecutedSequence calls.CallSequence) (bool, error) {
			// Update our coverage maps for each call executed in our sequence.
			lastExecutedSequenceElement := currentlyExecutedSequence[len(currentlyExecutedSequence)-1]
			covMaps := coverage.GetCoverageTracerResults(lastExecutedSequenceElement.ChainReference.MessageResults())
			_, _, covErr := c.coverageMaps.Update(covMaps)
			if covErr != nil {
				return true, covErr
			}
			return false, nil
		}

		// Execute each call sequence, populating runtime data and collecting coverage data along the way.
		_, err = calls.ExecuteCallSequenceIteratively(testChain, fetchElementFunc, executionCheckFunc)

		// If we failed to replay a sequence and measure coverage due to an unexpected error, report it.
		if err != nil {
			return fmt.Errorf("failed to initialize coverage maps from corpus, encountered an error while executing call sequence: %v\n", err)
		}

		// If the sequence was replayed successfully, we add it. If it was not, we exclude it with a warning.
		if sequenceInvalidError == nil {
			if useInMutations && c.mutationTargetSequenceChooser != nil {
				c.mutationTargetSequenceChooser.AddChoices(randomutils.NewWeightedRandomChoice[calls.CallSequence](sequence, big.NewInt(1)))
			}
			c.unexecutedCallSequences = append(c.unexecutedCallSequences, sequence)
		} else {
			c.logger.Debug("Corpus item ", colors.Bold, sequenceFileData.fileName, colors.Reset, " disabled due to error when replaying it", sequenceInvalidError)
		}

		// Revert chain state to our starting point to test the next sequence.
		err = testChain.RevertToBlockNumber(baseBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to reset the chain while seeding coverage: %v\n", err)
		}
	}
	return nil
}

// Initialize initializes any runtime data needed for a Corpus on startup. Call sequences are replayed on the post-setup
// (deployment) test chain to calculate coverage, while resolving references to compiled contracts.
// Returns the active number of corpus items, total number of corpus items, or an error if one occurred. If an error
// is returned, then the corpus counts returned will always be zero.
func (c *Corpus) Initialize(baseTestChain *chain.TestChain, contractDefinitions contracts.Contracts) (int, int, error) {
	// Acquire our call sequences lock during the duration of this method.
	c.callSequencesLock.Lock()
	defer c.callSequencesLock.Unlock()

	// Initialize our call sequence structures.
	c.mutationTargetSequenceChooser = randomutils.NewWeightedRandomChooser[calls.CallSequence]()
	c.unexecutedCallSequences = make([]calls.CallSequence, 0)

	// Create a coverage tracer to track coverage across all blocks.
	c.coverageMaps = coverage.NewCoverageMaps()
	coverageTracer := coverage.NewCoverageTracer()

	// Create our structure and event listeners to track deployed contracts
	deployedContracts := make(map[common.Address]*contracts.Contract, 0)

	// Clone our test chain, adding listeners for contract deployment events from genesis.
	testChain, err := baseTestChain.Clone(func(newChain *chain.TestChain) error {
		// After genesis, prior to adding other blocks, we attach our coverage tracer
		newChain.AddTracer(coverageTracer, true, false)

		// We also track any contract deployments, so we can resolve contract/method definitions for corpus call
		// sequences.
		newChain.Events.ContractDeploymentAddedEventEmitter.Subscribe(func(event chain.ContractDeploymentsAddedEvent) error {
			matchedContract := contractDefinitions.MatchBytecode(event.Contract.InitBytecode, event.Contract.RuntimeBytecode)
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
		return 0, 0, fmt.Errorf("failed to initialize coverage maps, base test chain cloning encountered error: %v", err)
	}

	// Set our coverage maps to those collected when replaying all blocks when cloning.
	c.coverageMaps = coverage.NewCoverageMaps()
	for _, block := range testChain.CommittedBlocks() {
		for _, messageResults := range block.MessageResults {
			covMaps := coverage.GetCoverageTracerResults(messageResults)
			_, _, covErr := c.coverageMaps.Update(covMaps)
			if covErr != nil {
				return 0, 0, err
			}
		}
	}

	// Next we replay every call sequence, checking its validity on this chain and measuring coverage. Valid sequences
	// are added to the corpus for mutations, re-execution, etc.
	//
	// The order of initializations here is important, as it determines the order of "unexecuted sequences" to replay
	// when the fuzzer's worker starts up. We want to replay test results first, so that other corpus items
	// do not trigger the same test failures instead.
	err = c.initializeSequences(c.testResultSequenceFiles, testChain, deployedContracts, false)
	if err != nil {
		return 0, 0, err
	}

	err = c.initializeSequences(c.mutableSequenceFiles, testChain, deployedContracts, true)
	if err != nil {
		return 0, 0, err
	}

	err = c.initializeSequences(c.immutableSequenceFiles, testChain, deployedContracts, false)
	if err != nil {
		return 0, 0, err
	}

	// Calculate corpus health metrics
	corpusSequencesTotal := len(c.mutableSequenceFiles.files) + len(c.immutableSequenceFiles.files) + len(c.testResultSequenceFiles.files)
	corpusSequencesActive := len(c.unexecutedCallSequences)

	return corpusSequencesActive, corpusSequencesTotal, nil
}

// addCallSequence adds a call sequence to the corpus in a given corpus directory.
// Returns an error, if one occurs.
func (c *Corpus) addCallSequence(sequenceFiles *corpusDirectory[calls.CallSequence], sequence calls.CallSequence, useInMutations bool, mutationChooserWeight *big.Int, flushImmediately bool) error {
	// Acquire a thread lock during modification of call sequence lists.
	c.callSequencesLock.Lock()

	// Check if call sequence has been added before, if so, exit without any action.
	seqHash, err := sequence.Hash()
	if err != nil {
		return err
	}

	// Verify no existing corpus item hash this same hash.
	for _, existingSeq := range sequenceFiles.files {
		// Calculate the existing sequence hash
		existingSeqHash, err := existingSeq.data.Hash()
		if err != nil {
			c.callSequencesLock.Unlock()
			return err
		}

		// Verify it is unique, if it is not, we quit immediately to avoid duplicate sequences being added.
		if bytes.Equal(existingSeqHash[:], seqHash[:]) {
			c.callSequencesLock.Unlock()
			return nil
		}
	}

	// Update our corpus directory with the new entry.
	fileName := fmt.Sprintf("%v-%v.json", time.Now().UnixNano(), uuid.New().String())
	err = sequenceFiles.addFile(fileName, sequence)
	if err != nil {
		return err
	}

	// If we want to use this sequence in mutations and initialized a chooser, add our call sequence item to it.
	if useInMutations && c.mutationTargetSequenceChooser != nil {
		if mutationChooserWeight == nil {
			mutationChooserWeight = big.NewInt(1)
		}
		c.mutationTargetSequenceChooser.AddChoices(randomutils.NewWeightedRandomChoice[calls.CallSequence](sequence, mutationChooserWeight))
	}

	// Unlock now, as flushing will lock on its own.
	c.callSequencesLock.Unlock()

	// Flush changes to disk if requested.
	if flushImmediately {
		return c.Flush()
	} else {
		return nil
	}
}

// AddTestResultCallSequence adds a call sequence recorded to the corpus due to a test case provider flagging it to be
// recorded.
// Returns an error, if one occurs.
func (c *Corpus) AddTestResultCallSequence(callSequence calls.CallSequence, mutationChooserWeight *big.Int, flushImmediately bool) error {
	return c.addCallSequence(c.testResultSequenceFiles, callSequence, false, mutationChooserWeight, flushImmediately)
}

// CheckSequenceCoverageAndUpdate checks if the most recent call executed in the provided call sequence achieved
// coverage the Corpus did not with any of its call sequences. If it did, the call sequence is added to the corpus
// and the Corpus coverage maps are updated accordingly.
// Returns an error if one occurs.
func (c *Corpus) CheckSequenceCoverageAndUpdate(callSequence calls.CallSequence, mutationChooserWeight *big.Int, flushImmediately bool) error {
	// If we have coverage-guided fuzzing disabled or no calls in our sequence, there is nothing to do.
	if len(callSequence) == 0 {
		return nil
	}

	// Obtain our coverage maps for our last call.
	lastCall := callSequence[len(callSequence)-1]
	lastCallChainReference := lastCall.ChainReference
	lastMessageResult := lastCallChainReference.Block.MessageResults[lastCallChainReference.TransactionIndex]
	lastMessageCoverageMaps := coverage.GetCoverageTracerResults(lastMessageResult)

	// If we have none, because a coverage tracer wasn't attached when processing this call, we can stop.
	if lastMessageCoverageMaps == nil {
		return nil
	}

	// Memory optimization: Remove them from the results now that we obtained them, to free memory later.
	coverage.RemoveCoverageTracerResults(lastMessageResult)

	// Merge the coverage maps into our total coverage maps and check if we had an update.
	coverageUpdated, revertedCoverageUpdated, err := c.coverageMaps.Update(lastMessageCoverageMaps)
	if err != nil {
		return err
	}

	// If we had an increase in non-reverted or reverted coverage, we save the sequence.
	// Note: We only want to save the sequence once. We're most interested if it can be used for mutations first.
	if coverageUpdated {
		// If we achieved new non-reverting coverage, save this sequence for mutation purposes.
		err = c.addCallSequence(c.mutableSequenceFiles, callSequence, true, mutationChooserWeight, flushImmediately)
		if err != nil {
			return err
		}
	} else if revertedCoverageUpdated {
		// If we did not achieve new successful coverage, but achieved an increase in reverted coverage, save this
		// sequence for non-mutation purposes.
		err = c.addCallSequence(c.immutableSequenceFiles, callSequence, false, mutationChooserWeight, flushImmediately)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnexecutedCallSequence returns a call sequence loaded from disk which has not yet been returned by this method.
// It is intended to be used by the fuzzer to run all un-executed call sequences (without mutations) to check for test
// failures. If a call sequence is returned, it will not be returned by this method again.
// Returns a call sequence loaded from disk which has not yet been executed, to check for test failures. If all
// sequences in the corpus have been executed, this will return nil.
func (c *Corpus) UnexecutedCallSequence() *calls.CallSequence {
	// Prior to thread locking, if we have no un-executed call sequences, quit.
	// This is a speed optimization, as thread locking on a central component affects performance.
	if len(c.unexecutedCallSequences) == 0 {
		return nil
	}

	// Acquire a thread lock for the duration of this method.
	c.callSequencesLock.Lock()
	defer c.callSequencesLock.Unlock()

	// Check that we have an item now that the thread is locked. This must be performed again as an item could've
	// been removed between time of check (the prior exit condition) and time of use (thread locked operations).
	if len(c.unexecutedCallSequences) == 0 {
		return nil
	}

	// Otherwise obtain the first item and remove it from the slice.
	firstSequence := c.unexecutedCallSequences[0]
	c.unexecutedCallSequences = c.unexecutedCallSequences[1:]

	// Return the first sequence
	return &firstSequence
}

// Flush writes corpus changes to disk. Returns an error if one occurs.
func (c *Corpus) Flush() error {
	// If our corpus directory is empty, it indicates we do not want to write corpus artifacts to persistent storage.
	if c.storageDirectory == "" {
		return nil
	}

	// Lock while flushing the corpus items to avoid concurrent access issues.
	c.callSequencesLock.Lock()
	defer c.callSequencesLock.Unlock()

	// Write mutation target call sequences.
	err := c.mutableSequenceFiles.writeFiles()
	if err != nil {
		return err
	}

	// Write test case provider related call sequences (test failures, etc).
	err = c.testResultSequenceFiles.writeFiles()
	if err != nil {
		return err
	}

	// Write other call sequences.
	err = c.immutableSequenceFiles.writeFiles()
	if err != nil {
		return err
	}

	return nil
}
