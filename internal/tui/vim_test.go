package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestVimState_Toggle(t *testing.T) {
	v := NewVimState()
	assert.False(t, v.Enabled)

	v.Toggle()
	assert.True(t, v.Enabled)

	v.Toggle()
	assert.False(t, v.Enabled)
	assert.Equal(t, VimInsert, v.Mode)
}

func TestVimMode_String(t *testing.T) {
	assert.Equal(t, "INSERT", VimInsert.String())
	assert.Equal(t, "NORMAL", VimNormal.String())
}

func TestVimState_Disabled(t *testing.T) {
	v := NewVimState()
	action := v.ProcessKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	assert.False(t, action.ConsumeKey)
}

func TestVimState_NormalToInsert(t *testing.T) {
	v := VimState{Enabled: true, Mode: VimNormal}

	action := v.ProcessKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	assert.True(t, action.ConsumeKey)
	assert.True(t, action.FocusInput)
	assert.Equal(t, VimInsert, v.Mode)
}

func TestVimState_InsertToNormal(t *testing.T) {
	v := VimState{Enabled: true, Mode: VimInsert}

	action := v.ProcessKey(tea.KeyMsg{Type: tea.KeyEscape})
	assert.True(t, action.ConsumeKey)
	assert.True(t, action.BlurInput)
	assert.Equal(t, VimNormal, v.Mode)
}

func TestVimState_NormalNavigation(t *testing.T) {
	v := VimState{Enabled: true, Mode: VimNormal}

	for _, key := range []rune{'j', 'k', 'h', 'l', 'w', 'b'} {
		action := v.ProcessKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		assert.True(t, action.ConsumeKey)
		assert.Equal(t, VimNormal, v.Mode, "key %c should not change mode", key)
	}
}

func TestVimState_InsertPassthrough(t *testing.T) {
	v := VimState{Enabled: true, Mode: VimInsert}

	// Regular keys should pass through in insert mode
	action := v.ProcessKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.False(t, action.ConsumeKey)
}
