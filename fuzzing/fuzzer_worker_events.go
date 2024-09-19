package fuzzing

import (
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/events"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/ethereum/go-ethereum/common"
)

// FuzzerWorkerEvents defines event emitters for a FuzzerWorker.
type FuzzerWorkerEvents struct {
	// ContractAdded emits events when the FuzzerWorker detects a newly deployed contract
	// on its underlying chain.
	ContractAdded events.EventEmitter[FuzzerWorkerContractAddedEvent]

	// ContractDeleted emits events when the FuzzerWorker detects a deployed contract no
	// longer exists on its underlying chain.
	ContractDeleted events.EventEmitter[FuzzerWorkerContractDeletedEvent]

	// FuzzerWorkerChainCreated emits events when the FuzzerWorker has created its chain and is about to begin chain
	// setup.
	FuzzerWorkerChainCreated events.EventEmitter[FuzzerWorkerChainCreatedEvent]

	// FuzzerWorkerChainSetup emits events when the FuzzerWorker has set up its chain and is about to begin fuzzing.
	FuzzerWorkerChainSetup events.EventEmitter[FuzzerWorkerChainSetupEvent]

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
	ContractDefinition *contracts.Contract
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
	ContractDefinition *contracts.Contract
}

// FuzzerWorkerChainCreatedEvent describes an event where a fuzzing.FuzzerWorker is created its underlying chain.
// This is an opportune to attach tracers to capture chain setup information.
type FuzzerWorkerChainCreatedEvent struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker

	// Chain represents the worker's chain.
	Chain *chain.TestChain
}

// FuzzerWorkerChainSetupEvent describes an event where a fuzzing.FuzzerWorker set up its underlying chain. This
// means the chain should have its initial contracts deployed and is ready for the fuzzing campaign to start.
type FuzzerWorkerChainSetupEvent struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker

	// Chain represents the worker's chain.
	Chain *chain.TestChain
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
	// Sequence represents the call sequence that was tested
	Sequence calls.CallSequence
}
