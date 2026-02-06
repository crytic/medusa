package tui

import (
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/logging"
)

// renderHeader renders the dashboard header
func (m Model) renderHeader() string {
	header := "MEDUSA FUZZING DASHBOARD"
	return logging.HeaderStyle.Width(m.width).Render(header)
}

// renderGlobalStats renders global fuzzing statistics
func (m Model) renderGlobalStats() string {
	// Use full width minus small margin
	boxWidth := m.width - 2
	if boxWidth < 40 {
		boxWidth = 40
	}

	// Get metrics (with nil checks)
	elapsed := time.Since(m.startTime)
	callsTested := big.NewInt(0)
	sequencesTested := big.NewInt(0)
	failedSequences := big.NewInt(0)
	gasUsed := big.NewInt(0)
	shrinkingWorkers := uint64(0)

	if metrics := m.provider.Metrics(); metrics != nil {
		callsTested = metrics.CallsTested()
		sequencesTested = metrics.SequencesTested()
		failedSequences = metrics.FailedSequences()
		gasUsed = metrics.GasUsed()
		shrinkingWorkers = metrics.WorkersShrinkingCount()
	}

	// Get coverage metrics (with nil checks)
	branches := uint64(0)
	if corpus := m.provider.Corpus(); corpus != nil {
		if coverageMaps := corpus.CoverageMaps(); coverageMaps != nil {
			branches = coverageMaps.BranchesHit()
		}
	}

	// Get corpus size (with nil check)
	corpusSize := uint64(0)
	if corpus := m.provider.Corpus(); corpus != nil {
		corpusSize = uint64(corpus.ActiveMutableSequenceCount())
	}

	// Calculate rates
	seconds := elapsed.Seconds()
	callsPerSec := uint64(0)
	seqPerSec := uint64(0)
	gasPerSec := uint64(0)
	if seconds > 0 {
		callsPerSec = uint64(float64(callsTested.Uint64()) / seconds)
		seqPerSec = uint64(float64(sequencesTested.Uint64()) / seconds)
		gasPerSec = uint64(float64(gasUsed.Uint64()) / seconds)
	}

	// Get worker counts
	totalWorkers := len(m.provider.Workers())
	activeWorkers := 0
	for _, worker := range m.provider.Workers() {
		if worker.Activity().Snapshot().IsActive() {
			activeWorkers++
		}
	}

	// Build stats
	var lines []string
	lines = append(lines, logging.TitleStyle.Render("Global Statistics"))
	lines = append(lines, "")

	// Line 1: Elapsed and Status
	line1 := fmt.Sprintf("%s %s                    %s %s",
		logging.LabelStyle.Render("Campaign Elapsed:"),
		logging.ValueStyle.Render(logging.FormatDuration(elapsed)),
		logging.LabelStyle.Render("Status:"),
		logging.ValueStyle.Render(m.getFuzzerStatus()),
	)
	lines = append(lines, line1)
	lines = append(lines, "")

	// Line 2: Calls and Coverage
	line2 := fmt.Sprintf("%s %s (%s)              %s %s",
		logging.LabelStyle.Render("Total Calls:"),
		logging.ValueStyle.Render(logging.FormatNumber(callsTested)),
		logging.MutedStyle.Render(logging.FormatRate(callsPerSec)),
		logging.LabelStyle.Render("Coverage:"),
		logging.ValueStyle.Render(fmt.Sprintf("%d branches", branches)),
	)
	lines = append(lines, line2)

	// Line 3: Sequences and Corpus
	line3 := fmt.Sprintf("%s %s (%s)              %s %s",
		logging.LabelStyle.Render("Sequences:"),
		logging.ValueStyle.Render(logging.FormatNumber(sequencesTested)),
		logging.MutedStyle.Render(logging.FormatRate(seqPerSec)),
		logging.LabelStyle.Render("Corpus Size:"),
		logging.ValueStyle.Render(fmt.Sprintf("%d sequences", corpusSize)),
	)
	lines = append(lines, line3)

	// Line 4: Failures and Gas
	failurePercent := logging.FormatPercentage(failedSequences, sequencesTested)
	line4 := fmt.Sprintf("%s %s/%s (%s)          %s %s",
		logging.LabelStyle.Render("Test Failures:"),
		logging.ErrorStyle.Render(logging.FormatNumber(failedSequences)),
		logging.MutedStyle.Render(logging.FormatNumber(sequencesTested)),
		logging.MutedStyle.Render(failurePercent),
		logging.LabelStyle.Render("Gas Used:"),
		logging.ValueStyle.Render(logging.FormatRate(gasPerSec)),
	)
	lines = append(lines, line4)

	// Line 5: Workers
	line5 := fmt.Sprintf("%s %s",
		logging.LabelStyle.Render("Workers:"),
		logging.ValueStyle.Render(fmt.Sprintf("%d/%d active, %d shrinking", activeWorkers, totalWorkers, shrinkingWorkers)),
	)
	lines = append(lines, line5)

	// Debug info if enabled
	if m.showDebug {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		lines = append(lines, "")
		lines = append(lines, logging.MutedStyle.Render(fmt.Sprintf("Memory: %s / %s",
			logging.FormatBytes(memStats.Alloc),
			logging.FormatBytes(memStats.Sys))))
		lines = append(lines, logging.MutedStyle.Render(fmt.Sprintf("Updates: %d", m.updateCount)))
	}

	content := strings.Join(lines, "\n")
	return logging.BoxStyle.Width(boxWidth).Render(content)
}

// renderTestCases renders test case status
func (m Model) renderTestCases() string {
	// Use full width minus small margin
	boxWidth := m.width - 2
	if boxWidth < 40 {
		boxWidth = 40
	}

	var lines []string
	lines = append(lines, logging.TitleStyle.Render("Test Cases"))
	lines = append(lines, "")

	// Calculate max lines based on terminal height
	// Reserve space for: header (3), global stats (~8), workers (~8), footer (2), margins (~4)
	// Remaining space goes to test cases
	reservedLines := 25
	availableLines := m.height - reservedLines
	if availableLines < 5 {
		availableLines = 5 // Minimum 5 lines
	}
	if availableLines > 30 {
		availableLines = 30 // Maximum 30 lines to prevent it from dominating
	}

	maxTestLines := availableLines
	currentLines := 0

	// Show failed tests (limited)
	failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
	runningTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusRunning)
	passedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusPassed)
	totalTests := len(failedTests) + len(runningTests) + len(passedTests)

	if len(failedTests) > 0 && currentLines < maxTestLines {
		lines = append(lines, logging.ErrorStyle.Render(fmt.Sprintf("Failed Tests (%d):", len(failedTests))))
		currentLines++
		for _, tc := range failedTests {
			if currentLines >= maxTestLines {
				break
			}
			lines = append(lines, fmt.Sprintf("  %s %s", logging.ErrorStyle.Render("✗"), tc.Name()))
			currentLines++
			totalTests--
		}
		lines = append(lines, "")
		currentLines++
	}

	// Show running tests (limited)
	if len(runningTests) > 0 && currentLines < maxTestLines {
		lines = append(lines, logging.ValueStyle.Render(fmt.Sprintf("Running Tests (%d):", len(runningTests))))
		currentLines++
		for _, tc := range runningTests {
			if currentLines >= maxTestLines {
				break
			}
			lines = append(lines, fmt.Sprintf("  %s %s", logging.ValueStyle.Render("⚡"), tc.Name()))
			currentLines++
			totalTests--
		}
		lines = append(lines, "")
		currentLines++
	}

	// Show passed tests count
	if len(passedTests) > 0 && currentLines < maxTestLines {
		lines = append(lines, fmt.Sprintf("%s %d tests passed", logging.ValueStyle.Render("✓"), len(passedTests)))
	}

	// If no test info to show, show a message
	if len(failedTests) == 0 && len(runningTests) == 0 && len(passedTests) == 0 {
		lines = append(lines, logging.MutedStyle.Render("No test results yet..."))
	} else if currentLines >= maxTestLines {
		lines = append(lines, logging.MutedStyle.Render(fmt.Sprintf("  ... and %d more (press f to focus)", totalTests)))
	}

	content := strings.Join(lines, "\n")
	return logging.BoxStyle.Width(boxWidth).Render(content)
}

// renderWorkerStatus renders individual worker status
func (m Model) renderWorkerStatus() string {
	// Use full width minus small margin
	boxWidth := m.width - 2
	if boxWidth < 40 {
		boxWidth = 40
	}

	workers := m.provider.Workers()

	var lines []string
	lines = append(lines, logging.TitleStyle.Render("Worker Status"))
	lines = append(lines, "")

	// Determine workers per row based on terminal width
	// Start with a reasonable fixed width that looks good
	baseWorkerWidth := 30 // includes content + borders + margins

	workersPerRow := m.width / baseWorkerWidth
	if workersPerRow < 1 {
		workersPerRow = 1
	}
	if workersPerRow > 5 {
		workersPerRow = 5 // Cap at 5 for readability
	}

	// Calculate max rows based on terminal height
	// Reserve space for: header (3), global stats (~8), test cases (~variable), footer (2), margins (~4)
	// Each worker row takes ~6 lines (including borders/spacing)
	reservedLines := 25
	availableLines := m.height - reservedLines
	if availableLines < 6 {
		availableLines = 6 // Minimum 1 row of workers
	}
	maxRows := availableLines / 6
	if maxRows < 1 {
		maxRows = 1 // Always show at least 1 row
	}
	if maxRows > 6 {
		maxRows = 6 // Maximum 6 rows
	}

	// Use fixed width - it's simpler and looks better
	workerBoxWidth := 24

	totalRows := (len(workers) + workersPerRow - 1) / workersPerRow
	displayRows := totalRows
	if displayRows > maxRows {
		displayRows = maxRows
	}

	for row := 0; row < displayRows; row++ {
		i := row * workersPerRow
		var rowBoxes []string
		for j := 0; j < workersPerRow && i+j < len(workers); j++ {
			workerIndex := i + j
			rowBoxes = append(rowBoxes, m.renderWorkerBox(workerIndex, workers[workerIndex], workerBoxWidth))
		}
		rowStr := lipgloss.JoinHorizontal(lipgloss.Top, rowBoxes...)
		lines = append(lines, rowStr)
	}

	// Show "more workers" message if truncated
	if totalRows > maxRows {
		remainingWorkers := len(workers) - (displayRows * workersPerRow)
		lines = append(lines, "")
		lines = append(lines, logging.MutedStyle.Render(fmt.Sprintf("... and %d more workers (press f to focus)", remainingWorkers)))
	}

	content := strings.Join(lines, "\n")
	return logging.BoxStyle.Width(boxWidth).Render(content)
}

// renderTestCasesContent returns just the content for test cases (no box wrapper)
// Used when test cases section is focused
func (m Model) renderTestCasesContent() string {
	var lines []string

	// Show failed tests (no limit - scrolling allows viewing all)
	failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
	if len(failedTests) > 0 {
		lines = append(lines, logging.ErrorStyle.Render(fmt.Sprintf("Failed Tests (%d):", len(failedTests))))
		for _, tc := range failedTests {
			lines = append(lines, fmt.Sprintf("  %s %s", logging.ErrorStyle.Render("✗"), tc.Name()))
		}
		lines = append(lines, "")
	}

	// Show running tests (no limit - scrolling allows viewing all)
	runningTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusRunning)
	if len(runningTests) > 0 {
		lines = append(lines, logging.ValueStyle.Render(fmt.Sprintf("Running Tests (%d):", len(runningTests))))
		for _, tc := range runningTests {
			lines = append(lines, fmt.Sprintf("  %s %s", logging.ValueStyle.Render("⚡"), tc.Name()))
		}
		lines = append(lines, "")
	}

	// Show passed tests count
	passedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusPassed)
	if len(passedTests) > 0 {
		lines = append(lines, fmt.Sprintf("%s %d tests passed", logging.ValueStyle.Render("✓"), len(passedTests)))
	}

	// If no test info to show, show a message
	if len(failedTests) == 0 && len(runningTests) == 0 && len(passedTests) == 0 {
		lines = append(lines, logging.MutedStyle.Render("No test results yet..."))
	}

	return strings.Join(lines, "\n")
}

// renderWorkerStatusContent returns just the content for worker status (no box wrapper)
// Used when workers section is focused
func (m Model) renderWorkerStatusContent() string {
	workers := m.provider.Workers()

	var lines []string

	// Determine workers per row based on terminal width
	baseWorkerWidth := 30
	workersPerRow := m.width / baseWorkerWidth
	if workersPerRow < 1 {
		workersPerRow = 1
	}
	if workersPerRow > 5 {
		workersPerRow = 5
	}

	// Use fixed width
	workerBoxWidth := 24

	for i := 0; i < len(workers); i += workersPerRow {
		var rowBoxes []string
		for j := 0; j < workersPerRow && i+j < len(workers); j++ {
			workerIndex := i + j
			rowBoxes = append(rowBoxes, m.renderWorkerBox(workerIndex, workers[workerIndex], workerBoxWidth))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowBoxes...)
		lines = append(lines, row)
	}

	return strings.Join(lines, "\n")
}

// renderWorkerBox renders a single worker's status box
func (m Model) renderWorkerBox(index int, worker *fuzzing.FuzzerWorker, width int) string {
	// Get worker activity snapshot
	activity := worker.Activity().Snapshot()

	// Get state style and icon
	stateStr := activity.State.String()
	icon := logging.GetStateIcon(stateStr)
	stateStyle := logging.GetStateStyle(stateStr)

	// Get worker metrics
	calls := worker.WorkerMetrics().CallsTested().Uint64()
	sequences := worker.WorkerMetrics().SequencesTested().Uint64()

	// Format content
	line1 := fmt.Sprintf("%s %s", icon, stateStyle.Render(stateStr))

	var line2 string
	progressBarWidth := width - 4 // Leave room for padding
	if progressBarWidth < 10 {
		progressBarWidth = 10
	}
	if activity.State.String() == "Shrinking" && activity.ShrinkLimit > 0 {
		// Show progress bar for shrinking
		progress := activity.ShrinkProgress()
		line2 = logging.RenderProgressBar(progress, progressBarWidth)
	} else if activity.Strategy != "" {
		line2 = logging.TruncateString(activity.Strategy, width-4)
	} else {
		line2 = ""
	}

	line3 := fmt.Sprintf("Calls: %s", logging.FormatNumber(big.NewInt(int64(calls))))
	line4 := fmt.Sprintf("Seqs: %s", logging.FormatNumber(big.NewInt(int64(sequences))))

	// Join lines
	var contentLines []string
	contentLines = append(contentLines, line1)
	if line2 != "" {
		contentLines = append(contentLines, logging.MutedStyle.Render(line2))
	}
	contentLines = append(contentLines, logging.MutedStyle.Render(line3))
	contentLines = append(contentLines, logging.MutedStyle.Render(line4))

	content := strings.Join(contentLines, "\n")

	// Create responsive worker box style
	responsiveWorkerBoxStyle := logging.WorkerBoxStyle.Width(width)
	return responsiveWorkerBoxStyle.Render(content)
}

// renderFooter renders the footer with help text
func (m Model) renderFooter() string {
	// Show scroll percentage if content is larger than viewport
	scrollInfo := ""
	if m.ready {
		scrollPercent := int(m.viewport.ScrollPercent() * 100)
		if m.viewport.TotalLineCount() > m.viewport.Height {
			scrollInfo = fmt.Sprintf(" | Scroll: %d%%", scrollPercent)
		}
	}

	// Show mouse mode status
	mouseInfo := ""
	if !m.mouseEnabled {
		mouseInfo = " | Mouse: OFF"
	}

	// Show focus mode status
	focusInfo := ""
	switch m.focusedSection {
	case FocusTestCases:
		focusInfo = " | Focused: Test Cases"
	case FocusWorkers:
		focusInfo = " | Focused: Workers"
	}

	var helpText string
	if m.showingTrace {
		// Trace view controls
		helpText = fmt.Sprintf("↑/↓: Next/Prev Test | PgUp/PgDn: Scroll Trace | Esc/t: Exit Trace | m: Mouse | q: Quit%s%s", scrollInfo, mouseInfo)
	} else if m.showingLogs {
		// Log view controls
		helpText = fmt.Sprintf("↑/↓: Scroll Logs | Esc/l: Exit Logs | m: Mouse | q: Quit%s%s", scrollInfo, mouseInfo)
	} else {
		// Normal controls
		failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
		logControls := ""
		if m.logBuffer != nil {
			logControls = " | l: Logs"
		}
		if len(failedTests) > 0 {
			helpText = fmt.Sprintf("↑/↓: Scroll | f/Tab: Focus Section | t/Enter: View Traces%s | m: Mouse | q: Quit%s%s%s", logControls, scrollInfo, mouseInfo, focusInfo)
		} else {
			helpText = fmt.Sprintf("↑/↓: Scroll | f/Tab: Focus Section%s | m: Mouse | q: Quit | d: Debug%s%s%s", logControls, scrollInfo, mouseInfo, focusInfo)
		}
	}

	return logging.FooterStyle.Width(m.width).Render(helpText)
}

// getFuzzerStatus returns the current fuzzer status string
func (m Model) getFuzzerStatus() string {
	if corpus := m.provider.Corpus(); corpus != nil && corpus.InitializingCorpus() {
		return logging.WarningStyle.Render("INITIALIZING")
	}
	return logging.ValueStyle.Render("FUZZING")
}
