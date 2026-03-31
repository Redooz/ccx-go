package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// StreamReader reads SSE events from a streaming API response.
type StreamReader struct {
	reader io.ReadCloser
	scanner *bufio.Scanner
}

// NewStreamReader creates a stream reader from a response body.
func NewStreamReader(r io.ReadCloser) *StreamReader {
	return &StreamReader{
		reader:  r,
		scanner: bufio.NewScanner(r),
	}
}

// Next reads the next event from the stream. Returns io.EOF when done.
func (s *StreamReader) Next() (*StreamEvent, error) {
	for s.scanner.Scan() {
		line := s.scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse "event: <type>" line
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")

			// Read the corresponding "data: <json>" line
			if !s.scanner.Scan() {
				return nil, fmt.Errorf("unexpected end of stream after event: %s", eventType)
			}
			dataLine := s.scanner.Text()

			if !strings.HasPrefix(dataLine, "data: ") {
				continue
			}
			data := strings.TrimPrefix(dataLine, "data: ")

			// Handle special events
			switch eventType {
			case "ping":
				continue
			case "error":
				return nil, fmt.Errorf("stream error: %s", data)
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				return nil, fmt.Errorf("unmarshaling event %q: %w", eventType, err)
			}
			event.Type = eventType

			return &event, nil
		}
	}

	if err := s.scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stream: %w", err)
	}

	return nil, io.EOF
}

// Close closes the underlying reader.
func (s *StreamReader) Close() error {
	return s.reader.Close()
}

// Accumulator builds a complete Response from a stream of events.
type Accumulator struct {
	response *Response
	blocks   []ContentBlock
}

// NewAccumulator creates a new stream accumulator.
func NewAccumulator() *Accumulator {
	return &Accumulator{}
}

// Process handles a stream event and updates the accumulated state.
// Returns true when the message is complete.
func (a *Accumulator) Process(event *StreamEvent) bool {
	switch event.Type {
	case "message_start":
		if event.Message != nil {
			a.response = event.Message
			a.blocks = nil
		}

	case "content_block_start":
		if event.ContentBlock != nil {
			block := *event.ContentBlock
			// Reset input for tool_use blocks — the real content arrives via input_json_delta
			if block.Type == "tool_use" {
				block.Input = nil
			}
			a.blocks = append(a.blocks, block)
		}

	case "content_block_delta":
		if event.Delta != nil && event.Index < len(a.blocks) {
			block := &a.blocks[event.Index]
			switch event.Delta.Type {
			case "text_delta":
				block.Text += event.Delta.Text
			case "thinking_delta":
				block.Thinking += event.Delta.Thinking
			case "input_json_delta":
				block.Input = appendJSON(block.Input, event.Delta.Text)
			}
		}

	case "message_delta":
		if event.Delta != nil && a.response != nil {
			a.response.StopReason = event.Delta.StopReason
		}
		if event.Usage != nil && a.response != nil {
			a.response.Usage.OutputTokens = event.Usage.OutputTokens
		}

	case "message_stop":
		if a.response != nil {
			a.response.Content = a.blocks
		}
		return true
	}

	return false
}

// Response returns the accumulated response.
func (a *Accumulator) Response() *Response {
	if a.response != nil {
		a.response.Content = a.blocks
	}
	return a.response
}

func appendJSON(existing json.RawMessage, fragment string) json.RawMessage {
	if len(existing) == 0 {
		return json.RawMessage(fragment)
	}
	return json.RawMessage(string(existing) + fragment)
}
