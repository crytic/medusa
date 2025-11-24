package tui

import (
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/crytic/medusa/fuzzing"
)

// FuzzerTUI is the bubbletea model for the fuzzer dashboard
type FuzzerTUI struct {
	fuzzer      *fuzzing.Fuzzer
	startTime   time.Time
	lastUpdate  time.Time
	width       int
	height      int
	showDebug   bool
	paused      bool
	updateCount int
	viewport    viewport.Model
	ready       bool
}

// NewFuzzerTUI creates a new TUI for the fuzzer
func NewFuzzerTUI(fuzzer *fuzzing.Fuzzer) *FuzzerTUI {
	return &FuzzerTUI{
		fuzzer:      fuzzer,
		startTime:   time.Now(),
		lastUpdate:  time.Time{},
		width:       80,
		height:      24,
		showDebug:   false,
		paused:      false,
		updateCount: 0,
	}
}

// Messages for the bubbletea update loop
type tickMsg time.Time

// Init initializes the TUI
func (m FuzzerTUI) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

// tickCmd returns a command that ticks every 500ms
func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages and updates the model
func (m FuzzerTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Stop fuzzing gracefully
			m.fuzzer.Stop()
			return m, tea.Quit
		case "p":
			m.paused = !m.paused
			return m, nil
		case "d":
			m.showDebug = !m.showDebug
			return m, nil
		case "up", "k":
			if m.ready {
				m.viewport.LineUp(1)
			}
			return m, nil
		case "down", "j":
			if m.ready {
				m.viewport.LineDown(1)
			}
			return m, nil
		case "pgup":
			if m.ready {
				m.viewport.ViewUp()
			}
			return m, nil
		case "pgdown":
			if m.ready {
				m.viewport.ViewDown()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			// Reserve space for footer
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 2
		}
		return m, nil

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		m.updateCount++
		return m, tickCmd()
	}

	// Update viewport
	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	return m, cmd
}

// View renders the TUI
func (m FuzzerTUI) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Check if fuzzing is done
	if m.fuzzer.IsStopped() {
		// Check if we stopped due to a test failure
		failedTests := m.fuzzer.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
		if len(failedTests) > 0 {
			return m.renderFailureScreen(failedTests)
		}
		return m.renderExitScreen()
	}

	// Build content for viewport
	var sections []string

	// Header (not in viewport, stays at top)
	header := m.renderHeader()

	// Content sections (will be scrollable)
	sections = append(sections, m.renderGlobalStats())
	sections = append(sections, m.renderTestCases())
	sections = append(sections, m.renderWorkerStatus())

	// Set viewport content
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	m.viewport.SetContent(content)

	// Footer (not in viewport, stays at bottom)
	footer := m.renderFooter()

	// Combine: header + viewport + footer
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.viewport.View(),
		footer,
	)
}

// renderHeader renders the dashboard header
func (m FuzzerTUI) renderHeader() string {
	header := "MEDUSA FUZZING DASHBOARD"
	if m.paused {
		header += " [PAUSED]"
	}
	return headerStyle.Width(m.width).Render(header)
}

// renderGlobalStats renders global fuzzing statistics
func (m FuzzerTUI) renderGlobalStats() string {
	// Dynamically set box width based on terminal width
	boxWidth := m.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	// Get metrics
	elapsed := time.Since(m.startTime)
	callsTested := m.fuzzer.Metrics().CallsTested()
	sequencesTested := m.fuzzer.Metrics().SequencesTested()
	failedSequences := m.fuzzer.Metrics().FailedSequences()

	// Get coverage metrics (with nil checks)
	branches := uint64(0)
	if corpus := m.fuzzer.Corpus(); corpus != nil {
		if coverageMaps := corpus.CoverageMaps(); coverageMaps != nil {
			branches = coverageMaps.BranchesHit()
		}
	}

	// Get corpus size (with nil check)
	corpusSize := uint64(0)
	if corpus := m.fuzzer.Corpus(); corpus != nil {
		corpusSize = uint64(corpus.ActiveMutableSequenceCount())
	}

	gasUsed := m.fuzzer.Metrics().GasUsed()

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
	totalWorkers := len(m.fuzzer.Workers())
	shrinkingWorkers := m.fuzzer.Metrics().WorkersShrinkingCount()
	activeWorkers := 0
	for _, worker := range m.fuzzer.Workers() {
		if worker.Activity().Snapshot().IsActive() {
			activeWorkers++
		}
	}

	// Build stats
	var lines []string
	lines = append(lines, titleStyle.Render("Global Statistics"))
	lines = append(lines, "")

	// Line 1: Elapsed and Status
	line1 := fmt.Sprintf("%s %s                    %s %s",
		labelStyle.Render("Campaign Elapsed:"),
		valueStyle.Render(formatDuration(elapsed)),
		labelStyle.Render("Status:"),
		valueStyle.Render(m.getFuzzerStatus()),
	)
	lines = append(lines, line1)
	lines = append(lines, "")

	// Line 2: Calls and Coverage
	line2 := fmt.Sprintf("%s %s (%s)              %s %s",
		labelStyle.Render("Total Calls:"),
		valueStyle.Render(formatNumber(callsTested)),
		mutedStyle.Render(formatRate(callsPerSec)),
		labelStyle.Render("Coverage:"),
		valueStyle.Render(fmt.Sprintf("%d branches", branches)),
	)
	lines = append(lines, line2)

	// Line 3: Sequences and Corpus
	line3 := fmt.Sprintf("%s %s (%s)              %s %s",
		labelStyle.Render("Sequences:"),
		valueStyle.Render(formatNumber(sequencesTested)),
		mutedStyle.Render(formatRate(seqPerSec)),
		labelStyle.Render("Corpus Size:"),
		valueStyle.Render(fmt.Sprintf("%d sequences", corpusSize)),
	)
	lines = append(lines, line3)

	// Line 4: Failures and Gas
	failurePercent := formatPercentage(failedSequences, sequencesTested)
	line4 := fmt.Sprintf("%s %s/%s (%s)          %s %s",
		labelStyle.Render("Test Failures:"),
		errorStyle.Render(formatNumber(failedSequences)),
		mutedStyle.Render(formatNumber(sequencesTested)),
		mutedStyle.Render(failurePercent),
		labelStyle.Render("Gas Used:"),
		valueStyle.Render(formatRate(gasPerSec)),
	)
	lines = append(lines, line4)

	// Line 5: Workers
	line5 := fmt.Sprintf("%s %s",
		labelStyle.Render("Workers:"),
		valueStyle.Render(fmt.Sprintf("%d/%d active, %d shrinking", activeWorkers, totalWorkers, shrinkingWorkers)),
	)
	lines = append(lines, line5)

	// Debug info if enabled
	if m.showDebug {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("Memory: %s / %s",
			formatBytes(memStats.Alloc),
			formatBytes(memStats.Sys))))
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("Updates: %d", m.updateCount)))
	}

	content := strings.Join(lines, "\n")
	return boxStyle.Width(boxWidth).Render(content)
}

// renderTestCases renders test case status
func (m FuzzerTUI) renderTestCases() string {
	// Dynamically set box width based on terminal width
	boxWidth := m.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Test Cases"))
	lines = append(lines, "")

	// Show failed tests
	failedTests := m.fuzzer.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
	if len(failedTests) > 0 {
		lines = append(lines, errorStyle.Render(fmt.Sprintf("Failed Tests (%d):", len(failedTests))))
		for i, tc := range failedTests {
			if i >= 5 { // Limit to 5 most recent
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("  ... and %d more", len(failedTests)-5)))
				break
			}
			lines = append(lines, fmt.Sprintf("  %s %s", errorStyle.Render("✗"), tc.Name()))
		}
		lines = append(lines, "")
	}

	// Show running tests
	runningTests := m.fuzzer.TestCasesWithStatus(fuzzing.TestCaseStatusRunning)
	if len(runningTests) > 0 {
		lines = append(lines, valueStyle.Render(fmt.Sprintf("Running Tests (%d):", len(runningTests))))
		for i, tc := range runningTests {
			if i >= 3 {
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("  ... and %d more", len(runningTests)-3)))
				break
			}
			lines = append(lines, fmt.Sprintf("  %s %s", valueStyle.Render("⚡"), tc.Name()))
		}
		lines = append(lines, "")
	}

	// Show passed tests count
	passedTests := m.fuzzer.TestCasesWithStatus(fuzzing.TestCaseStatusPassed)
	if len(passedTests) > 0 {
		lines = append(lines, fmt.Sprintf("%s %d tests passed", valueStyle.Render("✓"), len(passedTests)))
	}

	// If no test info to show, show a message
	if len(failedTests) == 0 && len(runningTests) == 0 && len(passedTests) == 0 {
		lines = append(lines, mutedStyle.Render("No test results yet..."))
	}

	content := strings.Join(lines, "\n")
	return boxStyle.Width(boxWidth).Render(content)
}

// renderWorkerStatus renders individual worker status
func (m FuzzerTUI) renderWorkerStatus() string {
	// Dynamically set box width based on terminal width
	boxWidth := m.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	workers := m.fuzzer.Workers()

	var lines []string
	lines = append(lines, titleStyle.Render("Worker Status"))
	lines = append(lines, "")

	// Render workers in rows of 3
	workersPerRow := 3
	for i := 0; i < len(workers); i += workersPerRow {
		var rowBoxes []string
		for j := 0; j < workersPerRow && i+j < len(workers); j++ {
			workerIndex := i + j
			rowBoxes = append(rowBoxes, m.renderWorkerBox(workerIndex, workers[workerIndex]))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowBoxes...)
		lines = append(lines, row)
	}

	content := strings.Join(lines, "\n")
	return boxStyle.Width(boxWidth).Render(content)
}

// renderWorkerBox renders a single worker's status box
func (m FuzzerTUI) renderWorkerBox(index int, worker *fuzzing.FuzzerWorker) string {
	// Get worker activity snapshot
	activity := worker.Activity().Snapshot()

	// Get state style and icon
	stateStr := activity.State.String()
	icon := getStateIcon(stateStr)
	stateStyle := getStateStyle(stateStr)

	// Get worker metrics
	calls := worker.WorkerMetrics().CallsTested().Uint64()
	sequences := worker.WorkerMetrics().SequencesTested().Uint64()

	// Format content
	line1 := fmt.Sprintf("%s %s", icon, stateStyle.Render(stateStr))

	var line2 string
	if activity.State.String() == "Shrinking" && activity.ShrinkLimit > 0 {
		// Show progress bar for shrinking
		progress := activity.ShrinkProgress()
		line2 = renderProgressBar(progress, 18)
	} else if activity.Strategy != "" {
		line2 = truncateString(activity.Strategy, 20)
	} else {
		line2 = ""
	}

	line3 := fmt.Sprintf("Calls: %s", formatNumber(big.NewInt(int64(calls))))
	line4 := fmt.Sprintf("Seqs: %s", formatNumber(big.NewInt(int64(sequences))))

	// Join lines
	var contentLines []string
	contentLines = append(contentLines, line1)
	if line2 != "" {
		contentLines = append(contentLines, mutedStyle.Render(line2))
	}
	contentLines = append(contentLines, mutedStyle.Render(line3))
	contentLines = append(contentLines, mutedStyle.Render(line4))

	content := strings.Join(contentLines, "\n")
	return workerBoxStyle.Render(content)
}

// renderFooter renders the footer with help text
func (m FuzzerTUI) renderFooter() string {
	// Show scroll percentage if content is larger than viewport
	scrollInfo := ""
	if m.ready {
		scrollPercent := int(m.viewport.ScrollPercent() * 100)
		if m.viewport.TotalLineCount() > m.viewport.Height {
			scrollInfo = fmt.Sprintf(" | Scroll: %d%%", scrollPercent)
		}
	}

	helpText := fmt.Sprintf("↑/↓: Scroll | q: Quit | p: Pause | d: Debug%s", scrollInfo)
	return footerStyle.Width(m.width).Render(helpText)
}

// renderFailureScreen renders the failure summary when tests fail
func (m FuzzerTUI) renderFailureScreen(failedTests []fuzzing.TestCase) string {
	var lines []string

	lines = append(lines, headerStyle.Width(m.width).Render("FUZZING STOPPED - TEST FAILURE"))
	lines = append(lines, "")

	// Show failed test information
	lines = append(lines, errorStyle.Render(fmt.Sprintf("Failed Tests (%d):", len(failedTests))))
	lines = append(lines, "")

	for i, tc := range failedTests {
		if i >= 10 { // Limit to first 10 failures
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("... and %d more failures", len(failedTests)-10)))
			break
		}

		lines = append(lines, errorStyle.Render(fmt.Sprintf("  ✗ %s", tc.Name())))

		// Get a short summary of the failure (first line only)
		message := tc.Message()
		messageLines := strings.Split(message, "\n")
		if len(messageLines) > 0 && messageLines[0] != "" {
			// Truncate long messages
			summary := messageLines[0]
			if len(summary) > 80 {
				summary = summary[:77] + "..."
			}
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("    %s", summary)))
		}
		lines = append(lines, "")
	}

	// Final statistics
	elapsed := time.Since(m.startTime)
	callsTested := m.fuzzer.Metrics().CallsTested()
	sequencesTested := m.fuzzer.Metrics().SequencesTested()

	lines = append(lines, titleStyle.Render("Campaign Statistics:"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Total Time: %s", formatDuration(elapsed)))
	lines = append(lines, fmt.Sprintf("  Calls Tested: %s", formatNumber(callsTested)))
	lines = append(lines, fmt.Sprintf("  Sequences Tested: %s", formatNumber(sequencesTested)))
	lines = append(lines, "")
	lines = append(lines, warningStyle.Render("Press 'q' to quit and see detailed logs"))

	return strings.Join(lines, "\n")
}

// renderExitScreen renders the exit summary
func (m FuzzerTUI) renderExitScreen() string {
	var lines []string

	lines = append(lines, headerStyle.Width(m.width).Render("FUZZING STOPPED"))
	lines = append(lines, "")

	// Final statistics
	elapsed := time.Since(m.startTime)
	callsTested := m.fuzzer.Metrics().CallsTested()
	sequencesTested := m.fuzzer.Metrics().SequencesTested()
	failedSequences := m.fuzzer.Metrics().FailedSequences()

	// Get coverage with nil checks
	branches := uint64(0)
	if corpus := m.fuzzer.Corpus(); corpus != nil {
		if coverageMaps := corpus.CoverageMaps(); coverageMaps != nil {
			branches = coverageMaps.BranchesHit()
		}
	}

	lines = append(lines, titleStyle.Render("Final Statistics:"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Total Time: %s", formatDuration(elapsed)))
	lines = append(lines, fmt.Sprintf("  Calls Tested: %s", formatNumber(callsTested)))
	lines = append(lines, fmt.Sprintf("  Sequences Tested: %s", formatNumber(sequencesTested)))
	lines = append(lines, fmt.Sprintf("  Branches Hit: %d", branches))
	lines = append(lines, fmt.Sprintf("  Test Failures: %s", formatNumber(failedSequences)))
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Check the logs for detailed test results."))

	return strings.Join(lines, "\n")
}

// getFuzzerStatus returns the current fuzzer status string
func (m FuzzerTUI) getFuzzerStatus() string {
	if m.paused {
		return warningStyle.Render("PAUSED")
	}
	if corpus := m.fuzzer.Corpus(); corpus != nil && corpus.InitializingCorpus() {
		return warningStyle.Render("INITIALIZING")
	}
	return valueStyle.Render("FUZZING")
}
