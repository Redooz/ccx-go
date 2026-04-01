package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/anton-abyzov/ccx-go/internal/api"
	"github.com/anton-abyzov/ccx-go/internal/config"
	"github.com/anton-abyzov/ccx-go/internal/cost"
	"github.com/anton-abyzov/ccx-go/internal/prompt"
	"github.com/anton-abyzov/ccx-go/internal/query"
	"github.com/anton-abyzov/ccx-go/internal/tui"
	agentTool "github.com/anton-abyzov/ccx-go/internal/tools/agent"
	"github.com/anton-abyzov/ccx-go/internal/tools/bash"
	"github.com/anton-abyzov/ccx-go/internal/tools/fileedit"
	"github.com/anton-abyzov/ccx-go/internal/tools/fileread"
	"github.com/anton-abyzov/ccx-go/internal/tools/filewrite"
	"github.com/anton-abyzov/ccx-go/internal/tools/glob"
	"github.com/anton-abyzov/ccx-go/internal/tools/grep"
	"github.com/anton-abyzov/ccx-go/internal/tools/notebookedit"
	"github.com/anton-abyzov/ccx-go/internal/tools/todowrite"
	"github.com/anton-abyzov/ccx-go/internal/tools/webfetch"
	"github.com/anton-abyzov/ccx-go/internal/tools/websearch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/anton-abyzov/ccx-go/internal/tool"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "claude [prompt]",
		Short:   "AI coding assistant CLI",
		Long:    "ccx-go — a Go implementation of an AI coding assistant powered by Claude.",
		Version: version,
		RunE:    runQuery,
		Args:    cobra.ArbitraryArgs,
	}

	rootCmd.Flags().StringP("model", "m", "claude-sonnet-4-20250514", "model to use")
	rootCmd.Flags().Int("max-turns", 0, "maximum conversation turns (0 = unlimited)")
	rootCmd.Flags().Int("max-tokens", 16384, "maximum tokens per response")
	rootCmd.Flags().Bool("no-stream", false, "disable streaming output")
	rootCmd.Flags().StringP("system", "s", "", "additional system prompt")
	rootCmd.Flags().Bool("tui", false, "use full-screen Bubbletea TUI (default: inline)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runQuery(cmd *cobra.Command, args []string) error {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required\n\nSet it with:\n  export ANTHROPIC_API_KEY=sk-ant-...")
	}

	model, _ := cmd.Flags().GetString("model")
	maxTurns, _ := cmd.Flags().GetInt("max-turns")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")
	extraSystem, _ := cmd.Flags().GetString("system")
	useTUI, _ := cmd.Flags().GetBool("tui")

	// Load configuration
	settings, _ := config.LoadSettings(config.DefaultSettingsPath())
	if settings == nil {
		settings = &config.Settings{}
	}
	if settings.Model != "" && !cmd.Flags().Changed("model") {
		model = settings.Model
	}
	if settings.MaxTokens > 0 && !cmd.Flags().Changed("max-tokens") {
		maxTokens = settings.MaxTokens
	}

	// Discover CLAUDE.md files
	cwd, _ := os.Getwd()
	claudeMD := config.DiscoverClaudeMD(cwd)

	// Initialize cost tracker
	tracker := cost.NewTracker()

	// Set up client and tools
	client := api.NewClient(apiKey)
	registry := registerTools(cwd, client, model)

	// Build CLAUDE.md content for system prompt
	var claudeMDContent string
	if claudeMD != nil && len(claudeMD.Entries) > 0 {
		var parts []string
		for _, entry := range claudeMD.Entries {
			parts = append(parts, fmt.Sprintf("## %s\n%s", entry.Path, entry.Content))
		}
		claudeMDContent = strings.Join(parts, "\n\n")
	}

	// Add settings system prompt and memory
	var extraParts []string
	if settings.SystemPrompt != "" {
		extraParts = append(extraParts, settings.SystemPrompt)
	}
	if extraSystem != "" {
		extraParts = append(extraParts, extraSystem)
	}
	memDir := config.MemoryDir()
	if memIndex, err := config.LoadMemoryIndex(memDir); err == nil && memIndex != "" {
		extraParts = append(extraParts, "# Memory\n"+memIndex)
	}

	// Build system prompt using the prompt package
	systemPrompt := prompt.BuildSystemPrompt(
		registry.All(),
		claudeMDContent,
		cwd,
		strings.Join(extraParts, "\n\n"),
	)

	// Enable agent log output
	agentTool.SetLogWriter(os.Stderr)

	// Detect interactive TTY vs piped input
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))

	if isTTY {
		if useTUI {
			return runFullscreenTUI(cmd.Context(), client, registry, cwd, model, maxTokens, maxTurns, systemPrompt, tracker, args)
		}
		return runInline(cmd.Context(), client, registry, cwd, model, maxTokens, maxTurns, systemPrompt, tracker, args)
	}

	return runPipe(cmd.Context(), client, registry, model, maxTokens, maxTurns, systemPrompt, tracker, args)
}

// runInline runs the Claude Code-style inline chat (default for TTY).
func runInline(parentCtx context.Context, client *api.Client, registry *tool.Registry, cwd, model string, maxTokens, maxTurns int, systemPrompt string, tracker *cost.Tracker, args []string) error {
	ctx, cancel := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Print welcome panel
	tui.RenderWelcomeInline(version, model, cwd, registry.Count())

	streaming := false

	loop := query.NewLoop(client, registry, query.LoopConfig{
		Model:          model,
		MaxTokens:      maxTokens,
		MaxTurns:       maxTurns,
		System:         systemPrompt,
		SuppressOutput: true,
		OnText: func(text string) {
			if !streaming {
				streaming = true
			}
			tui.RenderStreamChunk(text)
		},
		OnToolStart: func(name string) {
			if streaming {
				tui.RenderStreamEnd()
				streaming = false
			}
			tui.RenderToolStartInline(name, "")
		},
		OnToolDone: func(name, result string, isErr bool) {
			tui.RenderToolOutputInline(result, isErr)
		},
		OnTurnComplete: func(usage api.Usage) {
			tracker.Record(model, usage.InputTokens, usage.OutputTokens)
		},
	})

	// Handle initial prompt from args
	if len(args) > 0 {
		input := strings.Join(args, " ")
		tui.RenderUserMessageInline(input)
		streaming = false
		if err := loop.SendMessage(ctx, input); err != nil {
			tui.RenderErrorInline(err)
		}
		if streaming {
			tui.RenderStreamEnd()
			streaming = false
		}
		tui.RenderSeparator()
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		tui.RenderPromptInline()
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "/exit" || input == "/quit" {
			break
		}

		tui.RenderUserMessageInline(input)
		streaming = false

		if err := loop.SendMessage(ctx, input); err != nil {
			tui.RenderErrorInline(err)
		}
		if streaming {
			tui.RenderStreamEnd()
			streaming = false
		}
		tui.RenderSeparator()
	}

	fmt.Fprintf(os.Stderr, "\n--- session: %s ---\n", tracker.Summary())
	return nil
}

// runFullscreenTUI runs the original Bubbletea full-screen mode (--tui flag).
func runFullscreenTUI(parentCtx context.Context, client *api.Client, registry *tool.Registry, cwd, model string, maxTokens, maxTurns int, systemPrompt string, tracker *cost.Tracker, args []string) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	msgCh := make(chan tea.Msg, 100)

	loop := query.NewLoop(client, registry, query.LoopConfig{
		Model:          model,
		MaxTokens:      maxTokens,
		MaxTurns:       maxTurns,
		System:         systemPrompt,
		SuppressOutput: true,
		OnText: func(text string) {
			msgCh <- tui.StreamTextMsg{Text: text}
		},
		OnToolStart: func(name string) {
			msgCh <- tui.ToolExecMsg{ToolName: name}
		},
		OnToolDone: func(name, result string, isErr bool) {
			msgCh <- tui.ToolDoneMsg{ToolName: name, Content: result, IsError: isErr}
		},
		OnTurnComplete: func(usage api.Usage) {
			tracker.Record(model, usage.InputTokens, usage.OutputTokens)
		},
	})

	onSubmit := func(text string) {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					msgCh <- tui.ErrorMsg{Err: fmt.Errorf("panic: %v", r)}
					return
				}
			}()

			err := loop.SendMessage(ctx, text)
			if err != nil {
				msgCh <- tui.ErrorMsg{Err: err}
				return
			}
			msgCh <- tui.DoneMsg{}
		}()
	}

	// Build initial prompt from args
	initialPrompt := ""
	if len(args) > 0 {
		initialPrompt = strings.Join(args, " ")
	}

	m := tui.New(tui.AppConfig{
		ModelName:     model,
		Version:       version,
		CWD:           cwd,
		InitialPrompt: initialPrompt,
	}, msgCh, onSubmit)

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()

	// Print session summary after TUI exits
	fmt.Fprintf(os.Stderr, "\n--- session: %s ---\n", tracker.Summary())

	return err
}

func runPipe(parentCtx context.Context, client *api.Client, registry *tool.Registry, model string, maxTokens, maxTurns int, systemPrompt string, tracker *cost.Tracker, args []string) error {
	loop := query.NewLoop(client, registry, query.LoopConfig{
		Model:     model,
		MaxTokens: maxTokens,
		MaxTurns:  maxTurns,
		System:    systemPrompt,
		OnTurnComplete: func(usage api.Usage) {
			tracker.Record(model, usage.InputTokens, usage.OutputTokens)
		},
	})

	prompt := ""
	if len(args) > 0 {
		prompt = strings.Join(args, " ")
	}

	ctx, cancel := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := loop.Run(ctx, prompt)

	fmt.Fprintf(os.Stderr, "\n--- session: %s ---\n", tracker.Summary())

	return err
}

func registerTools(cwd string, client *api.Client, model string) *tool.Registry {
	registry := tool.NewRegistry()

	agentSystem := "You are a sub-agent. Complete the given task using available tools, then return a concise summary."

	tools := []tool.Tool{
		bash.New(cwd),
		fileread.New(),
		filewrite.New(),
		fileedit.New(),
		glob.New(),
		grep.New(),
		webfetch.New(),
		websearch.New(),
		todowrite.New(),
		notebookedit.New(),
		agentTool.New(client, registry, model, agentSystem),
	}

	for _, t := range tools {
		if err := registry.Register(t); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to register tool %s: %v\n", t.Name(), err)
		}
	}

	return registry
}
