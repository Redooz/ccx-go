package tui

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const asciiPet = `    ╱▔▔▔▔▔╲
   ╱ ●   ● ╲
  ╱    ▽     ╲
 ╱_____________╲`

func renderWelcome(width, height int, modelName, cwd string) string {
	if width < 40 || height < 10 {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, "ccx-go")
	}

	name := getUserFirstName()
	shortCwd := shortenPath(cwd)
	modelShort := shortModelName(modelName)

	// Content area inside border (border takes 2 chars width, 2 lines height)
	contentH := height - 2
	if contentH < 8 {
		contentH = 8
	}
	innerW := width - 2

	// Panel widths: left 55%, divider 1, right rest
	leftW := innerW * 55 / 100
	rightW := innerW - leftW - 1

	// Left panel: centered greeting + pet + model info
	leftContent := strings.Join([]string{
		fmt.Sprintf("Welcome back, %s!", name),
		"",
		asciiPet,
		"",
		welcomeModelStyle.Render(modelShort),
		welcomeDimStyle.Render(shortCwd),
	}, "\n")
	leftPanel := lipgloss.Place(leftW, contentH, lipgloss.Center, lipgloss.Center, leftContent)

	// Right panel: tips + recent activity
	rightContent := lipgloss.NewStyle().PaddingLeft(1).PaddingTop(1).Render(
		strings.Join([]string{
			welcomeTitleStyle.Render("Tips for getting started"),
			"",
			welcomeDimStyle.Render("Ask me to help you understand,"),
			welcomeDimStyle.Render("fix, or improve your code."),
			"",
			separatorStyle.Render(strings.Repeat("─", rightW-2)),
			"",
			welcomeTitleStyle.Render("Recent activity"),
			welcomeDimStyle.Render("No recent activity"),
		}, "\n"),
	)
	rightPanel := lipgloss.Place(rightW, contentH, lipgloss.Left, lipgloss.Top, rightContent)

	// Vertical divider column
	divLines := make([]string, contentH)
	for i := range divLines {
		divLines[i] = separatorStyle.Render("│")
	}
	divider := strings.Join(divLines, "\n")

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, divider, rightPanel)

	return welcomeBorderStyle.Width(innerW).Render(panels)
}

func getUserFirstName() string {
	if u, err := user.Current(); err == nil {
		if u.Name != "" {
			parts := strings.Fields(u.Name)
			if len(parts) > 0 {
				return parts[0]
			}
		}
		if u.Username != "" {
			return u.Username
		}
	}
	return "there"
}

func shortenPath(path string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if strings.HasPrefix(path, home) {
			return "~" + path[len(home):]
		}
	}
	return path
}

func shortModelName(model string) string {
	switch {
	case strings.Contains(model, "opus"):
		return "Opus 4"
	case strings.Contains(model, "sonnet"):
		return "Sonnet 4"
	case strings.Contains(model, "haiku"):
		return "Haiku 4"
	default:
		return model
	}
}
