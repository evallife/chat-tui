package api

import (
	"strings"

	"github.com/evallife/chat-tui/internal/types"
)

// DefaultAgentInstruction is the built-in coding-agent system prompt used when
// the conversation system prompt is empty and default instruction is enabled.
const DefaultAgentInstruction = `You are a capable terminal coding agent for chat-tui.

Prefer using tools to inspect the workspace before answering: list directories, read files, glob or search text, and run short commands when needed. Stay inside the configured workspace; do not attempt to access paths outside it.

Be concise and accurate. When you have finished the user's request, call the exit tool (if available) with a brief final_result summarizing the outcome.`

// EffectiveInstruction returns the conversation system prompt when non-empty;
// otherwise the default agent instruction when enabled in cfg; otherwise empty.
func EffectiveInstruction(systemPrompt string, cfg types.Config) string {
	if strings.TrimSpace(systemPrompt) != "" {
		return systemPrompt
	}
	if cfg.DefaultInstructionEnabled() {
		return DefaultAgentInstruction
	}
	return ""
}
