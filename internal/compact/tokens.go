package compact

// EstimateTokens estimates the token count for a string using character-based ratio.
// Claude models average ~4 characters per token for English text.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4 // ceiling division
}

// EstimateTokensForMessages estimates total tokens across multiple text blocks.
func EstimateTokensForMessages(texts []string) int {
	total := 0
	for _, t := range texts {
		total += EstimateTokens(t)
	}
	// Add overhead for message formatting (~4 tokens per message)
	total += len(texts) * 4
	return total
}

// DefaultThreshold is the token count at which auto-compaction triggers.
const DefaultThreshold = 100000
