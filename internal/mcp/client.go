package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Client communicates with an MCP server.
type Client struct {
	transport *StdioTransport
	tools     []ToolDef
}

// NewClient creates an MCP client connecting to the given server command.
func NewClient(ctx context.Context, command string, args ...string) (*Client, error) {
	transport, err := NewStdioTransport(ctx, command, args...)
	if err != nil {
		return nil, fmt.Errorf("creating transport: %w", err)
	}

	c := &Client{transport: transport}

	// Initialize the connection
	if err := c.initialize(); err != nil {
		transport.Close()
		return nil, fmt.Errorf("initializing: %w", err)
	}

	return c, nil
}

func (c *Client) initialize() error {
	params := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "ccx-go",
			"version": "1.0.0",
		},
	}

	resp, err := c.transport.Send("initialize", params)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}

	// Send initialized notification
	return c.transport.Notify("notifications/initialized", nil)
}

// ListTools returns the tools available from the MCP server.
func (c *Client) ListTools() ([]ToolDef, error) {
	resp, err := c.transport.Send("tools/list", nil)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	var result struct {
		Tools []ToolDef `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling tools: %w", err)
	}

	c.tools = result.Tools
	return result.Tools, nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(name string, arguments json.RawMessage) (*ToolCallResult, error) {
	params := map[string]any{
		"name":      name,
		"arguments": json.RawMessage(arguments),
	}

	resp, err := c.transport.Send("tools/call", params)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling tool result: %w", err)
	}

	return &result, nil
}

// ListResources returns available resources from the MCP server.
func (c *Client) ListResources() ([]Resource, error) {
	resp, err := c.transport.Send("resources/list", nil)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	var result struct {
		Resources []Resource `json:"resources"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling resources: %w", err)
	}

	return result.Resources, nil
}

// Close shuts down the MCP client and transport.
func (c *Client) Close() error {
	return c.transport.Close()
}
