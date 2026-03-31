package cost

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTracker_Record(t *testing.T) {
	tr := NewTracker()
	tr.Record("claude-sonnet-4-20250514", 1000, 500)

	assert.Equal(t, 1, tr.CallCount())

	input, output := tr.TotalTokens()
	assert.Equal(t, 1000, input)
	assert.Equal(t, 500, output)
}

func TestTracker_TotalCost(t *testing.T) {
	tr := NewTracker()

	// 1M input tokens at $3/M + 1M output tokens at $15/M = $18
	tr.Record("claude-sonnet-4-20250514", 1_000_000, 1_000_000)

	cost := tr.TotalCost()
	assert.InDelta(t, 18.0, cost, 0.01)
}

func TestTracker_MultipleCalls(t *testing.T) {
	tr := NewTracker()
	tr.Record("claude-sonnet-4-20250514", 1000, 500)
	tr.Record("claude-sonnet-4-20250514", 2000, 1000)

	assert.Equal(t, 2, tr.CallCount())

	input, output := tr.TotalTokens()
	assert.Equal(t, 3000, input)
	assert.Equal(t, 1500, output)
}

func TestTracker_Summary(t *testing.T) {
	tr := NewTracker()
	tr.Record("claude-sonnet-4-20250514", 10000, 5000)

	summary := tr.Summary()
	assert.Contains(t, summary, "1 calls")
	assert.Contains(t, summary, "10k in")
	assert.Contains(t, summary, "5k out")
	assert.Contains(t, summary, "$")
}

func TestTracker_Reset(t *testing.T) {
	tr := NewTracker()
	tr.Record("claude-sonnet-4-20250514", 1000, 500)
	assert.Equal(t, 1, tr.CallCount())

	tr.Reset()
	assert.Equal(t, 0, tr.CallCount())
}

func TestTracker_UnknownModel(t *testing.T) {
	tr := NewTracker()
	tr.Record("unknown-model", 1000, 500)

	// Should fall back to sonnet pricing
	cost := tr.TotalCost()
	assert.Greater(t, cost, 0.0)
}

func TestCostForUsage_Opus(t *testing.T) {
	u := Usage{Model: "claude-opus-4-20250514", InputTokens: 1_000_000, OutputTokens: 1_000_000}
	cost := costForUsage(u)
	assert.InDelta(t, 90.0, cost, 0.01) // $15 + $75
}
