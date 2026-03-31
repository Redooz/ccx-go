package query

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/api"
	"github.com/anton-abyzov/ccx-go/internal/tool"
)

// LoopConfig configures the query loop behavior.
type LoopConfig struct {
	Model     string
	MaxTokens int
	MaxTurns  int
	System    string
}

// Loop implements the core agentic query loop.
type Loop struct {
	client   *api.Client
	registry *tool.Registry
	config   LoopConfig
}

// NewLoop creates a new query loop.
func NewLoop(client *api.Client, registry *tool.Registry, config LoopConfig) *Loop {
	if config.MaxTokens == 0 {
		config.MaxTokens = 16384
	}
	return &Loop{
		client:   client,
		registry: registry,
		config:   config,
	}
}

// Run starts the interactive query loop.
func (l *Loop) Run(ctx context.Context, initialPrompt string) error {
	var messages []api.Message

	prompt := initialPrompt
	if prompt == "" {
		var err error
		prompt, err = readInput(os.Stdin, os.Stdout)
		if err != nil {
			return err
		}
	}

	messages = append(messages, api.Message{
		Role:    api.RoleUser,
		Content: []api.ContentBlock{api.NewTextBlock(prompt)},
	})

	turns := 0
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if l.config.MaxTurns > 0 && turns >= l.config.MaxTurns {
			return nil
		}
		turns++

		resp, err := l.sendRequest(ctx, messages)
		if err != nil {
			return fmt.Errorf("API request failed: %w", err)
		}

		messages = append(messages, api.Message{
			Role:    api.RoleAssistant,
			Content: resp.Content,
		})

		toolUses := extractToolUses(resp.Content)
		printTextBlocks(os.Stdout, resp.Content)

		if len(toolUses) == 0 || resp.StopReason == "end_turn" {
			return nil
		}

		results := l.executeTools(ctx, toolUses)
		messages = append(messages, api.Message{
			Role:    api.RoleUser,
			Content: results,
		})
	}
}

// RunSingle executes a single turn (useful for testing).
func (l *Loop) RunSingle(ctx context.Context, messages []api.Message) (*api.Response, error) {
	return l.sendRequest(ctx, messages)
}

func (l *Loop) sendRequest(ctx context.Context, messages []api.Message) (*api.Response, error) {
	req := &api.Request{
		Model:     l.config.Model,
		MaxTokens: l.config.MaxTokens,
		Messages:  messages,
		System:    l.config.System,
		Tools:     l.registry.APIDefinitions(),
	}

	stream, err := l.client.CreateMessageStream(ctx, req)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	acc := api.NewAccumulator()
	for {
		event, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading stream: %w", err)
		}

		// Print text deltas as they arrive
		if event.Type == "content_block_delta" && event.Delta != nil && event.Delta.Type == "text_delta" {
			fmt.Fprint(os.Stdout, event.Delta.Text)
		}

		if acc.Process(event) {
			break
		}
	}

	resp := acc.Response()
	if resp == nil {
		return nil, fmt.Errorf("no response received")
	}
	return resp, nil
}

func (l *Loop) executeTools(ctx context.Context, toolUses []api.ContentBlock) []api.ContentBlock {
	var results []api.ContentBlock

	for _, tu := range toolUses {
		t := l.registry.FindByName(tu.Name)
		if t == nil {
			results = append(results, api.NewToolResultBlock(
				tu.ID, fmt.Sprintf("tool %q not found", tu.Name), true,
			))
			continue
		}

		result, err := t.Execute(ctx, tu.Input)
		if err != nil {
			results = append(results, api.NewToolResultBlock(
				tu.ID, fmt.Sprintf("error: %v", err), true,
			))
			continue
		}

		results = append(results, api.NewToolResultBlock(
			tu.ID, result.Content, result.IsError,
		))
	}

	return results
}

func extractToolUses(blocks []api.ContentBlock) []api.ContentBlock {
	var toolUses []api.ContentBlock
	for _, b := range blocks {
		if b.Type == "tool_use" {
			toolUses = append(toolUses, b)
		}
	}
	return toolUses
}

func printTextBlocks(w io.Writer, blocks []api.ContentBlock) {
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			if !strings.HasSuffix(b.Text, "\n") {
				fmt.Fprintln(w)
			}
		}
	}
}

func readInput(in io.Reader, out io.Writer) (string, error) {
	fmt.Fprint(out, "> ")
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	return "", io.EOF
}

// ToolResultsFromJSON is a helper for building tool result messages in tests.
func ToolResultsFromJSON(results map[string]string) []api.ContentBlock {
	var blocks []api.ContentBlock
	for id, content := range results {
		blocks = append(blocks, api.NewToolResultBlock(id, content, false))
	}
	return blocks
}

// UserMessage is a convenience constructor for tests.
func UserMessage(text string) api.Message {
	return api.Message{
		Role:    api.RoleUser,
		Content: []api.ContentBlock{api.NewTextBlock(text)},
	}
}

// AssistantMessage builds a message with tool_use blocks for tests.
func AssistantMessage(blocks ...api.ContentBlock) api.Message {
	return api.Message{
		Role:    api.RoleAssistant,
		Content: blocks,
	}
}

// FakeToolInput builds a json.RawMessage from a map for tests.
func FakeToolInput(data map[string]any) json.RawMessage {
	b, _ := json.Marshal(data)
	return b
}
