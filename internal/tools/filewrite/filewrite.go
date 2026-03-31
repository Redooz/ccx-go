package filewrite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// Tool implements the FileWrite tool.
type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "Write" }
func (t *Tool) Description() string { return "Write content to a file, creating directories as needed." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {"type": "string", "description": "Absolute path to the file"},
			"content": {"type": "string", "description": "Content to write"}
		},
		"required": ["file_path", "content"]
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

	dir := filepath.Dir(in.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &tool.Result{Content: fmt.Sprintf("error creating directory: %v", err), IsError: true}, nil
	}

	if err := os.WriteFile(in.FilePath, []byte(in.Content), 0644); err != nil {
		return &tool.Result{Content: fmt.Sprintf("error writing file: %v", err), IsError: true}, nil
	}

	return &tool.Result{Content: fmt.Sprintf("wrote %d bytes to %s", len(in.Content), in.FilePath)}, nil
}
