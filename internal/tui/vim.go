package tui

import tea "github.com/charmbracelet/bubbletea"

// VimMode represents the current vim mode.
type VimMode int

const (
	VimInsert VimMode = iota
	VimNormal
)

func (m VimMode) String() string {
	switch m {
	case VimNormal:
		return "NORMAL"
	default:
		return "INSERT"
	}
}

// VimState tracks vim mode state for the input area.
type VimState struct {
	Mode    VimMode
	Enabled bool
}

// NewVimState creates a new vim state (disabled by default).
func NewVimState() VimState {
	return VimState{Mode: VimInsert, Enabled: false}
}

// Toggle enables or disables vim mode.
func (v *VimState) Toggle() {
	v.Enabled = !v.Enabled
	if !v.Enabled {
		v.Mode = VimInsert
	}
}

// HandleKey processes a key event in vim mode.
// Returns the key msg to pass through (nil if consumed) and whether to blur/focus input.
type VimAction struct {
	ConsumeKey bool
	BlurInput  bool
	FocusInput bool
}

// ProcessKey handles vim keybindings.
func (v *VimState) ProcessKey(msg tea.KeyMsg) VimAction {
	if !v.Enabled {
		return VimAction{}
	}

	switch v.Mode {
	case VimNormal:
		return v.handleNormalMode(msg)
	case VimInsert:
		return v.handleInsertMode(msg)
	}

	return VimAction{}
}

func (v *VimState) handleNormalMode(msg tea.KeyMsg) VimAction {
	switch msg.String() {
	case "i":
		v.Mode = VimInsert
		return VimAction{ConsumeKey: true, FocusInput: true}
	case "a":
		v.Mode = VimInsert
		return VimAction{ConsumeKey: true, FocusInput: true}
	case "I":
		v.Mode = VimInsert
		return VimAction{ConsumeKey: true, FocusInput: true}
	case "A":
		v.Mode = VimInsert
		return VimAction{ConsumeKey: true, FocusInput: true}
	case "o":
		v.Mode = VimInsert
		return VimAction{ConsumeKey: true, FocusInput: true}
	// Navigation in normal mode - these are consumed but don't change mode
	case "j", "k", "h", "l", "w", "b", "0", "$", "G", "g":
		return VimAction{ConsumeKey: true}
	}

	return VimAction{}
}

func (v *VimState) handleInsertMode(msg tea.KeyMsg) VimAction {
	switch msg.String() {
	case "esc":
		v.Mode = VimNormal
		return VimAction{ConsumeKey: true, BlurInput: true}
	}

	// In insert mode, all keys pass through to the textarea
	return VimAction{}
}
