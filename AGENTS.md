# PROJECT KNOWLEDGE BASE

**Updated:** 2026-09-15
**Branch:** feat/eino-agent

## OVERVIEW

Terminal **Eino ADK agent** chat client. Go 1.25+, tview TUI, SQLite storage. Completions run through CloudWeGo Eino `ChatModelAgent` + `Runner` + `AgentEvent` streaming. Multiple official `eino-ext` ChatModel providers (OpenAI-compatible, Ark, Ollama, Claude, Gemini, Qwen, DeepSeek).

## STRUCTURE

```
chat-tui/
├── cmd/chat-tui/main.go     # Entry point: config → storage → UI
├── internal/
│   ├── api/
│   │   ├── agent.go         # ChatModelAgent + Runner + AgentEvent stream
│   │   ├── provider.go      # Multi-provider ChatModel factory
│   │   ├── convert.go       # go-openai messages ↔ eino schema.Message
│   │   └── tools.go         # Common tools: time/cwd/list/read/write/http_get/run_command
│   ├── config/config.go     # JSON config: ~/.xftui.json
│   ├── storage/sqlite.go    # Pure-Go SQLite: ~/.xftui.db
│   ├── types/types.go       # Config (provider + credentials), Conversation, SystemPrompt
│   └── ui/
│       └── tview_ui.go      # tview UI: settings, AgentEvent consumer, cancel
├── go.mod
├── .github/workflows/release.yml  # Cross-platform release (v* tags)
└── todo.md                  # Feature backlog
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Agent pipeline | `internal/api/agent.go` | `NewChatModelAgent` + `Runner{EnableStreaming:true}` |
| Provider factory | `internal/api/provider.go` | openai / ark / ollama / claude / gemini / qwen / deepseek |
| Message conversion | `internal/api/convert.go` | UI/SQLite keep go-openai types |
| Tools | `internal/api/tools.go` | `extraTools()` → time/cwd/fs/http/shell |
| Config schema | `internal/types/types.go` | `provider`, `region`, `access_key`, `secret_key` |
| Storage operations | `internal/storage/sqlite.go` | CRUD + migrations |
| UI logic | `internal/ui/tview_ui.go` | Settings, stream consumer, Stop/Esc cancel |
| Themes | `internal/ui/tview_ui.go` | `themes` map: night/nord/gruvbox/solarized-dark/light |

## CONVENTIONS

- **Config path hardcoded**: `~/.xftui.json`, `~/.xftui.db`
- **Missing `provider` defaults to `openai`** — existing `base_url` / `api_key` / `model` still work
- **go-openai types in SQLite/UI** — convert at the `internal/api` boundary
- **UUID for IDs** — conversations and system prompts use `google/uuid`
- **Inline migrations** — `ALTER TABLE ADD COLUMN IF NOT EXISTS` pattern in `NewManager()`
- **Add tests** when touching storage, config, or api conversion/factory

## ANTI-PATTERNS (THIS PROJECT)

- **Ignored errors** — some `config.SaveConfig()` / storage writes still logged only in UI
- **Hardcoded defaults** — API key placeholder `"YOUR_API_KEY_HERE"` in main.go
- **Do not commit `main.exe` or other local binaries**

## COMMANDS

```bash
go build -o chat-tui ./cmd/chat-tui    # Build binary
go build ./...                         # All packages
go test ./internal/config ./internal/api
go install github.com/evallife/chat-tui/cmd/chat-tui@latest  # Install globally
```

## NOTES

- Release workflow triggers on `v*` tags, builds for linux/windows/darwin (amd64)
- System prompts have a manager UI (list / new / edit / delete / apply)
- ChatModelAgent ships with common tools in `extraTools()` (time, cwd, list/read/write file, http_get, run_command); append more `tool.BaseTool` there as needed
- GitHub Pages marketing site lives on the `gh-pages` branch (not this tree)
