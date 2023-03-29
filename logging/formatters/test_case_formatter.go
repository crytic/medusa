package formatters

import (
	"github.com/trailofbits/medusa/logging/colors"
	"regexp"
)

// TestCaseFormatter will colorize and update the format of a test case, its call sequence, and execution trace for console output
func TestCaseFormatter(fields map[string]any, msg string) string {
	var re *regexp.Regexp

	// Colorize [Execution Trace]
	re = regexp.MustCompile(executionTraceRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(`$1`, colors.COLOR_BOLD))

	// Colorize [Call Sequence]
	re = regexp.MustCompile(callSequenceRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(`$1`, colors.COLOR_BOLD))

	// Colorize [PASSED]
	re = regexp.MustCompile(passedRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, passedColor), colors.COLOR_BOLD))

	// Colorize [FAILED]
	re = regexp.MustCompile(failedRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, failedColor), colors.COLOR_BOLD))

	// Colorize [call]
	re = regexp.MustCompile(callRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, callColor), colors.COLOR_BOLD))

	// Colorize [proxy call]
	re = regexp.MustCompile(proxyRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, proxyColor), colors.COLOR_BOLD))

	// Colorize [creation]
	re = regexp.MustCompile(creationRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, creationColor), colors.COLOR_BOLD))

	// Colorize [event]
	re = regexp.MustCompile(eventRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, eventColor), colors.COLOR_BOLD))

	// Colorize [assertion failed]
	re = regexp.MustCompile(assertionFailedRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, assertionFailedColor), colors.COLOR_BOLD))

	// Colorize [selfdestruct]
	re = regexp.MustCompile(selfDestructRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, selfDestructColor), colors.COLOR_BOLD))

	// Colorize [return (%v)]
	re = regexp.MustCompile(returnRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, returnColor), colors.COLOR_BOLD))

	// Colorize [revert (%v)]
	re = regexp.MustCompile(revertRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, revertColor), colors.COLOR_BOLD))

	// Colorize [vm error (%v)]
	re = regexp.MustCompile(vmErrorRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(`$1`, vmErrorColor), colors.COLOR_BOLD))

	// Colorize and replace '=>'
	re = regexp.MustCompile(doubleLeftArrowRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(colors.DOWNWARD_LEFT_ARROW, colors.COLOR_GREEN), colors.COLOR_BOLD))

	// Colorize and replace '->'
	re = regexp.MustCompile(leftArrowRegex)
	msg = re.ReplaceAllString(msg, colors.Colorize(colors.Colorize(colors.LEFT_ARROW, colors.COLOR_GREEN), colors.COLOR_BOLD))

	return msg
}
