package exitcodes

const (
	// ================================
	// Platform-universal exit codes
	// ================================

	// ExitCodeSuccess indicates no errors or failures had occurred.
	ExitCodeSuccess = 0

	// ExitCodeGeneralError indicates some type of general error occurred.
	ExitCodeGeneralError = 1

	// ================================
	// Application-specific exit codes
	// ================================
	// Note: Despite not being standardized, exit codes 2-5 are often used for common use cases, so we avoid them.

	// ExitCodeTestFailed indicates a test case had failed.
	ExitCodeTestFailed = 7
)
