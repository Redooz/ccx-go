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

func TestGrep_CountMode(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"pattern": "func", "path": dir, "output_mode": "count",
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.NotContains(t, result.Content, "no matches")
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

func TestGrep_GlobFilter(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	// Search with glob filter for .go files
	input, _ := json.Marshal(map[string]any{
		"pattern": "func", "path": dir, "glob": "*.go", "output_mode": "content",
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// Should find matches in .go files
	assert.Contains(t, result.Content, "func")
}

func TestGrep_ContextLines(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"pattern": "Hello", "path": dir, "output_mode": "content", "context": 1,
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

func TestLimitLines(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5"

	result := limitLines(input, 3)
	assert.Contains(t, result, "line1")
	assert.Contains(t, result, "line3")
	assert.Contains(t, result, "2 more lines")

	result = limitLines(input, 10)
	assert.Equal(t, input, result)
}

func TestBuildArgs(t *testing.T) {
	t.Run("default files_with_matches", func(t *testing.T) {
		args := buildArgs(input{Pattern: "test", Path: "."})
		assert.Contains(t, args, "-l")
		assert.Contains(t, args, "test")
	})

	t.Run("content mode", func(t *testing.T) {
		args := buildArgs(input{Pattern: "test", OutputMode: "content"})
		assert.Contains(t, args, "-n")
		assert.NotContains(t, args, "-l")
	})

	t.Run("case insensitive", func(t *testing.T) {
		args := buildArgs(input{Pattern: "test", Insensitive: true})
		assert.Contains(t, args, "-i")
	})

	t.Run("multiline", func(t *testing.T) {
		args := buildArgs(input{Pattern: "test", Multiline: true})
		assert.Contains(t, args, "-U")
	})

	t.Run("with type filter", func(t *testing.T) {
		args := buildArgs(input{Pattern: "test", Type: "go"})
		assert.Contains(t, args, "--type")
		assert.Contains(t, args, "go")
	})
}

func TestGrep_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("UPPERCASE\nlowercase\nMiXeD"), 0644)
	tool := New()

	input, _ := json.Marshal(map[string]any{
		"pattern": "uppercase", "path": dir, "output_mode": "content", "-i": true,
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "UPPERCASE")
}
