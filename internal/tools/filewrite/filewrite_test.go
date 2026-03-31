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

func TestFileWrite_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exec.sh")
	os.WriteFile(path, []byte("#!/bin/sh\necho old"), 0755)
	tool := New()

	input, _ := json.Marshal(map[string]any{"file_path": path, "content": "#!/bin/sh\necho new"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	info, err := os.Stat(path)
	require.NoError(t, err)
	// Should preserve the executable permission
	assert.True(t, info.Mode().Perm()&0100 != 0, "expected executable permission preserved")
}

func TestFileWrite_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.txt")
	tool := New()

	// Write a file
	input, _ := json.Marshal(map[string]any{"file_path": path, "content": "atomic content"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "atomic content", string(data))

	// No temp files should be left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".ccx-write-", "temp file should be cleaned up")
	}
}

func TestFileWrite_LargeContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	tool := New()

	// Write 1MB of content
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte('A' + (i % 26))
	}

	input, _ := json.Marshal(map[string]any{"file_path": path, "content": string(content)})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "1048576")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, len(content), len(data))
}
