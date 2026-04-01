package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a loaded skill definition.
type Skill struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Trigger     string            `yaml:"trigger,omitempty"`
	Mode        string            `yaml:"mode,omitempty"` // "inline" or "agent"
	Model       string            `yaml:"model,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	Body        string            `yaml:"-"` // The markdown body after frontmatter
	FilePath    string            `yaml:"-"`
}

// LoadSkill reads a markdown skill file with YAML frontmatter.
func LoadSkill(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading skill file: %w", err)
	}

	content := string(data)
	skill, err := parseSkillContent(content)
	if err != nil {
		return nil, fmt.Errorf("parsing skill %s: %w", path, err)
	}

	skill.FilePath = path
	if skill.Name == "" {
		base := filepath.Base(path)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		if strings.EqualFold(name, "SKILL") {
			// For SKILL.md, derive name from parent directory
			name = filepath.Base(filepath.Dir(path))
		}
		skill.Name = name
	}

	return skill, nil
}

// LoadSkillsFromDir loads all .md skill files from a directory.
func LoadSkillsFromDir(dir string) ([]*Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var skills []*Skill
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		skill, err := LoadSkill(filepath.Join(dir, e.Name()))
		if err != nil {
			continue // skip broken skills
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func parseSkillContent(content string) (*Skill, error) {
	// Check for YAML frontmatter
	if !strings.HasPrefix(content, "---") {
		return &Skill{Body: content}, nil
	}

	// Find closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return &Skill{Body: content}, nil
	}

	frontmatter := rest[:idx]
	body := strings.TrimSpace(rest[idx+4:])

	var skill Skill
	if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	skill.Body = body
	return &skill, nil
}
