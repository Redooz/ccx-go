package api

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func TestStreamReader_TextDeltas(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"test","usage":{"input_tokens":5,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world!"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(nopCloser{strings.NewReader(sseData)})
	acc := NewAccumulator()

	var events []string
	for {
		event, err := reader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		events = append(events, event.Type)
		acc.Process(event)
	}

	assert.Contains(t, events, "message_start")
	assert.Contains(t, events, "content_block_start")
	assert.Contains(t, events, "content_block_delta")
	assert.Contains(t, events, "message_stop")

	resp := acc.Response()
	require.NotNil(t, resp)
	assert.Equal(t, "msg_1", resp.ID)
	assert.Equal(t, "end_turn", resp.StopReason)
	assert.Equal(t, 3, resp.Usage.OutputTokens)
	require.Len(t, resp.Content, 1)
	assert.Equal(t, "text", resp.Content[0].Type)
	assert.Equal(t, "Hello world!", resp.Content[0].Text)
}

func TestStreamReader_ToolUse(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_2","type":"message","role":"assistant","content":[],"model":"test","usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me read that file."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"Read","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"file_path\":"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"main.go\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":20}}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(nopCloser{strings.NewReader(sseData)})
	acc := NewAccumulator()

	for {
		event, err := reader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if acc.Process(event) {
			break
		}
	}

	resp := acc.Response()
	require.NotNil(t, resp)
	assert.Equal(t, "tool_use", resp.StopReason)
	require.Len(t, resp.Content, 2)

	assert.Equal(t, "text", resp.Content[0].Type)
	assert.Equal(t, "Let me read that file.", resp.Content[0].Text)

	assert.Equal(t, "tool_use", resp.Content[1].Type)
	assert.Equal(t, "toolu_1", resp.Content[1].ID)
	assert.Equal(t, "Read", resp.Content[1].Name)
	assert.Equal(t, `{"file_path":"main.go"}`, string(resp.Content[1].Input))
}

func TestStreamReader_ThinkingBlocks(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_3","type":"message","role":"assistant","content":[],"model":"test","usage":{"input_tokens":5,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Let me think..."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Here's my answer."}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":10}}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(nopCloser{strings.NewReader(sseData)})
	acc := NewAccumulator()

	for {
		event, err := reader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if acc.Process(event) {
			break
		}
	}

	resp := acc.Response()
	require.NotNil(t, resp)
	require.Len(t, resp.Content, 2)

	assert.Equal(t, "thinking", resp.Content[0].Type)
	assert.Equal(t, "Let me think...", resp.Content[0].Thinking)

	assert.Equal(t, "text", resp.Content[1].Type)
	assert.Equal(t, "Here's my answer.", resp.Content[1].Text)
}

func TestStreamReader_SkipsPing(t *testing.T) {
	sseData := `event: ping
data: {}

event: message_start
data: {"type":"message_start","message":{"id":"msg_4","type":"message","role":"assistant","content":[],"model":"test","usage":{"input_tokens":1,"output_tokens":0}}}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(nopCloser{strings.NewReader(sseData)})

	event, err := reader.Next()
	require.NoError(t, err)
	assert.Equal(t, "message_start", event.Type)
}

func TestStreamReader_Error(t *testing.T) {
	sseData := `event: error
data: {"type":"error","error":{"type":"overloaded_error","message":"overloaded"}}

`

	reader := NewStreamReader(nopCloser{strings.NewReader(sseData)})
	_, err := reader.Next()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stream error")
}

func TestStreamReader_EmptyStream(t *testing.T) {
	reader := NewStreamReader(nopCloser{strings.NewReader("")})
	_, err := reader.Next()
	assert.Equal(t, io.EOF, err)
}

func TestAccumulator_NoEvents(t *testing.T) {
	acc := NewAccumulator()
	assert.Nil(t, acc.Response())
}

func TestNewTextBlock(t *testing.T) {
	b := NewTextBlock("hello")
	assert.Equal(t, "text", b.Type)
	assert.Equal(t, "hello", b.Text)
}

func TestNewToolUseBlock(t *testing.T) {
	b := NewToolUseBlock("id1", "Read", []byte(`{"path":"x"}`))
	assert.Equal(t, "tool_use", b.Type)
	assert.Equal(t, "id1", b.ID)
	assert.Equal(t, "Read", b.Name)
}

func TestNewToolResultBlock(t *testing.T) {
	b := NewToolResultBlock("id1", "file content", false)
	assert.Equal(t, "tool_result", b.Type)
	assert.Equal(t, "id1", b.ToolUseID)
	assert.Equal(t, "file content", b.Content)
	assert.False(t, b.IsError)
}

func TestNewThinkingBlock(t *testing.T) {
	b := NewThinkingBlock("thinking...")
	assert.Equal(t, "thinking", b.Type)
	assert.Equal(t, "thinking...", b.Thinking)
}
