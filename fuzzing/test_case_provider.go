package fuzzing

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/fuzzing/types"
)

// TestCaseProvider is an interface for a provider which can report and update the status of test cases at given points
// during a fuzzing.Fuzzer's execution.
type TestCaseProvider interface {
	// OnFuzzerStarting is called by the fuzzing.Fuzzer upon the start of a fuzzing campaign. Any previously recorded
	// TestCase should be cleared from the provider and state should be reset.
	OnFuzzerStarting(fuzzer *Fuzzer)

	// OnFuzzerStopping is called when a fuzzing.Fuzzer's campaign is being stopped. Any TestCase which is still in a
	// running state should be updated during this step and put into a finalized state.
	OnFuzzerStopping()

	// OnWorkerCreated is called when a new fuzzing.FuzzerWorker is created by the fuzzing.Fuzzer.
	OnWorkerCreated(worker *FuzzerWorker)

	// OnWorkerDestroyed is called when a previously created fuzzing.FuzzerWorker is destroyed by the fuzzing.Fuzzer.
	OnWorkerDestroyed(worker *FuzzerWorker)

	// OnWorkerDeployedContractAdded is called when a fuzzing.FuzzerWorker detects a newly deployed contract in the
	// underlying TestNode. If the  contract could be matched to a definition registered with the fuzzing.Fuzzer,
	// it is provided as well. Otherwise, a nil contract definition is supplied.
	OnWorkerDeployedContractAdded(worker *FuzzerWorker, contractAddress common.Address, contract *types.Contract)

	// OnWorkerDeployedContractDeleted is called when a fuzzing.FuzzerWorker detects a previously reported deployed
	// contract that no longer exists in the underlying TestNode.
	OnWorkerDeployedContractDeleted(worker *FuzzerWorker, contractAddress common.Address, contract *types.Contract)

	// OnWorkerCallSequenceCallTested is called after a fuzzing.FuzzerWorker sends another call in a types.CallSequence
	// during a fuzzing campaign. It returns a ShrinkCallSequenceRequest set, which represents a set of requests for
	// shrunken call sequences alongside verifiers to guide the shrinking process. This signals to the FuzzerWorker
	// that this current call sequence was interesting, and it should stop building on it and find a shrunken
	// sequence that satisfies the conditions specified by the ShrinkCallSequenceRequest, before generating
	// entirely new call sequences. Thus, this method should only return ShrinkCallSequenceRequest instances
	// when it "found a result" (e.g., call sequence that violates some property). Returning one each
	// time will mean no call sequence is continued to be built upon after the initial call.
	OnWorkerCallSequenceCallTested(worker *FuzzerWorker, callSequence types.CallSequence) []ShrinkCallSequenceRequest
}

// ShrinkCallSequenceRequest is a structure signifying a request for a shrunken call sequence from the FuzzerWorker.
type ShrinkCallSequenceRequest struct {
	// VerifierFunction is a method is called upon by a FuzzerWorker to check if a shrunken call sequence satisfies
	// the needs of an original method.
	VerifierFunction func(worker *FuzzerWorker, callSequence types.CallSequence) bool
	FinishedCallback func(worker *FuzzerWorker, shrunkenCallSequence types.CallSequence)
}
