package ui

import (
	"fmt"
	"strings"
)

type escAction string

const (
	escNone         escAction = "none"
	escCancelStream escAction = "cancel_stream"
	escCloseSearch  escAction = "close_search"
	escBackChat     escAction = "back_chat"
	escBackPrompts  escAction = "back_prompts"
	escCloseOverlay escAction = "close_overlay"
)

func escActionFor(frontPage string, searchActive, streaming bool) escAction {
	switch frontPage {
	case "copy", "history", "settings", "system_prompts_mgr":
		return escBackChat
	case "prompt_editor", "confirm_delete_prompt":
		return escBackPrompts
	case "confirm-quit", "confirm-delete", "export-dialog":
		return escCloseOverlay
	case "confirm-tool":
		return escNone
	case "chat":
		if searchActive {
			return escCloseSearch
		}
		if streaming {
			return escCancelStream
		}
		return escNone
	default:
		return escNone
	}
}

func recallHistoryOnUp(cursorRow int) bool {
	return cursorRow == 0
}

func recallHistoryOnDown(historyIndex int) bool {
	return historyIndex != -1
}

func inputHeight(text string) int {
	n := strings.Count(text, "\n") + 1
	if n < 3 {
		return 3
	}
	if n > 8 {
		return 8
	}
	return n
}

func statusLine(provider, model string, streaming, sidebarVisible bool) string {
	var b strings.Builder
	if streaming {
		fmt.Fprintf(&b, "[%s/%s · streaming]", provider, model)
	} else {
		fmt.Fprintf(&b, "[%s/%s]", provider, model)
	}
	if sidebarVisible {
		b.WriteString("  menu on")
	}
	if streaming {
		b.WriteString("  Esc stop")
	} else {
		b.WriteString("  Ctrl+N new  Ctrl+H hist  Ctrl+S set  Ctrl+B menu  Ctrl+C quit")
	}
	return b.String()
}
