package glob

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

	files := []string{
		"a.go",
		"b.go",
		"c.txt",
		"sub/d.go",
		"sub/e.txt",
		"sub/deep/f.go",
	}

	for _, f := range files {
		path := filepath.Join(dir, f)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("test"), 0644)
	}

	return dir
}

func TestGlob_SimplePattern(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	input, _ := json.Marshal(map[string]any{"pattern": "*.go", "path": dir})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "a.go")
	assert.Contains(t, result.Content, "b.go")
	assert.NotContains(t, result.Content, "c.txt")
}

func TestGlob_DoublestarPattern(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	input, _ := json.Marshal(map[string]any{"pattern": "**/*.go", "path": dir})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "a.go")
	assert.Contains(t, result.Content, "d.go")
	assert.Contains(t, result.Content, "f.go")
}

func TestGlob_NoMatches(t *testing.T) {
	dir := setupTestDir(t)
	tool := New()

	input, _ := json.Marshal(map[string]any{"pattern": "*.rs", "path": dir})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "no files matched")
}

func TestGlob_EmptyPattern(t *testing.T) {
	tool := New()

	input, _ := json.Marshal(map[string]any{"pattern": ""})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
