package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// State represents the current app state.
type State int

const (
	StateInput State = iota
	StateWaiting
	StateToolRunning
	StatePermission
)

// ToolExecMsg signals that a tool is being executed.
type ToolExecMsg struct {
	ToolName string
}

// ToolDoneMsg signals tool execution completed.
type ToolDoneMsg struct {
	Content string
	IsError bool
}

// ResponseMsg delivers assistant text.
type ResponseMsg struct {
	Text string
}

// ErrorMsg delivers an error.
type ErrorMsg struct {
	Err error
}

// DoneMsg signals the conversation turn is complete.
type DoneMsg struct{}

// SubmitMsg carries user input to the outer program.
type SubmitMsg struct {
	Text string
}

// Model is the main bubbletea model for the TUI.
type Model struct {
	state    State
	messages []ChatMessage
	input    InputModel
	viewport viewport.Model
	spinner  spinner.Model
	width    int
	height   int
	ready    bool
	model    string
	status   string
}

// New creates a new TUI model.
func New(modelName string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return Model{
		state:   StateInput,
		input:   NewInputModel(),
		spinner: s,
		model:   modelName,
		status:  "ready",
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.input.Focus(),
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.state == StateInput {
				text := strings.TrimSpace(m.input.Value())
				if text == "" {
					return m, nil
				}
				if text == "/exit" || text == "/quit" {
					return m, tea.Quit
				}
				m.messages = append(m.messages, ChatMessage{Role: "user", Content: text})
				m.input.Reset()
				m.state = StateWaiting
				m.status = "thinking..."
				m.updateViewport()
				return m, func() tea.Msg { return SubmitMsg{Text: text} }
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width)

		headerH := 1
		inputH := 5
		statusH := 1
		vpHeight := msg.Height - headerH - inputH - statusH - 2

		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}
		m.updateViewport()

	case ResponseMsg:
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: msg.Text})
		m.updateViewport()

	case ToolExecMsg:
		m.state = StateToolRunning
		m.status = fmt.Sprintf("running %s...", msg.ToolName)
		cmds = append(cmds, m.spinner.Tick)

	case ToolDoneMsg:
		role := "tool"
		if msg.IsError {
			role = "error"
		}
		m.messages = append(m.messages, ChatMessage{Role: role, Content: msg.Content})
		m.updateViewport()

	case DoneMsg:
		m.state = StateInput
		m.status = "ready"
		cmds = append(cmds, m.input.Focus())

	case ErrorMsg:
		m.messages = append(m.messages, ChatMessage{Role: "error", Content: msg.Err.Error()})
		m.state = StateInput
		m.status = "ready"
		m.updateViewport()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update input
	if m.state == StateInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the entire TUI.
func (m Model) View() string {
	if !m.ready {
		return "initializing..."
	}

	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Render(fmt.Sprintf(" ccx-go (%s)", m.model))
	b.WriteString(header)
	b.WriteString("\n")

	// Chat viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Input or spinner
	if m.state == StateInput {
		b.WriteString(m.input.View())
	} else {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(m.status)
	}
	b.WriteString("\n")

	// Status bar
	b.WriteString(statusBarStyle.Render(m.status))

	return b.String()
}

func (m *Model) updateViewport() {
	var content strings.Builder
	for _, msg := range m.messages {
		content.WriteString(RenderMessage(msg, m.width))
	}
	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// AddMessage adds a message externally.
func (m *Model) AddMessage(msg ChatMessage) {
	m.messages = append(m.messages, msg)
	m.updateViewport()
}
