package ui

import (
	"strings"
	"testing"
)

func TestEscActionFor(t *testing.T) {
	tests := []struct {
		name      string
		page      string
		search    bool
		streaming bool
		want      escAction
	}{
		{name: "idle chat", page: "chat", want: escNone},
		{name: "streaming chat", page: "chat", streaming: true, want: escCancelStream},
		{name: "search", page: "chat", search: true, want: escCloseSearch},
		{name: "search during stream", page: "chat", search: true, streaming: true, want: escCloseSearch},
		{name: "history", page: "history", want: escBackChat},
		{name: "settings", page: "settings", want: escBackChat},
		{name: "copy", page: "copy", want: escBackChat},
		{name: "prompts", page: "system_prompts_mgr", want: escBackChat},
		{name: "prompt editor", page: "prompt_editor", want: escBackPrompts},
		{name: "delete prompt", page: "confirm_delete_prompt", want: escBackPrompts},
		{name: "confirm-tool", page: "confirm-tool", streaming: true, want: escNone},
		{name: "confirm-quit", page: "confirm-quit", want: escCloseOverlay},
		{name: "confirm-delete", page: "confirm-delete", want: escCloseOverlay},
		{name: "export-dialog", page: "export-dialog", want: escCloseOverlay},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escActionFor(tt.page, tt.search, tt.streaming)
			if got != tt.want {
				t.Fatalf("escActionFor(%q, search=%v, streaming=%v)=%q, want %q",
					tt.page, tt.search, tt.streaming, got, tt.want)
			}
		})
	}
}

func TestRecallHistoryOnUp(t *testing.T) {
	if !recallHistoryOnUp(0) {
		t.Fatal("row 0 should recall history")
	}
	if recallHistoryOnUp(1) {
		t.Fatal("row 1 should move the cursor, not recall history")
	}
}

func TestRecallHistoryOnDown(t *testing.T) {
	if recallHistoryOnDown(-1) {
		t.Fatal("idle input should not intercept Down")
	}
	if !recallHistoryOnDown(0) {
		t.Fatal("browsing history should intercept Down")
	}
	if !recallHistoryOnDown(3) {
		t.Fatal("historyIndex != -1 should intercept Down")
	}
}

func TestInputHeight(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{text: "", want: 3},
		{text: "one line", want: 3},
		{text: "a\nb", want: 3},
		{text: "a\nb\nc", want: 3},
		{text: "a\nb\nc\nd", want: 4},
		{text: "1\n2\n3\n4\n5\n6", want: 6},
		{text: "1\n2\n3\n4\n5\n6\n7\n8", want: 8},
		{text: "1\n2\n3\n4\n5\n6\n7\n8\n9\n10", want: 8},
	}
	for _, tt := range tests {
		if got := inputHeight(tt.text); got != tt.want {
			t.Fatalf("inputHeight(%q)=%d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestStatusLine(t *testing.T) {
	idle := statusLine("openai", "gpt-4o-mini", false, false)
	wantIdle := "[openai/gpt-4o-mini]  Ctrl+N new  Ctrl+H hist  Ctrl+S set  Ctrl+B menu  Ctrl+C quit"
	if idle != wantIdle {
		t.Fatalf("idle status=%q, want %q", idle, wantIdle)
	}
	stream := statusLine("openai", "gpt-4o-mini", true, false)
	wantStream := "[openai/gpt-4o-mini · streaming]  Esc stop"
	if stream != wantStream {
		t.Fatalf("stream status=%q, want %q", stream, wantStream)
	}
	menu := statusLine("ollama", "llama3", false, true)
	for _, part := range []string{"[ollama/llama3]", "menu on", "Ctrl+B menu"} {
		if !strings.Contains(menu, part) {
			t.Fatalf("menu status %q missing %q", menu, part)
		}
	}
}
