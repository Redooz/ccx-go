package bash

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBash_Execute(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "echo hello"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "hello")
	assert.False(t, result.IsError)
}

func TestBash_StderrOutput(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "echo error >&2"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "error")
}

func TestBash_DangerousCommand(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "rm -rf /"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "dangerous")
}

func TestBash_EmptyCommand(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": ""})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBash_Timeout(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "sleep 10", "timeout": 500})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "timed out")
}

func TestBash_WorkDirPersistence(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "subdir")
	os.MkdirAll(subDir, 0755)

	tool := New(dir)

	// Change directory
	input, _ := json.Marshal(map[string]any{"command": "cd subdir && pwd"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "subdir")

	// Verify working directory persisted (resolve symlinks for macOS /var -> /private/var)
	resolvedSubDir, _ := filepath.EvalSymlinks(subDir)
	resolvedWorkDir, _ := filepath.EvalSymlinks(tool.WorkDir())
	assert.Equal(t, resolvedSubDir, resolvedWorkDir)

	// Next command should run in the new directory
	input, _ = json.Marshal(map[string]any{"command": "pwd"})
	result, err = tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "subdir")
}

func TestBash_NonZeroExit(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "exit 1"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBash_EnvVars(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{
		"command": "echo $MY_TEST_VAR",
		"env":     map[string]string{"MY_TEST_VAR": "hello123"},
	})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "hello123")
}

func TestBash_StdoutAndStderr(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "echo stdout_line && echo stderr_line >&2"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "stdout_line")
	assert.Contains(t, result.Content, "stderr_line")
}

func TestBash_NoOutput(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "true"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "(no output)", result.Content)
}

func TestExtractCWD(t *testing.T) {
	t.Run("with marker", func(t *testing.T) {
		raw := "hello\nworld\n\n__CCX_CWD_MARKER__\n/tmp\n"
		output, cwd := extractCWD(raw)
		assert.Equal(t, "hello\nworld", output)
		assert.Equal(t, "/tmp", cwd)
	})

	t.Run("without marker", func(t *testing.T) {
		raw := "just output"
		output, cwd := extractCWD(raw)
		assert.Equal(t, "just output", output)
		assert.Equal(t, "", cwd)
	})
}

func TestTruncateOutput(t *testing.T) {
	short := "short"
	assert.Equal(t, short, truncateOutput(short))

	// Create a string larger than maxOutputSize
	long := make([]byte, maxOutputSize+100)
	for i := range long {
		long[i] = 'x'
	}
	result := truncateOutput(string(long))
	assert.Contains(t, result, "truncated")
	assert.LessOrEqual(t, len(result), maxOutputSize+50)
}
