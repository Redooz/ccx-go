package config

import (
	"os"
	"path/filepath"
	"strings"
)

// MemoryType categorizes memory entries.
type MemoryType string

const (
	MemoryUser      MemoryType = "user"
	MemoryFeedback  MemoryType = "feedback"
	MemoryProject   MemoryType = "project"
	MemoryReference MemoryType = "reference"
)

// MemoryEntry represents a stored memory.
type MemoryEntry struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        MemoryType `json:"type"`
	Content     string     `json:"content"`
	FilePath    string     `json:"file_path"`
}

// MemoryDir returns the default memory directory.
func MemoryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "memory")
}

// LoadMemoryIndex reads MEMORY.md and returns its content.
func LoadMemoryIndex(dir string) (string, error) {
	path := filepath.Join(dir, "MEMORY.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// ListMemoryFiles returns all .md files in the memory directory (excluding MEMORY.md).
func ListMemoryFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".md") && name != "MEMORY.md" {
			files = append(files, filepath.Join(dir, name))
		}
	}
	return files, nil
}
