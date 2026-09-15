package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sashabaranov/go-openai"
)

func (ui *TViewUI) setupHistoryView() {
	ui.HistoryList = tview.NewList()
	ui.HistoryList.SetBorder(true).SetTitle(" History (Enter/Double-Click to Load) ")
	ui.HistoryList.ShowSecondaryText(false)

	ui.HistoryList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			ui.Pages.SwitchToPage("chat")
			return nil
		}
		if event.Key() == tcell.KeyDelete || event.Rune() == 'd' {
			ui.confirmDeleteSelected()
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			idx := ui.HistoryList.GetCurrentItem()
			_, secondary := ui.HistoryList.GetItemText(idx)
			ui.loadConversation(secondary)
			return nil
		}
		return event
	})

	ui.HistoryList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		now := time.Now()
		_, id := ui.HistoryList.GetItemText(index)
		if id == "" {
			return
		}

		if index == ui.lastClickedIdx && now.Sub(ui.lastClickedTime) < 800*time.Millisecond {
			ui.loadConversation(id)
			ui.lastClickedIdx = -1
		} else {
			ui.lastClickedIdx = index
			ui.lastClickedTime = now
		}
	})

	ui.HistoryList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		ui.HistoryPreview.Clear()
		if secondaryText == "" {
			return
		}
		msgs, err := ui.storage.GetMessages(secondaryText)
		if err != nil {
			fmt.Fprintf(ui.HistoryPreview, "[red]Failed to load preview: %v[-]", err)
			return
		}
		if len(msgs) == 0 {
			fmt.Fprintf(ui.HistoryPreview, "[gray]No messages in this conversation.[-]")
			return
		}
		for i, m := range msgs {
			if i > 5 {
				break
			}
			roleColor := "purple"
			if m.Role == openai.ChatMessageRoleAssistant {
				roleColor = "green"
			}
			fmt.Fprintf(ui.HistoryPreview, "[%s][b]%s[-][/b]\n", roleColor, strings.ToUpper(m.Role))
			summary := m.Content
			if len(summary) > 200 {
				summary = summary[:197] + "..."
			}
			fmt.Fprintf(ui.HistoryPreview, "%s\n\n", summary)
		}
	})

	ui.HistoryPreview = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	ui.HistoryPreview.SetBorder(true).SetTitle(" Preview ")

	historyFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.HistoryList, 35, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(ui.HistoryPreview, 0, 1, false).
			AddItem(ui.buildHistoryBar(), 3, 1, false), 0, 2, false)

	ui.Pages.AddPage("history", historyFlex, true, false)
}

func (ui *TViewUI) loadConversation(id string) {
	if id == "" {
		return
	}
	ui.convID = id
	conv, err := ui.storage.GetConversation(ui.convID)
	if err != nil {
		ui.reportError("Failed to load conversation", err)
		return
	}
	ui.systemPrompt = conv.SystemPrompt
	msgs, err := ui.storage.GetMessages(ui.convID)
	if err != nil {
		ui.reportError("Failed to load messages", err)
		return
	}
	ui.messages = msgs
	ui.refreshChat()
	ui.Pages.SwitchToPage("chat")
}

func (ui *TViewUI) showHistory() {
	ui.HistoryList.Clear()
	ui.HistoryPreview.Clear()
	ui.lastClickedIdx = -1
	convs, err := ui.storage.ListConversations()
	if err != nil {
		ui.reportError("Failed to list conversations", err)
		return
	}
	if len(convs) == 0 {
		ui.HistoryList.AddItem("No history yet", "", 0, nil)
	} else {
		for _, c := range convs {
			ui.HistoryList.AddItem(c.Title, c.ID, 0, nil)
		}
	}
	ui.Pages.SwitchToPage("history")
}

func (ui *TViewUI) buildHistoryBar() *tview.Flex {
	bar := tview.NewFlex().SetDirection(tview.FlexColumn)
	bar.SetBorder(true).SetTitle(" History Actions ")
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
	return convID, convID != ""
}

func (ui *TViewUI) confirmDeleteSelected() {
	convID, ok := ui.getSelectedHistoryID()
	if !ok {
		return
	}

	modal := tview.NewModal().
		SetText("Delete this conversation?").
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
}
