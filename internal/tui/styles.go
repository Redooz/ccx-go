package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	accent    = lipgloss.Color("#7C3AED")
	dimColor  = lipgloss.Color("#666666")
	errColor  = lipgloss.Color("#EF4444")
	okColor   = lipgloss.Color("#22C55E")
	warnColor = lipgloss.Color("#F59E0B")

	// Message styles
	userStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E5E5"))

	errorStyle = lipgloss.NewStyle().
			Foreground(errColor).
			Bold(true)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E2E")).
			Foreground(dimColor).
			Padding(0, 1)

	// Tool display
	toolNameStyle = lipgloss.NewStyle().
			Foreground(warnColor).
			Bold(true)

	toolOutputStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			PaddingLeft(2)

	// Input area
	inputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accent).
				Padding(0, 1)

	// Spinner
	spinnerStyle = lipgloss.NewStyle().
			Foreground(accent)

	// Permission prompt
	permissionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warnColor).
			Padding(0, 1).
			MarginTop(1)

	permissionTitleStyle = lipgloss.NewStyle().
				Foreground(warnColor).
				Bold(true)
)
