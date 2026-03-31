package compact

import (
	"context"
	"fmt"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/api"
)

// SummarizeFunc calls the API to produce a summary of conversation history.
type SummarizeFunc func(ctx context.Context, text string) (string, error)

// AutoCompact checks if the conversation exceeds the token threshold and
// compresses it by summarizing older turns.
func AutoCompact(ctx context.Context, messages []api.Message, threshold int, summarize SummarizeFunc) ([]api.Message, bool, error) {
	// Estimate total tokens
	var texts []string
	for _, m := range messages {
		for _, b := range m.Content {
			texts = append(texts, b.Text+b.Content+b.Thinking)
		}
	}

	total := EstimateTokensForMessages(texts)
	if total < threshold {
		return messages, false, nil
	}

	// First apply micro-compaction
	messages = MicroCompact(messages, 4)

	// Build summary of older messages
	var historyParts []string
	cutoff := len(messages) / 2
	if cutoff < 2 {
		cutoff = 2
	}
	if cutoff >= len(messages) {
		return messages, false, nil
	}

	for _, m := range messages[:cutoff] {
		for _, b := range m.Content {
			if b.Text != "" {
				historyParts = append(historyParts, fmt.Sprintf("[%s] %s", m.Role, b.Text))
			}
		}
	}

	if len(historyParts) == 0 {
		return messages, false, nil
	}

	summary, err := summarize(ctx, strings.Join(historyParts, "\n"))
	if err != nil {
		// Fall back to micro-compact only
		return messages, true, nil
	}

	// Replace older messages with summary
	summaryMsg := api.Message{
		Role: api.RoleUser,
		Content: []api.ContentBlock{
			api.NewTextBlock(fmt.Sprintf("[Previous conversation summary]\n%s", summary)),
		},
	}

	compacted := make([]api.Message, 0, len(messages)-cutoff+1)
	compacted = append(compacted, summaryMsg)
	compacted = append(compacted, messages[cutoff:]...)

	return compacted, true, nil
}
