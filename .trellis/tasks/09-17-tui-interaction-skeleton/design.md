# Design — TUI interaction skeleton

## Architecture and boundaries

All behavior stays in `internal/ui`. No API, storage, or config schema changes.

Split key *policy* from tview *wiring*:

| Piece | Responsibility |
|---|---|
| `keys.go` (new) | Pure functions: Esc action, whether Up/Down should recall history, input row count, status text |
| `ui.go` `globalKeys` | Ask policy, then call existing page helpers |
| `layout.go` | Sidebar visibility flag, status bar, mount/toggle |
| `chat.go` | Input capture + grow on change |
| `layout.go` / `prompts.go` / `history.go` / `settings.go` / `copy.go` / `search.go` | Focus restore when returning to chat |

Do not merge files back into a single `tview_ui.go` (`internal/ui/AGENTS.md`).

## Esc policy

```
type escContext struct {
    FrontPage   string
    SearchActive bool
    Streaming   bool
}

type escAction string
const (
    escNone          escAction = "none"
    escCancelStream  escAction = "cancel_stream"
    escCloseSearch   escAction = "close_search"
    escBackChat      escAction = "back_chat"      // history, settings, copy
    escBackPrompts   escAction = "back_prompts"   // prompt_editor, confirm_delete_prompt
    escCloseOverlay  escAction = "close_overlay"  // confirm-quit, confirm-delete
    escIgnoreHITL    escAction = "ignore_hitl"    // confirm-tool: pass through? or none
)
```

Recommended mapping (see PRD R2):

- `copy` → hide copy (`escBackChat`)
- `history`, `settings`, `system_prompts_mgr` → switch to chat + focus input
- `prompt_editor` → remove editor page, `showSystemPrompts()`
- `confirm_delete_prompt` → remove that page, stay on prompts
- `confirm-quit`, `confirm-delete` → remove that page (Cancel)
- `confirm-tool` → `escNone` (leave modal focused; do not quit, do not auto-deny)
- `chat` + searchActive → clear search
- `chat` + streaming → cancel
- `chat` idle → `escNone`

`globalKeys` consumes Esc whenever the action is not `escNone`. For `escNone` on HITL, return `event` so the modal can keep focus; for idle chat, return `nil` to avoid tview treating Esc as “finish form item”, or return `event` if chat has no form finish handler. Implementation should verify TextArea does not finish on Esc; if it does, consume with no-op.

## Quit

- `Ctrl+C`: copy page → copy selection; else `confirmQuit()` (unchanged).
- Sidebar Quit item: `ui.confirmQuit` instead of `ui.App.Stop`.

## Layout

```
MainFlex (column)
  [Sidebar 20 | optional]
  ChatColumn (row)
    [searchInput 1 | optional]
    ChatView  (flex)
    InputField (fixed 3–8, proportion 0)
    StatusBar (fixed 1, proportion 0, no border)
```

State:

- `sidebarVisible bool` — default false. `toggleSidebar` uses the flag, not `GetItem(0)` type checks.
- `chatColumnFlex *tview.Flex` — keep a pointer so `ResizeItem(InputField, h, 0)` works after grow.
- `StatusBar *tview.TextView` — `SetDynamicColors(true)`, wrap off.

`mountChat` always rebuilds the chat page (already does). Preserve `sidebarVisible` across search toggle.

Status text builder (pure):

- idle: `[provider/model]  Ctrl+N new  Ctrl+H hist  Ctrl+S set  Ctrl+B menu  Ctrl+C quit`
- streaming: `[provider/model · streaming]  Esc stop`
- optional: `menu on` when sidebar visible

Update status when: mount, stream start/end, sidebar toggle, settings save (provider/model).

## Input grow and history

Pure helpers:

- `inputHeight(text string) int` → `clamp(strings.Count(text, "\n")+1, 3, 8)`
- `recallHistoryOnUp(cursorRow int) bool` → `cursorRow == 0`
- `recallHistoryOnDown(historyIndex int) bool` → `historyIndex != -1`

Wiring:

- `SetChangedFunc` on input → `ResizeItem` + `syncInputHeight`.
- Input capture: unmodified Up/Down consult helpers; Enter send unchanged.
- After `SetText` from history navigation, `cursorAtTheEnd=true` (already).

## Focus

Add `focusInput()` = `Pages.SwitchToPage("chat"); App.SetFocus(InputField)` and use it from:

- `hideCopyMode`
- history Esc / loadConversation (already switches page)
- settings Save/Cancel
- prompts Esc / after Use
- `clearSearch`
- confirm-quit Cancel, confirm-delete Cancel

Settings Cancel today does not `SetFocus`. History Esc (once it actually runs) does not `SetFocus`.

## Compatibility

- Slash commands, HITL, Copy Mode page, Ctrl+N/H/S/E/Y/F stay.
- README + `/help` drop “footer Stop” and “Esc quits”.
- Theme: status bar uses existing `tview.Styles` / theme tertiary color; apply in `applyTheme` if the widget exists.

## Trade-offs

- Removing footer buttons hurts mouse-only users. Accepted; Menu via `Ctrl+B` remains.
- Status bar is not clickable in P0.
- HITL Esc no-op may surprise people who want Esc=Deny. Safer to require an explicit button.

## Rollback

Revert `internal/ui/*`, README, `todo.md`. No migrations.
