package fileedit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// Tool implements the FileEdit tool for exact string replacement.
type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "Edit" }
func (t *Tool) Description() string { return "Perform exact string replacement in a file." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {"type": "string", "description": "Absolute path to the file"},
			"old_string": {"type": "string", "description": "The exact string to replace"},
			"new_string": {"type": "string", "description": "The replacement string"},
			"replace_all": {"type": "boolean", "description": "Replace all occurrences (default false)"}
		},
		"required": ["file_path", "old_string", "new_string"]
	}`)
}

func (t *Tool) Execute(ctx context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	if in.FilePath == "" {
		return &tool.Result{Content: "file_path is required", IsError: true}, nil
	}
	if in.OldString == in.NewString {
		return &tool.Result{Content: "old_string and new_string must be different", IsError: true}, nil
	}

	data, err := os.ReadFile(in.FilePath)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error reading file: %v", err), IsError: true}, nil
	}

	content := string(data)
	count := strings.Count(content, in.OldString)

	if count == 0 {
		return &tool.Result{Content: "old_string not found in file", IsError: true}, nil
	}

	if !in.ReplaceAll && count > 1 {
		return &tool.Result{
			Content: fmt.Sprintf("old_string found %d times; use replace_all or provide more context to make it unique", count),
			IsError: true,
		}, nil
	}

	var newContent string
	if in.ReplaceAll {
		newContent = strings.ReplaceAll(content, in.OldString, in.NewString)
	} else {
		newContent = strings.Replace(content, in.OldString, in.NewString, 1)
	}

	if err := os.WriteFile(in.FilePath, []byte(newContent), 0644); err != nil {
		return &tool.Result{Content: fmt.Sprintf("error writing file: %v", err), IsError: true}, nil
	}

	return &tool.Result{Content: fmt.Sprintf("replaced %d occurrence(s) in %s", count, in.FilePath)}, nil
}
