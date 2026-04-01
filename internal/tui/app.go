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
	StateInput      State = iota
	StateProcessing // API call in progress (streaming, tool execution, etc.)
)

// Message types for TUI communication.

// SubmitMsg carries user input to start processing.
type SubmitMsg struct {
	Text string
}

// StreamTextMsg delivers an incremental text chunk from streaming.
type StreamTextMsg struct {
	Text string
}

// StreamThinkingMsg delivers an incremental thinking/reasoning chunk.
type StreamThinkingMsg struct {
	Text string
}

// ToolExecMsg signals that a tool is being executed.
type ToolExecMsg struct {
	ToolName string
}

// ToolDoneMsg signals tool execution completed.
type ToolDoneMsg struct {
	ToolName string
	Content  string
	IsError  bool
}

// ResponseMsg delivers a complete assistant response (non-streaming).
type ResponseMsg struct {
	Text string
}

// ErrorMsg delivers an error.
type ErrorMsg struct {
	Err error
}

// DoneMsg signals the conversation turn is complete.
type DoneMsg struct{}

// AppConfig holds configuration for the TUI display.
type AppConfig struct {
	ModelName     string
	Version       string
	CWD           string
	InitialPrompt string
}

// SubmitFunc is called when the user submits a message.
type SubmitFunc func(text string)

// Model is the main bubbletea model for the TUI.
type Model struct {
	config      AppConfig
	state       State
	showWelcome bool
	messages    []ChatMessage
	input       InputModel
	viewport    viewport.Model
	spinner     spinner.Model
	width       int
	height      int
	ready       bool

	// Streaming state
	streaming  bool
	streamText string

	// Status message shown during processing
	statusMsg string

	// Async message channel and submit callback
	msgCh    chan tea.Msg
	onSubmit SubmitFunc
}

// New creates a new TUI model.
func New(config AppConfig, msgCh chan tea.Msg, onSubmit SubmitFunc) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return Model{
		config:      config,
		state:       StateInput,
		showWelcome: true,
		input:       NewInputModel(),
		spinner:     s,
		statusMsg:   "ready",
		msgCh:       msgCh,
		onSubmit:    onSubmit,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.spinner.Tick,
		m.input.Focus(),
	}
	if m.config.InitialPrompt != "" {
		prompt := m.config.InitialPrompt
		cmds = append(cmds, func() tea.Msg {
			return SubmitMsg{Text: prompt}
		})
	}
	return tea.Batch(cmds...)
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
				m.input.Reset()
				return m.handleSubmit(text)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width)
		m.recalcViewport()

	case SubmitMsg:
		return m.handleSubmit(msg.Text)

	case StreamTextMsg:
		if !m.streaming {
			m.streaming = true
			m.streamText = ""
			m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: ""})
		}
		m.streamText += msg.Text
		m.messages[len(m.messages)-1].Content = m.streamText
		m.statusMsg = "streaming..."
		m.updateViewport()
		return m, m.waitForMsg()

	case ToolExecMsg:
		if m.streaming {
			m.streaming = false
		}
		m.statusMsg = fmt.Sprintf("running %s...", msg.ToolName)
		m.messages = append(m.messages, ChatMessage{
			Role:   "tool",
			ToolID: msg.ToolName,
		})
		m.updateViewport()
		return m, tea.Batch(m.spinner.Tick, m.waitForMsg())

	case ToolDoneMsg:
		if last := len(m.messages) - 1; last >= 0 && m.messages[last].Role == "tool" {
			m.messages[last].Content = msg.Content
			if msg.IsError {
				m.messages[last].Role = "error"
			}
		}
		m.statusMsg = "thinking..."
		m.updateViewport()
		return m, m.waitForMsg()

	case ResponseMsg:
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: msg.Text})
		m.updateViewport()

	case DoneMsg:
		m.streaming = false
		m.state = StateInput
		m.statusMsg = "ready"
		m.updateViewport()
		return m, m.input.Focus()

	case ErrorMsg:
		m.streaming = false
		m.messages = append(m.messages, ChatMessage{Role: "error", Content: msg.Err.Error()})
		m.state = StateInput
		m.statusMsg = "error"
		m.updateViewport()
		return m, m.input.Focus()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update input in input state
	if m.state == StateInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport scroll
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleSubmit processes a user submission (from Enter key or SubmitMsg).
func (m Model) handleSubmit(text string) (tea.Model, tea.Cmd) {
	m.showWelcome = false
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: text})
	m.state = StateProcessing
	m.streaming = false
	m.streamText = ""
	m.statusMsg = "thinking..."
	m.updateViewport()

	if m.onSubmit != nil {
		m.onSubmit(text)
	}

	return m, tea.Batch(m.spinner.Tick, m.waitForMsg())
}

// waitForMsg returns a command that waits for the next async message.
func (m Model) waitForMsg() tea.Cmd {
	ch := m.msgCh
	return func() tea.Msg {
		return <-ch
	}
}

// View renders the entire TUI.
func (m Model) View() string {
	if !m.ready {
		return "initializing..."
	}

	header := m.renderHeader()

	var main string
	if m.showWelcome {
		main = renderWelcome(m.width, m.mainHeight(), m.config.ModelName, m.config.CWD)
	} else {
		main = m.viewport.View()
	}

	var inputLine string
	if m.state == StateInput {
		inputLine = m.input.View()
	} else {
		inputLine = m.spinner.View() + " " + m.statusMsg
	}

	status := m.renderStatusBar()

	return strings.Join([]string{
		strings.TrimRight(header, "\n"),
		main,
		inputLine,
		status,
	}, "\n")
}

func (m Model) renderHeader() string {
	title := fmt.Sprintf(" ccx-go v%s ", m.config.Version)
	titleW := lipgloss.Width(title)
	left := 6
	right := m.width - left - titleW
	if right < 0 {
		right = 0
	}
	return headerStyle.Render(strings.Repeat("─", left) + title + strings.Repeat("─", right))
}

func (m Model) renderStatusBar() string {
	left := statusLeftStyle.Render("? for shortcuts")
	modelShort := shortModelName(m.config.ModelName)
	right := statusRightStyle.Render(fmt.Sprintf("● %s · /effort", modelShort))
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) mainHeight() int {
	h := m.height - 3 // header(1) + input(1) + status(1)
	if h < 5 {
		h = 5
	}
	return h
}

func (m *Model) recalcViewport() {
	h := m.mainHeight()
	if !m.ready {
		m.viewport = viewport.New(m.width, h)
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = h
	}
	m.updateViewport()
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
