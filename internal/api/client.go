package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	defaultVersion   = "2023-06-01"
	defaultMaxTokens = 16384
)

// Client is an HTTP client for the Anthropic Messages API.
type Client struct {
	apiKey     string
	baseURL    string
	version    string
	httpClient *http.Client
}

// NewClient creates a new API client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		version:    defaultVersion,
		httpClient: &http.Client{},
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func (c *Client) WithBaseURL(url string) *Client {
	c.baseURL = url
	return c
}

// CreateMessage sends a non-streaming message request.
func (c *Client) CreateMessage(ctx context.Context, req *Request) (*Response, error) {
	req.Stream = false
	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(httpResp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("API error (status %d)", httpResp.StatusCode)
		}
		return nil, fmt.Errorf("API error (%s): %s", apiErr.Error.Type, apiErr.Error.Message)
	}

	var resp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &resp, nil
}

// CreateMessageStream sends a streaming message request and returns a stream reader.
func (c *Client) CreateMessageStream(ctx context.Context, req *Request) (*StreamReader, error) {
	req.Stream = true
	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		var apiErr struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(httpResp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("API error (status %d)", httpResp.StatusCode)
		}
		return nil, fmt.Errorf("API error (%s): %s", apiErr.Error.Type, apiErr.Error.Message)
	}

	return NewStreamReader(httpResp.Body), nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Anthropic-Version", c.version)
}
