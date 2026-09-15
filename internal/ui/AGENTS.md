# internal/ui — TUI Layer

## Where To Look

| Task | Location |
|------|----------|
| App shell, global keys, constructor | `ui.go` |
| Theme definitions (5 themes) | `theme.go` `themes` map |
| Sidebar, footer, quit, new chat | `layout.go` |
| Chat view, input, stream, refresh | `chat.go` |
| Slash commands and export | `commands.go` |
| In-chat search | `search.go` |
| History list / preview / delete | `history.go` |
| System prompt manager | `prompts.go` |
| Provider + agent settings | `settings.go` |
| Copy mode | `copy.go` |
| HITL Allow/Deny for write/run | `confirm.go` |

## Conventions

- Theme struct: `{Name, Primary, Secondary, Tertiary, Border, Title, Accent, InputBg, InputFg}`
- All tcell colors — mix of `tcell.Color*` constants and `tcell.NewRGBColor()`
- Clipboard via `atotto/clipboard` (no system clipboard dependency)
- Markdown rendering via `charmbracelet/glamour`
- Stream cancellation via `context.WithCancel` — Esc key during streaming
- `TViewUI` implements `api.ToolConfirmer`; wired in `NewTViewUI` via `apiClient.SetConfirmer(ui)`
- Persist errors from config/storage with `reportError` / `appendSystemMsg`

## Anti-Patterns

- Do not re-merge these files into a single `tview_ui.go`
- Do not replace `apiClient` with `NewClient` on settings save — use `UpdateConfig` so the confirmer stays attached
