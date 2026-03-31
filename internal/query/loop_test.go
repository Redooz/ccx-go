package query

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anton-abyzov/ccx-go/internal/api"
	"github.com/anton-abyzov/ccx-go/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// echoTool is a simple tool that echoes its input.
type echoTool struct{}

func (e *echoTool) Name() string                { return "Echo" }
func (e *echoTool) Description() string          { return "echoes input back" }
func (e *echoTool) InputSchema() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`) }
func (e *echoTool) Execute(ctx context.Context, input json.RawMessage) (*tool.Result, error) {
	var data struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(input, &data); err != nil {
		return nil, err
	}
	return &tool.Result{Content: "echo: " + data.Text}, nil
}

func sseResponse(text string, stopReason string) string {
	return fmt.Sprintf(`event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"test","usage":{"input_tokens":5,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"%s"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"%s"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`, text, stopReason)
}

func sseToolUseResponse(toolID, toolName, inputJSON string) string {
	return fmt.Sprintf(`event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"test","usage":{"input_tokens":5,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"%s","name":"%s","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","text":"%s"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":10}}

event: message_stop
data: {"type":"message_stop"}

`, toolID, toolName, escapeJSON(inputJSON))
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	// Strip the surrounding quotes added by json.Marshal
	return string(b[1 : len(b)-1])
}

func TestLoop_RunSingle_TextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseResponse("Hello!", "end_turn")))
	}))
	defer server.Close()

	client := api.NewClient("test-key").WithBaseURL(server.URL)
	registry := tool.NewRegistry()
	loop := NewLoop(client, registry, LoopConfig{Model: "test"})

	messages := []api.Message{UserMessage("Hi")}
	resp, err := loop.RunSingle(context.Background(), messages)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "end_turn", resp.StopReason)
	require.Len(t, resp.Content, 1)
	assert.Equal(t, "Hello!", resp.Content[0].Text)
}

func TestLoop_ToolExecution(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")
		if callCount == 1 {
			// First call: respond with tool_use
			w.Write([]byte(sseToolUseResponse("toolu_1", "Echo", `{"text":"hello"}`)))
		} else {
			// Second call: respond with final text
			w.Write([]byte(sseResponse("Done!", "end_turn")))
		}
	}))
	defer server.Close()

	client := api.NewClient("test-key").WithBaseURL(server.URL)
	registry := tool.NewRegistry()
	registry.Register(&echoTool{})

	loop := NewLoop(client, registry, LoopConfig{Model: "test", MaxTurns: 5})
	err := loop.Run(context.Background(), "Use the echo tool")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestLoop_UnknownTool(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")
		if callCount == 1 {
			w.Write([]byte(sseToolUseResponse("toolu_1", "NonExistent", `{}`)))
		} else {
			w.Write([]byte(sseResponse("OK", "end_turn")))
		}
	}))
	defer server.Close()

	client := api.NewClient("test-key").WithBaseURL(server.URL)
	registry := tool.NewRegistry()

	loop := NewLoop(client, registry, LoopConfig{Model: "test", MaxTurns: 5})
	err := loop.Run(context.Background(), "Use a tool")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestLoop_MaxTurns(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")
		// Always respond with tool_use to force looping
		w.Write([]byte(sseToolUseResponse("toolu_"+fmt.Sprint(callCount), "Echo", `{"text":"again"}`)))
	}))
	defer server.Close()

	client := api.NewClient("test-key").WithBaseURL(server.URL)
	registry := tool.NewRegistry()
	registry.Register(&echoTool{})

	loop := NewLoop(client, registry, LoopConfig{Model: "test", MaxTurns: 3})
	err := loop.Run(context.Background(), "Loop forever")

	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestLoop_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseToolUseResponse("toolu_1", "Echo", `{"text":"x"}`)))
	}))
	defer server.Close()

	client := api.NewClient("test-key").WithBaseURL(server.URL)
	registry := tool.NewRegistry()
	registry.Register(&echoTool{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	loop := NewLoop(client, registry, LoopConfig{Model: "test"})
	err := loop.Run(ctx, "Do something")

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestExtractToolUses(t *testing.T) {
	blocks := []api.ContentBlock{
		api.NewTextBlock("Some text"),
		api.NewToolUseBlock("id1", "Read", json.RawMessage(`{}`)),
		api.NewTextBlock("More text"),
		api.NewToolUseBlock("id2", "Write", json.RawMessage(`{}`)),
	}

	toolUses := extractToolUses(blocks)
	assert.Len(t, toolUses, 2)
	assert.Equal(t, "Read", toolUses[0].Name)
	assert.Equal(t, "Write", toolUses[1].Name)
}

func TestExtractToolUses_NoTools(t *testing.T) {
	blocks := []api.ContentBlock{
		api.NewTextBlock("Just text"),
	}
	toolUses := extractToolUses(blocks)
	assert.Nil(t, toolUses)
}

func TestUserMessage(t *testing.T) {
	msg := UserMessage("hello")
	assert.Equal(t, api.RoleUser, msg.Role)
	require.Len(t, msg.Content, 1)
	assert.Equal(t, "text", msg.Content[0].Type)
	assert.Equal(t, "hello", msg.Content[0].Text)
}
