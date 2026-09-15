package api

import (
	"github.com/cloudwego/eino/schema"
	"github.com/sashabaranov/go-openai"
)

// ToSchemaMessages converts stored go-openai messages to eino schema messages.
// System messages are skipped when skipSystem is true so ChatModelAgent Instruction
// remains the single system prompt source.
func ToSchemaMessages(msgs []openai.ChatCompletionMessage, skipSystem bool) []*schema.Message {
	out := make([]*schema.Message, 0, len(msgs))
	for _, m := range msgs {
		if skipSystem && (m.Role == openai.ChatMessageRoleSystem || m.Role == "developer") {
			continue
		}
		out = append(out, toSchemaMessage(m))
	}
	return out
}

func toSchemaMessage(m openai.ChatCompletionMessage) *schema.Message {
	role := schema.RoleType(m.Role)
	if m.Role == openai.ChatMessageRoleFunction {
		role = schema.Tool
	}
	msg := &schema.Message{
		Role:       role,
		Content:    m.Content,
		Name:       m.Name,
		ToolCallID: m.ToolCallID,
	}
	if len(m.ToolCalls) == 0 {
		return msg
	}
	msg.ToolCalls = make([]schema.ToolCall, 0, len(m.ToolCalls))
	for _, tc := range m.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, schema.ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: schema.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return msg
}
