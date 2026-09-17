package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sashabaranov/go-openai"
)

func (ui *TViewUI) setupSearch() {
	ui.searchInput = tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(40)
	ui.searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			if event.Modifiers()&tcell.ModShift != 0 {
				ui.searchPrev()
			} else if len(ui.searchHits) > 0 {
				ui.searchNext()
			} else {
				ui.searchQuery = ui.searchInput.GetText()
				ui.performSearch()
			}
			return nil
		case tcell.KeyEsc:
			ui.clearSearch()
			return nil
		}
		return event
	})
}

func (ui *TViewUI) toggleSearch() {
	if ui.searchActive {
		ui.clearSearch()
		return
	}
	ui.searchActive = true
	ui.searchInput.SetText("")
	ui.mountChat(true)
	ui.App.SetFocus(ui.searchInput)
}

func (ui *TViewUI) performSearch() {
	query := strings.ToLower(ui.searchQuery)
	if query == "" {
		ui.clearSearch()
		return
	}

	ui.searchHits = []string{}
	matchIdx := 0
	var buf strings.Builder

	if ui.systemPrompt != "" {
		fmt.Fprintf(&buf, "[gray][i]System Prompt: %s[-][/i]\n\n", ui.systemPrompt)
	}

	for _, m := range ui.messages {
		roleColor := "purple"
		if m.Role == openai.ChatMessageRoleAssistant {
			roleColor = "green"
		}
		fmt.Fprintf(&buf, "[%s][b]%s[-][/b]\n", roleColor, strings.ToUpper(m.Role))

		content := m.Content
		lower := strings.ToLower(content)
		start := 0
		for {
			idx := strings.Index(lower[start:], query)
			if idx < 0 {
				break
			}
			absIdx := start + idx
			regionID := fmt.Sprintf("hit_%d", matchIdx)
			ui.searchHits = append(ui.searchHits, regionID)
			matchIdx++

			buf.WriteString(content[:absIdx])
			fmt.Fprintf(&buf, `["%s"][::b]%s[""]`, regionID, content[absIdx:absIdx+len(query)])
			start = absIdx + len(query)
		}
		buf.WriteString(content[start:])
		buf.WriteString("\n\n")
	}

	if len(ui.searchHits) == 0 {
		ui.appendSystemMsg(fmt.Sprintf("No matches for: %s", ui.searchQuery))
		ui.clearSearch()
		return
	}

	ui.searchIdx = 0
	ui.ChatView.Clear()
	fmt.Fprint(ui.ChatView, buf.String())
	ui.ChatView.Highlight(ui.searchHits[ui.searchIdx])
	ui.ChatView.ScrollToHighlight()
	ui.ChatView.SetTitle(fmt.Sprintf(" Chat History (%d matches) ", len(ui.searchHits)))
	ui.App.SetFocus(ui.searchInput)
}

func (ui *TViewUI) clearSearch() {
	ui.searchActive = false
	ui.searchQuery = ""
	ui.searchHits = nil
	ui.searchIdx = 0

	ui.mountChat(false)
	ui.refreshChat()
	ui.focusChatInput()
}

func (ui *TViewUI) searchNext() {
	if len(ui.searchHits) == 0 {
		return
	}
	ui.searchIdx = (ui.searchIdx + 1) % len(ui.searchHits)
	ui.ChatView.Highlight(ui.searchHits[ui.searchIdx])
	ui.ChatView.ScrollToHighlight()
	ui.ChatView.SetTitle(fmt.Sprintf(" Chat History (%d/%d) ", ui.searchIdx+1, len(ui.searchHits)))
}

func (ui *TViewUI) searchPrev() {
	if len(ui.searchHits) == 0 {
		return
	}
	ui.searchIdx = (ui.searchIdx - 1 + len(ui.searchHits)) % len(ui.searchHits)
	ui.ChatView.Highlight(ui.searchHits[ui.searchIdx])
	ui.ChatView.ScrollToHighlight()
	ui.ChatView.SetTitle(fmt.Sprintf(" Chat History (%d/%d) ", ui.searchIdx+1, len(ui.searchHits)))
}
