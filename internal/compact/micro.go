package compact

import (
	"github.com/anton-abyzov/ccx-go/internal/api"
)

// MicroCompact strips large tool results from older conversation turns,
// keeping only the last N turns intact.
func MicroCompact(messages []api.Message, keepLastN int) []api.Message {
	if len(messages) <= keepLastN {
		return messages
	}

	result := make([]api.Message, len(messages))
	copy(result, messages)

	cutoff := len(messages) - keepLastN

	for i := 0; i < cutoff; i++ {
		msg := &result[i]
		var newBlocks []api.ContentBlock

		for _, block := range msg.Content {
			switch block.Type {
			case "tool_result":
				// Truncate large tool results
				content := block.Content
				if len(content) > 500 {
					content = content[:500] + "... (truncated by compaction)"
				}
				newBlocks = append(newBlocks, api.ContentBlock{
					Type:      block.Type,
					ToolUseID: block.ToolUseID,
					Content:   content,
					IsError:   block.IsError,
				})
			default:
				newBlocks = append(newBlocks, block)
			}
		}

		result[i].Content = newBlocks
	}

	return result
}
