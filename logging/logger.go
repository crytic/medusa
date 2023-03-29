package logging

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/trailofbits/medusa/logging/colors"
	"github.com/trailofbits/medusa/utils"
	"io"
	"os"
	"strings"
)

// GlobalLogger describes a Logger that is instantiated when a fuzzer is created and is then used to create
// sub-loggers for each individual module / package. This allows to create unique logging instances depending on the use case.
var GlobalLogger *Logger

// MultiLogger describes a custom logging object that can log events to any arbitrary channel and can handle specialized
// output to console as well
type Logger struct {
	// level describes the log level
	level zerolog.Level

	// multiLogger describes a logger that will be used to output logs to any arbitrary channel(s) in either structured
	// or unstructured format.
	multiLogger zerolog.Logger

	// consoleLogger describes a logger that will be used to output unstructured output to console.
	// We are creating a separate logger for console so that we can support specialized formatting / custom coloring.
	consoleLogger zerolog.Logger

	// writers describes a list of io.Writer objects where log output will go. This writers list can be appended to /
	// removed from.
	writers []io.Writer
}

// Formatter describes a function that can be used to provide custom formatting for specific log events.
// This is especially useful when logging to console where we want to colorize specific items
type Formatter func(fields map[string]any, msg string) string

// LogFormat describes what format to log in
type LogFormat string

const (
	// STRUCTURED describes that logging should be done in structured JSON format
	STRUCTURED LogFormat = "structured"
	// UNSTRUCTRED describes that logging should be done in an unstructured format
	UNSTRUCTURED LogFormat = "unstructured"
)

// NewLogger will create a new Logger object with a specific log level. The Logger can output to console, if enabled,
// and output logs to any number of arbitrary io.Writer channels
func NewLogger(level zerolog.Level, consoleEnabled bool, writers ...io.Writer) *Logger {
	// The two base loggers are effectively loggers that are disabled
	// We are creating instances of them so that we do not get nil pointer dereferences down the line
	baseMultiLogger := zerolog.New(os.Stdout).Level(zerolog.Disabled)
	baseConsoleLogger := zerolog.New(os.Stdout).Level(zerolog.Disabled)

	// If we are provided a list of writers, update the multi logger
	if len(writers) > 0 {
		baseMultiLogger = zerolog.New(zerolog.MultiLevelWriter(writers...)).Level(level).With().Timestamp().Logger()
	}

	// If console logging is enabled, update the console logger
	if consoleEnabled {
		consoleWriter := setupDefaultFormatting(zerolog.ConsoleWriter{Out: os.Stdout}, level, true)
		baseConsoleLogger = zerolog.New(consoleWriter).Level(level)
	}

	return &Logger{
		level:         level,
		multiLogger:   baseMultiLogger,
		consoleLogger: baseConsoleLogger,
		writers:       writers,
	}
}

// NewSubLogger will create a new Logger with unique context in the form of a key-value pair. The expected use of this
// function is for each package to have their own unique logger so that parsing of logs is "grep-able" based on some key
func (l *Logger) NewSubLogger(key string, value string) *Logger {
	subFileLogger := l.multiLogger.With().Str(key, value).Logger()
	subConsoleLonger := l.consoleLogger.With().Str(key, value).Logger()
	return &Logger{
		level:         l.level,
		multiLogger:   subFileLogger,
		consoleLogger: subConsoleLonger,
		writers:       l.writers,
	}
}

// AddWriter will add a writer to the list of channels where log output will be sent.
func (l *Logger) AddWriter(writer io.Writer, format LogFormat) {
	// Check to see if the writer is already in the array of writers
	for _, w := range l.writers {
		if writer == w {
			return
		}
	}

	// If we want unstructured output, wrap the base writer object into a console writer so that we get unstructured output with no ANSI coloring
	switch format {
	case UNSTRUCTURED:
		writer = zerolog.ConsoleWriter{Out: writer, NoColor: true}
	default:
		return
	}

	// Add it to the list of writers and update the multi logger
	l.writers = append(l.writers, writer)
	l.multiLogger = zerolog.New(zerolog.MultiLevelWriter(l.writers...)).Level(l.level).With().Timestamp().Logger()
}

// RemoveWriter will remove a writer from the list of writers that the logger manages. If the writer does not exist, this
// function is a no-op
func (l *Logger) RemoveWriter(writer io.Writer) {
	// Iterate through the writers
	for i, w := range l.writers {
		if writer == w {
			// Create a new slice without the writer at index i
			l.writers = append(l.writers[:i], l.writers[i+1])
		}
	}
}

// Level will get the log level of the Logger
func (l *Logger) Level() zerolog.Level {
	return l.level
}

// SetLevel will update the log level of the Logger
func (l *Logger) SetLevel(level zerolog.Level) {
	l.level = level
	l.multiLogger = l.multiLogger.Level(level)
	l.consoleLogger = l.consoleLogger.Level(level)
}

// Trace is a wrapper function that will log a trace event
func (l *Logger) Trace(msg string, fields map[string]any) {
	// Log to multi logger
	l.multiLogger.Trace().Fields(fields).Msg(msg)

	// Log to console
	// No additional formatting for trace logs
	l.consoleLogger.Trace().Fields(fields).Msg(msg)
}

// Debug is a wrapper function that will log a debug event
func (l *Logger) Debug(msg string, fields map[string]any) {
	// Log to multi logger
	l.multiLogger.Debug().Fields(fields).Msg(msg)

	// Log to console
	// No additional formatting for debug logs
	l.consoleLogger.Debug().Fields(fields).Msg(msg)
}

// Info is a wrapper function that will log an info event
// Specific log events may choose to have custom formatting for console
func (l *Logger) Info(msg string, fields map[string]any) {
	// Grab formatter field
	formatterFunc, fields := utils.GetAndRemoveKeyFromMapping(fields, "formatter")

	// Log to multi logger
	l.multiLogger.Info().Fields(fields).Msg(msg)

	// Colorize and format the fields + msg, if necessary
	if formatterFunc != nil {
		msg = formatterFunc.(Formatter)(fields, msg)
	}

	// Log to console
	// If we are in debug mode or below, add the fields. Otherwise, just output the message
	if l.consoleLogger.GetLevel() <= zerolog.DebugLevel {
		l.consoleLogger.Info().Fields(fields).Msg(msg)
	} else {
		l.consoleLogger.Info().Msg(msg)
	}
}

// Warn is a wrapper function that will log a warning event both on console
func (l *Logger) Warn(msg string, fields map[string]any) {
	// Log to multi logger
	l.multiLogger.Warn().Fields(fields).Msg(msg)

	// Log to console
	// If we are in debug mode or below, add the fields. Otherwise, just output the message
	if l.consoleLogger.GetLevel() <= zerolog.DebugLevel {
		l.consoleLogger.Warn().Fields(fields).Msg(msg)
	} else {
		l.consoleLogger.Warn().Msg(msg)
	}
}

// Error is a wrapper function that will log an error event
// Note that if the error key is not in the fields mapping, this function will panic
// TODO: Maybe we don't panic and instead just log the msg with an empty err field. Same applies to Fatal and Panic
func (l *Logger) Error(msg string, fields map[string]any) {
	// Grab error from fields
	err, fields := utils.GetAndRemoveKeyFromMapping(fields, "error")

	// Log to multi logger with stack
	l.multiLogger.Error().Err(err.(error)).Stack().Fields(fields).Msg(msg)

	// If we are in debug mode or below, log the stack and error. Otherwise, just output the error
	if l.consoleLogger.GetLevel() <= zerolog.DebugLevel {
		l.consoleLogger.Error().Err(err.(error)).Stack().Msg(msg)
	} else {
		l.consoleLogger.Error().Err(err.(error)).Msg(msg)
	}
}

// Fatal is a wrapper function that will log a fatal event
// Note that if the error key is not in the fields mapping, this function will panic
func (l *Logger) Fatal(msg string, fields map[string]any) {
	// Grab error from fields
	err, fields := utils.GetAndRemoveKeyFromMapping(fields, "error")

	// Log to multi logger
	l.multiLogger.Fatal().Err(err.(error)).Stack().Fields(fields).Msg(msg)

	// Log to console with no regard for formatting, all bets are off
	l.consoleLogger.Fatal().Err(err.(error)).Stack().Fields(fields).Msg(msg)
}

// Panic is a wrapper function that will log a panic event
// Note that if the error key is not in the fields mapping, this function will, ironically, panic
func (l *Logger) Panic(msg string, fields map[string]any) {
	// Grab error from fields
	err, fields := utils.GetAndRemoveKeyFromMapping(fields, "error")

	// Log to multi logger
	l.multiLogger.Panic().Err(err.(error)).Stack().Fields(fields).Msg(msg)

	// Log to console with no regard for formatting, all bets are off
	l.consoleLogger.Panic().Err(err.(error)).Stack().Fields(fields).Msg(msg)
}

// setupDefaultFormatting will update the console logger's formatting to the medusa standard
func setupDefaultFormatting(writer zerolog.ConsoleWriter, level zerolog.Level, colorable bool) zerolog.ConsoleWriter {
	// Get rid of the timestamp for console output
	writer.FormatTimestamp = func(i interface{}) string {
		return ""
	}

	// We will define a custom format for each level
	writer.FormatLevel = func(i any) string {
		// Create a level object for better switch logic
		level, err := zerolog.ParseLevel(i.(string))
		if err != nil {
			panic(fmt.Sprintf("unable to parse the log level: %v", err))
		}

		// Switch on the level and return a custom, colored string
		switch level {
		case zerolog.TraceLevel:
			// Return a bold, cyan "trace" string
			return colors.Colorize(colors.Colorize(strings.ToUpper(zerolog.LevelTraceValue), colors.COLOR_CYAN), colors.COLOR_BOLD)
		case zerolog.DebugLevel:
			// Return a bold, blue "debug" string
			return colors.Colorize(colors.Colorize(strings.ToUpper(zerolog.LevelDebugValue), colors.COLOR_BLUE), colors.COLOR_BOLD)
		case zerolog.InfoLevel:
			// Return a bold, green left arrow
			return colors.Colorize(colors.Colorize(colors.LEFT_ARROW, colors.COLOR_GREEN), colors.COLOR_BOLD)
		case zerolog.WarnLevel:
			// Return a bold, yellow "warn" string
			return colors.Colorize(colors.Colorize(strings.ToUpper(zerolog.LevelWarnValue), colors.COLOR_YELLOW), colors.COLOR_BOLD)
		case zerolog.ErrorLevel:
			// Return a bold, red "err" string
			return colors.Colorize(colors.Colorize(strings.ToUpper(zerolog.LevelErrorValue), colors.COLOR_RED), colors.COLOR_BOLD)
		case zerolog.FatalLevel:
			// Return a bold, red "fatal" string
			return colors.Colorize(colors.Colorize(strings.ToUpper(zerolog.LevelFatalValue), colors.COLOR_RED), colors.COLOR_BOLD)
		case zerolog.PanicLevel:
			// Return a bold, red "panic" string
			return colors.Colorize(colors.Colorize(strings.ToUpper(zerolog.LevelPanicValue), colors.COLOR_RED), colors.COLOR_BOLD)
		default:
			return i.(string)
		}
	}

	// If we are above debug level, we want to get rid of the `module` component when logging to console
	if level > zerolog.DebugLevel {
		writer.FieldsExclude = []string{"module"}
	}

	return writer
}
