package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel wraps a textinput for user input with ❯ prompt.
type InputModel struct {
	ti textinput.Model
}

// NewInputModel creates a new input model.
func NewInputModel() InputModel {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.PromptStyle = promptStyle
	ti.Placeholder = ""
	ti.CharLimit = 0
	ti.Focus()

	return InputModel{ti: ti}
}

// Update handles input events.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

// View renders the input area.
func (m InputModel) View() string {
	return m.ti.View()
}

// Value returns the current input text.
func (m InputModel) Value() string {
	return m.ti.Value()
}

// Reset clears the input.
func (m *InputModel) Reset() {
	m.ti.Reset()
}

// SetValue sets the input text (useful for testing).
func (m *InputModel) SetValue(s string) {
	m.ti.SetValue(s)
}

// SetWidth sets the textinput width.
func (m *InputModel) SetWidth(w int) {
	m.ti.Width = w - 4
}

// Focus sets focus on the textinput.
func (m *InputModel) Focus() tea.Cmd {
	return m.ti.Focus()
}

// Blur removes focus from the textinput.
func (m *InputModel) Blur() {
	m.ti.Blur()
}
