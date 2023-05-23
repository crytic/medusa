package formatters

import "github.com/crytic/medusa/logging/colors"

// The list of constants below are used to search and replace various elements of a call sequence, test case, execution trace,
// or test summary with a colorized, formatted version for console output
const (
	// passedRegex is the regex to find [PASSED] in the execution trace
	passedRegex = `(\[PASSED\])`
	// failedRegex is the regex to find [FAILED] in the execution trace
	failedRegex = `(\[FAILED\])`
	// callRegex is the regex to find [call] in the execution trace
	callRegex = `(\[call\])`
	// proxyRegex is the regex to find [proxy call] in the execution trace
	proxyRegex = `(\[proxy call\])`
	// creationRegex is the regex to find [creation] in the execution trace
	creationRegex = `(\[creation\])`
	// eventRegex is the regex to find [event] in the execution trace
	eventRegex = `(\[event\])`
	// assertionFailedRegex is the regex to find [assertion failed] in the execution trace
	assertionFailedRegex = `(\[assertion failed\])`
	// selfDestructRegex is the regex to find [selfdestruct] in the execution trace
	selfDestructRegex = `(\[selfdestruct\])`
	// revertRegex is the regex to find [revert (%v)] in the execution trace
	revertRegex = `(\[revert \(.*\)\])`
	// vmErrorRegex is the regex to find [vm error (%v)] in the execution trace
	vmErrorRegex = `(\[vm error \(.*\)\])`
	// returnRegex is the regex to find [return (%v)] in the execution trace
	returnRegex = `(\[return \(.*\)\])`
	// doubleLeftArrowRegex is the regex to find => in the execution trace
	doubleLeftArrowRegex = `(\=\>)`
	// leftArrowRegex is the regex to find -> in the execution trace
	leftArrowRegex = `(\-\>)`
	// executionTraceRegex is the regex to find [Execution Trace] in the execution trace
	executionTraceRegex = `(\[Execution Trace\])`
	// callSequenceRegex is the regex to find [Call Sequence] in the call sequence
	callSequenceRegex = `(\[Call Sequence\])`
	// integerRegex is the regex used to capture all integer and non-integer parts of a test summary string
	testSummaryRegex = `([-+]?\d+|\D+)`
)

// The list of constants below are used to map a specific color to a specific type of text for console output
const (
	// passedColor is the color to use for [PASSED] in the execution trace or the number of passed test cases
	passedColor = colors.COLOR_GREEN
	// returnColor is the color to use for [return (%v)] in the execution trace
	returnColor = colors.COLOR_GREEN
	// failedColor is the color to use for [FAILED] in the execution trace or the number of failed test cases
	failedColor = colors.COLOR_RED
	// revertColor is the color to use for [revert (%v)] in the execution trace
	revertColor = colors.COLOR_RED
	// vmErrorColor is the color to use for [vm error (%v)] in the execution trace
	vmErrorColor = colors.COLOR_RED
	// assertionFailedColor is the color to use for [assertion failed] in the execution trace
	assertionFailedColor = colors.COLOR_RED
	// callColor is the color to use for [call] in the execution trace
	callColor = colors.COLOR_BLUE
	// proxyColor is the color to use for [proxy call] in the execution trace
	proxyColor = colors.COLOR_CYAN
	// creationColor is the color to use for [creation] in the execution trace
	creationColor = colors.COLOR_YELLOW
	// eventColor is the color to use for [event] in the execution trace
	eventColor = colors.COLOR_MAGENTA
	// selfDestructColor is the color to use for [selfdestruct] in the execution trace
	selfDestructColor = colors.COLOR_MAGENTA
)
