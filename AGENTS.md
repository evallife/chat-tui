# PROJECT KNOWLEDGE BASE

**Updated:** 2026-09-15
**Branch:** main

## OVERVIEW

Terminal **Eino ADK agent** chat client. Go 1.25+, tview TUI, SQLite storage. Completions run through CloudWeGo Eino `ChatModelAgent` + `Runner` + `AgentEvent` streaming. Multiple official `eino-ext` ChatModel providers (OpenAI-compatible, Ark, Ollama, Claude, Gemini, Qwen, DeepSeek). Mutating tools (`write_file`, `run_command`) pause for TUI Allow/Deny.

## STRUCTURE

```
chat-tui/
├── cmd/chat-tui/main.go     # Entry: LoadOrCreate config → storage → UI
├── internal/
│   ├── api/
│   │   ├── agent.go         # ChatModelAgent + Runner + AgentEvent stream
│   │   ├── provider.go      # Multi-provider ChatModel factory
│   │   ├── convert.go       # go-openai messages ↔ eino schema.Message
│   │   ├── tools.go         # time/cwd/list/read/write/http_get/run_command/glob/search/mkdir
│   │   └── tools_workspace.go # Workspace sandbox + ToolConfirmer
│   ├── paths/paths.go       # ~/.chat-tui.json / .db / workspace; legacy xftui migrate
│   ├── config/config.go     # JSON config: ~/.chat-tui.json
│   ├── storage/
│   │   ├── sqlite.go        # CRUD; Open(path) for tests
│   │   └── migrate.go       # schema_migrations versioned DDL
│   ├── types/types.go       # Config (provider + credentials), Conversation, SystemPrompt
│   └── ui/
│       ├── ui.go            # TViewUI, constructor, global keys
│       ├── theme.go         # Themes
│       ├── layout.go        # Sidebar, footer, quit
│       ├── chat.go          # Input, stream consumer, refresh
│       ├── commands.go      # Slash commands + export
│       ├── search.go        # In-chat search
│       ├── history.go       # Conversation list
│       ├── prompts.go       # System prompt manager
│       ├── settings.go      # Provider / agent settings
│       ├── copy.go          # Copy mode
│       └── confirm.go       # HITL Allow/Deny for mutating tools
├── go.mod
├── .github/workflows/release.yml  # Cross-platform release (v* tags)
└── todo.md                  # Remaining backlog
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Agent pipeline | `internal/api/agent.go` | `NewChatModelAgent` + `Runner{EnableStreaming:true}` |
| Provider factory | `internal/api/provider.go` | openai / ark / ollama / claude / gemini / qwen / deepseek |
| Message conversion | `internal/api/convert.go` | UI/SQLite keep go-openai types |
| Tools | `internal/api/tools.go` | `extraToolsFor(cfg, confirm)` |
| Workspace + HITL | `internal/api/tools_workspace.go` | `resolveInWorkspace`, `ToolConfirmer` |
| Config / DB / workspace paths | `internal/paths/paths.go` | `~/.chat-tui.json`, `~/.chat-tui.db`, `~/chat-tui-workspace` |
| Config schema | `internal/types/types.go` | `provider`, `region`, `access_key`, `secret_key` |
| Storage operations | `internal/storage/sqlite.go` | CRUD |
| Schema migrations | `internal/storage/migrate.go` | `schema_migrations` table, current v2 |
| UI shell | `internal/ui/ui.go` | Constructor, keys, confirmer wiring |
| Stream + cancel | `internal/ui/chat.go` | AgentEvent consumer, Esc/Stop |
| Themes | `internal/ui/theme.go` | night/nord/gruvbox/solarized-dark/light |

## CONVENTIONS

- **Config path**: `~/.chat-tui.json`, `~/.chat-tui.db`; default workspace `~/chat-tui-workspace`
- **Legacy migrate**: `~/.xftui.json` / `~/.xftui.db` copied to the new names on first launch, then removed
- **Missing `provider` defaults to `openai`** — existing `base_url` / `api_key` / `model` still work
- **First run** writes default config (empty `api_key`) and enters the TUI
- **go-openai types in SQLite/UI** — convert at the `internal/api` boundary
- **UUID for IDs** — conversations and system prompts use `google/uuid`
- **Versioned migrations** — `schema_migrations`; add a new `migrateVN` and bump `currentSchemaVersion`
- **Add tests** when touching storage, config, or api conversion/factory/tools
- **Nil ToolConfirmer allows** mutating tools (tests); TUI always sets a confirmer

## ANTI-PATTERNS (THIS PROJECT)

- **Do not commit `chat-tui`, `main.exe`, or other local binaries** — they are gitignored
- **Do not swallow storage/config errors in the UI** — use `reportError` / `appendSystemMsg`

## COMMANDS

```bash
go build -o chat-tui ./cmd/chat-tui    # Build binary
go build ./...                         # All packages
go test ./internal/config ./internal/api ./internal/storage
go install github.com/evallife/chat-tui/cmd/chat-tui@latest  # Install globally
```

## NOTES

- Release workflow triggers on `v*` tags, builds for linux/windows/darwin (amd64)
- System prompts have a manager UI (list / new / edit / delete / apply)
- ChatModelAgent ships with common tools in `extraTools()`; append more `tool.BaseTool` there as needed
- GitHub Pages marketing site lives on the `gh-pages` branch (not this tree)
