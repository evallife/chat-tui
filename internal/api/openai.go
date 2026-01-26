package api

import (
	"context"
	"github.com/sashabaranov/go-openai"
	"github.com/evallife/chat-tui/internal/types"
)

type Client struct {
	openaiClient *openai.Client
	config       types.Config
}

func NewClient(cfg types.Config) *Client {
	config := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}
	return &Client{
		openaiClient: openai.NewClientWithConfig(config),
		config:       cfg,
	}
}

func (c *Client) StreamChat(ctx context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionStream, error) {
	req := openai.ChatCompletionRequest{
		Model:    c.config.Model,
		Messages: messages,
		Stream:   true,
	}
	return c.openaiClient.CreateChatCompletionStream(ctx, req)
}
