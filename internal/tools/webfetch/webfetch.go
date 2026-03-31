package webfetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout,omitempty"`
}

// Tool implements HTTP GET with HTML-to-text conversion.
type Tool struct {
	client *http.Client
}

func New() *Tool {
	return &Tool{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *Tool) Name() string        { return "WebFetch" }
func (t *Tool) Description() string { return "Fetch a URL and return its content as text." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {"type": "string", "description": "URL to fetch"},
			"timeout": {"type": "integer", "description": "Timeout in milliseconds"}
		},
		"required": ["url"]
	}`)
}

func (t *Tool) Execute(ctx context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	if in.URL == "" {
		return &tool.Result{Content: "url is required", IsError: true}, nil
	}

	if !strings.HasPrefix(in.URL, "http://") && !strings.HasPrefix(in.URL, "https://") {
		return &tool.Result{Content: "url must start with http:// or https://", IsError: true}, nil
	}

	timeout := 30 * time.Second
	if in.Timeout > 0 {
		timeout = time.Duration(in.Timeout) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, in.URL, nil)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error creating request: %v", err), IsError: true}, nil
	}
	req.Header.Set("User-Agent", "ccx-go/1.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error fetching URL: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error reading response: %v", err), IsError: true}, nil
	}

	content := string(body)

	// Strip HTML tags for HTML content
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		content = stripHTML(content)
	}

	if len(content) > 50000 {
		content = content[:50000] + "\n... (truncated)"
	}

	return &tool.Result{Content: content}, nil
}

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reTag    = regexp.MustCompile(`<[^>]+>`)
	reSpaces = regexp.MustCompile(`\n{3,}`)
)

func stripHTML(s string) string {
	s = reScript.ReplaceAllString(s, "")
	s = reStyle.ReplaceAllString(s, "")
	s = reTag.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = reSpaces.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
