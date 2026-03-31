package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSettings_NotFound(t *testing.T) {
	s, err := LoadSettings("/nonexistent/path/settings.json")
	require.NoError(t, err)
	assert.NotNil(t, s)
}

func TestLoadSettings_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{"model":"claude-opus-4","maxTokens":4096}`), 0644)

	s, err := LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4", s.Model)
	assert.Equal(t, 4096, s.MaxTokens)
}

func TestSaveSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	s := &Settings{Model: "test-model", MaxTokens: 1024}
	err := SaveSettings(path, s)
	require.NoError(t, err)

	loaded, err := LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, "test-model", loaded.Model)
}

func TestDiscoverClaudeMD(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	os.MkdirAll(sub, 0755)

	// Create CLAUDE.md at root and subdirectory
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("root instructions"), 0644)
	os.WriteFile(filepath.Join(sub, "CLAUDE.md"), []byte("sub instructions"), 0644)

	result := DiscoverClaudeMD(sub)
	// Should find at least the two local ones (may also find global)
	found := 0
	for _, e := range result.Entries {
		if e.Content == "root instructions" || e.Content == "sub instructions" {
			found++
		}
	}
	assert.GreaterOrEqual(t, found, 2)
}

func TestClaudeMD_CombinedContent(t *testing.T) {
	md := &ClaudeMD{
		Entries: []ClaudeMDEntry{
			{Content: "part1"},
			{Content: "part2"},
		},
	}
	combined := md.CombinedContent()
	assert.Contains(t, combined, "part1")
	assert.Contains(t, combined, "part2")
}

func TestLoadMemoryIndex_NotFound(t *testing.T) {
	content, err := LoadMemoryIndex("/nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", content)
}

func TestLoadMemoryIndex_Valid(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# Memories\n- item1"), 0644)

	content, err := LoadMemoryIndex(dir)
	require.NoError(t, err)
	assert.Contains(t, content, "item1")
}

func TestListMemoryFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("index"), 0644)
	os.WriteFile(filepath.Join(dir, "user_prefs.md"), []byte("prefs"), 0644)
	os.WriteFile(filepath.Join(dir, "feedback.md"), []byte("feedback"), 0644)

	files, err := ListMemoryFiles(dir)
	require.NoError(t, err)
	assert.Len(t, files, 2) // excludes MEMORY.md
}
