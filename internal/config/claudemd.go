package config

import (
	"os"
	"path/filepath"
	"strings"
)

const claudeMDFile = "CLAUDE.md"

// ClaudeMD represents discovered CLAUDE.md content.
type ClaudeMD struct {
	// Content from each discovered CLAUDE.md, ordered from root to cwd
	Entries []ClaudeMDEntry
}

// ClaudeMDEntry is a single CLAUDE.md file.
type ClaudeMDEntry struct {
	Path    string
	Content string
}

// CombinedContent returns all CLAUDE.md content concatenated.
func (c *ClaudeMD) CombinedContent() string {
	var parts []string
	for _, e := range c.Entries {
		parts = append(parts, e.Content)
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// DiscoverClaudeMD walks up the directory tree from startDir looking for CLAUDE.md files.
// Also checks ~/.claude/CLAUDE.md for global instructions.
func DiscoverClaudeMD(startDir string) *ClaudeMD {
	result := &ClaudeMD{}

	// Check global CLAUDE.md first
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".claude", claudeMDFile)
		if content, err := os.ReadFile(globalPath); err == nil {
			result.Entries = append(result.Entries, ClaudeMDEntry{
				Path:    globalPath,
				Content: string(content),
			})
		}
	}

	// Walk up from startDir
	dir := startDir
	var localEntries []ClaudeMDEntry
	for {
		mdPath := filepath.Join(dir, claudeMDFile)
		if content, err := os.ReadFile(mdPath); err == nil {
			localEntries = append(localEntries, ClaudeMDEntry{
				Path:    mdPath,
				Content: string(content),
			})
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Reverse so root comes first
	for i := len(localEntries) - 1; i >= 0; i-- {
		result.Entries = append(result.Entries, localEntries[i])
	}

	return result
}
