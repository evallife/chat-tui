package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sashabaranov/go-openai"
	"github.com/user/xftui/internal/api"
	"github.com/user/xftui/internal/config"
	"github.com/user/xftui/internal/storage"
	"github.com/user/xftui/internal/types"
)

type TViewUI struct {
	App          *tview.Application
	Pages        *tview.Pages
	ChatView     *tview.TextView
	InputField   *tview.InputField
	HistoryList  *tview.List
	SettingsForm *tview.Form
	
	config    types.Config
	storage   *storage.Manager
	apiClient *api.Client
	messages  []openai.ChatCompletionMessage
	convID    string
	renderer  *glamour.TermRenderer
}

func NewTViewUI(cfg types.Config, store *storage.Manager) *TViewUI {
	ui := &TViewUI{
		App:     tview.NewApplication(),
		Pages:   tview.NewPages(),
		config:  cfg,
		storage: store,
		apiClient: api.NewClient(cfg),
	}

	// Theme / styling
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	tview.Styles.ContrastBackgroundColor = tcell.ColorDarkSlateGray
	tview.Styles.BorderColor = tcell.ColorDarkSlateGray
	tview.Styles.TitleColor = tcell.ColorLightSkyBlue
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorGray
	tview.Styles.TertiaryTextColor = tcell.ColorLightGray

	ui.renderer, _ = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	ui.setupChatView()
	ui.setupHistoryView()
	ui.setupSettingsView()

	// Layout main chat
	footer := ui.buildFooterBar()
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.ChatView, 0, 1, false).
		AddItem(ui.InputField, 3, 1, true).
		AddItem(footer, 3, 1, false)

	ui.Pages.AddPage("chat", flex, true, true)
	ui.App.SetRoot(ui.Pages, true).EnableMouse(true)

	// Global key handlers
	ui.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlN:
			ui.newConversation()
			return nil
		case tcell.KeyCtrlH:
			ui.showHistory()
			return nil
		case tcell.KeyCtrlS:
			ui.showSettings()
			return nil
		case tcell.KeyCtrlE:
			ui.exportHistory()
			return nil
		}
		return event
	})

	return ui
}

func (ui *TViewUI) setupChatView() {
	ui.ChatView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			ui.App.Draw()
		})
	ui.ChatView.SetBorder(true).SetTitle(" Chat History ")
	ui.ChatView.SetTitleColor(tcell.ColorLightSkyBlue)
	ui.ChatView.SetBorderColor(tcell.ColorDarkSlateGray)

	ui.InputField = tview.NewInputField().
		SetLabel("> ").
		SetFieldWidth(0)
	ui.InputField.SetBorder(true).SetTitle(" Input (Enter to send) ")
	ui.InputField.SetTitleColor(tcell.ColorLightSkyBlue)
	ui.InputField.SetBorderColor(tcell.ColorDarkSlateGray)
	ui.InputField.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.InputField.SetFieldTextColor(tcell.ColorWhite)
	ui.InputField.SetLabelColor(tcell.ColorLightCyan)
	ui.InputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			ui.App.Stop()
			return nil
		}
		return event
	})

	ui.InputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := ui.InputField.GetText()
			if text == "" {
				return
			}
			ui.InputField.SetText("")
			ui.handleInput(text)
		}
	})
}

func (ui *TViewUI) handleInput(input string) {
	if strings.HasPrefix(input, "/read ") {
		filePath := strings.TrimSpace(input[6:])
		content, err := os.ReadFile(filePath)
		if err != nil {
			ui.appendSystemMsg(fmt.Sprintf("Error reading file: %v", err))
			return
		}
		ui.messages = append(ui.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Content of file %s:\n\n%s", filePath, string(content)),
		})
		ui.refreshChat()
		return
	}

	ui.messages = append(ui.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: input,
	})

	if ui.convID == "" {
		title := input
		if len(title) > 30 { title = title[:27] + "..." }
		id, _ := ui.storage.CreateConversation(title, ui.config.Model)
		ui.convID = id
	}
	ui.storage.SaveMessage(ui.convID, openai.ChatMessageRoleUser, input)

	ui.refreshChat()
	go ui.streamOpenAIResponse()
}

func (ui *TViewUI) streamOpenAIResponse() {
	ctx := context.Background()
	stream, err := ui.apiClient.StreamChat(ctx, ui.messages)
	if err != nil {
		ui.App.QueueUpdateDraw(func() {
			ui.appendSystemMsg(fmt.Sprintf("API Error: %v", err))
		})
		return
	}
	defer stream.Close()

	var fullResponse strings.Builder
	ui.App.QueueUpdateDraw(func() {
		fmt.Fprintf(ui.ChatView, "\n[yellow][b]ASSISTANT[-][/b]\n")
	})

	for {
		response, err := stream.Recv()
		if err != nil {
			break
		}
		content := response.Choices[0].Delta.Content
		if content != "" {
			fullResponse.WriteString(content)
			ui.App.QueueUpdateDraw(func() {
				// Simple incremental write
				fmt.Fprint(ui.ChatView, content)
			})
		}
	}

	ui.App.QueueUpdateDraw(func() {
		ui.messages = append(ui.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fullResponse.String(),
		})
		ui.storage.SaveMessage(ui.convID, openai.ChatMessageRoleAssistant, fullResponse.String())
		ui.refreshChat() // Re-render with Glamour
	})
}

func (ui *TViewUI) refreshChat() {
	ui.ChatView.Clear()
	for _, m := range ui.messages {
		roleColor := "purple"
		if m.Role == openai.ChatMessageRoleAssistant { roleColor = "green" }
		
		fmt.Fprintf(ui.ChatView, "[%s][b]%s[-][/b]\n", roleColor, strings.ToUpper(m.Role))
		rendered, _ := ui.renderer.Render(m.Content)
		fmt.Fprintf(ui.ChatView, "%s\n\n", tview.TranslateANSI(rendered))
	}
	ui.ChatView.ScrollToEnd()
}

func (ui *TViewUI) appendSystemMsg(msg string) {
	fmt.Fprintf(ui.ChatView, "[red][b]SYSTEM[-][/b]\n%s\n\n", msg)
	ui.ChatView.ScrollToEnd()
}

func (ui *TViewUI) setupHistoryView() {
	ui.HistoryList = tview.NewList().
		SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
			ui.convID = secondaryText
			ui.messages, _ = ui.storage.GetMessages(ui.convID)
			ui.refreshChat()
			ui.Pages.SwitchToPage("chat")
		})
	ui.HistoryList.SetBorder(true).SetTitle(" History (Enter to load) ")
	ui.HistoryList.SetTitleColor(tcell.ColorLightSkyBlue)
	ui.HistoryList.SetBorderColor(tcell.ColorDarkSlateGray)
	ui.HistoryList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			ui.Pages.SwitchToPage("chat")
			return nil
		}
		if event.Key() == tcell.KeyDelete || event.Rune() == 'd' {
			ui.confirmDeleteSelected()
			return nil
		}
		return event
	})

	historyFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.HistoryList, 0, 1, true).
		AddItem(ui.buildHistoryBar(), 3, 1, false)

	ui.Pages.AddPage("history", historyFlex, true, false)
}

func (ui *TViewUI) showHistory() {
	ui.HistoryList.Clear()
	convs, _ := ui.storage.ListConversations()
	for _, c := range convs {
		ui.HistoryList.AddItem(c.Title, c.ID, 0, nil)
	}
	ui.Pages.SwitchToPage("history")
}

func (ui *TViewUI) setupSettingsView() {
	ui.SettingsForm = tview.NewForm().
		AddInputField("API Key", ui.config.APIKey, 40, nil, nil).
		AddInputField("Base URL", ui.config.BaseURL, 40, nil, nil).
		AddInputField("Model", ui.config.Model, 40, nil, nil).
		AddButton("Save", func() {
			ui.config.APIKey = ui.SettingsForm.GetFormItem(0).(*tview.InputField).GetText()
			ui.config.BaseURL = ui.SettingsForm.GetFormItem(1).(*tview.InputField).GetText()
			ui.config.Model = ui.SettingsForm.GetFormItem(2).(*tview.InputField).GetText()
			config.SaveConfig(ui.config)
			ui.apiClient = api.NewClient(ui.config)
			ui.Pages.SwitchToPage("chat")
		}).
		AddButton("Cancel", func() {
			ui.Pages.SwitchToPage("chat")
		})
	ui.SettingsForm.SetBorder(true).SetTitle(" Settings ")
	ui.SettingsForm.SetTitleColor(tcell.ColorLightSkyBlue)
	ui.SettingsForm.SetBorderColor(tcell.ColorDarkSlateGray)
	ui.Pages.AddPage("settings", ui.SettingsForm, true, false)
}

func (ui *TViewUI) showSettings() {
	ui.Pages.SwitchToPage("settings")
}

func (ui *TViewUI) newConversation() {
	ui.messages = []openai.ChatCompletionMessage{}
	ui.convID = ""
	ui.ChatView.Clear()
	ui.Pages.SwitchToPage("chat")
	ui.appendSystemMsg("New conversation started.")
}

func (ui *TViewUI) exportHistory() {
	filename := fmt.Sprintf("chat_export_%d.md", time.Now().Unix())
	var sb strings.Builder
	for _, msg := range ui.messages {
		sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n---\n\n", strings.ToUpper(msg.Role), msg.Content))
	}
	os.WriteFile(filename, []byte(sb.String()), 0644)
	ui.appendSystemMsg("History exported to " + filename)
}

func (ui *TViewUI) makeButton(label string, action func()) *tview.Button {
	btn := tview.NewButton(label)
	btn.SetSelectedFunc(action)
	btn.SetBackgroundColor(tcell.ColorDarkSlateGray)
	btn.SetBackgroundColorActivated(tcell.ColorLightSkyBlue)
	btn.SetLabelColor(tcell.ColorWhite)
	btn.SetLabelColorActivated(tcell.ColorBlack)
	return btn
}

func (ui *TViewUI) buildFooterBar() *tview.Flex {
	bar := tview.NewFlex().SetDirection(tview.FlexColumn)
	bar.SetBorder(true).SetTitle(" Actions ")
	bar.SetTitleColor(tcell.ColorLightSkyBlue)
	bar.SetBorderColor(tcell.ColorDarkSlateGray)

	bar.AddItem(ui.makeButton("New", ui.newConversation), 0, 1, false)
	bar.AddItem(ui.makeButton("History", ui.showHistory), 0, 1, false)
	bar.AddItem(ui.makeButton("Settings", ui.showSettings), 0, 1, false)
	bar.AddItem(ui.makeButton("Export", ui.exportHistory), 0, 1, false)
	bar.AddItem(ui.makeButton("Quit", func() { ui.App.Stop() }), 0, 1, false)
	return bar
}

func (ui *TViewUI) buildHistoryBar() *tview.Flex {
	bar := tview.NewFlex().SetDirection(tview.FlexColumn)
	bar.SetBorder(true).SetTitle(" History Actions ")
	bar.SetTitleColor(tcell.ColorLightSkyBlue)
	bar.SetBorderColor(tcell.ColorDarkSlateGray)

	bar.AddItem(ui.makeButton("Delete", ui.confirmDeleteSelected), 0, 1, false)
	bar.AddItem(ui.makeButton("Back", func() { ui.Pages.SwitchToPage("chat") }), 0, 1, false)
	return bar
}

func (ui *TViewUI) getSelectedHistoryID() (string, bool) {
	idx := ui.HistoryList.GetCurrentItem()
	if idx < 0 {
		return "", false
	}
	_, convID := ui.HistoryList.GetItemText(idx)
	if convID == "" {
		return "", false
	}
	return convID, true
}

func (ui *TViewUI) confirmDeleteSelected() {
	convID, ok := ui.getSelectedHistoryID()
	if !ok {
		return
	}

	modal := tview.NewModal().
		SetText("Delete this conversation? This cannot be undone.").
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			ui.Pages.RemovePage("confirm-delete")
			if buttonLabel == "Delete" {
				if err := ui.storage.DeleteConversation(convID); err != nil {
					ui.appendSystemMsg(fmt.Sprintf("Delete failed: %v", err))
				} else if ui.convID == convID {
					ui.convID = ""
					ui.messages = []openai.ChatCompletionMessage{}
					ui.ChatView.Clear()
				}
				ui.showHistory()
			}
		})

	ui.Pages.AddPage("confirm-delete", modal, true, true)
	ui.Pages.SwitchToPage("confirm-delete")
}

func (ui *TViewUI) Run() error {
	return ui.App.Run()
}
