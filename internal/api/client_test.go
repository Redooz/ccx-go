package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c := NewClient("test-key")
	assert.Equal(t, "test-key", c.apiKey)
	assert.Equal(t, defaultBaseURL, c.baseURL)
	assert.Equal(t, defaultVersion, c.version)
}

func TestClient_WithBaseURL(t *testing.T) {
	c := NewClient("key").WithBaseURL("http://localhost:8080")
	assert.Equal(t, "http://localhost:8080", c.baseURL)
}

func TestClient_CreateMessage(t *testing.T) {
	resp := Response{
		ID:   "msg_123",
		Type: "message",
		Role: RoleAssistant,
		Content: []ContentBlock{
			{Type: "text", Text: "Hello!"},
		},
		StopReason: "end_turn",
		Usage:      Usage{InputTokens: 10, OutputTokens: 5},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
		assert.Equal(t, defaultVersion, r.Header.Get("Anthropic-Version"))
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)

		var req Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.False(t, req.Stream)
		assert.Equal(t, "claude-sonnet-4-20250514", req.Model)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key").WithBaseURL(server.URL)
	got, err := client.CreateMessage(context.Background(), &Request{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{NewTextBlock("Hi")}}},
	})

	require.NoError(t, err)
	assert.Equal(t, "msg_123", got.ID)
	assert.Equal(t, "Hello!", got.Content[0].Text)
	assert.Equal(t, "end_turn", got.StopReason)
}

func TestClient_CreateMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "authentication_error",
				"message": "invalid api key",
			},
		})
	}))
	defer server.Close()

	client := NewClient("bad-key").WithBaseURL(server.URL)
	_, err := client.CreateMessage(context.Background(), &Request{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{NewTextBlock("Hi")}}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication_error")
	assert.Contains(t, err.Error(), "invalid api key")
}

func TestClient_CreateMessageStream(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		assert.True(t, req.Stream)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseData))
	}))
	defer server.Close()

	client := NewClient("test-key").WithBaseURL(server.URL)
	stream, err := client.CreateMessageStream(context.Background(), &Request{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{NewTextBlock("Hi")}}},
	})
	require.NoError(t, err)
	defer stream.Close()

	acc := NewAccumulator()
	for {
		event, err := stream.Next()
		if err != nil {
			break
		}
		if acc.Process(event) {
			break
		}
	}

	resp := acc.Response()
	require.NotNil(t, resp)
	assert.Equal(t, "msg_1", resp.ID)
	assert.Equal(t, "end_turn", resp.StopReason)
	require.Len(t, resp.Content, 1)
	assert.Equal(t, "Hello world", resp.Content[0].Text)
}

func TestClient_SetHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "my-key", r.Header.Get("X-API-Key"))
		assert.Equal(t, defaultVersion, r.Header.Get("Anthropic-Version"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{})
	}))
	defer server.Close()

	client := NewClient("my-key").WithBaseURL(server.URL)
	client.CreateMessage(context.Background(), &Request{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{NewTextBlock("test")}}},
	})
}
