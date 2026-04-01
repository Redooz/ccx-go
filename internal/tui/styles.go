package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Claude Code color palette
	orange  = lipgloss.Color("204")
	cyan    = lipgloss.Color("86")
	dimGray = lipgloss.Color("242")
	green   = lipgloss.Color("78")
	yellow  = lipgloss.Color("220")
	red     = lipgloss.Color("196")

	// Prompt ❯
	promptStyle = lipgloss.NewStyle().
			Foreground(orange).
			Bold(true)

	// User messages
	userStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	// Error
	errorStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	// Tool display
	toolNameStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true)

	toolOutputStyle = lipgloss.NewStyle().
			Foreground(dimGray).
			PaddingLeft(2)

	// Spinner
	spinnerStyle = lipgloss.NewStyle().
			Foreground(orange)

	// Status bar (footer)
	statusLeftStyle = lipgloss.NewStyle().
			Foreground(dimGray)

	statusRightStyle = lipgloss.NewStyle().
				Foreground(dimGray)

	// Header line
	headerStyle = lipgloss.NewStyle().
			Foreground(orange)

	// Separator
	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	// Welcome panel
	welcomeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(orange)

	welcomeTitleStyle = lipgloss.NewStyle().
				Foreground(orange).
				Bold(true)

	welcomeDimStyle = lipgloss.NewStyle().
			Foreground(dimGray)

	welcomeModelStyle = lipgloss.NewStyle().
				Foreground(green)

	// Permission prompt (kept for future use)
	permissionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(yellow).
			Padding(0, 1).
			MarginTop(1)

	permissionTitleStyle = lipgloss.NewStyle().
				Foreground(yellow).
				Bold(true)
)
