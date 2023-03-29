package formatters

import (
	"github.com/trailofbits/medusa/logging/colors"
	"regexp"
	"strings"
)

// TestSummaryFormatter will colorize and update the format of the test summary for console output
func TestSummaryFormatter(fields map[string]any, msg string) string {
	// Use testSummaryRegex to split the summary into its integer and non-integer parts
	re := regexp.MustCompile(testSummaryRegex)
	matches := re.FindAllString(msg, -1)

	// The first index and 3rd index are the number of passed and failed tests, respectively
	matches[1] = colors.Colorize(colors.Colorize(matches[1], passedColor), colors.COLOR_BOLD)
	matches[3] = colors.Colorize(colors.Colorize(matches[3], failedColor), colors.COLOR_BOLD)

	// Merge the string back together
	return strings.Join(matches, "")
}
