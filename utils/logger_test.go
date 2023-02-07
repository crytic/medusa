package utils

import (
	// "github.com/stretchr/testify/assert"
	"encoding/json"
	"testing"
)

// testDirPath := "./testdata/tmp-logging-tests"

func isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// check that the context is being properly set & retrived
func TestLoggerFormat(t *testing.T) {
	log := NewLogger(0, true, "")
	log.Info("test log in json-mode", "foo", "bar")
	// does it look good?
	log = NewLogger(0, false, "")
	log.Info("test log in text-mode", "foo", "bar")
}

/*
// check that the context is being properly set & retrived
func TestLoggerContext(t *testing.T) {
	context := "foo"
	logger := NewLogger(context, Info)
	savedContext := logger.GetContext()
	assert.EqualValues(t, savedContext, context)
	context = "bar"
  logger.SetContext(context)
	savedContext = logger.GetContext()
	assert.EqualValues(t, savedContext, context)
}

// TestLogInfo tests whether logs work with informational severity
func TestLoggerLevel(t *testing.T) {
	maxLevel := Warn
	logger := NewLogger("foobar", maxLevel)
	gottenLevel := logger.GetMaxLevel()
	assert.EqualValues(t, gottenLevel, maxLevel)
  logger.Debug("debug log") // should not log
  logger.Info("informational log") // should not log
  logger.Warn("warning log") // should log
}
*/
