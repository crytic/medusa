package logging

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
var HeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorWhite).
	Background(colorPrimary).
	Padding(0, 2).
	MarginBottom(1)

// Box styles
var BoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorSecondary).
	Padding(1, 2).
	MarginBottom(1)

// Worker box style
var WorkerBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(colorBorder).
	Padding(0, 1).
	Width(24).
	Height(4).
	MarginRight(1)

// Focused box style (highlighted border)
var FocusedBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.ThickBorder()).
	BorderForeground(colorPrimary).
	Padding(1, 2).
	MarginBottom(1)

// Text styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	LabelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	ValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorError)

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWarning)

	MutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)

// State-specific styles
var (
	StateGeneratingStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	StateReplayingStyle = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true)

	StateShrinkingStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true)

	StateIdleStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true)
)

// Footer style
var FooterStyle = lipgloss.NewStyle().
	Foreground(colorMuted).
	Padding(0, 2).
	MarginTop(1)

// getStateStyle returns the appropriate style for a worker state
func GetStateStyle(state string) lipgloss.Style {
	switch state {
	case "Generating":
		return StateGeneratingStyle
	case "Replaying Corpus":
		return StateReplayingStyle
	case "Shrinking":
		return StateShrinkingStyle
	default:
		return StateIdleStyle
	}
}

// getStateIcon returns an icon for a worker state
func GetStateIcon(state string) string {
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
