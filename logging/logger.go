package logging

import (
	"fmt"
	"github.com/crytic/medusa/logging/colors"
	"github.com/rs/zerolog"
	"io"
	"os"
	"strings"
)

// GlobalLogger describes a Logger that is disabled by default and is instantiated when the fuzzer is created. Each module/package
// should create its own sub-logger. This allows to create unique logging instances depending on the use case.
var GlobalLogger = NewLogger(zerolog.Disabled, false, nil)

// Logger describes a custom logging object that can log events to any arbitrary channel and can handle specialized
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

// LogFormat describes what format to log in
type LogFormat string

const (
	// STRUCTURED describes that logging should be done in structured JSON format
	STRUCTURED LogFormat = "structured"
	// UNSTRUCTRED describes that logging should be done in an unstructured format
	UNSTRUCTURED LogFormat = "unstructured"
)

// StructuredLogInfo describes a key-value mapping that can be used to log structured data
type StructuredLogInfo map[string]any

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
		consoleWriter := setupDefaultFormatting(zerolog.ConsoleWriter{Out: os.Stdout}, level)
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
	if format == UNSTRUCTURED {
		writer = zerolog.ConsoleWriter{Out: writer, NoColor: true}
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
func (l *Logger) Trace(args ...any) {
	// Build the messages and retrieve any error or associated structured log info
	consoleMsg, multiMsg, err, info := buildMsgs(args...)

	// Instantiate log events
	consoleLog := l.consoleLogger.Trace()
	multiLog := l.multiLogger.Trace()

	// Chain the error
	chainError(consoleLog, multiLog, err, l.level <= zerolog.DebugLevel)

	// Chain the structured log info and messages and send off the logs
	chainStructuredLogInfoAndMsgs(consoleLog, multiLog, info, consoleMsg, multiMsg)
}

// Debug is a wrapper function that will log a debug event
func (l *Logger) Debug(args ...any) {
	// Build the messages and retrieve any error or associated structured log info
	consoleMsg, multiMsg, err, info := buildMsgs(args...)

	// Instantiate log events
	consoleLog := l.consoleLogger.Debug()
	multiLog := l.multiLogger.Debug()

	// Chain the error
	chainError(consoleLog, multiLog, err, l.level <= zerolog.DebugLevel)

	// Chain the structured log info and messages and send off the logs
	chainStructuredLogInfoAndMsgs(consoleLog, multiLog, info, consoleMsg, multiMsg)
}

// Info is a wrapper function that will log an info event
func (l *Logger) Info(args ...any) {
	// Build the messages and retrieve any error or associated structured log info
	consoleMsg, multiMsg, err, info := buildMsgs(args...)

	// Instantiate log events
	consoleLog := l.consoleLogger.Info()
	multiLog := l.multiLogger.Info()

	// Chain the error
	chainError(consoleLog, multiLog, err, l.level <= zerolog.DebugLevel)

	// Chain the structured log info and messages and send off the logs
	chainStructuredLogInfoAndMsgs(consoleLog, multiLog, info, consoleMsg, multiMsg)
}

// Warn is a wrapper function that will log a warning event both on console
func (l *Logger) Warn(args ...any) {
	// Build the messages and retrieve any error or associated structured log info
	consoleMsg, multiMsg, err, info := buildMsgs(args...)

	// Instantiate log events
	consoleLog := l.consoleLogger.Warn()
	multiLog := l.multiLogger.Warn()

	// Chain the error
	chainError(consoleLog, multiLog, err, l.level <= zerolog.DebugLevel)

	// Chain the structured log info and messages and send off the logs
	chainStructuredLogInfoAndMsgs(consoleLog, multiLog, info, consoleMsg, multiMsg)
}

// Error is a wrapper function that will log an error event.
func (l *Logger) Error(args ...any) {
	// Build the messages and retrieve any error or associated structured log info
	consoleMsg, multiMsg, err, info := buildMsgs(args...)

	// Instantiate log events
	consoleLog := l.consoleLogger.Error()
	multiLog := l.multiLogger.Error()

	// Chain the error
	chainError(consoleLog, multiLog, err, l.level <= zerolog.DebugLevel)

	// Chain the structured log info and messages and send off the logs
	chainStructuredLogInfoAndMsgs(consoleLog, multiLog, info, consoleMsg, multiMsg)
}

// Panic is a wrapper function that will log a panic event
func (l *Logger) Panic(args ...any) {
	// Build the messages and retrieve any error or associated structured log info
	consoleMsg, multiMsg, err, info := buildMsgs(args...)

	// Instantiate log events
	consoleLog := l.consoleLogger.Panic()
	multiLog := l.multiLogger.Panic()

	// Chain the error
	chainError(consoleLog, multiLog, err, true)

	// Chain the structured log info and messages and send off the logs
	chainStructuredLogInfoAndMsgs(consoleLog, multiLog, info, consoleMsg, multiMsg)
}

// buildMsgs describes a function that takes in a variadic list of arguments of any type and returns two strings and,
// optionally, an error and a StructuredLogInfo object. The first string will be a colorized-string that can be used for
// console logging while the second string will be a non-colorized one that can be used for file/structured logging.
// The error and the StructuredLogInfo can be used to add additional context to log messages
func buildMsgs(args ...any) (string, string, error, StructuredLogInfo) {
	// Guard clause
	if len(args) == 0 {
		return "", "", nil, nil
	}

	// Initialize the base color context, the string buffers and the structured log info object
	colorCtx := colors.Reset
	consoleOutput := make([]string, 0)
	fileOutput := make([]string, 0)
	var info StructuredLogInfo
	var err error

	// Iterate through each argument in the list and switch on type
	for _, arg := range args {
		switch t := arg.(type) {
		case colors.ColorFunc:
			// If the argument is a color function, switch the current color context
			colorCtx = t
		case StructuredLogInfo:
			// Note that only one structured log info can be provided for each log message
			info = t
		case error:
			// Note that only one error can be provided for each log message
			err = t
		default:
			// In the base case, append the object to the two string buffers. The console string buffer will have the
			// current color context applied to it.
			consoleOutput = append(consoleOutput, colorCtx(t))
			fileOutput = append(fileOutput, fmt.Sprintf("%v", t))
		}
	}

	return strings.Join(consoleOutput, ""), strings.Join(fileOutput, ""), err, info
}

// chainError is a helper function that takes in a *zerolog.Event for console and multi-log output and chains an error
// to both events. If debug is true, then a stack trace is added to both events as well.
func chainError(consoleLog *zerolog.Event, multiLog *zerolog.Event, err error, debug bool) {
	// First append the errors to each event. Note that even if err is nil, there will not be a panic here
	consoleLog.Err(err)
	multiLog.Err(err)

	// If we are in debug mode or below, then we will add the stack traces as well for debugging
	if debug {
		consoleLog.Stack()
		multiLog.Stack()
	}
}

// chainStructuredLogInfoAndMsgs is a helper function that takes in a *zerolog.Event for console and multi-log output,
// chains any StructuredLogInfo provided to it, adds the associated messages, and sends out the logs to their respective
// channels.
func chainStructuredLogInfoAndMsgs(consoleLog *zerolog.Event, multiLog *zerolog.Event, info StructuredLogInfo, consoleMsg string, multiMsg string) {
	// If we are provided a structured log info object, add that as a key-value pair to the events
	if info != nil {
		consoleLog.Any("info", info)
		multiLog.Any("info", info)
	}

	// Append the messages to each event. This will also result in the log events being sent out to their respective
	// streams. Note that we are deferring the msg to multi logger in case we are logging a panic and want to make sure that
	// all channels receive the panic log
	defer multiLog.Msg(multiMsg)
	consoleLog.Msg(consoleMsg)
}

// setupDefaultFormatting will update the console logger's formatting to the medusa standard
func setupDefaultFormatting(writer zerolog.ConsoleWriter, level zerolog.Level) zerolog.ConsoleWriter {
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
			return colors.CyanBold(zerolog.LevelTraceValue)
		case zerolog.DebugLevel:
			// Return a bold, blue "debug" string
			return colors.BlueBold(zerolog.LevelDebugValue)
		case zerolog.InfoLevel:
			// Return a bold, green left arrow
			return colors.GreenBold(colors.LEFT_ARROW)
		case zerolog.WarnLevel:
			// Return a bold, yellow "warn" string
			return colors.YellowBold(zerolog.LevelWarnValue)
		case zerolog.ErrorLevel:
			// Return a bold, red "err" string
			return colors.RedBold(zerolog.LevelErrorValue)
		case zerolog.FatalLevel:
			// Return a bold, red "fatal" string
			return colors.RedBold(zerolog.LevelFatalValue)
		case zerolog.PanicLevel:
			// Return a bold, red "panic" string
			return colors.RedBold(zerolog.LevelPanicValue)
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
