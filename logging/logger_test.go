package logging

import (
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TestAddAndRemoveWriter will test to Logger.AddWriter and Logger.RemoveWriter functions to ensure that they work as expected.
func TestAddAndRemoveWriter(t *testing.T) {
	// Create a base logger
	logger := NewLogger(zerolog.InfoLevel)

	// Add three types of writers
	// 1. Unstructured and colorized output to stdout
	logger.AddWriter(os.Stdout, UNSTRUCTURED, true)
	// 2. Unstructured and non-colorized output to stderr
	logger.AddWriter(os.Stderr, UNSTRUCTURED, false)
	// 3. Structured output to stdin
	logger.AddWriter(os.Stdin, STRUCTURED, false)

	// We should expect the underlying data structures are correctly updated
	assert.Equal(t, len(logger.unstructuredWriters), 1)
	assert.Equal(t, len(logger.unstructuredColorWriters), 1)
	assert.Equal(t, len(logger.structuredWriters), 1)

	// Try to add duplicate writers
	logger.AddWriter(os.Stdout, UNSTRUCTURED, true)
	logger.AddWriter(os.Stderr, UNSTRUCTURED, false)
	logger.AddWriter(os.Stdin, STRUCTURED, false)

	// Ensure that the lengths of the lists have not changed
	assert.Equal(t, len(logger.unstructuredWriters), 1)
	assert.Equal(t, len(logger.unstructuredColorWriters), 1)
	assert.Equal(t, len(logger.structuredWriters), 1)

	// Remove each writer
	logger.RemoveWriter(os.Stdout, UNSTRUCTURED, true)
	logger.RemoveWriter(os.Stderr, UNSTRUCTURED, false)
	logger.RemoveWriter(os.Stdin, STRUCTURED, false)

	// We should expect the underlying data structures are correctly updated
	assert.Equal(t, len(logger.unstructuredWriters), 0)
	assert.Equal(t, len(logger.unstructuredColorWriters), 0)
	assert.Equal(t, len(logger.structuredWriters), 0)
}
