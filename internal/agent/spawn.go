package agent

import (
	"context"
	"fmt"
)

// RunFunc is the function that executes agent work.
// It receives the agent definition and returns the result text.
type RunFunc func(ctx context.Context, def Def) (string, error)

// SpawnAgent starts an agent in a goroutine and returns a channel for the result.
// The agent is registered in the registry and can receive messages via its Inbox.
func SpawnAgent(ctx context.Context, def Def, registry *Registry, run RunFunc) <-chan Result {
	ch := make(chan Result, 1)

	ctx, cancel := context.WithCancel(ctx)

	entry := NewEntry(def, cancel)

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

// SpawnMultiple launches multiple agents concurrently and collects all results.
func SpawnMultiple(ctx context.Context, defs []Def, registry *Registry, run RunFunc) []<-chan Result {
	channels := make([]<-chan Result, len(defs))
	for i, def := range defs {
		channels[i] = SpawnAgent(ctx, def, registry, run)
	}
	return channels
}

// CollectResults waits for results from multiple agent channels.
func CollectResults(ctx context.Context, channels []<-chan Result) []Result {
	results := make([]Result, 0, len(channels))
	for _, ch := range channels {
		select {
		case <-ctx.Done():
			results = append(results, Result{
				Output:  "cancelled",
				IsError: true,
				Err:     ctx.Err(),
			})
		case result, ok := <-ch:
			if ok {
				results = append(results, result)
			}
		}
	}
	return results
}

// SendMessage sends a message to a named agent via the registry.
// The message is delivered to the agent's Inbox channel.
func SendMessage(registry *Registry, name string, message string) error {
	entry := registry.Get(name)
	if entry == nil {
		return fmt.Errorf("agent %q not found", name)
	}
	if entry.Status != StatusRunning {
		return fmt.Errorf("agent %q is %s, not running", name, entry.Status)
	}

	select {
	case entry.Inbox <- message:
		return nil
	default:
		return fmt.Errorf("agent %q inbox is full", name)
	}
}

// CancelAgent cancels a running agent by name.
func CancelAgent(registry *Registry, name string) error {
	entry := registry.Get(name)
	if entry == nil {
		return fmt.Errorf("agent %q not found", name)
	}
	if entry.Status != StatusRunning {
		return fmt.Errorf("agent %q is %s, not running", name, entry.Status)
	}

	entry.Cancel()
	entry.Status = StatusCancelled
	return nil
}
