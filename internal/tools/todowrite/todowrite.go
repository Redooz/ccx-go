package todowrite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

// TodoItem represents a single todo entry.
type TodoItem struct {
	Content string `json:"content"`
	Status  string `json:"status"` // "pending", "in_progress", "completed"
}

type input struct {
	Todos []TodoItem `json:"todos"`
}

// Tool manages an in-memory todo list.
type Tool struct {
	mu    sync.RWMutex
	todos []TodoItem
}

// New creates a new TodoWrite tool.
func New() *Tool {
	return &Tool{}
}

func (t *Tool) Name() string        { return "TodoWrite" }
func (t *Tool) Description() string { return "Write and manage a todo list for tracking tasks." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"todos": {
				"type": "array",
				"description": "The complete todo list (replaces existing list)",
				"items": {
					"type": "object",
					"properties": {
						"content": {"type": "string", "description": "The todo item text"},
						"status": {"type": "string", "enum": ["pending", "in_progress", "completed"], "description": "Status of the item"}
					},
					"required": ["content", "status"]
				}
			}
		},
		"required": ["todos"]
	}`)
}

func (t *Tool) Execute(_ context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	// Validate statuses
	for i, item := range in.Todos {
		if item.Content == "" {
			return &tool.Result{
				Content: fmt.Sprintf("todo item %d: content is required", i),
				IsError: true,
			}, nil
		}
		switch item.Status {
		case "pending", "in_progress", "completed":
			// valid
		default:
			return &tool.Result{
				Content: fmt.Sprintf("todo item %d: invalid status %q (must be pending, in_progress, or completed)", i, item.Status),
				IsError: true,
			}, nil
		}
	}

	t.mu.Lock()
	t.todos = make([]TodoItem, len(in.Todos))
	copy(t.todos, in.Todos)
	t.mu.Unlock()

	return &tool.Result{Content: t.format()}, nil
}

// Todos returns a copy of the current todo list.
func (t *Tool) Todos() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]TodoItem, len(t.todos))
	copy(out, t.todos)
	return out
}

func (t *Tool) format() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.todos) == 0 {
		return "Todo list is empty."
	}

	var sb strings.Builder
	pending, inProgress, completed := 0, 0, 0
	for _, item := range t.todos {
		switch item.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		}
	}

	sb.WriteString(fmt.Sprintf("Todo list updated (%d items: %d pending, %d in progress, %d completed)\n\n",
		len(t.todos), pending, inProgress, completed))

	for i, item := range t.todos {
		marker := "[ ]"
		switch item.Status {
		case "in_progress":
			marker = "[~]"
		case "completed":
			marker = "[x]"
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, marker, item.Content))
	}

	return sb.String()
}
