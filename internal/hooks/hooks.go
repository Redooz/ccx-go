package hooks

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Event represents when a hook fires.
type Event string

const (
	PreToolUse  Event = "pre_tool_use"
	PostToolUse Event = "post_tool_use"
)

// Hook defines a shell command to run on a specific event.
type Hook struct {
	Event   Event  `json:"event"`
	Pattern string `json:"pattern"` // tool name glob pattern
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"` // milliseconds
}

// HookResult contains the output of a hook execution.
type HookResult struct {
	Hook    Hook
	Output  string
	ExitOK  bool
	Blocked bool // true if hook blocked the tool call
}

// Runner executes hooks.
type Runner struct {
	hooks []Hook
}

// NewRunner creates a hook runner with the given hooks.
func NewRunner(hooks []Hook) *Runner {
	return &Runner{hooks: hooks}
}

// Run executes all hooks matching the event and tool name.
func (r *Runner) Run(ctx context.Context, event Event, toolName string, toolInput string) []HookResult {
	var results []HookResult

	for _, h := range r.hooks {
		if h.Event != event {
			continue
		}

		if h.Pattern != "" && h.Pattern != "*" && h.Pattern != toolName {
			continue
		}

		result := executeHook(ctx, h, toolName, toolInput)
		results = append(results, result)

		// If a pre-hook blocks, stop processing further hooks
		if event == PreToolUse && result.Blocked {
			break
		}
	}

	return results
}

// HasHooks returns true if any hooks are configured.
func (r *Runner) HasHooks() bool {
	return len(r.hooks) > 0
}

func executeHook(ctx context.Context, h Hook, toolName, toolInput string) HookResult {
	timeout := 30 * time.Second
	if h.Timeout > 0 {
		timeout = time.Duration(h.Timeout) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", h.Command)
	cmd.Env = append(cmd.Environ(),
		fmt.Sprintf("TOOL_NAME=%s", toolName),
		fmt.Sprintf("TOOL_INPUT=%s", toolInput),
		fmt.Sprintf("HOOK_EVENT=%s", h.Event),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += strings.TrimSpace(stderr.String())
	}

	result := HookResult{
		Hook:   h,
		Output: output,
		ExitOK: err == nil,
	}

	// Non-zero exit on pre_tool_use blocks the tool call
	if h.Event == PreToolUse && err != nil {
		result.Blocked = true
	}

	return result
}
