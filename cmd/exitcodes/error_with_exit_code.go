package exitcodes

// ErrorWithExitCode is an `error` type that wraps an existing error and exit code, providing exit codes
// for a given error if they are bubbled up to the top-level.
type ErrorWithExitCode struct {
	err      error
	exitCode int
}

// NewErrorWithExitCode creates a new error (ErrorWithExitCode) with the provided internal error and exit code.
func NewErrorWithExitCode(err error, exitCode int) *ErrorWithExitCode {
	return &ErrorWithExitCode{
		err:      err,
		exitCode: exitCode,
	}
}

// Error returns the error message string, implementing the `error` interface.
func (e *ErrorWithExitCode) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

// GetInnerErrorAndExitCode checks the given exit code that the application should exit with, if this error is bubbled
// to the top-level. This will be 0 for a nil error, 1 for a generic error, or arbitrary if the error is of type
// ErrorWithExitCode.
// Returns the error (or inner error if it is an ErrorWithExitCode error type), along with the exit code associated
// with the error.
func GetInnerErrorAndExitCode(err error) (error, int) {
	// If we have no error, return 0, if we have a generic error, return 1, if we have a custom error code, unwrap
	// and return it.
	if err == nil {
		return nil, ExitCodeSuccess
	} else if unwrappedErr, ok := err.(*ErrorWithExitCode); ok {
		return unwrappedErr.err, unwrappedErr.exitCode
	} else {
		return err, ExitCodeGeneralError
	}
}
