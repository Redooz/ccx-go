package taskcreate

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
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

// Tool creates a task for tracking work progress.
type Tool struct{}

// New creates a new TaskCreate tool.
func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "TaskCreate" }
func (t *Tool) Description() string { return "Create a task for tracking work progress." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"subject": {"type": "string", "description": "Short title of the task"},
			"description": {"type": "string", "description": "Detailed description of the task"}
		},
		"required": ["subject", "description"]
	}`)
}

func (t *Tool) Execute(_ context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}
	if in.Subject == "" {
		return &tool.Result{Content: "subject is required", IsError: true}, nil
	}
	if in.Description == "" {
		return &tool.Result{Content: "description is required", IsError: true}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("cannot resolve home dir: %v", err), IsError: true}, nil
	}

	// Find first team task directory
	tasksRoot := filepath.Join(home, ".claude", "tasks")
	entries, err := os.ReadDir(tasksRoot)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("no task dirs found: %v", err), IsError: true}, nil
	}

	var teamName string
	for _, e := range entries {
		if e.IsDir() {
			teamName = e.Name()
			break
		}
	}
	if teamName == "" {
		return &tool.Result{Content: "no team task directory found — create a team first", IsError: true}, nil
	}

	taskDir := filepath.Join(tasksRoot, teamName)

	// Count existing task files to auto-increment
	files, _ := os.ReadDir(taskDir)
	count := 0
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			count++
		}
	}
	taskID := fmt.Sprintf("task-%03d", count+1)

	task := map[string]string{
		"id":          taskID,
		"subject":     in.Subject,
		"description": in.Description,
		"status":      "pending",
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(task, "", "  ")
	taskPath := filepath.Join(taskDir, taskID+".json")
	if err := os.WriteFile(taskPath, data, 0o644); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to write task: %v", err), IsError: true}, nil
	}

	return &tool.Result{Content: fmt.Sprintf("Task created: %s\n  subject: %s\n  path: %s", taskID, in.Subject, taskPath)}, nil
}
