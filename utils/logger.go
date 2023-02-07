package utils

import (
	"golang.org/x/exp/slog"
	"io"
	"os"
)

const (
	LevelTrace  = slog.Level(-8)
	LevelDebug  = slog.Level(-4)
	LevelInfo   = slog.Level(0)
	LevelNotice = slog.Level(2)
	LevelWarn   = slog.Level(4)
	LevelError  = slog.Level(8)
	LevelFatal  = slog.Level(12)
)

type Logger struct {
	level   slog.Level
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
	if logFilePath == "" {
		writer = os.Stdout
	} else {
		fileWriter, err := os.OpenFile(
			logFilePath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644,
		)
		if err == nil {
			writer = fileWriter
		} else {
			// should we fail more loudly if we failed to access the given logfile?
			writer = os.Stdout
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
		level:   slog.Level(level),
		backend: backend,
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	if l.level <= LevelDebug {
		l.backend.Debug(msg, args...)
	}
}

func (l *Logger) Info(msg string, args ...any) {
	if l.level <= LevelInfo {
		l.backend.Info(msg, args...)
	}
}

func (l *Logger) Warn(msg string, args ...any) {
	if l.level <= LevelWarn {
		l.backend.Warn(msg, args...)
	}
}

/*
func Error(msg string, args ...any) {
	slog.Error(fmt.Sprintf(msg, args...), msg, args...)
}
*/
