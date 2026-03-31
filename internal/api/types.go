package api

import "encoding/json"

// Role represents a message role.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a conversation message.
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a block of content within a message.
type ContentBlock struct {
	Type string `json:"type"`

	// Text block
	Text string `json:"text,omitempty"`

	// Thinking block
	Thinking string `json:"thinking,omitempty"`

	// Tool use block
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// Tool result block
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// NewTextBlock creates a text content block.
func NewTextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// NewToolUseBlock creates a tool use content block.
func NewToolUseBlock(id, name string, input json.RawMessage) ContentBlock {
	return ContentBlock{Type: "tool_use", ID: id, Name: name, Input: input}
}

// NewToolResultBlock creates a tool result content block.
func NewToolResultBlock(toolUseID, content string, isError bool) ContentBlock {
	return ContentBlock{Type: "tool_result", ToolUseID: toolUseID, Content: content, IsError: isError}
}

// NewThinkingBlock creates a thinking content block.
func NewThinkingBlock(thinking string) ContentBlock {
	return ContentBlock{Type: "thinking", Thinking: thinking}
}

// ToolDefinition defines a tool for the API.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// Request represents an API request to Claude.
type Request struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []Message        `json:"messages"`
	System    string           `json:"system,omitempty"`
	Tools     []ToolDefinition `json:"tools,omitempty"`
	Stream    bool             `json:"stream"`
}

// Response represents a non-streaming API response.
type Response struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         Role           `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	Usage        Usage          `json:"usage"`
}

// Usage represents token usage information.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamEvent represents a server-sent event from the streaming API.
type StreamEvent struct {
	Type string `json:"type"`

	// message_start
	Message *Response `json:"message,omitempty"`

	// content_block_start, content_block_delta
	Index        int           `json:"index,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
	Delta        *Delta        `json:"delta,omitempty"`

	// message_delta
	Usage *Usage `json:"usage,omitempty"`
}

// Delta represents incremental content in a stream event.
type Delta struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	Thinking   string `json:"thinking,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}
