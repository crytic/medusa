package utils

import (
  "os"
	"golang.org/x/exp/slog"
)

/*
type LogLevel int64
const (
	Unknown LogLevel = iota
	Error
	Warn
	Info
	Debug
)
type LogFormat string
const (
	Text LogFormat = "Text" // one line with key=value pairs
	JSON LogFormat = "JSON" // one line in JSON format
)
type LogDestination string
const (
	Standard LogDestination = "Standard" // stdout
	File LogDestination = "File" // write to file on disk
	Return LogDestination = "Return" // return for further processing
)
type Logger struct {
	// indicates where in the system this log came from
	Context string "NoContext"
	// all logs with a severity less than MaxLevel will be silently ignored
	MaxLevel LogLevel "Info"
  Format LogFormat "Text"
  Destination LogDestination "Standard"
}
*/

func NewLogger(
  format string,
  maxlevel int8,
) *slog.Logger {
  var handler slog.Handler
  if format == "JSON" {
    handler = slog.NewJSONHandler(os.Stdout)
  } else {
    handler = slog.NewTextHandler(os.Stdout)
  }
  logger := slog.New(handler)
  return logger
  /*
	return &Logger{
		Context:  context,
		MaxLevel: maxlevel,
    Destination: destination,
    Format: format,
	}
  */
}

// using any type for print args because that's what Printf uses, see: https://cs.opensource.google/go/go/+/master:src/fmt/print.go;l=232
// args should be alternating key names (string) and values (any)
//   an odd number of args should ignore the dangling key name (TBD?)
/*
func (l *slog.Logger) Debug(baseMsg string, args ...any) {
	slog.Debug(fmt.Sprintf(baseMsg, args...))
}

func (l *slog.Logger) Info(baseMsg string, args ...any) {
	slog.Info(fmt.Sprintf(baseMsg, args...))
}

func (l *slog.Logger) Warn(baseMsg string, args ...any) {
	slog.Warn(fmt.Sprintf(baseMsg, args...))
}
*/

/*
func Error(baseMsg string, args ...any) {
	slog.Error(fmt.Sprintf(baseMsg, args...), baseMsg, args...)
}
*/
