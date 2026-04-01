package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanSkillDir_FlatMD(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "commit.md"), []byte("---\nname: commit\ndescription: Run git commit\n---\nCommit body"), 0644)
	os.WriteFile(filepath.Join(dir, "review.md"), []byte("---\nname: review\ndescription: Code review\n---\nReview body"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a skill"), 0644)

	skills := scanSkillDir(dir)
	assert.Len(t, skills, 2)

	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}
	assert.True(t, names["commit"])
	assert.True(t, names["review"])
}

func TestScanSkillDir_SubdirSKILLmd(t *testing.T) {
	dir := t.TempDir()

	// Create subdir with SKILL.md
	subdir := filepath.Join(dir, "sw-increment")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	os.WriteFile(filepath.Join(subdir, "SKILL.md"), []byte("---\nname: sw:increment\ndescription: Plan feature\n---\nIncrement body"), 0644)

	// Create another subdir without SKILL.md (should be ignored)
	emptyDir := filepath.Join(dir, "empty")
	require.NoError(t, os.MkdirAll(emptyDir, 0755))

	skills := scanSkillDir(dir)
	assert.Len(t, skills, 1)
	assert.Equal(t, "sw:increment", skills[0].Name)
}

func TestScanSkillDir_SubdirSKILLmd_NoNameInFrontmatter(t *testing.T) {
	dir := t.TempDir()

	// SKILL.md without name: field — name should come from directory
	subdir := filepath.Join(dir, "skill-creator")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	os.WriteFile(filepath.Join(subdir, "SKILL.md"), []byte("---\ndescription: Create skills\n---\nCreator body"), 0644)

	skills := scanSkillDir(dir)
	assert.Len(t, skills, 1)
	assert.Equal(t, "skill-creator", skills[0].Name)
	assert.Equal(t, "Create skills", skills[0].Description)
}

func TestScanSkillDir_NotExists(t *testing.T) {
	skills := scanSkillDir("/nonexistent/path")
	assert.Nil(t, skills)
}

func TestScanSkillDir_Mixed(t *testing.T) {
	dir := t.TempDir()

	// Flat .md file
	os.WriteFile(filepath.Join(dir, "simple.md"), []byte("---\nname: simple\n---\nSimple body"), 0644)

	// Subdir with SKILL.md
	subdir := filepath.Join(dir, "complex-skill")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	os.WriteFile(filepath.Join(subdir, "SKILL.md"), []byte("---\nname: complex\n---\nComplex body"), 0644)

	skills := scanSkillDir(dir)
	assert.Len(t, skills, 2)
}

func TestScanSpecweavePlugins_NotExists(t *testing.T) {
	skills := scanSpecweavePlugins("/nonexistent")
	assert.Nil(t, skills)
}

func TestScanSpecweavePlugins_WithSkills(t *testing.T) {
	home := t.TempDir()

	// Build: home/.nvm/versions/node/v22/lib/node_modules/specweave/plugins/core/skills/pm/SKILL.md
	skillDir := filepath.Join(home, ".nvm", "versions", "node", "v22", "lib", "node_modules", "specweave", "plugins", "core", "skills", "pm")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: sw:pm\ndescription: Product Manager\n---\nPM body"), 0644)

	skills := scanSpecweavePlugins(home)
	assert.Len(t, skills, 1)
	assert.Equal(t, "sw:pm", skills[0].Name)
}

func TestScanSpecweavePlugins_NoNameInFrontmatter(t *testing.T) {
	home := t.TempDir()

	// SKILL.md without name: — should derive sw:team-lead from path
	skillDir := filepath.Join(home, ".nvm", "versions", "node", "v22", "lib", "node_modules", "specweave", "plugins", "core", "skills", "team-lead")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: Orchestrate teams\n---\nTeam lead body"), 0644)

	skills := scanSpecweavePlugins(home)
	assert.Len(t, skills, 1)
	assert.Equal(t, "sw:team-lead", skills[0].Name)
	assert.Equal(t, "Orchestrate teams", skills[0].Description)
}
