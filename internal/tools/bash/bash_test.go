package bash

import (
	"context"
	"encoding/json"
	"os"
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

func TestBash_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	tool := New(dir)

	input, _ := json.Marshal(map[string]any{"command": "pwd"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, dir)
}

func TestBash_Timeout(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "sleep 10", "timeout": 100})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "timed out")
}

func TestBash_DangerousCommand(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "rm -rf /"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "rejected")
}

func TestBash_Stderr(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "echo err >&2"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "err")
}

func TestBash_EmptyCommand(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": ""})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "required")
}

func TestBash_NonZeroExit(t *testing.T) {
	tool := New(os.TempDir())

	input, _ := json.Marshal(map[string]any{"command": "exit 1"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
