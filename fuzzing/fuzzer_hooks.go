package fuzzing

import (
	"github.com/crytic/medusa/fuzzing/executiontracer"
	"math/rand"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
)

// FuzzerHooks defines the hooks that can be used for the Fuzzer on an API level.
type FuzzerHooks struct {
	// NewCallSequenceGeneratorConfigFunc describes the function to use to set up a new CallSequenceGeneratorConfig,
	// defining parameters for a new FuzzerWorker's CallSequenceGenerator.
	// The value generator provided must be either thread safe, or a new instance must be provided per invocation to
	// avoid concurrent access issues between workers.
	NewCallSequenceGeneratorConfigFunc NewCallSequenceGeneratorConfigFunc

	// NewShrinkingValueMutatorFunc describes the function used to set up a value mutator used to shrink call
	// values in the fuzzer's call sequence shrinking process.
	// The value mutator provided must be either thread safe, or a new instance must be provided per invocation to
	// avoid concurrent access issues between workers.
	NewShrinkingValueMutatorFunc NewShrinkingValueMutatorFunc

	// ChainSetupFunc describes the function to use to set up a new test chain's initial state prior to fuzzing.
	ChainSetupFunc TestChainSetupFunc

	// CallSequenceTestFuncs describes a list of functions to be called upon by a FuzzerWorker after every call
	// in a call sequence.
	CallSequenceTestFuncs []CallSequenceTestFunc
}

// NewShrinkingValueMutatorFunc describes the function used to set up a value mutator used to shrink call
// values in the fuzzer's call sequence shrinking process.
// Returns a new value mutator, or an error if one occurred.
type NewShrinkingValueMutatorFunc func(fuzzer *Fuzzer, valueSet *valuegeneration.ValueSet, randomProvider *rand.Rand) (valuegeneration.ValueMutator, error)

// NewCallSequenceGeneratorConfigFunc defines a method is called to create a new CallSequenceGeneratorConfig, defining
// the parameters for the new FuzzerWorker to use when creating its CallSequenceGenerator used to power fuzzing.
// Returns a new CallSequenceGeneratorConfig, or an error if one is encountered.
type NewCallSequenceGeneratorConfigFunc func(fuzzer *Fuzzer, valueSet *valuegeneration.ValueSet, randomProvider *rand.Rand) (*CallSequenceGeneratorConfig, error)

// TestChainSetupFunc describes a function which sets up a test chain's initial state prior to fuzzing.
// An execution trace can also be returned in case of a deployment error for an improved debugging experience
type TestChainSetupFunc func(fuzzer *Fuzzer, testChain *chain.TestChain) (error, *executiontracer.ExecutionTrace)

// CallSequenceTestFunc defines a method called after a fuzzing.FuzzerWorker sends another call in a types.CallSequence
// during a fuzzing campaign. It returns a ShrinkCallSequenceRequest set, which represents a set of requests for
// shrunken call sequences alongside verifiers to guide the shrinking process. This signals to the FuzzerWorker
// that this current call sequence was interesting, and it should stop building on it and find a shrunken
// sequence that satisfies the conditions specified by the ShrinkCallSequenceRequest, before generating
// entirely new call sequences. Shrink requests should not be unnecessarily requested, as this will cancel the
// current call sequence from being further generated and tested.
type CallSequenceTestFunc func(worker *FuzzerWorker, callSequence calls.CallSequence) ([]ShrinkCallSequenceRequest, error)

// ShrinkCallSequenceRequest is a structure signifying a request for a shrunken call sequence from the FuzzerWorker.
type ShrinkCallSequenceRequest struct {
	// VerifierFunction is a method is called upon by a FuzzerWorker to check if a shrunken call sequence satisfies
	// the needs of an original method.
	VerifierFunction func(worker *FuzzerWorker, callSequence calls.CallSequence) (bool, error)
	// FinishedCallback is a method called upon when the shrink request has concluded. It provides the finalized
	// shrunken call sequence.
	FinishedCallback func(worker *FuzzerWorker, shrunkenCallSequence calls.CallSequence) error
	// RecordResultInCorpus indicates whether the shrunken call sequence should be recorded in the corpus. If so, when
	// the shrinking operation is completed, the sequence will be added to the corpus if it doesn't already exist.
	RecordResultInCorpus bool
}
