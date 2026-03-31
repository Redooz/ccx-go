package todowrite

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTodoWrite_Name(t *testing.T) {
	tw := New()
	assert.Equal(t, "TodoWrite", tw.Name())
}

func TestTodoWrite_BasicWrite(t *testing.T) {
	tw := New()
	input, _ := json.Marshal(map[string]any{
		"todos": []map[string]any{
			{"content": "Write tests", "status": "pending"},
			{"content": "Fix bug", "status": "in_progress"},
			{"content": "Deploy", "status": "completed"},
		},
	})

	result, err := tw.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "3 items")
	assert.Contains(t, result.Content, "1 pending")
	assert.Contains(t, result.Content, "1 in progress")
	assert.Contains(t, result.Content, "1 completed")
	assert.Contains(t, result.Content, "[ ] Write tests")
	assert.Contains(t, result.Content, "[~] Fix bug")
	assert.Contains(t, result.Content, "[x] Deploy")
}

func TestTodoWrite_Replaces(t *testing.T) {
	tw := New()

	// First write
	input1, _ := json.Marshal(map[string]any{
		"todos": []map[string]any{
			{"content": "Task A", "status": "pending"},
		},
	})
	tw.Execute(context.Background(), input1)
	assert.Len(t, tw.Todos(), 1)

	// Second write replaces
	input2, _ := json.Marshal(map[string]any{
		"todos": []map[string]any{
			{"content": "Task B", "status": "completed"},
			{"content": "Task C", "status": "pending"},
		},
	})
	tw.Execute(context.Background(), input2)
	todos := tw.Todos()
	assert.Len(t, todos, 2)
	assert.Equal(t, "Task B", todos[0].Content)
	assert.Equal(t, "completed", todos[0].Status)
}

func TestTodoWrite_EmptyList(t *testing.T) {
	tw := New()
	input, _ := json.Marshal(map[string]any{
		"todos": []map[string]any{},
	})
	result, err := tw.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "empty")
}

func TestTodoWrite_InvalidStatus(t *testing.T) {
	tw := New()
	input, _ := json.Marshal(map[string]any{
		"todos": []map[string]any{
			{"content": "Bad", "status": "unknown"},
		},
	})
	result, err := tw.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid status")
}

func TestTodoWrite_EmptyContent(t *testing.T) {
	tw := New()
	input, _ := json.Marshal(map[string]any{
		"todos": []map[string]any{
			{"content": "", "status": "pending"},
		},
	})
	result, err := tw.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "content is required")
}

func TestTodoWrite_InvalidJSON(t *testing.T) {
	tw := New()
	result, err := tw.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid input")
}

func TestTodoWrite_ThreadSafety(t *testing.T) {
	tw := New()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			input, _ := json.Marshal(map[string]any{
				"todos": []map[string]any{
					{"content": "concurrent task", "status": "pending"},
				},
			})
			tw.Execute(context.Background(), input)
			_ = tw.Todos()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
