package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/anton-abyzov/ccx-go/internal/api"
	"github.com/anton-abyzov/ccx-go/internal/config"
	"github.com/anton-abyzov/ccx-go/internal/cost"
	"github.com/anton-abyzov/ccx-go/internal/query"
	"github.com/anton-abyzov/ccx-go/internal/tool"
	"github.com/anton-abyzov/ccx-go/internal/tools/bash"
	"github.com/anton-abyzov/ccx-go/internal/tools/fileedit"
	"github.com/anton-abyzov/ccx-go/internal/tools/fileread"
	"github.com/anton-abyzov/ccx-go/internal/tools/filewrite"
	"github.com/anton-abyzov/ccx-go/internal/tools/glob"
	"github.com/anton-abyzov/ccx-go/internal/tools/grep"
	"github.com/anton-abyzov/ccx-go/internal/tools/webfetch"
	"github.com/spf13/cobra"
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

	// Build system prompt
	systemPrompt := buildSystemPrompt(cwd, claudeMD, settings, extraSystem)

	// Initialize cost tracker
	tracker := cost.NewTracker()

	// Set up client and tools
	client := api.NewClient(apiKey)
	registry := registerTools(cwd)

	// Print welcome
	fmt.Fprintf(os.Stderr, "ccx-go v%s | model: %s | cwd: %s\n", version, model, cwd)
	if len(claudeMD.Entries) > 0 {
		fmt.Fprintf(os.Stderr, "Loaded %d CLAUDE.md file(s)\n", len(claudeMD.Entries))
	}
	fmt.Fprintf(os.Stderr, "Type /exit to quit, Ctrl+C to interrupt\n\n")

	loop := query.NewLoop(client, registry, query.LoopConfig{
		Model:     model,
		MaxTokens: maxTokens,
		MaxTurns:  maxTurns,
		System:    systemPrompt,
		OnTurnComplete: func(usage api.Usage) {
			tracker.Record(model, usage.InputTokens, usage.OutputTokens)
		},
	})

	// Build prompt from args
	prompt := ""
	if len(args) > 0 {
		prompt = strings.Join(args, " ")
	}

	// Handle Ctrl+C gracefully
	ctx := cmd.Context()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := loop.Run(ctx, prompt)

	// Print session summary
	fmt.Fprintf(os.Stderr, "\n--- session: %s ---\n", tracker.Summary())

	return err
}

func registerTools(cwd string) *tool.Registry {
	registry := tool.NewRegistry()

	tools := []tool.Tool{
		bash.New(cwd),
		fileread.New(),
		filewrite.New(),
		fileedit.New(),
		glob.New(),
		grep.New(),
		webfetch.New(),
	}

	for _, t := range tools {
		if err := registry.Register(t); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to register tool %s: %v\n", t.Name(), err)
		}
	}

	return registry
}

func buildSystemPrompt(cwd string, claudeMD *config.ClaudeMD, settings *config.Settings, extra string) string {
	var parts []string

	parts = append(parts, fmt.Sprintf(`You are an AI coding assistant. You help users with software engineering tasks.

# Environment
- Working directory: %s
- Platform: %s/%s
- Go version: %s`, cwd, runtime.GOOS, runtime.GOARCH, runtime.Version()))

	// Add CLAUDE.md content
	if claudeMD != nil && len(claudeMD.Entries) > 0 {
		parts = append(parts, "\n# Project Instructions (from CLAUDE.md)")
		for _, entry := range claudeMD.Entries {
			parts = append(parts, fmt.Sprintf("\n## %s\n%s", entry.Path, entry.Content))
		}
	}

	// Add settings system prompt
	if settings != nil && settings.SystemPrompt != "" {
		parts = append(parts, "\n# Custom Instructions\n"+settings.SystemPrompt)
	}

	// Add extra system prompt from CLI flag
	if extra != "" {
		parts = append(parts, "\n# Additional Instructions\n"+extra)
	}

	// Load memory index if available
	memDir := config.MemoryDir()
	if memIndex, err := config.LoadMemoryIndex(memDir); err == nil && memIndex != "" {
		parts = append(parts, "\n# Memory\n"+memIndex)
	}

	return strings.Join(parts, "\n")
}
