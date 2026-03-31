package notebookedit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotebookEdit_Name(t *testing.T) {
	ne := New()
	assert.Equal(t, "NotebookEdit", ne.Name())
}

func TestNotebookEdit_InvalidInput(t *testing.T) {
	ne := New()
	result, err := ne.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid input")
}

func TestNotebookEdit_EmptyPath(t *testing.T) {
	ne := New()
	input, _ := json.Marshal(map[string]any{
		"notebook_path": "",
		"cell_index":    0,
		"new_source":    "x = 1",
	})
	result, err := ne.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "notebook_path is required")
}

func TestNotebookEdit_FileNotFound(t *testing.T) {
	ne := New()
	input, _ := json.Marshal(map[string]any{
		"notebook_path": "/nonexistent/notebook.ipynb",
		"cell_index":    0,
		"new_source":    "x = 1",
	})
	result, err := ne.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "error reading notebook")
}

func TestNotebookEdit_CellOutOfRange(t *testing.T) {
	nb := createTestNotebook(t, 2)

	ne := New()
	input, _ := json.Marshal(map[string]any{
		"notebook_path": nb,
		"cell_index":    5,
		"new_source":    "x = 1",
	})
	result, err := ne.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "out of range")
}

func TestNotebookEdit_UpdateCell(t *testing.T) {
	nb := createTestNotebook(t, 2)

	ne := New()
	input, _ := json.Marshal(map[string]any{
		"notebook_path": nb,
		"cell_index":    0,
		"new_source":    "print('updated')\nx = 42",
	})
	result, err := ne.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Updated cell 0")

	// Verify the file was updated
	data, _ := os.ReadFile(nb)
	var parsed notebook
	json.Unmarshal(data, &parsed)

	assert.Equal(t, []string{"print('updated')\n", "x = 42"}, parsed.Cells[0].Source)
}

func TestNotebookEdit_ChangeCellType(t *testing.T) {
	nb := createTestNotebook(t, 2)

	ne := New()
	input, _ := json.Marshal(map[string]any{
		"notebook_path": nb,
		"cell_index":    0,
		"new_source":    "# Heading",
		"cell_type":     "markdown",
	})
	result, err := ne.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "markdown")

	data, _ := os.ReadFile(nb)
	var parsed notebook
	json.Unmarshal(data, &parsed)
	assert.Equal(t, "markdown", parsed.Cells[0].CellType)
	assert.Nil(t, parsed.Cells[0].Outputs)
	assert.Nil(t, parsed.Cells[0].ExecutionCount)
}

func TestNotebookEdit_NegativeIndex(t *testing.T) {
	nb := createTestNotebook(t, 1)

	ne := New()
	input, _ := json.Marshal(map[string]any{
		"notebook_path": nb,
		"cell_index":    -1,
		"new_source":    "x",
	})
	result, err := ne.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "out of range")
}

func TestSplitSourceLines(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"single line", []string{"single line"}},
		{"line1\nline2", []string{"line1\n", "line2"}},
		{"line1\nline2\n", []string{"line1\n", "line2\n", ""}},
		{"a\nb\nc", []string{"a\n", "b\n", "c"}},
	}

	for _, tt := range tests {
		result := splitSourceLines(tt.input)
		assert.Equal(t, tt.expected, result, "input: %q", tt.input)
	}
}

// createTestNotebook creates a minimal .ipynb file with N code cells and returns its path.
func createTestNotebook(t *testing.T, cellCount int) string {
	t.Helper()

	cells := make([]cell, cellCount)
	for i := range cells {
		cells[i] = cell{
			CellType: "code",
			Source:   []string{"# cell " + string(rune('0'+i)) + "\n"},
			Metadata: map[string]any{},
			Outputs:  []any{},
		}
	}

	nb := notebook{
		Cells:         cells,
		Metadata:      map[string]any{"kernelspec": map[string]any{"name": "python3"}},
		NBFormat:      4,
		NBFormatMinor: 5,
	}

	data, err := json.MarshalIndent(nb, "", " ")
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "test.ipynb")
	require.NoError(t, os.WriteFile(path, data, 0644))

	return path
}
