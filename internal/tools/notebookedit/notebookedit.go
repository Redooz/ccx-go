package notebookedit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	NotebookPath string `json:"notebook_path"`
	CellIndex    int    `json:"cell_index"`
	NewSource    string `json:"new_source"`
	CellType     string `json:"cell_type,omitempty"` // "code" or "markdown", default keeps existing
}

// Tool edits Jupyter notebook (.ipynb) JSON files.
type Tool struct{}

// New creates a new NotebookEdit tool.
func New() *Tool {
	return &Tool{}
}

func (t *Tool) Name() string        { return "NotebookEdit" }
func (t *Tool) Description() string { return "Edit a cell in a Jupyter notebook (.ipynb) file." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"notebook_path": {"type": "string", "description": "Absolute path to the .ipynb file"},
			"cell_index": {"type": "integer", "description": "Zero-based index of the cell to edit"},
			"new_source": {"type": "string", "description": "The new source content for the cell"},
			"cell_type": {"type": "string", "enum": ["code", "markdown"], "description": "Optionally change the cell type"}
		},
		"required": ["notebook_path", "cell_index", "new_source"]
	}`)
}

// notebook and cell represent the minimal ipynb JSON structure we need.
type notebook struct {
	Cells         []cell         `json:"cells"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	NBFormat      int            `json:"nbformat"`
	NBFormatMinor int            `json:"nbformat_minor"`
}

type cell struct {
	CellType       string         `json:"cell_type"`
	Source          []string       `json:"source"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Outputs        []any          `json:"outputs,omitempty"`
	ExecutionCount *int           `json:"execution_count,omitempty"`
}

func (t *Tool) Execute(_ context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	if in.NotebookPath == "" {
		return &tool.Result{Content: "notebook_path is required", IsError: true}, nil
	}

	// Read notebook
	data, err := os.ReadFile(in.NotebookPath)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error reading notebook: %v", err), IsError: true}, nil
	}

	var nb notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return &tool.Result{Content: fmt.Sprintf("error parsing notebook JSON: %v", err), IsError: true}, nil
	}

	if in.CellIndex < 0 || in.CellIndex >= len(nb.Cells) {
		return &tool.Result{
			Content: fmt.Sprintf("cell_index %d out of range (notebook has %d cells)", in.CellIndex, len(nb.Cells)),
			IsError: true,
		}, nil
	}

	// Update cell source — ipynb stores source as array of lines
	nb.Cells[in.CellIndex].Source = splitSourceLines(in.NewSource)

	// Optionally change cell type
	if in.CellType != "" {
		oldType := nb.Cells[in.CellIndex].CellType
		nb.Cells[in.CellIndex].CellType = in.CellType
		// If changing to markdown, clear code-specific fields
		if in.CellType == "markdown" && oldType == "code" {
			nb.Cells[in.CellIndex].Outputs = nil
			nb.Cells[in.CellIndex].ExecutionCount = nil
		}
		// If changing to code, ensure outputs exists
		if in.CellType == "code" && oldType == "markdown" {
			nb.Cells[in.CellIndex].Outputs = []any{}
			nb.Cells[in.CellIndex].ExecutionCount = nil
		}
	}

	// Write back
	out, err := json.MarshalIndent(nb, "", " ")
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error serializing notebook: %v", err), IsError: true}, nil
	}

	if err := os.WriteFile(in.NotebookPath, out, 0644); err != nil {
		return &tool.Result{Content: fmt.Sprintf("error writing notebook: %v", err), IsError: true}, nil
	}

	cellType := nb.Cells[in.CellIndex].CellType
	return &tool.Result{
		Content: fmt.Sprintf("Updated cell %d (%s) in %s", in.CellIndex, cellType, in.NotebookPath),
	}, nil
}

// splitSourceLines splits source into lines preserving newlines as ipynb expects.
// Each line except the last should end with \n.
func splitSourceLines(s string) []string {
	if s == "" {
		return []string{}
	}
	var lines []string
	remaining := s
	for {
		idx := indexOf(remaining, '\n')
		if idx < 0 {
			lines = append(lines, remaining)
			break
		}
		lines = append(lines, remaining[:idx+1])
		remaining = remaining[idx+1:]
	}
	return lines
}

func indexOf(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
