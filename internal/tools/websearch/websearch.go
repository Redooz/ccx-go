package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

type input struct {
	Query string `json:"query"`
}

// Tool implements web search via Brave Search API with HTML scrape fallback.
type Tool struct {
	client   *http.Client
	braveKey string // optional Brave Search API key
}

// New creates a new WebSearch tool.
func New() *Tool {
	return &Tool{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// NewWithBraveKey creates a WebSearch tool with a Brave Search API key.
func NewWithBraveKey(key string) *Tool {
	t := New()
	t.braveKey = key
	return t
}

func (t *Tool) Name() string        { return "WebSearch" }
func (t *Tool) Description() string { return "Search the web and return results." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "The search query"}
		},
		"required": ["query"]
	}`)
}

func (t *Tool) Execute(ctx context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	if in.Query == "" {
		return &tool.Result{Content: "query is required", IsError: true}, nil
	}

	// Try Brave Search API first if key is available
	if t.braveKey != "" {
		result, err := t.searchBrave(ctx, in.Query)
		if err == nil {
			return &tool.Result{Content: result}, nil
		}
		// Fall through to scrape fallback
	}

	// Fallback: scrape DuckDuckGo HTML (no API key needed, more reliable than Google scraping)
	result, err := t.searchDDG(ctx, in.Query)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("search failed: %v", err), IsError: true}, nil
	}

	return &tool.Result{Content: result}, nil
}

// searchBrave queries the Brave Search API.
func (t *Tool) searchBrave(ctx context.Context, query string) (string, error) {
	u := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=10", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", t.braveKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Brave API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}

	var result braveResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return formatBraveResults(query, result), nil
}

type braveResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

func formatBraveResults(query string, resp braveResponse) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))

	for i, r := range resp.Web.Results {
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Description))
	}

	if len(resp.Web.Results) == 0 {
		sb.WriteString("No results found.")
	}

	return sb.String()
}

// searchDDG scrapes DuckDuckGo HTML lite for search results.
func (t *Tool) searchDDG(ctx context.Context, query string) (string, error) {
	u := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ccx-go/1.0)")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DuckDuckGo returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}

	return parseDDGResults(query, string(body)), nil
}

var (
	reDDGLink  = regexp.MustCompile(`(?is)<a[^>]*class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	reDDGSnip  = regexp.MustCompile(`(?is)<a[^>]*class="result__snippet"[^>]*>(.*?)</a>`)
	reHTMLTags = regexp.MustCompile(`<[^>]+>`)
)

func parseDDGResults(query, html string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))

	links := reDDGLink.FindAllStringSubmatch(html, 10)
	snippets := reDDGSnip.FindAllStringSubmatch(html, 10)

	if len(links) == 0 {
		sb.WriteString("No results found.")
		return sb.String()
	}

	for i, link := range links {
		title := reHTMLTags.ReplaceAllString(link[2], "")
		href := link[1]
		snippet := ""
		if i < len(snippets) {
			snippet = reHTMLTags.ReplaceAllString(snippets[i][1], "")
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n\n", i+1, strings.TrimSpace(title), href, strings.TrimSpace(snippet)))
	}

	return sb.String()
}
