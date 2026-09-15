package ui

import (
	"context"
	"fmt"
	"sync"

	"github.com/evallife/chat-tui/internal/api"
	"github.com/rivo/tview"
)

const toolConfirmPage = "confirm-tool"

func (ui *TViewUI) ConfirmTool(ctx context.Context, name, detail string) error {
	ui.confirmMu.Lock()
	defer ui.confirmMu.Unlock()

	ui.confirmGen++
	gen := ui.confirmGen

	result := make(chan error, 1)
	var once sync.Once
	send := func(err error) {
		once.Do(func() {
			result <- err
		})
	}

	preview := detail
	if len(preview) > 400 {
		preview = preview[:400] + "…"
	}

	ui.App.QueueUpdateDraw(func() {
		if ui.confirmGen != gen {
			return
		}
		if ui.Pages.HasPage(toolConfirmPage) {
			ui.Pages.RemovePage(toolConfirmPage)
		}
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Agent wants to run %s:\n\n%s\n\nAllow this action?", name, preview)).
			AddButtons([]string{"Allow", "Deny"}).
			SetDoneFunc(func(_ int, buttonLabel string) {
				if ui.confirmGen != gen {
					return
				}
				if ui.Pages.HasPage(toolConfirmPage) {
					ui.Pages.RemovePage(toolConfirmPage)
				}
				if buttonLabel == "Allow" {
					send(nil)
					return
				}
				send(fmt.Errorf("%w: %s", api.ErrToolDenied, name))
			})
		ui.Pages.AddPage(toolConfirmPage, modal, true, true)
	})

	select {
	case <-ctx.Done():
		ui.confirmGen++
		ui.App.QueueUpdateDraw(func() {
			if ui.Pages.HasPage(toolConfirmPage) {
				ui.Pages.RemovePage(toolConfirmPage)
			}
		})
		return ctx.Err()
	case err := <-result:
		return err
	}
}
