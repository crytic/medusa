package logging

import (
	"bytes"
	"os"
	"testing"

	"github.com/crytic/medusa/logging/colors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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

// TestDisabledColors verifies the behavior of the unstructured colored logger when colors are disabled,
// ensuring that it does not output colors when the color feature is turned off.
func TestDisabledColors(t *testing.T) {
	// Force a deterministic terminal environment so the colored ConsoleWriter behavior is consistent.
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	// Create a base logger
	logger := NewLogger(zerolog.InfoLevel)

	// Reset global color state after the test because the color helpers are process-global.
	t.Cleanup(colors.EnableColor)

	// Add colorized logger
	var buf bytes.Buffer
	logger.AddWriter(&buf, UNSTRUCTURED, true)

	// We should expect the underlying data structures are correctly updated
	assert.Equal(t, len(logger.unstructuredColorWriters), 1)

	// Disable colors and log msg
	colors.DisableColor()
	logger.Info("foo")

	// Disabling the package-level colors should remove Medusa's custom level coloring even when the
	// underlying ConsoleWriter is configured for colored output.
	assert.Contains(t, buf.String(), colors.LEFT_ARROW)
	assert.NotContains(t, buf.String(), "\x1b[32m")
}
