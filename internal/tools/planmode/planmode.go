package planmode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

const markerFile = ".plan_mode"

func markerPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", markerFile), nil
}

// EnterTool sets read-only planning mode.
type EnterTool struct{}

// NewEnter creates a new EnterPlanMode tool.
func NewEnter() *EnterTool { return &EnterTool{} }

func (t *EnterTool) Name() string        { return "EnterPlanMode" }
func (t *EnterTool) Description() string { return "Set read-only planning mode. Tools that modify files will be blocked." }

func (t *EnterTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)
}

func (t *EnterTool) Execute(_ context.Context, _ json.RawMessage) (*tool.Result, error) {
	p, err := markerPath()
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("cannot resolve home dir: %v", err), IsError: true}, nil
	}

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to create dir: %v", err), IsError: true}, nil
	}

	if err := os.WriteFile(p, []byte("plan"), 0o644); err != nil {
		return &tool.Result{Content: fmt.Sprintf("failed to enter plan mode: %v", err), IsError: true}, nil
	}

	return &tool.Result{Content: fmt.Sprintf("Plan mode enabled. Marker: %s", p)}, nil
}

// ExitTool exits planning mode.
type ExitTool struct{}

// NewExit creates a new ExitPlanMode tool.
func NewExit() *ExitTool { return &ExitTool{} }

func (t *ExitTool) Name() string        { return "ExitPlanMode" }
func (t *ExitTool) Description() string { return "Exit planning mode and allow file modifications." }

func (t *ExitTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)
}

func (t *ExitTool) Execute(_ context.Context, _ json.RawMessage) (*tool.Result, error) {
	p, err := markerPath()
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("cannot resolve home dir: %v", err), IsError: true}, nil
	}

	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return &tool.Result{Content: fmt.Sprintf("failed to exit plan mode: %v", err), IsError: true}, nil
	}

	return &tool.Result{Content: "Plan mode disabled. File modifications allowed."}, nil
}
