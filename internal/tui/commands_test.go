package tui

import (
	"testing"

	"github.com/anton-abyzov/ccx-go/internal/cost"
	"github.com/anton-abyzov/ccx-go/internal/tool"
	"github.com/stretchr/testify/assert"
)

func newTestCommandCtx() CommandContext {
	return CommandContext{
		Version:  "1.0.0",
		Model:    "claude-sonnet-4-20250514",
		Tracker:  cost.NewTracker(),
		Registry: tool.NewRegistry(),
	}
}

func TestHandleSlashCommand_NotACommand(t *testing.T) {
	result := HandleSlashCommand("hello world", newTestCommandCtx())
	assert.Nil(t, result)
}

func TestHandleSlashCommand_EmptyInput(t *testing.T) {
	result := HandleSlashCommand("", newTestCommandCtx())
	assert.Nil(t, result)
}

func TestHandleSlashCommand_SlashAlone(t *testing.T) {
	result := HandleSlashCommand("/", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.Contains(t, result.Output, "/help")
	assert.Contains(t, result.Output, "/exit")
	assert.Contains(t, result.Output, "/tools")
}

func TestHandleSlashCommand_Help(t *testing.T) {
	result := HandleSlashCommand("/help", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.False(t, result.Quit)
	assert.Contains(t, result.Output, "Commands:")
	assert.Contains(t, result.Output, "Shortcuts:")
}

func TestHandleSlashCommand_Exit(t *testing.T) {
	result := HandleSlashCommand("/exit", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.True(t, result.Quit)
}

func TestHandleSlashCommand_Quit(t *testing.T) {
	result := HandleSlashCommand("/quit", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.True(t, result.Quit)
}

func TestHandleSlashCommand_Clear(t *testing.T) {
	result := HandleSlashCommand("/clear", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.True(t, result.Clear)
	assert.Contains(t, result.Output, "\033[2J")
}

func TestHandleSlashCommand_Cost(t *testing.T) {
	ctx := newTestCommandCtx()
	ctx.Tracker.Record("claude-sonnet-4-20250514", 1000, 500)
	result := HandleSlashCommand("/cost", ctx)
	assert.NotNil(t, result)
	assert.Contains(t, result.Output, "API calls:")
	assert.Contains(t, result.Output, "1000")
	assert.Contains(t, result.Output, "500")
}

func TestHandleSlashCommand_CostNilTracker(t *testing.T) {
	ctx := newTestCommandCtx()
	ctx.Tracker = nil
	result := HandleSlashCommand("/cost", ctx)
	assert.NotNil(t, result)
	assert.Contains(t, result.Output, "No cost data")
}

func TestHandleSlashCommand_Model(t *testing.T) {
	result := HandleSlashCommand("/model", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.Contains(t, result.Output, "Sonnet 4")
	assert.Contains(t, result.Output, "claude-sonnet-4-20250514")
}

func TestHandleSlashCommand_Compact(t *testing.T) {
	result := HandleSlashCommand("/compact", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.True(t, result.Compact)
	assert.Contains(t, result.Output, "compacted")
}

func TestHandleSlashCommand_Version(t *testing.T) {
	result := HandleSlashCommand("/version", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.Contains(t, result.Output, "1.0.0")
}

func TestHandleSlashCommand_Tools(t *testing.T) {
	result := HandleSlashCommand("/tools", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.Contains(t, result.Output, "No tools registered")
}

func TestHandleSlashCommand_ToolsNilRegistry(t *testing.T) {
	ctx := newTestCommandCtx()
	ctx.Registry = nil
	result := HandleSlashCommand("/tools", ctx)
	assert.NotNil(t, result)
	assert.Contains(t, result.Output, "No tools registered")
}

func TestHandleSlashCommand_Unknown(t *testing.T) {
	result := HandleSlashCommand("/foo", newTestCommandCtx())
	assert.NotNil(t, result)
	assert.Contains(t, result.Output, "Unknown command: /foo")
}
