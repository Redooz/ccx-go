package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	entry := NewEntry(Def{Name: "test-agent"}, nil)
	r.Register("test-agent", entry)
	assert.NotNil(t, r.Get("test-agent"))
	assert.Nil(t, r.Get("nonexistent"))

	all := r.All()
	assert.Len(t, all, 1)

	assert.Equal(t, 1, r.Count())

	r.Remove("test-agent")
	assert.Nil(t, r.Get("test-agent"))
	assert.Equal(t, 0, r.Count())
}

func TestRegistry_Running(t *testing.T) {
	r := NewRegistry()

	r.Register("a", NewEntry(Def{Name: "a"}, nil))
	r.Register("b", &Entry{Def: Def{Name: "b"}, Status: StatusDone})
	r.Register("c", NewEntry(Def{Name: "c"}, nil))

	running := r.Running()
	assert.Len(t, running, 2)
}

func TestSpawnAgent_Success(t *testing.T) {
	registry := NewRegistry()
	def := Def{Name: "worker", Description: "test worker"}

	ch := SpawnAgent(context.Background(), def, registry, func(ctx context.Context, d Def) (string, error) {
		return "done", nil
	})

	result := <-ch
	assert.Equal(t, "worker", result.Name)
	assert.Equal(t, "done", result.Output)
	assert.False(t, result.IsError)

	entry := registry.Get("worker")
	require.NotNil(t, entry)
	assert.Equal(t, StatusDone, entry.Status)
}

func TestSpawnAgent_Error(t *testing.T) {
	registry := NewRegistry()
	def := Def{Name: "failing"}

	ch := SpawnAgent(context.Background(), def, registry, func(ctx context.Context, d Def) (string, error) {
		return "", fmt.Errorf("something went wrong")
	})

	result := <-ch
	assert.True(t, result.IsError)
	assert.Contains(t, result.Output, "something went wrong")

	entry := registry.Get("failing")
	require.NotNil(t, entry)
	assert.Equal(t, StatusFailed, entry.Status)
}

func TestSpawnAgent_Cancellation(t *testing.T) {
	registry := NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())

	def := Def{Name: "cancellable"}

	ch := SpawnAgent(ctx, def, registry, func(ctx context.Context, d Def) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})

	time.Sleep(10 * time.Millisecond)
	cancel()

	result := <-ch
	assert.True(t, result.IsError)
}

func TestSpawnMultiple(t *testing.T) {
	registry := NewRegistry()
	defs := []Def{
		{Name: "agent-1"},
		{Name: "agent-2"},
		{Name: "agent-3"},
	}

	channels := SpawnMultiple(context.Background(), defs, registry, func(ctx context.Context, d Def) (string, error) {
		return "result-" + d.Name, nil
	})

	assert.Len(t, channels, 3)

	results := CollectResults(context.Background(), channels)
	assert.Len(t, results, 3)
	for _, r := range results {
		assert.False(t, r.IsError)
		assert.Contains(t, r.Output, "result-")
	}
}

func TestSendMessage_Success(t *testing.T) {
	registry := NewRegistry()
	entry := NewEntry(Def{Name: "receiver"}, nil)
	registry.Register("receiver", entry)

	err := SendMessage(registry, "receiver", "hello")
	require.NoError(t, err)

	// Message should be in the inbox
	select {
	case msg := <-entry.Inbox:
		assert.Equal(t, "hello", msg)
	default:
		t.Fatal("expected message in inbox")
	}
}

func TestSendMessage_NotFound(t *testing.T) {
	registry := NewRegistry()
	err := SendMessage(registry, "nonexistent", "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSendMessage_NotRunning(t *testing.T) {
	registry := NewRegistry()
	registry.Register("done-agent", &Entry{
		Def:    Def{Name: "done-agent"},
		Status: StatusDone,
		Inbox:  make(chan string, 1),
	})

	err := SendMessage(registry, "done-agent", "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestCancelAgent(t *testing.T) {
	registry := NewRegistry()
	_, cancel := context.WithCancel(context.Background())
	entry := NewEntry(Def{Name: "cancellable"}, cancel)
	registry.Register("cancellable", entry)

	err := CancelAgent(registry, "cancellable")
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, entry.Status)
}

func TestCancelAgent_NotFound(t *testing.T) {
	registry := NewRegistry()
	err := CancelAgent(registry, "nonexistent")
	assert.Error(t, err)
}

func TestStatus_String(t *testing.T) {
	assert.Equal(t, "pending", StatusPending.String())
	assert.Equal(t, "running", StatusRunning.String())
	assert.Equal(t, "done", StatusDone.String())
	assert.Equal(t, "failed", StatusFailed.String())
	assert.Equal(t, "cancelled", StatusCancelled.String())
}

func TestNewEntry(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	entry := NewEntry(Def{Name: "test", Description: "desc"}, cancel)

	assert.Equal(t, StatusRunning, entry.Status)
	assert.Equal(t, "test", entry.Def.Name)
	assert.NotNil(t, entry.Inbox)
	assert.NotNil(t, entry.Cancel)
}
