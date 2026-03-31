package prompt

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anton-abyzov/ccx-go/internal/tool"
	"github.com/stretchr/testify/assert"
)

type fakeTool struct {
	name string
	desc string
}

func (f *fakeTool) Name() string                                                           { return f.name }
func (f *fakeTool) Description() string                                                    { return f.desc }
func (f *fakeTool) InputSchema() json.RawMessage                                           { return json.RawMessage(`{}`) }
func (f *fakeTool) Execute(_ context.Context, _ json.RawMessage) (*tool.Result, error) { return nil, nil }

func TestBuildSystemPrompt_Basic(t *testing.T) {
	tools := []tool.Tool{
		&fakeTool{name: "Bash", desc: "Run commands"},
		&fakeTool{name: "Read", desc: "Read files"},
	}

	result := BuildSystemPrompt(tools, "", "/home/user/project", "")

	assert.Contains(t, result, "AI coding assistant CLI")
	assert.Contains(t, result, "/home/user/project")
	assert.Contains(t, result, "**Bash**: Run commands")
	assert.Contains(t, result, "**Read**: Read files")
}

func TestBuildSystemPrompt_WithClaudeMD(t *testing.T) {
	result := BuildSystemPrompt(nil, "## My Rules\nAlways use tabs.", "/tmp", "")

	assert.Contains(t, result, "Project Instructions")
	assert.Contains(t, result, "Always use tabs")
}

func TestBuildSystemPrompt_WithExtra(t *testing.T) {
	result := BuildSystemPrompt(nil, "", "/tmp", "Be extra careful.")

	assert.Contains(t, result, "Additional Instructions")
	assert.Contains(t, result, "Be extra careful")
}

func TestBuildSystemPrompt_NoToolsSection(t *testing.T) {
	result := BuildSystemPrompt(nil, "", "/tmp", "")

	assert.NotContains(t, result, "Available Tools")
}

func TestBuildSystemPrompt_AllParts(t *testing.T) {
	tools := []tool.Tool{&fakeTool{name: "Grep", desc: "Search"}}
	result := BuildSystemPrompt(tools, "# Rules\nDo X", "/workspace", "Think step by step")

	assert.Contains(t, result, "AI coding assistant")
	assert.Contains(t, result, "/workspace")
	assert.Contains(t, result, "**Grep**: Search")
	assert.Contains(t, result, "Do X")
	assert.Contains(t, result, "Think step by step")
}
