package fuzzing

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/fuzzing/types"
)

// OnFuzzerStarting describes an event where a fuzzing.Fuzzer has initialized all state variables and is about to
// begin spinning up instances of FuzzerWorker to start the fuzzing campaign.
type OnFuzzerStarting struct {
	// Fuzzer represents the instance of the fuzzing.Fuzzer for which the event occurred.
	Fuzzer *Fuzzer
}

// OnFuzzerStopping describes an event where a fuzzing.Fuzzer is exiting its main fuzzing loop.
type OnFuzzerStopping struct {
	// Fuzzer represents the instance of the fuzzing.Fuzzer for which the event occurred.
	Fuzzer *Fuzzer
}

// OnWorkerCreated describes an event where a fuzzing.FuzzerWorker is created by a fuzzing.Fuzzer.
type OnWorkerCreated struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker
}

// OnWorkerDestroyed describes an event where a fuzzing.FuzzerWorker is destroyed by a fuzzing.Fuzzer.
type OnWorkerDestroyed struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker
}

// OnWorkerDeployedContractAdded describes an event where a fuzzing.FuzzerWorker detects a newly deployed contract in
// its underlying test chain.
type OnWorkerDeployedContractAdded struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker

	// ContractAddress describes the address of the deployed contract.
	ContractAddress common.Address

	// ContractDefinition describes the compiled contract artifact definition which the fuzzing.Fuzzer matched to the
	// deployed bytecode. If this could not be resolved, a nil value is provided.
	ContractDefinition *types.Contract
}

// OnWorkerDeployedContractDeleted describes an event where a fuzzing.FuzzerWorker detects a previously reported
// deployed contract that no longer exists in the underlying test chain.
type OnWorkerDeployedContractDeleted struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker

	// ContractAddress describes the address of the deployed contract.
	ContractAddress common.Address

	// ContractDefinition describes the compiled contract artifact definition which the fuzzing.Fuzzer matched to the
	// deployed bytecode. If this could not be resolved, a nil value is provided.
	ContractDefinition *types.Contract
}

// OnWorkerCallSequenceTesting describes an event where a fuzzing.FuzzerWorker is about to generate and test a new call
// sequence.
type OnWorkerCallSequenceTesting struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker
}

// OnWorkerCallSequenceTested describes an event where a fuzzing.FuzzerWorker has finished generating and testing a new
// call sequence.
type OnWorkerCallSequenceTested struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker
}
