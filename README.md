# ccx-go

A Go implementation of an AI coding assistant CLI, inspired by Claude Code's architecture. Zero-dependency single binary with goroutine-based multi-agent orchestration.

## Backstory

On March 31, 2026, security researcher Chaofan Shou ([@Fried_rice](https://x.com/Fried_rice/status/2038894956459290963)) discovered that Anthropic's npm package `@anthropic-ai/claude-code` shipped with a 57MB source map file (`cli.js.map`) that exposed the full, unobfuscated TypeScript source -- 512,000 lines across 1,900+ files.

The architecture analysis revealed a sophisticated system: 43 built-in tools, 4-layer context compression, multi-agent orchestration, MCP protocol integration, a permission DSL, and hidden features like BUDDY AI pet, KAIROS daemon mode, and Auto-Dream memory consolidation. The full breakdown is available at [verified-skill.com/insights/claude-code](https://verified-skill.com/insights/claude-code).

Rather than just archiving the leak, the community began building clean-room implementations in multiple languages. Go was a natural first choice -- the language has a proven track record of rewriting developer tools into fast, single-binary alternatives (gh, lazygit, k9s, fzf). ccx-go follows that tradition: a full working implementation with real tool execution, TUI, and agent system -- not just a metadata wrapper.

For comparison, [instructkr/claw-code](https://github.com/instructkr/claw-code) (41.7k stars) took a Python metadata/harness approach. CCX takes a different path with full language-native implementations and real test suites.

- Original tweet: https://x.com/Fried_rice/status/2038894956459290963
- Architecture analysis: https://verified-skill.com/insights/claude-code
- CCX umbrella: https://github.com/anton-abyzov/ccx

## Why Go?

- **Single static binary** (~20-30MB) -- no runtime dependencies
- **Goroutines** -- perfect for parallel agent spawning, natural fit for multi-agent orchestration
- **Bubbletea/Charm** -- the best TUI ecosystem across any language
- **Fast compilation** -- full build in 5-15 seconds, cross-compile to all platforms
- **Proven ecosystem** -- follows the path of gh, lazygit, k9s, fzf

## Architecture

Based on analysis of Claude Code's 512K-line TypeScript architecture:

- **Tool System**: Pluggable tools with permission gating and concurrent execution
- **Agent Spawning**: Goroutine-per-agent with context cancellation and channel-based communication
- **TUI**: Bubbletea (Elm Architecture) with Lipgloss styling and Glamour markdown rendering
- **Context Management**: Multi-layer compression (micro, auto, session, full)
- **MCP Protocol**: Model Context Protocol client for tool/resource discovery
- **Permission System**: Rule-based DSL with interactive prompts
- **Streaming API**: SSE-based streaming from Claude API with tool_use handling

## Tech Stack

| Component | Library |
|-----------|---------|
| TUI Framework | bubbletea + lipgloss + bubbles |
| Markdown | glamour |
| Syntax Highlighting | chroma |
| CLI Parsing | cobra + viper |
| HTTP Client | net/http (stdlib) |
| Schema Validation | go-jsonschema |
| Config | viper (TOML/JSON/YAML) |
| Testing | testify + teatest |

## Project Structure

```
cmd/
  claude/          # Main CLI entry point
internal/
  agent/           # Agent spawning and lifecycle
  api/             # Claude API client (streaming, tool_use)
  compact/         # Context compression (4 layers)
  config/          # Settings cascade, CLAUDE.md parsing
  mcp/             # MCP protocol client
  permission/      # Permission DSL, rules, classifier
  query/           # Main query loop
  skill/           # Skill loading and execution
  tool/            # Tool interface and registry
  tools/           # Built-in tool implementations
    bash/
    fileread/
    fileedit/
    filewrite/
    glob/
    grep/
    agent/
    webfetch/
  tui/             # Bubbletea UI components
    chat/
    prompt/
    permission/
    progress/
  memory/          # Memory system (user, project, feedback, reference)
pkg/
  types/           # Shared types
  schema/          # JSON schema utilities
```

## Getting Started

```sh
go install github.com/anton-abyzov/ccx-go/cmd/claude@latest
```

## Development

```sh
git clone https://github.com/anton-abyzov/ccx-go.git
cd ccx-go
go build ./cmd/claude
go test ./...
```

## License

MIT
