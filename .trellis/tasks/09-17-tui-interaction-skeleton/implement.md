# Implement — TUI interaction skeleton

## Checklist

1. Add `internal/ui/keys.go` with exported-for-test helpers:
   - `escActionFor(frontPage string, searchActive, streaming bool) escAction`
   - `recallHistoryOnUp(cursorRow int) bool`
   - `recallHistoryOnDown(historyIndex int) bool`
   - `inputHeight(text string) int`
   - `statusLine(provider, model string, streaming, sidebarVisible bool) string`
2. Add `internal/ui/keys_test.go` covering:
   - idle chat Esc → none
   - streaming chat Esc → cancel_stream
   - history/settings/copy/prompts Esc → back
   - confirm-tool Esc → none
   - search Esc → close_search
   - Up only row 0; Down only when historyIndex != -1
   - height clamp 3–8
3. Rewrite `globalKeys` Esc branch to dispatch the table in `design.md`. Consume idle-chat Esc as no-op (`return nil`) so it cannot quit. HITL: do not call `confirmQuit`; leave the modal up (`return nil` or pass-through — pick the option that keeps Allow/Deny focused).
4. `layout.go`:
   - `sidebarVisible` default false
   - `mountChat` / `toggleSidebar` use the flag
   - replace `buildFooterBar` with `StatusBar` TextView height 1
   - Sidebar Quit → `confirmQuit`
   - keep a `chatColumn` flex pointer for `ResizeItem`
5. `chat.go`: cursor-aware Up/Down; `SetChangedFunc` to grow input; refresh status on stream start/end.
6. Focus restore: `focusChatInput()` used by history/settings/prompts/copy/search dismiss paths and confirm Cancel.
7. Update `/help` in `commands.go`, README shortcut table, `todo.md` (mark completed P0 interaction items that this covers).
8. `applyTheme` colors the status bar if present.
9. `go test ./internal/ui/` and `go test ./...`

## Validation

```bash
go test ./internal/ui/
go test ./...
go build -o /tmp/chat-tui ./cmd/chat-tui
```

Manual (after start, not in unit tests):

- Open app: no sidebar, no 8 buttons, status line visible
- Ctrl+B toggles menu; q on menu asks quit
- Esc idle: nothing; Esc while streaming: stop
- Ctrl+H then Esc: back to input
- Shift+Enter two extra lines, Up/Down moves inside the draft
- Up on first line recalls previous send

## Risky files

- `internal/ui/ui.go` — global capture; easy to break Ctrl+C copy vs quit
- `internal/ui/layout.go` — mountChat/search remount must keep sidebar flag
- `internal/ui/chat.go` — InputCapture must not swallow Shift+Enter

## Rollback

Git revert the UI/docs diff. No data format change.

## JSONL context

Sub-agents should load:

- `research/current-key-routing.md`
- `.trellis/spec/guides/code-reuse-thinking-guide.md`
- `.trellis/spec/backend/quality-guidelines.md`

Do not pre-register code paths in jsonl.

## Ready for start

Complex task artifacts: `prd.md`, `design.md`, `implement.md`, research note, jsonl manifests. Do not `task.py start` until the user approves the planning summary.
