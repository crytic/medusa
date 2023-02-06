package utils

import (
  "github.com/stretchr/testify/assert"
	"testing"
)

// TestLogInfo tests whether logs work with informational severity
func TestLoggerScratchpad(t *testing.T) {
  testString := "foobar"
  logger := NewLogger(testString)
  level := logger.GetLevel()
  assert.EqualValues(t, level, testString)
}

