package skill

import (
	"os"
	"path/filepath"
	"sort"
)

// DiscoverAll scans known skill directories and returns all discovered skills.
// Scans:
//   - ~/.claude/skills/ (.md files + subdirs with SKILL.md)
//   - .claude/skills/ in cwd (same pattern)
//   - ~/.nvm/versions/node/*/lib/node_modules/specweave/plugins/*/skills/*/SKILL.md
func DiscoverAll() []*Skill {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	cwd, _ := os.Getwd()
	seen := map[string]bool{}
	var all []*Skill

	add := func(skills []*Skill) {
		for _, s := range skills {
			if !seen[s.Name] {
				seen[s.Name] = true
				all = append(all, s)
			}
		}
	}

	// ~/.claude/skills/
	add(scanSkillDir(filepath.Join(home, ".claude", "skills")))

	// .claude/skills/ in cwd
	if cwd != "" {
		add(scanSkillDir(filepath.Join(cwd, ".claude", "skills")))
	}

	// Specweave plugin dirs
	add(scanSpecweavePlugins(home))

	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	return all
}

// scanSkillDir loads .md files from dir and SKILL.md from immediate subdirs.
func scanSkillDir(dir string) []*Skill {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var skills []*Skill
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(dir, e.Name(), "SKILL.md")
			if s, err := LoadSkill(p); err == nil {
				skills = append(skills, s)
			}
			continue
		}
		if filepath.Ext(e.Name()) == ".md" {
			if s, err := LoadSkill(filepath.Join(dir, e.Name())); err == nil {
				skills = append(skills, s)
			}
		}
	}
	return skills
}

// scanSpecweavePlugins finds skills under nvm specweave plugin directories.
func scanSpecweavePlugins(home string) []*Skill {
	nvmDir := filepath.Join(home, ".nvm", "versions", "node")
	nodeVersions, err := os.ReadDir(nvmDir)
	if err != nil {
		return nil
	}

	var skills []*Skill
	for _, nv := range nodeVersions {
		if !nv.IsDir() {
			continue
		}
		pluginsDir := filepath.Join(nvmDir, nv.Name(), "lib", "node_modules", "specweave", "plugins")
		plugins, err := os.ReadDir(pluginsDir)
		if err != nil {
			continue
		}
		for _, plugin := range plugins {
			if !plugin.IsDir() {
				continue
			}
			skillsDir := filepath.Join(pluginsDir, plugin.Name(), "skills")
			entries, err := os.ReadDir(skillsDir)
			if err != nil {
				continue
			}
			for _, se := range entries {
				if !se.IsDir() {
					continue
				}
				p := filepath.Join(skillsDir, se.Name(), "SKILL.md")
				if s, err := LoadSkill(p); err == nil {
					skills = append(skills, s)
				}
			}
		}
	}
	return skills
}
