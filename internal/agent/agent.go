package agent

import (
	"context"
	"sync"
)

// Def defines an agent to be spawned.
type Def struct {
	Name        string
	Description string
	Prompt      string
	Model       string
	Background  bool
}

// Result holds the output of a completed agent.
type Result struct {
	Name    string
	Output  string
	IsError bool
	Err     error
}

// Status represents agent lifecycle state.
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusDone
	StatusFailed
	StatusCancelled
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusDone:
		return "done"
	case StatusFailed:
		return "failed"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Entry tracks a running or completed agent.
type Entry struct {
	Def    Def
	Status Status
	Result *Result
	Cancel context.CancelFunc
	// Inbox receives messages sent to this agent via SendMessage.
	Inbox chan string
}

// NewEntry creates a new agent entry with a message inbox.
func NewEntry(def Def, cancel context.CancelFunc) *Entry {
	return &Entry{
		Def:    def,
		Status: StatusRunning,
		Cancel: cancel,
		Inbox:  make(chan string, 16), // buffered to avoid blocking senders
	}
}

// Registry tracks named agents with thread-safe access.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]*Entry
}

// NewRegistry creates a new agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]*Entry),
	}
}

// Register adds an agent entry.
func (r *Registry) Register(name string, entry *Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[name] = entry
}

// Get returns an agent entry by name.
func (r *Registry) Get(name string) *Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[name]
}

// All returns a snapshot of all registered agents.
func (r *Registry) All() map[string]*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*Entry, len(r.agents))
	for k, v := range r.agents {
		out[k] = v
	}
	return out
}

// Remove deletes an agent entry.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, name)
}

// Count returns the number of registered agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// Running returns all agents currently in StatusRunning.
func (r *Registry) Running() []*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var running []*Entry
	for _, e := range r.agents {
		if e.Status == StatusRunning {
			running = append(running, e)
		}
	}
	return running
}

// WaitAll blocks until all registered agents are done or the context is cancelled.
func (r *Registry) WaitAll(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		running := r.Running()
		if len(running) == 0 {
			return
		}

		// Simple polling - in production you'd use a condition variable
		select {
		case <-ctx.Done():
			return
		default:
			// Check again on next iteration
		}
	}
}
