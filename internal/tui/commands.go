package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/config"
	"github.com/anton-abyzov/ccx-go/internal/cost"
	"github.com/anton-abyzov/ccx-go/internal/tool"
)

// CommandContext provides data needed by slash commands.
type CommandContext struct {
	Version  string
	Model    string
	CWD      string
	Tracker  *cost.Tracker
	Registry *tool.Registry
}

// CommandResult holds the outcome of a slash command.
type CommandResult struct {
	Output  string
	Quit    bool
	Clear   bool
	Compact bool
}

// HandleSlashCommand processes /commands. Returns nil if input is not a command.
func HandleSlashCommand(input string, ctx CommandContext) *CommandResult {
	parts := strings.Fields(input)
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "/") {
		return nil
	}

	switch parts[0] {
	case "/":
		return &CommandResult{Output: commandList()}
	case "/help":
		return &CommandResult{Output: helpText()}
	case "/exit", "/quit":
		return &CommandResult{Quit: true}
	case "/clear":
		return &CommandResult{Clear: true, Output: "\033[2J\033[H"}
	case "/cost":
		return &CommandResult{Output: costOutput(ctx.Tracker)}
	case "/model":
		return &CommandResult{Output: fmt.Sprintf("Model: %s (%s)", shortModelName(ctx.Model), ctx.Model)}
	case "/compact":
		return &CommandResult{Compact: true, Output: "Context compacted."}
	case "/version":
		return &CommandResult{Output: fmt.Sprintf("ccx-go v%s", ctx.Version)}
	case "/tools":
		return &CommandResult{Output: toolsOutput(ctx.Registry)}
	case "/login":
		if err := config.RunOAuthLogin(); err != nil {
			return &CommandResult{Output: fmt.Sprintf("Login failed: %v", err)}
		}
		return &CommandResult{Output: ""}
	case "/init":
		return &CommandResult{Output: initProject(ctx.CWD)}
	case "/config":
		return &CommandResult{Output: configOutput()}
	case "/status":
		return &CommandResult{Output: statusOutput(ctx)}
	case "/doctor":
		return &CommandResult{Output: doctorOutput(ctx)}
	default:
		return nil // not a built-in command; may be a skill
	}
}

func commandList() string {
	return `Available commands:
  /help     Show available commands and shortcuts
  /login    Log in to Claude via OAuth
  /exit     Quit the session
  /clear    Clear the screen
  /cost     Show token usage and cost for this session
  /model    Show current model
  /compact  Compress conversation context
  /version  Show version info
  /tools    List all available tools with descriptions
  /init     Initialize a new project (.claude/ directory)
  /config   Show current configuration
  /status   Show session status
  /doctor   Run diagnostics`
}

func helpText() string {
	return `ccx-go — AI coding assistant

Commands:
  /help     Show this help
  /login    Log in to Claude via OAuth
  /exit     Quit the session
  /clear    Clear the screen
  /cost     Show token usage and cost
  /model    Show current model
  /compact  Compress conversation context
  /version  Show version info
  /tools    List available tools
  /init     Initialize a new project
  /config   Show current configuration
  /status   Show session status
  /doctor   Run diagnostics

Shortcuts:
  Ctrl+C    Quit
  Enter     Send message`
}

func costOutput(t *cost.Tracker) string {
	if t == nil {
		return "No cost data available."
	}
	input, output := t.TotalTokens()
	totalCost := t.TotalCost()
	calls := t.CallCount()
	return fmt.Sprintf("Session cost:\n  API calls:     %d\n  Input tokens:  %d\n  Output tokens: %d\n  Total cost:    $%.4f", calls, input, output, totalCost)
}

func toolsOutput(r *tool.Registry) string {
	if r == nil {
		return "No tools registered."
	}
	tools := r.All()
	if len(tools) == 0 {
		return "No tools registered."
	}

	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name() < tools[j].Name()
	})

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Available tools (%d):\n", len(tools)))
	for _, t := range tools {
		desc := t.Description()
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		b.WriteString(fmt.Sprintf("  %-16s %s\n", t.Name(), desc))
	}
	return strings.TrimRight(b.String(), "\n")
}

func initProject(cwd string) string {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Sprintf("Error getting working directory: %v", err)
		}
	}
	claudeDir := filepath.Join(cwd, ".claude")
	if info, err := os.Stat(claudeDir); err == nil && info.IsDir() {
		return fmt.Sprintf("Project already initialized: %s", claudeDir)
	}
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Sprintf("Error creating .claude directory: %v", err)
	}
	// Create a default settings.json
	settingsPath := filepath.Join(claudeDir, "settings.json")
	defaultSettings := &config.Settings{}
	if err := config.SaveSettings(settingsPath, defaultSettings); err != nil {
		return fmt.Sprintf("Created %s but failed to write settings: %v", claudeDir, err)
	}
	return fmt.Sprintf("Initialized project at %s", claudeDir)
}

func configOutput() string {
	settings, err := config.LoadSettings(config.DefaultSettingsPath())
	if err != nil {
		return fmt.Sprintf("Error loading settings: %v", err)
	}
	if settings == nil {
		settings = &config.Settings{}
	}

	var b strings.Builder
	b.WriteString("Configuration:\n")
	b.WriteString(fmt.Sprintf("  Settings file: %s\n", config.DefaultSettingsPath()))

	if settings.Model != "" {
		b.WriteString(fmt.Sprintf("  Model:         %s\n", settings.Model))
	} else {
		b.WriteString("  Model:         (default)\n")
	}

	if settings.MaxTokens > 0 {
		b.WriteString(fmt.Sprintf("  Max tokens:    %d\n", settings.MaxTokens))
	} else {
		b.WriteString("  Max tokens:    (default)\n")
	}

	if settings.SystemPrompt != "" {
		prompt := settings.SystemPrompt
		if len(prompt) > 50 {
			prompt = prompt[:47] + "..."
		}
		b.WriteString(fmt.Sprintf("  System prompt: %s\n", prompt))
	}

	if len(settings.Permissions) > 0 {
		b.WriteString(fmt.Sprintf("  Permissions:   %d rules\n", len(settings.Permissions)))
	}

	return strings.TrimRight(b.String(), "\n")
}

func statusOutput(ctx CommandContext) string {
	var b strings.Builder
	b.WriteString("Session status:\n")
	b.WriteString(fmt.Sprintf("  Model:    %s (%s)\n", shortModelName(ctx.Model), ctx.Model))
	b.WriteString(fmt.Sprintf("  Version:  %s\n", ctx.Version))

	if ctx.CWD != "" {
		b.WriteString(fmt.Sprintf("  CWD:      %s\n", ctx.CWD))
	}

	if ctx.Tracker != nil {
		input, output := ctx.Tracker.TotalTokens()
		calls := ctx.Tracker.CallCount()
		totalCost := ctx.Tracker.TotalCost()
		b.WriteString(fmt.Sprintf("  API calls:     %d\n", calls))
		b.WriteString(fmt.Sprintf("  Input tokens:  %d\n", input))
		b.WriteString(fmt.Sprintf("  Output tokens: %d\n", output))
		b.WriteString(fmt.Sprintf("  Total cost:    $%.4f\n", totalCost))
	}

	if ctx.Registry != nil {
		b.WriteString(fmt.Sprintf("  Tools:    %d loaded\n", ctx.Registry.Count()))
	}

	return strings.TrimRight(b.String(), "\n")
}

func doctorOutput(ctx CommandContext) string {
	var b strings.Builder
	b.WriteString("Diagnostics:\n")

	// Check API key
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		b.WriteString("  [ok] ANTHROPIC_API_KEY is set\n")
	} else {
		// Check OAuth credentials
		auth, err := config.ResolveAuth()
		if err != nil {
			b.WriteString("  [!!] No authentication found (set ANTHROPIC_API_KEY or run /login)\n")
		} else {
			b.WriteString(fmt.Sprintf("  [ok] Authenticated via %s\n", auth.Display))
		}
	}

	// Check tools
	if ctx.Registry != nil {
		count := ctx.Registry.Count()
		if count > 0 {
			b.WriteString(fmt.Sprintf("  [ok] %d tools loaded\n", count))
		} else {
			b.WriteString("  [!!] No tools loaded\n")
		}
	} else {
		b.WriteString("  [!!] Tool registry not available\n")
	}

	// Check settings file
	settingsPath := config.DefaultSettingsPath()
	if _, err := os.Stat(settingsPath); err == nil {
		b.WriteString(fmt.Sprintf("  [ok] Settings file exists: %s\n", settingsPath))
	} else {
		b.WriteString(fmt.Sprintf("  [--] No settings file at %s\n", settingsPath))
	}

	// Check .claude directory in CWD
	if ctx.CWD != "" {
		claudeDir := filepath.Join(ctx.CWD, ".claude")
		if _, err := os.Stat(claudeDir); err == nil {
			b.WriteString(fmt.Sprintf("  [ok] Project .claude/ exists: %s\n", claudeDir))
		} else {
			b.WriteString("  [--] No .claude/ in working directory (run /init to create)\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
