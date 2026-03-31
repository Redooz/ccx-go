package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel wraps a textarea for user input.
type InputModel struct {
	textarea textarea.Model
	focused  bool
}

// NewInputModel creates a new input model.
func NewInputModel() InputModel {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.Focus()

	return InputModel{
		textarea: ta,
		focused:  true,
	}
}

// Update handles input events.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the input area.
func (m InputModel) View() string {
	return inputBorderStyle.Render(m.textarea.View())
}

// Value returns the current input text.
func (m InputModel) Value() string {
	return m.textarea.Value()
}

// Reset clears the input.
func (m *InputModel) Reset() {
	m.textarea.Reset()
}

// SetWidth sets the textarea width.
func (m *InputModel) SetWidth(w int) {
	m.textarea.SetWidth(w - 4) // account for border padding
}

// Focus sets focus on the textarea.
func (m *InputModel) Focus() tea.Cmd {
	m.focused = true
	return m.textarea.Focus()
}

// Blur removes focus from the textarea.
func (m *InputModel) Blur() {
	m.focused = false
	m.textarea.Blur()
}
