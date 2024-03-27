package logging

import (
	"fmt"
	"github.com/crytic/medusa/logging/colors"
	"github.com/rs/zerolog"
	"io"
	"strings"
)

// GlobalLogger describes a Logger that is disabled by default and is instantiated when the fuzzer is created. Each module/package
// should create its own sub-logger. This allows to create unique logging instances depending on the use case.
var GlobalLogger *Logger

// Logger describes a custom logging object that can log events to any arbitrary channel in structured, unstructured with colors,
// and unstructured formats.
type Logger struct {
	// level describes the log level
	level zerolog.Level

	// structuredLogger describes a logger that will be used to output structured logs to any arbitrary channel.
	structuredLogger zerolog.Logger

	// structuredWriters describes the various channels that the output from the structuredLogger will go to.
	structuredWriters []io.Writer

	// unstructuredLogger describes a logger that will be used to stream un-colorized, unstructured output to any arbitrary channel.
	unstructuredLogger zerolog.Logger

	// unstructuredWriters describes the various channels that the output from the unstructuredLogger will go to.
	unstructuredWriters []io.Writer

	// unstructuredColorLogger describes a logger that will be used to stream colorized, unstructured output to any arbitrary channel.
	unstructuredColorLogger zerolog.Logger

	// unstructuredColorWriters describes the various channels that the output from the unstructuredColoredLogger will go to.
	unstructuredColorWriters []io.Writer
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

// NewLogger will create a new Logger object with a specific log level. By default, a logger that is instantiated
// with this function is not usable until a log channel is added. To add or remove channels that the logger
// streams logs to, call the Logger.AddWriter and Logger.RemoveWriter functions.
func NewLogger(level zerolog.Level) *Logger {
	return &Logger{
		level:                    level,
		structuredLogger:         zerolog.New(nil).Level(level),
		structuredWriters:        make([]io.Writer, 0),
		unstructuredLogger:       zerolog.New(nil).Level(level),
		unstructuredWriters:      make([]io.Writer, 0),
		unstructuredColorLogger:  zerolog.New(nil).Level(level),
		unstructuredColorWriters: make([]io.Writer, 0),
	}
}

// NewSubLogger will create a new Logger with unique context in the form of a key-value pair. The expected use of this
// is for each module or component of the system to create their own contextualized logs. The key can be used to search
// for logs from a specific module or component.
func (l *Logger) NewSubLogger(key string, value string) *Logger {
	// Create the sub-loggers with the new key-value context
	subStructuredLogger := l.structuredLogger.With().Str(key, value).Logger()
	subUnstructuredColoredLogger := l.unstructuredColorLogger.With().Str(key, value).Logger()
	subUnstructuredLogger := l.unstructuredLogger.With().Str(key, value).Logger()

	// Create new slices for the writers since we want to make a deep copy for each one
	subStructuredWriters := make([]io.Writer, len(l.structuredWriters))
	copy(subStructuredWriters, l.structuredWriters)

	subUnstructuredColorWriters := make([]io.Writer, len(l.unstructuredColorWriters))
	copy(subUnstructuredColorWriters, l.unstructuredColorWriters)

	subUnstructuredWriters := make([]io.Writer, len(l.unstructuredWriters))
	copy(subUnstructuredWriters, l.unstructuredWriters)

	// Return a new logger
	return &Logger{
		level:                    l.level,
		structuredLogger:         subStructuredLogger,
		structuredWriters:        subStructuredWriters,
		unstructuredColorLogger:  subUnstructuredColoredLogger,
		unstructuredColorWriters: subUnstructuredColorWriters,
		unstructuredLogger:       subUnstructuredLogger,
		unstructuredWriters:      subUnstructuredWriters,
	}
}

// AddWriter will add a writer to which log output will go to. If the format is structured then the writer will get
// structured output. If the writer is unstructured, then the writer has the choice to either receive colored or un-colored
// output. Note that unstructured writers will be converted into a zerolog.ConsoleWriter to maintain the same format
// across all unstructured output streams.
func (l *Logger) AddWriter(writer io.Writer, format LogFormat, colored bool) {
	// First, try to add the writer to the list of channels that want structured logs
	if format == STRUCTURED {
		for _, w := range l.structuredWriters {
			if w == writer {
				// Writer already exists, return
				return
			}
		}
		// Add the writer and recreate the logger
		l.structuredWriters = append(l.structuredWriters, writer)
		l.structuredLogger = zerolog.New(zerolog.MultiLevelWriter(l.structuredWriters...)).Level(l.level).With().Timestamp().Logger()
		return
	}

	// Now that we know we are going to create an unstructured writer, we will create an unstructured writer with(out) coloring
	// using zerolog's console writer object.
	unstructuredWriter := formatUnstructuredWriter(writer, l.level, colored)

	// Now, try to add the writer to the list of channels that want unstructured, colored logs
	if format == UNSTRUCTURED && colored {
		for _, w := range l.unstructuredColorWriters {
			// We must convert the writer to a console writer to correctly check for existence within the list
			if w.(zerolog.ConsoleWriter).Out == writer {
				// Writer already exists, return
				return
			}
		}
		// Add the unstructured writer and recreate the logger
		l.unstructuredColorWriters = append(l.unstructuredColorWriters, unstructuredWriter)
		l.unstructuredColorLogger = zerolog.New(zerolog.MultiLevelWriter(l.unstructuredColorWriters...)).Level(l.level).With().Timestamp().Logger()
	}

	// Otherwise, try to add the writer to the list of channels that want unstructured, un-colored logs
	if format == UNSTRUCTURED && !colored {
		for _, w := range l.unstructuredWriters {
			// We must convert the writer to a console writer to correctly check for existence within the list
			if w.(zerolog.ConsoleWriter).Out == writer {
				// Writer already exists, return
				return
			}
		}
		// Add the unstructured writer and recreate the logger
		l.unstructuredWriters = append(l.unstructuredWriters, unstructuredWriter)
		l.unstructuredLogger = zerolog.New(zerolog.MultiLevelWriter(l.unstructuredWriters...)).Level(l.level).With().Timestamp().Logger()
	}
}

// RemoveWriter will remove a writer from the list of writers that the logger manages. The writer will be either removed
// from the list of structured, unstructured and colored, or unstructured and un-colored writers. If the same writer
// is receiving multiple types of log output (e.g. structured and unstructured with color) then this function must be called
// multiple times. If the writer does not exist in any list, then this function is a no-op.
func (l *Logger) RemoveWriter(writer io.Writer, format LogFormat, colored bool) {
	// First, try to remove the writer from the list of structured writers
	if format == STRUCTURED {
		// Check for writer existence
		for i, w := range l.structuredWriters {
			if w == writer {
				// Remove the writer and recreate the logger
				l.structuredWriters = append(l.structuredWriters[:i], l.structuredWriters[i+1:]...)
				l.structuredLogger = zerolog.New(zerolog.MultiLevelWriter(l.structuredWriters...)).Level(l.level).With().Timestamp().Logger()
			}
		}
	}

	// Now, try to remove the writer from the list of unstructured, colored writers
	if format == UNSTRUCTURED && colored {
		// Check for writer existence
		for i, w := range l.unstructuredColorWriters {
			// We must convert the writer to a console writer to correctly check for existence within the list
			if w.(zerolog.ConsoleWriter).Out == writer {
				// Remove the writer and recreate the logger
				l.unstructuredColorWriters = append(l.unstructuredColorWriters[:i], l.unstructuredColorWriters[i+1:]...)
				l.unstructuredColorLogger = zerolog.New(zerolog.MultiLevelWriter(l.unstructuredColorWriters...)).Level(l.level).With().Timestamp().Logger()
			}
		}
	}

	// Otherwise, try to remove the writer from the list of unstructured, un-colored writers
	if format == UNSTRUCTURED && !colored {
		// Check for writer existence
		for i, w := range l.unstructuredWriters {
			// We must convert the writer to a console writer to correctly check for existence within the list
			if w.(zerolog.ConsoleWriter).Out == writer {
				// Remove the writer and recreate the logger
				l.unstructuredWriters = append(l.unstructuredWriters[:i], l.unstructuredWriters[i+1:]...)
				l.unstructuredLogger = zerolog.New(zerolog.MultiLevelWriter(l.unstructuredWriters...)).Level(l.level).With().Timestamp().Logger()
			}
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

	// Update the level of each underlying logger
	l.structuredLogger = l.structuredLogger.Level(level)
	l.unstructuredColorLogger = l.unstructuredColorLogger.Level(level)
	l.unstructuredLogger = l.unstructuredLogger.Level(level)

}

// Trace is a wrapper function that will log a trace event
func (l *Logger) Trace(args ...any) {
	// Build the messages and retrieve any errors or associated structured log info
	colorMsg, noColorMsg, errs, info := buildMsgs(args...)

	// Instantiate log events
	structuredLog := l.structuredLogger.Trace()
	unstructuredColoredLog := l.unstructuredColorLogger.Trace()
	unstructuredLog := l.unstructuredLogger.Trace()

	// Chain the structured log info, errors, and messages and send off the logs
	chainStructuredLogInfoErrorsAndMsgs(structuredLog, unstructuredColoredLog, unstructuredLog, info, errs, colorMsg, noColorMsg)
}

// Debug is a wrapper function that will log a debug event
func (l *Logger) Debug(args ...any) {
	// Build the messages and retrieve any errors or associated structured log info
	colorMsg, noColorMsg, errs, info := buildMsgs(args...)

	// Instantiate log events
	structuredLog := l.structuredLogger.Debug()
	unstructuredColoredLog := l.unstructuredColorLogger.Debug()
	unstructuredLog := l.unstructuredLogger.Debug()

	// Chain the structured log info, errors, and messages and send off the logs
	chainStructuredLogInfoErrorsAndMsgs(structuredLog, unstructuredColoredLog, unstructuredLog, info, errs, colorMsg, noColorMsg)
}

// Info is a wrapper function that will log an info event
func (l *Logger) Info(args ...any) {
	// Build the messages and retrieve any errors or associated structured log info
	colorMsg, noColorMsg, errs, info := buildMsgs(args...)

	// Instantiate log events
	structuredLog := l.structuredLogger.Info()
	unstructuredColoredLog := l.unstructuredColorLogger.Info()
	unstructuredLog := l.unstructuredLogger.Info()

	// Chain the structured log info, errors, and messages and send off the logs
	chainStructuredLogInfoErrorsAndMsgs(structuredLog, unstructuredColoredLog, unstructuredLog, info, errs, colorMsg, noColorMsg)
}

// Warn is a wrapper function that will log a warning event both on console
func (l *Logger) Warn(args ...any) {
	// Build the messages and retrieve any errors or associated structured log info
	colorMsg, noColorMsg, errs, info := buildMsgs(args...)

	// Instantiate log events
	structuredLog := l.structuredLogger.Warn()
	unstructuredColoredLog := l.unstructuredColorLogger.Warn()
	unstructuredLog := l.unstructuredLogger.Warn()

	// Chain the structured log info, errors, and messages and send off the logs
	chainStructuredLogInfoErrorsAndMsgs(structuredLog, unstructuredColoredLog, unstructuredLog, info, errs, colorMsg, noColorMsg)
}

// Error is a wrapper function that will log an error event.
func (l *Logger) Error(args ...any) {
	// Build the messages and retrieve any errors or associated structured log info
	colorMsg, noColorMsg, errs, info := buildMsgs(args...)

	// Instantiate log events
	structuredLog := l.structuredLogger.Error()
	unstructuredColoredLog := l.unstructuredColorLogger.Error()
	unstructuredLog := l.unstructuredLogger.Error()

	// Chain the structured log info, errors, and messages and send off the logs
	chainStructuredLogInfoErrorsAndMsgs(structuredLog, unstructuredColoredLog, unstructuredLog, info, errs, colorMsg, noColorMsg)
}

// Panic is a wrapper function that will log a panic event
func (l *Logger) Panic(args ...any) {
	// Build the messages and retrieve any errors or associated structured log info
	colorMsg, noColorMsg, errs, info := buildMsgs(args...)

	// Instantiate log events
	structuredLog := l.structuredLogger.Panic()
	unstructuredColoredLog := l.unstructuredColorLogger.Panic()
	unstructuredLog := l.unstructuredLogger.Panic()

	// Chain the structured log info, errors, and messages and send off the logs
	chainStructuredLogInfoErrorsAndMsgs(structuredLog, unstructuredColoredLog, unstructuredLog, info, errs, colorMsg, noColorMsg)
}

// buildMsgs describes a function that takes in a variadic list of arguments of any type and returns two strings and,
// optionally, a list of errors and a StructuredLogInfo object. The first string will be a colorized-message while the
// second string will be a non-colorized one. Colors are applied if one or more of the input arguments are of type
// colors.ColorFunc. The colorized message can be used for channels that request unstructured, colorized log output
// while the non-colorized one can be used for structured streams and unstructured streams that don't want color. The
// errors and the StructuredLogInfo can be used to add additional context to log messages.
func buildMsgs(args ...any) (string, string, []error, StructuredLogInfo) {
	// Guard clause
	if len(args) == 0 {
		return "", "", nil, nil
	}

	// Initialize the base color context, the string buffers and the structured log info object
	colorCtx := colors.Reset
	colorMsg := make([]string, 0)
	noColorMsg := make([]string, 0)
	errs := make([]error, 0)
	var info StructuredLogInfo

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
			// Append error to the list of errors
			errs = append(errs, t)
		default:
			// In the base case, append the object to the two string buffers. The colored string buffer will have the
			// current color context applied to it.
			colorMsg = append(colorMsg, colorCtx(t))
			noColorMsg = append(noColorMsg, fmt.Sprintf("%v", t))
		}
	}

	return strings.Join(colorMsg, ""), strings.Join(noColorMsg, ""), errs, info
}

// chainStructuredLogInfoErrorsAndMsgs describes a function that takes in a *zerolog.Event for the structured, unstructured
// with color, and unstructured without colors log streams, chains any StructuredLogInfo and errors provided to it,
// adds the associated messages, and sends out the logs to their respective channels. Note that the StructuredLogInfo object
// is only appended to the structured log event and not to the unstructured ones. Additionally, note that errors are appended as a
// formatted bulleted list for unstructured logging while for the structured logger they get appended as a key-value pair.
func chainStructuredLogInfoErrorsAndMsgs(structuredLog *zerolog.Event, unstructuredColoredLog *zerolog.Event, unstructuredLog *zerolog.Event, info StructuredLogInfo, errs []error, colorMsg string, noColorMsg string) {
	// First, we need to create a formatted error string for unstructured output
	var errStr string
	for _, err := range errs {
		// Append a bullet point and the formatted error to the error string
		errStr += "\n" + colors.BULLET_POINT + " " + err.Error()
	}

	// Add structured error element to the multi-log output and append the error string to the console message
	// TODO: Add support for stack traces in the future
	if len(errs) != 0 {
		structuredLog.Errs("errors", errs)
	}

	// The structured message will be the one without any potential errors appended to it since the errors will be provided
	// as a key-value pair
	structuredMsg := noColorMsg

	// Add the colorized and non-colorized version of the error string to the colorized and non-colorized messages, respectively.
	if len(errStr) > 0 {
		colorMsg += colors.Red(errStr)
		noColorMsg += errStr
	}

	// If we are provided a structured log info object, add that as a key-value pair to the structured log event
	if info != nil {
		structuredLog.Any("info", info)
	}

	// Append the messages to each event. This will also result in the log events being sent out to their respective
	// streams. Note that we are deferring the message to two of the three loggers multi logger in case we are logging a panic
	// and want to make sure that all channels receive the panic log.
	defer func() {
		structuredLog.Msg(structuredMsg)
		unstructuredLog.Msg(noColorMsg)
	}()
	unstructuredColoredLog.Msg(colorMsg)
}

// formatUnstructuredWriter will create a custom-formatted zerolog.ConsoleWriter from an arbitrary io.Writer. A zerolog.ConsoleWriter is
// what is used under-the-hood to support unstructured log output. Custom formatting is applied to specific fields,
// timestamps, and the log level strings. If requested, coloring may be applied to the log level strings.
func formatUnstructuredWriter(writer io.Writer, level zerolog.Level, colored bool) zerolog.ConsoleWriter {
	// Create the console writer
	consoleWriter := zerolog.ConsoleWriter{Out: writer, NoColor: !colored}

	// Get rid of the timestamp for unstructured output
	consoleWriter.FormatTimestamp = func(i interface{}) string {
		return ""
	}

	// If we are above debug level, we want to get rid of the `module` component when logging to unstructured streams
	if level > zerolog.DebugLevel {
		consoleWriter.FieldsExclude = []string{"module"}
	}

	// If coloring is enabled, we will return a custom, colored string for each log severity level
	// Otherwise, we will just return a non-colorized string for each log severity level
	consoleWriter.FormatLevel = func(i any) string {
		// Create a level object for better switch logic
		level, err := zerolog.ParseLevel(i.(string))
		if err != nil {
			panic(fmt.Sprintf("unable to parse the log level: %v", err))
		}

		// Switch on the level
		switch level {
		case zerolog.TraceLevel:
			if !colored {
				// No coloring for "trace" string
				return zerolog.LevelTraceValue
			}
			// Return a bold, cyan "trace" string
			return colors.CyanBold(zerolog.LevelTraceValue)
		case zerolog.DebugLevel:
			if !colored {
				// No coloring for "debug" string
				return zerolog.LevelDebugValue
			}
			// Return a bold, blue "debug" string
			return colors.BlueBold(zerolog.LevelDebugValue)
		case zerolog.InfoLevel:
			if !colored {
				// Return a left arrow without any coloring
				return colors.LEFT_ARROW
			}
			// Return a bold, green left arrow
			return colors.GreenBold(colors.LEFT_ARROW)
		case zerolog.WarnLevel:
			if !colored {
				// No coloring for "warn" string
				return zerolog.LevelWarnValue
			}
			// Return a bold, yellow "warn" string
			return colors.YellowBold(zerolog.LevelWarnValue)
		case zerolog.ErrorLevel:
			if !colored {
				// No coloring for "err" string
				return zerolog.LevelErrorValue
			}
			// Return a bold, red "err" string
			return colors.RedBold(zerolog.LevelErrorValue)
		case zerolog.FatalLevel:
			if !colored {
				// No coloring for "fatal" string
				return zerolog.LevelFatalValue
			}
			// Return a bold, red "fatal" string
			return colors.RedBold(zerolog.LevelFatalValue)
		case zerolog.PanicLevel:
			if !colored {
				// No coloring for "panic" string
				return zerolog.LevelPanicValue
			}
			// Return a bold, red "panic" string
			return colors.RedBold(zerolog.LevelPanicValue)
		default:
			return i.(string)
		}
	}

	return consoleWriter
}
