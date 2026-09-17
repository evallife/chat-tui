package ui

import (
	"fmt"

	"github.com/evallife/chat-tui/internal/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *TViewUI) showSystemPrompts() {
	prompts, err := ui.storage.ListSystemPrompts()
	if err != nil {
		ui.appendSystemMsg(fmt.Sprintf("Error loading prompts: %v", err))
		return
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	list := tview.NewList()
	list.SetBorder(true).SetTitle(" System Prompts (Enter=Use, e=Edit, d=Delete, n=New) ")
	for _, p := range prompts {
		pCopy := p
		preview := p.Content
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}
		if preview == "" {
			preview = "(no prompt)"
		}
		list.AddItem(p.Name, preview, 0, func() {
			ui.systemPrompt = pCopy.Content
			ui.appendSystemMsg(fmt.Sprintf("System prompt set to: %s", pCopy.Name))
			ui.Pages.RemovePage("system_prompts_mgr")
			ui.focusChatInput()
		})
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'n':
			ui.showPromptEditor(types.SystemPrompt{})
			return nil
		case 'e':
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(prompts) {
				ui.showPromptEditor(prompts[idx])
			}
			return nil
		case 'd':
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(prompts) {
				ui.confirmDeletePrompt(prompts[idx], list)
			}
			return nil
		}
		switch event.Key() {
		case tcell.KeyEsc:
			ui.Pages.RemovePage("system_prompts_mgr")
			ui.focusChatInput()
			return nil
		}
		return event
	})

	help := tview.NewTextView().SetDynamicColors(true)
	fmt.Fprintln(help, "[::b]n[::-] New  [::b]e[::-] Edit  [::b]d[::-] Delete  [::b]Enter[::-] Use  [::b]Esc[::-] Back")

	flex.AddItem(list, 0, 1, true)
	flex.AddItem(help, 1, 0, false)

	ui.Pages.AddPage("system_prompts_mgr", flex, true, true)
	ui.Pages.SwitchToPage("system_prompts_mgr")
}

func (ui *TViewUI) showPromptEditor(p types.SystemPrompt) {
	isNew := p.ID == ""
	title := " Edit Prompt "
	if isNew {
		title = " New Prompt "
	}

	form := tview.NewForm().
		AddInputField("Name", p.Name, 40, nil, nil).
		AddTextArea("Content", p.Content, 60, 6, 0, nil)

	form.AddButton("Save", func() {
		name := form.GetFormItem(0).(*tview.InputField).GetText()
		content := form.GetFormItem(1).(*tview.TextArea).GetText()
		if name == "" {
			return
		}
		prompt := types.SystemPrompt{
			ID:      p.ID,
			Name:    name,
			Content: content,
		}
		if err := ui.storage.SaveSystemPrompt(prompt); err != nil {
			ui.appendSystemMsg(fmt.Sprintf("Error saving prompt: %v", err))
		}
		ui.Pages.RemovePage("prompt_editor")
		ui.showSystemPrompts()
	}).
		AddButton("Cancel", func() {
			ui.Pages.RemovePage("prompt_editor")
			ui.showSystemPrompts()
		})

	form.SetBorder(true).SetTitle(title)
	ui.Pages.AddPage("prompt_editor", form, true, true)
	ui.Pages.SwitchToPage("prompt_editor")
}

func (ui *TViewUI) confirmDeletePrompt(p types.SystemPrompt, _ *tview.List) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete prompt \"%s\"?", p.Name)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				if err := ui.storage.DeleteSystemPrompt(p.ID); err != nil {
					ui.appendSystemMsg(fmt.Sprintf("Error deleting prompt: %v", err))
				}
				ui.Pages.RemovePage("confirm_delete_prompt")
				ui.showSystemPrompts()
			} else {
				ui.Pages.RemovePage("confirm_delete_prompt")
			}
		})
	ui.Pages.AddPage("confirm_delete_prompt", modal, true, true)
	ui.Pages.SwitchToPage("confirm_delete_prompt")
}
