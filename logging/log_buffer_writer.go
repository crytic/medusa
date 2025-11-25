package logging

import (
	"sync"
	"time"
)

// LogEntry represents a single log entry with timestamp
type LogEntry struct {
	Timestamp time.Time
	Message   string
}

// LogBufferWriter implements io.Writer and buffers log entries in a circular buffer
type LogBufferWriter struct {
	mu       sync.RWMutex
	entries  []LogEntry
	capacity int
	index    int  // Current write position for circular buffer
	full     bool // Whether we've filled the buffer at least once
}

// NewLogBufferWriter creates a new log buffer writer with the specified capacity
func NewLogBufferWriter(capacity int) *LogBufferWriter {
	return &LogBufferWriter{
		entries:  make([]LogEntry, capacity),
		capacity: capacity,
		index:    0,
		full:     false,
	}
}

// Write implements io.Writer interface
// This method is called by the logger when writing log entries
func (w *LogBufferWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Store the log entry with current timestamp
	w.entries[w.index] = LogEntry{
		Timestamp: time.Now(),
		Message:   string(p),
	}

	// Move to next position in circular buffer
	w.index++
	if w.index >= w.capacity {
		w.index = 0
		w.full = true
	}

	return len(p), nil
}

// GetEntries returns the most recent log entries (up to limit)
// If limit <= 0 or limit > total entries, returns all available entries
// Entries are returned in chronological order (oldest first)
func (w *LogBufferWriter) GetEntries(limit int) []LogEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Determine how many entries we have
	var totalEntries int
	if w.full {
		totalEntries = w.capacity
	} else {
		totalEntries = w.index
	}

	// If no entries yet, return empty slice
	if totalEntries == 0 {
		return []LogEntry{}
	}

	// Determine how many to return
	numToReturn := totalEntries
	if limit > 0 && limit < numToReturn {
		numToReturn = limit
	}

	// Create result slice
	result := make([]LogEntry, numToReturn)

	if w.full {
		// Buffer is full, need to read from the oldest entry (at current index)
		// Calculate the starting position for the requested number of entries
		startIdx := (w.index - numToReturn + w.capacity) % w.capacity

		// Copy entries in chronological order
		for i := 0; i < numToReturn; i++ {
			idx := (startIdx + i) % w.capacity
			result[i] = w.entries[idx]
		}
	} else {
		// Buffer not full yet, read from 0 to index
		// Calculate starting position
		startIdx := w.index - numToReturn
		if startIdx < 0 {
			startIdx = 0
			numToReturn = w.index
			result = make([]LogEntry, numToReturn)
		}

		// Copy entries
		copy(result, w.entries[startIdx:w.index])
	}

	return result
}

// GetAllEntries returns all available log entries in chronological order
func (w *LogBufferWriter) GetAllEntries() []LogEntry {
	return w.GetEntries(0)
}

// Clear clears all log entries from the buffer
func (w *LogBufferWriter) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.index = 0
	w.full = false
	// No need to clear the entries array, just reset the pointers
}

// Count returns the current number of log entries in the buffer
func (w *LogBufferWriter) Count() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.full {
		return w.capacity
	}
	return w.index
}
