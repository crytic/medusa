package log

// This is taken from zerolog's repo and will be used to colorize log output
// Source: https://github.com/rs/zerolog/blob/4fff5db29c3403bc26dee9895e12a108aacc0203/console.go
const (
	COLOR_BLACK = iota + 30
	COLOR_RED
	COLOR_GREEN
	COLOR_YELLOW
	COLOR_BLUE
	COLOR_MAGENTA
	COLOR_CYAN
	COLOR_WHITE

	COLOR_BOLD      = 1
	COLOR_DARK_GRAY = 90
)

// These constants are used to identify specialized formatting for various logs to console
const (
	// TEST_CASE_RESULT is the constant to identify that a test case result needs special console formatting
	TEST_CASE_RESULT = "testCaseResult"
	// TESTING_SUMMARY is the constant o identify that the testing summary needs special console formatting
	TESTING_SUMMARY = "testSummary"
)

// These constants are used to identify the various services that may do some logging
const (
	// COMPILATION_SERVICE is the constant used to identify the compilation package
	COMPILATION_SERVICE = "compilation"

	// FUZZING_SERVICE is the constant used to identify the fuzzing package
	FUZZING_SERVICE = "fuzzing"

	// CLI_SERVICE is the constant used to identify the cmd package
	CLI_SERVICE = "cli"
)
