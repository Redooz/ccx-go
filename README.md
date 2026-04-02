# ccx-go

AI coding assistant CLI in Go, part of the [CCX (Community Code Extended)](https://github.com/anton-abyzov/ccx) project. 9.3MB binary, 11 tools, full TUI. Built on Bubbletea/Charm with goroutine-based multi-agent orchestration.

## Why CCX?

CCX (Community Code Extended) is a custom AI coding assistant built from the ground up using publicly documented API specifications and common patterns in AI-assisted development. Each implementation is its own application with independent architecture decisions and language-idiomatic designs.

Go was a natural first choice -- the language has a proven track record of rewriting developer tools into fast, single-binary alternatives (gh, lazygit, k9s, fzf). ccx-go follows that tradition: a full working implementation with real tool execution, TUI, and agent system -- not just a metadata wrapper.

Unlike [instructkr/claw-code](https://github.com/instructkr/claw-code) (41.7k stars), which catalogs tool inventories as structured data, ccx-go is a ground-up Go implementation with real tool execution, goroutine-based agents, and a comprehensive test suite.

- Architecture analysis: https://verified-skill.com/insights/claude-code
- CCX umbrella: https://github.com/anton-abyzov/ccx

## Quick Start

```bash
git clone https://github.com/anton-abyzov/ccx-go.git
cd ccx-go
go build -o ccx-go ./cmd/claude
export ANTHROPIC_API_KEY="your-key-here"
./ccx-go
```

## Features

- **11 built-in tools** -- Bash, FileRead, FileEdit, FileWrite, Glob, Grep, Agent, WebFetch, NotebookEdit, TodoRead, TodoWrite
- **Slash commands with autocomplete** -- `/help`, `/compact`, `/clear`, `/model`, `/memory`, `/skills`
- **Streaming responses** -- SSE-based streaming from Claude API with real-time tool_use handling
- **Markdown rendering** -- Glamour-powered rich markdown in the terminal
- **Skill discovery** -- loads and executes markdown-based skills
- **Multi-agent orchestration** -- goroutine-per-agent with context cancellation and channel-based IPC
- **4-layer context compression** -- micro, auto, session, and full compaction
- **MCP protocol** -- Model Context Protocol client for tool/resource discovery
- **Permission system** -- rule-based DSL with interactive prompts
- **Memory persistence** -- user, project, feedback, and reference memory types

## Slash Commands

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/compact` | Compress conversation context |
| `/clear` | Clear conversation history |
| `/model` | Switch Claude model |
| `/memory` | View/manage memory entries |
| `/skills` | List available skills |
| `/config` | Show current configuration |

## Why Go?

- **9.3MB static binary** -- no runtime dependencies
- **Goroutines** -- perfect for parallel agent spawning, natural fit for multi-agent orchestration
- **Bubbletea/Charm** -- the best TUI ecosystem across any language
- **Fast compilation** -- full build in 5-15 seconds, cross-compile to all platforms
- **Proven ecosystem** -- follows the path of gh, lazygit, k9s, fzf

## Architecture

Based on publicly documented patterns in AI coding assistant architecture:

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

```bash
git clone https://github.com/anton-abyzov/ccx-go.git
cd ccx-go
go build ./cmd/claude
go test ./...
```

## License

MIT
