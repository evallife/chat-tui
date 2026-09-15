package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *TViewUI) setupCopyView() {
	ui.CopyView = tview.NewTextArea()
	ui.CopyView.SetBorder(true).SetTitle(" Copy Mode (Ctrl+C: Copy, Esc: Back) ")
	ui.CopyView.SetTitleColor(tview.Styles.TitleColor)
	ui.CopyView.SetWrap(true).SetWordWrap(true)
	ui.CopyView.SetTextStyle(tcell.StyleDefault.Foreground(tview.Styles.PrimaryTextColor).Background(tview.Styles.PrimitiveBackgroundColor))
	ui.CopyView.SetLabelStyle(tcell.StyleDefault.Foreground(tview.Styles.SecondaryTextColor).Background(tview.Styles.PrimitiveBackgroundColor))
	ui.CopyView.SetClipboard(func(text string) {
		_ = clipboard.WriteAll(text)
	}, func() string {
		text, _ := clipboard.ReadAll()
		return text
	})
	ui.CopyView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			ui.hideCopyMode()
			return nil
		case tcell.KeyCtrlC:
			ui.copySelectionToClipboard()
			return nil
		case tcell.KeyCtrlX, tcell.KeyBackspace, tcell.KeyDelete, tcell.KeyCtrlD, tcell.KeyCtrlH:
			return nil
		}
		if event.Rune() != 0 {
			return nil
		}
		return event
	})
}

func (ui *TViewUI) buildCopyText() string {
	var sb strings.Builder
	if ui.systemPrompt != "" {
		sb.WriteString(fmt.Sprintf("System Prompt: %s\n\n", ui.systemPrompt))
	}
	for _, msg := range ui.messages {
		sb.WriteString(fmt.Sprintf("%s:\n%s\n\n", strings.ToUpper(msg.Role), msg.Content))
	}
	return sb.String()
}

func (ui *TViewUI) copySelectionToClipboard() {
	if ui.CopyView == nil {
		return
	}
	text, _, _ := ui.CopyView.GetSelection()
	if text == "" {
		text = ui.CopyView.GetText()
	}
	if err := clipboard.WriteAll(text); err != nil {
		ui.reportError("Clipboard copy failed", err)
	}
}

func (ui *TViewUI) showCopyMode() {
	if ui.CopyView == nil {
		ui.setupCopyView()
		ui.Pages.AddPage("copy", ui.CopyView, true, true)
	}
	ui.CopyView.SetText(ui.buildCopyText(), true)
	ui.Pages.SwitchToPage("copy")
	ui.App.SetFocus(ui.CopyView)
}

func (ui *TViewUI) hideCopyMode() {
	ui.Pages.SwitchToPage("chat")
	ui.App.SetFocus(ui.InputField)
}
