package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/evallife/chat-tui/internal/types"
	"github.com/sashabaranov/go-openai"
)

// Client runs chat through an Eino ChatModelAgent + Runner.
type Client struct {
	config  types.Config
	confirm ToolConfirmer
}

func NewClient(cfg types.Config) *Client {
	cfg.Normalize()
	return &Client{config: cfg}
}

func (c *Client) UpdateConfig(cfg types.Config) {
	cfg.Normalize()
	c.config = cfg
}

func (c *Client) SetConfirmer(confirm ToolConfirmer) {
	c.confirm = confirm
}

func (c *Client) Config() types.Config {
	return c.config
}

type EventKind int

const (
	EventText EventKind = iota
	EventReasoning
	EventToolCall
	EventToolResult
	EventStatus
)

// Event is a UI-facing slice of an ADK AgentEvent stream.
type Event struct {
	Kind     EventKind
	Text     string
	ToolName string
	Status   string
}

// StreamAgent executes ChatModelAgent via Runner with EnableStreaming and
// emits AgentEvent deltas (assistant text, tool-call status, tool results).
func (c *Client) StreamAgent(ctx context.Context, instruction string, history []openai.ChatCompletionMessage, emit func(Event)) (string, error) {
	cm, err := NewChatModel(ctx, c.config)
	if err != nil {
		return "", fmt.Errorf("chat model (%s): %w", c.config.CanonicalProvider(), err)
	}

	c.config.Normalize()
	effective := EffectiveInstruction(instruction, c.config)
	name := c.config.AgentName
	if name == "" {
		name = "chat-tui"
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          name,
		Description:   "Terminal multi-provider chat agent",
		Instruction:   effective,
		Model:         cm,
		MaxIterations: c.config.MaxIterations,
		Exit:          adk.ExitTool{},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: extraToolsFor(c.config, c.confirm),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("chat model agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	input := ToSchemaMessages(history, true)
	if len(input) == 0 {
		return "", fmt.Errorf("no messages to send")
	}

	iter := runner.Run(ctx, input)
	var full strings.Builder
	seenTools := map[string]struct{}{}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			if ctx.Err() != nil {
				return full.String(), ctx.Err()
			}
			return full.String(), event.Err
		}
		if event.Action != nil {
			emitAction(event.Action, emit)
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		if mv.IsStreaming && mv.MessageStream != nil {
			sr := mv.MessageStream
			sr.SetAutomaticClose()
			for {
				chunk, recvErr := sr.Recv()
				if recvErr != nil {
					if errors.Is(recvErr, io.EOF) {
						break
					}
					sr.Close()
					if ctx.Err() != nil {
						return full.String(), ctx.Err()
					}
					return full.String(), recvErr
				}
				handleMessage(chunk, mv, &full, seenTools, emit)
			}
			sr.Close()
			continue
		}
		handleMessage(mv.Message, mv, &full, seenTools, emit)
	}

	return full.String(), nil
}

func emitAction(action *adk.AgentAction, emit func(Event)) {
	if emit == nil || action == nil {
		return
	}
	if action.Exit {
		emit(Event{Kind: EventStatus, Status: "agent exit"})
	}
	if action.Interrupted != nil {
		emit(Event{Kind: EventStatus, Status: "interrupted"})
	}
	if action.TransferToAgent != nil && action.TransferToAgent.DestAgentName != "" {
		emit(Event{Kind: EventStatus, Status: "transfer → " + action.TransferToAgent.DestAgentName})
	}
}

func handleMessage(msg *schema.Message, mv *adk.MessageVariant, full *strings.Builder, seen map[string]struct{}, emit func(Event)) {
	if msg == nil {
		return
	}
	role := mv.Role
	if role == "" {
		role = msg.Role
	}
	if msg.ReasoningContent != "" && emit != nil {
		emit(Event{Kind: EventReasoning, Text: msg.ReasoningContent})
	}
	if role == schema.Tool || msg.Role == schema.Tool {
		name := mv.ToolName
		if name == "" {
			name = msg.ToolName
		}
		if emit != nil {
			emit(Event{Kind: EventToolResult, ToolName: name, Text: msg.Content})
		}
		return
	}
	for _, tc := range msg.ToolCalls {
		name := tc.Function.Name
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if emit != nil {
			emit(Event{Kind: EventToolCall, ToolName: name, Text: tc.Function.Arguments})
		}
	}
	if msg.Content == "" {
		return
	}
	full.WriteString(msg.Content)
	if emit != nil {
		emit(Event{Kind: EventText, Text: msg.Content})
	}
}
