package corpus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/fuzzing/calls"
	"github.com/trailofbits/medusa/fuzzing/coverage"
	"github.com/trailofbits/medusa/utils"
	"github.com/trailofbits/medusa/utils/randomutils"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/trailofbits/medusa/fuzzing/contracts"
)

// Corpus describes an archive of fuzzer-generated artifacts used to further fuzzing efforts. These artifacts are
// reusable across fuzzer runs. Changes to the fuzzer/chain configuration or definitions within smart contracts
// may create incompatibilities with corpus items.
type Corpus struct {
	// storageDirectory describes the directory to save corpus callSequencesByFilePath within.
	storageDirectory string

	// coverageMaps describes the total code coverage known to be achieved across all corpus call sequences.
	coverageMaps *coverage.CoverageMaps

	// callSequences is a list of call sequences that increased coverage or otherwise were found to be valuable
	// to the fuzzer.
	callSequences []*corpusFile[calls.CallSequence]

	// unexecutedCallSequences defines the callSequences which have not yet been executed by the fuzzer. As each item
	// is selected for execution by the fuzzer on startup, it is removed. This way, all call sequences loaded from disk
	// are executed to check for test failures.
	unexecutedCallSequences []calls.CallSequence

	// weightedCallSequenceChooser is a provider that allows for weighted random selection of callSequences. If a
	// call sequence was not found to be compatible with this run, it is not added to the chooser.
	weightedCallSequenceChooser *randomutils.WeightedRandomChooser[calls.CallSequence]

	// callSequencesLock provides thread synchronization to prevent concurrent access errors into
	// callSequences.
	callSequencesLock sync.Mutex
}

// corpusFile represents corpus data and its state on the filesystem.
type corpusFile[T any] struct {
	// filePath describes the path the file should be written to. If blank, this indicates it has not yet been written.
	filePath string

	// data describes an object whose data should be written to the file.
	data T
}

// NewCorpus initializes a new Corpus object, reading artifacts from the provided directory. If the directory refers
// to an empty path, artifacts will not be persistently stored.
func NewCorpus(corpusDirectory string) (*Corpus, error) {
	corpus := &Corpus{
		storageDirectory:        corpusDirectory,
		coverageMaps:            coverage.NewCoverageMaps(),
		callSequences:           make([]*corpusFile[calls.CallSequence], 0),
		unexecutedCallSequences: make([]calls.CallSequence, 0),
	}

	// If we have a corpus directory set, parse it.
	if corpus.storageDirectory != "" {
		// Read all call sequences discovered in the relevant corpus directory.
		matches, err := filepath.Glob(filepath.Join(corpus.CallSequencesDirectory(), "*.json"))
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(matches); i++ {
			// Alias our file path.
			filePath := matches[i]

			// Read the call sequence data.
			b, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}

			// Parse the call sequence data.
			var seq calls.CallSequence
			err = json.Unmarshal(b, &seq)
			if err != nil {
				return nil, err
			}

			// Add entry to corpus
			corpus.callSequences = append(corpus.callSequences, &corpusFile[calls.CallSequence]{
				filePath: filePath,
				data:     seq,
			})
		}
	}

	// Initialize our weighted random chooser
	return corpus, nil
}

// CoverageMaps exposes coverage details for all call sequences known to the corpus.
func (c *Corpus) CoverageMaps() *coverage.CoverageMaps {
	return c.coverageMaps
}

// StorageDirectory returns the root directory path of the corpus. If this is empty, it indicates persistent storage
// will not be used.
func (c *Corpus) StorageDirectory() string {
	return c.storageDirectory
}

// CallSequencesDirectory returns the directory path where coverage increasing call sequences should be stored.
// This is a subdirectory of StorageDirectory. If StorageDirectory is empty, this is as well, indicating persistent
// storage will not be used.
func (c *Corpus) CallSequencesDirectory() string {
	if c.storageDirectory == "" {
		return ""
	}
	return filepath.Join(c.StorageDirectory(), "call_sequences")
}

// CallSequenceCount returns the total number of call sequences in the corpus, some of which may be inactive/not in use.
func (c *Corpus) CallSequenceCount() int {
	return len(c.callSequences)
}

// ActiveCallSequenceCount returns the count of call sequences recorded in the corpus which have been validated and are
// ready for use by RandomCallSequence.
func (c *Corpus) ActiveCallSequenceCount() int {
	if c.weightedCallSequenceChooser == nil {
		return 0
	}
	return c.weightedCallSequenceChooser.ChoiceCount()
}

// Initialize initializes any runtime data needed for a Corpus on startup. Call sequences are replayed on the post-setup
// (deployment) test chain to calculate coverage, while resolving references to compiled contracts.
func (c *Corpus) Initialize(baseTestChain *chain.TestChain, contractDefinitions contracts.Contracts) error {
	// Acquire our call sequences lock during the duration of this method.
	c.callSequencesLock.Lock()
	defer c.callSequencesLock.Unlock()

	// Initialize our call sequence structures.
	c.weightedCallSequenceChooser = randomutils.NewWeightedRandomChooser[calls.CallSequence]()
	c.unexecutedCallSequences = make([]calls.CallSequence, 0)

	// Create new coverage maps to track total coverage and a coverage tracer to do so.
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
		return fmt.Errorf("failed to initialize coverage maps, base test chain cloning encountered error: %v", err)
	}

	// Next we replay every call sequence, checking its validity on this chain and measuring coverage. If the sequence
	// is valid, we add it to our weighted list for future random selection.

	// Cache current HeadBlockNumber so that you can reset back to it after every sequence
	baseBlockNumber := testChain.HeadBlockNumber()

	// Loop for each sequence
	for _, sequenceFileData := range c.callSequences {
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
			if currentSequenceElement.Call.MsgTo == nil {
				return currentSequenceElement, nil
			}

			// We are calling a contract with this call, ensure we can resolve the contract call is targeting.
			resolvedContract, resolvedContractExists := deployedContracts[*currentSequenceElement.Call.MsgTo]
			if !resolvedContractExists {
				sequenceInvalidError = fmt.Errorf("contract at address '%v' could not be resolved", currentSequenceElement.Call.MsgTo.String())
				return nil, nil
			}
			currentSequenceElement.Contract = resolvedContract

			// Next, if our sequence element uses ABI values to produce call data, our deserialized data is not yet
			// sufficient for runtime use, until we use it to resolve runtime references.
			callAbiValues := currentSequenceElement.Call.MsgDataAbiValues
			if callAbiValues != nil {
				sequenceInvalidError = callAbiValues.Resolve(currentSequenceElement.Contract.CompiledContract().Abi)
				if sequenceInvalidError != nil {
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
			_, covErr := c.coverageMaps.Update(covMaps)
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

		// If the sequence was replayed successfully, we add a weighted choice for it, for future selection. If it was
		// not, we simply exclude it from our chooser and print a warning.
		if sequenceInvalidError == nil {
			c.weightedCallSequenceChooser.AddChoices(randomutils.NewWeightedRandomChoice[calls.CallSequence](sequence, big.NewInt(1)))
			c.unexecutedCallSequences = append(c.unexecutedCallSequences, sequence)
		} else {
			fmt.Printf("corpus item '%v' disabled due to error when replaying it: %v\n", sequenceFileData.filePath, sequenceInvalidError)
		}

		// Revert chain state to our starting point to test the next sequence.
		err = testChain.RevertToBlockNumber(baseBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to reset the chain while seeding coverage: %v\n", err)
		}
	}
	return nil
}

// AddCallSequence adds a call sequence to the corpus and returns an error in case of an issue
func (c *Corpus) AddCallSequence(seq calls.CallSequence, weight *big.Int, flushImmediately bool) error {
	// Acquire a thread lock during modification of call sequence lists.
	c.callSequencesLock.Lock()

	// Check if call sequence has been added before, if so, exit without any action.
	seqHash, err := seq.Hash()
	if err != nil {
		return err
	}

	// Verify no existing corpus item hash this same hash.
	for _, existingSeq := range c.callSequences {
		// Calculate the existing sequence hash
		existingSeqHash, err := existingSeq.data.Hash()
		if err != nil {
			return err
		}

		// Verify it is unique, if it is not, we quit immediately to avoid duplicate sequences being added.
		if bytes.Equal(existingSeqHash[:], seqHash[:]) {
			c.callSequencesLock.Unlock()
			return nil
		}
	}

	// Update our sequences with the new entry.
	c.callSequences = append(c.callSequences, &corpusFile[calls.CallSequence]{
		filePath: "",
		data:     seq,
	})

	// If we have initialized a chooser, add our call sequence item to it.
	if c.weightedCallSequenceChooser != nil {
		if weight == nil {
			weight = big.NewInt(1)
		}
		c.weightedCallSequenceChooser.AddChoices(randomutils.NewWeightedRandomChoice[calls.CallSequence](seq, weight))
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

// AddCallSequenceIfCoverageChanged checks if the most recent call executed in the provided call sequence achieved
// coverage the Corpus did not with any of its call sequences. If it did, the call sequence is added to the corpus
// and the Corpus coverage maps are updated accordingly.
// Returns an error if one occurs.
func (c *Corpus) AddCallSequenceIfCoverageChanged(callSequence calls.CallSequence, weight *big.Int, flushImmediately bool) error {
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
	coverageUpdated, err := c.coverageMaps.Update(lastMessageCoverageMaps)
	if err != nil {
		return err
	}
	if coverageUpdated {
		// New coverage has been found with this call sequence, so we add it to the corpus.
		err = c.AddCallSequence(callSequence, weight, flushImmediately)
		if err != nil {
			return err
		}
	}
	return nil
}

// RandomCallSequence returns a weighted random call sequence from the Corpus, or an error if one occurs.
func (c *Corpus) RandomCallSequence() (calls.CallSequence, error) {
	// If we didn't initialize a chooser, return an error
	if c.weightedCallSequenceChooser == nil {
		return nil, fmt.Errorf("corpus could not return a random call sequence because the corpus was not initialized")
	}

	// Pick a random call sequence, then clone it before returning it, so the original is untainted.
	seq, err := c.weightedCallSequenceChooser.Choose()
	if seq == nil || err != nil {
		return nil, err
	}
	return seq.Clone()
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

	// Ensure the corpus directories exists.
	err := utils.MakeDirectory(c.storageDirectory)
	if err != nil {
		return err
	}
	err = utils.MakeDirectory(c.CallSequencesDirectory())
	if err != nil {
		return err
	}

	// Write all call sequences to disk
	// TODO: This can be optimized by storing/indexing unwritten sequences separately and only iterating over those.
	for _, sequenceFile := range c.callSequences {
		if sequenceFile.filePath == "" {
			// Determine the file path to write this to.
			sequenceFile.filePath = filepath.Join(c.CallSequencesDirectory(), uuid.New().String()+".json")

			// Marshal the call sequence
			jsonEncodedData, err := json.MarshalIndent(sequenceFile.data, "", " ")
			if err != nil {
				return err
			}

			// Write the JSON encoded data.
			err = os.WriteFile(sequenceFile.filePath, jsonEncodedData, os.ModePerm)
			if err != nil {
				return fmt.Errorf("An error occurred while writing call sequence to disk: %v\n", err)
			}
		}
	}
	return nil
}
