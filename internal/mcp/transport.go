package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// StdioTransport communicates with an MCP server over stdio.
type StdioTransport struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  atomic.Int64
	pending map[int]chan *Response
	pendMu  sync.Mutex
	done    chan struct{}
}

// NewStdioTransport creates a transport that spawns a child process.
func NewStdioTransport(ctx context.Context, command string, args ...string) (*StdioTransport, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting MCP server: %w", err)
	}

	t := &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		pending: make(map[int]chan *Response),
		done:    make(chan struct{}),
	}

	go t.readLoop()

	return t, nil
}

// Send sends a JSON-RPC request and waits for the response.
func (t *StdioTransport) Send(method string, params any) (*Response, error) {
	id := int(t.nextID.Add(1))

	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshaling params: %w", err)
		}
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}

	ch := make(chan *Response, 1)
	t.pendMu.Lock()
	t.pending[id] = ch
	t.pendMu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	t.mu.Lock()
	_, err = fmt.Fprintf(t.stdin, "%s\n", data)
	t.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("writing request: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-t.done:
		return nil, fmt.Errorf("transport closed")
	}
}

// Notify sends a JSON-RPC notification (no response expected).
func (t *StdioTransport) Notify(method string, params any) error {
	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshaling params: %w", err)
		}
	}

	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshaling notification: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	_, err = fmt.Fprintf(t.stdin, "%s\n", data)
	return err
}

// Close shuts down the transport.
func (t *StdioTransport) Close() error {
	close(t.done)
	t.stdin.Close()
	return t.cmd.Wait()
}

func (t *StdioTransport) readLoop() {
	for {
		line, err := t.stdout.ReadBytes('\n')
		if err != nil {
			return
		}

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		t.pendMu.Lock()
		ch, ok := t.pending[resp.ID]
		if ok {
			delete(t.pending, resp.ID)
		}
		t.pendMu.Unlock()

		if ok {
			ch <- &resp
		}
	}
}
