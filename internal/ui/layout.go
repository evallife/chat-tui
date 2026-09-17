package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sashabaranov/go-openai"
)

func (ui *TViewUI) setupSidebar() {
	ui.Sidebar = tview.NewList().
		AddItem("New Chat", "Start fresh", 'n', ui.newConversation).
		AddItem("History", "Load past chats", 'h', ui.showHistory).
		AddItem("Settings", "Provider + credentials", 's', ui.showSettings).
		AddItem("System Prompts", "Change AI role", 'p', ui.showSystemPrompts).
		AddItem("Quit", "Exit app", 'q', ui.confirmQuit)

	ui.Sidebar.SetBorder(true).SetTitle(" Menu ")
	ui.Sidebar.SetTitleColor(tview.Styles.TitleColor)
}

func (ui *TViewUI) setupStatusBar() {
	ui.StatusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetTextAlign(tview.AlignLeft)
	ui.StatusBar.SetBorder(false)
	ui.StatusBar.SetTextColor(tview.Styles.TertiaryTextColor)
	ui.StatusBar.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	ui.refreshStatus()
}

func (ui *TViewUI) refreshStatus() {
	if ui.StatusBar == nil {
		return
	}
	ui.StatusBar.SetText(statusLine(
		ui.config.CanonicalProvider(),
		ui.config.Model,
		ui.isStreaming,
		ui.sidebarVisible,
	))
}

func (ui *TViewUI) chatColumn(withSearch bool) tview.Primitive {
	col := tview.NewFlex().SetDirection(tview.FlexRow)
	if withSearch {
		col.AddItem(ui.searchInput, 1, 0, true)
	}
	col.AddItem(ui.ChatView, 0, 1, false)
	h := 3
	if ui.InputField != nil {
		h = inputHeight(ui.InputField.GetText())
	}
	col.AddItem(ui.InputField, h, 0, !withSearch)
	if ui.StatusBar != nil {
		col.AddItem(ui.StatusBar, 1, 0, false)
	}
	ui.chatColumnFlex = col
	return col
}

func (ui *TViewUI) mountChat(withSearch bool) {
	ui.MainFlex = tview.NewFlex().SetDirection(tview.FlexColumn)
	if ui.sidebarVisible {
		ui.MainFlex.AddItem(ui.Sidebar, 20, 1, false)
	}
	ui.MainFlex.AddItem(ui.chatColumn(withSearch), 0, 4, true)
	ui.Pages.AddPage("chat", ui.MainFlex, true, true)
	ui.Pages.SwitchToPage("chat")
	ui.refreshStatus()
}

func (ui *TViewUI) toggleSidebar() {
	ui.sidebarVisible = !ui.sidebarVisible
	ui.mountChat(ui.searchActive)
	switch {
	case ui.sidebarVisible:
		ui.App.SetFocus(ui.Sidebar)
	case ui.searchActive:
		ui.App.SetFocus(ui.searchInput)
	default:
		ui.focusChatInput()
	}
	ui.refreshStatus()
}

func (ui *TViewUI) syncInputHeight() {
	if ui.InputField == nil {
		return
	}
	h := inputHeight(ui.InputField.GetText())
	ui.InputField.SetSize(h, 0)
	if ui.chatColumnFlex != nil {
		ui.chatColumnFlex.ResizeItem(ui.InputField, h, 0)
	}
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

func (ui *TViewUI) newConversation() {
	ui.messages = []openai.ChatCompletionMessage{}
	ui.convID = ""
	ui.ChatView.Clear()
	ui.focusChatInput()
	ui.appendSystemMsg(fmt.Sprintf("New conversation started. (provider=%s model=%s prompt=%s)",
		ui.config.CanonicalProvider(), ui.config.Model, ui.systemPrompt))
}

func (ui *TViewUI) confirmQuit() {
	if page, _ := ui.Pages.GetFrontPage(); page == "confirm-quit" {
		return
	}
	modal := tview.NewModal().
		SetText("Quit the application?").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			ui.Pages.RemovePage("confirm-quit")
			if buttonLabel == "Quit" {
				ui.App.Stop()
				return
			}
			ui.focusChatIfFront()
		})

	ui.Pages.AddPage("confirm-quit", modal, true, true)
}
