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

// Tool implements the FileWrite tool with atomic writes.
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

	// Preserve existing file permissions if the file exists
	perm := os.FileMode(0644)
	if info, err := os.Stat(in.FilePath); err == nil {
		perm = info.Mode().Perm()
	}

	// Atomic write: write to temp file in same directory, then rename.
	// This prevents partial writes from corrupting the file.
	tmp, err := os.CreateTemp(dir, ".ccx-write-*")
	if err != nil {
		// Fall back to direct write if temp file creation fails
		if err := os.WriteFile(in.FilePath, []byte(in.Content), perm); err != nil {
			return &tool.Result{Content: fmt.Sprintf("error writing file: %v", err), IsError: true}, nil
		}
		return &tool.Result{Content: fmt.Sprintf("wrote %d bytes to %s", len(in.Content), in.FilePath)}, nil
	}

	tmpPath := tmp.Name()

	// Write content to temp file
	_, err = tmp.WriteString(in.Content)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(tmpPath)
		return &tool.Result{Content: fmt.Sprintf("error writing temp file: %v", err), IsError: true}, nil
	}

	// Set permissions on temp file before rename
	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)
		return &tool.Result{Content: fmt.Sprintf("error setting permissions: %v", err), IsError: true}, nil
	}

	// Atomic rename
	if err := os.Rename(tmpPath, in.FilePath); err != nil {
		os.Remove(tmpPath)
		// Fall back to direct write if rename fails (e.g., cross-device)
		if err := os.WriteFile(in.FilePath, []byte(in.Content), perm); err != nil {
			return &tool.Result{Content: fmt.Sprintf("error writing file: %v", err), IsError: true}, nil
		}
	}

	return &tool.Result{Content: fmt.Sprintf("wrote %d bytes to %s", len(in.Content), in.FilePath)}, nil
}
