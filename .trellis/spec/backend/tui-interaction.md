# TUI Interaction

> Keyboard routing, chrome, and composer conventions for `internal/ui`.

## Scope / Trigger

Use this spec when changing global keys, overlays, the chat composer, sidebar, or status bar.

## Signatures

```go
func escActionFor(frontPage string, searchActive, streaming bool) escAction
func recallHistoryOnUp(cursorRow int) bool
func recallHistoryOnDown(historyIndex int) bool
func inputHeight(text string) int
func statusLine(provider, model string, streaming, sidebarVisible bool) string
```

Policy lives in `internal/ui/keys.go`. Wiring lives in `ui.go` / `layout.go` / `chat.go`. Do not inline a second Esc table in `globalKeys`.

## Contracts

| Input | Output |
|---|---|
| Front page `history` / `settings` / `copy` / `system_prompts_mgr` | `escBackChat` |
| `prompt_editor` / `confirm_delete_prompt` | `escBackPrompts` |
| `confirm-quit` / `confirm-delete` / `export-dialog` | `escCloseOverlay` |
| `confirm-tool` | `escNone` (consume Esc; do not quit; do not Allow/Deny) |
| `chat` + search | `escCloseSearch` |
| `chat` + streaming | `escCancelStream` |
| `chat` idle | `escNone` (consume Esc; do not quit) |

Quit is `Ctrl+C` → `confirmQuit()` only. Sidebar Quit must call `confirmQuit`, never `App.Stop()`.

Composer:

- Height = `clamp(newlines+1, 3, 8)`, Flex proportion 0.
- Up recalls send history only when `TextArea.GetCursor()` `fromRow == 0`.
- Down recalls only while `historyIndex != -1`.

Chrome:

- Sidebar hidden by default (`sidebarVisible == false`). `Ctrl+B` toggles via that flag, not `GetItem(0)` type checks.
- No eight-button footer. One-line `StatusBar` TextView.

Focus:

- Pages that return to chat call `focusChatInput()`.
- Closing an overlay on top of another page (e.g. delete-confirm on History) uses `focusChatIfFront()` so History stays.

## Validation & Error Matrix

| Condition | Behavior |
|---|---|
| Esc while HITL modal is front | No-op. tview `Modal` Esc invokes `DoneFunc("", "")`, and `confirm.go` treats non-Allow as Deny. |
| Esc while a page-local handler also binds Esc | Global `App.SetInputCapture` wins. Put routing in `escActionFor`. |
| Send during stream | Input capture ignores Enter. |

## Good / Base / Bad

- Good: idle chat Esc does nothing; History Esc returns to the composer.
- Base: streaming Esc cancels the run and refreshes the status line.
- Bad: idle Esc opens the quit modal; passing HITL Esc through auto-denies the tool.

## Tests Required

`internal/ui/keys_test.go` must cover:

- idle / streaming / search / search+stream
- history, settings, copy, prompts, prompt editor, HITL, confirm-quit, confirm-delete, export-dialog
- Up row 0 vs row 1; Down idle vs browsing
- height clamp 3–8
- idle vs streaming status text

## Wrong vs Correct

#### Wrong
```go
case tcell.KeyEsc:
    if ui.isStreaming { ui.streamCancel(); return nil }
    ui.confirmQuit()
    return nil
```

#### Correct
```go
case tcell.KeyEsc:
    return ui.handleEsc() // dispatches escActionFor
```

## Gotchas

> **Warning**: `App.SetInputCapture` runs before the focused primitive. History/Settings/Search local Esc handlers are fallbacks and will not see Esc unless globalKeys returns the event.

> **Warning**: tview `Modal` cancel (Esc) calls `DoneFunc` with an empty label. HITL must consume Esc (`return nil`) or a nervous Esc silently denies `write_file` / `run_command`.
