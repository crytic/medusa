package fuzzing

import (
	"github.com/trailofbits/medusa/events"
)

// FuzzerEvents defines event emitters for a Fuzzer.
type FuzzerEvents struct {
	// FuzzerStarting emits events when the Fuzzer initialized state and is ready to about to begin the main
	// execution loop for the fuzzing campaign.
	FuzzerStarting events.EventEmitter[FuzzerStartingEvent]

	// FuzzerStopping emits events when the Fuzzer is exiting its main fuzzing loop.
	FuzzerStopping events.EventEmitter[FuzzerStoppingEvent]

	// WorkerCreated emits events when the Fuzzer creates a new FuzzerWorker during the fuzzing campaign.
	WorkerCreated events.EventEmitter[FuzzerWorkerCreatedEvent]

	// WorkerDestroyed emits events when the Fuzzer destroys an existing FuzzerWorker during the fuzzing
	// campaign. This can occur even if a fuzzing campaign is not stopping, if a worker has reached resource limits.
	WorkerDestroyed events.EventEmitter[FuzzerWorkerDestroyedEvent]
}

// FuzzerStartingEvent describes an event where a fuzzing.Fuzzer has initialized all state variables and is about to
// begin spinning up instances of FuzzerWorker to start the fuzzing campaign.
type FuzzerStartingEvent struct {
	// Fuzzer represents the instance of the fuzzing.Fuzzer for which the event occurred.
	Fuzzer *Fuzzer
}

// FuzzerStoppingEvent describes an event where a fuzzing.Fuzzer is exiting the main fuzzing loop.
type FuzzerStoppingEvent struct {
	// Fuzzer represents the instance of the fuzzing.Fuzzer for which the event occurred.
	Fuzzer *Fuzzer

	// err describes a potential error returned by the fuzzer run.
	err error
}

// FuzzerWorkerCreatedEvent describes an event where a fuzzing.FuzzerWorker is created by a fuzzing.Fuzzer.
type FuzzerWorkerCreatedEvent struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker
}

// FuzzerWorkerDestroyedEvent describes an event where a fuzzing.FuzzerWorker is destroyed by a fuzzing.Fuzzer.
type FuzzerWorkerDestroyedEvent struct {
	// Worker represents the instance of the fuzzing.FuzzerWorker for which the event occurred.
	Worker *FuzzerWorker
}
