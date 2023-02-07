package utils

import (
	"golang.org/x/exp/slog"
	"io"
	"log"
	"os"
)

type Logger struct {
	level   int
	backend *slog.Logger
}

// NewLogger creates a new logger
func NewLogger(
	level int,
	useJSON bool,
	logFilePath string,
) *Logger {

	// set writer according to given logFilePath arg
	var writer io.Writer
	var err error
	if logFilePath == "" {
		writer = os.Stdout
	} else {
		writer, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
			// should we fall back to stdout if error accessing log file?
			// writer = os.Stdout
		}
	}

	// set handler according to given format arg
	var handler slog.Handler
	if useJSON {
		handler = slog.NewJSONHandler(writer)
	} else {
		handler = slog.NewTextHandler(writer)
	}

	// create new logger
	backend := slog.New(handler)
	return &Logger{
		backend: backend,
	}
}

func (l *Logger) Debug(baseMsg string, args ...any) {
	l.backend.Debug(baseMsg, args...)
}

func (l *Logger) Info(baseMsg string, args ...any) {
	l.backend.Info(baseMsg, args...)
}

func (l *Logger) Warn(baseMsg string, args ...any) {
	l.backend.Warn(baseMsg, args...)
}

/*
func Error(baseMsg string, args ...any) {
	slog.Error(fmt.Sprintf(baseMsg, args...), baseMsg, args...)
}
*/
