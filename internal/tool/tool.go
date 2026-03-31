package tool

import (
	"context"
	"encoding/json"
)

// Result represents the output of a tool execution.
type Result struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

// Tool defines the interface that all tools must implement.
type Tool interface {
	Name() string
	Description() string
	InputSchema() json.RawMessage
	Execute(ctx context.Context, input json.RawMessage) (*Result, error)
}
