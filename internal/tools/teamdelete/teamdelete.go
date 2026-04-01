package teamdelete

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	TeamName string `json:"team_name"`
}

// Tool removes a team and its task directory.
type Tool struct{}

// New creates a new TeamDelete tool.
func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "TeamDelete" }
func (t *Tool) Description() string { return "Remove a team and its task directory." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"team_name": {"type": "string", "description": "Name of the team to delete"}
		},
		"required": ["team_name"]
	}`)
}

func (t *Tool) Execute(_ context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}
	if in.TeamName == "" {
		return &tool.Result{Content: "team_name is required", IsError: true}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("cannot resolve home dir: %v", err), IsError: true}, nil
	}

	teamDir := filepath.Join(home, ".claude", "teams", in.TeamName)
	taskDir := filepath.Join(home, ".claude", "tasks", in.TeamName)

	var removed []string
	if err := os.RemoveAll(teamDir); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to remove team dir: %v", err), IsError: true}, nil
	}
	removed = append(removed, teamDir)

	if err := os.RemoveAll(taskDir); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to remove task dir: %v", err), IsError: true}, nil
	}
	removed = append(removed, taskDir)

	return &tool.Result{Content: fmt.Sprintf("Team %q deleted.\n  removed: %s\n  removed: %s", in.TeamName, removed[0], removed[1])}, nil
}
