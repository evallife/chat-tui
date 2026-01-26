package types

import (
	"time"
	"github.com/sashabaranov/go-openai"
)

type Config struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

type Conversation struct {
	ID        string                         `json:"id"`
	Title     string                         `json:"title"`
	Messages  []openai.ChatCompletionMessage `json:"messages"`
	CreatedAt time.Time                      `json:"created_at"`
}
