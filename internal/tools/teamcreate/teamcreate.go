package teamcreate

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
	TeamName    string `json:"team_name"`
	Description string `json:"description"`
}

// Tool creates a named team with task directory for multi-agent coordination.
type Tool struct{}

// New creates a new TeamCreate tool.
func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "TeamCreate" }
func (t *Tool) Description() string { return "Create a named team with task directory for multi-agent coordination." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"team_name": {"type": "string", "description": "Name of the team to create"},
			"description": {"type": "string", "description": "Description of the team purpose"}
		},
		"required": ["team_name", "description"]
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
	if in.Description == "" {
		return &tool.Result{Content: "description is required", IsError: true}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("cannot resolve home dir: %v", err), IsError: true}, nil
	}

	teamDir := filepath.Join(home, ".claude", "teams", in.TeamName)
	taskDir := filepath.Join(home, ".claude", "tasks", in.TeamName)

	if err := os.MkdirAll(teamDir, 0o755); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to create team dir: %v", err), IsError: true}, nil
	}
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to create task dir: %v", err), IsError: true}, nil
	}

	cfg := map[string]string{
		"name":        in.TeamName,
		"description": in.Description,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	cfgPath := filepath.Join(teamDir, "config.json")
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to write config: %v", err), IsError: true}, nil
	}

	return &tool.Result{Content: fmt.Sprintf("Team %q created.\n  team dir: %s\n  task dir: %s\n  config:   %s", in.TeamName, teamDir, taskDir, cfgPath)}, nil
}
