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
		AddItem("Quit", "Exit app", 'q', func() { ui.App.Stop() })

	ui.Sidebar.SetBorder(true).SetTitle(" Menu ")
	ui.Sidebar.SetTitleColor(tview.Styles.TitleColor)
}

func (ui *TViewUI) chatColumn(withSearch bool) tview.Primitive {
	if withSearch {
		searchFlex := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(ui.searchInput, 1, 0, true).
			AddItem(ui.ChatView, 0, 1, false).
			AddItem(ui.InputField, 3, 1, false)
		searchFlex.SetBorder(true).SetTitle(" Chat History ")
		return searchFlex
	}
	footer := ui.buildFooterBar()
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.ChatView, 0, 1, false).
		AddItem(ui.InputField, 3, 1, true).
		AddItem(footer, 3, 1, false)
}

func (ui *TViewUI) mountChat(withSearch bool) {
	ui.MainFlex = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.Sidebar, 20, 1, false).
		AddItem(ui.chatColumn(withSearch), 0, 4, true)
	ui.Pages.AddPage("chat", ui.MainFlex, true, true)
	ui.Pages.SwitchToPage("chat")
}

func (ui *TViewUI) toggleSidebar() {
	if _, item := ui.MainFlex.GetItem(0).(*tview.List); item {
		ui.MainFlex.RemoveItem(ui.Sidebar)
		return
	}
	oldFlex := ui.MainFlex
	ui.MainFlex = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.Sidebar, 20, 1, false).
		AddItem(oldFlex.GetItem(0), 0, 4, true)
	ui.Pages.AddPage("chat", ui.MainFlex, true, true)
	ui.Pages.SwitchToPage("chat")
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
	bar.AddItem(ui.makeButton("New", ui.newConversation), 0, 1, false)
	bar.AddItem(ui.makeButton("History", ui.showHistory), 0, 1, false)
	bar.AddItem(ui.makeButton("Export", ui.exportHistory), 0, 1, false)
	bar.AddItem(ui.makeButton("Copy", ui.showCopyMode), 0, 1, false)
	bar.AddItem(ui.makeButton("Prompts", ui.showSystemPrompts), 0, 1, false)
	bar.AddItem(ui.makeButton("Settings", ui.showSettings), 0, 1, false)
	bar.AddItem(ui.makeButton("Stop", ui.stopStreaming), 0, 1, false)
	bar.AddItem(ui.makeButton("Quit", func() { ui.App.Stop() }), 0, 1, false)
	return bar
}

func (ui *TViewUI) newConversation() {
	ui.messages = []openai.ChatCompletionMessage{}
	ui.convID = ""
	ui.ChatView.Clear()
	ui.Pages.SwitchToPage("chat")
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
			}
		})

	ui.Pages.AddPage("confirm-quit", modal, true, true)
}
