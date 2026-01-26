package fuzzing

import (
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/crytic/medusa/logging"
)

// FocusedSection represents which section has focus for independent scrolling
type FocusedSection int

const (
	FocusNone FocusedSection = iota
	FocusTestCases
	FocusWorkers
)

// FuzzerTUI is the bubbletea model for the fuzzer dashboard
type FuzzerTUI struct {
	fuzzer            *Fuzzer
	startTime         time.Time
	lastUpdate        time.Time
	width             int
	height            int
	showDebug         bool
	updateCount       int
	viewport          viewport.Model
	ready             bool
	selectedTestIdx   int                      // Index of selected failed test (-1 = none)
	showingTrace      bool                     // Whether we're showing the trace view
	showingLogs       bool                     // Whether we're showing the log view
	mouseEnabled      bool                     // Whether mouse scrolling is enabled
	focusedSection    FocusedSection           // Which section has focus for independent scrolling
	testCasesViewport viewport.Model           // Independent viewport for test cases section
	workersViewport   viewport.Model           // Independent viewport for workers section
	logsViewport      viewport.Model           // Independent viewport for logs section
	errChan           <-chan error             // Channel to receive fuzzer errors
	fuzzErr           error                    // Stores fuzzer error when it occurs
	logBuffer         *logging.LogBufferWriter // Buffer for capturing logs
}

// NewFuzzerTUI creates a new TUI for the fuzzer
func NewFuzzerTUI(fuzzer *Fuzzer) *FuzzerTUI {
	return &FuzzerTUI{
		fuzzer:          fuzzer,
		startTime:       time.Now(),
		lastUpdate:      time.Time{},
		width:           80,
		height:          24,
		showDebug:       false,
		updateCount:     0,
		selectedTestIdx: -1,
		showingTrace:    false,
		showingLogs:     false,
		mouseEnabled:    true,      // Start with mouse enabled
		focusedSection:  FocusNone, // No section focused initially
		errChan:         nil,       // No error channel by default
		fuzzErr:         nil,
		logBuffer:       nil, // No log buffer by default
	}
}

// NewFuzzerTUIWithErrChan creates a new TUI for the fuzzer with an error channel
// The error channel allows the TUI to detect when the fuzzer stops with an error in real-time
func NewFuzzerTUIWithErrChan(fuzzer *Fuzzer, errChan <-chan error) *FuzzerTUI {
	tui := NewFuzzerTUI(fuzzer)
	tui.errChan = errChan
	return tui
}

// SetLogBuffer sets the log buffer for the TUI
func (m *FuzzerTUI) SetLogBuffer(logBuffer *logging.LogBufferWriter) {
	m.logBuffer = logBuffer
}

// FuzzErr returns the fuzzer error if one was received
func (m FuzzerTUI) FuzzErr() error {
	return m.fuzzErr
}

// Messages for the bubbletea update loop
type tickMsg time.Time

// Init initializes the TUI
func (m FuzzerTUI) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
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
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// If fuzzer is already stopped with failures, just quit TUI
			// Otherwise, stop fuzzing first
			if !m.fuzzer.IsStopped() {
				m.fuzzer.Stop()
				// Don't quit yet - let the user see the failure screen first
				// The next render will show it, then they can press 'q' again to quit
				return m, nil
			}
			// Fuzzer already stopped - quit TUI
			return m, tea.Quit
		case "d":
			m.showDebug = !m.showDebug
			return m, nil
		case "m":
			// Toggle mouse mode
			m.mouseEnabled = !m.mouseEnabled
			if m.mouseEnabled {
				return m, tea.EnableMouseCellMotion
			}
			return m, tea.DisableMouse
		case "f", "tab":
			// Cycle through focused sections: None -> TestCases -> Workers -> None
			if m.showingTrace {
				return m, nil // Don't allow focus cycling in trace view
			}
			switch m.focusedSection {
			case FocusNone:
				m.focusedSection = FocusTestCases
			case FocusTestCases:
				m.focusedSection = FocusWorkers
			case FocusWorkers:
				m.focusedSection = FocusNone
			}
			return m, nil
		case "up", "k":
			// If showing trace view, navigate between failed tests
			if m.showingTrace {
				failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
				if len(failedTests) > 0 {
					m.selectedTestIdx--
					if m.selectedTestIdx < 0 {
						m.selectedTestIdx = len(failedTests) - 1
					}
					// Reset scroll position when changing tests
					m.viewport.GotoTop()
				}
				return m, nil
			}
			// Fall through to let viewport handle scrolling
		case "down", "j":
			// If showing trace view, navigate between failed tests
			if m.showingTrace {
				failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
				if len(failedTests) > 0 {
					m.selectedTestIdx++
					if m.selectedTestIdx >= len(failedTests) {
						m.selectedTestIdx = 0
					}
					// Reset scroll position when changing tests
					m.viewport.GotoTop()
				}
				return m, nil
			}
			// Fall through to let viewport handle scrolling
		case "pgup", "pgdown":
			// Let viewport handle page scrolling (falls through to viewport.Update)
		case "enter", "t":
			// Toggle trace view for failed tests
			failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
			if len(failedTests) > 0 {
				m.showingTrace = !m.showingTrace
				if m.showingTrace && m.selectedTestIdx == -1 {
					// Select first test when entering trace view
					m.selectedTestIdx = 0
				}
				m.viewport.GotoTop()
			}
			return m, nil
		case "esc":
			// Exit trace view or log view
			if m.showingTrace {
				m.showingTrace = false
				m.viewport.GotoTop()
			} else if m.showingLogs {
				m.showingLogs = false
				m.viewport.GotoTop()
			}
			return m, nil
		case "l":
			// Toggle log view
			if m.logBuffer != nil {
				m.showingLogs = !m.showingLogs
				if m.showingLogs {
					// Entering log view - update content
					m.updateLogViewContent()
					m.logsViewport.GotoBottom() // Start at bottom (most recent logs)
				} else {
					// Exiting log view
					m.viewport.GotoTop()
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve space for header (2 lines) and footer (2 lines)
		viewportHeight := msg.Height - 4
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.viewport.YPosition = 0

			// Initialize focused section viewports
			// Limit to reasonable height (not more than 1/2 viewport or 20 lines)
			focusedHeight := viewportHeight / 2
			if focusedHeight < 5 {
				focusedHeight = 5
			}
			if focusedHeight > 20 {
				focusedHeight = 20
			}
			m.testCasesViewport = viewport.New(msg.Width-4, focusedHeight)
			m.workersViewport = viewport.New(msg.Width-4, focusedHeight)
			m.logsViewport = viewport.New(msg.Width, viewportHeight)

			m.ready = true

			// Set initial content
			var sections []string
			sections = append(sections, m.renderGlobalStats())
			sections = append(sections, m.renderTestCases())
			sections = append(sections, m.renderWorkerStatus())
			content := lipgloss.JoinVertical(lipgloss.Left, sections...)
			m.viewport.SetContent(content)
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight

			// Update focused viewport dimensions
			// Limit to reasonable height (not more than 1/2 viewport or 20 lines)
			focusedHeight := viewportHeight / 2
			if focusedHeight < 5 {
				focusedHeight = 5
			}
			if focusedHeight > 20 {
				focusedHeight = 20
			}
			m.testCasesViewport.Width = msg.Width - 4
			m.testCasesViewport.Height = focusedHeight
			m.workersViewport.Width = msg.Width - 4
			m.workersViewport.Height = focusedHeight
			m.logsViewport.Width = msg.Width
			m.logsViewport.Height = viewportHeight
		}
		return m, nil

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		m.updateCount++

		// Non-blocking check for fuzzer error
		if m.errChan != nil {
			select {
			case err := <-m.errChan:
				m.fuzzErr = err
				// Fuzzer has stopped - the error will be displayed in View()
			default:
				// No error yet, continue normal operation
			}
		}

		// Update viewport content when we receive a tick
		// This is when content changes (fuzzer stats update)
		if m.showingTrace {
			// Update trace view content
			m.updateTraceViewContent()
		} else if m.showingLogs {
			// Update log view content
			m.updateLogViewContent()
		} else if m.fuzzer.IsStopped() {
			// Update failure screen content
			failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
			if len(failedTests) > 0 {
				content := m.renderFailureScreen(failedTests)
				m.viewport.SetContent(content)
			}
		} else {
			// Update main dashboard content
			if m.focusedSection == FocusNone {
				// Normal mode: all sections in main viewport
				var sections []string
				sections = append(sections, m.renderGlobalStats())
				sections = append(sections, m.renderTestCases())
				sections = append(sections, m.renderWorkerStatus())
				content := lipgloss.JoinVertical(lipgloss.Left, sections...)
				m.viewport.SetContent(content)
			} else {
				// Focused mode: update independent viewports
				m.testCasesViewport.SetContent(m.renderTestCasesContent())
				m.workersViewport.SetContent(m.renderWorkerStatusContent())
			}
		}

		return m, tickCmd()
	}

	// Update viewport (handles scrolling)
	if m.showingLogs {
		// Log view mode: update logs viewport
		m.logsViewport, cmd = m.logsViewport.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.focusedSection == FocusNone {
		// Normal mode: update main viewport
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		// Focused mode: only update the focused viewport
		switch m.focusedSection {
		case FocusTestCases:
			m.testCasesViewport, cmd = m.testCasesViewport.Update(msg)
			cmds = append(cmds, cmd)
		case FocusWorkers:
			m.workersViewport, cmd = m.workersViewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m FuzzerTUI) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Check if fuzzer encountered an error
	if m.fuzzErr != nil {
		return m.renderErrorScreen()
	}

	// If showing trace view, render that instead
	if m.showingTrace {
		return m.renderTraceView()
	}

	// If showing log view, render that instead
	if m.showingLogs {
		return m.renderLogView()
	}

	// Check if fuzzing is done
	if m.fuzzer.IsStopped() {
		// Check if we stopped due to a test failure
		failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
		if len(failedTests) > 0 {
			// Content was already set by tickMsg in Update()
			return lipgloss.JoinVertical(lipgloss.Left,
				logging.HeaderStyle.Width(m.width).Render("FUZZING STOPPED - TEST FAILURE"),
				m.viewport.View(),
				logging.FooterStyle.Width(m.width).Render("↑/↓: Scroll | q: Quit (logs will print)"),
			)
		}
		return m.renderExitScreen()
	}

	// Header (not in viewport, stays at top)
	header := m.renderHeader()

	// Footer (not in viewport, stays at bottom)
	footer := m.renderFooter()

	// Render based on focus mode
	if m.focusedSection == FocusNone {
		// Normal mode: all content in main viewport
		// Content was already set by tickMsg in Update()
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			m.viewport.View(),
			footer,
		)
	}

	// Focused mode: render sections with focused one in viewport
	var content string
	switch m.focusedSection {
	case FocusTestCases:
		// Render: stats (fixed) + focused test cases (scrollable) + workers (fixed)
		focusedTestCases := logging.FocusedBoxStyle.Width(m.width - 2).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				logging.TitleStyle.Render("Test Cases [FOCUSED - Press f/tab to unfocus]"),
				m.testCasesViewport.View(),
			),
		)
		content = lipgloss.JoinVertical(lipgloss.Left,
			m.renderGlobalStats(),
			focusedTestCases,
			m.renderWorkerStatus(),
		)
	case FocusWorkers:
		// Render: stats (fixed) + test cases (fixed) + focused workers (scrollable)
		focusedWorkers := logging.FocusedBoxStyle.Width(m.width - 2).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				logging.TitleStyle.Render("Worker Status [FOCUSED - Press f/tab to unfocus]"),
				m.workersViewport.View(),
			),
		)
		content = lipgloss.JoinVertical(lipgloss.Left,
			m.renderGlobalStats(),
			m.renderTestCases(),
			focusedWorkers,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderHeader renders the dashboard header
func (m FuzzerTUI) renderHeader() string {
	header := "MEDUSA FUZZING DASHBOARD"
	return logging.HeaderStyle.Width(m.width).Render(header)
}

// renderGlobalStats renders global fuzzing statistics
func (m FuzzerTUI) renderGlobalStats() string {
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

	if metrics := m.fuzzer.Metrics(); metrics != nil {
		callsTested = metrics.CallsTested()
		sequencesTested = metrics.SequencesTested()
		failedSequences = metrics.FailedSequences()
		gasUsed = metrics.GasUsed()
		shrinkingWorkers = metrics.WorkersShrinkingCount()
	}

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
	activeWorkers := 0
	for _, worker := range m.fuzzer.Workers() {
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
func (m FuzzerTUI) renderTestCases() string {
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
	failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
	runningTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusRunning)
	passedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusPassed)
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
func (m FuzzerTUI) renderWorkerStatus() string {
	// Use full width minus small margin
	boxWidth := m.width - 2
	if boxWidth < 40 {
		boxWidth = 40
	}

	workers := m.fuzzer.Workers()

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
func (m FuzzerTUI) renderTestCasesContent() string {
	var lines []string

	// Show failed tests (no limit - scrolling allows viewing all)
	failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
	if len(failedTests) > 0 {
		lines = append(lines, logging.ErrorStyle.Render(fmt.Sprintf("Failed Tests (%d):", len(failedTests))))
		for _, tc := range failedTests {
			lines = append(lines, fmt.Sprintf("  %s %s", logging.ErrorStyle.Render("✗"), tc.Name()))
		}
		lines = append(lines, "")
	}

	// Show running tests (no limit - scrolling allows viewing all)
	runningTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusRunning)
	if len(runningTests) > 0 {
		lines = append(lines, logging.ValueStyle.Render(fmt.Sprintf("Running Tests (%d):", len(runningTests))))
		for _, tc := range runningTests {
			lines = append(lines, fmt.Sprintf("  %s %s", logging.ValueStyle.Render("⚡"), tc.Name()))
		}
		lines = append(lines, "")
	}

	// Show passed tests count
	passedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusPassed)
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
func (m FuzzerTUI) renderWorkerStatusContent() string {
	workers := m.fuzzer.Workers()

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
func (m FuzzerTUI) renderWorkerBox(index int, worker *FuzzerWorker, width int) string {
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
func (m FuzzerTUI) renderFooter() string {
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
		failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
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

// renderFailureScreen renders the failure summary when tests fail
func (m FuzzerTUI) renderFailureScreen(failedTests []TestCase) string {
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
	if metrics := m.fuzzer.Metrics(); metrics != nil {
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
func (m *FuzzerTUI) updateTraceViewContent() {
	failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
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

func (m FuzzerTUI) renderTraceView() string {
	failedTests := m.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
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
func (m FuzzerTUI) renderExitScreen() string {
	var lines []string

	lines = append(lines, logging.HeaderStyle.Width(m.width).Render("FUZZING STOPPED"))
	lines = append(lines, "")

	// Final statistics
	elapsed := time.Since(m.startTime)
	callsTested := big.NewInt(0)
	sequencesTested := big.NewInt(0)
	failedSequences := big.NewInt(0)
	if metrics := m.fuzzer.Metrics(); metrics != nil {
		callsTested = metrics.CallsTested()
		sequencesTested = metrics.SequencesTested()
		failedSequences = metrics.FailedSequences()
	}

	// Get coverage with nil checks
	branches := uint64(0)
	if corpus := m.fuzzer.Corpus(); corpus != nil {
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
func (m *FuzzerTUI) updateLogViewContent() {
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
func (m FuzzerTUI) renderLogView() string {
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
func (m FuzzerTUI) renderErrorScreen() string {
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

// getFuzzerStatus returns the current fuzzer status string
func (m FuzzerTUI) getFuzzerStatus() string {
	if corpus := m.fuzzer.Corpus(); corpus != nil && corpus.InitializingCorpus() {
		return logging.WarningStyle.Render("INITIALIZING")
	}
	return logging.ValueStyle.Render("FUZZING")
}
