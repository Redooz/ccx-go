package glob

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// Tool implements file pattern matching.
type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "Glob" }
func (t *Tool) Description() string { return "Find files matching a glob pattern." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Glob pattern (e.g. **/*.go)"},
			"path": {"type": "string", "description": "Directory to search in"}
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

	root := in.Path
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return &tool.Result{Content: fmt.Sprintf("error getting working directory: %v", err), IsError: true}, nil
		}
	}

	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		matched, err := matchGlob(in.Pattern, rel)
		if err != nil {
			return nil
		}
		if matched {
			matches = append(matches, path)
		}
		return nil
	})

	if err != nil && err != context.Canceled {
		return &tool.Result{Content: fmt.Sprintf("error walking directory: %v", err), IsError: true}, nil
	}

	// Sort by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		si, _ := os.Stat(matches[i])
		sj, _ := os.Stat(matches[j])
		if si == nil || sj == nil {
			return false
		}
		return si.ModTime().After(sj.ModTime())
	})

	if len(matches) == 0 {
		return &tool.Result{Content: "no files matched"}, nil
	}

	return &tool.Result{Content: strings.Join(matches, "\n")}, nil
}

// matchGlob matches a file path against a glob pattern with ** support.
func matchGlob(pattern, name string) (bool, error) {
	// Handle ** patterns by trying to match against each path segment combination
	if strings.Contains(pattern, "**") {
		parts := strings.Split(name, string(filepath.Separator))
		patParts := strings.Split(pattern, string(filepath.Separator))
		return matchDoublestar(patParts, parts), nil
	}
	return filepath.Match(pattern, name)
}

func matchDoublestar(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}

	if pattern[0] == "**" {
		rest := pattern[1:]
		// Try matching rest against every suffix of path
		for i := 0; i <= len(path); i++ {
			if matchDoublestar(rest, path[i:]) {
				return true
			}
		}
		return false
	}

	if len(path) == 0 {
		return false
	}

	matched, _ := filepath.Match(pattern[0], path[0])
	if !matched {
		return false
	}
	return matchDoublestar(pattern[1:], path[1:])
}
