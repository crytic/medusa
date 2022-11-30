package corpus

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/fuzzing/coverage"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
)

// MeasureCorpusCoverage takes a test chain in its post-setup (deployment), pre-fuzzing campaign state, and a corpus,
// and returns coverage maps which represent the coverage achieved when replaying corpus call sequences over the
// provided chain.
func MeasureCorpusCoverage(baseTestChain *chain.TestChain, corpus *Corpus) (*coverage.CoverageMaps, error) {
	// Create our coverage maps and a coverage tracer
	coverageMaps := coverage.NewCoverageMaps()
	coverageTracer := coverage.NewCoverageTracer()

	// Clone our test chain with our coverage tracer.
	testChain, err := baseTestChain.Clone([]vm.EVMLogger{coverageTracer}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize coverage maps, base test chain cloning encountered error: %v", err)
	}

	// Next we measure coverage for every corpus call sequence.
	corpusCallSequences := corpus.CallSequences()

	// Cache current HeadBlockNumber so that you can reset back to it after every sequence
	baseBlockNumber := testChain.HeadBlockNumber()

	for _, sequence := range corpusCallSequences {
		// Execute each call sequence, collecting coverage and updating it along the way
		_, err = sequence.ExecuteOnChain(testChain, true, nil, func(index int) (bool, error) {
			// Update our coverage maps for each call executed in our sequence.
			covMaps := coverage.GetCoverageTracerResults(sequence[index].ChainReference.MessageResults())
			_, covErr := coverageMaps.Update(covMaps)
			if covErr != nil {
				return true, err
			}
			return false, nil
		})

		// If we failed to replay a sequence and measure coverage, report it.
		if err != nil {
			return nil, fmt.Errorf("failed to initialize coverage maps from corpus, encountered an error while executing call sequence: %v\n", err)
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
