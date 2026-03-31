package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSkill(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := `---
name: test-skill
description: A test skill
trigger: test
mode: inline
tags:
  - testing
---

This is the skill body.

It has multiple paragraphs.`

	os.WriteFile(path, []byte(content), 0644)

	skill, err := LoadSkill(path)
	require.NoError(t, err)
	assert.Equal(t, "test-skill", skill.Name)
	assert.Equal(t, "A test skill", skill.Description)
	assert.Equal(t, "test", skill.Trigger)
	assert.Equal(t, "inline", skill.Mode)
	assert.Contains(t, skill.Body, "skill body")
	assert.Contains(t, skill.Body, "multiple paragraphs")
	assert.Equal(t, []string{"testing"}, skill.Tags)
}

func TestLoadSkill_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "simple.md")
	os.WriteFile(path, []byte("Just a body"), 0644)

	skill, err := LoadSkill(path)
	require.NoError(t, err)
	assert.Equal(t, "simple", skill.Name) // from filename
	assert.Equal(t, "Just a body", skill.Body)
}

func TestLoadSkillsFromDir(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\nname: alpha\n---\nbody a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("---\nname: beta\n---\nbody b"), 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("not a skill"), 0644)

	skills, err := LoadSkillsFromDir(dir)
	require.NoError(t, err)
	assert.Len(t, skills, 2)
}

func TestLoadSkillsFromDir_NotExists(t *testing.T) {
	skills, err := LoadSkillsFromDir("/nonexistent")
	require.NoError(t, err)
	assert.Nil(t, skills)
}

func TestFindSkillByName(t *testing.T) {
	skills := []*Skill{
		{Name: "alpha"},
		{Name: "beta"},
	}

	assert.NotNil(t, FindSkillByName(skills, "alpha"))
	assert.NotNil(t, FindSkillByName(skills, "Alpha")) // case insensitive
	assert.Nil(t, FindSkillByName(skills, "gamma"))
}

func TestFindSkillByTrigger(t *testing.T) {
	skills := []*Skill{
		{Name: "commit", Trigger: "commit"},
		{Name: "review", Trigger: "review"},
		{Name: "no-trigger"},
	}

	matches := FindSkillByTrigger(skills, "please commit this")
	assert.Len(t, matches, 1)
	assert.Equal(t, "commit", matches[0].Name)

	matches = FindSkillByTrigger(skills, "nothing relevant")
	assert.Len(t, matches, 0)
}

func TestExecute(t *testing.T) {
	skill := &Skill{Name: "test", Body: "Do something"}

	run := func(ctx context.Context, prompt string) (string, error) {
		return "done: " + prompt, nil
	}

	result, err := Execute(context.Background(), skill, "arg1", run)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Output, "done")
	assert.Contains(t, result.Output, "arg1")
}

func TestExecute_NilSkill(t *testing.T) {
	result, err := Execute(context.Background(), nil, "", nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Output, "not found")
}
