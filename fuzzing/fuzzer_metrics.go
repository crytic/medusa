package fuzzing

// FuzzerMetrics represents a struct tracking metrics for a Fuzzer run.
type FuzzerMetrics struct {
	// workerMetrics describes the metrics for each individual worker. This expands as needed and some slots may be nil
	// while workers are initializing, as it corresponds to the indexes in Fuzzer.workers.
	workerMetrics []fuzzerWorkerMetrics
}

// fuzzerWorkerMetrics represents metrics for a single FuzzerWorker instance.
type fuzzerWorkerMetrics struct {
	// sequencesTested describes the amount of sequences of transactions which tests were run against.
	sequencesTested uint64

	// callsTested describes the amount of transactions/calls the fuzzer executed and ran tests against.
	callsTested uint64

	// workerStartupCount describes the amount of times the worker was generated, or re-generated for this index.
	workerStartupCount uint64
}

// newFuzzerMetrics obtains a new FuzzerMetrics struct for a given number of workers specified by workerCount.
// Returns the new FuzzerMetrics object.
func newFuzzerMetrics(workerCount int) *FuzzerMetrics {
	// Create a new metrics struct and return it with as many slots as required.
	metrics := FuzzerMetrics{
		workerMetrics: make([]fuzzerWorkerMetrics, workerCount),
	}
	return &metrics
}

// SequencesTested returns the amount of sequences of transactions the fuzzer executed and ran tests against.
func (m *FuzzerMetrics) SequencesTested() uint64 {
	sequencesTested := uint64(0)
	for _, workerMetrics := range m.workerMetrics {
		sequencesTested += workerMetrics.sequencesTested
	}
	return sequencesTested
}

// CallsTested returns the amount of transactions/calls the fuzzer executed and ran tests against.
func (m *FuzzerMetrics) CallsTested() uint64 {
	transactionsTested := uint64(0)
	for _, workerMetrics := range m.workerMetrics {
		transactionsTested += workerMetrics.callsTested
	}
	return transactionsTested
}

// WorkerStartupCount describes the amount of times the worker was spawned for this index. Workers are periodically
// reset.
func (m *FuzzerMetrics) WorkerStartupCount() uint64 {
	workerStartupCount := uint64(0)
	for _, workerMetrics := range m.workerMetrics {
		workerStartupCount += workerMetrics.workerStartupCount
	}
	return workerStartupCount
}
