package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anton-abyzov/ccx-go/internal/api"
	"github.com/anton-abyzov/ccx-go/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_Name(t *testing.T) {
	at := New(nil, nil, "", "")
	assert.Equal(t, "Agent", at.Name())
}

func TestAgent_EmptyPrompt(t *testing.T) {
	at := New(nil, nil, "", "")
	input, _ := json.Marshal(map[string]any{"prompt": ""})
	result, err := at.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "prompt is required")
}

func TestAgent_InvalidInput(t *testing.T) {
	at := New(nil, nil, "", "")
	result, err := at.Execute(context.Background(), json.RawMessage(`{invalid}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid input")
}

func TestAgent_SubAgentReturnsText(t *testing.T) {
	// Set up a mock server that returns a simple text response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseTextResponse("Hello from sub-agent"))
	}))
	defer server.Close()

	client := api.NewClient("test-key").WithBaseURL(server.URL)
	registry := tool.NewRegistry()

	// Suppress log output in tests
	SetLogWriter(io.Discard)
	defer SetLogWriter(io.Discard)

	at := New(client, registry, "test-model", "test system")
	input, _ := json.Marshal(map[string]any{"prompt": "say hello", "description": "test agent"})
	result, err := at.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Hello from sub-agent")
}

func TestAgent_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hang forever
		select {}
	}))
	defer server.Close()

	client := api.NewClient("test-key").WithBaseURL(server.URL)
	registry := tool.NewRegistry()
	SetLogWriter(io.Discard)

	at := New(client, registry, "test-model", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input, _ := json.Marshal(map[string]any{"prompt": "test"})
	result, err := at.Execute(ctx, input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// sseTextResponse builds a minimal SSE stream that returns a text-only response.
func sseTextResponse(text string) string {
	return fmt.Sprintf(`event: message_start
data: {"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"test","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":5}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"%s"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`, text)
}
