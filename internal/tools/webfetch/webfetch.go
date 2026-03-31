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

// Tool implements HTTP GET with HTML-to-markdown conversion.
type Tool struct {
	client *http.Client
}

func New() *Tool {
	return &Tool{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects (>10)")
				}
				return nil
			},
		},
	}
}

func (t *Tool) Name() string        { return "WebFetch" }
func (t *Tool) Description() string { return "Fetch a URL and return its content as text." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {"type": "string", "description": "URL to fetch"},
			"timeout": {"type": "integer", "description": "Timeout in milliseconds (default 30000)"}
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ccx-go/1.0; +https://github.com/anton-abyzov/ccx-go)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.5")

	resp, err := t.client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &tool.Result{Content: fmt.Sprintf("request timed out after %s", timeout), IsError: true}, nil
		}
		return &tool.Result{Content: fmt.Sprintf("error fetching URL: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &tool.Result{
			Content: fmt.Sprintf("HTTP %d %s", resp.StatusCode, resp.Status),
			IsError: true,
		}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error reading response: %v", err), IsError: true}, nil
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	// Convert HTML to readable text
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		content = htmlToMarkdown(content)
	}

	// Truncate to reasonable size
	if len(content) > 50000 {
		content = content[:50000] + "\n... (truncated)"
	}

	// Add metadata header
	header := fmt.Sprintf("URL: %s\nStatus: %d\nContent-Type: %s\n---\n", resp.Request.URL.String(), resp.StatusCode, contentType)

	return &tool.Result{Content: header + content}, nil
}

// htmlToMarkdown converts HTML to a readable markdown-like format.
func htmlToMarkdown(s string) string {
	// Extract title
	title := ""
	if m := reTitle.FindStringSubmatch(s); len(m) > 1 {
		title = strings.TrimSpace(m[1])
	}

	// Remove script, style, nav, footer, header elements
	s = reScript.ReplaceAllString(s, "")
	s = reStyle.ReplaceAllString(s, "")
	s = reNav.ReplaceAllString(s, "")
	s = reFooter.ReplaceAllString(s, "")

	// Convert headings
	s = reH1.ReplaceAllString(s, "\n# $1\n")
	s = reH2.ReplaceAllString(s, "\n## $1\n")
	s = reH3.ReplaceAllString(s, "\n### $1\n")
	s = reH4.ReplaceAllString(s, "\n#### $1\n")

	// Convert links
	s = reLink.ReplaceAllString(s, "[$2]($1)")

	// Convert emphasis
	s = reBold.ReplaceAllString(s, "**$1**")
	s = reItalic.ReplaceAllString(s, "*$1*")

	// Convert code blocks
	s = rePre.ReplaceAllString(s, "\n```\n$1\n```\n")
	s = reCode.ReplaceAllString(s, "`$1`")

	// Convert lists
	s = reLi.ReplaceAllString(s, "\n- $1")

	// Convert paragraphs and line breaks
	s = rePara.ReplaceAllString(s, "\n\n$1\n\n")
	s = reBr.ReplaceAllString(s, "\n")

	// Remove all remaining tags
	s = reTag.ReplaceAllString(s, "")

	// Decode HTML entities
	s = decodeEntities(s)

	// Clean up whitespace
	s = reSpaces.ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)

	if title != "" {
		s = "# " + title + "\n\n" + s
	}

	return s
}

func decodeEntities(s string) string {
	replacements := []struct{ old, new string }{
		{"&nbsp;", " "}, {"&amp;", "&"}, {"&lt;", "<"}, {"&gt;", ">"},
		{"&quot;", "\""}, {"&#39;", "'"}, {"&apos;", "'"},
		{"&mdash;", "—"}, {"&ndash;", "–"}, {"&laquo;", "«"}, {"&raquo;", "»"},
		{"&bull;", "•"}, {"&middot;", "·"}, {"&hellip;", "..."},
		{"&copy;", "©"}, {"&reg;", "®"}, {"&trade;", "™"},
	}
	for _, r := range replacements {
		s = strings.ReplaceAll(s, r.old, r.new)
	}
	// Numeric entities
	s = reNumEntity.ReplaceAllStringFunc(s, func(m string) string {
		var n int
		if _, err := fmt.Sscanf(m, "&#%d;", &n); err == nil && n > 0 && n < 0x10FFFF {
			return string(rune(n))
		}
		return m
	})
	return s
}

var (
	reTitle     = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reScript    = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle     = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reNav       = regexp.MustCompile(`(?is)<nav[^>]*>.*?</nav>`)
	reFooter    = regexp.MustCompile(`(?is)<footer[^>]*>.*?</footer>`)
	reH1        = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	reH2        = regexp.MustCompile(`(?is)<h2[^>]*>(.*?)</h2>`)
	reH3        = regexp.MustCompile(`(?is)<h3[^>]*>(.*?)</h3>`)
	reH4        = regexp.MustCompile(`(?is)<h4[^>]*>(.*?)</h4>`)
	reLink      = regexp.MustCompile(`(?is)<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	reBold      = regexp.MustCompile(`(?is)<(?:b|strong)[^>]*>(.*?)</(?:b|strong)>`)
	reItalic    = regexp.MustCompile(`(?is)<(?:i|em)[^>]*>(.*?)</(?:i|em)>`)
	rePre       = regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
	reCode      = regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`)
	reLi        = regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	rePara      = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	reBr        = regexp.MustCompile(`(?is)<br\s*/?>`)
	reTag       = regexp.MustCompile(`<[^>]+>`)
	reSpaces    = regexp.MustCompile(`\n{3,}`)
	reNumEntity = regexp.MustCompile(`&#\d+;`)
)
