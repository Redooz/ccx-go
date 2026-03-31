package bash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

// Maximum output size before truncation (512KB).
const maxOutputSize = 512 * 1024

var dangerousPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	":(){ :|:& };:",
	"fork bomb",
	"mkfs.",
	"dd if=/dev/zero of=/dev/sd",
	"> /dev/sda",
	"chmod -R 777 /",
}

type input struct {
	Command     string            `json:"command"`
	Timeout     int               `json:"timeout,omitempty"`
	Description string            `json:"description,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
}

// Tool implements the Bash tool for executing shell commands.
// Working directory persists between calls to support `cd` and other stateful operations.
type Tool struct {
	mu      sync.Mutex
	workDir string
	// Environment variables that persist between calls, set by commands.
	envOverrides map[string]string
}

// New creates a new Bash tool with the given working directory.
func New(workDir string) *Tool {
	return &Tool{
		workDir:      workDir,
		envOverrides: make(map[string]string),
	}
}

func (t *Tool) Name() string        { return "Bash" }
func (t *Tool) Description() string { return "Execute a bash command and return its output." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string", "description": "The bash command to execute"},
			"timeout": {"type": "integer", "description": "Timeout in milliseconds (default 120000, max 600000)"},
			"description": {"type": "string", "description": "Description of what the command does"},
			"env": {"type": "object", "description": "Additional environment variables", "additionalProperties": {"type": "string"}}
		},
		"required": ["command"]
	}`)
}

// WorkDir returns the current working directory (thread-safe).
func (t *Tool) WorkDir() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.workDir
}

func (t *Tool) Execute(ctx context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	if in.Command == "" {
		return &tool.Result{Content: "command is required", IsError: true}, nil
	}

	if msg := checkDangerous(in.Command); msg != "" {
		return &tool.Result{Content: msg, IsError: true}, nil
	}

	timeout := 120 * time.Second
	if in.Timeout > 0 {
		timeout = time.Duration(in.Timeout) * time.Millisecond
		if timeout > 600*time.Second {
			timeout = 600 * time.Second
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	t.mu.Lock()
	currentDir := t.workDir
	t.mu.Unlock()

	// Wrap the command to capture the working directory after execution.
	// This allows `cd` commands to persist between calls.
	wrappedCmd := fmt.Sprintf(`%s
__exit_code=$?
echo ""
echo "__CCX_CWD_MARKER__"
pwd
exit $__exit_code`, in.Command)

	cmd := exec.CommandContext(ctx, "bash", "-c", wrappedCmd)
	cmd.Dir = currentDir

	// Build environment: inherit current env + persisted overrides + per-call env
	env := os.Environ()
	for k, v := range t.envOverrides {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range in.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Parse stdout to extract the working directory marker
	rawOut := stdout.String()
	actualOutput, newDir := extractCWD(rawOut)

	// Update persisted working directory
	if newDir != "" {
		t.mu.Lock()
		t.workDir = newDir
		t.mu.Unlock()
	}

	// Merge stdout and stderr
	var out strings.Builder
	if actualOutput != "" {
		out.WriteString(actualOutput)
	}
	if stderr.Len() > 0 {
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString(stderr.String())
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &tool.Result{Content: fmt.Sprintf("command timed out after %s", timeout), IsError: true}, nil
		}
		content := out.String()
		if content == "" {
			content = err.Error()
		}
		return &tool.Result{Content: truncateOutput(content), IsError: true}, nil
	}

	result := out.String()
	if result == "" {
		result = "(no output)"
	}
	return &tool.Result{Content: truncateOutput(result)}, nil
}

// extractCWD separates the actual command output from the CWD marker.
func extractCWD(raw string) (output string, cwd string) {
	marker := "__CCX_CWD_MARKER__"
	idx := strings.LastIndex(raw, marker)
	if idx < 0 {
		return raw, ""
	}

	output = strings.TrimRight(raw[:idx], "\n")
	rest := strings.TrimSpace(raw[idx+len(marker):])
	if rest != "" {
		lines := strings.SplitN(rest, "\n", 2)
		cwd = strings.TrimSpace(lines[0])
	}
	return output, cwd
}

// truncateOutput trims output to maxOutputSize, appending a truncation notice.
func truncateOutput(s string) string {
	if len(s) <= maxOutputSize {
		return s
	}
	return s[:maxOutputSize] + "\n... (output truncated)"
}

func checkDangerous(cmd string) string {
	lower := strings.ToLower(cmd)
	for _, p := range dangerousPatterns {
		if strings.Contains(lower, p) {
			return fmt.Sprintf("command rejected: contains dangerous pattern %q", p)
		}
	}
	return ""
}
