package webfetch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebFetch_PlainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	tool := New()
	input, _ := json.Marshal(map[string]any{"url": srv.URL})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result.Content)
}

func TestWebFetch_HTMLStripping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><h1>Title</h1><p>Content</p><script>alert('x')</script></body></html>"))
	}))
	defer srv.Close()

	tool := New()
	input, _ := json.Marshal(map[string]any{"url": srv.URL})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Title")
	assert.Contains(t, result.Content, "Content")
	assert.NotContains(t, result.Content, "<h1>")
	assert.NotContains(t, result.Content, "alert")
}

func TestWebFetch_InvalidURL(t *testing.T) {
	tool := New()

	input, _ := json.Marshal(map[string]any{"url": "not-a-url"})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestWebFetch_EmptyURL(t *testing.T) {
	tool := New()

	input, _ := json.Marshal(map[string]any{"url": ""})
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "required")
}

func TestStripHTML(t *testing.T) {
	html := `<html><head><style>body{color:red}</style></head><body><p>Hello &amp; World</p></body></html>`
	result := stripHTML(html)
	assert.Contains(t, result, "Hello & World")
	assert.NotContains(t, result, "<p>")
	assert.NotContains(t, result, "color:red")
}
