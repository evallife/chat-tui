package api

import (
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestToSchemaMessagesSkipsSystem(t *testing.T) {
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: "sys"},
		{Role: openai.ChatMessageRoleUser, Content: "hi"},
		{Role: openai.ChatMessageRoleAssistant, Content: "hello"},
	}
	out := ToSchemaMessages(msgs, true)
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2", len(out))
	}
	if out[0].Content != "hi" || string(out[0].Role) != "user" {
		t.Fatalf("first=%+v", out[0])
	}
	if out[1].Content != "hello" {
		t.Fatalf("second=%+v", out[1])
	}
}

func TestToSchemaMessagesKeepsToolCalls(t *testing.T) {
	msgs := []openai.ChatCompletionMessage{{
		Role: openai.ChatMessageRoleAssistant,
		ToolCalls: []openai.ToolCall{{
			ID:   "c1",
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      "lookup",
				Arguments: `{"q":"x"}`,
			},
		}},
	}}
	out := ToSchemaMessages(msgs, true)
	if len(out) != 1 || len(out[0].ToolCalls) != 1 {
		t.Fatalf("unexpected: %+v", out)
	}
	if out[0].ToolCalls[0].Function.Name != "lookup" {
		t.Fatalf("tool name=%q", out[0].ToolCalls[0].Function.Name)
	}
}
