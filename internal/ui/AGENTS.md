# internal/ui — TUI Layer

## Where To Look

| Task | Location |
|------|----------|
| All rendering, keybinds, forms | `tview_ui.go` |
| Theme definitions (5 themes) | `tview_ui.go` `themes` map (line ~69) |
| Sidebar + conversation list | `tview_ui.go` |
| Copy mode, history, settings | `tview_ui.go` |
| Markdown rendering | glamour renderer in `tview_ui.go` |
| Streaming + cancellation | `tview_ui.go` `streamOpenAIResponse()` |

## Conventions

- Theme struct: `{Name, Primary, Secondary, Tertiary, Border, Title, Accent, InputBg, InputFg}`
- All tcell colors — mix of `tcell.Color*` constants and `tcell.NewRGBColor()`
- Clipboard via `atotto/clipboard` (no system clipboard dependency)
- Markdown rendering via `charmbracelet/glamour`
- Stream cancellation via `context.WithCancel` — Esc key during streaming

## Anti-Patterns

- No component separation — all code in one file
