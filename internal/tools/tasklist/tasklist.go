package tasklist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

// Tool lists all tasks and their statuses.
type Tool struct{}

// New creates a new TaskList tool.
func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "TaskList" }
func (t *Tool) Description() string { return "List all tasks and their statuses." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)
}

func (t *Tool) Execute(_ context.Context, _ json.RawMessage) (*tool.Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("cannot resolve home dir: %v", err), IsError: true}, nil
	}

	tasksRoot := filepath.Join(home, ".claude", "tasks")
	teams, err := os.ReadDir(tasksRoot)
	if err != nil {
		return &tool.Result{Content: "No tasks found."}, nil
	}

	var sb strings.Builder
	total := 0

	for _, team := range teams {
		if !team.IsDir() {
			continue
		}
		teamDir := filepath.Join(tasksRoot, team.Name())
		files, err := os.ReadDir(teamDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(teamDir, f.Name()))
			if err != nil {
				continue
			}
			var task struct {
				ID      string `json:"id"`
				Subject string `json:"subject"`
				Status  string `json:"status"`
			}
			if err := json.Unmarshal(data, &task); err != nil {
				continue
			}
			marker := "[ ]"
			switch task.Status {
			case "in_progress":
				marker = "[~]"
			case "completed":
				marker = "[x]"
			}
			sb.WriteString(fmt.Sprintf("%s %s — %s (team: %s)\n", marker, task.ID, task.Subject, team.Name()))
			total++
		}
	}

	if total == 0 {
		return &tool.Result{Content: "No tasks found."}, nil
	}

	return &tool.Result{Content: fmt.Sprintf("%d task(s):\n%s", total, sb.String())}, nil
}
