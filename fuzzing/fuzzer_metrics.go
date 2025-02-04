package fuzzing

import "math/big"

// FuzzerMetrics represents a struct tracking metrics for a Fuzzer run.
type FuzzerMetrics struct {
	// workerMetrics describes the metrics for each individual worker. This expands as needed and some slots may be nil
	// while workers are initializing, as it corresponds to the indexes in Fuzzer.workers.
	workerMetrics []fuzzerWorkerMetrics
}

// fuzzerWorkerMetrics represents metrics for a single FuzzerWorker instance.
type fuzzerWorkerMetrics struct {
	// sequencesTested is the amount of sequences of transactions which tests were run against.
	sequencesTested *big.Int

	// failedSequences is the amount of sequences of transactions which tests failed.
	failedSequences *big.Int

	// callsTested is the amount of transactions/calls the fuzzer executed and ran tests against.
	callsTested *big.Int

	// gasUsed is the amount of gas the fuzzer executed and ran tests against.
	gasUsed *big.Int

	// workerStartupCount is the amount of times the worker was generated, or re-generated for this index.
	workerStartupCount *big.Int

	// shrinking indicates whether the fuzzer worker is currently shrinking.
	shrinking bool
}

// newFuzzerMetrics obtains a new FuzzerMetrics struct for a given number of workers specified by workerCount.
// Returns the new FuzzerMetrics object.
func newFuzzerMetrics(workerCount int) *FuzzerMetrics {
	// Create a new metrics struct and return it with as many slots as required.
	metrics := FuzzerMetrics{
		workerMetrics: make([]fuzzerWorkerMetrics, workerCount),
	}
	for i := 0; i < len(metrics.workerMetrics); i++ {
		metrics.workerMetrics[i].sequencesTested = big.NewInt(0)
		metrics.workerMetrics[i].failedSequences = big.NewInt(0)
		metrics.workerMetrics[i].callsTested = big.NewInt(0)
		metrics.workerMetrics[i].workerStartupCount = big.NewInt(0)
		metrics.workerMetrics[i].gasUsed = big.NewInt(0)
	}
	return &metrics
}

// FailedSequences returns the number of sequences that led to failures across all workers
func (m *FuzzerMetrics) FailedSequences() *big.Int {
	failedSequences := big.NewInt(0)
	for _, workerMetrics := range m.workerMetrics {
		failedSequences.Add(failedSequences, workerMetrics.failedSequences)
	}
	return failedSequences
}

// SequencesTested returns the amount of sequences of transactions the fuzzer executed and ran tests against.
func (m *FuzzerMetrics) SequencesTested() *big.Int {
	sequencesTested := big.NewInt(0)
	for _, workerMetrics := range m.workerMetrics {
		sequencesTested.Add(sequencesTested, workerMetrics.sequencesTested)
	}
	return sequencesTested
}

// CallsTested returns the amount of transactions/calls the fuzzer executed and ran tests against.
func (m *FuzzerMetrics) CallsTested() *big.Int {
	transactionsTested := big.NewInt(0)
	for _, workerMetrics := range m.workerMetrics {
		transactionsTested.Add(transactionsTested, workerMetrics.callsTested)
	}
	return transactionsTested
}

func (m *FuzzerMetrics) GasUsed() *big.Int {
	gasUsed := big.NewInt(0)
	for _, workerMetrics := range m.workerMetrics {
		gasUsed.Add(gasUsed, workerMetrics.gasUsed)
	}
	return gasUsed
}

// WorkerStartupCount describes the amount of times the worker was spawned for this index. Workers are periodically
// reset.
func (m *FuzzerMetrics) WorkerStartupCount() *big.Int {
	workerStartupCount := big.NewInt(0)
	for _, workerMetrics := range m.workerMetrics {
		workerStartupCount.Add(workerStartupCount, workerMetrics.workerStartupCount)
	}
	return workerStartupCount
}

// WorkersShrinkingCount returns the amount of workers currently performing shrinking operations.
func (m *FuzzerMetrics) WorkersShrinkingCount() uint64 {
	shrinkingCount := uint64(0)
	for _, workerMetrics := range m.workerMetrics {
		if workerMetrics.shrinking {
			shrinkingCount++
		}
	}
	return shrinkingCount
}
