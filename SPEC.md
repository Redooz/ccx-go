# ccx-go -- Implementation Spec

## Phase 1: Foundation (Week 1-3)

### P1-01: Project scaffold
- Go module init, Cobra CLI, Makefile, CI
- `cmd/claude/main.go` entry point
- `internal/` package structure

### P1-02: Claude API client
- Anthropic Messages API with SSE streaming
- `internal/api/client.go` -- HTTP client with streaming
- `internal/api/types.go` -- Message, ToolUse, ToolResult types
- `internal/api/stream.go` -- SSE event parser
- Handle: tool_use blocks, thinking, text deltas, stop_reason
- API key from env (`ANTHROPIC_API_KEY`) or config file

### P1-03: Tool interface and registry
```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() json.RawMessage
    Execute(ctx context.Context, input json.RawMessage, toolCtx *ToolContext) (*ToolResult, error)
    IsConcurrencySafe(input json.RawMessage) bool
}
```
- Tool registry with `Register()` and `FindByName()`
- Tool-to-API-schema conversion for Claude API

### P1-04: Basic query loop
- `internal/query/loop.go` -- the core agentic loop
- User message -> API call -> parse response -> extract tool_use -> execute -> loop
- Streaming output to terminal
- Ctrl+C handling

## Phase 2: Core Tools (Week 3-6)

### P2-01: Bash tool
- Command execution with timeout
- Working directory tracking
- Output capture (stdout + stderr)
- Basic safety checks (no rm -rf /, no fork bombs)

### P2-02: File tools (Read, Write, Edit)
- FileRead: line numbers, offset/limit, image support
- FileWrite: create new files, overwrite existing
- FileEdit: exact string replacement with old_string/new_string

### P2-03: Search tools (Glob, Grep)
- Glob: fast file pattern matching via doublestar library
- Grep: ripgrep wrapper (shell out to system rg)

### P2-04: Web tools (WebFetch, WebSearch)
- WebFetch: HTTP GET with HTML-to-markdown conversion
- WebSearch: Brave/Google search API integration

## Phase 3: TUI (Week 6-9)

### P3-01: Bubbletea app shell
- Main model with chat view, input area, status bar
- Lipgloss styling (dark/light theme)
- Glamour for markdown rendering in responses
- Chroma for syntax highlighting in code blocks

### P3-02: Permission prompts
- Interactive allow/deny/always-allow dialog
- Permission rule display

### P3-03: Tool execution display
- Spinner for running tools
- Collapsible tool output
- Diff view for file edits

## Phase 4: Agent System (Week 9-12)

### P4-01: Agent spawning
```go
func SpawnAgent(ctx context.Context, def AgentDef, msgs []Message) <-chan AgentResult {
    ch := make(chan AgentResult, 1)
    go func() {
        defer close(ch)
        result := runAgentLoop(ctx, def, msgs)
        ch <- result
    }()
    return ch
}
```
- Goroutine per agent with context cancellation
- Named agents addressable via SendMessage
- Background agent support

### P4-02: Permission system
- Rule-based: allow/deny with glob patterns
- Settings cascade: CLI > session > project > user > defaults
- Interactive prompts for unmatched tools

### P4-03: Config and CLAUDE.md
- Settings from `~/.claude/settings.json`
- CLAUDE.md discovery (walk up directory tree)
- Memory system (`~/.claude/memory/`, `.claude/project/memory/`)

## Phase 5: Context & MCP (Week 12-16)

### P5-01: Context compression
- MicroCompact: strip large tool results
- AutoCompact: summarize at token threshold
- Token counting/estimation

### P5-02: MCP client
- JSON-RPC over stdio transport
- Tool and resource discovery
- Tool invocation forwarding

### P5-03: Skill system
- Markdown file loading with YAML frontmatter
- Skill execution (inline or forked to agent)

## Phase 6: Polish (Week 16-20)

### P6-01: Cross-compilation and distribution
- goreleaser config for all platforms
- Homebrew formula
- npm wrapper package (optional)

### P6-02: Vim mode, keybindings, history
### P6-03: Cost tracking and display
### P6-04: Hook system (pre/post tool use)

## Key Decisions

- **No CGO**: Pure Go for easy cross-compilation
- **ripgrep**: Shell out to system rg rather than pure Go regex (performance)
- **Token counting**: tiktoken-go for accurate counts, fallback to char estimation
- **Streaming**: Custom SSE parser over net/http (no heavy dependencies)
