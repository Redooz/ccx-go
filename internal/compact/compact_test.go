package compact

import (
	"context"
	"strings"
	"testing"

	"github.com/anton-abyzov/ccx-go/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateTokens(t *testing.T) {
	assert.Equal(t, 0, EstimateTokens(""))
	assert.Equal(t, 3, EstimateTokens("hello world")) // 11 chars / 4 = ~3
	assert.Equal(t, 1, EstimateTokens("hi"))           // 2 chars / 4 = 1
}

func TestMicroCompact_SmallConversation(t *testing.T) {
	msgs := []api.Message{
		{Role: api.RoleUser, Content: []api.ContentBlock{api.NewTextBlock("hi")}},
	}

	result := MicroCompact(msgs, 4)
	assert.Len(t, result, 1)
	assert.Equal(t, "hi", result[0].Content[0].Text)
}

func TestMicroCompact_TruncatesOldToolResults(t *testing.T) {
	longContent := strings.Repeat("x", 1000)
	msgs := []api.Message{
		{Role: api.RoleUser, Content: []api.ContentBlock{
			{Type: "tool_result", ToolUseID: "t1", Content: longContent},
		}},
		{Role: api.RoleAssistant, Content: []api.ContentBlock{api.NewTextBlock("ok")}},
		{Role: api.RoleUser, Content: []api.ContentBlock{api.NewTextBlock("next")}},
		{Role: api.RoleAssistant, Content: []api.ContentBlock{api.NewTextBlock("reply")}},
		{Role: api.RoleUser, Content: []api.ContentBlock{api.NewTextBlock("last")}},
	}

	result := MicroCompact(msgs, 3)
	assert.Len(t, result, 5)

	// First message (old) should be truncated
	firstContent := result[0].Content[0].Content
	assert.Less(t, len(firstContent), 600)
	assert.Contains(t, firstContent, "truncated")

	// Last messages should be preserved
	assert.Equal(t, "last", result[4].Content[0].Text)
}

func TestAutoCompact_BelowThreshold(t *testing.T) {
	msgs := []api.Message{
		{Role: api.RoleUser, Content: []api.ContentBlock{api.NewTextBlock("hi")}},
	}

	result, compacted, err := AutoCompact(context.Background(), msgs, 1000000, nil)
	require.NoError(t, err)
	assert.False(t, compacted)
	assert.Equal(t, msgs, result)
}

func TestAutoCompact_AboveThreshold(t *testing.T) {
	// Create enough messages to exceed threshold
	var msgs []api.Message
	for i := 0; i < 20; i++ {
		msgs = append(msgs, api.Message{
			Role:    api.RoleUser,
			Content: []api.ContentBlock{api.NewTextBlock(strings.Repeat("word ", 100))},
		})
		msgs = append(msgs, api.Message{
			Role:    api.RoleAssistant,
			Content: []api.ContentBlock{api.NewTextBlock(strings.Repeat("reply ", 100))},
		})
	}

	summarize := func(ctx context.Context, text string) (string, error) {
		return "Summary: discussed many things", nil
	}

	result, compacted, err := AutoCompact(context.Background(), msgs, 10, summarize)
	require.NoError(t, err)
	assert.True(t, compacted)
	assert.Less(t, len(result), len(msgs))

	// First message should be the summary
	assert.Contains(t, result[0].Content[0].Text, "Summary")
}
