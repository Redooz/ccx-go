package taskupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	TaskID string `json:"taskId"`
	Status string `json:"status"`
}

// Tool updates the status of an existing task.
type Tool struct{}

// New creates a new TaskUpdate tool.
func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "TaskUpdate" }
func (t *Tool) Description() string { return "Update the status of an existing task." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"taskId": {"type": "string", "description": "Task identifier (e.g. task-001)"},
			"status": {"type": "string", "enum": ["pending", "in_progress", "completed"], "description": "New status"}
		},
		"required": ["taskId", "status"]
	}`)
}

func (t *Tool) Execute(_ context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}
	if in.TaskID == "" {
		return &tool.Result{Content: "taskId is required", IsError: true}, nil
	}
	switch in.Status {
	case "pending", "in_progress", "completed":
		// valid
	default:
		return &tool.Result{Content: fmt.Sprintf("invalid status %q (must be pending, in_progress, or completed)", in.Status), IsError: true}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("cannot resolve home dir: %v", err), IsError: true}, nil
	}

	tasksRoot := filepath.Join(home, ".claude", "tasks")
	teams, err := os.ReadDir(tasksRoot)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("no task dirs found: %v", err), IsError: true}, nil
	}

	fileName := in.TaskID + ".json"
	for _, team := range teams {
		if !team.IsDir() {
			continue
		}
		taskPath := filepath.Join(tasksRoot, team.Name(), fileName)
		data, err := os.ReadFile(taskPath)
		if err != nil {
			continue
		}

		var task map[string]interface{}
		if err := json.Unmarshal(data, &task); err != nil {
			return &tool.Result{Content: fmt.Sprintf("failed to parse task file: %v", err), IsError: true}, nil
		}

		task["status"] = in.Status
		updated, _ := json.MarshalIndent(task, "", "  ")
		if err := os.WriteFile(taskPath, updated, 0o644); err != nil {
			return &tool.Result{Content: fmt.Sprintf("failed to write task: %v", err), IsError: true}, nil
		}

		return &tool.Result{Content: fmt.Sprintf("Task %s updated to %q. File: %s", in.TaskID, in.Status, taskPath)}, nil
	}

	return &tool.Result{Content: fmt.Sprintf("task %q not found", in.TaskID), IsError: true}, nil
}
