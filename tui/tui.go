package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/logging"
)

// FocusedSection represents which section has focus for independent scrolling
type FocusedSection int

const (
	FocusNone FocusedSection = iota
	FocusTestCases
	FocusWorkers
)

// Model is the bubbletea model for the fuzzer dashboard
type Model struct {
	provider          FuzzerDataProvider
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

// New creates a new TUI for the fuzzer
func New(provider FuzzerDataProvider) *Model {
	return &Model{
		provider:        provider,
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

// NewWithErrChan creates a new TUI for the fuzzer with an error channel
// The error channel allows the TUI to detect when the fuzzer stops with an error in real-time
func NewWithErrChan(provider FuzzerDataProvider, errChan <-chan error) *Model {
	tui := New(provider)
	tui.errChan = errChan
	return tui
}

// SetLogBuffer sets the log buffer for the TUI
func (m *Model) SetLogBuffer(logBuffer *logging.LogBufferWriter) {
	m.logBuffer = logBuffer
}

// FuzzErr returns the fuzzer error if one was received
func (m Model) FuzzErr() error {
	return m.fuzzErr
}

// Messages for the bubbletea update loop
type tickMsg time.Time

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
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
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// If fuzzer is already stopped with failures, just quit TUI
			// Otherwise, stop fuzzing first
			if !m.provider.IsStopped() {
				m.provider.Stop()
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
				failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
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
				failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
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
			failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
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
		} else if m.provider.IsStopped() {
			// Update failure screen content
			failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
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
func (m Model) View() string {
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
	if m.provider.IsStopped() {
		// Check if we stopped due to a test failure
		failedTests := m.provider.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)
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

// Run starts the TUI program and blocks until it exits
func Run(provider FuzzerDataProvider, logBuffer *logging.LogBufferWriter, errChan <-chan error) error {
	// Create TUI model
	model := NewWithErrChan(provider, errChan)
	if logBuffer != nil {
		model.SetLogBuffer(logBuffer)
	}

	// Run TUI in foreground (blocking)
	tuiProgram := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := tuiProgram.Run()

	// Return TUI error if any, or fuzzer error from model
	if err != nil {
		return err
	}
	return model.FuzzErr()
}
