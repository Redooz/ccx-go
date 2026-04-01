package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func initModel(config AppConfig, ch chan tea.Msg, onSubmit SubmitFunc) Model {
	m := New(config, ch, onSubmit)
	// Initialize viewport with a window size
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return result.(Model)
}

func TestNew(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := New(AppConfig{ModelName: "claude-sonnet-4", Version: "dev", CWD: "/tmp"}, ch, nil)
	assert.Equal(t, StateInput, m.state)
	assert.True(t, m.showWelcome)
	assert.Equal(t, "claude-sonnet-4", m.config.ModelName)
}

func TestModel_Init(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := New(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestModel_InitWithPrompt(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := New(AppConfig{Version: "dev", CWD: "/tmp", InitialPrompt: "hello"}, ch, nil)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestModel_WindowResize(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := New(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	assert.True(t, model.ready)
	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
}

func TestModel_StreamTextMsg(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := initModel(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)

	// First chunk creates a new assistant message
	ch <- DoneMsg{} // pre-load so waitForMsg returns
	updated, _ := m.Update(StreamTextMsg{Text: "hello"})
	model := updated.(Model)
	assert.True(t, model.streaming)
	assert.Len(t, model.messages, 1)
	assert.Equal(t, "assistant", model.messages[0].Role)
	assert.Equal(t, "hello", model.messages[0].Content)
}

func TestModel_ToolExec(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := initModel(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)

	ch <- DoneMsg{} // pre-load
	updated, _ := m.Update(ToolExecMsg{ToolName: "Bash"})
	model := updated.(Model)
	assert.Contains(t, model.statusMsg, "Bash")
	assert.Len(t, model.messages, 1)
	assert.Equal(t, "tool", model.messages[0].Role)
	assert.Equal(t, "Bash", model.messages[0].ToolID)
}

func TestModel_ToolDone(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := initModel(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)

	// Add a pending tool message
	m.messages = append(m.messages, ChatMessage{Role: "tool", ToolID: "Bash"})

	ch <- DoneMsg{} // pre-load
	updated, _ := m.Update(ToolDoneMsg{ToolName: "Bash", Content: "output", IsError: false})
	model := updated.(Model)
	assert.Equal(t, "output", model.messages[0].Content)
}

func TestModel_DoneMsg(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := initModel(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)
	m.state = StateProcessing

	updated, _ := m.Update(DoneMsg{})
	model := updated.(Model)
	assert.Equal(t, StateInput, model.state)
	assert.Equal(t, "ready", model.statusMsg)
}

func TestModel_ErrorMsg(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := initModel(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)
	m.state = StateProcessing

	updated, _ := m.Update(ErrorMsg{Err: fmt.Errorf("api error")})
	model := updated.(Model)
	assert.Equal(t, StateInput, model.state)
	assert.Len(t, model.messages, 1)
	assert.Equal(t, "error", model.messages[0].Role)
	assert.Contains(t, model.messages[0].Content, "api error")
}

func TestRenderMessage_User(t *testing.T) {
	msg := ChatMessage{Role: "user", Content: "hello"}
	result := RenderMessage(msg, 80)
	assert.Contains(t, result, "hello")
}

func TestRenderMessage_Error(t *testing.T) {
	msg := ChatMessage{Role: "error", Content: "something failed"}
	result := RenderMessage(msg, 80)
	assert.Contains(t, result, "something failed")
}

func TestRenderMessage_Tool(t *testing.T) {
	msg := ChatMessage{Role: "tool", ToolID: "Bash", Content: "output here"}
	result := RenderMessage(msg, 80)
	assert.Contains(t, result, "Bash")
	assert.Contains(t, result, "output here")
}

func TestShortModelName(t *testing.T) {
	assert.Equal(t, "Sonnet 4", shortModelName("claude-sonnet-4-20250514"))
	assert.Equal(t, "Opus 4", shortModelName("claude-opus-4-20250514"))
	assert.Equal(t, "Haiku 4", shortModelName("claude-haiku-4-20250514"))
	assert.Equal(t, "custom-model", shortModelName("custom-model"))
}

func TestWelcomeHidesOnSubmit(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	submitted := false
	m := initModel(AppConfig{Version: "dev", CWD: "/tmp"}, ch, func(text string) {
		submitted = true
	})

	assert.True(t, m.showWelcome)

	ch <- DoneMsg{} // pre-load
	updated, _ := m.Update(SubmitMsg{Text: "hello"})
	model := updated.(Model)
	assert.False(t, model.showWelcome)
	assert.True(t, submitted)
	assert.Equal(t, StateProcessing, model.state)
}

func TestMainHeight(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := initModel(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)
	assert.Equal(t, 37, m.mainHeight()) // 40 - 3

	m.height = 5
	assert.Equal(t, 5, m.mainHeight()) // minimum 5
}

func TestViewReady(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	m := New(AppConfig{Version: "dev", CWD: "/tmp"}, ch, nil)
	assert.Equal(t, "initializing...", m.View())

	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)
	assert.NotEqual(t, "initializing...", model.View())
}
