# TUI interaction skeleton

## Goal

Make the main chat view the product: less navigation chrome, Esc that dismisses instead of quitting, and a multiline input that can actually be edited. Keyboard-first users should stay in the composer without fighting the app.

## Background

The TUI currently stacks a 20-column sidebar, a 3-row eight-button footer, and a locked 3-line input around the transcript. The same actions are reachable from sidebar shortcuts, global Ctrl keys, footer buttons, and slash commands. `App.SetInputCapture` swallows Esc before page handlers, so History / Settings / Search Esc opens the quit modal. Input Up/Down always recalls send history, so a Shift+Enter draft cannot move between lines.

This task is the P0 “操作骨架” slice agreed after interaction review. Streaming re-render, copy-last-message, session rename, and settings-form split are later work.

## Confirmed facts

- Chat layout is `internal/ui/layout.go` (`mountChat`, `buildFooterBar`, `toggleSidebar`) plus `internal/ui/chat.go` (input capture, `SetSize(3, 0)`).
- Global keys live in `internal/ui/ui.go` `globalKeys`. Esc order today: copy page → cancel stream → `confirmQuit()`.
- History, settings, prompts, search, and HITL confirm each have (or lack) local Esc handlers that never run for Esc because global capture wins. Evidence: `research/current-key-routing.md`.
- tview `TextArea.GetCursor()` returns visual row 0 as the first row (`tview@v0.42.0`). `Flex.ResizeItem` can change input height without remounting.
- Sidebar Quit calls `App.Stop()`; `Ctrl+C` uses `confirmQuit()`.
- No existing UI unit tests. Key policy should be extracted so it can be tested without driving tview.

## Requirements

- **R1. Chat-first chrome.** On the chat page, the sidebar is hidden by default. `Ctrl+B` still toggles it. The eight-button footer is removed. A one-line status bar replaces it and shows provider/model, idle vs streaming, and the shortcuts that currently do something.
- **R2. Esc dismisses, it does not quit.** Esc closes the front overlay/page (history, settings, prompts, copy, search, confirm-delete, confirm-quit Cancel path). During a stream on chat, Esc cancels the run. Idle chat Esc is a no-op. HITL Allow/Deny (`confirm-tool`) is not dismissed by Esc; Allow/Deny remain the only answers.
- **R3. Quit is explicit.** `Ctrl+C` on non-copy pages still opens the quit confirm modal. Sidebar Quit uses that same modal. Copy Mode `Ctrl+C` still copies.
- **R4. Multiline input is editable.** Input grows with newline count, clamped 3–8 rows. Unmodified Up recalls send history only when the cursor is on visual row 0. Unmodified Down recalls newer history only while already browsing history; otherwise Down moves the cursor. Enter still sends; Shift+Enter still inserts a newline. Send remains blocked while streaming.
- **R5. Focus returns to the composer.** Dismissing history, settings, prompts, copy, search, or a confirm modal that returns to chat focuses the input.
- **R6. Docs match the new keys.** README shortcut table, `/help` text, and `todo.md` P0 items that this task completes stay consistent with the behavior above.

## Acceptance Criteria

- [ ] **AC1.** Fresh chat page shows no sidebar and no eight-button action bar. Transcript + input + one-line status bar fill the frame. `Ctrl+B` shows and hides the existing Menu list.
- [ ] **AC2.** Status bar includes canonical provider, model, and either an idle hint set (`Ctrl+N` / `Ctrl+H` / `Ctrl+S` / `Ctrl+B` / `Ctrl+C`) or a streaming hint (`Esc` stop).
- [ ] **AC3.** From History, Settings, System Prompts, Copy Mode, and in-chat Search, Esc returns to chat (or one prompt-editor step back) and does not open the quit modal. Idle chat Esc does not open the quit modal.
- [ ] **AC4.** `Ctrl+C` on chat still opens “Quit the application?”. Choosing Cancel leaves the session running. Sidebar Quit opens the same modal instead of calling `App.Stop()` directly.
- [ ] **AC5.** A three-line draft can move the cursor with Up/Down between those lines. Up on the first visual row still walks previous sends. Down while browsing history walks forward and restores the draft at the end.
- [ ] **AC6.** Pasting or typing 6+ newlines grows the input up to 8 rows; a single-line prompt stays at 3 rows.
- [ ] **AC7.** After Esc-back from History/Settings/Copy, the next keystroke goes into the input, not the sidebar or a leftover page.
- [ ] **AC8.** Unit tests cover Esc routing and Up/Down history interception without requiring a tview `Application.Run`. `go test ./internal/ui/` passes. README and `/help` no longer say Esc quits or that a footer Stop button exists.

## Out of scope

- Streaming `refreshChat()` wipe and persisting `⚙ tool:` lines (P1).
- Copy last assistant message / in-view selection without Copy Mode (P1).
- Session rename, tags, pin, auto-title (P2).
- Splitting Settings into Provider / Agent / Theme (P2).
- User-configurable keybindings.
- Command palette.
- Mouse clickable status-bar regions.
- Agent, storage, config schema, and provider changes.

## Key decisions

- Default sidebar hidden (keyboard density). Menu remains behind `Ctrl+B` for mouse users.
- Status bar is a `TextView`, not a button row. Stop is Esc during stream.
- Bare Esc never quits. Quit is `Ctrl+C` + modal only.
- HITL tool confirm ignores Esc so a nervous Esc does not silently deny/allow.
- History Up/Down uses visual row 0 and “already browsing” rather than last-line Down to start history (avoids fighting wrapped lines).

## Risks

- tview visual row vs wrapped text: GetCursor row 0 is visual, which is the intended “first line” for Up.
- Mouse-only users lose footer buttons; mitigated by `Ctrl+B` menu and README.
- Global input capture must keep Ctrl shortcuts working on every page, including Settings forms (already true today).
