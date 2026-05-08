# PROJECT KNOWLEDGE BASE

**Generated:** 2026-05-08T06:09:08Z
**Commit:** 863b481
**Branch:** main

## OVERVIEW

Terminal chat client for OpenAI-compatible APIs. Go 1.25+, two TUI frameworks (Bubble Tea + tview), SQLite storage.

## STRUCTURE

```
chat-tui/
├── cmd/chat-tui/main.go    # Entry point: config → storage → UI
├── internal/
│   ├── api/openai.go        # Streaming chat client (go-openai)
│   ├── config/config.go     # JSON config: ~/.xftui.json
│   ├── storage/sqlite.go    # Pure-Go SQLite: ~/.xftui.db
│   ├── types/types.go       # Shared structs (Config, Conversation, SystemPrompt)
│   └── ui/
│       └── tview_ui.go      # tview UI (active, ~960 lines)
├── go.mod
├── .github/workflows/release.yml  # Cross-platform release (v* tags)
└── todo.md                  # Feature backlog
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| API integration | `internal/api/openai.go` | Streaming only, 32 lines |
| Config schema | `internal/types/types.go` | Config, Conversation, SystemPrompt |
| Storage operations | `internal/storage/sqlite.go` | CRUD + migrations |
| UI logic | `internal/ui/tview_ui.go` | All rendering, keybinds, streaming, commands |
| Themes | `internal/ui/tview_ui.go` | `themes` map: night/nord/gruvbox/solarized-dark/light |

## CONVENTIONS

- **No tests exist** — add when touching storage or config
- **Config path hardcoded**: `~/.xftui.json`, `~/.xftui.db`
- **go-openai types used everywhere** — `openai.ChatCompletionMessage` is the message type, imported in types, storage, api, ui
- **UUID for IDs** — conversations and system prompts use `google/uuid`
- **Inline migrations** — `ALTER TABLE ADD COLUMN IF NOT EXISTS` pattern in `NewManager()`

## ANTI-PATTERNS (THIS PROJECT)

- **No error wrapping** — errors returned bare, no `fmt.Errorf("...: %w", err)`
- **Ignored errors** — `config.SaveConfig()` error ignored in main.go line 29
- **Hardcoded defaults** — API key placeholder `"YOUR_API_KEY_HERE"` in main.go

## COMMANDS

```bash
go build -o chat-tui ./cmd/chat-tui    # Build binary
go install github.com/evallife/chat-tui/cmd/chat-tui@latest  # Install globally
```

## NOTES

- Release workflow triggers on `v*` tags, builds for linux/windows/darwin (amd64)
- System prompt stored per-conversation but no UI to edit it yet (see `todo.md`)
