package fuzzing

// OnFuzzerStarting describes an event where a new fuzzer object has been created with the necessary initial setup
// (e.g. chain has been created, corpus has been created) but the workers have been not created and fuzzing loop
// has not started
type OnFuzzerStarting struct {
	// Fuzzer represents a pointer to the newly instantiated fuzzer object
	Fuzzer *Fuzzer
}

// TODO: Can add all fuzzer-associated events / callbacks here
