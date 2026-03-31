package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := New("claude-sonnet-4")
	assert.Equal(t, StateInput, m.state)
	assert.Equal(t, "ready", m.status)
	assert.Equal(t, "claude-sonnet-4", m.model)
}

func TestModel_Init(t *testing.T) {
	m := New("test")
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestModel_WindowResize(t *testing.T) {
	m := New("test")
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	assert.True(t, model.ready)
	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
}

func TestModel_ResponseMsg(t *testing.T) {
	m := New("test")
	// Init with window size first
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	updated, _ := m.Update(ResponseMsg{Text: "hello"})
	model := updated.(Model)
	assert.Len(t, model.messages, 1)
	assert.Equal(t, "assistant", model.messages[0].Role)
}

func TestModel_ToolExec(t *testing.T) {
	m := New("test")
	updated, _ := m.Update(ToolExecMsg{ToolName: "Bash"})
	model := updated.(Model)
	assert.Equal(t, StateToolRunning, model.state)
	assert.Contains(t, model.status, "Bash")
}

func TestModel_DoneMsg(t *testing.T) {
	m := New("test")
	m.state = StateWaiting
	updated, _ := m.Update(DoneMsg{})
	model := updated.(Model)
	assert.Equal(t, StateInput, model.state)
	assert.Equal(t, "ready", model.status)
}

func TestRenderMessage(t *testing.T) {
	msg := ChatMessage{Role: "user", Content: "hello"}
	result := RenderMessage(msg, 80)
	assert.Contains(t, result, "hello")

	msg = ChatMessage{Role: "error", Content: "something failed"}
	result = RenderMessage(msg, 80)
	assert.Contains(t, result, "something failed")
}

func TestChatMessage_Tool(t *testing.T) {
	msg := ChatMessage{Role: "tool", ToolID: "Bash", Content: "output here"}
	result := RenderMessage(msg, 80)
	assert.Contains(t, result, "Bash")
	assert.Contains(t, result, "output here")
}
