package tool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTool implements the Tool interface for testing.
type fakeTool struct {
	name        string
	description string
	schema      json.RawMessage
	execFn      func(ctx context.Context, input json.RawMessage) (*Result, error)
}

func (f *fakeTool) Name() string                { return f.name }
func (f *fakeTool) Description() string          { return f.description }
func (f *fakeTool) InputSchema() json.RawMessage { return f.schema }
func (f *fakeTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	if f.execFn != nil {
		return f.execFn(ctx, input)
	}
	return &Result{Content: "ok"}, nil
}

func newFakeTool(name string) *fakeTool {
	return &fakeTool{
		name:        name,
		description: name + " tool",
		schema:      json.RawMessage(`{"type":"object","properties":{}}`),
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	err := r.Register(newFakeTool("Read"))
	require.NoError(t, err)

	found := r.FindByName("Read")
	require.NotNil(t, found)
	assert.Equal(t, "Read", found.Name())
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newFakeTool("Read")))

	err := r.Register(newFakeTool("Read"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_FindByName_NotFound(t *testing.T) {
	r := NewRegistry()
	assert.Nil(t, r.FindByName("NonExistent"))
}

func TestRegistry_All(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeTool("Read"))
	r.Register(newFakeTool("Write"))
	r.Register(newFakeTool("Edit"))

	all := r.All()
	assert.Len(t, all, 3)

	names := make(map[string]bool)
	for _, t := range all {
		names[t.Name()] = true
	}
	assert.True(t, names["Read"])
	assert.True(t, names["Write"])
	assert.True(t, names["Edit"])
}

func TestRegistry_APIDefinitions(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeTool("Read"))
	r.Register(newFakeTool("Bash"))

	defs := r.APIDefinitions()
	assert.Len(t, defs, 2)

	for _, d := range defs {
		assert.NotEmpty(t, d.Name)
		assert.NotEmpty(t, d.Description)
		assert.NotEmpty(t, d.InputSchema)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	// Register tools concurrently
	done := make(chan bool, 10)
	for i := range 10 {
		go func(i int) {
			name := "tool_" + string(rune('A'+i))
			r.Register(newFakeTool(name))
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}

	// All should be retrievable
	all := r.All()
	assert.Len(t, all, 10)
}

func TestTool_Execute(t *testing.T) {
	tool := &fakeTool{
		name:        "Echo",
		description: "echoes input",
		schema:      json.RawMessage(`{"type":"object"}`),
		execFn: func(ctx context.Context, input json.RawMessage) (*Result, error) {
			return &Result{Content: string(input)}, nil
		},
	}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"msg":"hello"}`))
	require.NoError(t, err)
	assert.Equal(t, `{"msg":"hello"}`, result.Content)
	assert.False(t, result.IsError)
}
