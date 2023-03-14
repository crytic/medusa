package logging

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/trailofbits/medusa/utils"
	"os"
	"strconv"
	"strings"
	"time"
)

// GlobalLogger describes a MultiLogger that is instantiated when a fuzzer is created and is then used to create
// sub-loggers for each individual module / package. This allows to create unique logging instances depending on the use case.
var GlobalLogger *MultiLogger

// MultiLogger describes a custom logging object that can log events both to console and file
type MultiLogger struct {
	// fileLogger describes the structured logger that will be used to output logs to a given file
	fileLogger zerolog.Logger

	// consoleLogger describes the unstructured logger that will be used to output logs to console
	consoleLogger zerolog.Logger
}

// NewMultiLogger will create a MultiLogger object with a specific log level. The MultiLogger may output to file, console,
// both, or neither
func NewMultiLogger(level zerolog.Level, logDirectory string, consoleEnabled bool) (*MultiLogger, error) {
	// Set global parameters such as using enabling stack traces in logs and using unix timestamps for file logging
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// The two base loggers are effectively loggers that are disabled
	// We are creating instances of them so that we do not get nil pointer dereferences down the line
	baseFileLogger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	baseConsoleLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.Disabled)

	// First, we will create a file logger, if requested
	if logDirectory != "" {
		// Filename will be the "log-current_unix_timestamp.log"
		filename := "log-" + strconv.FormatInt(time.Now().Unix(), 10) + ".log"
		// Create the file
		// TODO: Add the file back after testing is done
		_, err := utils.CreateFile(logDirectory, filename)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		baseFileLogger = zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
	}

	// Next, we will create a console logger, if requested
	if consoleEnabled {
		consoleWriter := setupDefaultFormatting(zerolog.ConsoleWriter{Out: os.Stderr}, level)
		baseConsoleLogger = zerolog.New(consoleWriter).Level(level)
	}

	return &MultiLogger{
		fileLogger:    baseFileLogger,
		consoleLogger: baseConsoleLogger,
	}, nil
}

// NewSubLogger will create a new MultiLogger with unique context in the form of a key-value pair. The expected use of this
// function is for each package to have their own unique logger so that parsing of logs is "grepable" based on some key
func (l *MultiLogger) NewSubLogger(key string, value string) *MultiLogger {
	subFileLogger := l.fileLogger.With().Str(key, value).Logger()
	subConsoleLonger := l.consoleLogger.With().Str(key, value).Logger()
	return &MultiLogger{
		fileLogger:    subFileLogger,
		consoleLogger: subConsoleLonger,
	}
}

// SetLevel will update the log level of the MultiLogger which means the level for both the console and file logger
func (l *MultiLogger) SetLevel(level zerolog.Level) {
	l.fileLogger = l.fileLogger.Level(level)
	l.consoleLogger = l.consoleLogger.Level(level)
}

// SetConsoleLoggerLevel will update the log level of the console logger
func (l *MultiLogger) SetConsoleLoggerLevel(level zerolog.Level) {
	l.consoleLogger = l.consoleLogger.Level(level)
}

// SetFileLoggerLevel will update the log level of the file logger
func (l *MultiLogger) SetFileLoggerLevel(level zerolog.Level) {
	l.fileLogger = l.fileLogger.Level(level)
}

// Trace is a wrapper function that will log a trace event both on console and/or into file
// No additional formatting is done for console since this will be used mostly for debugging purposes
func (l *MultiLogger) Trace(msg string, fields map[string]any) {
	// Log to file
	l.fileLogger.Trace().Fields(fields).Msg(msg)

	// Log to console
	l.consoleLogger.Trace().Fields(fields).Msg(msg)
}

// Debug is a wrapper function that will log a debug event both on console and/or into file
// No additional formatting is done for console since this will be used mostly for debugging purposes
func (l *MultiLogger) Debug(msg string, fields map[string]any) {
	// Log to file
	l.fileLogger.Debug().Fields(fields).Msg(msg)

	// Log to console
	l.consoleLogger.Debug().Fields(fields).Msg(msg)
}

// Info is a wrapper function that will log an info event both on console and/or into the file
// Specific log events will have custom formatting for console
func (l *MultiLogger) Info(msg string, fields map[string]any) {
	// Log to file
	l.fileLogger.Info().Fields(fields).Msg(msg)

	// Log to console
	l.infoToConsole(msg, fields)
}

// Warn is a wrapper function that will log a warning event both on console and/or into file
func (l *MultiLogger) Warn(msg string, fields map[string]any) {
	// Log to file
	l.fileLogger.Warn().Fields(fields).Msg(msg)

	// Log to console
	// If we are in debug mode or below, add the fields. Otherwise, just output the message
	if l.consoleLogger.GetLevel() <= zerolog.DebugLevel {
		l.consoleLogger.Warn().Fields(fields).Msg(msg)
	} else {
		l.consoleLogger.Warn().Msg(msg)
	}
}

// Error is a wrapper function that will log an error event both on console and/or into file
// Note that if the error key is not in the fields mapping, this function will panic
func (l *MultiLogger) Error(msg string, fields map[string]any) {
	// Grab error from fields
	err, fields := getErrorFromFields(fields)

	// Log to file
	l.fileLogger.Error().Err(err).Stack().Fields(fields).Msg(msg)

	// If we are in debug mode or below, log the stack and error as fields. Otherwise, output a colorized error message
	if l.consoleLogger.GetLevel() <= zerolog.DebugLevel {
		l.consoleLogger.Error().Err(err).Stack().Msg(msg)
	} else {
		// Create colorized error message with an additional message
		if msg != "" {
			err = errors.WithMessage(err, msg)
		}
		l.consoleLogger.Error().Msg(colorize(err, COLOR_RED))
	}
}

// Fatal is a wrapper function that will log a fatal event both on console and/or into file
// Note that if the error key is not in the fields mapping, this function will panic
func (l *MultiLogger) Fatal(msg string, fields map[string]any) {
	// Grab error from fields
	err, fields := getErrorFromFields(fields)

	// Log to file
	l.fileLogger.Fatal().Err(err).Stack().Fields(fields).Msg(msg)

	// Log to console with no regard for formatting, all bets are off
	l.consoleLogger.Fatal().Err(err).Stack().Fields(fields).Msg(msg)
}

// Panic is a wrapper function that will log a panic event both on console and/or into file
// Note that if the error key is not in the fields mapping, this function will, ironically, panic
func (l *MultiLogger) Panic(msg string, fields map[string]any) {
	// Grab error from fields
	err, fields := getErrorFromFields(fields)

	// Log to file
	l.fileLogger.Panic().Err(err).Stack().Fields(fields).Msg(msg)

	// Log to console with no regard for formatting, all bets are off
	l.consoleLogger.Panic().Err(err).Stack().Fields(fields).Msg(msg)
}

// getErrorFromFields will grab the error from the fields mapping and then delete the error from the fields mapping
// Note that if fields does not have an error key, this function will panic
func getErrorFromFields(fields map[string]any) (error, map[string]any) {
	// Grab error before removing it from fields
	err := fields["error"].(error)
	delete(fields, "error")

	return err, fields
}

func (l *MultiLogger) infoToConsole(msg string, fields map[string]any) {
	switch fields["format"] {
	case TEST_CASE_RESULT:
		switch fields["status"].(string) {
		case "PASSED":
			statusString := colorize(colorize(fmt.Sprintf("[%s]", fields["status"]), COLOR_GREEN), COLOR_BOLD)
			finalMsg := fmt.Sprintf("%s %s\n%s", statusString, fields["name"].(string), fields["message"].(string))
			l.consoleLogger.Info().Msg(finalMsg)
		case "FAILED":
			statusString := colorize(colorize(fmt.Sprintf("[%s]", fields["status"]), COLOR_RED), COLOR_BOLD)
			msgString := colorize(fields["message"], COLOR_RED)
			finalMsg := fmt.Sprintf("%s %s\n%s", statusString, fields["name"].(string), msgString)
			l.consoleLogger.Info().Msg(finalMsg)
		default:
		}
	case TESTING_SUMMARY:
		passedString := colorize(colorize(fields["passed"], COLOR_GREEN), COLOR_BOLD)
		failedString := colorize(colorize(fields["failed"], COLOR_RED), COLOR_BOLD)
		finalMsg := fmt.Sprintf("%s: %s passed and %s failed tests", msg, passedString, failedString)
		l.consoleLogger.Info().Msg(finalMsg)
	default:
		l.consoleLogger.Info().Msg(msg)
	}
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
			return colorize(colorize(strings.ToUpper(zerolog.LevelTraceValue), COLOR_CYAN), COLOR_BOLD)
		case zerolog.DebugLevel:
			// Return a bold, blue "debug" string
			return colorize(colorize(strings.ToUpper(zerolog.LevelDebugValue), COLOR_BLUE), COLOR_BOLD)
		case zerolog.InfoLevel:
			// Return a bold, green left arrow
			return colorize(colorize(LEFT_ARROW, COLOR_GREEN), COLOR_BOLD)
		case zerolog.WarnLevel:
			// Return a bold, yellow "warn" string
			return colorize(colorize(strings.ToUpper(zerolog.LevelWarnValue), COLOR_YELLOW), COLOR_BOLD)
		case zerolog.ErrorLevel:
			// Return a bold, red "err" string
			return colorize(colorize(strings.ToUpper(zerolog.LevelErrorValue), COLOR_RED), COLOR_BOLD)
		case zerolog.FatalLevel:
			// Return a bold, red "fatal" string
			return colorize(colorize(strings.ToUpper(zerolog.LevelFatalValue), COLOR_RED), COLOR_BOLD)
		case zerolog.PanicLevel:
			// Return a bold, red "panic" string
			return colorize(colorize(strings.ToUpper(zerolog.LevelPanicValue), COLOR_RED), COLOR_BOLD)
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

// colorize returns the string s wrapped in ANSI code c, unless disabled is true.
// Source: https://github.com/rs/zerolog/blob/4fff5db29c3403bc26dee9895e12a108aacc0203/console.go
func colorize(s any, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
