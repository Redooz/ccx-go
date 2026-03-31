package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_DefaultRules(t *testing.T) {
	m := NewManager()

	// Read-only tools should be allowed
	assert.Equal(t, Allow, m.Check("Read"))
	assert.Equal(t, Allow, m.Check("Glob"))
	assert.Equal(t, Allow, m.Check("Grep"))

	// Write tools should ask
	assert.Equal(t, Ask, m.Check("Write"))
	assert.Equal(t, Ask, m.Check("Edit"))
	assert.Equal(t, Ask, m.Check("Bash"))

	// Unknown tools should ask
	assert.Equal(t, Ask, m.Check("Unknown"))
}

func TestManager_AlwaysAllow(t *testing.T) {
	m := NewManager()

	assert.Equal(t, Ask, m.Check("Bash"))
	m.SetAlwaysAllow("Bash")
	assert.Equal(t, Allow, m.Check("Bash"))
}

func TestManager_CustomRules(t *testing.T) {
	m := NewManager()
	m.LoadRules([]Rule{
		{ToolPattern: "Bash", Action: Allow},
		{ToolPattern: "*", Action: Deny},
	})

	assert.Equal(t, Allow, m.Check("Bash"))
	assert.Equal(t, Deny, m.Check("Write"))
	assert.Equal(t, Deny, m.Check("Edit"))
}

func TestParseDecision(t *testing.T) {
	assert.Equal(t, Allow, ParseDecision("y"))
	assert.Equal(t, Allow, ParseDecision("yes"))
	assert.Equal(t, Allow, ParseDecision("allow"))
	assert.Equal(t, Deny, ParseDecision("n"))
	assert.Equal(t, Deny, ParseDecision("no"))
	assert.Equal(t, Ask, ParseDecision("maybe"))
}

func TestDecision_String(t *testing.T) {
	assert.Equal(t, "allow", Allow.String())
	assert.Equal(t, "deny", Deny.String())
	assert.Equal(t, "ask", Ask.String())
}
