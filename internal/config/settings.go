package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings represents the user's configuration.
type Settings struct {
	// Permission rules
	Permissions []PermissionRule `json:"permissions,omitempty"`

	// Model preferences
	Model     string `json:"model,omitempty"`
	MaxTokens int    `json:"maxTokens,omitempty"`

	// Custom system prompt additions
	SystemPrompt string `json:"systemPrompt,omitempty"`

	// Hooks
	PreToolHooks  map[string]string `json:"preToolHooks,omitempty"`
	PostToolHooks map[string]string `json:"postToolHooks,omitempty"`
}

// PermissionRule defines a tool permission in settings.
type PermissionRule struct {
	Tool   string `json:"tool"`
	Action string `json:"action"` // "allow", "deny", "ask"
}

// DefaultSettingsPath returns the default settings file path.
func DefaultSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// LoadSettings reads settings from a JSON file.
func LoadSettings(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{}, nil
		}
		return nil, err
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// SaveSettings writes settings to a JSON file.
func SaveSettings(path string, s *Settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
