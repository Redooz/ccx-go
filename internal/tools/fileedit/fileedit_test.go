package fileedit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileEdit_Execute(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"file_path": path, "old_string": "hello", "new_string": "goodbye",
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "replaced 1")

	data, _ := os.ReadFile(path)
	assert.Equal(t, "goodbye world", string(data))
}

func TestFileEdit_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"file_path": path, "old_string": "xyz", "new_string": "abc",
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "not found")
}

func TestFileEdit_Ambiguous(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("aaa bbb aaa"), 0644)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"file_path": path, "old_string": "aaa", "new_string": "ccc",
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "2 times")
}

func TestFileEdit_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("aaa bbb aaa"), 0644)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"file_path": path, "old_string": "aaa", "new_string": "ccc", "replace_all": true,
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	data, _ := os.ReadFile(path)
	assert.Equal(t, "ccc bbb ccc", string(data))
}

func TestFileEdit_SameString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"file_path": path, "old_string": "hello", "new_string": "hello",
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "different")
}
