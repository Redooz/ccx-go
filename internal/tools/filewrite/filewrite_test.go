package filewrite

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileWrite_Execute(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": path, "content": "hello world"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "wrote")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestFileWrite_CreateDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "test.txt")
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": path, "content": "nested"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "nested", string(data))
}

func TestFileWrite_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("old"), 0644)
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": path, "content": "new"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
}

func TestFileWrite_EmptyPath(t *testing.T) {
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": "", "content": "x"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
