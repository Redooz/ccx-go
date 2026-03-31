package fileread

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, lines int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	f, err := os.Create(path)
	require.NoError(t, err)
	for i := 1; i <= lines; i++ {
		fmt.Fprintf(f, "line %d\n", i)
	}
	f.Close()
	return path
}

func TestFileRead_Execute(t *testing.T) {
	path := writeTestFile(t, 5)
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": path})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "1\tline 1")
	assert.Contains(t, result.Content, "5\tline 5")
}

func TestFileRead_Offset(t *testing.T) {
	path := writeTestFile(t, 10)
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": path, "offset": 5, "limit": 3})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "5\tline 5")
	assert.Contains(t, result.Content, "7\tline 7")
	assert.NotContains(t, result.Content, "4\tline 4")
	assert.NotContains(t, result.Content, "8\tline 8")
}

func TestFileRead_NotFound(t *testing.T) {
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": "/nonexistent/file.txt"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "error opening file")
}

func TestFileRead_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte{}, 0644)
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": path})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "empty")
}
