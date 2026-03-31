package fileread

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// Tool implements the FileRead tool.
type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "Read" }
func (t *Tool) Description() string { return "Read a file from the local filesystem." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {"type": "string", "description": "Absolute path to the file"},
			"offset": {"type": "integer", "description": "Line number to start reading from (0-based)"},
			"limit": {"type": "integer", "description": "Maximum number of lines to read"}
		},
		"required": ["file_path"]
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

	f, err := os.Open(in.FilePath)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error opening file: %v", err), IsError: true}, nil
	}
	defer f.Close()

	limit := in.Limit
	if limit <= 0 {
		limit = 2000
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer

	var b strings.Builder
	lineNum := 0
	linesRead := 0

	for scanner.Scan() {
		lineNum++
		if in.Offset > 0 && lineNum < in.Offset {
			continue
		}
		if linesRead >= limit {
			break
		}
		fmt.Fprintf(&b, "%d\t%s\n", lineNum, scanner.Text())
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		return &tool.Result{Content: fmt.Sprintf("error reading file: %v", err), IsError: true}, nil
	}

	content := b.String()
	if content == "" {
		content = "(empty file)"
	}

	return &tool.Result{Content: content}, nil
}
