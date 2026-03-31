package cost

import (
	"fmt"
	"sync"
	"time"
)

// Model pricing per million tokens (USD).
var pricing = map[string]struct {
	Input  float64
	Output float64
}{
	"claude-sonnet-4-20250514":    {Input: 3.0, Output: 15.0},
	"claude-opus-4-20250514":     {Input: 15.0, Output: 75.0},
	"claude-haiku-3-5-20241022":  {Input: 0.80, Output: 4.0},
}

// Usage records token usage for a single API call.
type Usage struct {
	Model        string
	InputTokens  int
	OutputTokens int
	Timestamp    time.Time
}

// Tracker accumulates API usage and costs for a session.
type Tracker struct {
	mu     sync.RWMutex
	usages []Usage
}

// NewTracker creates a new cost tracker.
func NewTracker() *Tracker {
	return &Tracker{}
}

// Record adds a usage entry.
func (t *Tracker) Record(model string, inputTokens, outputTokens int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usages = append(t.usages, Usage{
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Timestamp:    time.Now(),
	})
}

// TotalTokens returns cumulative input and output tokens.
func (t *Tracker) TotalTokens() (input int, output int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, u := range t.usages {
		input += u.InputTokens
		output += u.OutputTokens
	}
	return
}

// TotalCost returns the estimated total cost in USD.
func (t *Tracker) TotalCost() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var total float64
	for _, u := range t.usages {
		total += costForUsage(u)
	}
	return total
}

// Summary returns a human-readable cost summary.
func (t *Tracker) Summary() string {
	input, output := t.TotalTokens()
	cost := t.TotalCost()
	calls := t.CallCount()

	return fmt.Sprintf("%d calls | %dk in / %dk out | $%.4f",
		calls, input/1000, output/1000, cost)
}

// CallCount returns the number of API calls recorded.
func (t *Tracker) CallCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.usages)
}

// Reset clears all recorded usage.
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usages = nil
}

func costForUsage(u Usage) float64 {
	p, ok := pricing[u.Model]
	if !ok {
		// Default to sonnet pricing
		p = pricing["claude-sonnet-4-20250514"]
	}

	inputCost := float64(u.InputTokens) / 1_000_000 * p.Input
	outputCost := float64(u.OutputTokens) / 1_000_000 * p.Output

	return inputCost + outputCost
}
