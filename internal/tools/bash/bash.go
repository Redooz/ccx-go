package bash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

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
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// Tool implements the Bash tool for executing shell commands.
type Tool struct {
	workDir string
}

// New creates a new Bash tool with the given working directory.
func New(workDir string) *Tool {
	return &Tool{workDir: workDir}
}

func (t *Tool) Name() string        { return "Bash" }
func (t *Tool) Description() string { return "Execute a bash command and return its output." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string", "description": "The bash command to execute"},
			"timeout": {"type": "integer", "description": "Timeout in milliseconds (default 120000)"}
		},
		"required": ["command"]
	}`)
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
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", in.Command)
	cmd.Dir = t.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var out strings.Builder
	if stdout.Len() > 0 {
		out.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString(stderr.String())
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &tool.Result{Content: "command timed out", IsError: true}, nil
		}
		content := out.String()
		if content == "" {
			content = err.Error()
		}
		return &tool.Result{Content: content, IsError: true}, nil
	}

	result := out.String()
	if result == "" {
		result = "(no output)"
	}
	return &tool.Result{Content: result}, nil
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
