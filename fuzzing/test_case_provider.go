package fuzzing

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/fuzzing/types"
)

// ShrinkCallSequenceVerifier is called repetitively by a fuzzing.FuzzerWorker with varying call sequences to discover
// a shrunken call sequence which satisfies some requirement. Returns a boolean indicating whether the call sequence
// provided satisfies the requirement.
type ShrinkCallSequenceVerifier func(worker *FuzzerWorker, callSequenceToVerify types.CallSequence) bool

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

	// OnWorkerTestedCall is called after a fuzzing.FuzzerWorker sends another call in a types.CallSequence during
	// a fuzzing campaign. It returns a ShrinkCallSequenceVerifier for each request it makes for a shrunk call sequence
	// that is derived from this one.
	OnWorkerTestedCall(worker *FuzzerWorker, callSequence types.CallSequence) []ShrinkCallSequenceVerifier
}
