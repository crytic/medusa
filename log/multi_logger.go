package log

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/trailofbits/medusa/utils"
	"os"
	"strconv"
	"time"
)

type MultiLogger struct {
	// fileLogger describes the structured logger that will be used to output logs to a given file
	fileLogger *zerolog.Logger

	// consoleLogger describes the unstructured logger that will be used to output logs to console
	consoleLogger *zerolog.Logger
}

// NewMultiLogger will create a new MultiLogger that is capable of logging output to both console and / or file.
func NewMultiLogger(level zerolog.Level, logDirectory string, consoleEnabled bool) (*MultiLogger, error) {
	// Set global parameters such as using enabling stack traces in logs and using unix timestamp (more efficient)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// TODO: Add caller to logging
	// The two base loggers are effectively loggers that are disabled
	// We are creating instances of them so that we do not get null pointer dereferences down the line
	baseFileLogger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	baseConsoleLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.Disabled)

	// Guard clause to do no more computation if logging is disabled
	if logDirectory == "" && !consoleEnabled {
		return &MultiLogger{
			fileLogger:    &baseFileLogger,
			consoleLogger: &baseConsoleLogger,
		}, nil
	}

	// First, we will create a file logger, if requested
	if logDirectory != "" {
		// Filename will be the "log-current_unix_timestamp.log"
		filename := "log-" + strconv.FormatInt(time.Now().Unix(), 10) + ".log"
		// Create the file
		_, err := utils.CreateFile(logDirectory, filename)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// TODO: Figure out if we can update level without re-instantiating logger
		baseFileLogger = zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
	}

	// Next, we will create a console logger, if requested
	if consoleEnabled {
		// TODO: Figure out if we can update level without re-instantiating logger
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr}
		baseConsoleLogger = zerolog.New(setupDefaultFormatting(consoleWriter)).Level(level)
	}
	return &MultiLogger{
		fileLogger:    &baseFileLogger,
		consoleLogger: &baseConsoleLogger,
	}, nil
}

// setupDefaultFormatting will update the console logger's formatting to the medusa standard
// TODO: Make this a struct function
func setupDefaultFormatting(logger zerolog.ConsoleWriter) zerolog.ConsoleWriter {
	logger.FormatTimestamp = func(i interface{}) string {
		return ""
	}
	return logger
}

// Trace is a wrapper function that will log a trace event both on console and/or into file
func (l *MultiLogger) Trace(msg string, fields Fields) {
	// Log to file
	l.fileLogger.Trace().Fields(fields).Msg(msg)

	// Log to console
	// msg = cases.Title(language.English, cases.NoLower).String(msg)
	l.consoleLogger.Trace().Fields(fields).Msg(msg)
}

// Debug is a wrapper function that will log a debug event both on console and/or into file
func (l *MultiLogger) Debug(msg string, fields Fields) {
	// Log to file
	l.fileLogger.Debug().Fields(fields).Msg(msg)

	// Log to console
	l.consoleLogger.Debug().Fields(fields).Msg(msg)
}

// Info is a wrapper function that will log an info event both on console and/or into the file
func (l *MultiLogger) Info(msg string, fields Fields) {
	// Log to file
	l.fileLogger.Info().Fields(fields).Msg(msg)

	// Log to console
	// msg = cases.Title(language.English, cases.NoLower).String(msg)
	l.infoToConsole(msg, fields)
}

// Warn is a wrapper function that will log a warning event both on console and/or into file
func (l *MultiLogger) Warn(msg string, fields Fields) {
	// Log to file
	l.fileLogger.Warn().Fields(fields).Msg(msg)

	// Log to console
	l.consoleLogger.Warn().Fields(fields).Msg(msg)
}

// Error is a wrapper function that will log an error event both on console and/or into file
func (l *MultiLogger) Error(msg string, fields Fields) {
	// Log to file
	l.fileLogger.Error().Stack().Err(fields["error"].(error)).Msg(msg)

	// Create colorized error message with an additional message, if provided
	err := fields["error"].(error)
	if msg != "" {
		err = errors.WithMessage(err, msg)
	}

	// Log to console
	l.consoleLogger.Error().Msg(colorize(err, COLOR_RED))
}

// Fatal is a wrapper function that will log a fatal event both on console and/or into file
func (l *MultiLogger) Fatal(msg string, fields Fields) {
	// Log to file
	l.fileLogger.Fatal().Fields(fields).Msg(msg)

	// Log to console
	l.consoleLogger.Fatal().Fields(fields).Msg(msg)
}

// Panic is a wrapper function that will log a panic event both on console and/or into file
func (l *MultiLogger) Panic(msg string, fields Fields) {
	// Log to file
	l.fileLogger.Panic().Fields(fields).Msg(msg)

	// Log to console
	l.consoleLogger.Panic().Fields(fields).Msg(msg)
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

// colorize returns the string s wrapped in ANSI code c, unless disabled is true.
// Source: https://github.com/rs/zerolog/blob/4fff5db29c3403bc26dee9895e12a108aacc0203/console.go
func colorize(s any, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

// Fields describes a mapping that allows any log event to contain unique key-value pairs.
type Fields map[string]any

// NewFields creates a new Fields object given any given number of args. Note that if the length of the input arguments
// is not divisible by 2, the last value is automatically truncated.
func NewFields(args ...any) Fields {
	fields := make(Fields, 0)

	// If the length of args is leq than 1, return an empty fields object
	if len(args) <= 1 {
		return fields
	}

	// Truncate the last arg if args is not a multiple of 2
	if len(args)%2 != 0 {
		args = args[:len(args)-1]
	}

	// Iteratively create args object
	for i := 0; i < len(args)-1; i = i + 2 {
		fields[args[i].(string)] = args[i+1]
	}

	return fields
}

type CompilationServiceFields map[string]any
