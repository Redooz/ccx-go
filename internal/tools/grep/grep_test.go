package grep

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\nfunc Hello() string {\n\treturn \"hello\"\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "world.go"), []byte("package main\nfunc World() string {\n\treturn \"world\"\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("Hello World\n"), 0644)

	return dir
}

func TestGrep_FilesWithMatches(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"pattern": "Hello", "path": dir, "output_mode": "files_with_matches",
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// Should find at least the files containing "Hello"
	assert.NotContains(t, result.Content, "no matches")
}

func TestGrep_ContentMode(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"pattern": "func", "path": dir, "output_mode": "content",
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Hello")
	assert.Contains(t, result.Content, "World")
}

func TestGrep_NoMatches(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"pattern": "nonexistentstring12345", "path": dir,
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "no matches")
}

func TestGrep_EmptyPattern(t *testing.T) {
	tool := New()

	input, _ := json.Marshal(map[string]any{"pattern": ""})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
