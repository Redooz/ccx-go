package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	defaultVersion   = "2023-06-01"
	defaultMaxTokens = 16384
	maxRetries       = 5
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
		httpClient: &http.Client{Timeout: 300 * time.Second},
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func (c *Client) WithBaseURL(url string) *Client {
	c.baseURL = url
	return c
}

// CreateMessage sends a non-streaming message request with automatic retries.
func (c *Client) CreateMessage(ctx context.Context, req *Request) (*Response, error) {
	req.Stream = false
	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt, nil)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		c.setHeaders(httpReq)

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("sending request: %w", err)
			continue
		}

		if httpResp.StatusCode == http.StatusOK {
			defer httpResp.Body.Close()
			var resp Response
			if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
				return nil, fmt.Errorf("decoding response: %w", err)
			}
			return &resp, nil
		}

		respBody, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()

		apiErr := parseAPIError(httpResp.StatusCode, respBody)

		// Retry on rate limit (429) and server errors (5xx)
		if isRetryable(httpResp.StatusCode) {
			lastErr = apiErr
			continue
		}

		return nil, apiErr
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// CreateMessageStream sends a streaming message request with automatic retries.
func (c *Client) CreateMessageStream(ctx context.Context, req *Request) (*StreamReader, error) {
	req.Stream = true
	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt, nil)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		c.setHeaders(httpReq)

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("sending request: %w", err)
			continue
		}

		if httpResp.StatusCode == http.StatusOK {
			return NewStreamReader(httpResp.Body), nil
		}

		respBody, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()

		apiErr := parseAPIError(httpResp.StatusCode, respBody)

		if isRetryable(httpResp.StatusCode) {
			lastErr = apiErr
			continue
		}

		return nil, apiErr
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Anthropic-Version", c.version)
}

// APIError represents a structured error from the Anthropic API.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.Type, e.Message)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// IsOverloaded returns true if this is a 529 overloaded error.
func (e *APIError) IsOverloaded() bool { return e.StatusCode == 529 }

// IsRateLimit returns true if this is a 429 rate limit error.
func (e *APIError) IsRateLimit() bool { return e.StatusCode == 429 }

func parseAPIError(status int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status}

	var parsed struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Message != "" {
		apiErr.Type = parsed.Error.Type
		apiErr.Message = parsed.Error.Message
	} else {
		apiErr.Message = strings.TrimSpace(string(body))
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(status)
		}
	}

	return apiErr
}

func isRetryable(statusCode int) bool {
	return statusCode == 429 || statusCode == 529 || statusCode >= 500
}

// retryDelay computes exponential backoff delay.
// Respects Retry-After header if provided.
func retryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				return time.Duration(secs) * time.Second
			}
		}
	}
	// Exponential backoff: 1s, 2s, 4s, 8s, 16s (capped)
	base := math.Pow(2, float64(attempt-1))
	delay := time.Duration(base) * time.Second
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}
