package fuzzing

import "github.com/trailofbits/medusa/fuzzing/tracing"

// FuzzerMetrics represents a struct tracking metrics for a Fuzzer run.
type FuzzerMetrics struct {
	// coverageMaps describes the code coverage achieved across all transactions run by every FuzzerWorker.
	coverageMaps *tracing.CoverageMaps

	// workerMetrics describes the metrics for each individual worker. This expands as needed and some slots may be nil
	// while workers are initializing, as it corresponds to the indexes in Fuzzer.workers.
	workerMetrics []fuzzerWorkerMetrics
}

// fuzzerWorkerMetrics represents metrics for a single FuzzerWorker instance.
type fuzzerWorkerMetrics struct {
	// sequencesTested describes the amount of sequences of transactions which property tests were verified against.
	sequencesTested uint64

	// transactionsTested describes the amount of transactions which property tests were verified against.
	transactionsTested uint64

	// workerStartupCount describes the amount of times the worker was generated, or re-generated for this index.
	workerStartupCount uint64
}

// newFuzzerMetrics obtains a new FuzzerMetrics struct for a given number of workers specified by workerCount.
// Returns the new FuzzerMetrics object.
func newFuzzerMetrics(workerCount int) *FuzzerMetrics {
	// Create a new metrics struct and return it with as many slots as required.
	metrics := FuzzerMetrics{
		coverageMaps:  tracing.NewCoverageMaps(),
		workerMetrics: make([]fuzzerWorkerMetrics, workerCount),
	}
	return &metrics
}

// SequencesTested returns the amount of sequences of transactions which property tests were verified against across
// all workers.
func (m *FuzzerMetrics) SequencesTested() uint64 {
	sequencesTested := uint64(0)
	for _, workerMetrics := range m.workerMetrics {
		sequencesTested += workerMetrics.sequencesTested
	}
	return sequencesTested
}

// TransactionsTested returns the amount of transactions which ran and then property tests were executed against across
// all workers.
func (m *FuzzerMetrics) TransactionsTested() uint64 {
	transactionsTested := uint64(0)
	for _, workerMetrics := range m.workerMetrics {
		transactionsTested += workerMetrics.transactionsTested
	}
	return transactionsTested
}

// WorkerStartupCount describes the amount of times the worker was generated, or re-generated for this index.
// This could happen due cases such as hitting memory constraints where re-generation frees resources.
func (m *FuzzerMetrics) WorkerStartupCount() uint64 {
	workerStartupCount := uint64(0)
	for _, workerMetrics := range m.workerMetrics {
		workerStartupCount += workerMetrics.workerStartupCount
	}
	return workerStartupCount
}
