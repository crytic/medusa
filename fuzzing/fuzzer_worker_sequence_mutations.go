package fuzzing

import (
	"fmt"
	"github.com/trailofbits/medusa/fuzzing/calls"
	"github.com/trailofbits/medusa/fuzzing/corpus"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
)

func GenerateMutatedCallSequence(corpus *corpus.Corpus, valueGenerator valuegeneration.ValueGenerator, callSequenceLength int) (calls.CallSequence, error) {
	// Validate our supplied length.
	if callSequenceLength <= 0 {
		return nil, fmt.Errorf("could not generate a mutated call sequence as the target length should at least by 1, but %v was provided", callSequenceLength)
	}

	// Create our sequence of the given length.
	sequence := make(calls.CallSequence, callSequenceLength)

	// Next we'll determine a random strategy to use for our call sequence mutation methods.
	_ = sequence

	// Determine our strategy.
	return nil, nil
}
