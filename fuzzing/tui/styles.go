package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorPrimary   = lipgloss.Color("#7D56F4")
	colorSecondary = lipgloss.Color("#874BFD")
	colorSuccess   = lipgloss.Color("#04B575")
	colorWarning   = lipgloss.Color("#FFD700")
	colorError     = lipgloss.Color("#FF6B6B")
	colorMuted     = lipgloss.Color("#7D7D7D")
	colorWhite     = lipgloss.Color("#FAFAFA")
	colorBorder    = lipgloss.Color("#444444")
)

// Header styles
var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorWhite).
	Background(colorPrimary).
	Padding(0, 2).
	MarginBottom(1)

// Box styles
var boxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorSecondary).
	Padding(1, 2).
	MarginBottom(1)

// Worker box style
var workerBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(colorBorder).
	Padding(0, 1).
	Width(24).
	Height(4).
	MarginRight(1)

// Text styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorError)

	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWarning)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)

// State-specific styles
var (
	stateGeneratingStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	stateReplayingStyle = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true)

	stateShrinkingStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true)

	stateIdleStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true)
)

// Footer style
var footerStyle = lipgloss.NewStyle().
	Foreground(colorMuted).
	Padding(0, 2).
	MarginTop(1)

// getStateStyle returns the appropriate style for a worker state
func getStateStyle(state string) lipgloss.Style {
	switch state {
	case "Generating":
		return stateGeneratingStyle
	case "Replaying Corpus":
		return stateReplayingStyle
	case "Shrinking":
		return stateShrinkingStyle
	default:
		return stateIdleStyle
	}
}

// getStateIcon returns an icon for a worker state
func getStateIcon(state string) string {
	switch state {
	case "Generating":
		return "⚡"
	case "Replaying Corpus":
		return "📂"
	case "Shrinking":
		return "🔧"
	default:
		return "💤"
	}
}
