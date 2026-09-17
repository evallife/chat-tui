# Current TUI key routing and chrome (P0 evidence)

Inspected 2026-09-17 against `internal/ui/*.go` and tview v0.42.0.

## Global Esc is captured before page handlers

`TViewUI.globalKeys` is installed with `App.SetInputCapture` in `ui.go`. tview runs application input capture *before* the focused primitive. Current Esc branch:

1. If front page is `copy` → hide copy mode
2. Else if streaming → cancel
3. Else → `confirmQuit()`

Effects:

- History page has a local Esc handler (`history.go`) meant to return to chat, but global capture consumes Esc first, so History Esc opens the quit modal.
- Settings has no Esc handler; Esc also opens quit.
- System prompt manager Esc works only if globalKeys does not swallow it first (it does).
- Search Esc is on `searchInput`; also swallowed unless search is somehow not going through globalKeys (it is).

P0 must route Esc by front page / overlay / search / streaming, and must **not** quit on a bare Esc.

## Duplicate navigation chrome

Chat column (`layout.go`):

- Sidebar always mounted, width 20
- Input fixed height 3 (`chat.go` `SetSize(3, 0)` plus Flex `AddItem(..., 3, 1, true)`)
- Footer `buildFooterBar()` is a bordered Flex of 8 buttons, height 3

Same actions exist as sidebar items (`n/h/s/p/q` only when sidebar focused), global Ctrl keys, footer buttons, and slash commands.

Sidebar Quit calls `ui.App.Stop()` with no confirm; `Ctrl+C` / Esc use `confirmQuit()`.

## Input Up/Down always recalls send history

`InputField.SetInputCapture` intercepts unmodified Up/Down with no cursor check. tview `TextArea.GetCursor()` returns visual `fromRow` (0 = first row). Multiline drafts cannot move the cursor between lines.

Bash-like rule for P0:

- Up intercepts only when `fromRow == 0`
- Down intercepts only while already browsing `inputHistory` (`historyIndex != -1`)
- Otherwise arrows pass through to TextArea

## Input height

`tview.Flex.ResizeItem(p, fixedSize, proportion)` can change the input row without remounting. Grow from newline count, clamp 3–8, proportion 0.

## Pages that should dismiss on Esc

| Front page | Today | P0 |
|---|---|---|
| `copy` | hide copy | keep |
| `history` | quit modal | back to chat + focus input |
| `settings` | quit modal | back to chat + focus input |
| `system_prompts_mgr` / `prompt_editor` / `confirm_delete_prompt` | quit modal | back one step |
| `confirm-quit` | already on modal | Cancel / stay |
| `confirm-delete` | quit modal | close delete modal |
| `confirm-tool` | quit modal | leave HITL modal to its own buttons (do not quit) |
| `chat` + search | quit modal | clear search |
| `chat` + streaming | cancel | keep |
| `chat` idle | quit modal | no-op |

Quit remains `Ctrl+C` → confirm modal. Sidebar Quit must use the same modal.

## Out of this research (deferred P1)

Streaming still `refreshChat()` after the run, which drops live `⚙ tool:` lines. Copy Mode still dumps the full transcript. Not in P0.
