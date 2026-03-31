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

	entry := &Entry{
		Def:    Def{Name: "test-agent"},
		Status: StatusRunning,
	}

	r.Register("test-agent", entry)
	assert.NotNil(t, r.Get("test-agent"))
	assert.Nil(t, r.Get("nonexistent"))

	all := r.All()
	assert.Len(t, all, 1)

	r.Remove("test-agent")
	assert.Nil(t, r.Get("test-agent"))
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

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)
	cancel()

	result := <-ch
	assert.True(t, result.IsError)
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
	})

	err := SendMessage(registry, "done-agent", "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestStatus_String(t *testing.T) {
	assert.Equal(t, "pending", StatusPending.String())
	assert.Equal(t, "running", StatusRunning.String())
	assert.Equal(t, "done", StatusDone.String())
	assert.Equal(t, "failed", StatusFailed.String())
	assert.Equal(t, "cancelled", StatusCancelled.String())
}
