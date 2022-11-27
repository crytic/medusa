package fuzzing

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/events"
	"github.com/trailofbits/medusa/fuzzing/types"
)

// FuzzerWorkerEvents defines event emitters for a FuzzerWorker.
type FuzzerWorkerEvents struct {
	// ContractAdded emits events when the FuzzerWorker detects a newly deployed contract
	// on its underlying chain.
	ContractAdded events.EventEmitter[FuzzerWorkerContractAddedEvent]

	// ContractDeleted emits events when the FuzzerWorker detects a deployed contract no
	// longer exists on its underlying chain.
	ContractDeleted events.EventEmitter[FuzzerWorkerContractDeletedEvent]

	// CallSequenceTesting emits events when the FuzzerWorker is about to generate and test a new
	// call sequence.
	CallSequenceTesting events.EventEmitter[FuzzerWorkerCallSequenceTestingEvent]

	// CallSequenceTested emits events when the FuzzerWorker has finished generating and testing a
	// new call sequence.
	CallSequenceTested events.EventEmitter[FuzzerWorkerCallSequenceTestedEvent]
}

// FuzzerWorkerContractAddedEvent describes an event where a fuzzing.FuzzerWorker detects a newly deployed contract in
// its underlying test chain.
type FuzzerWorkerContractAddedEvent struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker

	// ContractAddress describes the address of the deployed contract.
	ContractAddress common.Address

	// ContractDefinition describes the compiled contract artifact definition which the fuzzing.Fuzzer matched to the
	// deployed bytecode. If this could not be resolved, a nil value is provided.
	ContractDefinition *types.Contract
}

// FuzzerWorkerContractDeletedEvent describes an event where a fuzzing.FuzzerWorker detects a previously reported
// deployed contract that no longer exists in the underlying test chain.
type FuzzerWorkerContractDeletedEvent struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker

	// ContractAddress describes the address of the deployed contract.
	ContractAddress common.Address

	// ContractDefinition describes the compiled contract artifact definition which the fuzzing.Fuzzer matched to the
	// deployed bytecode. If this could not be resolved, a nil value is provided.
	ContractDefinition *types.Contract
}

// FuzzerWorkerCallSequenceTestingEvent describes an event where a fuzzing.FuzzerWorker is about to generate and test a new call
// sequence.
type FuzzerWorkerCallSequenceTestingEvent struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker
}

// FuzzerWorkerCallSequenceTestedEvent describes an event where a fuzzing.FuzzerWorker has finished generating and testing a new
// call sequence.
type FuzzerWorkerCallSequenceTestedEvent struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker
}
