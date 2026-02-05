package tui

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/logging"
)

// renderFailureScreen renders the failure summary when tests fail
func (m Model) renderFailureScreen(failedTests []fuzzing.TestCase) string {
	var lines []string

	// Show failed test information with full traces
	lines = append(lines, logging.ErrorStyle.Render(fmt.Sprintf("Failed Tests (%d):", len(failedTests))))
	lines = append(lines, "")

	for i, tc := range failedTests {
		lines = append(lines, logging.ErrorStyle.Render(fmt.Sprintf("✗ %s", tc.Name())))
		lines = append(lines, "")

		// Show full message which includes call sequence and execution trace
		message := tc.Message()
		messageLines := strings.Split(message, "\n")

		skipFirstName := false
		for _, line := range messageLines {
			// Skip the status line (e.g., "[FAILED] Property Test: ...")
			if strings.HasPrefix(strings.TrimSpace(line), "[") && strings.Contains(line, tc.Name()) {
				skipFirstName = true
				continue
			}

			// Add all other lines with indentation for readability
			if strings.TrimSpace(line) != "" {
				lines = append(lines, "  "+line)
			} else if skipFirstName {
				// Allow empty lines after we've started showing content
				lines = append(lines, "")
			}
		}

		lines = append(lines, "")
		// Add separator between test failures
		if i < len(failedTests)-1 {
			separatorWidth := m.width - 4
			if separatorWidth > 80 {
				separatorWidth = 80
			}
			if separatorWidth < 20 {
				separatorWidth = 20
			}
			lines = append(lines, logging.MutedStyle.Render(strings.Repeat("─", separatorWidth)))
			lines = append(lines, "")
		}
	}

	// Final statistics
	elapsed := time.Since(m.startTime)
	callsTested := big.NewInt(0)
	sequencesTested := big.NewInt(0)
	if metrics := m.provider.Metrics(); metrics != nil {
		callsTested = metrics.CallsTested()
		sequencesTested = metrics.SequencesTested()
	}

	lines = append(lines, "")
	lines = append(lines, logging.TitleStyle.Render("Campaign Statistics:"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Total Time: %s", logging.FormatDuration(elapsed)))
	lines = append(lines, fmt.Sprintf("  Calls Tested: %s", logging.FormatNumber(callsTested)))
	lines = append(lines, fmt.Sprintf("  Sequences Tested: %s", logging.FormatNumber(sequencesTested)))

	return strings.Join(lines, "\n")
}

// renderTraceView renders the detailed trace view for a selected failed test
// updateTraceViewContent updates the viewport content for trace view
// This should be called from Update() when content changes, not from View()
func (m *Model) updateTraceViewContent() {
	failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
	if len(failedTests) == 0 {
		m.viewport.SetContent("No failed tests to display")
		return
	}

	// Ensure selected index is valid
	if m.selectedTestIdx < 0 || m.selectedTestIdx >= len(failedTests) {
		m.selectedTestIdx = 0
	}

	selectedTest := failedTests[m.selectedTestIdx]

	// Get the full trace message
	message := selectedTest.Message()
	messageLines := strings.Split(message, "\n")

	var lines []string
	skipFirstName := false
	for _, line := range messageLines {
		// Skip the status line (e.g., "[FAILED] Property Test: ...")
		if strings.HasPrefix(strings.TrimSpace(line), "[") && strings.Contains(line, selectedTest.Name()) {
			skipFirstName = true
			continue
		}

		// Add all other lines
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		} else if skipFirstName {
			// Allow empty lines after we've started showing content
			lines = append(lines, "")
		}
	}

	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)
}

func (m Model) renderTraceView() string {
	failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
	if len(failedTests) == 0 {
		return "No failed tests to display"
	}

	// Ensure selected index is valid
	if m.selectedTestIdx < 0 || m.selectedTestIdx >= len(failedTests) {
		m.selectedTestIdx = 0
	}

	selectedTest := failedTests[m.selectedTestIdx]

	// Build header showing which test we're viewing
	header := logging.HeaderStyle.Width(m.width).Render(
		fmt.Sprintf("TEST TRACE (%d/%d): %s", m.selectedTestIdx+1, len(failedTests), selectedTest.Name()),
	)

	// Footer
	footer := m.renderFooter()

	// Content was already set by updateTraceViewContent() in Update()
	// Combine: header + viewport + footer
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.viewport.View(),
		footer,
	)
}

// renderExitScreen renders the exit summary
func (m Model) renderExitScreen() string {
	var lines []string

	lines = append(lines, logging.HeaderStyle.Width(m.width).Render("FUZZING STOPPED"))
	lines = append(lines, "")

	// Final statistics
	elapsed := time.Since(m.startTime)
	callsTested := big.NewInt(0)
	sequencesTested := big.NewInt(0)
	failedSequences := big.NewInt(0)
	if metrics := m.provider.Metrics(); metrics != nil {
		callsTested = metrics.CallsTested()
		sequencesTested = metrics.SequencesTested()
		failedSequences = metrics.FailedSequences()
	}

	// Get coverage with nil checks
	branches := uint64(0)
	if corpus := m.provider.Corpus(); corpus != nil {
		if coverageMaps := corpus.CoverageMaps(); coverageMaps != nil {
			branches = coverageMaps.BranchesHit()
		}
	}

	lines = append(lines, logging.TitleStyle.Render("Final Statistics:"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Total Time: %s", logging.FormatDuration(elapsed)))
	lines = append(lines, fmt.Sprintf("  Calls Tested: %s", logging.FormatNumber(callsTested)))
	lines = append(lines, fmt.Sprintf("  Sequences Tested: %s", logging.FormatNumber(sequencesTested)))
	lines = append(lines, fmt.Sprintf("  Branches Hit: %d", branches))
	lines = append(lines, fmt.Sprintf("  Test Failures: %s", logging.FormatNumber(failedSequences)))
	lines = append(lines, "")
	lines = append(lines, logging.MutedStyle.Render("Check the logs for detailed test results."))

	return strings.Join(lines, "\n")
}

// updateLogViewContent updates the viewport content for log view
// This should be called from Update() when content changes, not from View()
func (m *Model) updateLogViewContent() {
	if m.logBuffer == nil {
		m.logsViewport.SetContent("No logs available")
		return
	}

	// Get all log entries
	entries := m.logBuffer.GetAllEntries()
	if len(entries) == 0 {
		m.logsViewport.SetContent("No logs yet...")
		return
	}

	// Format log entries
	var lines []string
	for _, entry := range entries {
		// Format timestamp
		timestamp := entry.Timestamp.Format("15:04:05.000")

		// Clean up the message (remove trailing newlines)
		message := strings.TrimRight(entry.Message, "\n")

		// Format: [timestamp] message
		line := fmt.Sprintf("[%s] %s", logging.MutedStyle.Render(timestamp), message)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	m.logsViewport.SetContent(content)
}

// renderLogView renders the log view
func (m Model) renderLogView() string {
	if m.logBuffer == nil {
		return "No log buffer available"
	}

	// Build header showing log count
	logCount := m.logBuffer.Count()
	header := logging.HeaderStyle.Width(m.width).Render(
		fmt.Sprintf("LOGS (%d entries)", logCount),
	)

	// Footer
	footer := m.renderFooter()

	// Content was already set by updateLogViewContent() in Update()
	// Combine: header + viewport + footer
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.logsViewport.View(),
		footer,
	)
}

// renderErrorScreen renders an error screen when the fuzzer encounters a fatal error
func (m Model) renderErrorScreen() string {
	header := logging.HeaderStyle.Width(m.width).Render("FUZZER ERROR")

	var lines []string
	lines = append(lines, "")
	lines = append(lines, logging.ErrorStyle.Render("The fuzzer encountered an error and has stopped:"))
	lines = append(lines, "")

	// Display the error message
	if m.fuzzErr != nil {
		errorLines := strings.Split(m.fuzzErr.Error(), "\n")
		for _, line := range errorLines {
			lines = append(lines, "  "+line)
		}
	} else {
		lines = append(lines, "  Unknown error")
	}

	lines = append(lines, "")
	lines = append(lines, logging.MutedStyle.Render("Press 'q' to exit. The error details will be printed to the console."))

	content := strings.Join(lines, "\n")
	footer := logging.FooterStyle.Width(m.width).Render("q: Quit")

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
