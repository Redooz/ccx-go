package hooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_NoHooks(t *testing.T) {
	r := NewRunner(nil)
	assert.False(t, r.HasHooks())

	results := r.Run(context.Background(), PreToolUse, "Bash", "{}")
	assert.Len(t, results, 0)
}

func TestRunner_PreHook_Allow(t *testing.T) {
	r := NewRunner([]Hook{
		{Event: PreToolUse, Pattern: "*", Command: "echo allowed"},
	})

	results := r.Run(context.Background(), PreToolUse, "Bash", "{}")
	require.Len(t, results, 1)
	assert.True(t, results[0].ExitOK)
	assert.False(t, results[0].Blocked)
	assert.Contains(t, results[0].Output, "allowed")
}

func TestRunner_PreHook_Block(t *testing.T) {
	r := NewRunner([]Hook{
		{Event: PreToolUse, Pattern: "*", Command: "echo blocked && exit 1"},
	})

	results := r.Run(context.Background(), PreToolUse, "Bash", "{}")
	require.Len(t, results, 1)
	assert.False(t, results[0].ExitOK)
	assert.True(t, results[0].Blocked)
}

func TestRunner_PostHook(t *testing.T) {
	r := NewRunner([]Hook{
		{Event: PostToolUse, Pattern: "Bash", Command: "echo post-hook"},
	})

	results := r.Run(context.Background(), PostToolUse, "Bash", "{}")
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Output, "post-hook")
}

func TestRunner_PatternFiltering(t *testing.T) {
	r := NewRunner([]Hook{
		{Event: PreToolUse, Pattern: "Bash", Command: "echo match"},
	})

	// Should match
	results := r.Run(context.Background(), PreToolUse, "Bash", "{}")
	assert.Len(t, results, 1)

	// Should not match
	results = r.Run(context.Background(), PreToolUse, "Read", "{}")
	assert.Len(t, results, 0)
}

func TestRunner_EnvVars(t *testing.T) {
	r := NewRunner([]Hook{
		{Event: PreToolUse, Pattern: "*", Command: "echo $TOOL_NAME"},
	})

	results := r.Run(context.Background(), PreToolUse, "Bash", "{}")
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Output, "Bash")
}

func TestRunner_Timeout(t *testing.T) {
	r := NewRunner([]Hook{
		{Event: PreToolUse, Pattern: "*", Command: "sleep 10", Timeout: 100},
	})

	results := r.Run(context.Background(), PreToolUse, "Bash", "{}")
	require.Len(t, results, 1)
	assert.False(t, results[0].ExitOK)
}
