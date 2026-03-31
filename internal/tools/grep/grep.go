package grep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path,omitempty"`
	Glob       string `json:"glob,omitempty"`
	OutputMode string `json:"output_mode,omitempty"`
	Context    int    `json:"context,omitempty"`
}

// Tool implements search via ripgrep.
type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "Grep" }
func (t *Tool) Description() string { return "Search file contents using ripgrep." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Regex pattern to search for"},
			"path": {"type": "string", "description": "Directory or file to search"},
			"glob": {"type": "string", "description": "Glob filter (e.g. *.go)"},
			"output_mode": {"type": "string", "enum": ["content", "files_with_matches", "count"], "description": "Output mode"},
			"context": {"type": "integer", "description": "Lines of context around matches"}
		},
		"required": ["pattern"]
	}`)
}

func (t *Tool) Execute(ctx context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	if in.Pattern == "" {
		return &tool.Result{Content: "pattern is required", IsError: true}, nil
	}

	args := buildArgs(in)

	cmd := exec.CommandContext(ctx, "rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// rg exits with 1 when no matches found - that's ok
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return &tool.Result{Content: "no matches found"}, nil
		}
		// Check if rg is not installed, fall back to grep
		if isNotFound(err) {
			return t.fallbackGrep(ctx, in)
		}
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &tool.Result{Content: fmt.Sprintf("search error: %s", errMsg), IsError: true}, nil
	}

	output := stdout.String()
	if output == "" {
		return &tool.Result{Content: "no matches found"}, nil
	}
	return &tool.Result{Content: strings.TrimRight(output, "\n")}, nil
}

func buildArgs(in input) []string {
	var args []string

	switch in.OutputMode {
	case "files_with_matches", "":
		args = append(args, "-l")
	case "count":
		args = append(args, "-c")
	default:
		args = append(args, "-n") // content mode with line numbers
	}

	if in.Context > 0 {
		args = append(args, "-C", fmt.Sprintf("%d", in.Context))
	}

	if in.Glob != "" {
		args = append(args, "--glob", in.Glob)
	}

	args = append(args, in.Pattern)

	if in.Path != "" {
		args = append(args, in.Path)
	} else {
		args = append(args, ".")
	}

	return args
}

func (t *Tool) fallbackGrep(ctx context.Context, in input) (*tool.Result, error) {
	args := []string{"-rn", in.Pattern}
	if in.Path != "" {
		args = append(args, in.Path)
	} else {
		args = append(args, ".")
	}

	cmd := exec.CommandContext(ctx, "grep", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return &tool.Result{Content: "no matches found"}, nil
		}
		return &tool.Result{Content: fmt.Sprintf("search error: %v", err), IsError: true}, nil
	}

	output := stdout.String()
	if output == "" {
		return &tool.Result{Content: "no matches found"}, nil
	}
	return &tool.Result{Content: strings.TrimRight(output, "\n")}, nil
}

func isNotFound(err error) bool {
	if _, ok := err.(*exec.Error); ok {
		return true
	}
	return false
}
