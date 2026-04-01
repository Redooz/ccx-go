package sendmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	To      string `json:"to"`
	Message string `json:"message"`
	Summary string `json:"summary"`
}

// Tool sends a message to a teammate or broadcasts to all team members.
type Tool struct{}

// New creates a new SendMessage tool.
func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "SendMessage" }
func (t *Tool) Description() string { return "Send a message to a teammate or broadcast to all team members." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"to": {"type": "string", "description": "Recipient name or 'all' for broadcast"},
			"message": {"type": "string", "description": "Message content"},
			"summary": {"type": "string", "description": "Optional short summary of the message"}
		},
		"required": ["to", "message"]
	}`)
}

func (t *Tool) Execute(_ context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}
	if in.To == "" {
		return &tool.Result{Content: "to is required", IsError: true}, nil
	}
	if in.Message == "" {
		return &tool.Result{Content: "message is required", IsError: true}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("cannot resolve home dir: %v", err), IsError: true}, nil
	}

	// Find first team directory
	teamsRoot := filepath.Join(home, ".claude", "teams")
	entries, err := os.ReadDir(teamsRoot)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("no teams found: %v", err), IsError: true}, nil
	}

	var teamName string
	for _, e := range entries {
		if e.IsDir() {
			teamName = e.Name()
			break
		}
	}
	if teamName == "" {
		return &tool.Result{Content: "no teams found", IsError: true}, nil
	}

	msgDir := filepath.Join(teamsRoot, teamName, "messages")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to create messages dir: %v", err), IsError: true}, nil
	}

	record := map[string]string{
		"from":      "agent",
		"to":        in.To,
		"message":   in.Message,
		"summary":   in.Summary,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	line, _ := json.Marshal(record)
	line = append(line, '\n')

	msgFile := filepath.Join(msgDir, in.To+".jsonl")
	f, err := os.OpenFile(msgFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to open message file: %v", err), IsError: true}, nil
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to write message: %v", err), IsError: true}, nil
	}

	return &tool.Result{Content: fmt.Sprintf("Message sent to %q in team %q. File: %s", in.To, teamName, msgFile)}, nil
}
