package fuzzing

import (
	"github.com/trailofbits/medusa/chain"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
)

// FuzzerHooks defines the hooks that can be used for the Fuzzer on an API level.
type FuzzerHooks struct {
	// NewValueGeneratorFunc describes the function to use to set up a new value generator per worker.
	NewValueGeneratorFunc NewValueGeneratorFunc
	// ChainSetupFunc describes the function to use to set up a new test chain's initial state prior to fuzzing.
	ChainSetupFunc TestChainSetupFunc

	// CallSequenceTestFuncs describes a list of functions to be called upon by a FuzzerWorker after every call
	// in a call sequence.
	CallSequenceTestFuncs []CallSequenceTestFunc
}

// NewValueGeneratorFunc defines a method which is called to create a valuegeneration.ValueGenerator for a worker
// when it is created. It takes the current fuzzer as an argument for context, and is expected to return a generator,
// or an error if one is encountered.
type NewValueGeneratorFunc func(fuzzer *Fuzzer, valueSet *valuegeneration.ValueSet) (valuegeneration.ValueGenerator, error)

// TestChainSetupFunc describes a function which sets up a test chain's initial state prior to fuzzing.
type TestChainSetupFunc func(fuzzer *Fuzzer, testChain *chain.TestChain) error

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
