package corpus

import (
	"math/rand"

	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/coverage"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/utils"
	"github.com/crytic/medusa/utils/randomutils"
	"github.com/google/uuid"
)

// Corpus describes an archive of fuzzer-generated artifacts used to further fuzzing efforts. These artifacts are
// reusable across fuzzer runs. Changes to the fuzzer/chain configuration or definitions within smart contracts
// may create incompatibilities with corpus items.
type Corpus struct {
	// storageDirectory describes the directory to save corpus callSequenceFiles within.
	storageDirectory string

	// coverageMaps describes the total code coverage known to be achieved across all corpus call sequences.
	coverageMaps *coverage.CoverageMaps

	// callSequenceFiles represents a corpus directory with files that should be used for mutations.
	callSequenceFiles *corpusDirectory[calls.CallSequence]

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

	// initializationTotal captures the total number of corpus sequences that need to be executed to initialize the fuzzer.
	initializationTotal uint64
	// initializationProcessed tracks how many initialization sequences have been executed.
	initializationProcessed atomic.Uint64
	// initializationSuccessful tracks how many initialization sequences were executed successfully.
	initializationSuccessful atomic.Uint64
	// initializationOnce ensures the initializationDoneCallback is invoked only once.
	initializationOnce sync.Once
	// initializationDoneCallback is invoked when all initialization sequences finish execution to notify the fuzzer that the corpus has been initialized.
	initializationDoneCallback func(active uint64, total uint64)

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
		callSequenceFiles:       newCorpusDirectory[calls.CallSequence](""),
		testResultSequenceFiles: newCorpusDirectory[calls.CallSequence](""),
		unexecutedCallSequences: make([]calls.CallSequence, 0),
		logger:                  logging.GlobalLogger.NewSubLogger("module", "corpus"),
	}

	// If we have a corpus directory set, parse our call sequences.
	if corpus.storageDirectory != "" {
		// Migrate the legacy corpus structure
		// Note that it is important to call this first since we want to move all the call sequence files before reading
		// them into the corpus
		err = corpus.migrateLegacyCorpus()
		if err != nil {
			return nil, err
		}

		// Read call sequences.
		corpus.callSequenceFiles.path = filepath.Join(corpus.storageDirectory, "call_sequences")
		err = corpus.callSequenceFiles.readFiles("*.json")
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

// migrateLegacyCorpus is used to read in the legacy corpus standard where call sequences were stored in two separate
// directories (mutable/immutable).
func (c *Corpus) migrateLegacyCorpus() error {
	// Check to see if the mutable and/or the immutable directories exist
	callSequencePath := filepath.Join(c.storageDirectory, "call_sequences")
	mutablePath := filepath.Join(c.storageDirectory, "call_sequences", "mutable")
	immutablePath := filepath.Join(c.storageDirectory, "call_sequences", "immutable")

	// Only return an error if the error is something other than "filepath does not exist"
	mutableDirInfo, err := os.Stat(mutablePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	immutableDirInfo, err := os.Stat(immutablePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Return early if these directories do not exist
	if mutableDirInfo == nil && immutableDirInfo == nil {
		return nil
	}

	// Now, we need to notify the user that we have detected a legacy structure
	c.logger.Info("Migrating legacy corpus")

	// If the mutable directory exists, read in all the files and add them to the call sequence files
	if mutableDirInfo != nil {
		// Discover all corpus files in the given directory.
		filePaths, err := filepath.Glob(filepath.Join(mutablePath, "*.json"))
		if err != nil {
			return err
		}

		// Move each file from the mutable directory to the parent call_sequences directory
		for _, filePath := range filePaths {
			err = utils.MoveFile(filePath, filepath.Join(callSequencePath, filepath.Base(filePath)))
			if err != nil {
				return err
			}
		}

		// Delete the mutable directory
		err = utils.DeleteDirectory(mutablePath)
		if err != nil {
			return err
		}
	}

	// If the immutable directory exists, read in all the files and add them to the call sequence files
	if immutableDirInfo != nil {
		// Discover all corpus files in the given directory.
		filePaths, err := filepath.Glob(filepath.Join(immutablePath, "*.json"))
		if err != nil {
			return err
		}

		// Move each file from the immutable directory to the parent call_sequences directory
		for _, filePath := range filePaths {
			err = utils.MoveFile(filePath, filepath.Join(callSequencePath, filepath.Base(filePath)))
			if err != nil {
				return err
			}
		}

		// Delete the immutable directory
		err = utils.DeleteDirectory(immutablePath)
		if err != nil {
			return err
		}
	}

	return nil
}

// Initialize initializes the in-memory corpus state but does not actually replay any of the sequences stored in the corpus.
// It seeds coverage information from the post-setup chain while enqueueing all persisted sequences for execution. The fuzzer workers
// will concurrently execute all the sequences stored in the corpus and then the onComplete hook is invoked to notify the fuzzer that
// the corpus has been initialized. Returns an error if seeding fails.
func (c *Corpus) Initialize(baseTestChain *chain.TestChain, contractDefinitions contracts.Contracts, onComplete func(active uint64, total uint64)) error {
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
		newChain.AddTracer(coverageTracer.NativeTracer(), true, false)

		// We also track any contract deployments, so we can resolve contract/method definitions for corpus call
		// sequences.
		newChain.Events.ContractDeploymentAddedEventEmitter.Subscribe(func(event chain.ContractDeploymentsAddedEvent) error {
			if contractDefinitions != nil {
				matchedContract := contractDefinitions.MatchBytecode(event.Contract.InitBytecode, event.Contract.RuntimeBytecode)
				if matchedContract != nil {
					deployedContracts[event.Contract.Address] = matchedContract
				}
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
	defer testChain.Close()

	// Freeze a set of deployedContracts's keys so that we have a set of addresses present in baseTestChain.
	// Feed this set to the coverage tracer.
	initialContractsSet := make(map[common.Address]struct{}, len(deployedContracts))
	for addr := range deployedContracts {
		initialContractsSet[addr] = struct{}{}
	}
	coverageTracer.SetInitialContractsSet(&initialContractsSet)

	// Set our coverage maps to those collected when replaying all blocks when cloning.
	c.coverageMaps = coverage.NewCoverageMaps()
	for _, block := range testChain.CommittedBlocks() {
		for _, messageResults := range block.MessageResults {
			// Grab the coverage maps
			covMaps := coverage.GetCoverageTracerResults(messageResults)

			// Memory optimization: Remove the coverage maps from the message results
			coverage.RemoveCoverageTracerResults(messageResults)

			// Update the global coverage maps
			_, covErr := c.coverageMaps.Update(covMaps)
			if covErr != nil {
				return covErr
			}
		}
	}

	totalSequences := len(c.callSequenceFiles.files) + len(c.testResultSequenceFiles.files)
	c.unexecutedCallSequences = make([]calls.CallSequence, 0, totalSequences)
	for _, sequenceFileData := range c.testResultSequenceFiles.files {
		c.unexecutedCallSequences = append(c.unexecutedCallSequences, sequenceFileData.data)
	}
	for _, sequenceFileData := range c.callSequenceFiles.files {
		c.unexecutedCallSequences = append(c.unexecutedCallSequences, sequenceFileData.data)
	}

	// Reset warmup tracking counters.
	c.initializationProcessed.Store(0)
	c.initializationSuccessful.Store(0)
	c.initializationOnce = sync.Once{}
	c.initializationDoneCallback = onComplete
	c.initializationTotal = uint64(len(c.unexecutedCallSequences))

	// If there are no sequences to process, trigger the callback immediately.
	if c.initializationTotal == 0 && c.initializationDoneCallback != nil {
		c.initializationOnce.Do(func() {
			c.initializationDoneCallback(0, 0)
		})
	}

	return nil
}

// CoverageMaps exposes coverage details for all call sequences known to the corpus.
func (c *Corpus) CoverageMaps() *coverage.CoverageMaps {
	return c.coverageMaps
}

// CallSequenceEntryCount returns the total number of call sequences that increased coverage and also any test results
// that led to a failure.
func (c *Corpus) CallSequenceEntryCount() (int, int) {
	return len(c.callSequenceFiles.files), len(c.testResultSequenceFiles.files)
}

// InitializingCorpus returns true if the corpus is still initializing, false otherwise.
func (c *Corpus) InitializingCorpus() bool {
	return len(c.unexecutedCallSequences) > 0
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
		c.mutationTargetSequenceChooser.AddChoices(randomutils.NewWeightedRandomChoice(sequence, mutationChooserWeight))
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

// checkSequenceCoverageAndUpdate checks if the most recent call executed in the provided call sequence achieved
// coverage the not already included in coverageMaps. If it did, coverageMaps is updated accordingly.
// Returns a boolean indicating whether any change happened, and an error if one occurs.
func checkSequenceCoverageAndUpdate(callSequence calls.CallSequence, coverageMaps *coverage.CoverageMaps) (bool, error) {
	// If we have coverage-guided fuzzing disabled or no calls in our sequence, there is nothing to do.
	if len(callSequence) == 0 {
		return false, nil
	}

	// Obtain our coverage maps for our last call.
	lastCall := callSequence[len(callSequence)-1]
	lastCallChainReference := lastCall.ChainReference
	lastMessageResult := lastCallChainReference.Block.MessageResults[lastCallChainReference.TransactionIndex]
	lastMessageCoverageMaps := coverage.GetCoverageTracerResults(lastMessageResult)

	// If we have none, because a coverage tracer wasn't attached when processing this call, we can stop.
	if lastMessageCoverageMaps == nil {
		return false, nil
	}

	// Memory optimization: Remove them from the results now that we obtained them, to free memory later.
	coverage.RemoveCoverageTracerResults(lastMessageResult)

	// Merge the coverage maps into our total coverage maps and check if we had an update.
	return coverageMaps.Update(lastMessageCoverageMaps)
}

// CheckSequenceCoverageAndUpdate checks if the most recent call executed in the provided call sequence achieved
// coverage the Corpus did not with any of its call sequences. If it did, the call sequence is added to the corpus
// and the Corpus coverage maps are updated accordingly.
// Returns a boolean indicating whether coverage increased, and an error if one occurs.
func (c *Corpus) CheckSequenceCoverageAndUpdate(callSequence calls.CallSequence, mutationChooserWeight *big.Int, flushImmediately bool) (bool, error) {
	coverageUpdated, err := checkSequenceCoverageAndUpdate(callSequence, c.coverageMaps)
	if err != nil {
		return false, err
	}

	// If we had an increase in coverage, we save the sequence.
	if coverageUpdated {
		// If we achieved new coverage, save this sequence for mutation purposes.
		err = c.addCallSequence(c.callSequenceFiles, callSequence, true, mutationChooserWeight, flushImmediately)
		if err != nil {
			return true, err
		}
	}
	return coverageUpdated, nil
}

// MarkCorpusElementForMutation records that a corpus element has been successfully executed and can be used for mutations.
// The sequence is cloned, stripped of runtime metadata, and registered with the mutation chooser so it can participate
// in future mutations.
func (c *Corpus) MarkCorpusElementForMutation(sequence calls.CallSequence, mutationChooserWeight *big.Int) error {
	// If no weight is provided, set it to 1.
	if mutationChooserWeight == nil {
		mutationChooserWeight = big.NewInt(1)
	}

	// Add the sequence to the mutation chooser
	c.mutationTargetSequenceChooser.AddChoices(randomutils.NewWeightedRandomChoice(sequence, mutationChooserWeight))
	return nil
}

// IncrementValid records that a previously unexecuted corpus element has finished executing.
// The valid parameter should be true when the call sequence execution succeeded (even if it triggered a test failure),
// and false if it was skipped due to incompatibility or other errors.
func (c *Corpus) IncrementValid(valid bool) {
	// Guard clause
	total := c.initializationTotal
	if total == 0 {
		return
	}

	// Increment the processed counter.
	processed := c.initializationProcessed.Add(1)

	// If the call sequence execution was successful, increment the successful counter.
	if valid {
		c.initializationSuccessful.Add(1)
	}

	// If we have processed all corpus elements, invoke the completion callback.
	if processed == total {
		c.initializationOnce.Do(func() {
			// Invoke the completion callback if it is set.
			if c.initializationDoneCallback != nil {
				// Invoke the completion callback with the total number of corpus elements and the number of successful corpus elements.
				c.initializationDoneCallback(c.initializationSuccessful.Load(), total)
			}
		})
	}
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

	// Write all coverage-increasing call sequences.
	err := c.callSequenceFiles.writeFiles()
	if err != nil {
		return err
	}

	// Write test case provider related call sequences (test failures, etc).
	err = c.testResultSequenceFiles.writeFiles()
	if err != nil {
		return err
	}

	return nil
}

// PruneSequences removes unnecessary entries from the corpus. It does this by:
//   - Initialize a blank coverage map tmpMap
//   - Grab all sequences in the corpus
//   - Randomize the order
//   - For each transaction, see whether it adds anything new to tmpMap.
//     If it does, add the new coverage and continue.
//     If it doesn't, remove it from the corpus.
//
// By doing this, we hope to find a smaller set of txn sequences that still preserves our current coverage.
// PruneSequences takes a chain.TestChain parameter used to run transactions.
// It returns an int indicating the number of sequences removed from the corpus, and an error if any occurred.
func (c *Corpus) PruneSequences(ctx context.Context, chain *chain.TestChain) (int, error) {
	if c.mutationTargetSequenceChooser == nil {
		return 0, nil
	}

	chainOriginalIndex := uint64(len(chain.CommittedBlocks()))
	tmpMap := coverage.NewCoverageMaps()

	c.callSequencesLock.Lock()
	seqs := make([]calls.CallSequence, len(c.mutationTargetSequenceChooser.Choices))
	for i, seq := range c.mutationTargetSequenceChooser.Choices {
		seqCloned, err := seq.Data.Clone()
		if err != nil {
			c.callSequencesLock.Unlock()
			return 0, err
		}
		seqs[i] = seqCloned
	}
	c.callSequencesLock.Unlock()
	// We don't need to lock during the next part as long as the ordering of Choices doesn't change.
	// New items could get added in the meantime, but older items won't be touched.

	toRemove := map[int]bool{}

	// Iterate seqs in a random order
	for _, i := range rand.Perm(len(seqs)) {
		if utils.CheckContextDone(ctx) {
			return 0, nil
		}

		seq := seqs[i]

		fetchElementFunc := func(currentIndex int) (*calls.CallSequenceElement, error) {
			if currentIndex >= len(seq) {
				return nil, nil
			}
			return seq[currentIndex], nil
		}

		// Never quit early
		executionCheckFunc := func(currentlyExecutedSequence calls.CallSequence) (bool, error) { return false, nil }

		seq, err := calls.ExecuteCallSequenceIteratively(chain, fetchElementFunc, executionCheckFunc)
		if err != nil {
			return 0, err
		}

		coverageUpdated, err := checkSequenceCoverageAndUpdate(seq, tmpMap)
		if err != nil {
			return 0, err
		}

		if !coverageUpdated {
			// No new coverage was added. We can remove this from the corpus.
			toRemove[i] = true
		}

		err = chain.RevertToBlockIndex(chainOriginalIndex)
		if err != nil {
			return 0, err
		}
	}

	c.mutationTargetSequenceChooser.RemoveChoices(toRemove)
	return len(toRemove), nil
}
