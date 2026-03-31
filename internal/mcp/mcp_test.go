package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest_Marshal(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"jsonrpc":"2.0"`)
	assert.Contains(t, string(data), `"method":"tools/list"`)
}

func TestResponse_WithError(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Error:   &RPCError{Code: -32601, Message: "method not found"},
	}

	assert.Equal(t, "method not found", resp.Error.Error())
}

func TestToolDef_Unmarshal(t *testing.T) {
	data := `{"name":"test","description":"a test tool","inputSchema":{"type":"object"}}`

	var tool ToolDef
	err := json.Unmarshal([]byte(data), &tool)
	require.NoError(t, err)
	assert.Equal(t, "test", tool.Name)
	assert.Equal(t, "a test tool", tool.Description)
}

func TestToolCallResult_Unmarshal(t *testing.T) {
	data := `{"content":[{"type":"text","text":"hello"}],"isError":false}`

	var result ToolCallResult
	err := json.Unmarshal([]byte(data), &result)
	require.NoError(t, err)
	assert.Len(t, result.Content, 1)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Equal(t, "hello", result.Content[0].Text)
	assert.False(t, result.IsError)
}

func TestNotification_Marshal(t *testing.T) {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	data, err := json.Marshal(notif)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"notifications/initialized"`)
}

func TestResource_Unmarshal(t *testing.T) {
	data := `{"uri":"file:///test.txt","name":"test","mimeType":"text/plain"}`

	var res Resource
	err := json.Unmarshal([]byte(data), &res)
	require.NoError(t, err)
	assert.Equal(t, "file:///test.txt", res.URI)
	assert.Equal(t, "text/plain", res.MIMEType)
}

func TestPrompt_Unmarshal(t *testing.T) {
	data := `{"name":"greet","description":"greeting","arguments":[{"name":"name","required":true}]}`

	var p Prompt
	err := json.Unmarshal([]byte(data), &p)
	require.NoError(t, err)
	assert.Equal(t, "greet", p.Name)
	assert.Len(t, p.Arguments, 1)
	assert.True(t, p.Arguments[0].Required)
}
