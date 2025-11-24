package fuzzing

import (
	"fmt"
	"sync/atomic"
	"time"
)

// WorkerState represents the current state of a fuzzer worker
type WorkerState int32

const (
	// WorkerStateIdle indicates the worker is not actively processing
	WorkerStateIdle WorkerState = iota
	// WorkerStateGenerating indicates the worker is generating new call sequences
	WorkerStateGenerating
	// WorkerStateReplayingCorpus indicates the worker is replaying corpus entries
	WorkerStateReplayingCorpus
	// WorkerStateShrinking indicates the worker is shrinking a failing test case
	WorkerStateShrinking
)

// String returns a human-readable string representation of the worker state
func (s WorkerState) String() string {
	switch s {
	case WorkerStateGenerating:
		return "Generating"
	case WorkerStateReplayingCorpus:
		return "Replaying Corpus"
	case WorkerStateShrinking:
		return "Shrinking"
	case WorkerStateIdle:
		return "Idle"
	default:
		return "Unknown"
	}
}

// WorkerActivity tracks detailed activity information for a worker.
// All fields use atomic operations to ensure thread-safe access without locks.
type WorkerActivity struct {
	// state is the current state of the worker (atomic access)
	state atomic.Int32

	// strategy describes the current generation strategy (e.g., "splice", "mutate")
	// Stored as atomic.Value containing a string
	strategy atomic.Value

	// corpusEntryIndex tracks which corpus entry is being replayed (-1 if not replaying)
	corpusEntryIndex atomic.Int32

	// shrinkIteration tracks the current shrinking iteration
	shrinkIteration atomic.Int32

	// shrinkLimit tracks the maximum number of shrinking iterations
	shrinkLimit atomic.Int32

	// lastUpdate tracks when this activity was last updated (Unix timestamp)
	lastUpdate atomic.Int64
}

// NewWorkerActivity creates a new WorkerActivity tracker initialized to idle state
func NewWorkerActivity() *WorkerActivity {
	wa := &WorkerActivity{}
	wa.state.Store(int32(WorkerStateIdle))
	wa.corpusEntryIndex.Store(-1)
	wa.strategy.Store("")
	wa.shrinkIteration.Store(0)
	wa.shrinkLimit.Store(0)
	wa.lastUpdate.Store(time.Now().Unix())
	return wa
}

// SetState atomically sets the worker state and updates the last update timestamp
func (wa *WorkerActivity) SetState(state WorkerState) {
	wa.state.Store(int32(state))
	wa.lastUpdate.Store(time.Now().Unix())
}

// GetState atomically gets the current worker state
func (wa *WorkerActivity) GetState() WorkerState {
	return WorkerState(wa.state.Load())
}

// SetStrategy atomically sets the generation strategy and updates the timestamp
func (wa *WorkerActivity) SetStrategy(strategy string) {
	wa.strategy.Store(strategy)
	wa.lastUpdate.Store(time.Now().Unix())
}

// GetStrategy atomically gets the current generation strategy
func (wa *WorkerActivity) GetStrategy() string {
	val := wa.strategy.Load()
	if val == nil {
		return ""
	}
	return val.(string)
}

// SetCorpusReplay marks the worker as replaying a specific corpus entry
func (wa *WorkerActivity) SetCorpusReplay(entryIndex int) {
	wa.SetState(WorkerStateReplayingCorpus)
	wa.corpusEntryIndex.Store(int32(entryIndex))
}

// GetCorpusEntryIndex atomically gets the current corpus entry index being replayed
func (wa *WorkerActivity) GetCorpusEntryIndex() int {
	return int(wa.corpusEntryIndex.Load())
}

// ClearCorpusReplay clears corpus replay tracking
func (wa *WorkerActivity) ClearCorpusReplay() {
	wa.corpusEntryIndex.Store(-1)
}

// SetShrinking marks the worker as shrinking with the given iteration and limit
func (wa *WorkerActivity) SetShrinking(iteration, limit int) {
	wa.SetState(WorkerStateShrinking)
	wa.shrinkIteration.Store(int32(iteration))
	wa.shrinkLimit.Store(int32(limit))
}

// GetShrinkProgress atomically gets the current shrinking progress
func (wa *WorkerActivity) GetShrinkProgress() (iteration int, limit int) {
	return int(wa.shrinkIteration.Load()), int(wa.shrinkLimit.Load())
}

// ClearShrinking clears shrinking state
func (wa *WorkerActivity) ClearShrinking() {
	wa.shrinkIteration.Store(0)
	wa.shrinkLimit.Store(0)
	// Also clear the strategy when exiting shrinking
	wa.strategy.Store("")
}

// Snapshot returns a read-only snapshot of the current activity.
// This is the preferred way to read worker activity as it provides a consistent view.
func (wa *WorkerActivity) Snapshot() WorkerActivitySnapshot {
	return WorkerActivitySnapshot{
		State:            wa.GetState(),
		Strategy:         wa.GetStrategy(),
		CorpusEntryIndex: wa.GetCorpusEntryIndex(),
		ShrinkIteration:  int(wa.shrinkIteration.Load()),
		ShrinkLimit:      int(wa.shrinkLimit.Load()),
		LastUpdate:       time.Unix(wa.lastUpdate.Load(), 0),
	}
}

// WorkerActivitySnapshot is a read-only snapshot of worker activity at a point in time
type WorkerActivitySnapshot struct {
	// State is the worker state at snapshot time
	State WorkerState
	// Strategy is the generation strategy being used
	Strategy string
	// CorpusEntryIndex is the corpus entry being replayed (-1 if not replaying)
	CorpusEntryIndex int
	// ShrinkIteration is the current shrinking iteration
	ShrinkIteration int
	// ShrinkLimit is the maximum shrinking iterations
	ShrinkLimit int
	// LastUpdate is when the activity was last updated
	LastUpdate time.Time
}

// Description returns a human-readable description of the activity
func (s WorkerActivitySnapshot) Description() string {
	switch s.State {
	case WorkerStateGenerating:
		if s.Strategy != "" {
			return fmt.Sprintf("Generating (%s)", s.Strategy)
		}
		return "Generating"
	case WorkerStateReplayingCorpus:
		if s.CorpusEntryIndex >= 0 {
			return fmt.Sprintf("Replaying corpus #%d", s.CorpusEntryIndex)
		}
		return "Replaying corpus"
	case WorkerStateShrinking:
		if s.ShrinkLimit > 0 {
			progress := float64(s.ShrinkIteration) / float64(s.ShrinkLimit) * 100
			return fmt.Sprintf("Shrinking (%d/%d, %.1f%%)", s.ShrinkIteration, s.ShrinkLimit, progress)
		}
		return "Shrinking"
	case WorkerStateIdle:
		return "Idle"
	default:
		return "Unknown"
	}
}

// ShrinkProgress returns the shrinking progress as a fraction (0.0 to 1.0)
func (s WorkerActivitySnapshot) ShrinkProgress() float64 {
	if s.ShrinkLimit == 0 {
		return 0.0
	}
	progress := float64(s.ShrinkIteration) / float64(s.ShrinkLimit)
	if progress > 1.0 {
		return 1.0
	}
	return progress
}

// IsActive returns true if the worker is actively doing work (not idle)
func (s WorkerActivitySnapshot) IsActive() bool {
	return s.State != WorkerStateIdle
}
