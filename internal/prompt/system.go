package prompt

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

// BuildSystemPrompt constructs the system prompt sent to Claude.
// It includes the role description, available tools, working directory,
// CLAUDE.md content, and any extra instructions.
func BuildSystemPrompt(tools []tool.Tool, claudeMd string, cwd string, extra string) string {
	var parts []string

	// Role description
	parts = append(parts, `You are an AI coding assistant CLI (ccx-go). You help users with software engineering tasks including writing code, debugging, explaining code, running commands, and managing files.

# Key Behaviors
- Read files before modifying them
- Use the most specific tool available (e.g., Read instead of Bash cat)
- Execute commands when asked, show results
- Be concise and direct`)

	// Environment
	parts = append(parts, fmt.Sprintf(`
# Environment
- Working directory: %s
- Platform: %s/%s
- Runtime: %s`, cwd, runtime.GOOS, runtime.GOARCH, runtime.Version()))

	// Available tools
	if len(tools) > 0 {
		var toolList strings.Builder
		toolList.WriteString("\n# Available Tools\n")
		for _, t := range tools {
			toolList.WriteString(fmt.Sprintf("- **%s**: %s\n", t.Name(), t.Description()))
		}
		parts = append(parts, toolList.String())
	}

	// CLAUDE.md content
	if claudeMd != "" {
		parts = append(parts, "\n# Project Instructions (CLAUDE.md)\n"+claudeMd)
	}

	// Extra system prompt
	if extra != "" {
		parts = append(parts, "\n# Additional Instructions\n"+extra)
	}

	return strings.Join(parts, "\n")
}
