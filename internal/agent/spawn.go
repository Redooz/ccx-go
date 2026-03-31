package agent

import (
	"context"
	"fmt"
)

// RunFunc is the function that executes agent work.
// It receives the agent definition and returns the result text.
type RunFunc func(ctx context.Context, def Def) (string, error)

// SpawnAgent starts an agent in a goroutine and returns a channel for the result.
func SpawnAgent(ctx context.Context, def Def, registry *Registry, run RunFunc) <-chan Result {
	ch := make(chan Result, 1)

	ctx, cancel := context.WithCancel(ctx)

	entry := &Entry{
		Def:    def,
		Status: StatusRunning,
		Cancel: cancel,
	}

	if def.Name != "" {
		registry.Register(def.Name, entry)
	}

	go func() {
		defer close(ch)
		defer cancel()

		output, err := run(ctx, def)

		result := Result{
			Name:   def.Name,
			Output: output,
		}

		if err != nil {
			result.IsError = true
			result.Err = err
			result.Output = fmt.Sprintf("agent error: %v", err)
			entry.Status = StatusFailed
		} else {
			entry.Status = StatusDone
		}

		entry.Result = &result
		ch <- result
	}()

	return ch
}

// SendMessage sends a message to a named agent via the registry.
// Returns an error if the agent is not found or not running.
func SendMessage(registry *Registry, name string, message string) error {
	entry := registry.Get(name)
	if entry == nil {
		return fmt.Errorf("agent %q not found", name)
	}
	if entry.Status != StatusRunning {
		return fmt.Errorf("agent %q is %s, not running", name, entry.Status)
	}
	// In a real implementation, this would send via a channel on the entry.
	// For now, this is a placeholder for the message-passing protocol.
	return nil
}
