package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/evallife/chat-tui/internal/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sashabaranov/go-openai"
)

func (ui *TViewUI) setupChatView() {
	ui.ChatView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			ui.App.Draw()
		})
	ui.ChatView.SetBorder(true).SetTitle(fmt.Sprintf(" Chat History · %s/%s ", ui.config.CanonicalProvider(), ui.config.Model))
	ui.ChatView.SetTitleColor(tview.Styles.TitleColor)

	ui.InputField = tview.NewTextArea().
		SetLabel("> ").
		SetPlaceholder("Type a message (Shift+Enter for new line)...")
	ui.InputField.SetSize(3, 0)
	ui.InputField.SetChangedFunc(func() {
		ui.syncInputHeight()
	})
	ui.InputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() == 0 {
			switch event.Key() {
			case tcell.KeyUp:
				fromRow, _, _, _ := ui.InputField.GetCursor()
				if recallHistoryOnUp(fromRow) {
					ui.navigateHistory(-1)
					return nil
				}
				return event
			case tcell.KeyDown:
				if recallHistoryOnDown(ui.historyIndex) {
					ui.navigateHistory(1)
					return nil
				}
				return event
			case tcell.KeyEnter:
				if ui.isProcessingInput || ui.isStreaming {
					return nil
				}

				text := ui.InputField.GetText()
				if text == "" {
					return nil
				}

				ui.isProcessingInput = true
				ui.InputField.SetText("", true)
				ui.handleInput(text)
				ui.isProcessingInput = false
				return nil
			}
		}
		return event
	})
	ui.InputField.SetBorder(true).SetTitle(" Input (Enter to send, Shift+Enter for new line) ")
	ui.InputField.SetTitleColor(tview.Styles.TitleColor)
	ui.InputField.SetTextStyle(tcell.StyleDefault.Foreground(tview.Styles.PrimaryTextColor).Background(tview.Styles.PrimitiveBackgroundColor))
	ui.InputField.SetLabelStyle(tcell.StyleDefault.Foreground(tview.Styles.SecondaryTextColor).Background(tview.Styles.PrimitiveBackgroundColor))
	ui.InputField.SetPlaceholderStyle(tcell.StyleDefault.Foreground(tview.Styles.TertiaryTextColor).Background(tview.Styles.PrimitiveBackgroundColor))
}

func (ui *TViewUI) handleInput(input string) {
	if strings.HasPrefix(input, "/") {
		ui.handleCommand(input)
		return
	}

	ui.addInputHistory(input)
	ui.messages = append(ui.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: input,
	})

	if ui.convID == "" {
		title := input
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		id, err := ui.storage.CreateConversation(title, ui.config.Model, ui.systemPrompt)
		if err != nil {
			ui.reportError("Failed to create conversation", err)
		} else {
			ui.convID = id
		}
	}
	if ui.convID != "" {
		if err := ui.storage.SaveMessage(ui.convID, openai.ChatMessageRoleUser, input); err != nil {
			ui.reportError("Failed to save message", err)
		}
	}

	ui.refreshChat()
	go ui.streamAgentResponse()
}

func (ui *TViewUI) addInputHistory(input string) {
	if input == "" {
		return
	}
	ui.inputHistory = append(ui.inputHistory, input)
	ui.historyIndex = -1
	ui.draftInput = ""
}

func (ui *TViewUI) navigateHistory(direction int) {
	if len(ui.inputHistory) == 0 {
		return
	}

	if ui.historyIndex == -1 {
		ui.draftInput = ui.InputField.GetText()
	}

	switch direction {
	case -1:
		if ui.historyIndex == -1 {
			ui.historyIndex = len(ui.inputHistory) - 1
		} else if ui.historyIndex > 0 {
			ui.historyIndex--
		}
	case 1:
		if ui.historyIndex == -1 {
			return
		}
		if ui.historyIndex < len(ui.inputHistory)-1 {
			ui.historyIndex++
		} else {
			ui.historyIndex = -1
			ui.InputField.SetText(ui.draftInput, true)
			return
		}
	}

	if ui.historyIndex >= 0 && ui.historyIndex < len(ui.inputHistory) {
		ui.InputField.SetText(ui.inputHistory[ui.historyIndex], true)
	}
}

func (ui *TViewUI) stopStreaming() {
	if ui.isStreaming && ui.streamCancel != nil {
		ui.streamCancel()
	}
}

func (ui *TViewUI) streamAgentResponse() {
	ctx, cancel := context.WithCancel(context.Background())
	ui.isStreaming = true
	ui.streamCancel = cancel
	defer func() {
		cancel()
		ui.isStreaming = false
		ui.streamCancel = nil
		ui.App.QueueUpdateDraw(func() {
			ui.InputField.SetTitle(" Input (Enter to send, Shift+Enter for new line) ")
			ui.refreshStatus()
		})
	}()

	provider := ui.config.CanonicalProvider()
	ui.App.QueueUpdateDraw(func() {
		ui.InputField.SetTitle(" Input (Esc to cancel) ")
		ui.ChatView.SetTitle(fmt.Sprintf(" Chat History · %s/%s · streaming ", provider, ui.config.Model))
		fmt.Fprintf(ui.ChatView, "\n[green][b]ASSISTANT[-][/b]\n")
		ui.refreshStatus()
	})

	fullResponse, err := ui.apiClient.StreamAgent(ctx, ui.systemPrompt, ui.messages, func(ev api.Event) {
		text := ev.Text
		tool := ev.ToolName
		status := ev.Status
		kind := ev.Kind
		ui.App.QueueUpdateDraw(func() {
			switch kind {
			case api.EventText:
				fmt.Fprint(ui.ChatView, text)
			case api.EventReasoning:
				fmt.Fprintf(ui.ChatView, "[gray]%s[-]", text)
			case api.EventToolCall:
				args := strings.TrimSpace(text)
				if len(args) > 180 {
					args = args[:180] + "…"
				}
				if args != "" {
					fmt.Fprintf(ui.ChatView, "\n[yellow]⚙ tool:%s[-] [gray]%s[-]\n", tool, args)
				} else {
					fmt.Fprintf(ui.ChatView, "\n[yellow]⚙ tool:%s calling…[-]\n", tool)
				}
				ui.ChatView.SetTitle(fmt.Sprintf(" Chat History · tool:%s ", tool))
			case api.EventToolResult:
				name := tool
				if name == "" {
					name = "tool"
				}
				body := strings.TrimSpace(text)
				if len(body) > 600 {
					body = body[:600] + "\n…(truncated)"
				}
				if body == "" {
					fmt.Fprintf(ui.ChatView, "[yellow]⚙ tool:%s ✓[-]\n", name)
				} else {
					fmt.Fprintf(ui.ChatView, "[yellow]⚙ tool:%s ✓[-]\n[gray]%s[-]\n", name, body)
				}
			case api.EventStatus:
				if status != "" {
					ui.ChatView.SetTitle(fmt.Sprintf(" Chat History · %s ", status))
				}
			}
			ui.ChatView.ScrollToEnd()
		})
	})

	ui.App.QueueUpdateDraw(func() {
		if fullResponse != "" {
			ui.messages = append(ui.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: fullResponse,
			})
			if ui.convID != "" {
				if saveErr := ui.storage.SaveMessage(ui.convID, openai.ChatMessageRoleAssistant, fullResponse); saveErr != nil {
					ui.reportError("Failed to save assistant message", saveErr)
				}
			}
		}
		ui.refreshChat()
		if err != nil {
			if ctx.Err() != nil {
				ui.appendSystemMsg("[Stream cancelled]")
			} else {
				ui.appendSystemMsg(fmt.Sprintf("API Error: %v", err))
			}
		}
	})
}

func (ui *TViewUI) refreshChat() {
	ui.ChatView.Clear()
	ui.ChatView.SetTitle(fmt.Sprintf(" Chat History · %s/%s ", ui.config.CanonicalProvider(), ui.config.Model))
	if ui.systemPrompt != "" {
		fmt.Fprintf(ui.ChatView, "[gray][i]System Prompt: %s[-][/i]\n\n", ui.systemPrompt)
	}
	for _, m := range ui.messages {
		roleColor := "purple"
		if m.Role == openai.ChatMessageRoleAssistant {
			roleColor = "green"
		}

		fmt.Fprintf(ui.ChatView, "[%s][b]%s[-][/b]\n", roleColor, strings.ToUpper(m.Role))
		content := m.Content
		if ui.renderer != nil {
			if rendered, err := ui.renderer.Render(m.Content); err == nil {
				content = tview.TranslateANSI(rendered)
			}
		}
		fmt.Fprintf(ui.ChatView, "%s\n\n", content)
	}
	ui.ChatView.ScrollToEnd()
}

func (ui *TViewUI) appendSystemMsg(msg string) {
	fmt.Fprintf(ui.ChatView, "[red][b]SYSTEM[-][/b]\n%s\n\n", msg)
	ui.ChatView.ScrollToEnd()
}
